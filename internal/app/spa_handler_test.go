package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
