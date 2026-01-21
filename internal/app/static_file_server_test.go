package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadFileServer_ServesFile(t *testing.T) {
	tempDir := t.TempDir()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	full := filepath.Join(tempDir, "upload", "images", "2026", "01", "10", "a.txt")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	a := &App{}
	h := a.uploadFileServer()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/upload/images/2026/01/10/a.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("body=%q, want %q", rr.Body.String(), "ok")
	}
}

func TestUploadFileServer_PathTraversalBlocked(t *testing.T) {
	tempDir := t.TempDir()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	// 即便 upload 目录为空，也应拒绝通过 .. 访问工作目录外文件
	a := &App{}
	h := a.uploadFileServer()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/upload/../go.mod", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Fatalf("expected traversal blocked, got 200")
	}
}

func TestUploadFileServer_ServesTempVideoExtractInput(t *testing.T) {
	tempDir := t.TempDir()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	tempBaseAbs := filepath.Join(tempDir, "temp_inputs")
	full := filepath.Join(tempBaseAbs, "2026", "01", "21", "a.txt")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	a := &App{
		fileStorage: &FileStorageService{baseTempAbs: tempBaseAbs},
	}
	h := a.uploadFileServer()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/upload/tmp/video_extract_inputs/2026/01/21/a.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("body=%q, want %q", rr.Body.String(), "ok")
	}
}
