package app

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetryUploadPNGAsJPEGIfNeeded_MoreBranches(t *testing.T) {
	initialResp := `{"state":"fail","code":-3}`

	t.Run("nil file header", func(t *testing.T) {
		app := &App{}
		got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", nil, "image/png", initialResp, "", "r", "ua")
		if got != initialResp {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("open file error", func(t *testing.T) {
		req, fh := newMultipartRequest(t, http.MethodPost, "http://api.local/upload", "file", "a.png", "image/png", []byte("png"), map[string]string{"userid": "u1"})
		_ = req

		oldOpen := openMultipartFileHeaderFn
		openMultipartFileHeaderFn = func(*multipart.FileHeader) (multipart.File, error) {
			return nil, errors.New("open fail")
		}
		t.Cleanup(func() { openMultipartFileHeaderFn = oldOpen })

		app := &App{}
		got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", fh, "image/png", initialResp, "", "r", "ua")
		if got != initialResp {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("read original bytes error", func(t *testing.T) {
		req, fh := newMultipartRequest(t, http.MethodPost, "http://api.local/upload", "file", "a.png", "image/png", []byte("png"), map[string]string{"userid": "u1"})
		_ = req

		oldOpen := openMultipartFileHeaderFn
		openMultipartFileHeaderFn = func(*multipart.FileHeader) (multipart.File, error) {
			return &errMultipartFile{}, nil
		}
		t.Cleanup(func() { openMultipartFileHeaderFn = oldOpen })

		app := &App{}
		got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", fh, "image/png", initialResp, "", "r", "ua")
		if got != initialResp {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("convert image error", func(t *testing.T) {
		req, fh := newMultipartRequest(t, http.MethodPost, "http://api.local/upload", "file", "a.png", "image/png", []byte("not-a-real-png"), map[string]string{"userid": "u1"})
		_ = req

		app := &App{}
		got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", fh, "image/png", initialResp, "", "r", "ua")
		if got != initialResp {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("retry upload error", func(t *testing.T) {
		req, fh := newMultipartRequest(t, http.MethodPost, "http://api.local/upload", "file", "a.png", "image/png", buildTinyPNG(t), map[string]string{"userid": "u1"})
		_ = req

		app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network fail")
		})}}
		got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", fh, "image/png", initialResp, "", "r", "ua")
		if got != initialResp {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("retry upload success", func(t *testing.T) {
		req, fh := newMultipartRequest(t, http.MethodPost, "http://api.local/upload", "file", "a.png", "image/png", buildTinyPNG(t), map[string]string{"userid": "u1"})
		_ = req

		app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			_, _ = io.ReadAll(req.Body)
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(`{"state":"OK"}`)), Header: make(http.Header), Request: req}, nil
		})}}
		got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", fh, "image/png", initialResp, "c=1", "r", "ua")
		if !strings.Contains(got, `"OK"`) {
			t.Fatalf("got=%q", got)
		}
	})
}

func TestShouldRetryPNGAsJPEG_AndHelpers_MoreBranches(t *testing.T) {
	if shouldRetryPNGAsJPEG("image/png", nil, "") {
		t.Fatalf("empty response should not retry")
	}
	if !shouldRetryPNGAsJPEG("image/png", nil, `{"state":"fail"}`) {
		t.Fatalf("state fail should retry")
	}
	if shouldRetryPNGAsJPEG("image/png", nil, `{"state":"other","msg":"ok"}`) {
		t.Fatalf("state other should not retry")
	}
	if !shouldRetryPNGAsJPEG("image/png", nil, `not-json "state":"error"`) {
		t.Fatalf("fallback state error should retry")
	}
	if !shouldRetryPNGAsJPEG("image/png", nil, `not-json "code":-3`) {
		t.Fatalf("fallback code -3 should retry")
	}
	if shouldRetryPNGAsJPEG("image/png", nil, `not-json`) {
		t.Fatalf("plain invalid json should not retry")
	}

	if isLikelyPNGUpload("application/octet-stream", nil) {
		t.Fatalf("nil header and non-png content type should be false")
	}

	fh := &multipart.FileHeader{Filename: "a.png"}
	if !isLikelyPNGUpload("application/octet-stream", fh) {
		t.Fatalf(".png filename should be true")
	}

	if got := truncateStringForLog(" abc ", 0); got != "abc" {
		t.Fatalf("got=%q", got)
	}
	if got := truncateStringForLog("abcdef", 3); !strings.HasPrefix(got, "abc") || !strings.Contains(got, "truncated") {
		t.Fatalf("got=%q", got)
	}
}

func TestUploadBytesToUpstream_MoreBranches(t *testing.T) {
	ctx := context.Background()
	app := &App{}

	t.Run("bad URL", func(t *testing.T) {
		if _, err := app.uploadBytesToUpstream(ctx, "http://[::1", "example.com:9003", "a.jpg", []byte("x"), "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("Do error", func(t *testing.T) {
		app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network fail")
		})}}
		if _, err := app.uploadBytesToUpstream(ctx, "http://example.com/upload", "example.com:9003", "a.jpg", []byte("x"), "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("read body error", func(t *testing.T) {
		app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: errReadCloserUserHistory{}, Header: make(http.Header)}, nil
		})}}
		if _, err := app.uploadBytesToUpstream(ctx, "http://example.com/upload", "example.com:9003", "a.jpg", []byte("x"), "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("non-2xx", func(t *testing.T) {
		app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Status: "502 Bad Gateway", Body: io.NopCloser(strings.NewReader("bad")), Header: make(http.Header)}, nil
		})}}
		if _, err := app.uploadBytesToUpstream(ctx, "http://example.com/upload", "example.com:9003", "a.jpg", []byte("x"), "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success with cookie", func(t *testing.T) {
		app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Cookie") != "c=1" {
				t.Fatalf("cookie=%q", req.Header.Get("Cookie"))
			}
			_, _ = io.ReadAll(req.Body)
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("OK")), Header: make(http.Header), Request: req}, nil
		})}}
		got, err := app.uploadBytesToUpstream(ctx, "http://example.com/upload", "example.com:9003", "a.jpg", []byte("x"), "c=1", "r", "ua")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if got != "OK" {
			t.Fatalf("got=%q", got)
		}
	})
}

func TestRetryUploadPNGAsJPEGIfNeeded_NonPNGNoRetry(t *testing.T) {
	req, fh := newMultipartRequest(t, http.MethodPost, "http://api.local/upload", "file", "a.jpg", "image/jpeg", []byte("jpg"), map[string]string{"userid": "u1"})
	_ = req
	app := &App{}
	got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", fh, "image/jpeg", `{"state":"fail"}`, "", "r", "ua")
	if got != `{"state":"fail"}` {
		t.Fatalf("got=%q", got)
	}
}

func TestUploadBytesToUpstream_HostHeaderSet(t *testing.T) {
	app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Host != "example.com" {
			t.Fatalf("host=%q", req.Host)
		}
		_, _ = io.ReadAll(req.Body)
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("OK")), Header: make(http.Header), Request: req}, nil
	})}}
	if _, err := app.uploadBytesToUpstream(context.Background(), "http://example.com/upload", "example.com:9003", "a.jpg", []byte("x"), "", "r", "ua"); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestRetryUploadPNGAsJPEGIfNeeded_RealFileOpenPath(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "a.png")
	if err := os.WriteFile(filePath, buildTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	req, fh := newMultipartRequest(t, http.MethodPost, "http://api.local/upload", "file", "a.png", "image/png", buildTinyPNG(t), map[string]string{"userid": "u1"})
	_ = req

	app := &App{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		_, _ = io.ReadAll(req.Body)
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(`{"state":"OK"}`)), Header: make(http.Header), Request: req}, nil
	})}}

	got := app.retryUploadPNGAsJPEGIfNeeded(context.Background(), "http://example.com/upload", "example.com:9003", fh, "image/png", `{"state":"fail","code":-3}`, "", "r", "ua")
	if !strings.Contains(got, `"OK"`) {
		t.Fatalf("got=%q", got)
	}
}
