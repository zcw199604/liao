package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"liao/internal/config"
)

const workerPreviewPrefix = "/worker-preview"
const workerPreviewCookie = "liao_worker_preview"
const workerPreviewMaxDocument = 16 << 20

type workerPreviewService struct {
	target             *url.URL
	username, password string
	client             *http.Client
	mediaClient        *http.Client
	sessionJWT         *JWTService
	mu                 sync.Mutex
	token              string
	expires            time.Time
}

func newWorkerPreviewService(cfg config.Config, client *http.Client) *workerPreviewService {
	target, _ := url.Parse(strings.TrimRight(cfg.WorkerPreviewURL, "/"))
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	// 登录凭据和代理 Bearer Token 不得随重定向发送到其他站点。
	proxyClient := *client
	proxyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	mediaClient := proxyClient
	// http.Client.Timeout 包含整个响应体，不能用普通 API 的 60 秒限制截断视频。
	// 媒体仍限制等待响应头的时间，传输生命周期由下游请求的 Context 控制。
	mediaClient.Timeout = 0
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if base, ok := transport.(*http.Transport); ok {
		mediaTransport := base.Clone()
		mediaTransport.ResponseHeaderTimeout = client.Timeout
		if mediaTransport.ResponseHeaderTimeout <= 0 {
			mediaTransport.ResponseHeaderTimeout = 60 * time.Second
		}
		mediaClient.Transport = mediaTransport
	}
	key := sha256.Sum256([]byte("liao/worker-preview/session\x00" + cfg.JWTSecret))
	return &workerPreviewService{
		target: target, username: cfg.WorkerPreviewUsername, password: cfg.WorkerPreviewPassword,
		client: &proxyClient, mediaClient: &mediaClient, sessionJWT: NewJWTService(string(key[:]), cfg.TokenExpireHours),
	}
}

func (s *workerPreviewService) configured() bool {
	return s != nil && s.target != nil && s.target.Host != "" &&
		(s.target.Scheme == "https" || s.target.Scheme == "http") && s.username != "" && s.password != ""
}

// 所有访问共用服务端账号；串行登录防止首次打开时间线时的并发请求重复登录。
func (s *workerPreviewService) accessToken(ctx context.Context, rejectedToken string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && s.token != rejectedToken && time.Now().Before(s.expires) {
		return s.token, nil
	}
	body, _ := json.Marshal(map[string]string{"username": s.username, "password": s.password})
	u := s.target.ResolveReference(&url.URL{Path: "/api/auth/login"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("无法创建预览登录请求")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("无法连接媒体预览服务")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("媒体预览自动登录失败（HTTP %d），请检查服务端账号配置", resp.StatusCode)
	}
	var result struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil || result.Token == "" || result.ExpiresIn <= 0 {
		return "", fmt.Errorf("媒体预览登录响应无效")
	}
	s.token = result.Token
	s.expires = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return s.token, nil
}

func workerPreviewError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Cache-Control", "private, no-store")
	// Worker 原页面读取 error，liao 外层页面读取 msg。
	writeJSON(w, status, map[string]any{"code": -1, "msg": message, "error": message})
}

func (a *App) handleWorkerPreviewSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method == http.MethodDelete {
		http.SetCookie(w, &http.Cookie{Name: workerPreviewCookie, Path: workerPreviewPrefix + "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		writeJSON(w, http.StatusOK, map[string]any{"code": 0})
		return
	}
	s := a.workerPreview
	if !s.configured() {
		workerPreviewError(w, http.StatusServiceUnavailable, "尚未配置媒体预览服务，请配置 WORKER_PREVIEW_URL、WORKER_PREVIEW_USERNAME、WORKER_PREVIEW_PASSWORD")
		return
	}
	if _, err := s.accessToken(r.Context(), ""); err != nil {
		workerPreviewError(w, http.StatusBadGateway, err.Error())
		return
	}
	// iframe 无法附带 Authorization。签发仅限此代理的 Cookie，失效时间不晚于当前 liao 登录。
	var parent jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), &parent,
		func(*jwt.Token) (any, error) { return a.jwt.secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		workerPreviewError(w, http.StatusUnauthorized, "登录已失效，请重新登录")
		return
	}
	claims := jwt.RegisteredClaims{Subject: "worker-preview", ExpiresAt: parent.ExpiresAt, IssuedAt: jwt.NewNumericDate(time.Now())}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.sessionJWT.secret)
	if err != nil {
		workerPreviewError(w, http.StatusInternalServerError, "无法创建媒体预览会话")
		return
	}
	// TLS 或 HTTPS 反代入口启用 Secure，同时兼容现有 HTTP 局域网访问。
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{Name: workerPreviewCookie, Value: token, Path: workerPreviewPrefix + "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, Expires: parent.ExpiresAt.Time})
	writeJSON(w, http.StatusOK, map[string]any{"code": 0, "data": map[string]string{"url": workerPreviewPrefix + "/preview"}})
}

func (a *App) handleWorkerPreviewProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	fail := func(status int, message string) {
		if r.URL.Path != workerPreviewPrefix+"/preview" {
			workerPreviewError(w, status, message)
			return
		}
		// iframe 的 load 事件不会区分 HTTP 成败，页面失败时直接展示可读提示。
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'self'")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.WriteHeader(status)
		fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>媒体预览暂不可用</title><body style="margin:0;min-height:100vh;display:grid;place-items:center;background:#09090b;color:#e4e4e7;font:15px system-ui;text-align:center"><main style="padding:24px"><h2>暂时无法打开媒体预览</h2><p>%s</p><p>请点击顶部“刷新预览”重试</p></main></body></html>`, html.EscapeString(message))
	}
	s := a.workerPreview
	cookie, err := r.Cookie(workerPreviewCookie)
	if err != nil || s == nil || !s.sessionJWT.ValidateToken(cookie.Value) {
		fail(http.StatusUnauthorized, "预览会话已失效，请刷新重试")
		return
	}
	if !s.configured() {
		fail(http.StatusServiceUnavailable, "尚未配置媒体预览服务")
		return
	}
	upstreamPath := strings.TrimPrefix(r.URL.Path, workerPreviewPrefix)
	media := upstreamPath == "/api/media"
	allowed := upstreamPath == "/preview" || upstreamPath == "/api/auth/session" || media
	switch upstreamPath {
	case "/api/preview/twitter_is_download", "/api/preview/twitter_link_media", "/api/preview/twitter_following", "/api/preview/tg_message":
		allowed = true
	}
	if !allowed {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && !(media && r.Method == http.MethodHead) {
		w.Header().Set("Allow", "GET")
		if media {
			w.Header().Set("Allow", "GET, HEAD")
		}
		fail(http.StatusMethodNotAllowed, "不支持的请求方式")
		return
	}
	query := r.URL.Query()
	query.Del("token") // 不接受浏览器提供 Worker Token，也不触发 Worker 的 query-token Cookie 跳转。
	target := s.target.ResolveReference(&url.URL{Path: upstreamPath, RawQuery: query.Encode()})
	token, err := s.accessToken(r.Context(), "")
	if err != nil {
		fail(http.StatusBadGateway, err.Error())
		return
	}
	var resp *http.Response
	for attempt := 0; attempt < 2; attempt++ {
		req, reqErr := http.NewRequestWithContext(r.Context(), r.Method, target.String(), nil)
		if reqErr != nil {
			fail(http.StatusBadGateway, "无法创建媒体预览请求")
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept-Encoding", "identity")
		for _, name := range []string{"Accept", "Range", "If-Range", "If-None-Match", "If-Modified-Since"} {
			if value := r.Header.Get(name); value != "" && (media || name == "Accept") {
				req.Header.Set(name, value)
			}
		}
		requestClient := s.client
		if media {
			requestClient = s.mediaClient
		}
		resp, err = requestClient.Do(req)
		if err != nil {
			fail(http.StatusBadGateway, "媒体预览请求失败，请稍后重试")
			return
		}
		loginRedirect := false
		if location, locationErr := resp.Location(); locationErr == nil {
			loginRedirect = location.Scheme == s.target.Scheme && location.Host == s.target.Host && location.Path == "/login"
		}
		if resp.StatusCode != http.StatusUnauthorized && !loginRedirect {
			break
		}
		resp.Body.Close()
		if attempt == 1 {
			fail(http.StatusBadGateway, "媒体预览登录已失效，请检查服务端账号配置")
			return
		}
		token, err = s.accessToken(r.Context(), token)
		if err != nil {
			fail(http.StatusBadGateway, err.Error())
			return
		}
	}
	defer resp.Body.Close()
	// 预览服务不需要其他跳转；不将上游地址、登录页或认证 Cookie 暴露给浏览器。
	if resp.StatusCode >= 300 && resp.StatusCode < 400 && resp.StatusCode != http.StatusNotModified {
		fail(http.StatusBadGateway, "媒体预览服务返回了意外跳转")
		return
	}
	if media {
		for _, name := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Content-Disposition", "ETag", "Last-Modified"} {
			if value := resp.Header.Get(name); value != "" {
				w.Header().Set(name, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		if r.Method != http.MethodHead {
			_, _ = io.Copy(w, resp.Body)
		}
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, workerPreviewMaxDocument+1))
	if err != nil || len(body) > workerPreviewMaxDocument {
		fail(http.StatusBadGateway, "媒体预览响应读取失败")
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/html") {
		// Worker 页面包含 JSON 编码的内联脚本；原样替换根 API 路径也覆盖这些脚本。
		body = bytes.ReplaceAll(body, []byte("/api/"), []byte(workerPreviewPrefix+"/api/"))
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'self'")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	} else if strings.HasPrefix(contentType, "application/json") {
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		var data any
		if err := decoder.Decode(&data); err != nil {
			fail(http.StatusBadGateway, "媒体预览返回了无效数据")
			return
		}
		body, err = json.Marshal(s.rewriteMediaURLs(data))
		if err != nil {
			fail(http.StatusBadGateway, "媒体预览数据处理失败")
			return
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Referrer-Policy", "same-origin")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (s *workerPreviewService) rewriteMediaURLs(data any) any {
	switch value := data.(type) {
	case map[string]any:
		for key, child := range value {
			value[key] = s.rewriteMediaURLs(child)
		}
	case []any:
		for index, child := range value {
			value[index] = s.rewriteMediaURLs(child)
		}
	case string:
		u, err := url.Parse(value)
		if err == nil && u.Path == "/api/media" && ((u.Scheme == "" && u.Host == "") || (u.Scheme == s.target.Scheme && u.Host == s.target.Host)) {
			return workerPreviewPrefix + u.RequestURI()
		}
	}
	return data
}
