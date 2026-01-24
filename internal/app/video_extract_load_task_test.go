package app

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVideoExtractService_loadTaskRow(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnError(sql.ErrNoRows)

		svc := &VideoExtractService{db: db}
		_, err := svc.loadTaskRow(context.Background(), "t1")
		if err == nil || !strings.Contains(err.Error(), "任务不存在") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("other error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnError(sql.ErrConnDone)

		svc := &VideoExtractService{db: db}
		_, err := svc.loadTaskRow(context.Background(), "t1")
		if err == nil || !strings.Contains(err.Error(), "conn") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{
			"task_id", "source_type", "source_ref", "input_abs_path",
			"output_dir_local_path", "output_format", "jpg_quality",
			"mode", "keyframe_mode", "fps", "scene_threshold",
			"start_sec", "end_sec", "max_frames_total", "frames_extracted",
			"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
		}).AddRow(
			"t1", "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{Int64: 90, Valid: true},
			"all", sql.NullString{String: "", Valid: false}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 2, 0,
			1920, 1080, sql.NullFloat64{Float64: 3.3, Valid: true}, sql.NullFloat64{}, VideoExtractStatusPending,
		)

		mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
			WithArgs("t1").
			WillReturnRows(rows)

		svc := &VideoExtractService{db: db}
		out, err := svc.loadTaskRow(context.Background(), "t1")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if out.TaskID != "t1" || out.MaxFramesTotal != 2 || out.VideoWidth != 1920 {
			t.Fatalf("out=%+v", out)
		}
	})
}
