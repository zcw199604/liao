package app

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDetectAvailablePortWithPorts_EmptyHost(t *testing.T) {
	got := detectAvailablePortWithPorts("", []string{"9006"}, func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("should not be called")
	})
	if got != "9006" {
		t.Fatalf("got=%q, want %q", got, "9006")
	}
}

func TestDetectAvailablePortDefault_EmptyHost(t *testing.T) {
	if got := detectAvailablePortDefault(""); got != "9006" {
		t.Fatalf("got=%q, want %q", got, "9006")
	}
}

func TestDetectAvailablePortWithPorts_FindsOpenPort(t *testing.T) {
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		if strings.HasSuffix(address, ":8002") {
			c1, c2 := net.Pipe()
			_ = c2.Close()
			return c1, nil
		}
		return nil, errors.New("dial fail")
	}

	got := detectAvailablePortWithPorts("127.0.0.1:1234", []string{"8001", "8002", "8003"}, dial)
	if got != "8002" {
		t.Fatalf("got=%q, want %q", got, "8002")
	}
}

func TestDetectAvailablePortWithPorts_ContextTimeout(t *testing.T) {
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	got := detectAvailablePortWithPorts("127.0.0.1", []string{"8001"}, dial)
	if got != "9006" {
		t.Fatalf("got=%q, want %q", got, "9006")
	}
}

func TestDetectAvailablePortWithPorts_CtxDoneBranch(t *testing.T) {
	ports := []string{"8001", "8002"}
	block := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(len(ports))

	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		defer wg.Done()
		<-block
		return nil, errors.New("blocked")
	}

	got := detectAvailablePortWithPorts("127.0.0.1", ports, dial)
	if got != "9006" {
		t.Fatalf("got=%q, want %q", got, "9006")
	}

	// release dial goroutines after the function returns via ctx.Done().
	close(block)

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("dial goroutines did not exit")
	}
}
