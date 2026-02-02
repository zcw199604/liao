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

func TestVideoExtractService_toUploadURL(t *testing.T) {
	svc := &VideoExtractService{cfg: config.Config{ServerPort: 8080}}
	if svc.toUploadURL("", "") != "" {
		t.Fatalf("expected empty")
	}
	if got := svc.toUploadURL("a/b", ""); got != "http://localhost:8080/upload/a/b" {
		t.Fatalf("got=%q", got)
	}
	if got := svc.toUploadURL("/a/b", "example.com:123"); got != "http://example.com:123/upload/a/b" {
		t.Fatalf("got=%q", got)
	}
}

func TestVideoExtractService_ListTasks(t *testing.T) {
	t.Run("not init", func(t *testing.T) {
		svc := &VideoExtractService{}
		if _, _, err := svc.ListTasks(context.Background(), 1, 10, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("count error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)SELECT COUNT.*FROM video_extract_task`).
			WillReturnError(sql.ErrConnDone)

		svc := &VideoExtractService{db: wrapMySQLDB(db)}
		if _, _, err := svc.ListTasks(context.Background(), 1, 10, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("rows error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)SELECT COUNT.*FROM video_extract_task`).
			WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))
		mock.ExpectQuery(`(?s)FROM video_extract_task ORDER BY updated_at DESC LIMIT [?] OFFSET [?]`).
			WithArgs(10, 0).
			WillReturnError(sql.ErrConnDone)

		svc := &VideoExtractService{db: wrapMySQLDB(db)}
		if _, _, err := svc.ListTasks(context.Background(), 1, 10, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ok with runtime", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.MatchExpectationsInOrder(false)

		mock.ExpectQuery(`(?s)SELECT COUNT.*FROM video_extract_task`).
			WillReturnRows(sqlmock.NewRows([]string{"cnt"}).AddRow(1))

		now := time.Now()
		taskRows := sqlmock.NewRows([]string{
			"task_id", "user_id", "source_type", "source_ref",
			"output_dir_local_path", "output_format", "jpg_quality",
			"mode", "keyframe_mode", "fps", "scene_threshold",
			"start_sec", "end_sec", "max_frames_total", "frames_extracted",
			"video_width", "video_height", "duration_sec", "cursor_out_time_sec",
			"status", "stop_reason", "last_error", "created_at", "updated_at",
		}).AddRow(
			"t1", sql.NullString{String: "u1", Valid: true}, "upload", "/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{Int64: 90, Valid: true},
			"all", sql.NullString{String: "", Valid: false}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 1,
			1920, 1080, sql.NullFloat64{Float64: 3.3, Valid: true}, sql.NullFloat64{Float64: 1.1, Valid: true},
			VideoExtractStatusRunning, sql.NullString{String: "", Valid: false}, sql.NullString{String: "", Valid: false}, now, now,
		)

		mock.ExpectQuery(`(?s)FROM video_extract_task ORDER BY updated_at DESC LIMIT [?] OFFSET [?]`).
			WithArgs(20, 0).
			WillReturnRows(taskRows)

		rt := &videoExtractRuntime{}
		rt.appendLog("progress=continue")
		rt.setProgress(1, 1_000_000, "1.0x")

		svc := &VideoExtractService{
			db:       wrapMySQLDB(db),
			cfg:      config.Config{ServerPort: 8080, VideoExtractFramePageSz: 20},
			runtimes: map[string]*videoExtractRuntime{"t1": rt},
		}
		items, total, err := svc.ListTasks(context.Background(), 0, 0, "")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if total != 1 || len(items) != 1 {
			t.Fatalf("total/items=%d/%d", total, len(items))
		}
		if items[0].OutputDirURL == "" || items[0].Runtime == nil || len(items[0].Runtime.Logs) == 0 {
			t.Fatalf("item=%+v", items[0])
		}
	})
}

func TestVideoExtractService_GetTaskDetail(t *testing.T) {
	t.Run("not init", func(t *testing.T) {
		svc := &VideoExtractService{}
		if _, _, err := svc.GetTaskDetail(context.Background(), "t1", 0, 10, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("empty task", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		svc := &VideoExtractService{db: wrapMySQLDB(db)}
		if _, _, err := svc.GetTaskDetail(context.Background(), " ", 0, 10, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`(?s)FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnError(sql.ErrNoRows)

		svc := &VideoExtractService{db: wrapMySQLDB(db)}
		if _, _, err := svc.GetTaskDetail(context.Background(), "t1", 0, 10, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ok uses lastLogs when no runtime", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.MatchExpectationsInOrder(false)

		now := time.Now()
		lastLogs, _ := json.Marshal([]string{"a", "b"})
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
				"/extract/t1", "jpg", sql.NullInt64{},
				"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
				sql.NullFloat64{}, sql.NullFloat64{}, 10, 2,
				1920, 1080, sql.NullFloat64{}, sql.NullFloat64{},
				VideoExtractStatusFinished, sql.NullString{}, sql.NullString{}, sql.NullString{String: string(lastLogs), Valid: true}, now, now,
			))

		frameRows := sqlmock.NewRows([]string{"seq", "rel_path"}).
			AddRow(1, "/extract/t1/frames/frame_000001.jpg").
			AddRow(2, "/extract/t1/frames/frame_000002.jpg").
			AddRow(3, "/extract/t1/frames/frame_000003.jpg") // triggers hasMore when limit=2

		mock.ExpectQuery(`(?s)FROM video_extract_frame WHERE task_id = [?] AND seq > [?] ORDER BY seq ASC LIMIT [?]`).
			WithArgs("t1", 0, 3).
			WillReturnRows(frameRows)

		svc := &VideoExtractService{db: wrapMySQLDB(db), cfg: config.Config{ServerPort: 8080, VideoExtractFramePageSz: 2}}
		task, frames, err := svc.GetTaskDetail(context.Background(), "t1", -1, 2, "")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if task.Runtime == nil || len(task.Runtime.Logs) != 2 {
			t.Fatalf("runtime=%+v", task.Runtime)
		}
		if len(frames.Items) != 2 || !frames.HasMore || frames.NextCursor != 2 {
			t.Fatalf("frames=%+v", frames)
		}
	})

	t.Run("frames scan and rows err", func(t *testing.T) {
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
				1920, 1080, sql.NullFloat64{}, sql.NullFloat64{},
				VideoExtractStatusRunning, sql.NullString{}, sql.NullString{}, sql.NullString{}, now, now,
			))

		badRows := sqlmock.NewRows([]string{"seq", "rel_path"}).AddRow("bad", "/x")
		mock.ExpectQuery(`(?s)FROM video_extract_frame WHERE task_id = [?] AND seq > [?] ORDER BY seq ASC LIMIT [?]`).
			WithArgs("t1", 0, 2).
			WillReturnRows(badRows)

		svc := &VideoExtractService{db: wrapMySQLDB(db), cfg: config.Config{ServerPort: 8080, VideoExtractFramePageSz: 1}}
		if _, _, err := svc.GetTaskDetail(context.Background(), "t1", 0, 1, ""); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("frames rows err", func(t *testing.T) {
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
				1920, 1080, sql.NullFloat64{}, sql.NullFloat64{},
				VideoExtractStatusRunning, sql.NullString{}, sql.NullString{}, sql.NullString{}, now, now,
			))

		rows := sqlmock.NewRows([]string{"seq", "rel_path"}).
			AddRow(1, "/x").
			AddRow(2, "/y").
			RowError(1, sql.ErrConnDone)
		mock.ExpectQuery(`(?s)FROM video_extract_frame WHERE task_id = [?] AND seq > [?] ORDER BY seq ASC LIMIT [?]`).
			WithArgs("t1", 0, 2).
			WillReturnRows(rows)

		svc := &VideoExtractService{db: wrapMySQLDB(db), cfg: config.Config{ServerPort: 8080, VideoExtractFramePageSz: 1}}
		if _, _, err := svc.GetTaskDetail(context.Background(), "t1", 0, 1, ""); err == nil {
			t.Fatalf("expected error")
		}
	})
}
