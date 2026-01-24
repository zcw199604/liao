package app

import (
	"net/http/httptest"
	"testing"
)

func TestRequestHostHeader(t *testing.T) {
	if got := requestHostHeader(nil); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Host = " origin.example.com "
	if got := requestHostHeader(req); got != "origin.example.com" {
		t.Fatalf("got=%q, want %q", got, "origin.example.com")
	}

	req2 := httptest.NewRequest("GET", "http://example.com/", nil)
	req2.Header.Set("X-Forwarded-Host", "public.example.com, internal.example.com")
	if got := requestHostHeader(req2); got != "public.example.com" {
		t.Fatalf("got=%q, want %q", got, "public.example.com")
	}

	req3 := httptest.NewRequest("GET", "http://example.com/", nil)
	req3.Header.Set("X-Forwarded-Host", ",fallback.example.com")
	if got := requestHostHeader(req3); got != ",fallback.example.com" {
		t.Fatalf("got=%q, want %q", got, ",fallback.example.com")
	}
}
