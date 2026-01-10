package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetCachedImages_Empty(t *testing.T) {
	app := &App{
		imageCache: NewImageCacheService(),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/getCachedImages?userid=u1", nil)
	rr := httptest.NewRecorder()

	app.handleGetCachedImages(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got, _ := resp["port"].(string); got != "9006" {
		t.Fatalf("port=%q, want %q", got, "9006")
	}
	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatalf("data type=%T, want []any", resp["data"])
	}
	if len(data) != 0 {
		t.Fatalf("data len=%d, want 0", len(data))
	}
}

func TestHandleGetCachedImages_WithCache(t *testing.T) {
	oldDetect := detectAvailablePort
	detectAvailablePort = func(string) string { return "9001" }
	t.Cleanup(func() { detectAvailablePort = oldDetect })

	app := &App{
		imageCache:  NewImageCacheService(),
		imageServer: NewImageServerService("img-host", "9003"),
		mediaUpload: &MediaUploadService{serverPort: 8080},
	}

	app.imageCache.AddImageToCache("u1", "/images/2026/01/10/a.png")

	req := httptest.NewRequest(http.MethodGet, "http://internal/api/getCachedImages?userid=u1", nil)
	req.Host = "internal:1"
	req.Header.Set("X-Forwarded-Host", "public.example:99")

	rr := httptest.NewRecorder()
	app.handleGetCachedImages(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["port"].(string); got != "9001" {
		t.Fatalf("port=%q, want %q", got, "9001")
	}
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data=%v, want single item", resp["data"])
	}
	if got, _ := data[0].(string); got != "http://public.example:99/upload/images/2026/01/10/a.png" {
		t.Fatalf("url=%q, want %q", got, "http://public.example:99/upload/images/2026/01/10/a.png")
	}
}
