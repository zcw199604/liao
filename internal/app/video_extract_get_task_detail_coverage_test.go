package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func TestVideoExtractService_GetTaskDetail_QueryRowOtherError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnError(sql.ErrConnDone)

	svc := &VideoExtractService{db: db, cfg: config.Config{ServerPort: 8080, VideoExtractFramePageSz: 1}}
	if _, _, err := svc.GetTaskDetail(context.Background(), "t1", 0, 10, ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_GetTaskDetail_RuntimeBranchAndOptionalFields(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	now := time.Now()
	lastLogs, _ := json.Marshal([]string{"ignored"})
	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{
			"task_id", "user_id", "source_type", "source_ref",
			"output_dir_local_path", "output_format", "jpg_quality",
			"mode", "keyframe_mode", "fps", "scene_threshold",
			"start_sec", "end_sec", "max_frames_total", "frames_extracted",
			"video_width", "video_height", "duration_sec", "cursor_out_time_sec",
			"status", "stop_reason", "last_error", "last_logs", "created_at", "updated_at",
		}).AddRow(
			"t1", sql.NullString{String: "u1", Valid: true}, "upload", "/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{Int64: 90, Valid: true},
			"keyframe", sql.NullString{String: "scene", Valid: true}, sql.NullFloat64{Float64: 2.2, Valid: true}, sql.NullFloat64{Float64: 0.9, Valid: true},
			sql.NullFloat64{Float64: 1.1, Valid: true}, sql.NullFloat64{Float64: 2.2, Valid: true}, 10, 3,
			100, 200, sql.NullFloat64{Float64: 9.9, Valid: true}, sql.NullFloat64{Float64: 1.2, Valid: true},
			VideoExtractStatusPausedLimit, sql.NullString{String: string(VideoExtractStopReasonEndSec), Valid: true}, sql.NullString{String: "oops", Valid: true}, sql.NullString{String: string(lastLogs), Valid: true}, now, now,
		))

	mock.ExpectQuery(`(?s)FROM video_extract_frame WHERE task_id = [?] AND seq > [?] ORDER BY seq ASC LIMIT [?]`).
		WithArgs("t1", 0, 3).
		WillReturnRows(sqlmock.NewRows([]string{"seq", "rel_path"}).AddRow(1, "/extract/t1/frames/frame_000001.jpg"))

	rt := &videoExtractRuntime{}
	rt.appendLog("hello")
	svc := &VideoExtractService{
		db:       db,
		cfg:      config.Config{ServerPort: 8080, VideoExtractFramePageSz: 2},
		runtimes: map[string]*videoExtractRuntime{"t1": rt},
	}

	task, frames, err := svc.GetTaskDetail(context.Background(), "t1", -1, 0, "example.com:123")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if task.Runtime == nil || len(task.Runtime.Logs) == 0 {
		t.Fatalf("expected runtime logs")
	}
	if task.JPGQuality == nil || task.FPS == nil || task.SceneThresh == nil || task.StartSec == nil || task.EndSec == nil || task.DurationSec == nil || task.CursorOutTimeSec == nil {
		t.Fatalf("optional fields missing: %+v", task)
	}
	if len(frames.Items) != 1 || frames.NextCursor != 1 || frames.HasMore {
		t.Fatalf("frames=%+v", frames)
	}
}

func TestVideoExtractService_GetTaskDetail_FrameLimitCap(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{
			"task_id", "user_id", "source_type", "source_ref",
			"output_dir_local_path", "output_format", "jpg_quality",
			"mode", "keyframe_mode", "fps", "scene_threshold",
			"start_sec", "end_sec", "max_frames_total", "frames_extracted",
			"video_width", "video_height", "duration_sec", "cursor_out_time_sec",
			"status", "stop_reason", "last_error", "last_logs", "created_at", "updated_at",
		}).AddRow(
			"t1", sql.NullString{}, "upload", "/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{},
			"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
			100, 200, sql.NullFloat64{}, sql.NullFloat64{},
			VideoExtractStatusRunning, sql.NullString{}, sql.NullString{}, sql.NullString{}, now, now,
		))

	// frameLimit is capped at 300 -> query limit=301
	mock.ExpectQuery(`(?s)FROM video_extract_frame WHERE task_id = [?] AND seq > [?] ORDER BY seq ASC LIMIT [?]`).
		WithArgs("t1", 0, 301).
		WillReturnRows(sqlmock.NewRows([]string{"seq", "rel_path"}))

	svc := &VideoExtractService{db: db, cfg: config.Config{ServerPort: 8080, VideoExtractFramePageSz: 2}, runtimes: make(map[string]*videoExtractRuntime)}
	if _, _, err := svc.GetTaskDetail(context.Background(), "t1", 0, 1000, ""); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestVideoExtractService_GetTaskDetail_FramesQueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{
			"task_id", "user_id", "source_type", "source_ref",
			"output_dir_local_path", "output_format", "jpg_quality",
			"mode", "keyframe_mode", "fps", "scene_threshold",
			"start_sec", "end_sec", "max_frames_total", "frames_extracted",
			"video_width", "video_height", "duration_sec", "cursor_out_time_sec",
			"status", "stop_reason", "last_error", "last_logs", "created_at", "updated_at",
		}).AddRow(
			"t1", sql.NullString{}, "upload", "/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{},
			"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
			100, 200, sql.NullFloat64{}, sql.NullFloat64{},
			VideoExtractStatusRunning, sql.NullString{}, sql.NullString{}, sql.NullString{}, now, now,
		))

	mock.ExpectQuery(`(?s)FROM video_extract_frame WHERE task_id = [?] AND seq > [?] ORDER BY seq ASC LIMIT [?]`).
		WithArgs("t1", 0, 2).
		WillReturnError(sql.ErrConnDone)

	svc := &VideoExtractService{db: db, cfg: config.Config{ServerPort: 8080, VideoExtractFramePageSz: 1}, runtimes: make(map[string]*videoExtractRuntime)}
	if _, _, err := svc.GetTaskDetail(context.Background(), "t1", 0, 1, ""); err == nil {
		t.Fatalf("expected error")
	}
}
