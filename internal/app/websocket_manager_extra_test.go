package app

import "testing"

func TestMD5HexLower(t *testing.T) {
	if got := md5HexLower(" "); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := md5HexLower("x"); got != "9dd4e461268c8034f5c8564e155c67a6" {
		t.Fatalf("got=%q", got)
	}
}

func TestUpstreamWebSocketClient_CloseUnexpected(t *testing.T) {
	c := NewUpstreamWebSocketClient("u1", "ws://example", nil)
	c.expectedClose.Store(true)

	c.CloseUnexpected()
	if c.expectedClose.Load() {
		t.Fatalf("expectedClose should be false")
	}

	select {
	case <-c.done:
	default:
		t.Fatalf("expected done closed")
	}
}

func TestUpstreamWebSocketManager_HandleUpstreamDisconnect(t *testing.T) {
	m := NewUpstreamWebSocketManager(nil, "", nil, nil, nil)
	session := &DownstreamSession{}

	m.upstreamClients["u1"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m.connectionCreateMilli["u1"] = 123
	m.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{session: {}}

	m.HandleUpstreamDisconnect("u1")

	if _, ok := m.upstreamClients["u1"]; ok {
		t.Fatalf("expected upstream removed")
	}
	if _, ok := m.connectionCreateMilli["u1"]; ok {
		t.Fatalf("expected ts removed")
	}
	if _, ok := m.downstreamSessions["u1"]; ok {
		t.Fatalf("expected downstream removed")
	}

	// sessions 为空应直接返回
	m.upstreamClients["u2"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m.connectionCreateMilli["u2"] = 456
	m.HandleUpstreamDisconnect("u2")
	if _, ok := m.upstreamClients["u2"]; ok {
		t.Fatalf("expected upstream removed")
	}
}
