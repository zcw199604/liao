package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"liao/internal/config"
)

func TestHandleAuthLogin_ParseFormError(t *testing.T) {
	a := &App{
		cfg: config.Config{AuthAccessCode: "code-1"},
		jwt: NewJWTService("secret-1", 1),
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/login", strings.NewReader("accessCode=%ZZ"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	a.handleAuthLogin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "访问码不能为空" {
		t.Fatalf("msg=%q, want %q", got, "访问码不能为空")
	}
}

func TestHandleAuthLogin_EmptyAccessCode(t *testing.T) {
	a := &App{
		cfg: config.Config{AuthAccessCode: "code-1"},
		jwt: NewJWTService("secret-1", 1),
	}

	form := url.Values{}
	form.Set("accessCode", "")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/auth/login", form)
	rr := httptest.NewRecorder()

	a.handleAuthLogin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "访问码不能为空" {
		t.Fatalf("msg=%q, want %q", got, "访问码不能为空")
	}
}

func TestHandleAuthLogin_WrongAccessCode(t *testing.T) {
	a := &App{
		cfg: config.Config{AuthAccessCode: "code-1"},
		jwt: NewJWTService("secret-1", 1),
	}

	form := url.Values{}
	form.Set("accessCode", "bad")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/auth/login", form)
	rr := httptest.NewRecorder()

	a.handleAuthLogin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "访问码错误" {
		t.Fatalf("msg=%q, want %q", got, "访问码错误")
	}
}

func TestHandleAuthLogin_Success(t *testing.T) {
	a := &App{
		cfg: config.Config{AuthAccessCode: "code-1"},
		jwt: NewJWTService("secret-1", 1),
	}

	form := url.Values{}
	form.Set("accessCode", "code-1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/auth/login", form)
	rr := httptest.NewRecorder()

	a.handleAuthLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["code"].(float64); got != 0 {
		t.Fatalf("code=%v, want 0", resp["code"])
	}
	if got, _ := resp["msg"].(string); got != "登录成功" {
		t.Fatalf("msg=%q, want %q", got, "登录成功")
	}
	if got, _ := resp["token"].(string); got == "" {
		t.Fatalf("token should not be empty")
	}
}

func TestHandleAuthLogin_TokenGenerationFailed(t *testing.T) {
	a := &App{
		cfg: config.Config{AuthAccessCode: "code-1"},
		jwt: NewJWTService("", 1),
	}

	form := url.Values{}
	form.Set("accessCode", "code-1")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/auth/login", form)
	rr := httptest.NewRecorder()

	a.handleAuthLogin(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "登录失败" {
		t.Fatalf("msg=%q, want %q", got, "登录失败")
	}
}

func TestHandleAuthVerify_MissingToken(t *testing.T) {
	a := &App{jwt: NewJWTService("secret-1", 1)}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/auth/verify", nil)
	rr := httptest.NewRecorder()
	a.handleAuthVerify(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "Token缺失" {
		t.Fatalf("msg=%q, want %q", got, "Token缺失")
	}
}

func TestHandleAuthVerify_InvalidToken(t *testing.T) {
	a := &App{jwt: NewJWTService("secret-1", 1)}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/auth/verify", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()
	a.handleAuthVerify(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["valid"].(bool); got {
		t.Fatalf("valid=true, want false")
	}
	if got, _ := resp["msg"].(string); got != "Token无效" {
		t.Fatalf("msg=%q, want %q", got, "Token无效")
	}
}

func TestHandleAuthVerify_ValidToken(t *testing.T) {
	jwtService := NewJWTService("secret-1", 1)
	token, err := jwtService.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	a := &App{jwt: jwtService}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/auth/verify", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	a.handleAuthVerify(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["valid"].(bool); !got {
		t.Fatalf("valid=false, want true")
	}
	if got, _ := resp["msg"].(string); got != "Token有效" {
		t.Fatalf("msg=%q, want %q", got, "Token有效")
	}
}

func TestHandleAuthLogin_AllowsGetWithQueryAccessCode(t *testing.T) {
	a := &App{
		cfg: config.Config{AuthAccessCode: "code-1"},
		jwt: NewJWTService("secret-1", 1),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/auth/login?accessCode=code-1", nil)
	rr := httptest.NewRecorder()

	a.handleAuthLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["msg"].(string); got != "登录成功" {
		t.Fatalf("msg=%q, want %q", got, "登录成功")
	}
	if got, _ := resp["token"].(string); got == "" {
		t.Fatalf("token should not be empty")
	}
}

func TestHandleAuthVerify_MalformedAuthorizationHeaders(t *testing.T) {
	a := &App{jwt: NewJWTService("secret-1", 1)}

	type testCase struct {
		name      string
		header    string
		wantMsg   string
		wantValid *bool
	}

	falseValue := false
	cases := []testCase{
		{
			name:      "BearerWithoutSpaceSuffix",
			header:    "Bearer",
			wantMsg:   "Token缺失",
			wantValid: nil,
		},
		{
			name:      "NonBearerPrefix",
			header:    "bad-token",
			wantMsg:   "Token缺失",
			wantValid: nil,
		},
		{
			name:      "BearerWithTab",
			header:    "Bearer	abc",
			wantMsg:   "Token缺失",
			wantValid: nil,
		},
		{
			name:      "BearerWithOnlySpaces",
			header:    "Bearer   ",
			wantMsg:   "Token无效",
			wantValid: &falseValue,
		},
		{
			name:      "BearerWithTwoTokens",
			header:    "Bearer token another",
			wantMsg:   "Token无效",
			wantValid: &falseValue,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/api/auth/verify", nil)
			req.Header.Set("Authorization", tc.header)
			rr := httptest.NewRecorder()

			a.handleAuthVerify(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
			}

			var resp map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if got, _ := resp["msg"].(string); got != tc.wantMsg {
				t.Fatalf("msg=%q, want %q", got, tc.wantMsg)
			}

			if tc.wantValid == nil {
				if _, exists := resp["valid"]; exists {
					t.Fatalf("valid should be absent for header=%q", tc.header)
				}
				return
			}

			gotValid, ok := resp["valid"].(bool)
			if !ok {
				t.Fatalf("valid type=%T, want bool", resp["valid"])
			}
			if gotValid != *tc.wantValid {
				t.Fatalf("valid=%v, want %v", gotValid, *tc.wantValid)
			}
		})
	}
}
