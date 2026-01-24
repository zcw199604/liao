package app

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

var detectAvailablePort = detectAvailablePortDefault

func detectAvailablePortDefault(imgServerHost string) string {
	ports := []string{"9006", "9005", "9003", "9002", "9001", "8006", "8005", "8003", "8002", "8001"}
	dialer := &net.Dialer{}
	return detectAvailablePortWithPorts(imgServerHost, ports, dialer.DialContext)
}

func detectAvailablePortWithPorts(
	imgServerHost string,
	ports []string,
	dial func(ctx context.Context, network, address string) (net.Conn, error),
) string {
	host := strings.TrimSpace(imgServerHost)
	if host == "" {
		return "9006"
	}

	// 兼容传入 host:port 的情况（避免误拼接成 host:port:port）。
	if h, _, err := net.SplitHostPort(host); err == nil && strings.TrimSpace(h) != "" {
		host = strings.TrimSpace(h)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	type result struct {
		port string
		ok   bool
	}
	results := make(chan result, len(ports))

	var wg sync.WaitGroup
	wg.Add(len(ports))
	for _, port := range ports {
		port := port
		go func() {
			defer wg.Done()
			dialCtx, dialCancel := context.WithTimeout(ctx, 300*time.Millisecond)
			defer dialCancel()

			conn, err := dial(dialCtx, "tcp", net.JoinHostPort(host, port))
			if err == nil && conn != nil {
				_ = conn.Close()
				results <- result{port: port, ok: true}
				return
			}
			results <- result{port: port, ok: false}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for {
		select {
		case res, ok := <-results:
			if !ok {
				return "9006"
			}
			if res.ok {
				cancel()
				return res.port
			}
		case <-ctx.Done():
			return "9006"
		}
	}
}
