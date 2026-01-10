package app

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestImagePortResolver_ResolveByRealRequest_Success(t *testing.T) {
	payload := strings.Repeat("x", 3000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/img/Upload/") {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte(payload))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url failed: %v", err)
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split host port failed: %v", err)
	}

	resolver := NewImagePortResolver(srv.Client())
	resolver.ports = []string{"1", "2", port}

	got, ok := resolver.ResolveByRealRequest(context.Background(), host, "a.jpg", 2048)
	if !ok {
		t.Fatalf("ResolveByRealRequest ok=false, want true")
	}
	if got != port {
		t.Fatalf("port=%q, want %q", got, port)
	}
}

func TestImagePortResolver_ResolveByRealRequest_RejectsHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(strings.Repeat("x", 5000)))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	host, port, _ := net.SplitHostPort(u.Host)

	resolver := NewImagePortResolver(srv.Client())
	resolver.ports = []string{port}

	if got, ok := resolver.ResolveByRealRequest(context.Background(), host, "a.jpg", 2048); ok || got != "" {
		t.Fatalf("expected reject html, got port=%q ok=%v", got, ok)
	}
}

func TestImagePortResolver_ResolveByRealRequest_RejectsTooSmall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(strings.Repeat("x", 10)))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	host, port, _ := net.SplitHostPort(u.Host)

	resolver := NewImagePortResolver(srv.Client())
	resolver.ports = []string{port}

	if got, ok := resolver.ResolveByRealRequest(context.Background(), host, "a.png", 256); ok || got != "" {
		t.Fatalf("expected reject too small, got port=%q ok=%v", got, ok)
	}
}

func TestNormalizeRemoteUploadPath(t *testing.T) {
	if got := normalizeRemoteUploadPath("http://h:1/img/Upload/a.png"); got != "a.png" {
		t.Fatalf("got=%q, want %q", got, "a.png")
	}
	if got := normalizeRemoteUploadPath("/img/Upload/2026/01/a.png"); got != "2026/01/a.png" {
		t.Fatalf("got=%q, want %q", got, "2026/01/a.png")
	}
	if got := normalizeRemoteUploadPath("../x.png"); got != "" {
		t.Fatalf("expected empty for traversal, got=%q", got)
	}
}

