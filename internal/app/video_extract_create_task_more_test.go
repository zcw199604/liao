package app

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

type stubMtPhotoResolver struct {
	item *MtPhotoFilePath
	err  error
}

func (s *stubMtPhotoResolver) ResolveFilePath(context.Context, string) (*MtPhotoFilePath, error) {
	return s.item, s.err
}

func TestVideoExtractService_CreateTask_MoreBranches(t *testing.T) {
	t.Run("service not init", func(t *testing.T) {
		var svc *VideoExtractService
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/x.mp4", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("mode keyframe defaults invalid keyframe mode", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
			SourceType:   VideoExtractSourceUpload,
			LocalPath:    " ", // stop after validation
			Mode:         VideoExtractModeKeyframe,
			KeyframeMode: VideoExtractKeyframeMode("bad"),
			MaxFrames:    1,
		}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("mode keyframe scene defaults threshold", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
			SourceType:   VideoExtractSourceUpload,
			LocalPath:    " ", // stop after validation
			Mode:         VideoExtractModeKeyframe,
			KeyframeMode: VideoExtractKeyframeScene,
			MaxFrames:    1,
		}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("mode keyframe scene invalid threshold", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		th := 2.0
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
			SourceType:   VideoExtractSourceUpload,
			LocalPath:    "/x.mp4",
			Mode:         VideoExtractModeKeyframe,
			KeyframeMode: VideoExtractKeyframeScene,
			SceneThresh:  &th,
			MaxFrames:    1,
		}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("endSec invalid", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		end := -1.0
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/x.mp4", Mode: VideoExtractModeAll, EndSec: &end, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("outputFormat invalid", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/x.mp4", Mode: VideoExtractModeAll, OutputFormat: VideoExtractOutputFormat("bad"), MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("jpgQuality invalid", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		q := 0
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/x.mp4", Mode: VideoExtractModeAll, JPGQuality: &q, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("upload localPath empty", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: " ", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("upload resolveUploadAbsPath error", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		uploadRoot := t.TempDir()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: uploadRoot}}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/videos/missing.mp4", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("mtPhoto md5 empty/invalid and nil mtPhoto", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceMtPhoto, MD5: " ", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceMtPhoto, MD5: "not-md5", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceMtPhoto, MD5: "0123456789abcdef0123456789abcdef", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("mtPhoto ResolveFilePath error and nil item", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{
			db:        wrapMySQLDB(db),
			fileStore: &FileStorageService{baseUploadAbs: t.TempDir()},
			mtPhoto:   &stubMtPhotoResolver{err: errors.New("boom")},
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceMtPhoto, MD5: "0123456789abcdef0123456789abcdef", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}

		svc.mtPhoto = &stubMtPhotoResolver{item: nil, err: nil}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceMtPhoto, MD5: "0123456789abcdef0123456789abcdef", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("mtPhoto resolveLspLocalPath error", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{
			db:        wrapMySQLDB(db),
			fileStore: &FileStorageService{baseUploadAbs: t.TempDir()},
			mtPhoto:   &stubMtPhotoResolver{item: &MtPhotoFilePath{ID: 1, FilePath: "/notlsp/x.mp4"}},
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceMtPhoto, MD5: "0123456789abcdef0123456789abcdef", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ProbeVideo error and width/height invalid", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		uploadRoot := t.TempDir()
		inputAbs := filepath.Join(uploadRoot, "in.mp4")
		if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		svc := &VideoExtractService{
			db:        wrapMySQLDB(db),
			cfg:       config.Config{FFprobePath: filepath.Join(t.TempDir(), "missing-ffprobe")},
			fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/in.mp4", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}

		ffprobeZero := filepath.Join(t.TempDir(), "ffprobe-zero")
		if err := os.WriteFile(ffprobeZero, []byte("#!/bin/sh\necho '{\"streams\":[{\"width\":0,\"height\":0,\"avg_frame_rate\":\"30/1\"}],\"format\":{\"duration\":\"1\"}}'\n"), 0o755); err != nil {
			t.Fatalf("write ffprobe: %v", err)
		}
		svc.cfg.FFprobePath = ffprobeZero
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/in.mp4", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("MkdirAll error", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		uploadRoot := t.TempDir()
		inputAbs := filepath.Join(uploadRoot, "in.mp4")
		if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		ffprobeOK := filepath.Join(t.TempDir(), "ffprobe-ok")
		if err := os.WriteFile(ffprobeOK, []byte("#!/bin/sh\necho '{\"streams\":[{\"width\":1,\"height\":1,\"avg_frame_rate\":\"30/1\"}],\"format\":{\"duration\":\"1\"}}'\n"), 0o755); err != nil {
			t.Fatalf("write ffprobe: %v", err)
		}

		uploadRootFile := filepath.Join(t.TempDir(), "uploadRootFile")
		if err := os.WriteFile(uploadRootFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		svc := &VideoExtractService{
			db:        wrapMySQLDB(db),
			cfg:       config.Config{FFprobePath: ffprobeOK},
			fileStore: &FileStorageService{baseUploadAbs: uploadRootFile},
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/in.mp4", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("MkdirAll output frames dir error when extract is file", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		uploadRoot := t.TempDir()
		inputAbs := filepath.Join(uploadRoot, "in.mp4")
		if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		// output frames dir is {baseUploadAbs}/extract/{taskId}/frames; make "extract" a file to force MkdirAll error.
		if err := os.WriteFile(filepath.Join(uploadRoot, "extract"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write extract: %v", err)
		}

		ffprobeOK := filepath.Join(t.TempDir(), "ffprobe-ok")
		if err := os.WriteFile(ffprobeOK, []byte("#!/bin/sh\necho '{\"streams\":[{\"width\":10,\"height\":10,\"avg_frame_rate\":\"30/1\"}],\"format\":{\"duration\":\"1\"}}'\n"), 0o755); err != nil {
			t.Fatalf("write ffprobe: %v", err)
		}

		svc := &VideoExtractService{
			db:        wrapMySQLDB(db),
			cfg:       config.Config{FFprobePath: ffprobeOK},
			fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/in.mp4", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("durationSec <= 0 stores NULL and png clears jpgQuality", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		uploadRoot := t.TempDir()
		inputAbs := filepath.Join(uploadRoot, "in.mp4")
		if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		ffprobeDur0 := filepath.Join(t.TempDir(), "ffprobe-dur0")
		if err := os.WriteFile(ffprobeDur0, []byte("#!/bin/sh\necho '{\"streams\":[{\"width\":10,\"height\":10,\"avg_frame_rate\":\"30/1\"}],\"format\":{\"duration\":\"0\"}}'\n"), 0o755); err != nil {
			t.Fatalf("write ffprobe: %v", err)
		}

		q := 10
		svc := &VideoExtractService{
			db:        wrapMySQLDB(db),
			cfg:       config.Config{FFprobePath: ffprobeDur0},
			fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		}

		mock.ExpectExec(`(?s)INSERT INTO video_extract_task`).
			WithArgs(
				sqlmock.AnyArg(),
				nil,
				"upload",
				"/in.mp4",
				inputAbs,
				sqlmock.AnyArg(),
				"png",
				nil,
				"all",
				nil,
				nil,
				nil,
				nil,
				nil,
				1,
				0,
				10,
				10,
				nil, // duration_sec
				nil,
				"PENDING",
				nil,
				nil,
				nil,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{
			SourceType:   VideoExtractSourceUpload,
			LocalPath:    "/in.mp4",
			Mode:         VideoExtractModeAll,
			OutputFormat: VideoExtractOutputPNG,
			JPGQuality:   &q, // should be cleared because outputFormat=png
			MaxFrames:    1,
		}); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("db insert error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		uploadRoot := t.TempDir()
		inputAbs := filepath.Join(uploadRoot, "in.mp4")
		if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		ffprobeOK := filepath.Join(t.TempDir(), "ffprobe-ok")
		if err := os.WriteFile(ffprobeOK, []byte("#!/bin/sh\necho '{\"streams\":[{\"width\":10,\"height\":10,\"avg_frame_rate\":\"30/1\"}],\"format\":{\"duration\":\"1\"}}'\n"), 0o755); err != nil {
			t.Fatalf("write ffprobe: %v", err)
		}

		mock.ExpectExec(`(?s)INSERT INTO video_extract_task`).
			WillReturnError(sql.ErrConnDone)

		svc := &VideoExtractService{
			db:        wrapMySQLDB(db),
			cfg:       config.Config{FFprobePath: ffprobeOK},
			fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		}
		if _, _, err := svc.CreateTask(context.Background(), VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/in.mp4", Mode: VideoExtractModeAll, MaxFrames: 1}); err == nil {
			t.Fatalf("expected error")
		}
	})
}
