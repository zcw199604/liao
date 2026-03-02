package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestVideoPoster_AdditionalBranches(t *testing.T) {
	t.Run("resolveUploadAbsPath filepathRel error", func(t *testing.T) {
		s := &FileStorageService{baseUploadAbs: t.TempDir()}
		oldRel := filepathRelFn
		t.Cleanup(func() { filepathRelFn = oldRel })
		filepathRelFn = func(basepath, targpath string) (string, error) {
			return "", errors.New("rel failed")
		}
		if _, err := s.resolveUploadAbsPath("/videos/a.mp4"); err == nil {
			t.Fatalf("expected rel error")
		}
	})

	t.Run("probeVideoRotationDegrees fallback branches", func(t *testing.T) {
		input := filepath.Join(t.TempDir(), "a.mp4")
		_ = os.WriteFile(input, []byte("x"), 0o644)

		ffprobeBadJSON := writeExecutable(t, "ffprobe-badjson", `#!/bin/sh
case "$*" in
  *stream_tags=rotate*) exit 0 ;;
  *-show_streams*) echo "{" ; exit 0 ;;
esac
exit 1
`)
		if got := probeVideoRotationDegrees(context.Background(), ffprobeBadJSON, input); got != 0 {
			t.Fatalf("bad json rotate=%d", got)
		}

		ffprobeNoStreams := writeExecutable(t, "ffprobe-nostream", `#!/bin/sh
case "$*" in
  *stream_tags=rotate*) exit 0 ;;
  *-show_streams*) echo '{"streams":[]}' ; exit 0 ;;
esac
exit 1
`)
		if got := probeVideoRotationDegrees(context.Background(), ffprobeNoStreams, input); got != 0 {
			t.Fatalf("no streams rotate=%d", got)
		}

		ffprobeTagRotate := writeExecutable(t, "ffprobe-tagrotate", `#!/bin/sh
case "$*" in
  *stream_tags=rotate*) exit 0 ;;
  *-show_streams*) echo '{"streams":[{"tags":{"rotate":"180"},"side_data_list":[]}]}' ; exit 0 ;;
esac
exit 1
`)
		if got := probeVideoRotationDegrees(context.Background(), ffprobeTagRotate, input); got != 180 {
			t.Fatalf("tag rotate=%d", got)
		}

		ffprobeNoData := writeExecutable(t, "ffprobe-nodata", `#!/bin/sh
case "$*" in
  *stream_tags=rotate*) exit 0 ;;
  *-show_streams*) echo '{"streams":[{"tags":{},"side_data_list":[]}]}' ; exit 0 ;;
esac
exit 1
`)
		if got := probeVideoRotationDegrees(context.Background(), ffprobeNoData, input); got != 0 {
			t.Fatalf("no data rotate=%d", got)
		}
	})

	t.Run("EnsureVideoPoster nil and validation errors", func(t *testing.T) {
		var nilSvc *FileStorageService
		if _, err := nilSvc.EnsureVideoPoster(context.Background(), "ffmpeg", "ffprobe", "/videos/a.mp4", false); err == nil {
			t.Fatalf("nil service should fail")
		}

		s := &FileStorageService{baseUploadAbs: t.TempDir()}
		if _, err := s.EnsureVideoPoster(context.Background(), "ffmpeg", "ffprobe", " ", false); err == nil {
			t.Fatalf("empty local path should fail")
		}
		if _, err := s.EnsureVideoPoster(context.Background(), "ffmpeg", "ffprobe", "/../a.mp4", false); err == nil {
			t.Fatalf("traversal path should fail")
		}
	})

	t.Run("EnsureVideoPoster rotation 270 and no output branch", func(t *testing.T) {
		uploadRoot := t.TempDir()
		s := &FileStorageService{baseUploadAbs: uploadRoot}
		videoLocalPath := "/videos/r270.mp4"
		videoAbs := filepath.Join(uploadRoot, filepath.FromSlash("videos/r270.mp4"))
		if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
			t.Fatalf("mkdir err=%v", err)
		}
		if err := os.WriteFile(videoAbs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write video err=%v", err)
		}

		ffprobe270 := writeExecutable(t, "ffprobe-270", "#!/bin/sh\necho 270\n")
		ffmpegNoOutput := writeExecutable(t, "ffmpeg-no-output", "#!/bin/sh\nexit 0\n")
		if _, err := s.EnsureVideoPoster(context.Background(), ffmpegNoOutput, ffprobe270, videoLocalPath, true); err == nil {
			t.Fatalf("expected no output error")
		}
	})

	t.Run("EnsureVideoPoster mkdir failure and logged empty result", func(t *testing.T) {
		rootFile := filepath.Join(t.TempDir(), "upload-file")
		if err := os.WriteFile(rootFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write root file err=%v", err)
		}
		s := &FileStorageService{baseUploadAbs: rootFile}
		if _, err := s.EnsureVideoPoster(context.Background(), "ffmpeg", "ffprobe", "/videos/a.mp4", true); err == nil {
			t.Fatalf("expected mkdir failure")
		}

		s2 := &FileStorageService{baseUploadAbs: t.TempDir()}
		if lp, url := s2.EnsureVideoPosterLogged(context.Background(), "   ", "ffprobe", "/videos/a.mp4", false); lp != "" || url != "" {
			t.Fatalf("logged empty expected, got lp=%q url=%q", lp, url)
		}
	})
}
