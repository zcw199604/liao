package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleRepairMediaHistory_InvalidJSON(t *testing.T) {
	app := &App{mediaUpload: &MediaUploadService{}}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairMediaHistory", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.handleRepairMediaHistory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "invalid json body") {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestHandleRepairMediaHistory_UnknownField(t *testing.T) {
	app := &App{mediaUpload: &MediaUploadService{}}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairMediaHistory", strings.NewReader(`{"unknown":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.handleRepairMediaHistory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleRepairMediaHistory_NegativeLimits(t *testing.T) {
	app := &App{mediaUpload: &MediaUploadService{}}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairMediaHistory", strings.NewReader(`{"limitMissingMd5":-1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.handleRepairMediaHistory(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rr.Body.String(), "invalid limits") {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestHandleRepairMediaHistory_EmptyBody_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)SELECT id, user_id, local_path.*FROM media_upload_history.*id > [?].*LIMIT [?]`).
		WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_path"}))

	mock.ExpectQuery(`(?s)SELECT file_md5, COUNT.*FROM media_upload_history.*HAVING COUNT.*> 1.*LIMIT [?]`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{"file_md5", "cnt"}))

	app := &App{mediaUpload: &MediaUploadService{db: wrapMySQLDB(db), fileStore: &FileStorageService{db: wrapMySQLDB(db)}}}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/repairMediaHistory", strings.NewReader(""))
	req = req.WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	app.handleRepairMediaHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v; body=%q", err, rr.Body.String())
	}
	if got, _ := out["fixMissingMd5"].(bool); !got {
		t.Fatalf("fixMissingMd5=%v, want true", out["fixMissingMd5"])
	}
	if got, _ := out["deduplicateByMd5"].(bool); !got {
		t.Fatalf("deduplicateByMd5=%v, want true", out["deduplicateByMd5"])
	}
}
