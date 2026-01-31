package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNormalizeRotationDegrees(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 0},
		{90, 90},
		{180, 180},
		{270, 270},
		{360, 0},
		{-90, 270},
		{10, 0},
		{50, 90},
		{181, 180},
		{359, 0},
	}
	for _, tc := range cases {
		if got := normalizeRotationDegrees(tc.in); got != tc.want {
			t.Fatalf("normalizeRotationDegrees(%d)=%d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestBuildVideoPosterLocalPath(t *testing.T) {
	if got := buildVideoPosterLocalPath(""); got != "" {
		t.Fatalf("empty: got=%q", got)
	}
	if got := buildVideoPosterLocalPath("videos/a.mp4"); got != "/videos/a.poster.jpg" {
		t.Fatalf("got=%q, want %q", got, "/videos/a.poster.jpg")
	}
	if got := buildVideoPosterLocalPath("/videos/a.mp4?token=1"); got != "/videos/a.poster.jpg" {
		t.Fatalf("query: got=%q, want %q", got, "/videos/a.poster.jpg")
	}
	if got := buildVideoPosterLocalPath("http://x/upload/videos/a.mp4"); got != "/videos/a.poster.jpg" {
		t.Fatalf("url: got=%q, want %q", got, "/videos/a.poster.jpg")
	}

	// cover the name empty branch (e.g. basename is just extension).
	if got := buildVideoPosterLocalPath("/videos/.mp4"); got == "" {
		t.Fatalf("dot-only name should still produce a poster path")
	}
}

func TestFileStorageService_resolveUploadAbsPath_and_posterURLFromLocalPath(t *testing.T) {
	root := t.TempDir()
	s := &FileStorageService{baseUploadAbs: root}

	if _, err := (*FileStorageService)(nil).resolveUploadAbsPath("/videos/a.mp4"); err == nil {
		t.Fatalf("expected error for nil service")
	}
	if _, err := s.resolveUploadAbsPath(""); err == nil {
		t.Fatalf("expected error for empty localPath")
	}
	if _, err := s.resolveUploadAbsPath("/"); err == nil {
		t.Fatalf("expected error for illegal localPath")
	}
	if _, err := s.resolveUploadAbsPath("/../x.mp4"); err == nil {
		t.Fatalf("expected error for path traversal")
	}

	got, err := s.resolveUploadAbsPath("/videos/a.mp4")
	if err != nil {
		t.Fatalf("resolveUploadAbsPath: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "/videos/a.mp4") {
		t.Fatalf("got=%q", got)
	}

	if got := s.posterURLFromLocalPath("videos/a.poster.jpg"); got != "/upload/videos/a.poster.jpg" {
		t.Fatalf("posterURL=%q, want %q", got, "/upload/videos/a.poster.jpg")
	}
	if got := s.posterURLFromLocalPath(""); got != "" {
		t.Fatalf("empty posterURL=%q, want empty", got)
	}
}

func TestProbeVideoRotationDegrees_FastAndFallback(t *testing.T) {
	uploadRoot := t.TempDir()
	inputAbs := filepath.Join(uploadRoot, "videos", "a.mp4")
	if err := os.MkdirAll(filepath.Dir(inputAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// 1) fast path: rotate tag
	ffprobeFast := writeExecutable(t, "ffprobe-fast", "#!/bin/sh\necho 90\n")
	if got := probeVideoRotationDegrees(context.Background(), ffprobeFast, inputAbs); got != 90 {
		t.Fatalf("fast rotate=%d, want %d", got, 90)
	}

	// 2) fallback: no rotate tag, parse JSON side_data_list.rotation
	ffprobeFallback := writeExecutable(t, "ffprobe-fallback", `#!/bin/sh
args="$*"
case "$args" in
  *stream_tags=rotate*) exit 0 ;;
  *"-of json"*) echo '{"streams":[{"tags":{},"side_data_list":[{"rotation":-90}]}]}' ; exit 0 ;;
esac
exit 1
`)
	if got := probeVideoRotationDegrees(context.Background(), ffprobeFallback, inputAbs); got != 270 {
		t.Fatalf("fallback rotate=%d, want %d", got, 270)
	}

	// empty inputs => 0
	if got := probeVideoRotationDegrees(context.Background(), "", inputAbs); got != 0 {
		t.Fatalf("empty ffprobePath rotate=%d, want 0", got)
	}
	if got := probeVideoRotationDegrees(context.Background(), ffprobeFast, ""); got != 0 {
		t.Fatalf("empty inputAbs rotate=%d, want 0", got)
	}
}

func TestFileStorageService_EnsureVideoPoster_EarlyAndSuccess(t *testing.T) {
	uploadRoot := t.TempDir()
	s := &FileStorageService{baseUploadAbs: uploadRoot}
	ctx := context.Background()

	// ffmpeg not configured => skip (no error)
	if poster, err := s.EnsureVideoPoster(ctx, "   ", "ffprobe", "/videos/a.mp4", false); err != nil || poster != "" {
		t.Fatalf("skip: poster=%q err=%v", poster, err)
	}

	// missing video file => error
	if _, err := s.EnsureVideoPoster(ctx, "ffmpeg", "ffprobe", "/videos/missing.mp4", false); err == nil {
		t.Fatalf("expected error for missing video")
	}

	// prepare a fake video file
	videoLocalPath := "/videos/2026/01/30/a.mp4"
	videoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(videoLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(videoAbs, []byte("not-a-real-mp4"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	posterLocalPath := buildVideoPosterLocalPath(videoLocalPath)
	posterAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(posterLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(posterAbs), 0o755); err != nil {
		t.Fatalf("mkdir poster dir: %v", err)
	}
	if err := os.WriteFile(posterAbs, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write poster: %v", err)
	}

	// poster already exists and not forced => fast return without calling ffmpeg.
	if got, err := s.EnsureVideoPoster(ctx, "ffmpeg-not-needed", "ffprobe", videoLocalPath, false); err != nil || got != posterLocalPath {
		t.Fatalf("existing poster: got=%q err=%v", got, err)
	}

	// force regenerate: use fake ffmpeg that fails once then succeeds.
	marker := filepath.Join(t.TempDir(), "marker")
	ffmpeg := writeExecutable(t, "ffmpeg-flaky", fmt.Sprintf(`#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
if test ! -f %q; then
  echo 1 > %q
  exit 1
fi
echo poster > "$out"
exit 0
`, marker, marker))

	// fake ffprobe fast path output "90" => rotateFilter != ""
	ffprobe := writeExecutable(t, "ffprobe-rotate", "#!/bin/sh\necho 90\n")

	// remove existing poster to ensure it needs regeneration.
	_ = os.Remove(posterAbs)
	got, err := s.EnsureVideoPoster(ctx, ffmpeg, ffprobe, videoLocalPath, true)
	if err != nil {
		t.Fatalf("EnsureVideoPoster: %v", err)
	}
	if got != posterLocalPath {
		t.Fatalf("posterLocalPath=%q, want %q", got, posterLocalPath)
	}
	if fi, err := os.Stat(posterAbs); err != nil || fi.Size() == 0 {
		t.Fatalf("poster not created: %v err=%v", fi, err)
	}
}

func TestFileStorageService_DeleteVideoPoster(t *testing.T) {
	uploadRoot := t.TempDir()
	s := &FileStorageService{baseUploadAbs: uploadRoot}

	if ok := (*FileStorageService)(nil).DeleteVideoPoster("/videos/a.mp4"); ok {
		t.Fatalf("nil service should return false")
	}
	if ok := s.DeleteVideoPoster(""); ok {
		t.Fatalf("empty path should return false")
	}

	videoLocalPath := "/videos/2026/01/30/a.mp4"
	posterLocalPath := buildVideoPosterLocalPath(videoLocalPath)
	posterAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(posterLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(posterAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(posterAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if ok := s.DeleteVideoPoster(videoLocalPath); !ok {
		t.Fatalf("expected DeleteVideoPoster to succeed")
	}
	if _, err := os.Stat(posterAbs); err == nil {
		t.Fatalf("poster should be deleted")
	}
}

func TestFileStorageService_EnsureVideoPosterLogged(t *testing.T) {
	uploadRoot := t.TempDir()
	s := &FileStorageService{baseUploadAbs: uploadRoot}

	// error path => returns empty strings
	if lp, url := s.EnsureVideoPosterLogged(context.Background(), "ffmpeg", "ffprobe", "/videos/missing.mp4", false); lp != "" || url != "" {
		t.Fatalf("lp=%q url=%q, want empty", lp, url)
	}

	// success path: use fake ffmpeg and existing poster to avoid invoking ffmpeg.
	videoLocalPath := "/videos/2026/01/30/a.mp4"
	videoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(videoLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(videoAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	posterLocalPath := buildVideoPosterLocalPath(videoLocalPath)
	posterAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(posterLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(posterAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(posterAbs, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write poster: %v", err)
	}

	lp, url := s.EnsureVideoPosterLogged(context.Background(), "ffmpeg", "ffprobe", videoLocalPath, false)
	if lp != posterLocalPath {
		t.Fatalf("lp=%q, want %q", lp, posterLocalPath)
	}
	if url != "/upload"+posterLocalPath {
		t.Fatalf("url=%q, want %q", url, "/upload"+posterLocalPath)
	}
}

func TestMediaUploadService_RepairVideoPosters_Validation(t *testing.T) {
	// Ensure video_poster_repair.go validation branches are covered without requiring ffmpeg/ffprobe.
	svc := &MediaUploadService{db: mustNewSQLMockDB(t), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	t.Cleanup(func() { _ = svc.db.Close() })

	if _, err := svc.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{Commit: false, Source: "bad"}); err == nil {
		t.Fatalf("expected error for invalid source")
	}
	if _, err := svc.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{Commit: false, Limit: -1}); err == nil {
		t.Fatalf("expected error for negative limit")
	}
}

func TestMediaUploadService_RepairVideoPosters_LimitClampAndDryRun(t *testing.T) {
	uploadRoot := t.TempDir()
	videoLocalPath := "/videos/2026/01/30/a.mp4"
	videoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(videoLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(videoAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// Limit=0 => default 200; queryLimit = limit+1 => 201.
	mock.ExpectQuery(`FROM media_file`).
		WithArgs(int64(0), 201).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(1), videoLocalPath, "video/mp4", "mp4"))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: uploadRoot}}
	res, err := svc.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{
		Commit: false,
		Limit:  0,
	})
	if err != nil {
		t.Fatalf("RepairVideoPosters: %v", err)
	}
	if res.Limit != 200 {
		t.Fatalf("limit=%d, want %d", res.Limit, 200)
	}
	if res.PosterMissing != 1 || res.PosterGenerated != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
