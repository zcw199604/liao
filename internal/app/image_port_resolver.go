package app

// ImagePortResolver 用于根据“真实媒体请求”判定图片服务端口，并对结果做内存缓存。

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	imagePortRealRequestTimeout = 800 * time.Millisecond
)

type ImagePortResolver struct {
	httpClient *http.Client
	ports      []string

	mu    sync.RWMutex
	cache map[string]string // host -> port
}

func NewImagePortResolver(httpClient *http.Client) *ImagePortResolver {
	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}

	// 端口候选顺序优先覆盖现有默认：先 900x 再 800x（与历史实现一致）。
	ports := []string{"9006", "9005", "9003", "9002", "9001", "8006", "8005", "8003", "8002", "8001"}
	return &ImagePortResolver{
		httpClient: client,
		ports:      ports,
		cache:      make(map[string]string),
	}
}

func (r *ImagePortResolver) ClearHost(host string) {
	if r == nil {
		return
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return
	}
	r.mu.Lock()
	delete(r.cache, host)
	r.mu.Unlock()
}

func (r *ImagePortResolver) ClearAll() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.cache = make(map[string]string)
	r.mu.Unlock()
}

func (r *ImagePortResolver) GetCached(host string) string {
	if r == nil {
		return ""
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	r.mu.RLock()
	port := r.cache[host]
	r.mu.RUnlock()
	return port
}

func (r *ImagePortResolver) ResolveByRealRequest(ctx context.Context, host, uploadPath string, minBytes int64) (string, bool) {
	if r == nil || strings.TrimSpace(host) == "" {
		return "", false
	}

	host = strings.TrimSpace(host)
	uploadPath = normalizeRemoteUploadPath(uploadPath)
	if uploadPath == "" {
		return "", false
	}

	if minBytes <= 0 {
		minBytes = defaultSystemConfig.ImagePortRealMinBytes
	}

	// 先验证缓存端口，避免不必要的全量扫描。
	if cached := r.GetCached(host); cached != "" {
		if r.checkPort(ctx, host, cached, uploadPath, minBytes) {
			return cached, true
		}
		r.ClearHost(host)
	}

	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		port string
		ok   bool
	}
	results := make(chan result, len(r.ports))

	for _, port := range r.ports {
		port := port
		go func() {
			ok := r.checkPort(raceCtx, host, port, uploadPath, minBytes)
			results <- result{port: port, ok: ok}
		}()
	}

	for i := 0; i < len(r.ports); i++ {
		select {
		case res := <-results:
			if !res.ok {
				continue
			}
			cancel()
			r.mu.Lock()
			r.cache[host] = res.port
			r.mu.Unlock()
			return res.port, true
		case <-raceCtx.Done():
			return "", false
		}
	}

	return "", false
}

func (r *ImagePortResolver) checkPort(ctx context.Context, host, port, uploadPath string, minBytes int64) bool {
	if host == "" || port == "" || uploadPath == "" || minBytes <= 0 {
		return false
	}

	target := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
		Path:   "/img/Upload/" + strings.TrimPrefix(uploadPath, "/"),
	}

	reqCtx, cancel := context.WithTimeout(ctx, imagePortRealRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target.String(), nil)
	if err != nil {
		return false
	}
	// 尽量只读取最小判定字节，避免下载完整媒体。
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", minBytes-1))

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}

	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if strings.Contains(contentType, "text/html") {
		return false
	}

	// 只读取前 minBytes，用于判断“确实是媒体内容”。
	limit := minBytes
	if limit > 64*1024 {
		limit = 64 * 1024
	}
	buf, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return false
	}

	if int64(len(buf)) < minBytes {
		return false
	}

	// 规避常见 HTML 占位页：去掉空白后以 "<" 开头。
	head := bytes.TrimSpace(buf)
	if len(head) > 0 && head[0] == '<' {
		return false
	}

	return true
}

func normalizeRemoteUploadPath(path string) string {
	path = strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	if path == "" {
		return ""
	}
	if idx := strings.IndexAny(path, "?#"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// 兼容传入完整 URL 或带 /img/Upload/ 前缀的情况。
	if u, err := url.Parse(path); err == nil && u != nil && u.Host != "" {
		path = u.Path
	}

	path = strings.ReplaceAll(path, "//", "/")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "img/Upload/")
	path = strings.TrimPrefix(path, "/img/Upload/")
	path = strings.TrimPrefix(path, "/")

	// 最小化清洗：禁止明显的路径穿越片段，且限制长度避免异常请求。
	if strings.Contains(path, "..") {
		return ""
	}
	if len(path) > 1024 {
		return ""
	}

	return path
}
