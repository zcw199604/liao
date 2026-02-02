package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func TestVideoExtractService_ListTasks_PageSizeCapAndOptionalFields(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery(`(?s)SELECT COUNT.*FROM video_extract_task`).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"task_id", "user_id", "source_type", "source_ref",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec",
		"status", "stop_reason", "last_error", "created_at", "updated_at",
	}).AddRow(
		"t1", sql.NullString{String: "u1", Valid: true}, "upload", "/in.mp4",
		"/extract/t1", "jpg", sql.NullInt64{Int64: 5, Valid: true},
		"keyframe", sql.NullString{String: "scene", Valid: true}, sql.NullFloat64{Float64: 2.2, Valid: true}, sql.NullFloat64{Float64: 0.9, Valid: true},
		sql.NullFloat64{Float64: 1.1, Valid: true}, sql.NullFloat64{Float64: 2.2, Valid: true}, 10, 3,
		100, 200, sql.NullFloat64{Float64: 9.9, Valid: true}, sql.NullFloat64{Float64: 1.2, Valid: true},
		VideoExtractStatusPausedLimit, sql.NullString{String: string(VideoExtractStopReasonEndSec), Valid: true}, sql.NullString{String: "oops", Valid: true}, now, now,
	)

	mock.ExpectQuery(`(?s)FROM video_extract_task ORDER BY updated_at DESC LIMIT [?] OFFSET [?]`).
		WithArgs(100, 0).
		WillReturnRows(rows)

	svc := &VideoExtractService{db: wrapMySQLDB(db), cfg: config.Config{ServerPort: 8080}, runtimes: make(map[string]*videoExtractRuntime)}
	items, total, err := svc.ListTasks(context.Background(), 1, 1000, "example.com:123")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total/items=%d/%d", total, len(items))
	}
	if items[0].JPGQuality == nil || items[0].FPS == nil || items[0].SceneThresh == nil || items[0].StartSec == nil || items[0].EndSec == nil || items[0].DurationSec == nil || items[0].CursorOutTimeSec == nil {
		t.Fatalf("optional fields missing: %+v", items[0])
	}
	if items[0].KeyframeMode != VideoExtractKeyframeScene || items[0].StopReason != VideoExtractStopReasonEndSec || items[0].LastError == "" {
		t.Fatalf("fields=%+v", items[0])
	}
}

func TestVideoExtractService_ListTasks_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery(`(?s)SELECT COUNT.*FROM video_extract_task`).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"task_id", "user_id", "source_type", "source_ref",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec",
		"status", "stop_reason", "last_error", "created_at", "updated_at",
	}).AddRow(
		nil, sql.NullString{}, "upload", "/in.mp4",
		"/extract/t1", "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		100, 200, sql.NullFloat64{}, sql.NullFloat64{},
		VideoExtractStatusPending, sql.NullString{}, sql.NullString{}, now, now,
	)

	mock.ExpectQuery(`(?s)FROM video_extract_task ORDER BY updated_at DESC LIMIT [?] OFFSET [?]`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	svc := &VideoExtractService{db: wrapMySQLDB(db), cfg: config.Config{ServerPort: 8080}, runtimes: make(map[string]*videoExtractRuntime)}
	if _, _, err := svc.ListTasks(context.Background(), 1, 10, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_ListTasks_RowsErr(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery(`(?s)SELECT COUNT.*FROM video_extract_task`).
		WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(2))

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"task_id", "user_id", "source_type", "source_ref",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec",
		"status", "stop_reason", "last_error", "created_at", "updated_at",
	}).AddRow(
		"t1", sql.NullString{}, "upload", "/in.mp4",
		"/extract/t1", "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		100, 200, sql.NullFloat64{}, sql.NullFloat64{},
		VideoExtractStatusPending, sql.NullString{}, sql.NullString{}, now, now,
	).AddRow(
		"t2", sql.NullString{}, "upload", "/in.mp4",
		"/extract/t2", "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		100, 200, sql.NullFloat64{}, sql.NullFloat64{},
		VideoExtractStatusPending, sql.NullString{}, sql.NullString{}, now, now,
	).RowError(1, sql.ErrConnDone)

	mock.ExpectQuery(`(?s)FROM video_extract_task ORDER BY updated_at DESC LIMIT [?] OFFSET [?]`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	svc := &VideoExtractService{db: wrapMySQLDB(db), cfg: config.Config{ServerPort: 8080}, runtimes: make(map[string]*videoExtractRuntime)}
	if _, _, err := svc.ListTasks(context.Background(), 1, 10, ""); err == nil {
		t.Fatalf("expected error")
	}
}
