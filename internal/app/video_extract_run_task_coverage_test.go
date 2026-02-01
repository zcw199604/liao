package app

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"liao/internal/config"
)

func TestVideoExtractService_runTask_ReturnsNilWhenAlreadyRunning(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{},
			"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
			1, 1, sql.NullFloat64{}, sql.NullFloat64{}, VideoExtractStatusPending,
		))

	svc := &VideoExtractService{
		db:       db,
		runtimes: map[string]*videoExtractRuntime{taskID: {}},
	}
	if err := svc.runTask(taskID); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestVideoExtractService_runTask_SetStatusError_And_EndSecReachedEarly(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{},
			"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{Float64: 1.0, Valid: true}, // start_sec
			sql.NullFloat64{Float64: 2.0, Valid: true}, // end_sec
			10, 0,
			1, 1,
			sql.NullFloat64{},
			sql.NullFloat64{Float64: 2.0, Valid: true}, // cursor_out_time_sec -> startSecAbs=2.001
			VideoExtractStatusPending,
		))

	// First status update fails -> should be logged into runtime.
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnError(sql.ErrConnDone)
	// EndSec reached early -> PAUSED_LIMIT.
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:       db,
		runtimes: make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestVideoExtractService_runTask_FramesRemainingZero(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{},
			"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 1, 1,
			1, 1, sql.NullFloat64{}, sql.NullFloat64{}, VideoExtractStatusPending,
		))

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:       db,
		runtimes: make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestVideoExtractService_runTask_OutputFormatDefaultAndMkdirAllError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadRoot := t.TempDir()
	// frames dir is {baseUploadAbs}/extract/...; make "extract" a file to force MkdirAll error.
	if err := os.WriteFile(filepath.Join(uploadRoot, "extract"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write extract: %v", err)
	}

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"extract/t1", "", sql.NullInt64{}, // no leading "/" + empty outputFormat
			"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
			1, 1, sql.NullFloat64{}, sql.NullFloat64{}, VideoExtractStatusPending,
		))

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_runTask_KeyframeScene_StdoutPipeError(t *testing.T) {
	oldExec := execCommandContext
	t.Cleanup(func() { execCommandContext = oldExec })
	execCommandContext = func(context.Context, string, ...string) *exec.Cmd {
		return &exec.Cmd{Stdout: &bytes.Buffer{}}
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadRoot := t.TempDir()
	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{Int64: 3, Valid: true},
			"keyframe", sql.NullString{String: "scene", Valid: true}, sql.NullFloat64{}, sql.NullFloat64{Float64: 0.8, Valid: true},
			sql.NullFloat64{Float64: 1.0, Valid: true}, // start_sec -> -ss
			sql.NullFloat64{Float64: 3.0, Valid: true}, // end_sec -> -t
			10, 0,
			1, 1, sql.NullFloat64{}, sql.NullFloat64{}, VideoExtractStatusPending,
		))

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: "ffmpeg"},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_runTask_KeyframeIFrame_StderrPipeError(t *testing.T) {
	oldExec := execCommandContext
	t.Cleanup(func() { execCommandContext = oldExec })
	execCommandContext = func(context.Context, string, ...string) *exec.Cmd {
		return &exec.Cmd{Stderr: &bytes.Buffer{}}
	}

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadRoot := t.TempDir()
	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{},
			"keyframe", sql.NullString{String: "iframe", Valid: true}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
			1, 1, sql.NullFloat64{}, sql.NullFloat64{}, VideoExtractStatusPending,
		))

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: "ffmpeg"},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_runTask_FPS_StartError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	uploadRoot := t.TempDir()
	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/t1", "jpg", sql.NullInt64{},
			"fps", sql.NullString{}, sql.NullFloat64{Float64: 2.5, Valid: true}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
			1, 1, sql.NullFloat64{}, sql.NullFloat64{}, VideoExtractStatusPending,
		))

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: filepath.Join(t.TempDir(), "missing-ffmpeg")},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_runTask_CancelAndTickerAndStderrAndBlankLine(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	uploadRoot := t.TempDir()

	ffmpegPath := writeExecutable(t, "ffmpeg-slow", `#!/bin/sh
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

# print a blank line for scanner branch coverage
printf "\n"

echo "progress=continue"
echo "warn: test-stderr" 1>&2

# keep running long enough for ticker to fire; will be canceled by test.
sleep 5
exit 0
`)

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}

	firstRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusPending,
	)
	secondRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusRunning,
	)

	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(firstRow)
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(secondRow)

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: ffmpegPath},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		queue:     make(chan string, 1),
		closing:   make(chan struct{}),
		runtimes:  make(map[string]*videoExtractRuntime),
	}

	done := make(chan error, 1)
	go func() { done <- svc.runTask(taskID) }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting runtime")
		}
		if rt := svc.GetRuntime(taskID); rt != nil {
			rt.mu.Lock()
			ready := rt.cmd != nil
			rt.mu.Unlock()
			if ready {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	// ensure ticker has time to tick at least once.
	time.Sleep(1200 * time.Millisecond)
	if !svc.CancelTask(taskID) {
		t.Fatalf("expected cancel true")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting runTask")
	}
}

func TestVideoExtractService_runTask_LoadTaskRowAfterRunError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	uploadRoot := t.TempDir()
	ffmpegPath := writeExecutable(t, "ffmpeg-ok", "#!/bin/sh\nexit 0\n")

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			taskID, "upload", "/in.mp4", "/tmp/in.mp4",
			"/extract/"+taskID, "jpg", sql.NullInt64{},
			"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
			sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
			1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusPending,
		))

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnError(sql.ErrConnDone)

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: ffmpegPath},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_runTask_FFmpegExitError_UsesLastStderrLine(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	uploadRoot := t.TempDir()
	ffmpegPath := writeExecutable(t, "ffmpeg-fail", "#!/bin/sh\necho boom 1>&2\nexit 7\n")

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}

	firstRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusPending,
	)
	secondRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusRunning,
	)

	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(firstRow)
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(secondRow)

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: ffmpegPath},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_runTask_FFmpegExitError_FallbackToWaitErr(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	uploadRoot := t.TempDir()
	ffmpegPath := writeExecutable(t, "ffmpeg-fail-no-stderr", "#!/bin/sh\nexit 7\n")

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}

	firstRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusPending,
	)
	secondRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusRunning,
	)

	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(firstRow)
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(secondRow)

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: ffmpegPath},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVideoExtractService_runTask_EndSecStopReason(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()
	mock.MatchExpectationsInOrder(false)

	uploadRoot := t.TempDir()
	ffmpegPath := writeExecutable(t, "ffmpeg-ok", "#!/bin/sh\nexit 0\n")

	taskID := "t1"
	cols := []string{
		"task_id", "source_type", "source_ref", "input_abs_path",
		"output_dir_local_path", "output_format", "jpg_quality",
		"mode", "keyframe_mode", "fps", "scene_threshold",
		"start_sec", "end_sec", "max_frames_total", "frames_extracted",
		"video_width", "video_height", "duration_sec", "cursor_out_time_sec", "status",
	}

	firstRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{}, sql.NullFloat64{}, 10, 0,
		1, 1, sql.NullFloat64{Float64: 10, Valid: true}, sql.NullFloat64{}, VideoExtractStatusPending,
	)
	secondRow := sqlmock.NewRows(cols).AddRow(
		taskID, "upload", "/in.mp4", "/tmp/in.mp4",
		"/extract/"+taskID, "jpg", sql.NullInt64{},
		"all", sql.NullString{}, sql.NullFloat64{}, sql.NullFloat64{},
		sql.NullFloat64{},
		sql.NullFloat64{Float64: 3.0, Valid: true}, // end_sec
		10, 1,
		1, 1,
		sql.NullFloat64{Float64: 10.0, Valid: true}, // duration_sec
		sql.NullFloat64{Float64: 3.0, Valid: true},  // cursor_out_time_sec
		VideoExtractStatusRunning,
	)

	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(firstRow)
	mock.ExpectQuery(`(?s)SELECT task_id, source_type, source_ref, input_abs_path.*FROM video_extract_task WHERE task_id = [?]`).
		WithArgs(taskID).
		WillReturnRows(secondRow)

	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE video_extract_task\s+SET status =`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &VideoExtractService{
		db:        db,
		cfg:       config.Config{FFmpegPath: ffmpegPath},
		fileStore: &FileStorageService{baseUploadAbs: uploadRoot},
		runtimes:  make(map[string]*videoExtractRuntime),
	}
	if err := svc.runTask(taskID); err != nil {
		t.Fatalf("err=%v", err)
	}
}
