package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleDownloadImgUpload_Success(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path != "/img/Upload/2026/01/a.jpg" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("abc"))
	}))
	defer upstream.Close()

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	host := u.Hostname()
	port := u.Port()

	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	sys := NewSystemConfigService(wrapMySQLDB(db))
	sys.loaded = true
	sys.cached = SystemConfig{
		ImagePortMode:         ImagePortModeFixed,
		ImagePortFixed:        port,
		ImagePortRealMinBytes: defaultSystemConfig.ImagePortRealMinBytes,
	}

	a := &App{
		httpClient:   upstream.Client(),
		systemConfig: sys,
		imageServer:  NewImageServerService(host, port),
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/a.jpg", nil)
	rec := httptest.NewRecorder()
	a.handleDownloadImgUpload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/img/Upload/2026/01/a.jpg" {
		t.Fatalf("upstream path=%q", gotPath)
	}
	if ct := strings.TrimSpace(rec.Header().Get("Content-Type")); ct != "image/jpeg" {
		t.Fatalf("Content-Type=%q", ct)
	}
	cd := strings.TrimSpace(rec.Header().Get("Content-Disposition"))
	if !strings.Contains(cd, "attachment;") || !strings.Contains(cd, "filename*=") {
		t.Fatalf("Content-Disposition=%q", cd)
	}
	if rec.Body.String() != "abc" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestHandleDownloadImgUpload_InvalidPath(t *testing.T) {
	a := &App{}
	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=../a.jpg", nil)
	rec := httptest.NewRecorder()
	a.handleDownloadImgUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
	got := decodeJSONBody(t, rec.Body)
	if got["error"].(string) != "path 非法" {
		t.Fatalf("got=%v", got)
	}
}

func TestHandleDownloadImgUpload_UpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("nope"))
	}))
	defer upstream.Close()

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	host := u.Hostname()
	port := u.Port()

	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	sys := NewSystemConfigService(wrapMySQLDB(db))
	sys.loaded = true
	sys.cached = SystemConfig{
		ImagePortMode:         ImagePortModeFixed,
		ImagePortFixed:        port,
		ImagePortRealMinBytes: defaultSystemConfig.ImagePortRealMinBytes,
	}

	a := &App{
		httpClient:   upstream.Client(),
		systemConfig: sys,
		imageServer:  NewImageServerService(host, port),
	}

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/a.jpg", nil)
	rec := httptest.NewRecorder()
	a.handleDownloadImgUpload(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d, want 502", rec.Code)
	}
	got := decodeJSONBody(t, rec.Body)
	if !strings.Contains(got["error"].(string), "404") {
		t.Fatalf("got=%v", got)
	}
}
