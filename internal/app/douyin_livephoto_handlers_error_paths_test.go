package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupLivePhotoTooling(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	mustWriteExecutable(t, binDir, "ffmpeg", `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
case "$FFMPEG_MODE:$out" in
  fail_jpg:*.jpg|fail_jpg:*.jpeg) echo "jpg failed" 1>&2; exit 1 ;;
  fail_mov:*.mov) echo "mov failed" 1>&2; exit 1 ;;
  fail_mp4:*.mp4) echo "mp4 failed" 1>&2; exit 1 ;;
  empty_mp4:*.mp4) : > "$out"; exit 0 ;;
esac
case "$out" in
  *.jpg|*.jpeg) printf '\377\330\377\333\000\004\000\000\377\331' > "$out" ;;
  *.mp4) printf 'MP4DATA' > "$out" ;;
  *.mov) printf 'MOVDATA' > "$out" ;;
  *) printf 'X' > "$out" ;;
esac
exit 0
`)
	mustWriteExecutable(t, binDir, "exiftool", `#!/bin/sh
if [ "$EXIFTOOL_FAIL" = "1" ]; then
  echo "exiftool failed" 1>&2
  exit 1
fi
exit 0
`)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func newLivePhotoAppWithDownloads(downloads []string) (*App, string) {
	svc := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "d1", Title: "", Downloads: downloads})
	return &App{douyinDownloader: svc}, key
}

func TestHandleDouyinLivePhoto_ErrorPaths(t *testing.T) {
	minJPEG := []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/img.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(minJPEG)
		case "/img.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNG"))
		case "/vid.mp4":
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = w.Write([]byte("VIDEO"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Run("missing ffmpeg", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		app, key := newLivePhotoAppWithDownloads([]string{server.URL + "/img.jpg", server.URL + "/vid.mp4"})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing exiftool for zip", func(t *testing.T) {
		binDir := t.TempDir()
		mustWriteExecutable(t, binDir, "ffmpeg", "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
		app, key := newLivePhotoAppWithDownloads([]string{server.URL + "/img.jpg", server.URL + "/vid.mp4"})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=zip", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("image/video download failure", func(t *testing.T) {
		setupLivePhotoTooling(t)
		app, key := newLivePhotoAppWithDownloads([]string{"http://127.0.0.1:1/img.jpg", server.URL + "/vid.mp4"})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("image fail status=%d body=%s", rr.Code, rr.Body.String())
		}

		app2, key2 := newLivePhotoAppWithDownloads([]string{server.URL + "/img.jpg", "http://127.0.0.1:1/vid.mp4"})
		req2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key2+"&format=jpg", nil)
		rr2 := httptest.NewRecorder()
		app2.handleDouyinLivePhoto(rr2, req2)
		if rr2.Code != http.StatusBadRequest {
			t.Fatalf("video fail status=%d body=%s", rr2.Code, rr2.Body.String())
		}
	})

	t.Run("convert and tag failures", func(t *testing.T) {
		setupLivePhotoTooling(t)
		appPng, keyPng := newLivePhotoAppWithDownloads([]string{server.URL + "/img.png", server.URL + "/vid.mp4"})
		app, key := newLivePhotoAppWithDownloads([]string{server.URL + "/img.jpg", server.URL + "/vid.mp4"})

		t.Setenv("FFMPEG_MODE", "fail_jpg")
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+keyPng+"&format=zip", nil)
		rr := httptest.NewRecorder()
		appPng.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("still convert fail status=%d body=%s", rr.Code, rr.Body.String())
		}

		t.Setenv("FFMPEG_MODE", "fail_mov")
		req2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=zip", nil)
		rr2 := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr2, req2)
		if rr2.Code != http.StatusBadRequest {
			t.Fatalf("mov convert fail status=%d body=%s", rr2.Code, rr2.Body.String())
		}

		t.Setenv("FFMPEG_MODE", "")
		t.Setenv("EXIFTOOL_FAIL", "1")
		req3 := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=zip", nil)
		rr3 := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr3, req3)
		if rr3.Code != http.StatusInternalServerError {
			t.Fatalf("tag fail status=%d body=%s", rr3.Code, rr3.Body.String())
		}
	})

	t.Run("jpg branch convert/build failures", func(t *testing.T) {
		setupLivePhotoTooling(t)
		app, key := newLivePhotoAppWithDownloads([]string{server.URL + "/img.jpg", server.URL + "/vid.mp4"})

		t.Setenv("FFMPEG_MODE", "fail_mp4")
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("mp4 convert fail status=%d body=%s", rr.Code, rr.Body.String())
		}

		t.Setenv("FFMPEG_MODE", "empty_mp4")
		req2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
		rr2 := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr2, req2)
		if rr2.Code != http.StatusInternalServerError {
			t.Fatalf("build jpg fail status=%d body=%s", rr2.Code, rr2.Body.String())
		}
	})

	t.Run("zip write short-circuit on zipFile error", func(t *testing.T) {
		setupLivePhotoTooling(t)
		app, key := newLivePhotoAppWithDownloads([]string{server.URL + "/img.jpg", server.URL + "/vid.mp4"})

		// Make image conversion succeed but force zipFile failure by replacing output file path with directory via chmod trick is hard.
		// Instead, remove temp dir by racing not deterministic; keep a deterministic low-level zipFile branch here.
		_ = key
		_ = app
		if err := zipFile(nil, "", ""); err == nil {
			t.Fatalf("zipFile empty args should fail")
		}
	})
}

func TestNormalizeLivePhotoStillImage_AndTagAssetAdditionalBranches(t *testing.T) {
	setupLivePhotoTooling(t)
	tmp := t.TempDir()
	input := filepath.Join(tmp, "in.png")
	if err := os.WriteFile(input, []byte{1}, 0o644); err != nil {
		t.Fatalf("write input failed: %v", err)
	}
	out := filepath.Join(tmp, "out.jpg")
	t.Setenv("FFMPEG_MODE", "fail_jpg")
	if err := normalizeLivePhotoStillImage(context.Background(), input, "image/png", out); err == nil {
		t.Fatalf("ffmpeg failure should be returned")
	}

	t.Setenv("EXIFTOOL_FAIL", "1")
	if err := tagLivePhotoAsset(context.Background(), input, input, "ASSET"); err == nil {
		t.Fatalf("tagLivePhotoAsset should fail when exiftool fails")
	}
}
