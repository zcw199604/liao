package app

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScaledSize(t *testing.T) {
	cases := []struct {
		width, height, target int
		wantW, wantH          int
	}{
		{width: 40, height: 30, target: 320, wantW: 40, wantH: 30},
		{width: 800, height: 600, target: 320, wantW: 320, wantH: 240},
		{width: 600, height: 800, target: 320, wantW: 240, wantH: 320},
		{width: 0, height: 0, target: 320, wantW: 1, wantH: 1},
		{width: 3, height: 1000, target: 2, wantW: 1, wantH: 2},
	}
	for _, tc := range cases {
		gotW, gotH := scaledSize(tc.width, tc.height, tc.target)
		if gotW != tc.wantW || gotH != tc.wantH {
			t.Fatalf("scaledSize(%d,%d,%d)=(%d,%d), want (%d,%d)", tc.width, tc.height, tc.target, gotW, gotH, tc.wantW, tc.wantH)
		}
	}
}

func TestBuildIntegrationThumb_ResizesJPEG(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.jpg")
	writeTestJPEG(t, source, 800, 400)

	data, err := buildIntegrationThumb(source, 160)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	if img.Bounds().Dx() != 160 || img.Bounds().Dy() != 80 {
		t.Fatalf("thumb=%dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestBuildIntegrationThumb_Unsupported(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "not-image.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := buildIntegrationThumb(source, 160); !errors.Is(err, errIntegrationThumbUnsupported) {
		t.Fatalf("err=%v", err)
	}

	if _, err := buildIntegrationThumb(filepath.Join(dir, "missing.jpg"), 160); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestIntegrationThumbCache_ReuseAndMtime(t *testing.T) {
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "a.jpg")
	writeTestJPEG(t, source, 200, 100)

	cache := newIntegrationThumbCache(t.TempDir())
	first, etag1, _, err := cache.thumbnail(context.Background(), sourceDir, "local:/a.jpg|image", source, 80)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if len(first) == 0 || etag1 == "" {
		t.Fatalf("empty thumb")
	}

	second, etag2, _, err := cache.thumbnail(context.Background(), sourceDir, "local:/a.jpg|image", source, 80)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !bytes.Equal(first, second) || etag1 != etag2 {
		t.Fatalf("cache miss on second call")
	}

	// 改变源文件 mtime 后应生成新缓存键。
	newTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(source, newTime, newTime); err != nil {
		t.Fatal(err)
	}
	_, etag3, _, err := cache.thumbnail(context.Background(), sourceDir, "local:/a.jpg|image", source, 80)
	if err != nil {
		t.Fatalf("third: %v", err)
	}
	if etag3 == etag1 {
		t.Fatalf("etag should change after mtime change")
	}
}

func TestIntegrationThumbCache_Errors(t *testing.T) {
	if _, _, _, err := newIntegrationThumbCache("").thumbnail(context.Background(), t.TempDir(), "x", "y", 0); err == nil {
		t.Fatalf("expected error for empty dir")
	}

	dir := t.TempDir()
	cache := newIntegrationThumbCache(dir)
	empty := filepath.Join(dir, "empty.jpg")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := cache.thumbnail(context.Background(), dir, "x", empty, 160); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err=%v", err)
	}
}

func TestWriteThumbFile_Atomic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "ab", "cd.jpg")
	if err := writeThumbFile(target, []byte("data")); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, ok := readThumbFile(target)
	if !ok || string(got) != "data" {
		t.Fatalf("read=%q ok=%v", got, ok)
	}
	if _, ok := readThumbFile(filepath.Join(dir, "missing.jpg")); ok {
		t.Fatalf("missing file should not be readable")
	}
	if _, ok := readThumbFile(dir); ok {
		t.Fatalf("directory should not be readable as thumb")
	}
}

func TestWriteTestJPEGSanity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.jpg")
	writeTestJPEG(t, path, 10, 10)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil || cfg.Width != 10 || cfg.Height != 10 {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}
