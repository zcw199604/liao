package app

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustWriteExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()

	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s failed: %v", name, err)
	}
	return p
}

func TestParseOptionalInt(t *testing.T) {
	if v, err := parseOptionalInt(""); err != nil || v != nil {
		t.Fatalf("empty: v=%v err=%v", v, err)
	}
	if v, err := parseOptionalInt("  "); err != nil || v != nil {
		t.Fatalf("spaces: v=%v err=%v", v, err)
	}

	v, err := parseOptionalInt(" 12 ")
	if err != nil || v == nil || *v != 12 {
		t.Fatalf("valid: v=%v err=%v", v, err)
	}

	if _, err := parseOptionalInt("bad"); err == nil {
		t.Fatalf("expected error for invalid int")
	}
}

func TestSelectDouyinLivePhotoPair_MoreBranches(t *testing.T) {
	// empty list
	if _, _, msg := selectDouyinLivePhotoPair(nil, nil, nil); msg == "" {
		t.Fatalf("expected error for empty downloads")
	}

	downloads := []string{
		"https://example.com/a.jpg",
		"https://example.com/b.mp4",
		"https://example.com/c.jpg",
	}

	// out of range
	bad := 3
	if _, _, msg := selectDouyinLivePhotoPair(downloads, &bad, nil); msg == "" {
		t.Fatalf("expected error for imageIndex out of range")
	}
	if _, _, msg := selectDouyinLivePhotoPair(downloads, nil, &bad); msg == "" {
		t.Fatalf("expected error for videoIndex out of range")
	}

	// both provided but mismatched type
	imgIdx := 1 // points to video
	vidIdx := 2 // points to image
	if _, _, msg := selectDouyinLivePhotoPair(downloads, &imgIdx, &vidIdx); msg == "" {
		t.Fatalf("expected error for mismatched types")
	}

	// single-side provided but wrong type should also error
	if _, _, msg := selectDouyinLivePhotoPair(downloads, &imgIdx, nil); msg == "" {
		t.Fatalf("expected error for imageIndex points to video")
	}
	if _, _, msg := selectDouyinLivePhotoPair(downloads, nil, &vidIdx); msg == "" {
		t.Fatalf("expected error for videoIndex points to image")
	}

	// only imageIndex provided -> prefer next video, else fallback search
	imgIdx = 0
	gotImg, gotVid, msg := selectDouyinLivePhotoPair(downloads, &imgIdx, nil)
	if msg != "" || gotImg != 0 || gotVid != 1 {
		t.Fatalf("imageIndex only: img=%d vid=%d msg=%q", gotImg, gotVid, msg)
	}

	imgIdx = 2 // no video after index 2, should fallback to earlier video(1)
	gotImg, gotVid, msg = selectDouyinLivePhotoPair(downloads, &imgIdx, nil)
	if msg != "" || gotImg != 2 || gotVid != 1 {
		t.Fatalf("imageIndex fallback: img=%d vid=%d msg=%q", gotImg, gotVid, msg)
	}

	// only videoIndex provided -> prefer previous image, else fallback search
	vidIdx = 1
	gotImg, gotVid, msg = selectDouyinLivePhotoPair(downloads, nil, &vidIdx)
	if msg != "" || gotImg != 0 || gotVid != 1 {
		t.Fatalf("videoIndex only: img=%d vid=%d msg=%q", gotImg, gotVid, msg)
	}

	downloads2 := []string{
		"https://example.com/v.mp4",
		"https://example.com/i.jpg",
	}
	vidIdx = 0 // no image before index 0, should fallback to later image(1)
	gotImg, gotVid, msg = selectDouyinLivePhotoPair(downloads2, nil, &vidIdx)
	if msg != "" || gotImg != 1 || gotVid != 0 {
		t.Fatalf("videoIndex fallback: img=%d vid=%d msg=%q", gotImg, gotVid, msg)
	}

	// none provided -> first image + first video
	gotImg, gotVid, msg = selectDouyinLivePhotoPair(downloads2, nil, nil)
	if msg != "" || gotImg != 1 || gotVid != 0 {
		t.Fatalf("auto select: img=%d vid=%d msg=%q", gotImg, gotVid, msg)
	}

	// 多视频场景下，尾部非实况图片不应被强行配对
	downloads3 := []string{
		"https://example.com/i1.jpg",
		"https://example.com/i2.jpg",
		"https://example.com/v1.mp4",
		"https://example.com/v2.mp4",
		"https://example.com/i_tail.jpg",
	}
	tailIdx := 4
	if _, _, msg := selectDouyinLivePhotoPair(downloads3, &tailIdx, nil); msg == "" {
		t.Fatalf("expected unpaired error for tail image")
	}

	// missing either image or video
	if _, _, msg := selectDouyinLivePhotoPair([]string{"https://example.com/x.jpg"}, nil, nil); msg == "" {
		t.Fatalf("expected error for image-only downloads")
	}
	if _, _, msg := selectDouyinLivePhotoPair([]string{"https://example.com/x.mp4"}, nil, nil); msg == "" {
		t.Fatalf("expected error for video-only downloads")
	}
}

func TestDownloadDouyinResourceToFile(t *testing.T) {
	t.Parallel()

	const payload = "hello"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(payload))
		case "/bad":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("nope"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	dst := filepath.Join(t.TempDir(), "out.bin")
	ct, err := downloadDouyinResourceToFile(context.Background(), ts.Client(), ts.URL+"/ok", dst)
	if err != nil {
		t.Fatalf("download ok: %v", err)
	}
	if ct != "text/plain" {
		t.Fatalf("contentType=%q, want %q", ct, "text/plain")
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(b) != payload {
		t.Fatalf("dst=%q, want %q", string(b), payload)
	}

	if _, err := downloadDouyinResourceToFile(context.Background(), ts.Client(), "", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatalf("expected error for empty remoteURL")
	}
	if _, err := downloadDouyinResourceToFile(context.Background(), ts.Client(), "not a url", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatalf("expected error for invalid remoteURL")
	}
	if _, err := downloadDouyinResourceToFile(context.Background(), ts.Client(), ts.URL+"/bad", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}

func TestNormalizeLivePhotoStillImage_CopyAndConvert(t *testing.T) {
	// copy branch
	tmp := t.TempDir()
	in := filepath.Join(tmp, "in.jpg")
	if err := os.WriteFile(in, []byte("A"), 0o644); err != nil {
		t.Fatalf("write in: %v", err)
	}
	out := filepath.Join(tmp, "out.jpg")
	if err := normalizeLivePhotoStillImage(context.Background(), in, "image/jpeg", out); err != nil {
		t.Fatalf("normalize copy: %v", err)
	}
	if b, _ := os.ReadFile(out); string(b) != "A" {
		t.Fatalf("out=%q, want %q", string(b), "A")
	}

	// convert branch (ffmpeg) using a fake executable.
	binDir := t.TempDir()
	mustWriteExecutable(t, binDir, "ffmpeg", "#!/bin/sh\nout=\"\"\nfor a in \"$@\"; do out=\"$a\"; done\ncase \"$out\" in\n  *.jpg|*.jpeg) printf '\\377\\330\\377\\333\\000\\004\\000\\000\\377\\331' > \"$out\" ;;\n  *) printf 'X' > \"$out\" ;;\nesac\nexit 0\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	inPng := filepath.Join(tmp, "in.png")
	if err := os.WriteFile(inPng, []byte("PNG"), 0o644); err != nil {
		t.Fatalf("write in png: %v", err)
	}
	out2 := filepath.Join(tmp, "out2.jpg")
	if err := normalizeLivePhotoStillImage(context.Background(), inPng, "image/png", out2); err != nil {
		t.Fatalf("normalize convert: %v", err)
	}
	if st, err := os.Stat(out2); err != nil || st.Size() <= 0 {
		t.Fatalf("out2 not created: st=%v err=%v", st, err)
	}
}

func TestRunCommand_ErrorUsesOutput(t *testing.T) {
	t.Parallel()

	err := runCommand(context.Background(), "sh", []string{"-c", "echo boom 1>&2; exit 2"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("err=%v, want %q", err, "boom")
	}
}

func TestCopyFileAndZipFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "a", "in.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(src, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	dst := filepath.Join(tmp, "b", "out.txt")
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "abc" {
		t.Fatalf("dst=%q, want %q", string(b), "abc")
	}
	if err := copyFile(filepath.Join(tmp, "nope"), filepath.Join(tmp, "x")); err == nil {
		t.Fatalf("expected error for missing src")
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := zipFile(zw, "", "x"); err == nil {
		t.Fatalf("expected error for empty srcPath")
	}
	if err := zipFile(zw, src, "in.txt"); err != nil {
		t.Fatalf("zipFile: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip reader: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "in.txt" {
		t.Fatalf("zip entries=%v", zr.File)
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatalf("zip open: %v", err)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)
	if string(body) != "abc" {
		t.Fatalf("zip body=%q, want %q", string(body), "abc")
	}
}

func TestMotionPhotoSegmentsAndJPEGParsing(t *testing.T) {
	t.Parallel()

	if _, err := buildMotionPhotoXMPAPP1Segment(0); err == nil {
		t.Fatalf("expected error for invalid microVideoOffset")
	}
	xmp, err := buildMotionPhotoXMPAPP1Segment(123)
	if err != nil {
		t.Fatalf("xmp: %v", err)
	}
	if len(xmp) < 4 || xmp[0] != 0xFF || xmp[1] != 0xE1 {
		t.Fatalf("xmp marker invalid: %v", xmp[:4])
	}

	exif, err := buildMotionPhotoExifAPP1Segment(3, 2)
	if err != nil {
		t.Fatalf("exif: %v", err)
	}
	if len(exif) < 4 || exif[0] != 0xFF || exif[1] != 0xE1 {
		t.Fatalf("exif marker invalid: %v", exif[:4])
	}

	seg := []byte{0xAA, 0xBB}
	if _, err := injectJPEGSegmentAfterAPP0([]byte("nope"), seg); err == nil {
		t.Fatalf("expected error for non-jpeg input")
	}

	// SOI only -> insert at 2
	j1 := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	j1o, err := injectJPEGSegmentAfterAPP0(j1, seg)
	if err != nil {
		t.Fatalf("inject SOI: %v", err)
	}
	if !bytes.Equal(j1o[2:4], seg) {
		t.Fatalf("segment not inserted at 2: %v", j1o)
	}

	// With APP0 segment -> insert after APP0 block.
	j2 := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x04, 0x11, 0x22, 0xFF, 0xD9}
	j2o, err := injectJPEGSegmentAfterAPP0(j2, seg)
	if err != nil {
		t.Fatalf("inject APP0: %v", err)
	}
	// insertAt = 2 + 2 + 4 = 8
	if !bytes.Equal(j2o[8:10], seg) {
		t.Fatalf("segment not inserted after APP0: %v", j2o)
	}

	// Invalid APP0 length
	j3 := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0xFF, 0xFF, 0x00}
	if _, err := injectJPEGSegmentAfterAPP0(j3, seg); err == nil {
		t.Fatalf("expected error for invalid APP0 length")
	}

	// findJPEGFirstDQTOffset
	minJPEG := []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	off, err := findJPEGFirstDQTOffset(minJPEG)
	if err != nil || off != 2 {
		t.Fatalf("dqt offset=%d err=%v", off, err)
	}
	// TEM marker should be skipped.
	temJPEG := []byte{0xFF, 0xD8, 0xFF, 0x01, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	off, err = findJPEGFirstDQTOffset(temJPEG)
	if err != nil || off != 4 {
		t.Fatalf("tem dqt offset=%d err=%v", off, err)
	}
	// RSTn marker should be skipped.
	rstJPEG := []byte{0xFF, 0xD8, 0xFF, 0xD0, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	off, err = findJPEGFirstDQTOffset(rstJPEG)
	if err != nil || off != 4 {
		t.Fatalf("rst dqt offset=%d err=%v", off, err)
	}
	// SOS before DQT => error
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0xFF, 0xDA}); err == nil {
		t.Fatalf("expected error for SOS before DQT")
	}
	// insufficient length bytes
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}); err == nil {
		t.Fatalf("expected error for invalid segment length (missing bytes)")
	}
	// invalid length field (<2)
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x01, 0x00}); err == nil {
		t.Fatalf("expected error for invalid segment length (<2)")
	}
	// no DQT at all (only RSTn) => final error path
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0xFF, 0xD0}); err == nil {
		t.Fatalf("expected error for missing DQT")
	}
	if _, err := findJPEGFirstDQTOffset([]byte{0x00}); err == nil {
		t.Fatalf("expected error for non-jpeg")
	}
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0x00}); err == nil {
		t.Fatalf("expected error for invalid segment parse")
	}

	// parseJPEGDimensions success
	sof := []byte{0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x07, 0x08, 0x00, 0x02, 0x00, 0x03}
	w, h, err := parseJPEGDimensions(sof)
	if err != nil || w != 3 || h != 2 {
		t.Fatalf("w=%d h=%d err=%v", w, h, err)
	}
	// non-jpeg
	if _, _, err := parseJPEGDimensions([]byte("nope")); err == nil {
		t.Fatalf("expected error for non-jpeg")
	}
	// SOS/EOI before SOF
	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF, 0xDA}); err == nil {
		t.Fatalf("expected error for SOS without SOF")
	}
	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF, 0xD9}); err == nil {
		t.Fatalf("expected error for EOI without SOF")
	}
	// SOF with invalid n (<7)
	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x06, 0x00}); err == nil {
		t.Fatalf("expected error for short SOF length")
	}
	// invalid size
	sofBad := []byte{0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x07, 0x08, 0x00, 0x00, 0x00, 0x00}
	if _, _, err := parseJPEGDimensions(sofBad); err == nil {
		t.Fatalf("expected error for invalid dimensions")
	}
}

func TestBuildMotionPhotoJPG_SuccessWithMinimalInputs(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	still := filepath.Join(tmp, "still.jpg")
	motion := filepath.Join(tmp, "motion.mp4")
	out := filepath.Join(tmp, "out.jpg")

	// Minimal JPEG: SOI + DQT(length=4) + EOI
	minJPEG := []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	if err := os.WriteFile(still, minJPEG, 0o644); err != nil {
		t.Fatalf("write still: %v", err)
	}
	if err := os.WriteFile(motion, []byte("MP4DATA"), 0o644); err != nil {
		t.Fatalf("write motion: %v", err)
	}

	if err := buildMotionPhotoJPG(still, motion, out); err != nil {
		t.Fatalf("buildMotionPhotoJPG: %v", err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if len(b) <= len(minJPEG) {
		t.Fatalf("out too small: %d", len(b))
	}
	if !bytes.HasPrefix(b, []byte{0xFF, 0xD8}) {
		t.Fatalf("out missing SOI prefix: %v", b[:2])
	}
	if !bytes.Contains(b, []byte("MP4DATA")) {
		t.Fatalf("out should contain motion payload")
	}
}

func TestBuildMotionPhotoJPG_ErrorBranches(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	still := filepath.Join(tmp, "still.jpg")
	motion := filepath.Join(tmp, "motion.mp4")

	// missing motion mp4
	if err := os.WriteFile(still, []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}, 0o644); err != nil {
		t.Fatalf("write still: %v", err)
	}
	if err := buildMotionPhotoJPG(still, filepath.Join(tmp, "nope.mp4"), filepath.Join(tmp, "out.jpg")); err == nil {
		t.Fatalf("expected error for missing motion file")
	}

	// empty motion mp4
	if err := os.WriteFile(motion, []byte{}, 0o644); err != nil {
		t.Fatalf("write motion: %v", err)
	}
	if err := buildMotionPhotoJPG(still, motion, filepath.Join(tmp, "out2.jpg")); err == nil {
		t.Fatalf("expected error for empty motion mp4")
	}

	// still missing EOI
	stillBad := filepath.Join(tmp, "still_bad.jpg")
	if err := os.WriteFile(stillBad, []byte{0xFF, 0xD8, 0xFF, 0xDB, 0x00, 0x04, 0x00, 0x00}, 0o644); err != nil {
		t.Fatalf("write still bad: %v", err)
	}
	if err := os.WriteFile(motion, []byte("X"), 0o644); err != nil {
		t.Fatalf("write motion: %v", err)
	}
	if err := buildMotionPhotoJPG(stillBad, motion, filepath.Join(tmp, "out3.jpg")); err == nil {
		t.Fatalf("expected error for still jpg without EOI")
	}
}

func TestNormalizeMotionVideoBranches(t *testing.T) {
	// Use a fake ffmpeg to make runCommand deterministic.
	binDir := t.TempDir()
	mustWriteExecutable(t, binDir, "ffmpeg", "#!/bin/sh\nout=\"\"\nfor a in \"$@\"; do out=\"$a\"; done\nprintf 'X' > \"$out\"\nexit 0\n")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	tmp := t.TempDir()
	in := filepath.Join(tmp, "in.raw")
	_ = os.WriteFile(in, []byte("x"), 0o644)

	if err := normalizeLivePhotoMotionVideo(context.Background(), in, "video/quicktime", filepath.Join(tmp, "out.mov")); err != nil {
		t.Fatalf("normalizeLivePhotoMotionVideo quicktime: %v", err)
	}
	if err := normalizeMotionPhotoMotionVideo(context.Background(), in, "video/unknown", filepath.Join(tmp, "out.mp4")); err != nil {
		t.Fatalf("normalizeMotionPhotoMotionVideo unknown: %v", err)
	}
}

func TestRunCommand_EmptyOutputUsesErrString(t *testing.T) {
	t.Parallel()

	err := runCommand(context.Background(), "sh", []string{"-c", "exit 2"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("err=%q, want to contain %q", err.Error(), "exit status")
	}
}

func TestZipFile_MissingSourceFile(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := zipFile(zw, filepath.Join(t.TempDir(), "nope.txt"), "nope.txt"); err == nil {
		t.Fatalf("expected error for missing source file")
	}
	_ = zw.Close()
}

func TestHandleDouyinLivePhoto_RequirementsAndSuccess(t *testing.T) {
	// Use fake ffmpeg/exiftool so the handler can execute the happy path without system deps.
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
	if strings.TrimSpace(key) == "" {
		t.Fatalf("cache key should not be empty")
	}

	app := &App{douyinDownloader: svc}

	t.Run("uninitialized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key=x", nil)
		rr := httptest.NewRecorder()
		(*App)(nil).handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		badSvc := &DouyinDownloaderService{api: NewTikTokDownloaderClient("", "", ts.Client()), cache: newLRUCache(10, time.Minute)}
		app2 := &App{douyinDownloader: badSvc}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key=x", nil)
		rr := httptest.NewRecorder()
		app2.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("cache expired", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key=missing", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key=", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("bad imageIndex/videoIndex", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&imageIndex=bad", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad imageIndex: status=%d, want %d", rr.Code, http.StatusBadRequest)
		}

		req2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&videoIndex=bad", nil)
		rr2 := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr2, req2)
		if rr2.Code != http.StatusBadRequest {
			t.Fatalf("bad videoIndex: status=%d, want %d", rr2.Code, http.StatusBadRequest)
		}
	})

	t.Run("select pair error", func(t *testing.T) {
		key2 := svc.CacheDetail(&douyinCachedDetail{
			DetailID:  "d2",
			Title:     "t2",
			Downloads: []string{ts.URL + "/img.jpg"}, // no video
		})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key2+"&format=jpg", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=png", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("remote url empty", func(t *testing.T) {
		key3 := svc.CacheDetail(&douyinCachedDetail{
			DetailID:  "d3",
			Title:     "t3",
			Downloads: []string{"", ts.URL + "/vid.mp4"},
		})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key3+"&format=zip", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("zip success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=zip", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d, body=%q", rr.Code, http.StatusOK, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/zip" {
			t.Fatalf("Content-Type=%q, want %q", ct, "application/zip")
		}

		zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
		if err != nil {
			t.Fatalf("zip reader: %v", err)
		}
		if len(zr.File) != 2 {
			t.Fatalf("zip entries=%d, want 2", len(zr.File))
		}
		names := []string{zr.File[0].Name, zr.File[1].Name}
		if !strings.Contains(strings.Join(names, ","), "t_01.jpg") || !strings.Contains(strings.Join(names, ","), "t_01.mov") {
			t.Fatalf("zip names=%v", names)
		}
	})

	t.Run("jpg success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinLivePhoto(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d, body=%q", rr.Code, http.StatusOK, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); ct != "image/jpeg" {
			t.Fatalf("Content-Type=%q, want %q", ct, "image/jpeg")
		}
		body := rr.Body.Bytes()
		if len(body) < 2 || body[0] != 0xFF || body[1] != 0xD8 {
			t.Fatalf("jpg response missing SOI prefix")
		}
		if !bytes.Contains(body, []byte("MP4DATA")) {
			t.Fatalf("jpg response should contain motion payload")
		}
	})
}

func TestHandleDouyinLivePhoto_MissingFfmpegOrExiftool(t *testing.T) {
	// Intentionally do NOT provide ffmpeg/exiftool in PATH.
	t.Setenv("PATH", t.TempDir())

	svc := NewDouyinDownloaderService("http://upstream.example", "", "", "", time.Second)
	key := svc.CacheDetail(&douyinCachedDetail{
		DetailID:  "d1",
		Title:     "t",
		Downloads: []string{"https://example.com/a.jpg", "https://example.com/b.mp4"},
	})
	app := &App{douyinDownloader: svc}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=jpg", nil)
	rr := httptest.NewRecorder()
	app.handleDouyinLivePhoto(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}

	// Provide ffmpeg but not exiftool: zip should fail at exiftool check.
	binDir := t.TempDir()
	mustWriteExecutable(t, binDir, "ffmpeg", "#!/bin/sh\nexit 0\n")
	// Keep PATH isolated so an installed system exiftool won't make this test flaky.
	t.Setenv("PATH", binDir)

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/livephoto?key="+key+"&format=zip", nil)
	rr2 := httptest.NewRecorder()
	app.handleDouyinLivePhoto(rr2, req2)
	if rr2.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr2.Code, http.StatusInternalServerError)
	}
}
