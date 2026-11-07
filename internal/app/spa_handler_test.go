package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSpaHandler_CachePolicyAndMissingAssets(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", `<script type="module" src="/assets/index-abcd1234.js"></script>`)
	writeTestFile(t, staticDir, "assets/index-abcd1234.js", "APP")
	writeTestFile(t, staticDir, "favicon.svg", "ICON")
	h := (&App{staticDir: staticDir}).spaHandler()
	for _, tc := range []struct {
		path, cache string
		status      int
	}{
		{"/", "no-cache", 200},
		{"/index.html", "no-cache", 200},
		{"/chat/u1", "no-cache", 200},
		{"/assets/index-abcd1234.js", "public, max-age=31536000, immutable", 200},
		{"/favicon.svg", "no-cache", 200},
		{"/assets/missing.js", "no-store", 404},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept", "text/html")
			if tc.cache == "no-store" {
				req.Header.Set("If-Modified-Since", "Wed, 01 Jan 2031 00:00:00 GMT")
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tc.status || rr.Header().Get("Cache-Control") != tc.cache {
				t.Fatalf("status=%d cache=%q", rr.Code, rr.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestSpaHandler_HTMLVersionRevalidation(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", "VERSION_A")
	indexPath := filepath.Join(staticDir, "index.html")
	stamp := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(indexPath, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	h := (&App{staticDir: staticDir}).spaHandler()
	request := func(method, path, etag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if etag != "" {
			r.Header.Set("If-None-Match", etag)
		}
		// 不能因文件时间未变化或客户端时钟偏差误判新版 HTML 未更新。
		r.Header.Set("If-Modified-Since", "Wed, 01 Jan 2031 00:00:00 GMT")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, r)
		return rr
	}
	first := request(http.MethodGet, "/", "")
	etag := first.Header().Get("ETag")
	if first.Code != http.StatusOK || etag == "" || first.Body.String() != "VERSION_A" {
		t.Fatalf("first status=%d etag=%q body=%s", first.Code, etag, first.Body.String())
	}
	for _, path := range []string{"/", "/index.html", "/chat/u1"} {
		cached := request(http.MethodGet, path, etag)
		if cached.Code != http.StatusNotModified || cached.Body.Len() != 0 || cached.Header().Get("ETag") != etag || cached.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("cached %s: status=%d headers=%v body=%s", path, cached.Code, cached.Header(), cached.Body.String())
		}
	}
	writeTestFile(t, staticDir, "index.html", "VERSION_B")
	if err := os.Chtimes(indexPath, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	updated := request(http.MethodGet, "/chat/u1", etag)
	if updated.Code != http.StatusOK || updated.Body.String() != "VERSION_B" || updated.Header().Get("ETag") == etag {
		t.Fatalf("changed content kept old version: status=%d body=%s", updated.Code, updated.Body.String())
	}
	head := request(http.MethodHead, "/", updated.Header().Get("ETag"))
	if head.Code != http.StatusNotModified || head.Body.Len() != 0 {
		t.Fatalf("HEAD status=%d body=%s", head.Code, head.Body.String())
	}
}

func TestSpaHandler_VersionTracksIndexContent(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", `<script type="module" crossorigin src="/assets/index-abcd1234.js"></script>`)
	h := (&App{staticDir: staticDir}).spaHandler()
	readVersion := func() map[string]string {
		t.Helper()
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/version.json", nil))
		if rr.Code != 200 || rr.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("response=%+v", rr)
		}
		var result map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	first := readVersion()
	if first["version"] == "" || first["entry"] != "/assets/index-abcd1234.js" {
		t.Fatalf("version=%v", first)
	}
	if readVersion()["version"] != first["version"] {
		t.Fatal("unchanged index changed version")
	}
	writeTestFile(t, staticDir, "index.html", `<script type="module" src="/assets/index-new12345.js"></script>`)
	if readVersion()["version"] == first["version"] {
		t.Fatal("changed index kept old version")
	}
}

func TestSpaHandler_HTMLRemainsInsideStaticDirectory(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	writeTestFile(t, staticDir, "index.html", "INDEX")
	writeTestFile(t, root, "outside.html", "PRIVATE")
	rr := httptest.NewRecorder()
	(&App{staticDir: staticDir}).spaHandler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/../outside.html", nil))
	if strings.Contains(rr.Body.String(), "PRIVATE") {
		t.Fatal("served HTML outside static directory")
	}
}

func writeTestFile(t *testing.T, dir, relativePath, content string) {
	t.Helper()

	fullPath := filepath.Join(dir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func TestSpaHandler_ServesIndexForRoutes(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", "INDEX_OK")
	writeTestFile(t, staticDir, "assets/app.js", "APP_OK")

	a := &App{staticDir: staticDir}
	h := a.spaHandler()

	cases := []string{
		"/",
		"/login",
		"/identity",
		"/list",
		"/list/",
		"/chat",
		"/chat/u1",
		"/chat/u1/",
		"/new-route",
		"/new/route/deep",
	}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://example.com"+path, nil)
		req.Header.Set("Accept", "text/html")
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d, want %d", path, rr.Code, http.StatusOK)
		}
		if !strings.Contains(rr.Body.String(), "INDEX_OK") {
			t.Fatalf("path=%s body=%q, want index", path, rr.Body.String())
		}
	}
}

func TestSpaHandler_ServesExistingStaticFile(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", "INDEX_OK")
	writeTestFile(t, staticDir, "assets/app.js", "APP_OK")

	a := &App{staticDir: staticDir}
	h := a.spaHandler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/assets/app.js", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "APP_OK" {
		t.Fatalf("body=%q, want %q", rr.Body.String(), "APP_OK")
	}
}

func TestSpaHandler_MissingAssetReturns404(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", "INDEX_OK")

	a := &App{staticDir: staticDir}
	h := a.spaHandler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/assets/missing.js", nil)
	req.Header.Set("Accept", "*/*")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
	}
}
