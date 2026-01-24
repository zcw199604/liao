package app

import (
	"testing"

	"liao/internal/config"
)

func TestNewVideoExtractService_Shutdown_Branches(t *testing.T) {
	svc := NewVideoExtractService(nil, config.Config{VideoExtractWorkers: 0, VideoExtractQueueSize: 1}, nil, nil)

	cancelled := 0
	svc.mu.Lock()
	svc.runtimes["t1"] = &videoExtractRuntime{cancel: func() { cancelled++ }}
	svc.mu.Unlock()

	svc.Shutdown()
	if cancelled != 1 {
		t.Fatalf("cancelled=%d, want 1", cancelled)
	}

	// Second call should be a noop.
	svc.Shutdown()
	if cancelled != 1 {
		t.Fatalf("cancelled=%d, want 1", cancelled)
	}
}

func TestVideoExtractService_Enqueue_Branches(t *testing.T) {
	svc := &VideoExtractService{
		queue:   make(chan string, 1),
		closing: make(chan struct{}),
	}

	if err := svc.Enqueue(" "); err == nil {
		t.Fatalf("expected error")
	}

	if err := svc.Enqueue("t1"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// buffer full
	if err := svc.Enqueue("t2"); err == nil {
		t.Fatalf("expected error")
	}

	close(svc.closing)
	if err := svc.Enqueue("t3"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewVideoExtractService_RespectsWorkers(t *testing.T) {
	svc := NewVideoExtractService(nil, config.Config{VideoExtractWorkers: 2, VideoExtractQueueSize: 1}, nil, nil)
	svc.Shutdown()
}
