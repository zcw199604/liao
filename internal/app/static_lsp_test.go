package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpaHandler_RejectsNonGetHead(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", "INDEX_OK")

	a := &App{staticDir: staticDir}
	h := a.spaHandler()

	req := httptest.NewRequest(http.MethodPost, "http://example.com/login", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestSpaHandler_ServesIndexWhenNoExtEvenWithoutHTMLAccept(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", "INDEX_OK")

	a := &App{staticDir: staticDir}
	h := a.spaHandler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/route", nil)
	req.Header.Set("Accept", "*/*")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "INDEX_OK") {
		t.Fatalf("body=%q, want index", rr.Body.String())
	}
}

func TestSpaHandler_HeadStaticFile(t *testing.T) {
	staticDir := t.TempDir()
	writeTestFile(t, staticDir, "index.html", "INDEX_OK")
	writeTestFile(t, staticDir, "assets/app.js", "APP_OK")

	a := &App{staticDir: staticDir}
	h := a.spaHandler()

	req := httptest.NewRequest(http.MethodHead, "http://example.com/assets/app.js", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("head body should be empty, got=%q", rr.Body.String())
	}
}

func TestResolveLspLocalPath_CoversGuards(t *testing.T) {
	tmp := t.TempDir()

	if _, err := resolveLspLocalPath(tmp, "/not-lsp/a.txt"); err == nil {
		t.Fatalf("expected error for unsupported prefix")
	}
	if _, err := resolveLspLocalPath(tmp, "/lsp"); err == nil {
		t.Fatalf("expected error for directory access")
	}
	if _, err := resolveLspLocalPath(tmp, "/lsp/..foo"); err == nil {
		t.Fatalf("expected error for traversal-like path")
	}

	got, err := resolveLspLocalPath("", "/lsp/a.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != filepath.Join(string(filepath.Separator), "lsp", "a.txt") {
		t.Fatalf("got=%q, want %q", got, filepath.Join(string(filepath.Separator), "lsp", "a.txt"))
	}
}

func TestResolveLspLocalPath_FilepathRelError(t *testing.T) {
	old := filepathRelFn
	filepathRelFn = func(base, target string) (string, error) {
		return "", fmt.Errorf("boom")
	}
	t.Cleanup(func() { filepathRelFn = old })

	if _, err := resolveLspLocalPath("/tmp", "/lsp/a.txt"); err == nil || !strings.Contains(err.Error(), "路径解析失败") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestResolveLspLocalPath_OutOfBoundsGuards(t *testing.T) {
	old := filepathRelFn
	t.Cleanup(func() { filepathRelFn = old })

	filepathRelFn = func(base, target string) (string, error) { return ".", nil }
	if _, err := resolveLspLocalPath("/tmp", "/lsp/a.txt"); err == nil || !strings.Contains(err.Error(), "检测到路径越界") {
		t.Fatalf("expected out-of-bounds error (.), got %v", err)
	}

	filepathRelFn = func(base, target string) (string, error) { return ".." + string(filepath.Separator) + "x", nil }
	if _, err := resolveLspLocalPath("/tmp", "/lsp/a.txt"); err == nil || !strings.Contains(err.Error(), "检测到路径越界") {
		t.Fatalf("expected out-of-bounds error (../x), got %v", err)
	}

	filepathRelFn = func(base, target string) (string, error) { return "..", nil }
	if _, err := resolveLspLocalPath("/tmp", "/lsp/a.txt"); err == nil || !strings.Contains(err.Error(), "检测到路径越界") {
		t.Fatalf("expected out-of-bounds error (..), got %v", err)
	}
}

func TestLspFileServer_ServesFile(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "a", "b.txt")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("OK"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	a := &App{}
	a.cfg.LspRoot = root
	h := a.lspFileServer()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/lsp/a/b.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "OK" {
		t.Fatalf("body=%q, want %q", rr.Body.String(), "OK")
	}
}

func TestLspFileServer_ServesEscapedSpaceFilenameFallback(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "a", "name%20%20with-space.txt")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("SPACE_OK"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	a := &App{}
	a.cfg.LspRoot = root
	h := a.lspFileServer()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/lsp/a/name%20%20with-space.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d, body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Body.String() != "SPACE_OK" {
		t.Fatalf("body=%q, want %q", rr.Body.String(), "SPACE_OK")
	}
}

func TestLspFileServer_ServesEscapedHashFilename(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "a", "name#tag #topic.mp4")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("HASH_OK"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	a := &App{}
	a.cfg.LspRoot = root
	h := a.lspFileServer()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/lsp/a/name%23tag%20%23topic.mp4", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d, body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}
	if rr.Body.String() != "HASH_OK" {
		t.Fatalf("body=%q, want %q", rr.Body.String(), "HASH_OK")
	}
}

func TestLspPathSpaceVariantsAndDecodeBranches(t *testing.T) {
	variants := lspRequestPathSpaceVariants("/lsp/a/name with space.txt")
	if len(variants) != 2 {
		t.Fatalf("variants=%v, want original plus escaped-space variant", variants)
	}
	if variants[1] != "/lsp/a/name%20with%20space.txt" {
		t.Fatalf("escaped variant=%q", variants[1])
	}

	escaped := lspRequestPathSpaceVariants("/lsp/a/name%20with%20space.txt")
	if len(escaped) != 2 {
		t.Fatalf("escaped variants=%v, want original plus decoded-space variant", escaped)
	}
	if escaped[1] != "/lsp/a/name with space.txt" {
		t.Fatalf("decoded variant=%q", escaped[1])
	}

	if got := decodeEscapedSpacesOnly("/lsp/a/%2F%2x%20%2"); got != "/lsp/a/%2F%2x %2" {
		t.Fatalf("decodeEscapedSpacesOnly=%q", got)
	}
	if got := decodeEscapedSpacesOnly("/lsp/a/plain.txt"); got != "/lsp/a/plain.txt" {
		t.Fatalf("decode without percent=%q", got)
	}
}

func TestLspFileServer_NotFoundPaths(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	a := &App{}
	a.cfg.LspRoot = root
	h := a.lspFileServer()

	cases := []string{
		"/not-lsp/a.txt",       // resolve error
		"/lsp/missing.txt",     // stat error
		"/lsp/d",               // stat ok but dir
		"/lsp/..foo",           // traversal-like
		"/lsp/",                // directory access
		"/lsp",                 // directory access
		"/lsp/d/",              // directory access after clean
		"/lsp/d/..foo.txt",     // traversal-like segment
		"/lsp/d/../missing",    // clean to /missing inside root, stat error
		"/lsp/d/..foo/missing", // traversal-like prefix /.. triggers
	}
	for _, p := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://example.com"+p, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("path=%q status=%d, want %d", p, rr.Code, http.StatusNotFound)
		}
	}
}
