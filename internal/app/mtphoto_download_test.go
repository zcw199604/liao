package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"liao/internal/config"
)

func TestDownloadMtPhotoOriginal_RequiresJWT(t *testing.T) {
	jwtSvc := NewJWTService("secret", 24)
	a := &App{
		cfg:       config.Config{LspRoot: "/lsp"},
		staticDir: ".",
		jwt:       jwtSvc,
		mtPhoto:   NewMtPhotoService("http://example.invalid", "u", "/lsp", &http.Client{Timeout: 2 * time.Second}),
	}
	a.handler = a.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}
}

func TestDownloadMtPhotoOriginal_StreamsFileWithDisposition(t *testing.T) {
	const md5Value = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const body = "JPEGDATA"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"auth_code":  "ac",
				"expires_in": time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/fileDownload/1/" + md5Value:
			if got := r.Header.Get("x-api-key"); got != "api-key-1" {
				t.Fatalf("x-api-key=%q, want api-key-1", got)
			}
			if got := r.Header.Get("jwt"); got != "" {
				t.Fatalf("jwt header should be empty, got %q", got)
			}
			if got := r.Header.Get("Cookie"); got != "" {
				t.Fatalf("Cookie header should be empty, got %q", got)
			}
			if got := r.URL.Query().Get("auth_code"); got != "ac" {
				t.Fatalf("auth_code=%q, want ac", got)
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = io.WriteString(w, body)
			return
		case "/gateway/filesInMD5":
			// 由 handler 用于补齐下载文件名
			if got := r.Header.Get("x-api-key"); got != "api-key-1" {
				t.Fatalf("x-api-key(filesInMD5)=%q, want api-key-1", got)
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "filePath": "/lsp/path/pic.jpg"},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)

	jwtSvc := NewJWTService("secret", 24)
	token, err := jwtSvc.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	a := &App{
		cfg:       config.Config{LspRoot: "/lsp"},
		staticDir: ".",
		jwt:       jwtSvc,
		mtPhoto:   NewMtPhotoService(srv.URL, "api-key-1", "/lsp", srv.Client()),
	}
	a.handler = a.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadMtPhotoOriginal?id=1&md5="+md5Value, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Header().Get("Content-Type")); got != "image/jpeg" {
		t.Fatalf("Content-Type=%q, want image/jpeg", got)
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Fatalf("Content-Disposition=%q, want attachment", cd)
	}
	if !strings.Contains(cd, "filename*=UTF-8''pic.jpg") {
		t.Fatalf("Content-Disposition=%q, want filename*=UTF-8''pic.jpg", cd)
	}
	if rec.Body.String() != body {
		t.Fatalf("body=%q, want %q", rec.Body.String(), body)
	}
}
