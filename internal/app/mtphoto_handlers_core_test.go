package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMtPhotoService_GatewayGet_ParamValidation(t *testing.T) {
	svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
	if _, err := svc.GatewayGet(context.Background(), "s260", "m"); err == nil {
		t.Fatalf("expected error for not configured")
	}

	svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
	if _, err := svc.GatewayGet(context.Background(), "", "m"); err == nil {
		t.Fatalf("expected error for empty size")
	}
	if _, err := svc.GatewayGet(context.Background(), "s260", ""); err == nil {
		t.Fatalf("expected error for empty md5")
	}
}

func TestMtPhotoHandlers_Core(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/api-album":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "a", "cover": "c", "count": 2},
			})
			return
		case "/api-album/filesV2/1":
			if r.URL.Query().Get("listVer") != "v2" {
				t.Fatalf("listVer=%q, want v2", r.URL.Query().Get("listVer"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{
					{
						"day":  "2026-01-01",
						"addr": "",
						"list": []map[string]any{
							{"id": 1, "fileType": "JPEG", "MD5": "m1", "width": 10, "height": 20},
						},
					},
				},
				"totalCount": 1,
				"ver":        1,
			})
			return
		case "/gateway/s260/abc":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "max-age=1")
			_, _ = w.Write([]byte("thumb"))
			return
		case "/gateway/filesInMD5":
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 9, "filePath": "/lsp/a/b.jpg"},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)

	mt := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	app := &App{mtPhoto: mt}

	t.Run("handleGetMtPhotoAlbums not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbums", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoAlbums(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})

	t.Run("handleGetMtPhotoAlbums error", func(t *testing.T) {
		bad := &App{mtPhoto: NewMtPhotoService("", "", "", "", "/lsp", nil)}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbums", nil)
		rr := httptest.NewRecorder()
		bad.handleGetMtPhotoAlbums(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("handleGetMtPhotoAlbums ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbums", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoAlbums(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
	})

	t.Run("handleGetMtPhotoAlbumFiles not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbumFiles", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoAlbumFiles(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})

	t.Run("handleGetMtPhotoAlbumFiles error", func(t *testing.T) {
		bad := &App{mtPhoto: NewMtPhotoService("", "", "", "", "/lsp", nil)}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbumFiles?albumId=1", nil)
		rr := httptest.NewRecorder()
		bad.handleGetMtPhotoAlbumFiles(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("handleGetMtPhotoAlbumFiles ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbumFiles?albumId=1&page=1&pageSize=1", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoAlbumFiles(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
	})

	t.Run("handleGetMtPhotoThumb not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoThumb?size=s260&md5=abc", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoThumb(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("handleGetMtPhotoThumb missing params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoThumb", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoThumb(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("handleGetMtPhotoThumb size not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoThumb?size=bad&md5=abc", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoThumb(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("handleGetMtPhotoThumb gateway error", func(t *testing.T) {
		bad := &App{mtPhoto: NewMtPhotoService("", "", "", "", "/lsp", nil)}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoThumb?size=s260&md5=abc", nil)
		rr := httptest.NewRecorder()
		bad.handleGetMtPhotoThumb(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("handleGetMtPhotoThumb ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoThumb?size=s260&md5=abc", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoThumb(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
		if strings.TrimSpace(rr.Header().Get("Content-Type")) != "image/jpeg" {
			t.Fatalf("content-type=%q", rr.Header().Get("Content-Type"))
		}
		if rr.Body.String() != "thumb" {
			t.Fatalf("body=%q", rr.Body.String())
		}
	})

	t.Run("handleResolveMtPhotoFilePath not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/resolveMtPhotoFilePath?md5=m1", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleResolveMtPhotoFilePath(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})

	t.Run("handleResolveMtPhotoFilePath error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/resolveMtPhotoFilePath?md5=", nil)
		rr := httptest.NewRecorder()
		app.handleResolveMtPhotoFilePath(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("handleResolveMtPhotoFilePath ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/resolveMtPhotoFilePath?md5=m1", nil)
		rr := httptest.NewRecorder()
		app.handleResolveMtPhotoFilePath(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
	})

	// GatewayGet success: ensure URL is built and auth/cookie flow works.
	t.Run("GatewayGet ok", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		t.Cleanup(cancel)
		resp, err := mt.GatewayGet(ctx, "s260", "abc")
		if err != nil {
			t.Fatalf("GatewayGet error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	// keep url package used (avoids unused import in some build tags)
	_ = url.Values{}
}
