package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVideoExtractService_ContinueTask_LoadTaskRowError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnError(sql.ErrNoRows)

	svc := &VideoExtractService{db: wrapMySQLDB(db)}
	if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", MaxFrames: ptrInt(10)}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_ContinueTask_UpdateExecError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

	mock.ExpectExec(`(?s)UPDATE video_extract_task SET .*WHERE task_id = [?]`).
		WillReturnError(sql.ErrConnDone)

	svc := &VideoExtractService{db: wrapMySQLDB(db), queue: make(chan string, 1), closing: make(chan struct{})}
	if err := svc.ContinueTask(context.Background(), VideoExtractContinueRequest{TaskID: "t1", MaxFrames: ptrInt(10)}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_DeleteTask_EmptyTaskID(t *testing.T) {
	db, _, cleanup := newSQLMock(t)
	defer cleanup()

	svc := &VideoExtractService{db: wrapMySQLDB(db)}
	if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: " "}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_DeleteTask_LoadTaskRowError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnError(sql.ErrNoRows)

	svc := &VideoExtractService{db: wrapMySQLDB(db), runtimes: make(map[string]*videoExtractRuntime)}
	if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: "t1"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_DeleteTask_DeleteFrameExecError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

	mock.ExpectExec(`(?s)DELETE FROM video_extract_frame WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnError(sql.ErrConnDone)

	svc := &VideoExtractService{db: wrapMySQLDB(db), runtimes: make(map[string]*videoExtractRuntime)}
	if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: "t1"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_DeleteTask_DeleteTaskExecError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnRows(makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "/extract/t1"))

	mock.ExpectExec(`(?s)DELETE FROM video_extract_frame WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)DELETE FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnError(sql.ErrConnDone)

	svc := &VideoExtractService{db: wrapMySQLDB(db), runtimes: make(map[string]*videoExtractRuntime)}
	if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: "t1"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_DeleteTask_DeleteFiles_OutputDirWithoutLeadingSlash(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	uploadRoot := t.TempDir()
	outDir := filepath.Join(uploadRoot, "extract", "t1")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	taskRow := makeLoadTaskRowRows("t1", VideoExtractStatusPausedUser, sql.NullFloat64{}, sql.NullFloat64{}, 10, 0, "extract/t1")
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
		db:        wrapMySQLDB(db),
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.DeleteTask(context.Background(), VideoExtractDeleteRequest{TaskID: "t1", DeleteFiles: true}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(outDir); err == nil {
		t.Fatalf("expected outDir removed")
	}
}
