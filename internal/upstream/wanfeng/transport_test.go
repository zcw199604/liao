package wanfeng

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPAndWebSocketShareTunnelDialer(t *testing.T) {
	f := newFakeUpstream(t)
	upstreamHandler := f.srv.Config.Handler
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/files/proxy-test.png" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("downloaded image"))
			return
		}
		upstreamHandler.ServeHTTP(w, r)
	})
	var dials atomic.Int64
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr != "wanfeng.invalid:80" {
			t.Errorf("unexpected tunnel target: %s", addr)
		}
		dials.Add(1)
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(f.URL(), "http://"))
	}
	defer transport.CloseIdleConnections()
	api := NewAPIClient(APIClientConfig{
		BaseURL:    "http://wanfeng.invalid",
		HTTPClient: &http.Client{Transport: transport, Timeout: 2 * time.Second},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Media clients retain the tunnel while isolating identity cookies/sessions.
	clone := api.Clone()
	if clone.http.Jar == api.http.Jar {
		t.Fatal("identity clients must have separate cookie jars")
	}
	if _, err := clone.Session(ctx); err != nil {
		t.Fatalf("HTTP handshake through tunnel: %v", err)
	}
	if _, err := clone.UploadMedia(ctx, "image.png", "image/png", []byte("image data"), "image"); err != nil {
		t.Fatalf("media upload through tunnel: %v", err)
	}
	data, _, err := clone.DownloadMedia(ctx, "/files/proxy-test.png")
	if err != nil || string(data) != "downloaded image" {
		t.Fatalf("media download through tunnel: data=%q err=%v", data, err)
	}
	client := NewClient(ClientConfig{
		IdentityID: "proxy-test", API: api, Bindings: NewMemoryBindingStore(), Emitter: &recordingEmitter{},
	})
	client.Start(ctx)
	defer client.Close()
	select {
	case <-f.connectedSent:
	case <-ctx.Done():
		t.Fatal("WebSocket bypassed the configured HTTP tunnel dialer")
	}
	if dials.Load() < 2 {
		t.Fatal("HTTP and WebSocket should both open tunnel connections")
	}
}
