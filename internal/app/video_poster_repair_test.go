package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_RepairVideoPosters_Local_GeneratesPoster(t *testing.T) {
	uploadRoot := t.TempDir()

	// Prepare a fake video file under upload/videos/...
	videoLocalPath := "/videos/2026/01/30/a.mp4"
	videoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(videoLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(videoAbs, []byte("not-a-real-mp4"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	// Fake ffmpeg: write to last argument.
	ffmpegOK := writeExecutable(t, "ffmpeg-ok", `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
echo poster > "$out"
exit 0
`)

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, local_path, file_type, file_extension\s+FROM media_file\s+WHERE .*id > \?\s+ORDER BY id ASC\s+LIMIT \?`).
		WithArgs(int64(0), 11).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(1), videoLocalPath, "video/mp4", "mp4"))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: uploadRoot}}
	res, err := svc.RepairVideoPosters(context.Background(), ffmpegOK, RepairVideoPostersRequest{
		Commit:       true,
		Source:       "local",
		StartAfterID: 0,
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("RepairVideoPosters: %v", err)
	}
	if res.Scanned != 1 || res.PosterGenerated != 1 || res.PosterMissing != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.HasMore {
		t.Fatalf("hasMore=true, want false")
	}

	posterLocalPath := buildVideoPosterLocalPath(videoLocalPath)
	posterAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(posterLocalPath, "/")))
	fi, err := os.Stat(posterAbs)
	if err != nil || fi.IsDir() || fi.Size() == 0 {
		t.Fatalf("poster not created: %s err=%v", posterAbs, err)
	}
}

func TestMediaUploadService_RepairVideoPosters_CommitRequiresFFmpeg(t *testing.T) {
	db := mustNewSQLMockDB(t)
	t.Cleanup(func() { _ = db.Close() })
	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	if _, err := svc.RepairVideoPosters(context.Background(), "   ", RepairVideoPostersRequest{Commit: true}); err == nil {
		t.Fatalf("expected error")
	}
}
