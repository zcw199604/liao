package app

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestImagePortResolver_GetCached_NilOrEmptyHost(t *testing.T) {
	var nilResolver *ImagePortResolver
	if got := nilResolver.GetCached("h"); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}

	r := NewImagePortResolver(nil)
	if got := r.GetCached(" "); got != "" {
		t.Fatalf("got=%q, want empty", got)
	}
}

func TestImagePortResolver_ResolveByRealRequest_EarlyReturnsAndContextDone(t *testing.T) {
	var nilResolver *ImagePortResolver
	if port, ok := nilResolver.ResolveByRealRequest(context.Background(), "h", "a.jpg", 1); ok || port != "" {
		t.Fatalf("expected false")
	}

	r := NewImagePortResolver(nil)
	if port, ok := r.ResolveByRealRequest(context.Background(), " ", "a.jpg", 1); ok || port != "" {
		t.Fatalf("expected false")
	}
	if port, ok := r.ResolveByRealRequest(context.Background(), "h", "", 1); ok || port != "" {
		t.Fatalf("expected false")
	}

	r.ports = nil
	if port, ok := r.ResolveByRealRequest(context.Background(), "h", "a.jpg", 0); ok || port != "" {
		t.Fatalf("expected false")
	}

	// ctx.Done 分支
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r = NewImagePortResolver(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			time.Sleep(20 * time.Millisecond)
			return nil, context.Canceled
		}),
	})
	r.ports = []string{"9003"}
	if port, ok := r.ResolveByRealRequest(ctx, "h", "a.jpg", 2048); ok || port != "" {
		t.Fatalf("expected false")
	}
}

func TestImagePortResolver_ResolveByRealRequest_CachedPortHitAndClear(t *testing.T) {
	{
		minBytes := int64(8)
		r := NewImagePortResolver(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				res := &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewBuffer(bytes.Repeat([]byte("x"), int(minBytes)))),
					Request:    req,
				}
				res.Header.Set("Content-Type", "image/png")
				return res, nil
			}),
		})
		r.ports = nil
		r.cache["h"] = "9003"

		port, ok := r.ResolveByRealRequest(context.Background(), "h", "a.jpg", minBytes)
		if !ok || port != "9003" {
			t.Fatalf("port=%q ok=%v", port, ok)
		}
	}

	{
		r := NewImagePortResolver(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				res := &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("nope")),
					Request:    req,
				}
				res.Header.Set("Content-Type", "image/png")
				return res, nil
			}),
		})
		r.ports = nil
		r.cache["h"] = "9003"

		port, ok := r.ResolveByRealRequest(context.Background(), "h", "a.jpg", 8)
		if ok || port != "" {
			t.Fatalf("expected miss, got port=%q ok=%v", port, ok)
		}
		if got := r.GetCached("h"); got != "" {
			t.Fatalf("expected cache cleared, got=%q", got)
		}
	}
}

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

func TestImagePortResolver_ResolveByRealRequest_RaceChoosesFastest(t *testing.T) {
	payload := strings.Repeat("x", 3000)

	slowListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen slow failed: %v", err)
	}
	slow := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/img/Upload/") {
			time.Sleep(300 * time.Millisecond)
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte(payload))
			return
		}
		http.NotFound(w, r)
	}))
	slow.Listener = slowListener
	slow.Start()
	defer slow.Close()

	fastListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fast failed: %v", err)
	}
	fast := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/img/Upload/") {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte(payload))
			return
		}
		http.NotFound(w, r)
	}))
	fast.Listener = fastListener
	fast.Start()
	defer fast.Close()

	uSlow, _ := url.Parse(slow.URL)
	host, slowPort, _ := net.SplitHostPort(uSlow.Host)
	uFast, _ := url.Parse(fast.URL)
	_, fastPort, _ := net.SplitHostPort(uFast.Host)

	resolver := NewImagePortResolver(&http.Client{})
	resolver.ports = []string{slowPort, fastPort}

	got, ok := resolver.ResolveByRealRequest(context.Background(), host, "a.jpg", 2048)
	if !ok {
		t.Fatalf("ResolveByRealRequest ok=false, want true")
	}
	if got != fastPort {
		t.Fatalf("port=%q, want %q", got, fastPort)
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
	if got := normalizeRemoteUploadPath("   "); got != "" {
		t.Fatalf("expected empty, got=%q", got)
	}
	if got := normalizeRemoteUploadPath("?x=1"); got != "" {
		t.Fatalf("expected empty, got=%q", got)
	}
	if got := normalizeRemoteUploadPath("http://h:1/img/Upload/a.png"); got != "a.png" {
		t.Fatalf("got=%q, want %q", got, "a.png")
	}
	if got := normalizeRemoteUploadPath("/img/Upload/2026/01/a.png"); got != "2026/01/a.png" {
		t.Fatalf("got=%q, want %q", got, "2026/01/a.png")
	}
	if got := normalizeRemoteUploadPath("/img/Upload/a.png?x=1#y"); got != "a.png" {
		t.Fatalf("got=%q, want %q", got, "a.png")
	}
	if got := normalizeRemoteUploadPath("../x.png"); got != "" {
		t.Fatalf("expected empty for traversal, got=%q", got)
	}
	if got := normalizeRemoteUploadPath(strings.Repeat("a", 1025)); got != "" {
		t.Fatalf("expected empty for too long, got=%q", got)
	}
}

func TestImagePortResolver_ClearHostAndClearAll(t *testing.T) {
	var nilResolver *ImagePortResolver
	nilResolver.ClearHost("h")
	nilResolver.ClearAll()

	r := NewImagePortResolver(nil)
	r.cache["h1"] = "9003"
	r.cache["h2"] = "8003"

	r.ClearHost(" ")
	if r.GetCached("h1") == "" {
		t.Fatalf("unexpected delete")
	}

	r.ClearHost("h1")
	if got := r.GetCached("h1"); got != "" {
		t.Fatalf("expected cleared, got=%q", got)
	}

	r.ClearAll()
	if got := r.GetCached("h2"); got != "" {
		t.Fatalf("expected cleared, got=%q", got)
	}
}

func TestImagePortResolver_CheckPort_EdgeCases(t *testing.T) {
	r := NewImagePortResolver(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", 10))),
			Request:    req,
		}
		res.Header.Set("Content-Type", "image/jpeg")
		return res, nil
	})})

	if r.checkPort(context.Background(), "", "9003", "a.jpg", 1) {
		t.Fatalf("expected false")
	}

	// NewRequestWithContext error
	if r.checkPort(context.Background(), "bad host", "9003", "a.jpg", 1) {
		t.Fatalf("expected false")
	}

	// status code reject
	r2 := NewImagePortResolver(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		res := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("oops")),
			Request:    req,
		}
		res.Header.Set("Content-Type", "image/jpeg")
		return res, nil
	})})
	if r2.checkPort(context.Background(), "h", "9003", "a.jpg", 1) {
		t.Fatalf("expected false")
	}

	// limit clamp + too small after clamp
	minBytes := int64(70 * 1024)
	r3 := NewImagePortResolver(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBuffer(bytes.Repeat([]byte("x"), int(minBytes)))),
			Request:    req,
		}
		res.Header.Set("Content-Type", "image/jpeg")
		return res, nil
	})})
	if r3.checkPort(context.Background(), "h", "9003", "a.jpg", minBytes) {
		t.Fatalf("expected false")
	}

	// read error
	r4 := NewImagePortResolver(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       errReadCloser{},
			Request:    req,
		}
		res.Header.Set("Content-Type", "image/jpeg")
		return res, nil
	})})
	if r4.checkPort(context.Background(), "h", "9003", "a.jpg", 1) {
		t.Fatalf("expected false")
	}

	// "<" head reject
	r5 := NewImagePortResolver(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("<html>not really html</html>")),
			Request:    req,
		}
		res.Header.Set("Content-Type", "image/jpeg")
		return res, nil
	})})
	if r5.checkPort(context.Background(), "h", "9003", "a.jpg", 1) {
		t.Fatalf("expected false")
	}
}
