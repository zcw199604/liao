package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCorsMiddleware_PreflightWithOrigin(t *testing.T) {
	a := &App{}
	h := a.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "http://api.local/api/getSystemConfig", nil)
	req.Header.Set("Origin", "http://example.local")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.local" {
		t.Fatalf("allow-origin=%q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow-credentials=%q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Fatalf("allow-headers=%q", got)
	}
}

func TestCorsMiddleware_NoOrigin(t *testing.T) {
	a := &App{}
	nextCalled := false
	h := a.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getSystemConfig", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin=%q, want *", got)
	}
}

func TestJWTMiddleware_RejectsMissingBearer(t *testing.T) {
	a := &App{jwt: NewJWTService("secret", 24)}
	h := a.jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getSystemConfig", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(payload["code"].(float64)) != 401 {
		t.Fatalf("code=%v, want 401", payload["code"])
	}
}

func TestJWTMiddleware_AllowsWhitelistPath_GetMtPhotoThumb(t *testing.T) {
	a := &App{jwt: NewJWTService("secret", 24)}
	nextCalled := false
	h := a.jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoThumb?size=s260&md5=abc", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next called")
	}
}

func TestJWTMiddleware_AllowsWhitelistPath_DouyinDownload(t *testing.T) {
	a := &App{jwt: NewJWTService("secret", 24)}
	nextCalled := false
	h := a.jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/douyin/download?key=abc&index=0", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next called")
	}
}

func TestJWTMiddleware_AllowsWhitelistPath_DouyinCover(t *testing.T) {
	a := &App{jwt: NewJWTService("secret", 24)}
	nextCalled := false
	h := a.jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/douyin/cover?key=abc", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next called")
	}
}

func TestJWTMiddleware_AllowsValidBearer(t *testing.T) {
	jwtSvc := NewJWTService("secret", 24)
	token, err := jwtSvc.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	a := &App{jwt: jwtSvc}
	nextCalled := false
	h := a.jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getSystemConfig", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next called")
	}
}

func TestJWTMiddleware_AllowsOptionsPassthrough(t *testing.T) {
	a := &App{jwt: NewJWTService("secret", 24)}
	nextCalled := false
	h := a.jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "http://api.local/api/getSystemConfig", strings.NewReader(""))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next called")
	}
}
