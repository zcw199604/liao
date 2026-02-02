package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mime/multipart"

	"liao/internal/config"
)

func TestHandleUploadVideoExtractInput_CoverageBranches(t *testing.T) {
	t.Run("not init", func(t *testing.T) {
		var a *App
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/uploadVideoExtractInput", nil)
		rec := httptest.NewRecorder()
		a.handleUploadVideoExtractInput(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("ParseMultipartForm error", func(t *testing.T) {
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: t.TempDir()}}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/uploadVideoExtractInput", bytes.NewBufferString("x"))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		app.handleUploadVideoExtractInput(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("missing file field", func(t *testing.T) {
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: t.TempDir()}}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/uploadVideoExtractInput", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		app.handleUploadVideoExtractInput(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("SaveTempVideoExtractInput error", func(t *testing.T) {
		baseTempAbsFile := filepath.Join(t.TempDir(), "tempfile")
		if err := os.WriteFile(baseTempAbsFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: baseTempAbsFile}}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "a.mp4")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		_, _ = part.Write([]byte("x"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/uploadVideoExtractInput", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		app.handleUploadVideoExtractInput(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandleCleanupVideoExtractInput_CoverageBranches(t *testing.T) {
	t.Run("not init", func(t *testing.T) {
		var a *App
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cleanupVideoExtractInput", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		a.handleCleanupVideoExtractInput(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: t.TempDir()}}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cleanupVideoExtractInput", bytes.NewBufferString("{"))
		rec := httptest.NewRecorder()
		app.handleCleanupVideoExtractInput(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("localPath empty", func(t *testing.T) {
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: t.TempDir()}}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cleanupVideoExtractInput", bytes.NewBufferString(`{"localPath":" "}`))
		rec := httptest.NewRecorder()
		app.handleCleanupVideoExtractInput(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("default baseTempAbs and file missing => deleted=false", func(t *testing.T) {
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: ""}}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cleanupVideoExtractInput", bytes.NewBufferString(`{"localPath":"/tmp/video_extract_inputs/2099/01/01/not-exist.mp4"}`))
		rec := httptest.NewRecorder()
		app.handleCleanupVideoExtractInput(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"deleted":false`) {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})

	t.Run("invalid cleanInner", func(t *testing.T) {
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: t.TempDir()}}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cleanupVideoExtractInput", bytes.NewBufferString(`{"localPath":"/tmp/video_extract_inputs/"}`))
		rec := httptest.NewRecorder()
		app.handleCleanupVideoExtractInput(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("path escape rejected", func(t *testing.T) {
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: t.TempDir()}}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cleanupVideoExtractInput", bytes.NewBufferString(`{"localPath":"/tmp/video_extract_inputs/../x.mp4"}`))
		rec := httptest.NewRecorder()
		app.handleCleanupVideoExtractInput(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("os.Remove error", func(t *testing.T) {
		baseTempAbs := t.TempDir()
		app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir(), baseTempAbs: baseTempAbs}}

		localPath := "/tmp/video_extract_inputs/2026/01/21/a.mp4"
		inner := strings.TrimPrefix(localPath, "/tmp/video_extract_inputs/")
		full := filepath.Join(baseTempAbs, filepath.FromSlash(inner))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		// Make parent dir non-writable so os.Remove fails.
		parent := filepath.Dir(full)
		if err := os.Chmod(parent, 0o555); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cleanupVideoExtractInput", bytes.NewBufferString(`{"localPath":"`+localPath+`"}`))
		rec := httptest.NewRecorder()
		app.handleCleanupVideoExtractInput(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestVideoExtractHandlers_OtherErrorBranches(t *testing.T) {
	t.Run("handleCreateVideoExtractTask CreateTask error", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()
		app := &App{videoExtract: &VideoExtractService{db: wrapMySQLDB(db), fileStore: &FileStorageService{baseUploadAbs: t.TempDir()}}}
		body, _ := json.Marshal(VideoExtractCreateRequest{SourceType: VideoExtractSourceUpload, LocalPath: "/x.mp4", Mode: VideoExtractModeAll, MaxFrames: 0})
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/createVideoExtractTask", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.handleCreateVideoExtractTask(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("handleGetVideoExtractTaskList ListTasks error", func(t *testing.T) {
		app := &App{videoExtract: &VideoExtractService{}}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getVideoExtractTaskList", nil)
		rec := httptest.NewRecorder()
		app.handleGetVideoExtractTaskList(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("handleGetVideoExtractTaskDetail not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getVideoExtractTaskDetail?taskId=t1", nil)
		rec := httptest.NewRecorder()
		(&App{}).handleGetVideoExtractTaskDetail(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("handleGetVideoExtractTaskDetail pageSize<=0 and service error", func(t *testing.T) {
		app := &App{videoExtract: &VideoExtractService{}, cfg: config.Config{VideoExtractFramePageSz: 2}}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getVideoExtractTaskDetail?taskId=t1&pageSize=-1", nil)
		rec := httptest.NewRecorder()
		app.handleGetVideoExtractTaskDetail(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("handleCancelVideoExtractTask not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cancelVideoExtractTask", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		(&App{}).handleCancelVideoExtractTask(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("handleCancelVideoExtractTask CancelAndMark error", func(t *testing.T) {
		app := &App{videoExtract: &VideoExtractService{runtimes: make(map[string]*videoExtractRuntime)}}
		body, _ := json.Marshal(VideoExtractCancelRequest{TaskID: " "})
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/cancelVideoExtractTask", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.handleCancelVideoExtractTask(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("handleContinueVideoExtractTask not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/continueVideoExtractTask", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		(&App{}).handleContinueVideoExtractTask(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("handleContinueVideoExtractTask bad json", func(t *testing.T) {
		app := &App{videoExtract: &VideoExtractService{}}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/continueVideoExtractTask", bytes.NewBufferString("{"))
		rec := httptest.NewRecorder()
		app.handleContinueVideoExtractTask(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("handleDeleteVideoExtractTask not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/deleteVideoExtractTask", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		(&App{}).handleDeleteVideoExtractTask(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("handleDeleteVideoExtractTask DeleteTask error", func(t *testing.T) {
		app := &App{videoExtract: &VideoExtractService{}}
		body, _ := json.Marshal(VideoExtractDeleteRequest{TaskID: "t1", DeleteFiles: false})
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/deleteVideoExtractTask", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		app.handleDeleteVideoExtractTask(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
