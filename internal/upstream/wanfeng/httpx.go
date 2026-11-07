package wanfeng

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 会话相关的错误码（对方在 401 时返回，表示需要重新握手）。
const (
	apiCodeSessionExpired = 40310
	apiBodyLimit          = 4 << 20
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

	success, _ := node["success"].(bool)
	code := PickString(node, "code")
	message := PickString(node, "message")
	if !success {
		return nil, &APIError{Code: code, Message: message}
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

func looksLikeJSON(b []byte) bool {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}
