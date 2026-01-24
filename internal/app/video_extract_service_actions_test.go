package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func makeLoadTaskRowRows(taskID string, status VideoExtractTaskStatus, startSec, endSec sql.NullFloat64, maxFramesTotal int, framesExtracted int, outputDirLocalPath string) *sqlmock.Rows {
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	return sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		outputDirLocalPath, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		startSec, endSec, maxFramesTotal, framesExtracted,
		1920, 1080, sql.NullFloat64{}, sql.NullFloat64{}, status,
	)
}

func TestVideoExtractService_ContinueTask(t *testing.T) {
	t.Run("not init", func(t *testing.T) {
		svc := &VideoExtractService{}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1"}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("empty task", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: db}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: " "}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("status running", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusRunning, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

		svc := &VideoExtractService{db: db}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", MaxFrames: ptrInt(10)}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("status not allowed", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusFailed, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

		svc := &VideoExtractService{db: db}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", MaxFrames: ptrInt(10)}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("endSec invalid", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedLimit, sql.NullFloat64{Float64: 2, Valid: true}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

		svc := &VideoExtractService{db: db}
		end := -1.0
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", EndSec: &end}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("endSec <= startSec", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedLimit, sql.NullFloat64{Float64: 2, Valid: true}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

		svc := &VideoExtractService{db: db}
		end := 1.0
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", EndSec: &end}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("maxFrames invalid", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 5, "/extract/t1"))

		svc := &VideoExtractService{db: db}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", MaxFrames: ptrInt(0)}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("maxFrames < extracted", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 5, "/extract/t1"))

		svc := &VideoExtractService{db: db}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", MaxFrames: ptrInt(4)}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("no updates", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

		svc := &VideoExtractService{db: db}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1"}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ok and enqueue", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.MatchExpectationsInOrder(false)

		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{Float64: 1, Valid: true}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

		mock.ExpectExec(`(?s)UPDATE video_extract_task SET .*WHERE task_id = [?]`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "t1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		svc := &VideoExtractService{db: db, queue: make(chan string, 1), closing: make(chan struct{})}
		end := 3.0
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", EndSec: &end}); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("enqueue full", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.MatchExpectationsInOrder(false)

		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

		mock.ExpectExec(`(?s)UPDATE video_extract_task SET .*WHERE task_id = [?]`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "t1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		queue := make(chan string, 1)
		queue <- "filled"
		svc := &VideoExtractService{db: db, queue: queue, closing: make(chan struct{})}
		if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", MaxFrames: ptrInt(10)}); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestVideoExtractService_DeleteTask_and_CancelAndMark(t *testing.T) {
	t.Run("DeleteTask invalid", func(t *testing.T) {
		svc := &VideoExtractService{}
		if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: "t1"}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("DeleteTask delete files safe", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.MatchExpectationsInOrder(false)

		uploadRoot := t.TempDir()
		outDir := filepath.Join(uploadRoot, "extract", "t1")
		if err := os.MkdirAll(filepath.Join(outDir, "frames"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "frames", "a.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		taskRow := makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1")
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(taskRow)

		mock.ExpectExec(`(?s)DELETE FROM video_extract_frame WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`(?s)DELETE FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		cancelCalled := false
		svc := &VideoExtractService{
			db:        db,
			fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
			runtimes:  map[string]*videoExtractRuntime{"t1": {cancel: func() { cancelCalled = true }}},
		}
		if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: "t1", DeleteFiles: true}); err != nil {
			t.Fatalf("err=%v", err)
		}
		if !cancelCalled {
			t.Fatalf("expected cancel called")
		}
		if _, err := os.Stat(outDir); err == nil {
			t.Fatalf("expected output dir removed")
		}
	})

	t.Run("DeleteTask does not delete out of base", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		uploadRoot := t.TempDir()
		escapeDir := filepath.Join(uploadRoot, "escape")
		if err := os.MkdirAll(escapeDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		taskRow := makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/../escape")
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(taskRow)
		mock.ExpectExec(`(?s)DELETE FROM video_extract_frame WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`(?s)DELETE FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		svc := &VideoExtractService{
			db:        db,
			fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
			runtimes:  make(map[string]*videoExtractRuntime),
		}
		if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: "t1", DeleteFiles: true}); err != nil {
			t.Fatalf("err=%v", err)
		}
		if _, err := os.Stat(escapeDir); err != nil {
			t.Fatalf("escapeDir should still exist: %v", err)
		}
	})

	t.Run("CancelAndMark empty task", func(t *testing.T) {
		svc := &VideoExtractService{}
		if err := svc.CancelAndMark(context.Background(), VideoExtractCancelRequest{TaskID: " "}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("CancelAndMark updates status", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectExec(`(?s)UPDATE video_extract_task SET status =`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		svc := &VideoExtractService{db: db, runtimes: make(map[string]*videoExtractRuntime)}
		if err := svc.CancelAndMark(context.Background(), VideoExtractCancelRequest{TaskID: "t1"}); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	if strings.TrimSpace("ok") == "" {
		t.Fatalf("unreachable")
	}
}

func ptrInt(v int) *int { return &v }
