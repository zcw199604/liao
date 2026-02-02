package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

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
	if parseFFprobeFrameRate("") != 0 {
		t.Fatalf("expected 0")
	}
	got := parseFFprobeFrameRate("30000/1001")
	if got < 29.9 || got > 30.1 {
		t.Fatalf("got=%v, want about 29.97", got)
	}
	if parseFFprobeFrameRate(" 30 ") != 30 {
		t.Fatalf("expected 30")
	}
	if parseFFprobeFrameRate("0/1") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("1/0") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("1/-1") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("nan") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("inf") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("1/2/3") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("x/1") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("inf/1") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeFrameRate("bad") != 0 {
		t.Fatalf("expected 0")
	}
}

func TestParseFFprobeDuration(t *testing.T) {
	if parseFFprobeDuration("") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeDuration("bad") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeDuration("-1") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeDuration("nan") != 0 {
		t.Fatalf("expected 0")
	}
	if parseFFprobeDuration("inf") != 0 {
		t.Fatalf("expected 0")
	}
	if got := parseFFprobeDuration("1.25"); got < 1.24 || got > 1.26 {
		t.Fatalf("got=%v", got)
	}
}

func TestNullIntIfNilAndNullFloatIfZero(t *testing.T) {
	if nullIntIfNil(nil) != nil {
		t.Fatalf("expected nil")
	}
	i := 1
	if nullIntIfNil(&i) != 1 {
		t.Fatalf("expected 1")
	}

	if nullFloatIfZero(0) != nil {
		t.Fatalf("expected nil")
	}
	if nullFloatIfZero(-1) != nil {
		t.Fatalf("expected nil")
	}
	if nullFloatIfZero(0.1) != 0.1 {
		t.Fatalf("expected 0.1")
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

func TestResolveUploadAbsPath_MoreBranches(t *testing.T) {
	tmp := t.TempDir()
	fileStore := &FileStorageService{baseUploadAbs: tmp, baseTempAbs: filepath.Join(t.TempDir(), "temp")}
	svc := &VideoExtractService{fileStore: fileStore, cfg: config.Config{ServerPort: 8080}}

	if _, err := svc.resolveUploadAbsPath(" "); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("adds leading slash", func(t *testing.T) {
		videosDir := filepath.Join(tmp, "videos")
		if err := os.MkdirAll(videosDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		target := filepath.Join(videosDir, "x.mp4")
		if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		got, err := svc.resolveUploadAbsPath("videos/x.mp4")
		if err != nil || got != target {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})

	t.Run("temp path requires filestore", func(t *testing.T) {
		svc2 := &VideoExtractService{fileStore: nil}
		if _, err := svc2.resolveUploadAbsPath("/tmp/video_extract_inputs/2026/01/21/a.mp4"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("temp base defaults to os.TempDir when empty", func(t *testing.T) {
		baseTempAbs := filepath.Join(os.TempDir(), "video_extract_inputs")
		t.Cleanup(func() { _ = os.RemoveAll(baseTempAbs) })

		target := filepath.Join(baseTempAbs, "2026", "01", "21", "b.mp4")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		svc2 := &VideoExtractService{fileStore: &FileStorageService{baseUploadAbs: tmp, baseTempAbs: ""}}
		got, err := svc2.resolveUploadAbsPath("/tmp/video_extract_inputs/2026/01/21/b.mp4")
		if err != nil || got != target {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})

	t.Run("temp inner clean '.' rejected", func(t *testing.T) {
		if _, err := svc.resolveUploadAbsPath("/tmp/video_extract_inputs/"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("temp filepathRelFn error", func(t *testing.T) {
		old := filepathRelFn
		filepathRelFn = func(string, string) (string, error) { return "", errors.New("rel err") }
		t.Cleanup(func() { filepathRelFn = old })

		if _, err := svc.resolveUploadAbsPath("/tmp/video_extract_inputs/2026/01/21/c.mp4"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("temp file not found", func(t *testing.T) {
		if _, err := svc.resolveUploadAbsPath("/tmp/video_extract_inputs/2026/01/21/missing.mp4"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("upload clean '.' rejected", func(t *testing.T) {
		if _, err := svc.resolveUploadAbsPath("/"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("upload filepathRelFn error", func(t *testing.T) {
		old := filepathRelFn
		filepathRelFn = func(string, string) (string, error) { return "", errors.New("rel err") }
		t.Cleanup(func() { filepathRelFn = old })

		if _, err := svc.resolveUploadAbsPath("/videos/a.mp4"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("upload file not found", func(t *testing.T) {
		if _, err := svc.resolveUploadAbsPath("/videos/missing.mp4"); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestVideoExtractService_CreateTask_ValidatesRequest(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	fileStore := &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: filepath.Join(t.TempDir(), "temp")}
	svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: fileStore, cfg: config.Config{FFprobePath: "ffprobe"}}

	if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
		SourceType: VideoExtractSourceUpload,
		LocalPath:  "/x.mp4",
		Mode:       VideoExtractModeAll,
		MaxFrames:  0,
	}); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
		SourceType: VideoExtractSourceType("bad"),
		Mode:       VideoExtractModeAll,
		MaxFrames:  1,
	}); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
		SourceType: VideoExtractSourceUpload,
		Mode:       VideoExtractMode("bad"),
		MaxFrames:  1,
	}); err == nil {
		t.Fatalf("expected error")
	}

	if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
		SourceType: VideoExtractSourceUpload,
		Mode:       VideoExtractModeFPS,
		MaxFrames:  1,
	}); err == nil {
		t.Fatalf("expected error")
	}

	start := -1.0
	if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
		SourceType: VideoExtractSourceUpload,
		Mode:       VideoExtractModeAll,
		StartSec:   &start,
		MaxFrames:  1,
	}); err == nil {
		t.Fatalf("expected error")
	}

	end := 1.0
	start = 2.0
	if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
		SourceType: VideoExtractSourceUpload,
		Mode:       VideoExtractModeAll,
		StartSec:   &start,
		EndSec:     &end,
		MaxFrames:  1,
	}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_CreateTask_Success_WithFakeFFprobe(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadRoot := t.TempDir()
	inputAbs := filepath.Join(uploadRoot, "in.mp4")
	if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ffprobePath := filepath.Join(t.TempDir(), "ffprobe")
	script := `#!/bin/sh
echo '{"streams":[{"width":1920,"height":1080,"avg_frame_rate":"30/1"}],"format":{"duration":"10.0"}}'
`
	if err := os.WriteFile(ffprobePath, []byte(script), 0o755); err != nil {
		t.Fatalf("write ffprobe: %v", err)
	}

	fileStore := &FileStorageService{baseUploadAbs: uploadRoot, baseTempAbs: filepath.Join(t.TempDir(), "temp")}
	closing := make(chan struct{})
	close(closing)

	svc := &VideoExtractService{
		db:        wrapMySQLDB(db),
		cfg:       config.Config{FFprobePath: ffprobePath},
		fileStore: fileStore,
		queue:     make(chan string, 1),
		closing:   closing,
		runtimes:  make(map[string]*videoExtractRuntime),
	}

	mock.ExpectExec(`(?s)INSERT INTO video_extract_task`).
		WithArgs(
			sqlmock.AnyArg(),
			"u1",
			"upload",
			"/in.mp4",
			inputAbs,
			sqlmock.AnyArg(),
			"jpg",
			nil,
			"all",
			nil,
			nil,
			nil,
			nil,
			nil,
			5,
			0,
			1920,
			1080,
			10.0,
			nil,
			"PENDING",
			nil,
			nil,
			nil,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	taskID, probe, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
		UserID:     "u1",
		SourceType: VideoExtractSourceUpload,
		LocalPath:  "/in.mp4",
		Mode:       VideoExtractModeAll,
		MaxFrames:  5,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if taskID == "" {
		t.Fatalf("expected taskID")
	}
	if probe.Width != 1920 || probe.Height != 1080 {
		t.Fatalf("probe=%+v", probe)
	}

	framesDir := filepath.Join(uploadRoot, "extract", taskID, "frames")
	if fi, err := os.Stat(framesDir); err != nil || !fi.IsDir() {
		t.Fatalf("frames dir not found: %v", err)
	}
}
