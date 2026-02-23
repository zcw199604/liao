package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func mustDouyinDetailPayload(t *testing.T, downloads []string, secUserID, coverURL string) string {
	t.Helper()
	data := map[string]any{
		"desc":      "t",
		"type":      "视频",
		"downloads": downloads,
	}
	if strings.TrimSpace(secUserID) != "" {
		data["sec_user_id"] = strings.TrimSpace(secUserID)
	}
	if strings.TrimSpace(coverURL) != "" {
		data["static_cover"] = strings.TrimSpace(coverURL)
	}
	payload := map[string]any{"message": "OK", "data": data}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return string(b)
}

func expectFavoriteAwemeUpdate(mock sqlmock.Sqlmock, secUserID, awemeID string) {
	mock.ExpectExec(`(?s)UPDATE douyin_favorite_user_aweme`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), secUserID, awemeID).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestHandleDouyinDownload_ExtraBranches(t *testing.T) {
	t.Run("cross-host redirect drops cookie", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/old.mp4"
		finalURL := "http://cdn.example.com/final.mp4"

		var firstCookie, finalCookie string
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case oldURL:
				firstCookie = strings.TrimSpace(r.Header.Get("Cookie"))
				h := make(http.Header)
				h.Set("Location", finalURL)
				return &http.Response{StatusCode: http.StatusFound, Status: "302 Found", Header: h, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case finalURL:
				finalCookie = strings.TrimSpace(r.Header.Get("Cookie"))
				h := make(http.Header)
				h.Set("Content-Type", "video/mp4")
				h.Set("Content-Length", "1")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("x")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-redirect", Title: "t", Downloads: []string{oldURL}})

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if firstCookie != "sid=abc" {
			t.Fatalf("first cookie=%q", firstCookie)
		}
		if finalCookie != "" {
			t.Fatalf("final cookie should be dropped, got=%q", finalCookie)
		}
	})

	t.Run("needCookie can be inferred from final host", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://cdn.example.com/a.mp4"
		finalURL := "http://www.douyin.com/protected.mp4"

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != oldURL {
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
			u, _ := url.Parse(finalURL)
			r2 := r.Clone(r.Context())
			r2.URL = u
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden")), Request: r2}, nil
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-final", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("cookie provider error branch", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "", "", 60*time.Second)
		svc.SetCookieProvider(cookieProviderFunc(func(ctx context.Context) (string, error) {
			return "", errors.New("cookie unavailable")
		}))
		oldURL := "http://www.douyin.com/video/no-cookie.mp4"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("boom")), Request: r}, nil
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-cookie", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("refresh error", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/refresh-error.mp4"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("refresh-fail")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-refresh-error", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "403") {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})

	t.Run("refresh no usable url for video", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL0 := "http://img.example.com/old.jpg"
		oldURL1 := "http://www.douyin.com/video/old.mp4"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-index")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{"http://img.example.com/new.jpg"}, "", "")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-no-url", Title: "t", Downloads: []string{oldURL0, oldURL1}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=1", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("refresh fallback first_video and persist then success", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL0 := "http://img.example.com/old.jpg"
		oldURL1 := "http://www.douyin.com/video/old.mp4"
		newURL := "http://www.douyin.com/video/new.mp4"
		newCover := "http://www.douyin.com/image/new.jpg"

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		expectFavoriteAwemeUpdate(mock, "sec-new", "aweme-new")

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{newURL}, "sec-new", newCover)
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == newURL:
				h := make(http.Header)
				h.Set("Content-Type", "video/mp4")
				h.Set("Content-Length", "1")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("x")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{
			douyinDownloader: svc,
			douyinFavorite:   NewDouyinFavoriteService(wrapMySQLDB(db)),
		}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "aweme-new", Title: "t", SecUserID: "", CoverURL: "", Downloads: []string{oldURL0, oldURL1}})

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=1", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}

		updated, ok := svc.GetCachedDetail(key)
		if !ok || updated == nil {
			t.Fatalf("cache miss")
		}
		if strings.TrimSpace(updated.Downloads[1]) != newURL {
			t.Fatalf("downloads=%v", updated.Downloads)
		}
		if strings.TrimSpace(updated.SecUserID) != "sec-new" {
			t.Fatalf("secUserID=%q", updated.SecUserID)
		}
		if strings.TrimSpace(updated.CoverURL) != newCover {
			t.Fatalf("cover=%q", updated.CoverURL)
		}
	})

	t.Run("refresh retry request error", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/old-retry-err.mp4"
		newURL := "http://www.douyin.com/video/new-retry-err.mp4"

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{newURL}, "", "")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == newURL:
				return nil, io.EOF
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-retry-err", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "EOF") {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})

	t.Run("refresh retry non-2xx", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/old-retry-502.mp4"
		newURL := "http://www.douyin.com/video/new-retry-502.mp4"

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{newURL}, "", "")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == newURL:
				h := make(http.Header)
				h.Set("Server", "mock-cdn")
				h.Set("Content-Type", "text/plain")
				return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Header: h, Body: io.NopCloser(strings.NewReader("still-bad")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-retry-502", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "still-bad") {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})
}

func TestHandleDouyinCover_ExtraBranches(t *testing.T) {
	t.Run("cookie provider error branch", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "", "", 60*time.Second)
		svc.SetCookieProvider(cookieProviderFunc(func(ctx context.Context) (string, error) {
			return "", errors.New("cookie unavailable")
		}))
		coverURL := "http://www.douyin.com/cover/cookie.jpg"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("boom")), Request: r}, nil
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "c1", Title: "t", CoverURL: coverURL, Downloads: []string{"http://x"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("refresh error", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldCover := "http://www.douyin.com/cover/old.jpg"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldCover:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("refresh-fail")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}
		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "c2", Title: "t", CoverURL: oldCover, Downloads: []string{"http://x"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("refresh success persist then retry error", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldCover := "http://www.douyin.com/cover/old-persist.jpg"
		newCover := "http://www.douyin.com/cover/new-persist.jpg"

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		expectFavoriteAwemeUpdate(mock, "sec-cover", "aweme-cover")

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.String() == oldCover:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{"http://www.douyin.com/video/v1.mp4"}, "sec-cover", newCover)
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == newCover:
				return nil, io.EOF
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc, douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "aweme-cover", Title: "t", SecUserID: "", CoverURL: oldCover, Downloads: []string{"http://x"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "EOF") {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})
}

func TestHandleDouyinDownloadHead_ExtraBranches(t *testing.T) {
	t.Run("HEAD forbidden refresh error", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/head-old.mp4"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("refresh-fail")), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("range-fail")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "h1", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("HEAD forbidden refresh no usable url", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL0 := "http://img.example.com/old.jpg"
		oldURL1 := "http://www.douyin.com/video/head-old2.mp4"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{"http://img.example.com/new.jpg"}, "", "")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("range-fail")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "h2", Title: "t", Downloads: []string{oldURL0, oldURL1}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=1", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("HEAD forbidden refresh success persist", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL0 := "http://img.example.com/old.jpg"
		oldURL1 := "http://www.douyin.com/video/head-old3.mp4"
		newURL := "http://www.douyin.com/video/head-new3.mp4"
		newCover := "http://www.douyin.com/image/head-new3.jpg"

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		expectFavoriteAwemeUpdate(mock, "sec-head", "aweme-head")

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{newURL}, "sec-head", newCover)
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodHead && r.URL.String() == newURL:
				h := make(http.Header)
				h.Set("Content-Type", "video/mp4")
				h.Set("Content-Length", "7")
				h.Set("Accept-Ranges", "bytes")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc, douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "aweme-head", Title: "t", SecUserID: "", CoverURL: "", Downloads: []string{oldURL0, oldURL1}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=1", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Content-Length") != "7" {
			t.Fatalf("headers=%v", rec.Header())
		}
	})

	t.Run("Range forbidden refresh error", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/range-old.mp4"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusMethodNotAllowed, Status: "405 Method Not Allowed", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == oldURL && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0":
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-range")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("refresh-fail")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "hr1", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Range forbidden refresh no usable url", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL0 := "http://img.example.com/old.jpg"
		oldURL1 := "http://www.douyin.com/video/range-old2.mp4"
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusMethodNotAllowed, Status: "405 Method Not Allowed", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == oldURL1 && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0":
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-range")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{"http://img.example.com/new.jpg"}, "", "")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "hr2", Title: "t", Downloads: []string{oldURL0, oldURL1}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=1", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Range forbidden refresh success persist", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL0 := "http://img.example.com/old.jpg"
		oldURL1 := "http://www.douyin.com/video/range-old3.mp4"
		newURL := "http://www.douyin.com/video/range-new3.mp4"
		newCover := "http://www.douyin.com/image/range-new3.jpg"

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		expectFavoriteAwemeUpdate(mock, "sec-range", "aweme-range")

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL1:
				return &http.Response{StatusCode: http.StatusMethodNotAllowed, Status: "405 Method Not Allowed", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == oldURL1 && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0":
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-range")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{newURL}, "sec-range", newCover)
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == newURL && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0":
				h := make(http.Header)
				h.Set("Content-Type", "video/mp4")
				h.Set("Accept-Ranges", "bytes")
				h.Set("Content-Range", "bytes 0-0/321")
				h.Set("Content-Length", "1")
				return &http.Response{StatusCode: http.StatusPartialContent, Status: "206 Partial Content", Header: h, Body: io.NopCloser(strings.NewReader("x")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc, douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "aweme-range", Title: "t", SecUserID: "", CoverURL: "", Downloads: []string{oldURL0, oldURL1}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=1", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Content-Length") != "321" {
			t.Fatalf("headers=%v", rec.Header())
		}
	})
}

func TestHandleDouyinDownload_RefreshFallbackSkipsBlankCandidate(t *testing.T) {
	svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
	oldURL := "http://www.douyin.com/video/old-empty.mp4"
	newURL := "http://www.douyin.com/video/new-empty.mp4"

	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldURL:
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			body := mustDouyinDetailPayload(t, []string{"", newURL}, "", "")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		case r.Method == http.MethodGet && r.URL.String() == newURL:
			h := make(http.Header)
			h.Set("Content-Type", "video/mp4")
			h.Set("Content-Length", "1")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("x")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	a := &App{douyinDownloader: svc}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d-empty-loop", Title: "t", Downloads: []string{oldURL}})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
	rec := httptest.NewRecorder()
	a.handleDouyinDownload(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleDouyinCover_RefreshReturnsEmptyCover(t *testing.T) {
	svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
	oldCover := "http://www.douyin.com/cover/old-empty.jpg"

	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldCover:
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-old")), Request: r}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			body := mustDouyinDetailPayload(t, []string{"http://www.douyin.com/video/v1.mp4"}, "", "")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	a := &App{douyinDownloader: svc}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "c-empty", Title: "t", CoverURL: oldCover, Downloads: []string{"http://x"}})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
	rec := httptest.NewRecorder()
	a.handleDouyinCover(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "403") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleDouyinDownloadHead_InvalidRefreshedURLTriggersBuildError(t *testing.T) {
	svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
	oldURL0 := "http://img.example.com/old.jpg"
	oldURL1 := "http://www.douyin.com/video/head-invalid-old.mp4"

	svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodHead && r.URL.String() == oldURL1:
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			body := mustDouyinDetailPayload(t, []string{"http://img.example.com/new.jpg", "://bad-url"}, "", "")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	a := &App{douyinDownloader: svc}
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "h-invalid", Title: "t", Downloads: []string{oldURL0, oldURL1}})
	req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=1", nil)
	rec := httptest.NewRecorder()
	a.handleDouyinDownload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "下载失败") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleDouyinDownloadHead_CookieProviderErrorAndRangeRedirectDropsCookie(t *testing.T) {
	t.Run("cookie provider error on head", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "", "", 60*time.Second)
		svc.SetCookieProvider(cookieProviderFunc(func(ctx context.Context) (string, error) {
			return "", errors.New("cookie unavailable")
		}))
		oldURL := "http://www.douyin.com/video/head-cookie-err.mp4"

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusMethodNotAllowed, Status: "405 Method Not Allowed", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == oldURL && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0":
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("range-fail")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "head-cookie-err", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("range redirect cross-host drops cookie", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/head-range-redirect.mp4"
		finalURL := "http://cdn.example.com/final.mp4"

		var firstCookie, finalCookie string
		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.String() {
			case oldURL:
				if r.Method == http.MethodHead {
					return &http.Response{StatusCode: http.StatusMethodNotAllowed, Status: "405 Method Not Allowed", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
				}
				if r.Method == http.MethodGet && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0" {
					firstCookie = strings.TrimSpace(r.Header.Get("Cookie"))
					h := make(http.Header)
					h.Set("Location", finalURL)
					return &http.Response{StatusCode: http.StatusFound, Status: "302 Found", Header: h, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
				}
			case finalURL:
				finalCookie = strings.TrimSpace(r.Header.Get("Cookie"))
				h := make(http.Header)
				h.Set("Content-Type", "video/mp4")
				h.Set("Accept-Ranges", "bytes")
				h.Set("Content-Range", "bytes 0-0/11")
				h.Set("Content-Length", "1")
				return &http.Response{StatusCode: http.StatusPartialContent, Status: "206 Partial Content", Header: h, Body: io.NopCloser(strings.NewReader("x")), Request: r}, nil
			}
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "head-range-redirect", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if firstCookie != "sid=abc" {
			t.Fatalf("first cookie=%q", firstCookie)
		}
		if finalCookie != "" {
			t.Fatalf("final cookie should be dropped, got=%q", finalCookie)
		}
	})
}

func TestHandleDouyinDownloadHead_RefreshFallbackSkipsBlankCandidate(t *testing.T) {
	t.Run("head 403 refresh fallback first_video skips blank", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/head-fallback-empty.mp4"
		newURL := "http://www.douyin.com/video/head-fallback-new.mp4"

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{"", newURL}, "", "")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodHead && r.URL.String() == newURL:
				h := make(http.Header)
				h.Set("Content-Type", "video/mp4")
				h.Set("Content-Length", "3")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "head-fallback-empty", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("range 403 refresh fallback first_video skips blank", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", 60*time.Second)
		oldURL := "http://www.douyin.com/video/range-fallback-empty.mp4"
		newURL := "http://www.douyin.com/video/range-fallback-new.mp4"

		svc.api.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodHead && r.URL.String() == oldURL:
				return &http.Response{StatusCode: http.StatusMethodNotAllowed, Status: "405 Method Not Allowed", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == oldURL && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0":
				return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden-range")), Request: r}, nil
			case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
				body := mustDouyinDetailPayload(t, []string{"", newURL}, "", "")
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			case r.Method == http.MethodGet && r.URL.String() == newURL && strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-0":
				h := make(http.Header)
				h.Set("Content-Type", "video/mp4")
				h.Set("Accept-Ranges", "bytes")
				h.Set("Content-Range", "bytes 0-0/66")
				h.Set("Content-Length", "1")
				return &http.Response{StatusCode: http.StatusPartialContent, Status: "206 Partial Content", Header: h, Body: io.NopCloser(strings.NewReader("x")), Request: r}, nil
			default:
				return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
			}
		})}

		a := &App{douyinDownloader: svc}
		key := svc.CacheDetail(&douyinCachedDetail{DetailID: "range-fallback-empty", Title: "t", Downloads: []string{oldURL}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
