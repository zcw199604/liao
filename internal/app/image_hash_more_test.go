package app

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"mime/multipart"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestImageHashService_FindByMD5Hash_Empty(t *testing.T) {
	svc := &ImageHashService{}
	got, err := svc.FindByMD5Hash(context.Background(), " ", 10)
	if err != nil {
		t.Fatalf("FindByMD5Hash: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got=%v", got)
	}
}

func TestImageHashService_FindByMD5Hash_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs("abc", 1).
		WillReturnError(errors.New("query fail"))

	svc := NewImageHashService(wrapMySQLDB(db))
	if _, err := svc.FindByMD5Hash(context.Background(), " abc ", 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestImageHashService_FindByMD5Hash_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs("abc", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	svc := NewImageHashService(wrapMySQLDB(db))
	if _, err := svc.FindByMD5Hash(context.Background(), "abc", 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestImageHashService_FindByMD5Hash_Success_NullsAndClamp(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs("abc", 500).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at",
		}).AddRow(
			int64(1),
			"/images/x.png",
			"x.png",
			nil,
			"abc",
			int64(123),
			nil,
			now,
		))

	svc := NewImageHashService(wrapMySQLDB(db))
	matches, err := svc.FindByMD5Hash(context.Background(), " abc ", 999)
	if err != nil {
		t.Fatalf("FindByMD5Hash: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("len=%d, want 1", len(matches))
	}
	if matches[0].FileDir != "" || matches[0].FileSize != 0 {
		t.Fatalf("unexpected null mapping: %+v", matches[0])
	}
	if matches[0].Distance != 0 || matches[0].Similarity != 1 {
		t.Fatalf("unexpected match fields: %+v", matches[0])
	}
	if matches[0].CreatedAt == "" {
		t.Fatalf("expected CreatedAt")
	}
}

func TestImageHashService_FindSimilarByPHash_QueryError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?is)(BIT_COUNT|length\(replace\().*FROM image_hash.*<= \?.*LIMIT \?`).
		WithArgs(int64(123), int64(123), phashBitLength, 1).
		WillReturnError(errors.New("query fail"))

	svc := NewImageHashService(wrapMySQLDB(db))
	if _, err := svc.FindSimilarByPHash(context.Background(), 123, 999, 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestImageHashService_FindSimilarByPHash_ScanError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?is)(BIT_COUNT|length\(replace\().*FROM image_hash.*<= \?.*LIMIT \?`).
		WithArgs(int64(123), int64(123), 10, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	svc := NewImageHashService(wrapMySQLDB(db))
	if _, err := svc.FindSimilarByPHash(context.Background(), 123, 10, 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestImageHashService_FindSimilarByPHash_Success_NullsAndClamp(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`(?is)(BIT_COUNT|length\(replace\().*FROM image_hash.*<= \?.*LIMIT \?`).
		WithArgs(int64(123), int64(123), 0, 500).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at", "distance",
		}).AddRow(
			int64(2),
			"/images/y.png",
			"y.png",
			nil,
			"md5",
			int64(456),
			nil,
			now,
			0,
		))

	svc := NewImageHashService(wrapMySQLDB(db))
	matches, err := svc.FindSimilarByPHash(context.Background(), 123, -1, 999)
	if err != nil {
		t.Fatalf("FindSimilarByPHash: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("len=%d, want 1", len(matches))
	}
	if matches[0].FileDir != "" || matches[0].FileSize != 0 {
		t.Fatalf("unexpected null mapping: %+v", matches[0])
	}
	if matches[0].Distance != 0 || matches[0].Similarity != 1 {
		t.Fatalf("unexpected match fields: %+v", matches[0])
	}
}

func TestImageHashService_CalculatePHash_NilFile(t *testing.T) {
	svc := &ImageHashService{}
	if _, err := svc.CalculatePHash(nil); err != ErrPHashUnsupported {
		t.Fatalf("expected ErrPHashUnsupported, got %v", err)
	}
}

func TestImageHashService_CalculatePHash_OpenError(t *testing.T) {
	oldOpen := openMultipartFileHeaderFn
	openMultipartFileHeaderFn = func(*multipart.FileHeader) (multipart.File, error) {
		return nil, errors.New("open fail")
	}
	t.Cleanup(func() { openMultipartFileHeaderFn = oldOpen })

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{R: 10, G: 10, B: 10, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	_, fh := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", buf.Bytes(), nil)

	svc := &ImageHashService{}
	if _, err := svc.CalculatePHash(fh); err != ErrPHashUnsupported {
		t.Fatalf("expected ErrPHashUnsupported, got %v", err)
	}
}

func TestImageHashService_CalculatePHash_ResizeError(t *testing.T) {
	oldResize := resizeToGrayFn
	resizeToGrayFn = func(image.Image, int, int) (*image.Gray, error) {
		return nil, errors.New("resize fail")
	}
	t.Cleanup(func() { resizeToGrayFn = oldResize })

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{R: 10, G: 10, B: 10, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	_, fh := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", buf.Bytes(), nil)

	svc := &ImageHashService{}
	if _, err := svc.CalculatePHash(fh); err != ErrPHashUnsupported {
		t.Fatalf("expected ErrPHashUnsupported, got %v", err)
	}
}

func TestImageHashService_CalculatePHash_DCTLengthMismatch(t *testing.T) {
	oldDCT := dctLowFreq8x8Fn
	dctLowFreq8x8Fn = func(*image.Gray) []float64 { return []float64{1} }
	t.Cleanup(func() { dctLowFreq8x8Fn = oldDCT })

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{R: 10, G: 10, B: 10, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	_, fh := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", buf.Bytes(), nil)

	svc := &ImageHashService{}
	if _, err := svc.CalculatePHash(fh); err == nil {
		t.Fatalf("expected error")
	}
}

func TestResizeToGray_Errors(t *testing.T) {
	if _, err := resizeToGray(nil, 1, 1); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := resizeToGray(image.NewGray(image.Rect(0, 0, 1, 1)), 0, 1); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := resizeToGray(image.NewGray(image.Rect(0, 0, 0, 0)), 1, 1); err == nil {
		t.Fatalf("expected error")
	}
}

func TestResizeToGray_SumWZero_UsesNearestNeighbor(t *testing.T) {
	oldKernel := lanczosKernelFunc
	lanczosKernelFunc = func(float64, float64) float64 { return 0 }
	t.Cleanup(func() { lanczosKernelFunc = oldKernel })

	src := image.NewGray(image.Rect(0, 0, 8, 8))
	src.SetGray(3, 3, color.Gray{Y: 123})

	dst, err := resizeToGray(src, 8, 8)
	if err != nil {
		t.Fatalf("resizeToGray: %v", err)
	}
	if got := dst.GrayAt(3, 3).Y; got != 123 {
		t.Fatalf("got=%d, want 123", got)
	}
}

func TestResizeToGray_ClampBelowZero(t *testing.T) {
	oldKernel := lanczosKernelFunc
	lanczosKernelFunc = func(x, _ float64) float64 {
		if math.Abs(x) < 1e-9 {
			return 1
		}
		return -0.01
	}
	t.Cleanup(func() { lanczosKernelFunc = oldKernel })

	src := image.NewGray(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	src.SetGray(3, 3, color.Gray{Y: 0})

	dst, err := resizeToGray(src, 8, 8)
	if err != nil {
		t.Fatalf("resizeToGray: %v", err)
	}
	if got := dst.GrayAt(3, 3).Y; got != 0 {
		t.Fatalf("got=%d, want 0", got)
	}
}

func TestResizeToGray_ClampAbove255(t *testing.T) {
	oldKernel := lanczosKernelFunc
	lanczosKernelFunc = func(x, _ float64) float64 {
		if math.Abs(x) < 1e-9 {
			return 1
		}
		return -0.01
	}
	t.Cleanup(func() { lanczosKernelFunc = oldKernel })

	src := image.NewGray(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.SetGray(x, y, color.Gray{Y: 0})
		}
	}
	src.SetGray(3, 3, color.Gray{Y: 255})

	dst, err := resizeToGray(src, 8, 8)
	if err != nil {
		t.Fatalf("resizeToGray: %v", err)
	}
	if got := dst.GrayAt(3, 3).Y; got != 255 {
		t.Fatalf("got=%d, want 255", got)
	}
}

func TestSincBranches(t *testing.T) {
	if got := sinc(0); got != 1 {
		t.Fatalf("sinc(0)=%v, want 1", got)
	}
	if got := sinc(0.5); got == 1 {
		t.Fatalf("sinc(0.5) should not be 1")
	}
}

func TestDCTLowFreq8x8_EarlyReturnNil(t *testing.T) {
	if got := dctLowFreq8x8(nil); got != nil {
		t.Fatalf("expected nil")
	}
	if got := dctLowFreq8x8(image.NewGray(image.Rect(0, 0, 1, 1))); got != nil {
		t.Fatalf("expected nil")
	}
}

func TestMedianFloat64_Branches(t *testing.T) {
	if got := medianFloat64(nil); got != 0 {
		t.Fatalf("median(nil)=%v, want 0", got)
	}
	if got := medianFloat64([]float64{3, 1, 2}); got != 2 {
		t.Fatalf("median odd=%v, want 2", got)
	}
	if got := medianFloat64([]float64{4, 2, 3, 1}); got != 2.5 {
		t.Fatalf("median even=%v, want 2.5", got)
	}
}
