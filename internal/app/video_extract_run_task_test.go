package app

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func TestVideoExtractService_runTask_Success_MaxFrames(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	uploadRoot := t.TempDir()
	inputAbs := filepath.Join(uploadRoot, "in.mp4")
	if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	ffmpegPath := filepath.Join(t.TempDir(), "ffmpeg")
	script := `#!/bin/sh
start=1
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-start_number" ]; then
    shift
    start="$1"
  fi
  out="$1"
  shift
done
dir=$(dirname "$out")
ext="${out##*.}"
mkdir -p "$dir"
printf "frame=1\nout_time_ms=1000000\nspeed=1.0x\nprogress=continue\n"
f1=$(printf "frame_%06d.%s" "$start" "$ext")
: > "$dir/$f1"
f2=$(printf "frame_%06d.%s" $(($start+1)) "$ext")
: > "$dir/$f2"
printf "frame=2\nout_time_ms=2000000\nspeed=1.0x\nprogress=end\n"
exit 0
`
	if err := os.WriteFile(ffmpegPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write ffmpeg: %v", err)
	}

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: ffmpegPath},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		queue:     make(chan string, 1),
		closing:   make(chan struct{}),
		runtimes:  make(map[string]*videoExtractRuntime),
	}

	taskID := "task-1"

	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}

	firstRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", inputAbs,
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 2, 0,
		1920, 1080, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusPending,
	)
	secondRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", inputAbs,
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 2, 2,
		1920, 1080, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{Float64: 2.0, Valid: true}, VideoExtractStatusRunning,
	)

	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(firstRow)
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(secondRow)

	mock.ExpectExec(`(?s)UPDATE video_extract_task SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`(?s)UPDATE video_extract_task SET cursor_out_time_sec =`).
		WithArgs(1.0, sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task SET cursor_out_time_sec =`).
		WithArgs(2.0, sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`(?s)INSERT INTO video_extract_frame`).
		WithArgs(taskID, 1, "/extract/"+taskID+"/frames/frame_000001.jpg", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task SET frames_extracted = GREATEST`).
		WithArgs(1, sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO video_extract_frame`).
		WithArgs(taskID, 2, "/extract/"+taskID+"/frames/frame_000002.jpg", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task SET frames_extracted = GREATEST`).
		WithArgs(2, sqlmock.AnyArg(), taskID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.runTask(taskID); err != nil {
		t.Fatalf("runTask error: %v", err)
	}
	if svc.GetRuntime(taskID) != nil {
		t.Fatalf("expected runtime removed")
	}
}

func TestVideoExtractService_workerLoop_StopsOnClose(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	// runTask will fail with "任务不存在" quickly.
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	svc := &VideoExtractService{
		db:       db,
		queue:    make(chan string, 1),
		closing:  make(chan struct{}),
		runtimes: make(map[string]*videoExtractRuntime),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.workerLoop()
	}()

	svc.queue <- "  "
	svc.queue <- "missing"

	close(svc.closing)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("worker did not exit")
	}
}
