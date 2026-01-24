package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type errReadCloserUploadAbs struct{}

func (errReadCloserUploadAbs) Read(p []byte) (int, error) { return 0, errors.New("read err") }
func (errReadCloserUploadAbs) Close() error               { return nil }

func TestMtPhotoHandlerHelpers_CoversBranches(t *testing.T) {
	t.Run("inferContentTypeFromFilename", func(t *testing.T) {
		cases := []struct {
			in   string
			want string
		}{
			{"a.jpg", "image/jpeg"},
			{"a.jpeg", "image/jpeg"},
			{"a.png", "image/png"},
			{"a.gif", "image/gif"},
			{"a.webp", "image/webp"},
			{"a.mp4", "video/mp4"},
			{"a.unknown", ""},
			{"  ", ""},
		}
		for _, tc := range cases {
			if got := inferContentTypeFromFilename(tc.in); got != tc.want {
				t.Fatalf("inferContentTypeFromFilename(%q)=%q, want %q", tc.in, got, tc.want)
			}
		}
	})

	t.Run("isValidMD5Hex", func(t *testing.T) {
		if isValidMD5Hex("") {
			t.Fatalf("expected false")
		}
		if isValidMD5Hex("0123") {
			t.Fatalf("expected false")
		}
		if isValidMD5Hex(strings.Repeat("g", 32)) {
			t.Fatalf("expected false")
		}
		if !isValidMD5Hex("0123456789abcdef0123456789ABCDEF") {
			t.Fatalf("expected true")
		}
	})

	t.Run("guessExtFromContentType", func(t *testing.T) {
		cases := []struct {
			in   string
			want string
		}{
			{"", ""},
			{"image/jpeg", ".jpg"},
			{"image/jpeg; charset=utf-8", ".jpg"},
			{"image/png", ".png"},
			{"image/gif", ".gif"},
			{"image/webp", ".webp"},
			{"video/mp4", ".mp4"},
			{"text/plain", ""},
		}
		for _, tc := range cases {
			if got := guessExtFromContentType(tc.in); got != tc.want {
				t.Fatalf("guessExtFromContentType(%q)=%q, want %q", tc.in, got, tc.want)
			}
		}
	})

	t.Run("buildDownloadContentDisposition", func(t *testing.T) {
		md5Value := "0123456789abcdef0123456789abcdef"
		disp := buildDownloadContentDisposition("", md5Value, "image/jpeg")
		if !strings.Contains(disp, md5Value+".jpg") {
			t.Fatalf("disp=%q", disp)
		}

		disp = buildDownloadContentDisposition("hello world.png", md5Value, "")
		if !strings.Contains(disp, "hello%20world.png") {
			t.Fatalf("disp=%q", disp)
		}
	})
}

func TestUploadAbsPathToUpstream_CoversBranches(t *testing.T) {
	tempDir := t.TempDir()
	absFile := filepath.Join(tempDir, "a.txt")
	if err := os.WriteFile(absFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Run("bad url", func(t *testing.T) {
		a := &App{httpClient: &http.Client{Timeout: time.Second}}
		if _, err := a.uploadAbsPathToUpstream(context.Background(), "http://[::1", "example.com:1", absFile, "a.txt", "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("CreateFormFile error via early close", func(t *testing.T) {
		a := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_ = req.Body.Close()
				return nil, errors.New("boom")
			}),
		}}
		if _, err := a.uploadAbsPathToUpstream(context.Background(), "http://example.com/upload", "example.com:1", absFile, "a.txt", "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("openLocalFileForRead error", func(t *testing.T) {
		a := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}}
		if _, err := a.uploadAbsPathToUpstream(context.Background(), "http://example.com/upload", "example.com:1", filepath.Join(tempDir, "missing"), "a.txt", "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("io.Copy error when absPath is directory", func(t *testing.T) {
		a := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}}
		if _, err := a.uploadAbsPathToUpstream(context.Background(), "http://example.com/upload", "example.com:1", tempDir, "a.txt", "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("httpClient.Do error (drain then fail)", func(t *testing.T) {
		a := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, _ = io.ReadAll(req.Body)
				return nil, errors.New("net err")
			}),
		}}
		if _, err := a.uploadAbsPathToUpstream(context.Background(), "http://example.com/upload", "example.com:1", absFile, "a.txt", "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("resp read error", func(t *testing.T) {
		a := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, _ = io.ReadAll(req.Body)
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       errReadCloserUploadAbs{},
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}}
		if _, err := a.uploadAbsPathToUpstream(context.Background(), "http://example.com/upload", "example.com:1", absFile, "a.txt", "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("non-2xx status", func(t *testing.T) {
		a := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, _ = io.ReadAll(req.Body)
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Status:     "502 Bad Gateway",
					Body:       io.NopCloser(strings.NewReader("bad")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}}
		if _, err := a.uploadAbsPathToUpstream(context.Background(), "http://example.com/upload", "example.com:1", absFile, "a.txt", "", "r", "ua"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		a := &App{httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if strings.TrimSpace(req.Header.Get("Cookie")) != "c=1" {
					t.Fatalf("cookie=%q", req.Header.Get("Cookie"))
				}
				if req.Host != "example.com" {
					t.Fatalf("host=%q", req.Host)
				}
				_, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(`{"state":"OK"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		}}

		got, err := a.uploadAbsPathToUpstream(context.Background(), "http://example.com/upload", "example.com:1", absFile, "a.txt", "c=1", "r", "ua")
		if err != nil || strings.TrimSpace(got) == "" {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})
}

