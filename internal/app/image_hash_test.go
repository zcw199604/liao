package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestImageHashService_CalculatePHash_Deterministic(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 3), G: uint8(y * 3), B: uint8((x + y) * 2), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png failed: %v", err)
	}
	content := buf.Bytes()

	_, fh1 := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", content, nil)
	_, fh2 := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", content, nil)

	svc := &ImageHashService{}
	ph1, err := svc.CalculatePHash(fh1)
	if err != nil {
		t.Fatalf("CalculatePHash failed: %v", err)
	}
	ph2, err := svc.CalculatePHash(fh2)
	if err != nil {
		t.Fatalf("CalculatePHash failed: %v", err)
	}
	if ph1 != ph2 {
		t.Fatalf("pHash not deterministic: ph1=%d ph2=%d", ph1, ph2)
	}
}

func TestImageHashService_CalculatePHash_Unsupported(t *testing.T) {
	_, fh := newMultipartRequest(t, "POST", "http://example.com/api/checkDuplicateMedia", "file", "a.txt", "text/plain", []byte("not-an-image"), nil)

	svc := &ImageHashService{}
	if _, err := svc.CalculatePHash(fh); err == nil {
		t.Fatalf("expected error, got nil")
	} else if err != ErrPHashUnsupported {
		t.Fatalf("expected ErrPHashUnsupported, got %v", err)
	}
}

func TestResolvePHashThreshold(t *testing.T) {
	thresholdType, similarity, distance, err := resolvePHashThreshold("90", "", "")
	if err != nil {
		t.Fatalf("resolvePHashThreshold failed: %v", err)
	}
	if thresholdType != "similarity" {
		t.Fatalf("thresholdType=%q, want %q", thresholdType, "similarity")
	}
	if similarity < 0.89 || similarity > 0.91 {
		t.Fatalf("similarity=%v, want about 0.9", similarity)
	}
	if distance <= 0 || distance >= phashBitLength {
		t.Fatalf("distance=%d out of range", distance)
	}

	thresholdType, similarity, distance, err = resolvePHashThreshold("", "10", "")
	if err != nil {
		t.Fatalf("resolvePHashThreshold failed: %v", err)
	}
	if thresholdType != "distance" || distance != 10 {
		t.Fatalf("thresholdType=%q distance=%d, want distance=10", thresholdType, distance)
	}
	if similarity <= 0 || similarity >= 1 {
		t.Fatalf("similarity=%v out of range", similarity)
	}
}

func TestSimilarityThresholdToDistance(t *testing.T) {
	if got := similarityThresholdToDistance(1); got != 0 {
		t.Fatalf("similarityThresholdToDistance(1)=%d, want 0", got)
	}
	if got := similarityThresholdToDistance(0); got != phashBitLength {
		t.Fatalf("similarityThresholdToDistance(0)=%d, want %d", got, phashBitLength)
	}
}
