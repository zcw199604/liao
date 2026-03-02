package app

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newLivePhotoResourceServer(t *testing.T) *httptest.Server {
	t.Helper()
	minJPEG := []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
}

func installLivePhotoTools(t *testing.T, exifScript string) {
	t.Helper()
	binDir := t.TempDir()
	mustWriteExecutable(t, binDir, "ffmpeg", `#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
case "$out" in
  *.jpg|*.jpeg) printf '\377\330\377\333\000\004\000\000\377\331' > "$out" ;;
  *.mp4) printf 'MP4DATA' > "$out" ;;
  *.mov) printf 'MOVDATA' > "$out" ;;
  *) printf 'X' > "$out" ;;
esac
exit 0
`)
	if strings.TrimSpace(exifScript) == "" {
		exifScript = "#!/bin/sh\nexit 0\n"
	}
	mustWriteExecutable(t, binDir, "exiftool", exifScript)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func TestSelectDouyinLivePhotoPair_VideoUnpairedBranch(t *testing.T) {
	downloads := []string{
		"https://example.com/i1.jpg",
		"https://example.com/i2.jpg",
		"https://example.com/aweme/v1/play/?video_id=1",
		"https://example.com/aweme/v1/play/?video_id=2",
		"https://example.com/aweme/v1/play/?video_id=3",
	}
	videoIndex := 4
	if _, _, msg := selectDouyinLivePhotoPair(downloads, nil, &videoIndex); msg != "未找到与该视频对应的图片资源" {
		t.Fatalf("msg=%q", msg)
	}
}

func TestHandleDouyinLivePhoto_MkdirTempAndBaseFallbackBranches(t *testing.T) {
	srv := newLivePhotoResourceServer(t)
	defer srv.Close()

	t.Run("mkdir temp failed", func(t *testing.T) {
		installLivePhotoTools(t, "")

		tmpFile := filepath.Join(t.TempDir(), "tmp-as-file")
		if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("write tmp file: %v", err)
		}
		t.Setenv("TMPDIR", tmpFile)

		svc := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
		key := svc.CacheDetail(&douyinCachedDetail{
			DetailID:  "d1",
			Title:     "title",
			Downloads: []string{srv.URL + "/img.jpg", srv.URL + "/vid.mp4"},
		})
		app := &App{douyinDownloader: svc}

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("blank title uses livephoto fallback base", func(t *testing.T) {
		installLivePhotoTools(t, "")
		svc := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
		key := svc.CacheDetail(&douyinCachedDetail{
			DetailID:  "d2",
			Title:     "   ",
			Downloads: []string{srv.URL + "/img.jpg", srv.URL + "/vid.mp4"},
		})
		app := &App{douyinDownloader: svc}

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		cd := rr.Header().Get("Content-Disposition")
		if !strings.Contains(cd, "d2_01_live.jpg") {
			t.Fatalf("content-disposition=%q", cd)
		}
	})
}

func TestHandleDouyinLivePhoto_ZipFileFailureBranches(t *testing.T) {
	srv := newLivePhotoResourceServer(t)
	defer srv.Close()

	t.Run("zip first file failed", func(t *testing.T) {
		installLivePhotoTools(t, `#!/bin/sh
last=""
for a in "$@"; do last="$a"; done
case "$last" in
  *.jpg|*.jpeg) rm -f "$last" ;;
esac
exit 0
`)
		svc := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
		key := svc.CacheDetail(&douyinCachedDetail{
			DetailID:  "d1",
			Title:     "t",
			Downloads: []string{srv.URL + "/img.jpg", srv.URL + "/vid.mp4"},
		})
		app := &App{douyinDownloader: svc}

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=zip", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
		if err != nil {
			t.Fatalf("zip parse err=%v", err)
		}
		if len(zr.File) != 0 {
			t.Fatalf("zip files=%d, want 0", len(zr.File))
		}
	})

	t.Run("zip second file failed", func(t *testing.T) {
		counter := filepath.Join(t.TempDir(), "counter")
		t.Setenv("EXIF_COUNTER_FILE", counter)
		installLivePhotoTools(t, `#!/bin/sh
counter="$EXIF_COUNTER_FILE"
n=0
if [ -f "$counter" ]; then
  n=$(cat "$counter")
fi
n=$((n+1))
echo "$n" > "$counter"
last=""
for a in "$@"; do last="$a"; done
if [ "$n" -eq 2 ]; then
  rm -f "$last"
fi
exit 0
`)
		svc := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
		key := svc.CacheDetail(&douyinCachedDetail{
			DetailID:  "d2",
			Title:     "t2",
			Downloads: []string{srv.URL + "/img.jpg", srv.URL + "/vid.mp4"},
		})
		app := &App{douyinDownloader: svc}

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=zip", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
		if err != nil {
			t.Fatalf("zip parse err=%v", err)
		}
		if len(zr.File) != 1 {
			t.Fatalf("zip files=%d, want 1", len(zr.File))
		}
	})
}

func TestTagLivePhotoAsset_SecondCommandErrorBranch(t *testing.T) {
	binDir := t.TempDir()
	mustWriteExecutable(t, binDir, "exiftool", `#!/bin/sh
last=""
for a in "$@"; do last="$a"; done
case "$last" in
  *.mov) echo "mov failed" 1>&2; exit 1 ;;
esac
exit 0
`)
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	tmp := t.TempDir()
	still := filepath.Join(tmp, "still.jpg")
	motion := filepath.Join(tmp, "motion.mov")
	_ = os.WriteFile(still, []byte("x"), 0o644)
	_ = os.WriteFile(motion, []byte("x"), 0o644)

	if err := tagLivePhotoAsset(context.Background(), still, motion, "ASSET-ID"); err == nil {
		t.Fatalf("expected second exiftool error")
	}
}

func TestCopyFile_AndBuildMotionPhotoJPG_UncoveredErrors(t *testing.T) {
	t.Run("copyFile io.Copy error", func(t *testing.T) {
		if _, err := os.Stat("/dev/full"); err != nil {
			t.Skip("/dev/full not available")
		}
		src := filepath.Join(t.TempDir(), "a.txt")
		if err := os.WriteFile(src, []byte("abc"), 0o644); err != nil {
			t.Fatalf("write src: %v", err)
		}
		if err := copyFile(src, "/dev/full"); err == nil {
			t.Fatalf("expected write error")
		}
	})

	t.Run("buildMotionPhotoJPG error branches", func(t *testing.T) {
		minJPEG := []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
		tmp := t.TempDir()
		still := filepath.Join(tmp, "still.jpg")
		motion := filepath.Join(tmp, "motion.mp4")
		if err := os.WriteFile(still, minJPEG, 0o644); err != nil {
			t.Fatalf("write still: %v", err)
		}
		if err := os.WriteFile(motion, []byte("MP4DATA"), 0o644); err != nil {
			t.Fatalf("write motion: %v", err)
		}

		if err := buildMotionPhotoJPG(filepath.Join(tmp, "missing.jpg"), motion, filepath.Join(tmp, "out.jpg")); err == nil {
			t.Fatalf("expected still read error")
		}

		if _, err := os.Stat("/dev/null"); err == nil {
			if err := buildMotionPhotoJPG(still, motion, "/dev/null/out.jpg"); err == nil {
				t.Fatalf("expected mkdir all error")
			}
		}

		outDir := filepath.Join(tmp, "outdir")
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			t.Fatalf("mkdir outdir: %v", err)
		}
		if err := buildMotionPhotoJPG(still, motion, outDir); err == nil {
			t.Fatalf("expected create output error")
		}

		if _, err := os.Stat("/dev/full"); err == nil {
			if err := buildMotionPhotoJPG(still, motion, "/dev/full"); err == nil {
				t.Fatalf("expected output write error")
			}
		}

		if err := buildMotionPhotoJPG(still, filepath.Join(tmp, "missing-motion.mp4"), filepath.Join(tmp, "out2.jpg")); err == nil {
			t.Fatalf("expected motion open error")
		}

		motionDir := filepath.Join(tmp, "motion-dir")
		if err := os.MkdirAll(motionDir, 0o755); err != nil {
			t.Fatalf("mkdir motion-dir: %v", err)
		}
		if err := buildMotionPhotoJPG(still, motionDir, filepath.Join(tmp, "out3.jpg")); err == nil {
			t.Fatalf("expected io.Copy error from dir source")
		}
	})
}

func TestJPEGParsers_UncoveredBranches(t *testing.T) {
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0x00}); err == nil {
		t.Fatalf("expected parse failure when marker doesn't start with 0xFF")
	}
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0xFF, 0xFF}); err == nil {
		t.Fatalf("expected parse failure with trailing marker prefix")
	}

	jpegWithAPP0ThenDQT := []byte{
		0xFF, 0xD8,
		0xFF, 0xE0, 0x00, 0x04, 0x11, 0x22,
		0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00,
		0xFF, 0xD9,
	}
	off, err := findJPEGFirstDQTOffset(jpegWithAPP0ThenDQT)
	if err != nil || off != 8 {
		t.Fatalf("offset=%d err=%v", off, err)
	}

	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0x00}); err == nil {
		t.Fatalf("expected parse failure when marker doesn't start with 0xFF")
	}
}
