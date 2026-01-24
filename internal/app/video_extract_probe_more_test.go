package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"liao/internal/config"
)

func TestVideoExtractService_ProbeVideo_Branches(t *testing.T) {
	svc := &VideoExtractService{cfg: config.Config{FFprobePath: "ffprobe"}}

	if _, err := svc.ProbeVideo(context.Background(), " "); err == nil {
		t.Fatalf("expected error")
	}

	if _, err := svc.ProbeVideo(context.Background(), filepath.Join(t.TempDir(), "missing.mp4")); err == nil {
		t.Fatalf("expected error")
	}

	if _, err := svc.ProbeVideo(context.Background(), t.TempDir()); err == nil {
		t.Fatalf("expected error")
	}

	inputAbs := filepath.Join(t.TempDir(), "in.mp4")
	if err := os.WriteFile(inputAbs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	svc.cfg.FFprobePath = filepath.Join(t.TempDir(), "missing-ffprobe")
	if _, err := svc.ProbeVideo(context.Background(), inputAbs); err == nil {
		t.Fatalf("expected error")
	}

	ffprobeBad := filepath.Join(t.TempDir(), "ffprobe-bad")
	if err := os.WriteFile(ffprobeBad, []byte("#!/bin/sh\necho not-json\n"), 0o755); err != nil {
		t.Fatalf("write ffprobe: %v", err)
	}
	svc.cfg.FFprobePath = ffprobeBad
	if _, err := svc.ProbeVideo(context.Background(), inputAbs); err == nil {
		t.Fatalf("expected error")
	}

	ffprobeEmptyStreams := filepath.Join(t.TempDir(), "ffprobe-empty")
	if err := os.WriteFile(ffprobeEmptyStreams, []byte("#!/bin/sh\necho '{\"streams\":[],\"format\":{\"duration\":\"1.0\"}}'\n"), 0o755); err != nil {
		t.Fatalf("write ffprobe: %v", err)
	}
	svc.cfg.FFprobePath = ffprobeEmptyStreams

	probe, err := svc.ProbeVideo(context.Background(), inputAbs)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if probe.Width != 0 || probe.Height != 0 || probe.DurationSec != 1.0 {
		t.Fatalf("probe=%+v", probe)
	}
}
