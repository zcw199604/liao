package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"liao/internal/config"
)

func TestIntegrationRoutes_NotRegisteredWithoutToken(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("spa"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{cfg: config.Config{}, staticDir: staticDir}
	router := a.buildRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "spa") {
		t.Fatalf("integration path fell back to SPA")
	}
}

func TestIntegrationRoutes_MountMTAndRequireToken(t *testing.T) {
	const token = "0123456789abcdef01234567"
	a := &App{cfg: config.Config{IntegrationToken: token}}
	router := a.buildRouter()

	paths := []string{
		"/api/integration/v1/media",
		"/api/integration/v1/mt/albums",
		"/api/integration/v1/mt/album-files",
		"/api/integration/v1/mt/folders/root",
		"/api/integration/v1/mt/folders/content",
		"/api/integration/v1/mt/folders/breadcrumbs",
		"/api/integration/v1/mt/favorites",
		"/api/integration/v1/mt/thumb",
		"/api/integration/v1/mt/file",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s status=%d, want 401", path, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/mt/albums", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500 from uninitialized mtPhoto", rec.Code)
	}
}
