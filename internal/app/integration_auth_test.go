package app

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"liao/internal/config"
)

func TestIntegrationAuth_DisabledReturns404(t *testing.T) {
	a := &App{cfg: config.Config{}}
	handler := a.integrationAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}

func TestIntegrationAuth_MissingAndWrongToken(t *testing.T) {
	a := &App{cfg: config.Config{IntegrationToken: "0123456789abcdef01234567"}}
	handler := a.integrationAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, token := range []string{"", "wrong-token"} {
		req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("token=%q status=%d, want 401", token, rec.Code)
		}
	}
}

func TestIntegrationAuth_AcceptsCorrectToken(t *testing.T) {
	const token = "0123456789abcdef01234567"
	a := &App{cfg: config.Config{IntegrationToken: token}}
	handler := a.integrationAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media", nil)
	req.Header.Set("Authorization", "Bearer  "+token+"  ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

func TestIntegrationAuth_IPAllowlist(t *testing.T) {
	const token = "0123456789abcdef01234567"
	a := &App{cfg: config.Config{
		IntegrationToken:          token,
		IntegrationAllowedIPs:     []string{"10.0.0.0/8"},
		IntegrationTrustedProxies: []string{"192.0.2.1"},
	}}
	handler := a.integrationAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name string
		ip   string
		want int
	}{
		{name: "allowed", ip: "10.1.2.3", want: http.StatusOK},
		{name: "denied", ip: "192.168.1.1", want: http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/integration/v1/media", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Real-IP", tc.ip)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestIntegrationIPAllowed_Table(t *testing.T) {
	rules := []string{"127.0.0.1", "10.0.0.0/8", "2001:db8::/32"}
	cases := []struct {
		ip   string
		want bool
	}{
		{ip: "127.0.0.1", want: true},
		{ip: "10.20.30.40", want: true},
		{ip: "2001:db8::1", want: true},
		{ip: "11.0.0.1", want: false},
		{ip: "", want: false},
	}
	for _, tc := range cases {
		got := integrationIPAllowed(parseIP(tc.ip), rules)
		if got != tc.want {
			t.Fatalf("ip=%q got=%v want=%v", tc.ip, got, tc.want)
		}
	}
}

func TestIntegrationClientIP_PrefersProxyHeaders(t *testing.T) {
	app := &App{cfg: config.Config{IntegrationTrustedProxies: []string{"192.0.2.10", "10.0.0.1"}}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	if got := app.integrationClientIP(req); !got.Equal(net.ParseIP("203.0.113.5")) {
		t.Fatalf("got=%v", got)
	}

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "198.51.100.7")
	if got := app.integrationClientIP(req); !got.Equal(net.ParseIP("198.51.100.7")) {
		t.Fatalf("got=%v", got)
	}

	// XFF 存在但只有可信代理时，不回退 X-Real-IP。
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("X-Real-IP", "198.51.100.7")
	if got := app.integrationClientIP(req); !got.Equal(net.ParseIP("192.0.2.10")) {
		t.Fatalf("trusted-only XFF should use RemoteAddr, got=%v", got)
	}

	untrusted := &App{cfg: config.Config{IntegrationTrustedProxies: []string{"10.0.0.1"}}}
	if got := untrusted.integrationClientIP(req); !got.Equal(net.ParseIP("192.0.2.10")) {
		t.Fatalf("untrusted proxy should use RemoteAddr, got=%v", got)
	}
	if app.integrationClientIP(nil) != nil {
		t.Fatalf("nil request should return nil")
	}
}

func TestIntegrationClientIP_HeaderParsingEdgeCases(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-a-host-port"
	req.Header.Set("X-Forwarded-For", " , ")
	if got := (&App{}).integrationClientIP(req); got != nil {
		t.Fatalf("got=%v, want nil", got)
	}
	if got := firstHeaderIP(strings.Repeat(" ", 3)); got != nil {
		t.Fatalf("got=%v, want nil", got)
	}
}
