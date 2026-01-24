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
