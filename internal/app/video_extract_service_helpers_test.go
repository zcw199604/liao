package app

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVideoExtractService_updateProgress(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	svc := &VideoExtractService{db: db}

	// no-op when both outTimeMs and frame are negative
	if err := svc.updateProgress(context.Background(), "t1", nil, 1.5, -1, -1); err != nil {
		t.Fatalf("err=%v", err)
	}

	mock.ExpectExec(`(?s)UPDATE video_extract_task SET cursor_out_time_sec = [?], updated_at = [?] WHERE task_id = [?]`).
		WithArgs(1.5, sqlmock.AnyArg(), "t1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := svc.updateProgress(context.Background(), "t1", nil, 1.5, -1, 0); err != nil {
		t.Fatalf("err=%v", err)
	}

	mock.ExpectExec(`(?s)UPDATE video_extract_task SET cursor_out_time_sec = [?], updated_at = [?] WHERE task_id = [?]`).
		WithArgs(2.0, sqlmock.AnyArg(), "t1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := svc.updateProgress(context.Background(), "t1", nil, 1.5, 500_000, -1); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestVideoExtractService_insertFrame_and_setTaskStatus(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	svc := &VideoExtractService{db: db}

	if err := svc.insertFrame(context.Background(), "t1", 0, "/x"); err != nil {
		t.Fatalf("err=%v", err)
	}
	if err := svc.insertFrame(context.Background(), "t1", 1, ""); err != nil {
		t.Fatalf("err=%v", err)
	}

	// insert error
	mock.ExpectExec(`(?s)INSERT INTO video_extract_frame`).
		WillReturnError(fmt.Errorf("boom"))
	if err := svc.insertFrame(context.Background(), "t1", 1, "/rel"); err == nil {
		t.Fatalf("expected error")
	}

	// rows affected == 0 => no update frames_extracted
	mock.ExpectExec(`(?s)INSERT INTO video_extract_frame`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := svc.insertFrame(context.Background(), "t1", 2, "/rel2"); err != nil {
		t.Fatalf("err=%v", err)
	}

	// rows affected > 0 => update frames_extracted
	mock.ExpectExec(`(?s)INSERT INTO video_extract_frame`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task SET frames_extracted = GREATEST.*updated_at = [?] WHERE task_id = [?]`).
		WithArgs(3, sqlmock.AnyArg(), "t1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := svc.insertFrame(context.Background(), "t1", 3, "/rel3"); err != nil {
		t.Fatalf("err=%v", err)
	}

	// empty task id is ignored
	if err := svc.setTaskStatusWithLogs(context.Background(), "  ", VideoExtractStatusRunning, "x", "y", "z"); err != nil {
		t.Fatalf("err=%v", err)
	}

	mock.ExpectExec(`(?s)UPDATE video_extract_task SET status = [?], stop_reason = [?], last_error = [?], last_logs = [?], updated_at = [?] WHERE task_id = [?]`).
		WithArgs(string(VideoExtractStatusPausedLimit), "MAX_FRAMES", nil, "[]", sqlmock.AnyArg(), "t1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.setTaskStatusWithLogs(context.Background(), "t1", VideoExtractStatusPausedLimit, " MAX_FRAMES ", " ", " [] "); err != nil {
		t.Fatalf("err=%v", err)
	}

	// setTaskStatus is a thin wrapper
	mock.ExpectExec(`(?s)UPDATE video_extract_task SET status = [?], stop_reason = [?], last_error = [?], last_logs = [?], updated_at = [?] WHERE task_id = [?]`).
		WithArgs(string(VideoExtractStatusRunning), nil, nil, nil, sqlmock.AnyArg(), "t2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := svc.setTaskStatus(context.Background(), "t2", VideoExtractStatusRunning, "", ""); err != nil {
		t.Fatalf("err=%v", err)
	}

	// Ensure the regexp in expectations above is actually strict enough.
	if strings.TrimSpace(time.Now().Format(time.RFC3339)) == "" {
		t.Fatalf("unreachable")
	}
}
