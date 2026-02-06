package app

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleDouyinDownload_ErrorBranches(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		var a *App
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key=x&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key=x&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("bad key/index", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key=&index=x", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("missing cached", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key=missing&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("index out of range", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", Downloads: []string{"http://example.com/a"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=9", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("remote url empty", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", Downloads: []string{" "}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("request build error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", Downloads: []string{"http://[::1"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("do error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, io.EOF
		})}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", Downloads: []string{"http://example.com/a"}})

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("upstream status error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		})}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", Downloads: []string{"http://example.com/a"}})

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinDownload(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandleDouyinDownload_HeaderAndExtFallbacks(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		h := make(http.Header)
		body := "x"
		switch {
		case strings.Contains(r.URL.String(), "ct"):
			h.Set("Content-Type", "video/mp4")
		case strings.Contains(r.URL.String(), ".jpg"):
			// no content-type
		default:
			// no content-type and no ext
		}
		h.Set("Accept-Ranges", "bytes")
		h.Set("Content-Range", "bytes 0-0/1")
		h.Set("Content-Length", "1")
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     h,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})}

	cases := []struct {
		name      string
		remoteURL string
		wantCT    string
		wantExt   string
	}{
		{name: "from content-type", remoteURL: "http://example.com/ct", wantCT: "video/mp4", wantExt: ".mp4"},
		{name: "from url", remoteURL: "http://example.com/a.jpg", wantCT: "application/octet-stream", wantExt: ".jpg"},
		{name: "fallback bin", remoteURL: "http://example.com/noext", wantCT: "application/octet-stream", wantExt: ".bin"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", Downloads: []string{tc.remoteURL}})
			req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
			rec := httptest.NewRecorder()
			a.handleDouyinDownload(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if ct := strings.TrimSpace(rec.Header().Get("Content-Type")); ct != tc.wantCT {
				t.Fatalf("content-type=%q", ct)
			}
			cd := rec.Header().Get("Content-Disposition")
			if !strings.Contains(cd, tc.wantExt) {
				t.Fatalf("cd=%q", cd)
			}
			if rec.Header().Get("Accept-Ranges") != "bytes" {
				t.Fatalf("ar=%q", rec.Header().Get("Accept-Ranges"))
			}
			if rec.Header().Get("Content-Range") == "" || rec.Header().Get("Content-Length") != "1" {
				t.Fatalf("hdr=%v", rec.Header())
			}
		})
	}
}

func TestHandleDouyinDownload_ForbiddenAutoRefresh(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}

	oldURL := "http://media.example.com/old.mp4"
	newURL := "http://media.example.com/new.mp4"
	key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "123", Title: "t", Type: "视频", Downloads: []string{oldURL}})

	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldURL:
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("<html>403</html>")),
			}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			// Simulate TikTokDownloader /douyin/detail response providing a refreshed download URL.
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"message":"OK","data":{"desc":"t","type":"视频","static_cover":"","downloads":["` + newURL + `"]}}`,
				)),
			}, nil
		case r.Method == http.MethodGet && r.URL.String() == newURL:
			h := make(http.Header)
			h.Set("Content-Type", "video/mp4")
			h.Set("Accept-Ranges", "bytes")
			h.Set("Content-Length", "1")
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     h,
				Body:       io.NopCloser(strings.NewReader("x")),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("unexpected")),
			}, nil
		}
	})}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
	rec := httptest.NewRecorder()
	a.handleDouyinDownload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Cache should be updated so subsequent requests reuse the refreshed URL list.
	got, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || got == nil || len(got.Downloads) != 1 || strings.TrimSpace(got.Downloads[0]) != newURL {
		t.Fatalf("cache not refreshed: ok=%v got=%v", ok, got)
	}
}

func TestHandleDouyinCover_ErrorBranches(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		var a *App
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key=x", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key=x", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("bad key", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key=", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("missing cached", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key=missing", nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("cover empty", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", CoverURL: " ", Downloads: []string{"x"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("method fallback to GET", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatalf("method=%q", r.Method)
			}
			h := make(http.Header)
			h.Set("Content-Type", "image/jpeg")
			h.Set("Content-Length", "1")
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     h,
				Body:       io.NopCloser(strings.NewReader("x")),
			}, nil
		})}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", CoverURL: "http://example.com/c.jpg", Downloads: []string{"x"}})
		req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("request build error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", CoverURL: "http://[::1", Downloads: []string{"x"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("do error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, io.EOF
		})}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", CoverURL: "http://example.com/c.jpg", Downloads: []string{"x"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("status error", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		})}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", CoverURL: "http://example.com/c.jpg", Downloads: []string{"x"}})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("head no body", func(t *testing.T) {
		a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)}
		a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("Content-Type", "image/jpeg")
			h.Set("Content-Length", "1")
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     h,
				Body:       io.NopCloser(strings.NewReader("x")),
			}, nil
		})}
		key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", CoverURL: "http://example.com/c.jpg", Downloads: []string{"x"}})
		req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/cover?key="+key, nil)
		rec := httptest.NewRecorder()
		a.handleDouyinCover(rec, req)
		if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.Bytes())
		}
	})
}

func TestHandleDouyinCover_ForbiddenRefreshSuccess(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://upstream.local", "", "", "", time.Second)}

	oldCover := "http://media.example.com/cover-old.jpg"
	newCover := "http://media.example.com/cover-new.jpg"
	key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "123", Title: "t", Type: "视频", CoverURL: oldCover, Downloads: []string{"http://media.example.com/v1.mp4"}})

	var oldCoverHits, detailHits, newCoverHits int
	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldCover:
			oldCoverHits++
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("403-old")),
				Request:    r,
			}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			detailHits++
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"message":"OK","data":{"desc":"t","type":"视频","static_cover":"` + newCover + `","downloads":["http://media.example.com/v1.mp4"]}}`,
				)),
				Request: r,
			}, nil
		case r.Method == http.MethodGet && r.URL.String() == newCover:
			newCoverHits++
			h := make(http.Header)
			h.Set("Content-Type", "image/jpeg")
			h.Set("Content-Length", "1")
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     h,
				Body:       io.NopCloser(strings.NewReader("x")),
				Request:    r,
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("unexpected")),
				Request:    r,
			}, nil
		}
	})}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
	rec := httptest.NewRecorder()
	a.handleDouyinCover(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if oldCoverHits != 1 || detailHits != 1 || newCoverHits != 1 {
		t.Fatalf("hits old=%d detail=%d new=%d", oldCoverHits, detailHits, newCoverHits)
	}

	updated, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || updated == nil || strings.TrimSpace(updated.CoverURL) != newCover {
		t.Fatalf("cover not refreshed: ok=%v updated=%v", ok, updated)
	}
}

func TestHandleDouyinCover_ForbiddenRefreshRetryFailed(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://upstream.local", "", "", "", time.Second)}

	oldCover := "http://media.example.com/cover-old.jpg"
	newCover := "http://media.example.com/cover-new.jpg"
	key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "123", Title: "t", Type: "视频", CoverURL: oldCover, Downloads: []string{"http://media.example.com/v1.mp4"}})

	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.String() == oldCover:
			return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("403-old")), Request: r}, nil
		case r.Method == http.MethodPost && r.URL.Path == "/douyin/detail":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"message":"OK","data":{"desc":"t","type":"视频","static_cover":"` + newCover + `","downloads":["http://media.example.com/v1.mp4"]}}`,
				)),
				Request: r,
			}, nil
		case r.Method == http.MethodGet && r.URL.String() == newCover:
			return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("bad-gateway")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
	rec := httptest.NewRecorder()
	a.handleDouyinCover(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONBody(t, rec.Body)
	if !strings.Contains(body["error"].(string), "502 Bad Gateway") || !strings.Contains(body["error"].(string), "bad-gateway") {
		t.Fatalf("body=%v", body)
	}
}

func TestHandleDouyinCover_CrossHostRedirectDropsCookie(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://unused.local", "", "sid=abc", "", time.Second)}

	firstURL := "http://www.douyin.com/cover.jpg"
	finalURL := "http://cdn.example.com/final.jpg"
	key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "t", Type: "视频", CoverURL: firstURL, Downloads: []string{"http://x"}})

	var firstCookie, finalCookie string
	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case firstURL:
			firstCookie = strings.TrimSpace(r.Header.Get("Cookie"))
			h := make(http.Header)
			h.Set("Location", finalURL)
			return &http.Response{StatusCode: http.StatusFound, Status: "302 Found", Header: h, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		case finalURL:
			finalCookie = strings.TrimSpace(r.Header.Get("Cookie"))
			h := make(http.Header)
			h.Set("Content-Type", "image/jpeg")
			h.Set("Content-Length", "1")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("x")), Request: r}, nil
		default:
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected")), Request: r}, nil
		}
	})}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
	rec := httptest.NewRecorder()
	a.handleDouyinCover(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if firstCookie != "sid=abc" {
		t.Fatalf("first cookie=%q", firstCookie)
	}
	if finalCookie != "" {
		t.Fatalf("final cookie should be dropped, got %q", finalCookie)
	}
}
