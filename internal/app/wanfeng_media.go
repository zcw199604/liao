package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"liao/internal/upstream/wanfeng"
)

const (
	// 媒体搬运结果缓存时长：避免同一条消息重复搬运（对方可能重复推送/自己回显）。
	wanfengMediaCacheTTL = 30 * time.Minute
	// 缓存条目上限。
	wanfengMediaCacheMax = 512
	// 失败记忆时长：对方媒体过期（404）后不可能恢复，记住一段时间避免每次拉历史都重试+刷日志。
	wanfengMediaFailTTL = 10 * time.Minute
	// 单条媒体搬运超时。
	wanfengMediaTimeout = 60 * time.Second
	// 单次搬运读取上限（对方限制：图片≤10MB、视频≤50MB）。
	mediaBridgeBodyLimit = 80 << 20
)

// wanfengMediaBridge 实现 wanfeng.MediaBridge：
// 在「chat2019 媒体服务器（我方前端渲染用）」和「晚风媒体服务」之间搬运字节。
//
// 之所以两边都走 chat2019 媒体服务器：前端解析 "[文件名]" 时统一用 imgServer 拼 URL，
// 这样晚风线路的图片/视频渲染、历史预览、乐观消息合并等逻辑全部复用，无需改前端。
type wanfengMediaBridge struct {
	app   *App
	api   *wanfeng.APIClient
	log   *slog.Logger
	proto wanfeng.Protocol

	mu      sync.Mutex
	outward map[string]mediaCacheEntry // chat2019 文件名 → 晚风 URL
	inward  map[string]mediaCacheEntry // 晚风 URL 键 → chat2019 文件名
	failed  map[string]time.Time       // 已知失效的晚风 URL 键 → 记录时间
	// clients 按身份缓存独立的对方客户端（对方用 Cookie 认定身份，必须隔离）。
	clients map[string]*mediaClient
	// mediaMap 是「晚风媒体键 → 我方文件名」的持久映射（可为 nil）。
	// 对方媒体约 30 分钟就过期，只有持久化映射才能让已搬运过的历史媒体永久可解析。
	mediaMap wanfengMediaMapStore
}

// mediaClient 是某个身份专用（独立 Cookie Jar）的媒体上传客户端。
type mediaClient struct {
	api   *wanfeng.APIClient
	token string
	since time.Time
}

type mediaCacheEntry struct {
	value   string
	expires time.Time
}

func newWanfengMediaBridge(app *App, api *wanfeng.APIClient, logger *slog.Logger) *wanfengMediaBridge {
	if logger == nil {
		logger = slog.Default()
	}
	return &wanfengMediaBridge{
		app:     app,
		api:     api,
		log:     logger.With("mod", "wanfeng-media"),
		proto:   api.Protocol(),
		outward: make(map[string]mediaCacheEntry),
		inward:  make(map[string]mediaCacheEntry),
		failed:  make(map[string]time.Time),
		clients: make(map[string]*mediaClient),
	}
}

// MirrorOutbound 把 chat2019 媒体文件名上传到对方，返回对方可访问的 URL。
func (b *wanfengMediaBridge) MirrorOutbound(identityID, kind, remoteFilename string) (string, error) {
	remoteFilename = strings.TrimSpace(remoteFilename)
	if remoteFilename == "" {
		return "", fmt.Errorf("媒体文件名为空")
	}
	if cached, ok := b.get(b.outward, remoteFilename); ok {
		return cached, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), wanfengMediaTimeout)
	defer cancel()

	data, contentType, err := b.loadLocalOrFetch(ctx, remoteFilename)
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = contentTypeFromFilename(remoteFilename)
	}

	api, err := b.clientFor(ctx, identityID)
	if err != nil {
		return "", err
	}
	uploaded, err := api.UploadMedia(ctx, path.Base(remoteFilename), contentType, data, b.bucketFor(kind))
	if err != nil {
		return "", err
	}

	// 对方媒体只有约 30 分钟有效期，站点前端也是「发送前刷新一次」；
	// 这里同样续期，保证对方点开时文件还在（失败不影响发送）。
	if refreshErr := api.RefreshMediaExpiry(ctx, uploaded.FileID, api.UploadUserID(), false); refreshErr != nil {
		b.log.Debug("刷新对方媒体有效期失败", "fileId", uploaded.FileID, "error", refreshErr)
	}

	b.put(b.outward, remoteFilename, uploaded.URL)
	// 双向登记，避免自己消息回显时又下载一遍
	b.put(b.inward, wanfeng.MediaURLKey(uploaded.URL), remoteFilename)
	b.rememberPersistent(wanfeng.MediaURLKey(uploaded.URL), remoteFilename, identityID)
	b.log.Info("媒体已上传到对方", "filename", remoteFilename, "kind", kind, "bytes", len(data))
	return uploaded.URL, nil
}

// MirrorInbound 把对方媒体下载并上传到 chat2019 媒体服务器，返回文件名。
func (b *wanfengMediaBridge) MirrorInbound(identityID, kind, remoteURL string) (string, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", fmt.Errorf("媒体地址为空")
	}
	key := wanfeng.MediaURLKey(remoteURL)
	if cached, ok := b.get(b.inward, key); ok {
		return cached, nil
	}
	// 内存缓存过期不代表没了：只要搬运过一次，持久映射里就还在（对方文件本身已经过期）
	if name, ok := b.lookupPersistent(key); ok {
		b.put(b.inward, key, name)
		return name, nil
	}
	if b.knownFailed(key) {
		return "", wanfeng.ErrMediaExpired
	}

	ctx, cancel := context.WithTimeout(context.Background(), wanfengMediaTimeout)
	defer cancel()

	api, err := b.clientFor(ctx, identityID)
	if err != nil {
		return "", err
	}
	data, contentType, err := api.DownloadMedia(ctx, remoteURL)
	if err != nil {
		// 404 = 对方文件已过期/不存在（不可恢复），记下来别再重试
		if isExpiredMediaError(err) {
			b.markFailed(key)
			return "", wanfeng.ErrMediaExpired
		}
		return "", err
	}

	filename := mediaFilenameFor(key, kind, contentType, remoteURL)
	if contentType == "" {
		contentType = contentTypeFromFilename(filename)
	}

	remoteFilename, err := b.uploadToImageServer(ctx, identityID, filename, data)
	if err != nil {
		return "", err
	}

	b.put(b.inward, key, remoteFilename)
	b.put(b.outward, remoteFilename, remoteURL)
	b.rememberPersistent(key, remoteFilename, identityID)
	b.log.Info("对方媒体已转存到媒体服务器", "kind", kind, "bytes", len(data), "filename", remoteFilename)
	return remoteFilename, nil
}

// lookupPersistent 查询持久映射（无映射存储时直接未命中）。
func (b *wanfengMediaBridge) lookupPersistent(key string) (string, bool) {
	if b.mediaMap == nil || strings.TrimSpace(key) == "" {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	name, ok, err := b.mediaMap.Get(ctx, key)
	if err != nil {
		b.log.Debug("查询晚风媒体映射失败", "key", key, "error", err)
		return "", false
	}
	return name, ok
}

// rememberPersistent 记录持久映射（失败只记日志：内存缓存 + 失败记忆仍可兜底）。
func (b *wanfengMediaBridge) rememberPersistent(key, remoteFilename, identityID string) {
	if b.mediaMap == nil || strings.TrimSpace(key) == "" || strings.TrimSpace(remoteFilename) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := b.mediaMap.Put(ctx, key, remoteFilename, identityID); err != nil {
		b.log.Debug("保存晚风媒体映射失败", "key", key, "error", err)
	}
}

// knownFailed 判断该晚风媒体是否已被判定为过期（带记忆时长，避免重复请求）。
func (b *wanfengMediaBridge) knownFailed(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	since, ok := b.failed[key]
	if !ok {
		return false
	}
	if time.Since(since) > wanfengMediaFailTTL {
		delete(b.failed, key)
		return false
	}
	return true
}

// markFailed 记录一个已失效的晚风媒体键（顺带清理过期记录，避免无界增长）。
func (b *wanfengMediaBridge) markFailed(key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failed == nil {
		b.failed = make(map[string]time.Time)
	}
	if len(b.failed) >= wanfengMediaCacheMax {
		for k, since := range b.failed {
			if time.Since(since) > wanfengMediaFailTTL {
				delete(b.failed, k)
			}
		}
	}
	b.failed[key] = time.Now()
}

// isExpiredMediaError 判断下载失败是否属于「对方文件已过期/不存在」。
func isExpiredMediaError(err error) bool {
	var apiErr *wanfeng.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Status == http.StatusNotFound || apiErr.Code == "404" {
		return true
	}
	msg := apiErr.Message
	return strings.Contains(msg, "已失效") || strings.Contains(msg, "不存在")
}

// clientFor 返回该身份专用的对方客户端（独立 Cookie Jar + 已绑定身份 Cookie）。
//
// 同进程可能有多个身份同时聊天，而对方的媒体上传按 Cookie 认定归属，
// 因此不能共用同一个 Cookie Jar，否则会出现「图片传到别人名下」。
func (b *wanfengMediaBridge) clientFor(ctx context.Context, identityID string) (*wanfeng.APIClient, error) {
	if b.api == nil {
		return nil, fmt.Errorf("媒体客户端未初始化")
	}
	if b.app == nil || b.app.wanfeng == nil || strings.TrimSpace(identityID) == "" {
		return b.api, nil
	}

	binding, err := b.app.wanfeng.BindingOf(ctx, identityID)
	if err != nil || binding == nil || strings.TrimSpace(binding.UpstreamUID) == "" {
		// 还没绑定成功：退回共享客户端（至少能下载对方媒体）
		return b.api, nil
	}

	b.mu.Lock()
	cached := b.clients[identityID]
	b.mu.Unlock()
	if cached != nil && cached.token == binding.IdentityToken {
		return cached.api, nil
	}

	api := b.api.Clone()
	if api == nil {
		return b.api, nil
	}
	api.SetUploadUserID(binding.UpstreamUID)
	if token := strings.TrimSpace(binding.IdentityToken); token != "" {
		if bindErr := api.BindIdentityCookie(ctx, token); bindErr != nil {
			// Cookie 绑定失败不阻塞：上传接口也可能接受裸 userId
			b.log.Debug("绑定身份 Cookie 失败", "identityID", identityID, "error", bindErr)
		}
	}

	b.mu.Lock()
	if b.clients == nil {
		b.clients = make(map[string]*mediaClient)
	}
	b.clients[identityID] = &mediaClient{api: api, token: binding.IdentityToken, since: time.Now()}
	b.mu.Unlock()
	return api, nil
}

// loadLocalOrFetch 优先读本地已上传文件，其次从 chat2019 媒体服务器下载。
func (b *wanfengMediaBridge) loadLocalOrFetch(ctx context.Context, remoteFilename string) ([]byte, string, error) {
	if b.app != nil && b.app.mediaUpload != nil && b.app.mediaUpload.db != nil && b.app.fileStorage != nil {
		if record, err := b.app.mediaUpload.findMediaFileByRemoteFilename(ctx, remoteFilename, ""); err == nil && record != nil {
			if localPath := strings.TrimSpace(record.LocalPath); localPath != "" {
				if data, readErr := b.app.fileStorage.ReadLocalFile(localPath); readErr == nil && len(data) > 0 {
					return data, record.FileType, nil
				}
			}
		}
	}

	if b.app == nil {
		return nil, "", fmt.Errorf("媒体服务未初始化")
	}

	// 端口可能有多个候选：配置里的 host:port 最可信，其次是系统配置解析出的端口。
	var (
		lastErr  error
		attempts = 0
	)
	for _, port := range b.imagePortCandidates(ctx, remoteFilename) {
		attempts++
		url := fmt.Sprintf("http://%s:%s/img/Upload/%s", b.app.getImageServerHostOnly(), port, strings.TrimPrefix(remoteFilename, "/"))
		data, contentType, err := b.fetchBytes(ctx, url)
		if err == nil {
			return data, contentType, nil
		}
		lastErr = err
	}
	if attempts == 0 {
		return nil, "", fmt.Errorf("未获取到媒体服务器地址")
	}
	return nil, "", fmt.Errorf("读取我方媒体失败: %w", lastErr)
}

// imagePortCandidates 返回拉取我方媒体时可尝试的端口（去重、保序）。
func (b *wanfengMediaBridge) imagePortCandidates(ctx context.Context, remoteFilename string) []string {
	ports := make([]string, 0, 3)
	add := func(port string) {
		port = strings.TrimSpace(port)
		if port == "" {
			return
		}
		for _, existing := range ports {
			if existing == port {
				return
			}
		}
		ports = append(ports, port)
	}

	if host := b.app.imageServer.GetImgServerHost(); host != "" {
		if idx := strings.LastIndex(host, ":"); idx >= 0 {
			add(host[idx+1:])
		}
	}
	add(b.app.resolveImagePortByConfig(ctx, remoteFilename))
	add(defaultImagePort)
	return ports
}

// uploadToImageServer 把字节上传到 chat2019 媒体服务器，返回远程文件名。
func (b *wanfengMediaBridge) uploadToImageServer(ctx context.Context, identityID, filename string, data []byte) (string, error) {
	if b.app == nil || b.app.mediaUpload == nil {
		return "", fmt.Errorf("媒体服务未初始化")
	}
	imgHost := b.app.imageServer.GetImgServerHost()
	if imgHost == "" {
		return "", fmt.Errorf("未获取到媒体服务器地址")
	}

	userID := strings.TrimSpace(identityID)
	cookie := buildImageServerCookie(userID, b.nicknameOf(userID))
	uploadURL := fmt.Sprintf("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s", imgHost, userID)

	body, err := b.app.mediaUpload.uploadLocalFileToImageServer(
		ctx, uploadURL, imgHost, imageServerReferer, defaultImageServerUserAgent(), cookie, filename, data,
	)
	if err != nil {
		return "", fmt.Errorf("转存到我方媒体服务器失败: %w", err)
	}

	var parsed struct {
		State string `json:"state"`
		Msg   string `json:"msg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		// 部分实现直接返回文件名
		trimmed := strings.TrimSpace(string(body))
		if trimmed != "" && !strings.ContainsAny(trimmed, "{}\"") {
			return trimmed, nil
		}
		return "", fmt.Errorf("解析媒体服务器响应失败: %w", err)
	}
	if strings.ToUpper(strings.TrimSpace(parsed.State)) != "OK" || strings.TrimSpace(parsed.Msg) == "" {
		return "", fmt.Errorf("媒体服务器返回异常: %s", strings.TrimSpace(string(body)))
	}
	return strings.TrimSpace(parsed.Msg), nil
}

// nicknameOf 取身份昵称（用于合成媒体服务器 Cookie，对方不严格校验）。
func (b *wanfengMediaBridge) nicknameOf(identityID string) string {
	if b.app == nil || identityID == "" || b.app.wanfeng == nil {
		return "user"
	}
	if profile, ok := b.app.wanfeng.ProfileOf(identityID); ok && strings.TrimSpace(profile.Nickname) != "" {
		return profile.Nickname
	}
	return "user"
}

// fetchBytes 直接搬运一个 http(s) 地址。
func (b *wanfengMediaBridge) fetchBytes(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", defaultImageServerUserAgent())
	req.Header.Set("Origin", "http://v1.chat2019.cn")
	req.Header.Set("Referer", imageServerReferer)

	client := b.app.httpClient
	if client == nil {
		client = &http.Client{Timeout: wanfengMediaTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("媒体服务器返回 %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, mediaBridgeBodyLimit))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("媒体内容为空")
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func (b *wanfengMediaBridge) bucketFor(kind string) string {
	if kind == wanfeng.MediaKindAudio {
		return b.proto.BucketAudio
	}
	return b.proto.BucketImage
}

func (b *wanfengMediaBridge) get(m map[string]mediaCacheEntry, key string) (string, bool) {
	if key == "" {
		return "", false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := m[key]
	if !ok || time.Now().After(entry.expires) {
		if ok {
			delete(m, key)
		}
		return "", false
	}
	return entry.value, true
}

func (b *wanfengMediaBridge) put(m map[string]mediaCacheEntry, key, value string) {
	if key == "" || value == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(m) >= wanfengMediaCacheMax {
		for k := range m {
			delete(m, k)
			break
		}
	}
	m[key] = mediaCacheEntry{value: value, expires: time.Now().Add(wanfengMediaCacheTTL)}
}

// mediaFilenameFor 为转存的媒体生成文件名（对方 URL 最后一段可能已过期/无扩展名）。
func mediaFilenameFor(fileID, kind, contentType, remoteURL string) string {
	ext := extensionFromContentType(contentType)
	if ext == "" {
		if fromName := path.Ext(wanfeng.MediaURLKey(remoteURL)); fromName != "" {
			ext = fromName
		}
	}
	if ext == "" {
		switch kind {
		case wanfeng.MediaKindVideo:
			ext = ".mp4"
		case wanfeng.MediaKindAudio:
			ext = ".m4a"
		default:
			ext = ".jpg"
		}
	}

	base := fileID
	if base == "" {
		base = fmt.Sprintf("wf_%d", time.Now().UnixNano())
	}
	// 去掉已有扩展名后统一追加，避免出现 "a.jpg.jpg"
	if existing := path.Ext(base); existing != "" {
		base = strings.TrimSuffix(base, existing)
	}
	return base + ext
}

func extensionFromContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	switch ct {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/heic":
		return ".heic"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "video/webm":
		return ".webm"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/mp4", "audio/x-m4a":
		return ".m4a"
	case "audio/webm":
		return ".webm"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/amr":
		return ".amr"
	default:
		return ""
	}
}

func contentTypeFromFilename(filename string) string {
	switch strings.ToLower(path.Ext(strings.TrimSpace(filename))) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".heic":
		return "image/heic"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	case ".amr":
		return "audio/amr"
	default:
		return "application/octet-stream"
	}
}

// buildImageServerCookie 合成 chat2019 媒体服务器 Cookie（格式：id_昵称_时间戳_随机串）。
func buildImageServerCookie(userID, nickname string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = "0"
	}
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = "user"
	}
	return fmt.Sprintf("%s_%s_%d_%s", userID, nickname, time.Now().Unix(), randomSuffix(6))
}

func randomSuffix(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	now := time.Now().UnixNano()
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = charset[int(now>>uint(i*3))%len(charset)]
	}
	return string(out)
}

// imageServerReferer 是上传 chat2019 媒体服务器时携带的 Referer（与其它上传链路保持一致）。
const imageServerReferer = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj"

func defaultImageServerUserAgent() string {
	return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
}
