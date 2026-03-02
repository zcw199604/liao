package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProbeVideoRotationDegrees_FallbackCommandErrorBranch(t *testing.T) {
	input := filepath.Join(t.TempDir(), "a.mp4")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input err=%v", err)
	}
	ffprobeFail := writeExecutable(t, "ffprobe-fail-both", "#!/bin/sh\nexit 1\n")
	if got := probeVideoRotationDegrees(context.Background(), ffprobeFail, input); got != 0 {
		t.Fatalf("rotate=%d", got)
	}
}

func TestEnsureVideoPoster_ResolvePosterAbsErrorBranch(t *testing.T) {
	uploadRoot := t.TempDir()
	videoAbs := filepath.Join(uploadRoot, "videos", "a.mp4")
	if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
		t.Fatalf("mkdir err=%v", err)
	}
	if err := os.WriteFile(videoAbs, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video err=%v", err)
	}

	oldRel := filepathRelFn
	t.Cleanup(func() { filepathRelFn = oldRel })
	filepathRelFn = func(basepath, targpath string) (string, error) {
		if strings.Contains(targpath, ".poster.") {
			return "", errors.New("poster rel failed")
		}
		return oldRel(basepath, targpath)
	}

	s := &FileStorageService{baseUploadAbs: uploadRoot}
	if _, err := s.EnsureVideoPoster(context.Background(), "ffmpeg", "", "/videos/a.mp4", true); err == nil {
		t.Fatalf("expected resolve poster abs error")
	}
}

func TestEnsureVideoPoster_Rotation180Branch(t *testing.T) {
	uploadRoot := t.TempDir()
	videoAbs := filepath.Join(uploadRoot, "videos", "r180.mp4")
	if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
		t.Fatalf("mkdir err=%v", err)
	}
	if err := os.WriteFile(videoAbs, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video err=%v", err)
	}

	ffprobe180 := writeExecutable(t, "ffprobe-180", "#!/bin/sh\necho 180\n")
	ffmpegCheck180 := writeExecutable(t, "ffmpeg-check-180", `#!/bin/sh
args="$*"
echo "$args" | grep 'transpose=2,transpose=2' >/dev/null || exit 3
out=""
for a in "$@"; do out="$a"; done
printf 'jpg' > "$out"
exit 0
`)

	s := &FileStorageService{baseUploadAbs: uploadRoot}
	posterLocalPath, err := s.EnsureVideoPoster(context.Background(), ffmpegCheck180, ffprobe180, "/videos/r180.mp4", true)
	if err != nil {
		t.Fatalf("EnsureVideoPoster err=%v", err)
	}
	if strings.TrimSpace(posterLocalPath) == "" || !strings.HasSuffix(posterLocalPath, ".poster.jpg") {
		t.Fatalf("posterLocalPath=%q", posterLocalPath)
	}
}
