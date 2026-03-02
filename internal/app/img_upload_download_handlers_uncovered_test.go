package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleDownloadImgUpload_UncoveredBranches(t *testing.T) {
	t.Run("client.Do error branch", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		sys := NewSystemConfigService(wrapMySQLDB(db))
		sys.loaded = true
		sys.cached = SystemConfig{
			ImagePortMode:         ImagePortModeFixed,
			ImagePortFixed:        "9006",
			ImagePortRealMinBytes: defaultSystemConfig.ImagePortRealMinBytes,
		}

		a := &App{
			httpClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("network down")
			})},
			systemConfig: sys,
			imageServer:  NewImageServerService("example.com", "9006"),
		}

		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=2026/01/a.jpg", nil)
		rr := httptest.NewRecorder()
		a.handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("filename fallback to download", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/img/Upload/." {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
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

		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/downloadImgUpload?path=.", nil)
		rr := httptest.NewRecorder()
		a.handleDownloadImgUpload(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}

		cd := strings.TrimSpace(rr.Header().Get("Content-Disposition"))
		if !strings.Contains(cd, "download.jpg") {
			t.Fatalf("content-disposition=%q", cd)
		}
	})
}

