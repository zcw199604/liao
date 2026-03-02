package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRepairVideoPosters_AdditionalBranches(t *testing.T) {
	t.Run("commit requires ffprobe", func(t *testing.T) {
		svc := &MediaUploadService{db: wrapMySQLDB(mustNewSQLMockDB(t)), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		t.Cleanup(func() { _ = svc.db.Close() })
		ffmpegOK := writeExecutable(t, "ffmpeg-ok-repair", "#!/bin/sh\nexit 0\n")

		if _, err := svc.RepairVideoPosters(context.Background(), ffmpegOK, "", RepairVideoPostersRequest{Commit: true}); err == nil {
			t.Fatalf("expected ffprobe required error")
		}
		if _, err := svc.RepairVideoPosters(context.Background(), ffmpegOK, "definitely-not-found-ffprobe", RepairVideoPostersRequest{Commit: true}); err == nil {
			t.Fatalf("expected ffprobe lookpath error")
		}
	})

	t.Run("query and scan errors", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}

		mock.ExpectQuery(`FROM media_file`).WithArgs(int64(0), 201).WillReturnError(errors.New("query failed"))
		if _, err := svc.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{Commit: false, Limit: 0}); err == nil {
			t.Fatalf("expected query error")
		}

		db2, mock2, cleanup2 := newSQLMock(t)
		defer cleanup2()
		svc2 := &MediaUploadService{db: wrapMySQLDB(db2), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		mock2.ExpectQuery(`FROM media_file`).WithArgs(int64(0), 201).WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow("bad-id", "/videos/a.mp4", "video/mp4", "mp4"))
		if _, err := svc2.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{Commit: false, Limit: 0}); err == nil {
			t.Fatalf("expected scan error")
		}
	})

	t.Run("poster path resolve error and rows.Err", func(t *testing.T) {
		uploadRoot := t.TempDir()
		videoLocalPath := "/videos/a.mp4"
		videoAbs := filepath.Join(uploadRoot, filepath.FromSlash(strings.TrimPrefix(videoLocalPath, "/")))
		if err := os.MkdirAll(filepath.Dir(videoAbs), 0o755); err != nil {
			t.Fatalf("mkdir err=%v", err)
		}
		if err := os.WriteFile(videoAbs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write err=%v", err)
		}

		oldRel := filepathRelFn
		t.Cleanup(func() { filepathRelFn = oldRel })
		filepathRelFn = func(basepath, targpath string) (string, error) {
			if strings.Contains(targpath, ".poster.") {
				return "", errors.New("poster rel failed")
			}
			return oldRel(basepath, targpath)
		}

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: uploadRoot}}

		mock.ExpectQuery(`FROM media_file`).WithArgs(int64(0), 6).WillReturnRows(sqlmock.NewRows([]string{"id", "local_path", "file_type", "file_extension"}).
			AddRow(int64(1), videoLocalPath, "video/mp4", "mp4").
			AddRow(int64(2), " ", "video/mp4", "mp4"))

		res, err := svc.RepairVideoPosters(context.Background(), "ffmpeg", "ffprobe", RepairVideoPostersRequest{Commit: false, Limit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Skipped == 0 && res.VideoMissing == 0 {
			t.Fatalf("expected branch counters to change: %+v", res)
		}
	})
}
