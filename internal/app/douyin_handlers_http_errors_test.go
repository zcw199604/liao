package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleDouyinAccount_ErrorsAndDefaults(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		var a *App
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		a.handleDouyinAccount(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", bytes.NewBufferString(`{"input":"x"}`))
		rec := httptest.NewRecorder()
		a.handleDouyinAccount(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("decode error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", bytes.NewBufferString(`not-json`))
		rec := httptest.NewRecorder()
		a.handleDouyinAccount(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("input empty", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", bytes.NewBufferString(`{"input":" "}`))
		rec := httptest.NewRecorder()
		a.handleDouyinAccount(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("resolve error", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		t.Cleanup(upstream.Close)

		a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", bytes.NewBufferString(`{"input":"x","cursor":0,"count":1}`))
		rec := httptest.NewRecorder()
		a.handleDouyinAccount(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("fetch error contains resolved", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/share":
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": "https://www.douyin.com/user/u1"})
			case "/douyin/account":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("boom"))
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(upstream.Close)

		a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", bytes.NewBufferString(`{"input":"short","cursor":0,"count":1}`))
		rec := httptest.NewRecorder()
		a.handleDouyinAccount(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		body := decodeJSONBody(t, rec.Body)
		if !strings.Contains(body["error"].(string), "resolved=") {
			t.Fatalf("body=%v", body)
		}
	})

	t.Run("success defaults and fallback fields", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/share":
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": "https://www.douyin.com/user/u1"})
			case "/douyin/account":
				var payload map[string]any
				_ = json.NewDecoder(r.Body).Decode(&payload)
				if int(payload["cursor"].(float64)) != 0 {
					t.Fatalf("cursor=%v", payload["cursor"])
				}
				if int(payload["count"].(float64)) != 18 {
					t.Fatalf("count=%v", payload["count"])
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": "ok",
					"data": map[string]any{
						"cursor":     0,
						"max_cursor": 99,
						"has_more":   0,
						"hasMore":    1,
						"aweme_list": []any{},
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(upstream.Close)

		a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", bytes.NewBufferString(`{"input":"short","cursor":-1,"count":0}`))
		rec := httptest.NewRecorder()
		a.handleDouyinAccount(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var resp douyinAccountResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Tab != "post" || resp.Cursor != 99 || !resp.HasMore || resp.Items == nil || len(resp.Items) != 0 {
			t.Fatalf("resp=%+v", resp)
		}
	})
}

func TestHandleDouyinDetail_ErrorsAndBranches(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		var a *App
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":"x"}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("decode error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`not-json`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("input empty", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":" "}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("resolve error", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		t.Cleanup(upstream.Close)

		a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":"x"}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("fetch error contains resolved", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/share":
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": "https://www.douyin.com/video/123"})
			case "/douyin/detail":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("boom"))
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(upstream.Close)

		a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":"short"}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		body := decodeJSONBody(t, rec.Body)
		if !strings.Contains(body["error"].(string), "resolved=") {
			t.Fatalf("body=%v", body)
		}
	})

	t.Run("cache failure", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/share":
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": "https://www.douyin.com/video/123"})
			case "/douyin/detail":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": "ok",
					"data": map[string]any{
						"desc":      "t",
						"type":      "视频",
						"downloads": "http://example.com/v.mp4",
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(upstream.Close)

		svc := NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)
		svc.cache = nil
		a := &App{douyinDownloader: svc}

		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":"short"}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("success image type and default ext", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/share":
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": "https://www.douyin.com/video/123"})
			case "/douyin/detail":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": "ok",
					"data": map[string]any{
						"desc":      "t",
						"type":      "图集",
						"downloads": []any{"http://example.com/noext", " "},
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(upstream.Close)

		a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":"short"}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var resp douyinDetailResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Type != "图集" || len(resp.Items) != 1 || resp.Items[0].Type != "image" || !strings.HasSuffix(resp.Items[0].OriginalFilename, ".jpg") {
			t.Fatalf("resp=%+v", resp)
		}
	})

	t.Run("success video default ext", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/share":
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": "https://www.douyin.com/video/123"})
			case "/douyin/detail":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": "ok",
					"data": map[string]any{
						"desc":      "t",
						"type":      "视频",
						"downloads": []any{"http://example.com/noext"},
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(upstream.Close)

		a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":"short"}`))
		rec := httptest.NewRecorder()
		a.handleDouyinDetail(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var resp douyinDetailResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Type != "视频" || len(resp.Items) != 1 || resp.Items[0].Type != "video" || !strings.HasSuffix(resp.Items[0].OriginalFilename, ".mp4") {
			t.Fatalf("resp=%+v", resp)
		}
	})
}
