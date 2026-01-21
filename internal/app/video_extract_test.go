package app

import (
	"os"
	"path/filepath"
	"testing"

	"liao/internal/config"
)

func TestIsHexMD5(t *testing.T) {
	if !isHexMD5("0123456789abcdef0123456789ABCDEF") {
		t.Fatalf("expected true")
	}
	if isHexMD5("not-md5") {
		t.Fatalf("expected false")
	}
	if isHexMD5("") {
		t.Fatalf("expected false")
	}
}

func TestParseFFprobeFrameRate(t *testing.T) {
	got := parseFFprobeFrameRate("30000/1001")
	if got < 29.9 || got > 30.1 {
		t.Fatalf("got=%v, want about 29.97", got)
	}
	if parseFFprobeFrameRate("0/1") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("bad") != 0 {
		t.Fatalf("expected 0")
	}
}

func TestResolveUploadAbsPath(t *testing.T) {
	tmp := t.TempDir()
	tempBase := filepath.Join(tmp, "temp")
	fileStore := &FileStorageService{baseUploadAbs: tmp, baseTempAbs: tempBase}
	svc := &VideoExtractService{fileStore: fileStore, cfg: config.Config{ServerPort: 8080}}

	if _, err := svc.resolveUploadAbsPath("/../../etc/passwd"); err == nil {
		t.Fatalf("expected error for traversal")
	}

	videosDir := filepath.Join(tmp, "videos")
	if err := os.MkdirAll(videosDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	target := filepath.Join(videosDir, "a.mp4")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := svc.resolveUploadAbsPath("/videos/a.mp4")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got != target {
		t.Fatalf("got=%q, want %q", got, target)
	}

	// 临时输入视频：/tmp/video_extract_inputs/...
	if _, err := svc.resolveUploadAbsPath("/tmp/video_extract_inputs/../../etc/passwd"); err == nil {
		t.Fatalf("expected error for temp traversal")
	}

	tempTarget := filepath.Join(tempBase, "2026", "01", "21", "b.mp4")
	if err := os.MkdirAll(filepath.Dir(tempTarget), 0o755); err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	if err := os.WriteFile(tempTarget, []byte("y"), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	gotTemp, err := svc.resolveUploadAbsPath("/tmp/video_extract_inputs/2026/01/21/b.mp4")
	if err != nil {
		t.Fatalf("resolve temp failed: %v", err)
	}
	if gotTemp != tempTarget {
		t.Fatalf("got=%q, want %q", gotTemp, tempTarget)
	}
}
