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

