package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func decodeJSONBody(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestHandleGetConnectionStats_NoManager(t *testing.T) {
	a := &App{}
	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getConnectionStats", nil)
	rec := httptest.NewRecorder()
	a.handleGetConnectionStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != -1 {
		t.Fatalf("code=%v, want -1", got["code"])
	}
}

func TestHandleGetConnectionStats_WithManager(t *testing.T) {
	a := &App{
		wsManager: NewUpstreamWebSocketManager(nil, "ws://localhost:9999", NewForceoutManager(), NewMemoryUserInfoCacheService(), nil),
	}
	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getConnectionStats", nil)
	rec := httptest.NewRecorder()
	a.handleGetConnectionStats(rec, req)

	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", got["code"])
	}
	data := got["data"].(map[string]any)
	if int(data["active"].(float64)) != 0 {
		t.Fatalf("active=%v, want 0", data["active"])
	}
}

func TestHandleForceoutCountAndClear(t *testing.T) {
	m := NewForceoutManager()
	m.AddForceoutUser("u1")
	a := &App{forceoutManager: m}

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getForceoutUserCount", nil)
	rec := httptest.NewRecorder()
	a.handleGetForceoutUserCount(rec, req)
	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", got["code"])
	}
	if int(got["data"].(float64)) != 1 {
		t.Fatalf("data=%v, want 1", got["data"])
	}

	req2 := httptest.NewRequest(http.MethodPost, "http://api.local/api/clearForceoutUsers", nil)
	rec2 := httptest.NewRecorder()
	a.handleClearForceoutUsers(rec2, req2)
	got2 := decodeJSONBody(t, rec2.Body)
	if int(got2["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", got2["code"])
	}
}

func TestHandleDeleteUpstreamUser_ValidatesParams(t *testing.T) {
	a := &App{}
	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/deleteUpstreamUser", nil)
	rec := httptest.NewRecorder()
	a.handleDeleteUpstreamUser(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestHandleDeleteUpstreamUser_UsesHTTPClient(t *testing.T) {
	body := "OK"
	a := &App{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				res := &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Request:    r,
				}
				return res, nil
			}),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "http://api.local/api/deleteUpstreamUser", bytes.NewBufferString("myUserId=1&userToId=2"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	a.handleDeleteUpstreamUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	got := decodeJSONBody(t, rec.Body)
	if int(got["code"].(float64)) != 0 {
		t.Fatalf("code=%v, want 0", got["code"])
	}
	if got["data"].(string) != body {
		t.Fatalf("data=%v, want %q", got["data"], body)
	}
}
