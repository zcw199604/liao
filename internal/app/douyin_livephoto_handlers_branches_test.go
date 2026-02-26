package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type errReadCloserLivePhoto struct{}

func (errReadCloserLivePhoto) Read(_ []byte) (int, error) { return 0, errors.New("read err") }
func (errReadCloserLivePhoto) Close() error               { return nil }

func TestFindDouyinMediaRank_NotFound(t *testing.T) {
	if got := findDouyinMediaRank([]int{1, 3, 5}, 9); got != -1 {
		t.Fatalf("rank=%d, want -1", got)
	}
}

func TestDownloadDouyinResourceToFile_MoreErrorBranches(t *testing.T) {
	t.Run("client do error", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("net error")
			}),
		}
		if _, err := downloadDouyinResourceToFile(context.Background(), client, "https://example.com/a.jpg", filepath.Join(t.TempDir(), "a.bin")); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("create dst file error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("abc"))
		}))
		defer srv.Close()

		dst := filepath.Join(t.TempDir(), "missing", "a.bin")
		if _, err := downloadDouyinResourceToFile(context.Background(), srv.Client(), srv.URL+"/x", dst); err == nil {
			t.Fatalf("expected create file error")
		}
	})

	t.Run("copy body error", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
					Body:       errReadCloserLivePhoto{},
				}, nil
			}),
		}

		if _, err := downloadDouyinResourceToFile(context.Background(), client, "https://example.com/a.jpg", filepath.Join(t.TempDir(), "a.bin")); err == nil {
			t.Fatalf("expected copy error")
		}
	})
}

func TestTagLivePhotoAsset_EmptyID(t *testing.T) {
	if err := tagLivePhotoAsset(context.Background(), "/tmp/a.jpg", "/tmp/b.mov", " "); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCopyFile_MkdirAndCreateErrors(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// MkdirAll error: parent path is an existing file.
	parentFile := filepath.Join(tmp, "parent-file")
	if err := os.WriteFile(parentFile, []byte("f"), 0o644); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	if err := copyFile(src, filepath.Join(parentFile, "out.txt")); err == nil {
		t.Fatalf("expected mkdir error")
	}

	// os.Create error: destination itself is an existing directory.
	dstDir := filepath.Join(tmp, "dst-dir")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir dstDir: %v", err)
	}
	if err := copyFile(src, dstDir); err == nil {
		t.Fatalf("expected create error")
	}
}

func TestJPEGParsing_AdditionalErrorBranches(t *testing.T) {
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0xFF, 0xD9}); err == nil {
		t.Fatalf("expected EOI-before-DQT error")
	}

	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0x00}); err == nil {
		t.Fatalf("expected segment parse error")
	}
	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF}); err == nil {
		t.Fatalf("expected truncated marker error")
	}
	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}); err == nil {
		t.Fatalf("expected truncated length error")
	}
}

func TestHandleDouyinLivePhoto_DefaultFormatAndJPEGAlias(t *testing.T) {
	binDir := t.TempDir()
	mustWriteExecutable(t, binDir, "ffmpeg", "#!/bin/sh\nout=\"\"\nfor a in \"$@\"; do out=\"$a\"; done\ncase \"$out\" in\n  *.jpg|*.jpeg) printf '\\377\\330\\377\\333\\000\\004\\000\\000\\377\\331' > \"$out\" ;;\n  *.mp4) printf 'MP4DATA' > \"$out\" ;;\n  *.mov) printf 'MOVDATA' > \"$out\" ;;\n  *) printf 'X' > \"$out\" ;;\nesac\nexit 0\n")
	mustWriteExecutable(t, binDir, "exiftool", "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	minJPEG := []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/img.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(minJPEG)
		case "/vid.mp4":
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = w.Write([]byte("VIDEO"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	svc := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
	key := svc.CacheDetail(&douyinCachedDetail{
		DetailID:  "d1",
		Title:     "t",
		Downloads: []string{ts.URL + "/img.jpg", ts.URL + "/vid.mp4"},
	})
	app := &App{douyinDownloader: svc}

	// format omitted -> default zip
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key, nil)
	rr := httptest.NewRecorder()
	app.handleDouyinLivePhoto(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("default format status=%d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("default format content-type=%q", ct)
	}

	// format=jpeg -> alias to jpg
	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpeg", nil)
	rr2 := httptest.NewRecorder()
	app.handleDouyinLivePhoto(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("jpeg alias status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	if ct := rr2.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("jpeg alias content-type=%q", ct)
	}

	var parsed map[string]any
	if strings.HasPrefix(rr2.Body.String(), "{") {
		_ = json.Unmarshal(rr2.Body.Bytes(), &parsed)
	}
}

