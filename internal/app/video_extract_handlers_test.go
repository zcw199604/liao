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
)

func TestHandleUploadVideoExtractInput_SaveTemp(t *testing.T) {
	tempDir := t.TempDir()

	app := &App{
		fileStorage: &FileStorageService{baseUploadAbs: tempDir, baseTempAbs: tempDir},
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "a.mp4")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("x")); err != nil {
		t.Fatalf("write part: %v", err)
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "http://api.local:8080/api/uploadVideoExtractInput", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	app.handleUploadVideoExtractInput(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["code"].(float64); int(got) != 0 {
		t.Fatalf("code=%v, want 0", resp["code"])
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("data=%T, want map", resp["data"])
	}
	localPath, _ := data["localPath"].(string)
	if !strings.HasPrefix(localPath, "/tmp/video_extract_inputs/") {
		t.Fatalf("localPath=%q, want prefix %q", localPath, "/tmp/video_extract_inputs/")
	}

	inner := strings.TrimPrefix(localPath, "/tmp/video_extract_inputs/")
	full := filepath.Join(tempDir, filepath.FromSlash(inner))
	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		t.Fatalf("saved file missing: %v", err)
	}
}

func TestHandleCleanupVideoExtractInput_Success(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{
		fileStorage: &FileStorageService{baseUploadAbs: tempDir, baseTempAbs: tempDir},
	}

	localPath := "/tmp/video_extract_inputs/2026/01/21/a.mp4"
	inner := strings.TrimPrefix(localPath, "/tmp/video_extract_inputs/")
	full := filepath.Join(tempDir, filepath.FromSlash(inner))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://api.local:8080/api/cleanupVideoExtractInput", bytes.NewBufferString(`{"localPath":"`+localPath+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	app.handleCleanupVideoExtractInput(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["code"].(float64); int(got) != 0 {
		t.Fatalf("code=%v, want 0", resp["code"])
	}

	if _, err := os.Stat(full); err == nil {
		t.Fatalf("expected file deleted")
	}
}

func TestHandleCleanupVideoExtractInput_RejectNonTempPath(t *testing.T) {
	tempDir := t.TempDir()
	app := &App{
		fileStorage: &FileStorageService{baseUploadAbs: tempDir, baseTempAbs: tempDir},
	}

	req := httptest.NewRequest(http.MethodPost, "http://api.local:8080/api/cleanupVideoExtractInput", bytes.NewBufferString(`{"localPath":"/videos/a.mp4"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	app.handleCleanupVideoExtractInput(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}
