package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zcw199604/scfproxy/pkg/scfproxy"
	"liao/internal/config"
)

func TestWanfengSCFProxyRoutesTrafficAndClosesConnections(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "proxied "+r.URL.Path)
	}))
	defer target.Close()
	var tunnels atomic.Int64
	upgrader := websocket.Upgrader{}
	// A local SCF protocol peer forwards TCP to the test target. The requested
	// hostname cannot resolve, so a direct connection cannot satisfy this test.
	tunnel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		var hello struct {
			Target string `json:"target"`
			Sign   string `json:"sign"`
		}
		if err := ws.ReadJSON(&hello); err != nil {
			t.Error(err)
			return
		}
		if hello.Target != "wanfeng.invalid:80" || hello.Sign == "" {
			t.Errorf("invalid SCF handshake target/sign: %s", hello.Target)
			return
		}
		conn, err := net.DialTimeout("tcp", strings.TrimPrefix(target.URL, "http://"), time.Second)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		if err := ws.WriteJSON(map[string]any{"ok": true}); err != nil {
			return
		}
		tunnels.Add(1)
		go func() {
			defer ws.Close()
			buf := make([]byte, 8192)
			for {
				n, err := conn.Read(buf)
				if n > 0 {
					if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()
		for {
			kind, reader, err := ws.NextReader()
			if err != nil {
				return
			}
			if kind != websocket.BinaryMessage {
				t.Errorf("unexpected frame: %d", kind)
				return
			}
			if _, err := io.Copy(conn, reader); err != nil {
				return
			}
		}
	}))
	defer tunnel.Close()
	endpoints := filepath.Join(t.TempDir(), "endpoints.txt")
	if err := os.WriteFile(endpoints, []byte("ws"+strings.TrimPrefix(tunnel.URL, "http")), 0600); err != nil {
		t.Fatal(err)
	}
	client, _, closeProxy, err := newWanfengHTTPClient(config.Config{
		WanfengProxyEnabled: true, WanfengTimeoutSeconds: 3,
		SCFProxyEndpointsFile: endpoints, SCFProxyToken: "test-only-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer closeProxy()
	// Environment proxies must not redirect requests outside the SCF tunnel.
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	response, err := client.Get("http://wanfeng.invalid/media")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(body) != "proxied /media" || tunnels.Load() != 1 {
		t.Fatalf("body=%q tunnels=%d err=%v", body, tunnels.Load(), err)
	}
	// Shutdown closes active tunnels as well as idle HTTP connections.
	transport := client.Transport.(*http.Transport)
	conn, err := transport.DialContext(context.Background(), "tcp", "wanfeng.invalid:80")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	closeProxy()
	if _, err := conn.Write([]byte("after shutdown")); err == nil {
		t.Fatal("shutdown left the tunnel writable")
	}
	if _, err := client.Get("http://wanfeng.invalid/after-close"); err == nil {
		t.Fatal("closed proxy unexpectedly succeeded")
	}
}

func writeWanfengTestPool(t *testing.T, size int) string {
	t.Helper()
	entries := []map[string]any{
		{"url": "ws://127.0.0.1:19001", "region": "ap-shanghai", "function": "scfproxy-main", "egress_ip": "192.0.2.1"},
		{"url": "ws://127.0.0.1:19002", "region": "ap-beijing", "function": "scfproxy-standby", "egress_ip": "192.0.2.2"},
		{"url": "ws://127.0.0.1:19003", "region": "ap-chengdu", "function": "scfproxy-spare", "egress_ip": "192.0.2.3"},
	}
	data, err := json.Marshal(map[string]any{"version": 1, "active": entries[:size], "retired": []any{}, "banned_ips": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "pool.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestWanfengIPBanSwitchesPoolOnceForConcurrentReports(t *testing.T) {
	file := writeWanfengTestPool(t, 3)
	_, egress, cleanup, err := newWanfengHTTPClient(config.Config{WanfengProxyEnabled: true, SCFProxyPoolFile: file, SCFProxyToken: "test-only-token"})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	old := egress.Current()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := egress.Switch(old); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if egress.Current() != "ws://127.0.0.1:19002" {
		t.Fatalf("stale ban rotated a healthy standby: %s", egress.Current())
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var pool struct {
		Active    []json.RawMessage `json:"active"`
		Retired   []json.RawMessage `json:"retired"`
		BannedIPs []json.RawMessage `json:"banned_ips"`
	}
	if err := json.Unmarshal(data, &pool); err != nil {
		t.Fatal(err)
	}
	if len(pool.Active) != 2 || len(pool.Retired) != 1 || len(pool.BannedIPs) != 1 {
		t.Fatalf("unexpected pool state: %s", data)
	}
}

func TestWanfengIPBanWithoutStandbyDoesNotMutatePool(t *testing.T) {
	file := writeWanfengTestPool(t, 1)
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	_, egress, cleanup, err := newWanfengHTTPClient(config.Config{WanfengProxyEnabled: true, SCFProxyPoolFile: file, SCFProxyToken: "test-only-token"})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := egress.Switch(egress.Current()); !errors.Is(err, scfproxy.ErrNoStandby) {
		t.Fatalf("expected no standby: %v", err)
	}
	after, err := os.ReadFile(file)
	if err != nil || string(before) != string(after) {
		t.Fatalf("failed switch mutated pool: %v", err)
	}
}

func TestWanfengProxyConfigurationDoesNotFallBackToDirect(t *testing.T) {
	for _, cfg := range []config.Config{
		{WanfengProxyEnabled: true},
		{WanfengProxyEnabled: true, SCFProxyToken: "test", SCFProxyEndpointsFile: "/missing/endpoints.txt"},
		{WanfengProxyEnabled: true, SCFProxyToken: "test", SCFProxyPoolFile: "/missing/pool.json"},
	} {
		client, _, cleanup, err := newWanfengHTTPClient(cfg)
		if err == nil || client != nil || cleanup != nil {
			t.Fatalf("invalid proxy configuration returned a usable client: err=%v", err)
		}
	}
	client, _, cleanup, err := newWanfengHTTPClient(config.Config{WanfengTimeoutSeconds: 7})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if client.Transport != nil || client.Timeout != 7*time.Second {
		t.Fatal("disabled proxy should preserve the existing HTTP client")
	}
}

func TestWanfengProxyFailureDoesNotReachDirectTarget(t *testing.T) {
	var directRequests atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		directRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	tunnel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		if _, _, err := ws.ReadMessage(); err != nil {
			return
		}
		_ = ws.WriteJSON(map[string]any{"ok": false, "code": "dial_failed", "message": "test rejection"})
	}))
	defer tunnel.Close()
	file := filepath.Join(t.TempDir(), "endpoints.txt")
	if err := os.WriteFile(file, []byte("ws"+strings.TrimPrefix(tunnel.URL, "http")), 0600); err != nil {
		t.Fatal(err)
	}
	client, _, cleanup, err := newWanfengHTTPClient(config.Config{
		WanfengProxyEnabled: true, WanfengTimeoutSeconds: 2,
		SCFProxyEndpointsFile: file, SCFProxyToken: "test-only-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	response, err := client.Get(target.URL)
	if response != nil {
		response.Body.Close()
	}
	if err == nil || directRequests.Load() != 0 {
		t.Fatalf("failed tunnel fell back to reachable direct target: requests=%d err=%v", directRequests.Load(), err)
	}
}
