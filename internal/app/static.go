package app

import (
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

		switch {
		case r.URL.Path == "/",
			r.URL.Path == "/login",
			r.URL.Path == "/identity",
			r.URL.Path == "/list",
			r.URL.Path == "/chat",
			strings.HasPrefix(r.URL.Path, "/chat/"):
			http.ServeFile(w, r, indexPath)
			return
		}

		// 尝试按静态文件返回（若不存在则 404）
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		fullPath := filepath.Join(a.staticDir, clean)
		if fi, err := os.Stat(fullPath); err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, r)
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
		// 去除 /lsp 前缀
		path := strings.TrimPrefix(r.URL.Path, "/lsp")
		if path == "" || path == "/" {
			http.NotFound(w, r)
			return
		}

		// 构建完整文件路径
		fullPath := filepath.Join("lsp", filepath.FromSlash(path))

		// 检查文件是否存在
		fi, err := os.Stat(fullPath)
		if err != nil || fi.IsDir() {
			http.NotFound(w, r)
			return
		}

		// 提供文件服务
		http.ServeFile(w, r, fullPath)
	})
}

