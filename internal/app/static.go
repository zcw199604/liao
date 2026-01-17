package app

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) spaHandler() http.Handler {
	indexPath := filepath.Join(a.staticDir, "index.html")
	fileServer := http.FileServer(http.Dir(a.staticDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		// 尝试按静态文件返回（若不存在则交给 SPA 回退判定）
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		fullPath := filepath.Join(a.staticDir, clean)
		if fi, err := os.Stat(fullPath); err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA 回退：支持 createWebHistory() 刷新（/list、/chat/... 等）
		// - 浏览器刷新/直达页面通常带 Accept: text/html
		// - 无扩展名路径也视为前端路由（避免尾随 / 或新增路由漏配导致 404）
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/html") || filepath.Ext(r.URL.Path) == "" {
			http.ServeFile(w, r, indexPath)
			return
		}

		http.NotFound(w, r)
	})
}

func (a *App) uploadFileServer() http.Handler {
	return http.StripPrefix("/upload", http.FileServer(http.Dir("upload")))
}

func (a *App) lspFileServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("lsp文件请求", "rawPath", r.URL.Path, "requestURI", r.RequestURI)

		// 直接使用 r.URL.Path 作为绝对路径（已包含 /lsp 前缀）
		fullPath := r.URL.Path
		slog.Info("lsp文件路径", "fullPath", fullPath)

		// 检查文件是否存在
		fi, err := os.Stat(fullPath)
		if err != nil {
			slog.Error("lsp文件不存在", "fullPath", fullPath, "error", err)
			http.NotFound(w, r)
			return
		}
		if fi.IsDir() {
			slog.Warn("lsp路径是目录", "fullPath", fullPath)
			http.NotFound(w, r)
			return
		}

		slog.Info("lsp文件找到", "fullPath", fullPath, "size", fi.Size())
		// 提供文件服务
		http.ServeFile(w, r, fullPath)
	})
}
