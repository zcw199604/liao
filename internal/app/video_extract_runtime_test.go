package app

import (
	"context"
	"testing"
)

func TestVideoExtractRuntime_SnapshotAppendLogSetProgress(t *testing.T) {
	rt := &videoExtractRuntime{}

	// blank log is ignored
	rt.appendLog("   ")

	// append more than 200 logs to trigger trimming
	for i := 0; i < 205; i++ {
		rt.appendLog("line")
	}

	rt.setProgress(-1, -1, " ")
	rt.setProgress(7, 2_500_000, " 1.2x ")

	view := rt.snapshot(50)
	if view.Frame != 7 {
		t.Fatalf("frame=%d, want 7", view.Frame)
	}
	if view.OutTimeSec < 2.49 || view.OutTimeSec > 2.51 {
		t.Fatalf("outTimeSec=%v, want about 2.5", view.OutTimeSec)
	}
	if view.Speed != "1.2x" {
		t.Fatalf("speed=%q, want %q", view.Speed, "1.2x")
	}
	if len(view.Logs) != 50 {
		t.Fatalf("logs=%d, want 50", len(view.Logs))
	}
}

func TestVideoExtractService_GetRuntimeAndCancelTask(t *testing.T) {
	svc := &VideoExtractService{runtimes: make(map[string]*videoExtractRuntime)}

	if svc.CancelTask("") {
		t.Fatalf("expected false")
	}
	if svc.CancelTask("missing") {
		t.Fatalf("expected false")
	}
	if svc.GetRuntime("missing") != nil {
		t.Fatalf("expected nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelled := false
	rt := &videoExtractRuntime{
		cancel: func() {
			cancelled = true
			cancel()
		},
	}
	svc.runtimes["t1"] = rt

	if svc.GetRuntime("t1") == nil {
		t.Fatalf("expected runtime")
	}
	if !svc.CancelTask("t1") {
		t.Fatalf("expected true")
	}
	if !cancelled || ctx.Err() == nil {
		t.Fatalf("expected canceled")
	}
}
