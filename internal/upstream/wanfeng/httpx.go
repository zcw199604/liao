package wanfeng

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"
)

// 会话相关的错误码（对方在 401 时返回，表示需要重新握手）。
const (
	apiCodeSessionExpired = 40310
	apiBodyLimit          = 4 << 20
	// mediaBodyLimit 是单次媒体搬运的体积上限（对方限制：图片≤10MB、视频≤50MB）。
	mediaBodyLimit = 80 << 20
)

// Identity 表示对方下发的匿名身份。
type Identity struct {
	UID      string
	Nickname string
}

// APIError 表示对方返回的业务错误。
type APIError struct {
	Code    string
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return fmt.Sprintf("晚风接口错误 code=%s status=%d msg=%s", e.Code, e.Status, e.Message)
	}
	return fmt.Sprintf("晚风接口错误 status=%d msg=%s", e.Status, e.Message)
}

// APIClient 负责与对方站点的 HTTP 交互：会话握手、加密信封、身份下发。
// 所有易变参数来自 Protocol，对方改版时通过配置覆盖即可。
type APIClient struct {
	baseURL  string
	wsURL    string
	origin   string
	referer  string
	proto    Protocol
	http     *http.Client
	sessions *sessionStore
	logger   *slog.Logger
	now      func() time.Time

	uploadUserMu sync.RWMutex
	uploadUserID string
}

// APIClientConfig 是 APIClient 的构造参数。
type APIClientConfig struct {
	BaseURL string
	// WSURL 可显式指定 WebSocket 地址（留空则按 BaseURL 推导）。
	WSURL      string
	HTTPClient *http.Client
	Protocol   Protocol
	Logger     *slog.Logger
}

// NewAPIClient 创建对方站点的 HTTP 客户端。
func NewAPIClient(cfg APIClientConfig) *APIClient {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	proto := cfg.Protocol
	if strings.TrimSpace(proto.HandshakePath) == "" {
		proto = DefaultProtocol()
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	// 对方用 Cookie 承载匿名身份（identity-cookie 接口下发），媒体上传等接口按 Cookie 认定身份，
	// 因此这里必须带上 Cookie Jar，否则上传会返回 40300「请求错误，请刷新页面」。
	if cfg.HTTPClient.Jar == nil {
		if jar, err := cookiejar.New(nil); err == nil {
			clientCopy := *cfg.HTTPClient
			clientCopy.Jar = jar
			cfg.HTTPClient = &clientCopy
		}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	origin := base
	if u, err := url.Parse(base); err == nil && u.Scheme != "" && u.Host != "" {
		origin = u.Scheme + "://" + u.Host
	}

	c := &APIClient{
		baseURL: base,
		wsURL:   strings.TrimSpace(cfg.WSURL),
		origin:  origin,
		referer: origin + "/niming/",
		proto:   proto,
		http:    cfg.HTTPClient,
		logger:  logger,
		now:     time.Now,
	}
	c.sessions = newSessionStore(c.handshake, c.now)
	return c
}

// Protocol 返回当前使用的协议参数。
func (c *APIClient) Protocol() Protocol { return c.proto }

// BaseURL 返回站点根地址。
func (c *APIClient) BaseURL() string { return c.baseURL }

// BuildWSURL 生成 WebSocket 地址（带时间戳参数，对齐对方前端行为）。
func (c *APIClient) BuildWSURL() string {
	base := c.baseURL
	if c.wsURL != "" {
		sep := "?"
		if strings.Contains(c.wsURL, "?") {
			sep = "&"
		}
		return fmt.Sprintf("%s%s%s=%d", c.wsURL, sep, c.proto.WSTimestampParam, c.now().UnixMilli())
	}
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	sep := "?"
	if strings.Contains(c.proto.WSPath, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%s%s%s=%d", base, c.proto.WSPath, sep, c.proto.WSTimestampParam, c.now().UnixMilli())
}

// Session 返回当前可用会话（必要时自动握手）。
func (c *APIClient) Session(ctx context.Context) (*Session, error) {
	return c.sessions.get(ctx)
}

// SealFrameData 把明文帧数据加密为 "v3.<iv>.<ct>" 字符串。
func (c *APIClient) SealFrameData(sess *Session, plaintext []byte) (string, error) {
	if sess == nil {
		return "", ErrBadEnvelope
	}
	return sealEnvelope(sess.Key, sess.ID, plaintext, c.proto.EnvelopePrefix)
}

// OpenFrameData 解密帧数据；不是密文信封时按明文返回（对方若取消加密仍可用）。
func (c *APIClient) OpenFrameData(sess *Session, data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, nil
	}
	if !strings.HasPrefix(data, c.proto.EnvelopePrefix) {
		return []byte(data), nil
	}
	if sess == nil {
		return nil, ErrBadEnvelope
	}
	return openEnvelope(sess.Key, sess.ID, data, c.proto.EnvelopePrefix)
}

// handshake 执行一次 ECDH 握手并派生会话密钥。
func (c *APIClient) handshake(ctx context.Context) (*Session, error) {
	kp, err := newHandshakeKeypair()
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"v":   c.proto.EnvelopeVersion,
		"pub": kp.pubB64,
	}
	raw, err := c.post(ctx, c.proto.HandshakePath, body, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			SID string `json:"sid"`
			Pub string `json:"pub"`
			Exp int64  `json:"exp"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("解析握手响应失败: %w", err)
	}
	if !resp.Success || resp.Data.SID == "" || resp.Data.Pub == "" {
		return nil, &APIError{Code: "HANDSHAKE_FAILED", Message: resp.Message}
	}

	key, err := kp.deriveSessionKey(resp.Data.Pub, resp.Data.SID, c.proto.HKDFInfo)
	if err != nil {
		return nil, err
	}

	exp := c.now().Add(23 * time.Hour)
	if resp.Data.Exp > 0 {
		exp = time.UnixMilli(resp.Data.Exp)
	}

	return &Session{ID: resp.Data.SID, Key: key, Exp: exp}, nil
}

// Call 调用业务接口：自动加会话信封、自动解密响应 data、会话过期自动重试一次。
func (c *APIClient) Call(ctx context.Context, path string, params map[string]any) (map[string]any, error) {
	if params == nil {
		params = map[string]any{}
	}

	sess, err := c.sessions.get(ctx)
	if err != nil {
		return nil, err
	}

	envelope, err := c.buildRequestEnvelope(sess, params)
	if err != nil {
		return nil, err
	}

	raw, err := c.post(ctx, path, envelope, sess)
	if err != nil {
		return nil, err
	}

	node, apiErr := c.decodeResponse(raw, sess)
	if apiErr != nil {
		// 会话过期：作废后重试一次
		var ae *APIError
		sessionExpired := errors.As(apiErr, &ae) &&
			(ae.Code == fmt.Sprintf("%d", apiCodeSessionExpired) || ae.Status == http.StatusUnauthorized)
		if sessionExpired {
			c.sessions.invalidate(sess.ID)
			sess2, err2 := c.sessions.get(ctx)
			if err2 != nil {
				return nil, err2
			}
			envelope2, err2 := c.buildRequestEnvelope(sess2, params)
			if err2 != nil {
				return nil, err2
			}
			raw2, err2 := c.post(ctx, path, envelope2, sess2)
			if err2 != nil {
				return nil, err2
			}
			return c.decodeResponseData(raw2, sess2)
		}
		return nil, apiErr
	}
	return node, nil
}

// GenerateIdentity 向对方申请一个新的匿名身份（uid + 昵称）。
func (c *APIClient) GenerateIdentity(ctx context.Context) (*Identity, error) {
	data, err := c.Call(ctx, c.proto.GenerateIdentityPath, map[string]any{})
	if err != nil {
		return nil, err
	}
	uid := PickString(data, c.proto.UIDKeys...)
	if uid == "" {
		return nil, &APIError{Code: "IDENTITY_EMPTY", Message: "未获取到身份 uid"}
	}
	nickname := PickString(data, c.proto.NicknameKeys...)
	return &Identity{UID: uid, Nickname: nickname}, nil
}

// BindIdentityCookie 把 identityToken 绑定到服务端 Cookie（对方的身份「换发」机制，可选调用）。
func (c *APIClient) BindIdentityCookie(ctx context.Context, identityToken string) error {
	if strings.TrimSpace(identityToken) == "" {
		return nil
	}
	_, err := c.Call(ctx, c.proto.IdentityCookiePath, map[string]any{"identityToken": identityToken})
	return err
}

func (c *APIClient) buildRequestEnvelope(sess *Session, params map[string]any) (map[string]any, error) {
	plain, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("序列化请求参数失败: %w", err)
	}
	ivB64, ctB64, err := sealParts(sess.Key, sess.ID, plain)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"v":   c.proto.EnvelopeVersion,
		"sid": sess.ID,
		"iv":  ivB64,
		"ct":  ctB64,
	}
	return body, nil
}

// post 发送 JSON 请求并返回原始响应体。
func (c *APIClient) post(ctx context.Context, path string, body map[string]any, sess *Session) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", c.proto.AcceptHeader)
	req.Header.Set("User-Agent", c.proto.UserAgent)
	req.Header.Set("Origin", c.origin)
	req.Header.Set("Referer", c.referer)
	if c.proto.GetViaPostHeader != "" {
		req.Header.Set(c.proto.GetViaPostHeader, c.proto.GetViaPostValue)
	}
	if sess != nil && c.proto.SessionHeader != "" {
		req.Header.Set(c.proto.SessionHeader, sess.ID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求晚风接口失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, apiBodyLimit))
	if err != nil {
		return nil, fmt.Errorf("读取晚风响应失败: %w", err)
	}
	if resp.StatusCode >= 400 && !looksLikeJSON(data) {
		// 常见于 WAF 拦截：返回 HTML 403/5xx 页面
		return nil, &APIError{Status: resp.StatusCode, Message: "晚风接口被拦截或不可用（可能触发风控）"}
	}
	return data, nil
}

// decodeResponse 解析响应并返回解密后的 data 对象。
func (c *APIClient) decodeResponse(raw []byte, sess *Session) (map[string]any, error) {
	return c.decodeResponseData(raw, sess)
}

func (c *APIClient) decodeResponseData(raw []byte, sess *Session) (map[string]any, error) {
	var node map[string]any
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, fmt.Errorf("解析晚风响应失败: %w", err)
	}

	success, hasSuccess := node["success"].(bool)
	code := PickString(node, "code")
	message := PickString(node, "message")
	if hasSuccess && !success {
		// 对方一部分接口成功时省略 success 字段、失败时给 success=false + code；
		// 另一部分成功时 success=true + code=0。这里只在「明确失败」时报错，
		// 避免把 success=false 但 data 正常的历史接口误判成失败。
		if code != "" && code != "0" {
			return nil, &APIError{Code: code, Message: message}
		}
		if message != "" && !hasDataField(node) {
			return nil, &APIError{Code: code, Message: message}
		}
	}

	data, ok := node["data"]
	if !ok || data == nil {
		return map[string]any{}, nil
	}

	switch v := data.(type) {
	case string:
		plain, err := c.OpenFrameData(sess, v)
		if err != nil {
			// 解密失败不致命：原样返回让上层按纯文本处理
			c.logger.Warn("晚风响应 data 解密失败", "error", err)
			return map[string]any{}, nil
		}
		var out map[string]any
		if err := json.Unmarshal(plain, &out); err != nil {
			return map[string]any{}, nil
		}
		return out, nil
	case map[string]any:
		return v, nil
	default:
		return map[string]any{}, nil
	}
}

// UploadResult 是媒体上传到对方后的结果。
type UploadResult struct {
	// URL 是对方可访问的完整媒体地址（已拼好域名）。
	URL string
	// Path 是对方返回的原始路径（可能是相对路径）。
	Path string
	// FileID 是路径最后一段（对方用它刷新有效期）。
	FileID string
}

// UploadMedia 把媒体字节以 multipart/form-data 上传到对方站点。
//
// 该接口不使用加密信封：对方前端也是裸 fetch + credentials:include，
// 身份完全靠 Cookie（由 BindIdentityCookie 换取），因此这里不带任何会话头。
//
// 若对方返回 40300/40310（会话或身份 Cookie 失效），会重建会话并重试一次。
func (c *APIClient) UploadMedia(ctx context.Context, filename, contentType string, data []byte, bucket string) (*UploadResult, error) {
	result, err := c.uploadMediaOnce(ctx, filename, contentType, data, bucket)
	if err == nil {
		return result, nil
	}
	if !isSessionOrCookieError(err) {
		return nil, err
	}
	// 会话过期：作废本地缓存，重新握手后再试一次（Cookie 由调用方在重连时刷新）。
	if sess := c.sessions.cached(); sess != nil {
		c.sessions.invalidate(sess.ID)
	}
	if _, herr := c.Session(ctx); herr != nil {
		return nil, err
	}
	c.logger.Debug("媒体上传重试", "reason", err)
	return c.uploadMediaOnce(ctx, filename, contentType, data, bucket)
}

// isSessionOrCookieError 判断是否为「会话/身份 Cookie 失效」这类可重试错误。
func isSessionOrCookieError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case "40300", "40310", "SESSION_EXPIRED":
		return true
	}
	return apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden
}

func (c *APIClient) uploadMediaOnce(ctx context.Context, filename, contentType string, data []byte, bucket string) (*UploadResult, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("媒体内容为空")
	}
	proto := c.proto
	if strings.TrimSpace(bucket) == "" {
		bucket = proto.BucketImage
	}
	if strings.TrimSpace(filename) == "" {
		filename = "media"
	}

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)
	// 与浏览器表单一致：文件分段带真实 Content-Type（对方按它判断图片/视频）。
	partHeader := make(textproto.MIMEHeader)
	ct := strings.TrimSpace(contentType)
	if ct == "" {
		ct = "application/octet-stream"
	}
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		escapeMultipartQuotes(proto.UploadFileField), escapeMultipartQuotes(filename)))
	partHeader.Set("Content-Type", ct)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, fmt.Errorf("构造上传表单失败: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("写入上传表单失败: %w", err)
	}
	// userId：对方按此归属媒体；取当前绑定的 uid（会话由 WS 登录建立，这里失败也不阻塞）
	if uid := c.currentUploadUserID(); uid != "" && proto.UploadUserField != "" {
		_ = writer.WriteField(proto.UploadUserField, uid)
	}
	if proto.UploadBucketField != "" {
		_ = writer.WriteField(proto.UploadBucketField, bucket)
	}
	if proto.UploadHashField != "" {
		sum := sha256.Sum256(data)
		_ = writer.WriteField(proto.UploadHashField, hex.EncodeToString(sum[:])[:32])
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("结束上传表单失败: %w", err)
	}

	reqURL := c.baseURL + proto.UploadPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bodyBuf)
	if err != nil {
		return nil, fmt.Errorf("构造上传请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", proto.AcceptHeader)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("User-Agent", proto.UserAgent)
	req.Header.Set("Origin", c.origin)
	req.Header.Set("Referer", c.referer)
	// 对方前端的媒体上传是普通 fetch（不做信封加密），只带 Cookie（credentials: include），
	// 不带会话头 / X-Get-Via-Post；这里保持一致，避免多余头触发 40300。

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("上传媒体失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, apiBodyLimit))
	if err != nil {
		return nil, fmt.Errorf("读取上传响应失败: %w", err)
	}
	if !looksLikeJSON(raw) {
		return nil, &APIError{Status: resp.StatusCode, Message: "上传接口返回非 JSON（可能被风控拦截）"}
	}

	var node map[string]any
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, fmt.Errorf("解析上传响应失败: %w", err)
	}

	// 部分环境下响应 data 仍可能是 v3 信封，尽量解密（失败就按原值处理）。
	respData := node["data"]
	if s, ok := respData.(string); ok {
		plain, derr := c.OpenFrameData(c.sessions.cached(), s)
		if derr == nil && len(plain) > 0 {
			var out any
			if json.Unmarshal(plain, &out) == nil {
				respData = out
			}
		}
	}

	filePath := ""
	switch v := respData.(type) {
	case map[string]any:
		filePath = PickString(v, proto.UploadFilePathKeys...)
	case string:
		filePath = strings.TrimSpace(v)
	}
	if filePath == "" {
		return nil, &APIError{Code: PickString(node, "code"), Message: firstNonEmpty(PickString(node, "message", "msg"), "上传未返回文件路径")}
	}

	url := filePath
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		if !strings.HasPrefix(url, "/") {
			url = "/" + url
		}
		url = c.origin + url
	}

	return &UploadResult{URL: url, Path: filePath, FileID: MediaURLKey(filePath)}, nil
}

// DownloadMedia 下载对方媒体（一次性读入内存，带体积上限）。
func (c *APIClient) DownloadMedia(ctx context.Context, mediaURL string) ([]byte, string, error) {
	mediaURL = strings.TrimSpace(mediaURL)
	if mediaURL == "" {
		return nil, "", fmt.Errorf("媒体地址为空")
	}
	if !strings.HasPrefix(mediaURL, "http://") && !strings.HasPrefix(mediaURL, "https://") {
		if !strings.HasPrefix(mediaURL, "/") {
			mediaURL = "/" + mediaURL
		}
		mediaURL = c.origin + mediaURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("构造下载请求失败: %w", err)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", c.proto.UserAgent)
	req.Header.Set("Origin", c.origin)
	req.Header.Set("Referer", c.referer)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("下载媒体失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &APIError{Status: resp.StatusCode, Message: "下载媒体返回异常状态"}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, mediaBodyLimit))
	if err != nil {
		return nil, "", fmt.Errorf("读取媒体内容失败: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("媒体内容为空（可能已过期）")
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// RefreshMediaExpiry 尽力刷新对方媒体有效期（失败忽略）。
func (c *APIClient) RefreshMediaExpiry(ctx context.Context, fileID, userID string, flash bool) error {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil
	}
	params := map[string]any{"fileId": fileID, "userId": userID}
	if flash {
		params["flash"] = true
	}
	_, err := c.Call(ctx, c.proto.RefreshExpiryPath, params)
	return err
}

// SetUploadUserID 设置上传时携带的对方 uid（由 WS 登录后的绑定提供）。
func (c *APIClient) SetUploadUserID(uid string) {
	c.uploadUserMu.Lock()
	c.uploadUserID = strings.TrimSpace(uid)
	c.uploadUserMu.Unlock()
}

func (c *APIClient) currentUploadUserID() string {
	c.uploadUserMu.RLock()
	defer c.uploadUserMu.RUnlock()
	return c.uploadUserID
}

func looksLikeJSON(b []byte) bool {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}

// Clone 复制出一个独立的客户端（独立 Cookie Jar 与握手会话），用于按身份隔离的媒体上传。
//
// 对方用 Cookie 承载匿名身份，多个身份共用同一 Cookie Jar 会把媒体归属串号，
// 因此每个身份各自持有一个客户端。
func (c *APIClient) Clone() *APIClient {
	if c == nil {
		return nil
	}
	transport := http.DefaultTransport
	if c.http != nil && c.http.Transport != nil {
		transport = c.http.Transport
	}
	clone := NewAPIClient(APIClientConfig{
		BaseURL:    c.baseURL,
		WSURL:      c.wsURL,
		Protocol:   c.proto,
		Logger:     c.logger,
		HTTPClient: &http.Client{Timeout: 60 * time.Second, Transport: transport},
	})
	return clone
}

// escapeMultipartQuotes 转义 multipart 头里的引号/反斜杠，避免文件名破坏分段头。
func escapeMultipartQuotes(v string) string {
	v = strings.ReplaceAll(v, "\\", "_")
	v = strings.ReplaceAll(v, "\"", "_")
	v = strings.ReplaceAll(v, "\r", "_")
	v = strings.ReplaceAll(v, "\n", "_")
	return v
}

// hasDataField 判断响应里是否带 data（用于区分「有数据的软失败」与真空响应）。
func hasDataField(node map[string]any) bool {
	data, ok := node["data"]
	if !ok || data == nil {
		return false
	}
	switch v := data.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case map[string]any:
		return len(v) > 0
	case []any:
		return len(v) > 0
	default:
		return true
	}
}
