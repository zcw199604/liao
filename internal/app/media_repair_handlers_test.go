package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
