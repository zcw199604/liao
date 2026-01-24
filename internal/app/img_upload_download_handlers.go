package app

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// handleDownloadImgUpload 代理下载上游图片服务器的 /img/Upload/{path} 资源，并强制返回 attachment。
//
// 场景：前端预览面板内的“下载”按钮对跨域 URL 无法依赖 <a download> 强制下载，浏览器通常会新开标签页显示图片。
// 通过后端同源代理可稳定触发下载且保留文件名。
func (a *App) handleDownloadImgUpload(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}

	rawPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if rawPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "path 为空"})
		return
	}

	uploadPath := normalizeRemoteUploadPath(rawPath)
	if uploadPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "path 非法"})
		return
	}

	imgHost := strings.TrimSpace(a.getImageServerHostOnly())
	if imgHost == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "图片服务器未配置"})
		return
	}

	port := strings.TrimSpace(a.resolveImagePortByConfig(r.Context(), uploadPath))
	if port == "" {
		port = defaultSystemConfig.ImagePortFixed
	}

	target := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(imgHost, port),
		Path:   "/img/Upload/" + strings.TrimPrefix(uploadPath, "/"),
	}

	client := a.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求构造失败"})
		return
	}
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "下载失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", resp.Status, msg)})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "下载失败: " + resp.Status})
		return
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if contentLength := strings.TrimSpace(resp.Header.Get("Content-Length")); contentLength != "" {
		w.Header().Set("Content-Length", contentLength)
	}

	filename := strings.TrimSpace(path.Base(uploadPath))
	if filename == "" || filename == "." || filename == "/" {
		filename = "download"
	}
	if ext := strings.TrimSpace(path.Ext(filename)); ext == "" {
		if guessed := guessExtFromContentType(contentType); guessed != "" {
			filename += guessed
		}
	}

	fallback := sanitizeFilename(filename)
	if fallback == "" {
		fallback = "download"
	}

	w.Header().Set("Content-Disposition", buildAttachmentContentDisposition(fallback, filename))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}
