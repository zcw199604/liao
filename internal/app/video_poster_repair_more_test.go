package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMediaUploadService_RepairVideoPosters_MixedBranchesAndHasMore(t *testing.T) {
	uploadRoot := t.TempDir()

	// existing poster case
	existingVideo := "/videos/existing.mp4"
	existingVideoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(existingVideo, "/")))
	if err := os.MkdirAll(filepath.Dir(existingVideoAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(existingVideoAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	existingPoster := buildVideoPosterLocalPath(existingVideo)
	existingPosterAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(existingPoster, "/")))
	if err := os.MkdirAll(filepath.Dir(existingPosterAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(existingPosterAbs, []byte("poster"), 0o644); err != nil {
		t.Fatalf("write poster: %v", err)
	}

	// poster missing (dry-run)
	needVideo := "/videos/need.mp4"
	needVideoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(needVideo, "/")))
	if err := os.MkdirAll(filepath.Dir(needVideoAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(needVideoAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// Limit=5 => queryLimit=6
	mock.ExpectQuery(`FROM media_file`).
		WithArgs(int64(0), 6).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(1), "   ", "video/mp4", "mp4").                 // localPath invalid -> skipped
			AddRow(int64(2), "/../x.mp4", "video/mp4", "mp4").           // resolve abs error -> videoMissing
			AddRow(int64(3), "/videos/missing.mp4", "video/mp4", "mp4"). // stat missing -> videoMissing
			AddRow(int64(4), existingVideo, "video/mp4", "mp4").         // poster existing
			AddRow(int64(5), needVideo, "video/mp4", "mp4").             // poster missing, dry-run
			AddRow(int64(6), "/videos/extra.mp4", "video/mp4", "mp4"))   // extra row => hasMore

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: uploadRoot}}
	res, err := svc.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{
		Commit: false,
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("RepairVideoPosters: %v", err)
	}

	if res.Scanned != 5 {
		t.Fatalf("scanned=%d, want %d", res.Scanned, 5)
	}
	if !res.HasMore {
		t.Fatalf("hasMore=false, want true")
	}
	if res.NextAfterID != 5 {
		t.Fatalf("nextAfterId=%d, want %d", res.NextAfterID, 5)
	}
	if res.Skipped != 1 {
		t.Fatalf("skipped=%d, want %d", res.Skipped, 1)
	}
	if res.VideoMissing != 2 {
		t.Fatalf("videoMissing=%d, want %d", res.VideoMissing, 2)
	}
	if res.PosterExisting != 1 {
		t.Fatalf("posterExisting=%d, want %d", res.PosterExisting, 1)
	}
	if res.PosterMissing != 1 {
		t.Fatalf("posterMissing=%d, want %d", res.PosterMissing, 1)
	}
	if res.PosterGenerated != 0 || res.PosterFailed != 0 {
		t.Fatalf("unexpected poster counters: %+v", res)
	}
	if len(res.Warnings) == 0 {
		t.Fatalf("warnings should not be empty")
	}
}

func TestMediaUploadService_RepairVideoPosters_Local_PosterGenerationFailed(t *testing.T) {
	uploadRoot := t.TempDir()

	videoLocalPath := "/videos/2026/01/30/fail.mp4"
	videoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(videoLocalPath, "/")))
	if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(videoAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	// Fake ffmpeg: always fail.
	ffmpegBad := writeExecutable(t, "ffmpeg-bad", "#!/bin/sh\necho fail 1>&2\nexit 1\n")
	ffprobeOK := writeExecutable(t, "ffprobe-ok", "#!/bin/sh\nexit 0\n")

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM media_file`).
		WithArgs(int64(0), 11).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(1), videoLocalPath, "video/mp4", "mp4"))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: uploadRoot}}
	res, err := svc.RepairVideoPosters(context.Background(), ffmpegBad, ffprobeOK, RepairVideoPostersRequest{
		Commit:       true,
		Source:       "local",
		StartAfterID: 0,
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("RepairVideoPosters: %v", err)
	}
	if res.PosterMissing != 1 || res.PosterFailed != 1 || res.PosterGenerated != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestMediaUploadService_RepairVideoPosters_NotInitialized(t *testing.T) {
	if _, err := (*MediaUploadService)(nil).RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMediaUploadService_RepairVideoPosters_LimitClampAndDouyinSource(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	// Limit will be clamped to 2000 => queryLimit=2001.
	mock.ExpectQuery(`FROM douyin_media_file`).
		WithArgs(int64(0), 2001).
		WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}))

	svc := &MediaUploadService{db: db, fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	res, err := svc.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{
		Commit: false,
		Source: "douyin",
		Limit:  5000,
	})
	if err != nil {
		t.Fatalf("RepairVideoPosters: %v", err)
	}
	if res.Source != "douyin" {
		t.Fatalf("source=%q, want %q", res.Source, "douyin")
	}
	if res.Limit != 2000 {
		t.Fatalf("limit=%d, want %d", res.Limit, 2000)
	}
}

func TestMediaUploadService_RepairVideoPosters_CommitFFmpegNotFound(t *testing.T) {
	svc := &MediaUploadService{db: mustNewSQLMockDB(t), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
	t.Cleanup(func() { _ = svc.db.Close() })

	if _, err := svc.RepairVideoPosters(context.Background(), "definitely-not-a-real-ffmpeg-bin", "ffprobe", RepairVideoPostersRequest{Commit: true}); err == nil {
		t.Fatalf("expected error")
	}
}
