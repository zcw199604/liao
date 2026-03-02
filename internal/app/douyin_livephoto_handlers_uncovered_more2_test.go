package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMotionPhotoJPG_MissingDQTBranch(t *testing.T) {
	tmp := t.TempDir()
	still := filepath.Join(tmp, "still.jpg")
	motion := filepath.Join(tmp, "motion.mp4")
	out := filepath.Join(tmp, "out.jpg")

	// SOI + APP0 + EOI (no DQT)
	noDQTJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x04, 0x00, 0x00, 0xFF, 0xD9}
	if err := os.WriteFile(still, noDQTJPEG, 0o644); err != nil {
		t.Fatalf("write still err=%v", err)
	}
	if err := os.WriteFile(motion, []byte("mp4"), 0o644); err != nil {
		t.Fatalf("write motion err=%v", err)
	}

	if err := buildMotionPhotoJPG(still, motion, out); err == nil {
		t.Fatalf("expected missing-DQT error")
	}
}

func TestJPEGParsers_AdditionalUncoveredBranches(t *testing.T) {
	if _, err := findJPEGFirstDQTOffset([]byte{0xFF, 0xD8, 0x00, 0x00}); err == nil {
		t.Fatalf("expected segment-head parse error")
	}

	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0x00, 0x00}); err == nil {
		t.Fatalf("expected non-0xFF segment parse error")
	}

	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF, 0xFF}); err == nil {
		t.Fatalf("expected marker-truncation parse error")
	}

	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF, 0xD0, 0xFF, 0xD9}); err == nil {
		t.Fatalf("expected missing-SOF after RST marker")
	}

	// SOF length=6 (valid segment bytes length, but too short for SOF payload)
	if _, _, err := parseJPEGDimensions([]byte{0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x06, 0x08, 0x00, 0x01, 0x00}); err == nil {
		t.Fatalf("expected short-SOF error")
	}
}
