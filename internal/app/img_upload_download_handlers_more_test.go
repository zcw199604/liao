package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleDownloadImgUpload_MoreBranches(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=a.jpg", nil)
		rr := httptest.NewRecorder()
		(*App)(nil).handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=%20", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("image server not configured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/a.jpg", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("client do error (nil httpClient -> default client)", func(t *testing.T) {
		a := &App{
			// invalid host forces default client.Do error path
			imageServer: NewImageServerService("invalid host", "9006"),
		}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/a.jpg", nil)
		rr := httptest.NewRecorder()
		a.handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("upstream non-2xx without body", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer upstream.Close()

		u, err := url.Parse(upstream.URL)
		if err != nil {
			t.Fatalf("parse url: %v", err)
		}

		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		sys := NewSystemConfigService(wrapMySQLDB(db))
		sys.loaded = true
		sys.cached = SystemConfig{
			ImagePortMode:         ImagePortModeFixed,
			ImagePortFixed:        u.Port(),
			ImagePortRealMinBytes: defaultSystemConfig.ImagePortRealMinBytes,
		}

		a := &App{
			httpClient:   upstream.Client(),
			systemConfig: sys,
			imageServer:  NewImageServerService(u.Hostname(), u.Port()),
		}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/a.jpg", nil)
		rr := httptest.NewRecorder()
		a.handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		got := decodeJSONBody(t, rr.Body)
		if !strings.Contains(got["error"].(string), "502 Bad Gateway") {
			t.Fatalf("got=%v", got)
		}
	})

	t.Run("filename without extension guesses ext from content-type", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Content-Length", "3")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("png"))
		}))
		defer upstream.Close()

		u, err := url.Parse(upstream.URL)
		if err != nil {
			t.Fatalf("parse url: %v", err)
		}

		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		sys := NewSystemConfigService(wrapMySQLDB(db))
		sys.loaded = true
		sys.cached = SystemConfig{
			ImagePortMode:         ImagePortModeFixed,
			ImagePortFixed:        u.Port(),
			ImagePortRealMinBytes: defaultSystemConfig.ImagePortRealMinBytes,
		}

		a := &App{
			httpClient:   upstream.Client(),
			systemConfig: sys,
			imageServer:  NewImageServerService(u.Hostname(), u.Port()),
		}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/noext", nil)
		rr := httptest.NewRecorder()
		a.handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		cd := rr.Header().Get("Content-Disposition")
		if !strings.Contains(cd, "noext.png") {
			t.Fatalf("content-disposition=%q", cd)
		}
		if strings.TrimSpace(rr.Header().Get("Content-Length")) != "3" {
			t.Fatalf("content-length=%q", rr.Header().Get("Content-Length"))
		}
	})
}
