package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMtPhotoService_GetAlbums_UsesAPIKeyHeader(t *testing.T) {
	var albumCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api-album":
			atomic.AddInt32(&albumCalls, 1)
			if got := r.Header.Get("x-api-key"); got != "api-key-1" {
				t.Fatalf("x-api-key=%q, want api-key-1", got)
			}
			if got := r.Header.Get("jwt"); got != "" {
				t.Fatalf("jwt header should not be sent, got %q", got)
			}
			if got := r.Header.Get("Cookie"); got != "" {
				t.Fatalf("Cookie header should not be sent, got %q", got)
			}
			if atomic.LoadInt32(&albumCalls) > 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "a", "cover": "m", "count": 2},
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "api-key-1", "/tmp", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	albums, err := svc.GetAlbums(ctx)
	if err != nil {
		t.Fatalf("GetAlbums error: %v", err)
	}
	if len(albums) != 1 || albums[0].ID != 1 {
		t.Fatalf("albums=%v, want single id=1", albums)
	}

	if atomic.LoadInt32(&albumCalls) != 1 {
		t.Fatalf("albumCalls=%d, want 1", albumCalls)
	}
}

func TestMtPhotoService_GetAlbumFilesPage_CacheAndPaginate(t *testing.T) {
	var filesCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/api-album/filesV2/3":
			atomic.AddInt32(&filesCalls, 1)
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
							{"id": 2, "fileType": "MP4", "MD5": "m2", "width": 11, "height": 21, "duration": 1.2},
							{"id": 3, "fileType": "JPEG", "MD5": "m3", "width": 12, "height": 22},
						},
					},
				},
				"totalCount": 3,
				"ver":        2,
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "api-key-1", "/tmp", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	items1, total, pages, err := svc.GetAlbumFilesPage(ctx, 3, 1, 2)
	if err != nil {
		t.Fatalf("GetAlbumFilesPage error: %v", err)
	}
	if total != 3 || pages != 2 {
		t.Fatalf("total/pages=%d/%d, want 3/2", total, pages)
	}
	if len(items1) != 2 || items1[0].MD5 != "m1" || items1[1].MD5 != "m2" {
		t.Fatalf("items1=%v, want [m1 m2]", items1)
	}
	if items1[1].Type != "video" {
		t.Fatalf("items1[1].Type=%q, want video", items1[1].Type)
	}

	items2, total2, pages2, err := svc.GetAlbumFilesPage(ctx, 3, 2, 2)
	if err != nil {
		t.Fatalf("GetAlbumFilesPage(page2) error: %v", err)
	}
	if total2 != 3 || pages2 != 2 {
		t.Fatalf("total/pages=%d/%d, want 3/2", total2, pages2)
	}
	if len(items2) != 1 || items2[0].MD5 != "m3" {
		t.Fatalf("items2=%v, want [m3]", items2)
	}

	if atomic.LoadInt32(&filesCalls) != 1 {
		t.Fatalf("filesCalls=%d, want 1 (cached)", filesCalls)
	}
}

func TestMtPhotoService_GatewayGet_RefreshesAuthCodeOn401(t *testing.T) {
	var authCodeCalls int32
	var gatewayCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			n := atomic.AddInt32(&authCodeCalls, 1)
			authCode := "ac-1"
			if n >= 2 {
				authCode = "ac-2"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"auth_code":  authCode,
				"expires_in": time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/s260/m":
			atomic.AddInt32(&gatewayCalls, 1)
			if got := r.Header.Get("x-api-key"); got != "api-key-1" {
				t.Fatalf("x-api-key=%q, want api-key-1", got)
			}
			if got := r.Header.Get("jwt"); got != "" {
				t.Fatalf("jwt header should not be sent, got %q", got)
			}
			if got := r.Header.Get("Cookie"); got != "" {
				t.Fatalf("Cookie header should not be sent, got %q", got)
			}
			switch r.URL.Query().Get("auth_code") {
			case "ac-1":
				w.WriteHeader(http.StatusUnauthorized)
				return
			case "ac-2":
				_, _ = w.Write([]byte("ok"))
				return
			default:
				t.Fatalf("auth_code query=%q", r.URL.Query().Get("auth_code"))
			}
		default:
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "api-key-1", "/tmp", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	resp, err := svc.GatewayGet(ctx, "s260", "m")
	if err != nil {
		t.Fatalf("GatewayGet error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
	if atomic.LoadInt32(&authCodeCalls) != 2 {
		t.Fatalf("authCodeCalls=%d, want 2", authCodeCalls)
	}
	if atomic.LoadInt32(&gatewayCalls) != 2 {
		t.Fatalf("gatewayCalls=%d, want 2", gatewayCalls)
	}
}
