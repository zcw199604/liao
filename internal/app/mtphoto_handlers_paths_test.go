package app

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"liao/internal/config"
)

func TestMtPhotoPathNormalizationAndMD5Helpers(t *testing.T) {
	if got := normalizeMtPhotoLookupLocalPath(" "); got != "" {
		t.Fatalf("empty normalize should return empty, got %q", got)
	}

	rawURL := "https://example.com/upload/images/%E6%B5%8B%E8%AF%95.jpg?x=1#hash"
	if got := normalizeMtPhotoLookupLocalPath(rawURL); got != "/upload/images/测试.jpg" {
		t.Fatalf("normalized url=%q", got)
	}
	if got := normalizeMtPhotoLookupLocalPath("upload\\images\\a.jpg"); got != "/upload/images/a.jpg" {
		t.Fatalf("normalized windows path=%q", got)
	}

	if _, err := calculateMD5FromAbsPath(""); err == nil {
		t.Fatalf("expected error for empty abs path")
	}
	if _, err := calculateMD5FromAbsPath(t.TempDir()); err == nil {
		t.Fatalf("expected error for directory path")
	}

	file := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(file, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	md5Value, err := calculateMD5FromAbsPath(file)
	if err != nil {
		t.Fatalf("calculateMD5FromAbsPath error: %v", err)
	}
	expected := md5.Sum([]byte("abc"))
	if md5Value != hex.EncodeToString(expected[:]) {
		t.Fatalf("md5=%q", md5Value)
	}
}

func TestApp_calculateMD5FromSupportedLocalPath_AndResolveMD5(t *testing.T) {
	if _, err := (*App)(nil).calculateMD5FromSupportedLocalPath("/lsp/a.jpg"); err == nil {
		t.Fatalf("nil app should fail")
	}

	app := &App{}
	if _, err := app.calculateMD5FromSupportedLocalPath(" "); err == nil {
		t.Fatalf("blank path should fail")
	}

	lspRoot := t.TempDir()
	lspFile := filepath.Join(lspRoot, "a", "b.txt")
	if err := os.MkdirAll(filepath.Dir(lspFile), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(lspFile, []byte("lsp-data"), 0o644); err != nil {
		t.Fatalf("write lsp file failed: %v", err)
	}

	uploadRoot := t.TempDir()
	uploadLocalPath := "/images/2026/03/u.txt"
	uploadAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(uploadLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(uploadAbs), 0o755); err != nil {
		t.Fatalf("mkdir upload dir failed: %v", err)
	}
	if err := os.WriteFile(uploadAbs, []byte("upload-data"), 0o644); err != nil {
		t.Fatalf("write upload file failed: %v", err)
	}

	app = &App{
		cfg:         config.Config{LspRoot: lspRoot},
		fileStorage: &FileStorageService{baseUploadAbs: uploadRoot},
	}

	if _, err := app.calculateMD5FromSupportedLocalPath("/lsp/../../etc/passwd"); err == nil {
		t.Fatalf("invalid lsp path should fail")
	}

	lspMD5, err := app.calculateMD5FromSupportedLocalPath("/lsp/a/b.txt")
	if err != nil || strings.TrimSpace(lspMD5) == "" {
		t.Fatalf("lsp md5 failed: md5=%q err=%v", lspMD5, err)
	}

	if _, err := app.calculateMD5FromSupportedLocalPath("/tmp/other.txt"); err == nil {
		t.Fatalf("unsupported prefix should fail")
	}
	if _, err := app.calculateMD5FromSupportedLocalPath("/upload/docs/a.txt"); err == nil {
		t.Fatalf("unsupported upload subdir should fail")
	}

	appNoStorage := &App{cfg: config.Config{LspRoot: lspRoot}}
	if _, err := appNoStorage.calculateMD5FromSupportedLocalPath("/upload/images/2026/03/u.txt"); err == nil {
		t.Fatalf("missing fileStorage should fail")
	}

	uploadMD5, err := app.calculateMD5FromSupportedLocalPath("/upload/images/2026/03/u.txt")
	if err != nil || strings.TrimSpace(uploadMD5) == "" {
		t.Fatalf("upload md5 failed: md5=%q err=%v", uploadMD5, err)
	}

	if _, err := app.resolveMtPhotoSameMediaMD5("not-md5", ""); err == nil {
		t.Fatalf("invalid md5 should fail")
	}
	if got, err := app.resolveMtPhotoSameMediaMD5("A5D5C1E3A5D5C1E3A5D5C1E3A5D5C1E3", ""); err != nil || got != "a5d5c1e3a5d5c1e3a5d5c1e3a5d5c1e3" {
		t.Fatalf("resolve md5 direct failed: got=%q err=%v", got, err)
	}
	if _, err := app.resolveMtPhotoSameMediaMD5("", ""); err == nil {
		t.Fatalf("empty md5/localPath should fail")
	}
	if got, err := app.resolveMtPhotoSameMediaMD5("", "/upload/images/2026/03/u.txt"); err != nil || got != uploadMD5 {
		t.Fatalf("resolve by local path failed: got=%q err=%v", got, err)
	}
}
