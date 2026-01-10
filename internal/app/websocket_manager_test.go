package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type wsConnTracker struct {
	mu    sync.Mutex
	conns []*websocket.Conn
}

func (t *wsConnTracker) add(conn *websocket.Conn) {
	t.mu.Lock()
	t.conns = append(t.conns, conn)
	t.mu.Unlock()
}

func (t *wsConnTracker) closeAll() {
	t.mu.Lock()
	conns := append([]*websocket.Conn(nil), t.conns...)
	t.conns = nil
	t.mu.Unlock()

	for _, c := range conns {
		_ = c.Close()
	}
}

func toWSURL(httpURL string) string {
	if strings.HasPrefix(httpURL, "https://") {
		return "wss://" + strings.TrimPrefix(httpURL, "https://")
	}
	return "ws://" + strings.TrimPrefix(httpURL, "http://")
}

func newUpstreamWSServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handler(conn)
	}))
}

func TestUpstreamWebSocketManager_RegisterDownstream_SendsSignMessage(t *testing.T) {
	received := make(chan string, 1)
	tracker := &wsConnTracker{}
	srv := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		_, data, err := conn.ReadMessage()
		if err == nil {
			received <- string(data)
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		srv.Close()
	})

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil)
	t.Cleanup(m.CloseAllConnections)

	session := &DownstreamSession{}
	sign := `{"act":"sign","id":"u1"}`
	m.RegisterDownstream("u1", session, sign)

	select {
	case got := <-received:
		if got != sign {
			t.Fatalf("sign=%q, want %q", got, sign)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting sign message")
	}
}

func TestUpstreamWebSocketManager_UnregisterDownstream_ClosesAfterDelay(t *testing.T) {
	oldDelay := wsCloseDelay
	wsCloseDelay = 20 * time.Millisecond
	t.Cleanup(func() { wsCloseDelay = oldDelay })

	tracker := &wsConnTracker{}
	srv := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		srv.Close()
	})

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil)
	t.Cleanup(m.CloseAllConnections)

	session := &DownstreamSession{}
	m.RegisterDownstream("u1", session, "")

	m.mu.Lock()
	_, existsBefore := m.upstreamClients["u1"]
	m.mu.Unlock()
	if !existsBefore {
		t.Fatalf("expected upstream client created")
	}

	m.UnregisterDownstream("u1", session)

	time.Sleep(50 * time.Millisecond)

	m.mu.Lock()
	_, existsAfter := m.upstreamClients["u1"]
	_, taskExists := m.pendingCloseTasks["u1"]
	m.mu.Unlock()
	if existsAfter {
		t.Fatalf("expected upstream client closed")
	}
	if taskExists {
		t.Fatalf("expected pending close task cleaned")
	}
}

func TestUpstreamWebSocketManager_ReRegister_CancelsPendingClose(t *testing.T) {
	oldDelay := wsCloseDelay
	wsCloseDelay = 60 * time.Millisecond
	t.Cleanup(func() { wsCloseDelay = oldDelay })

	tracker := &wsConnTracker{}
	srv := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		srv.Close()
	})

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil)
	t.Cleanup(m.CloseAllConnections)

	session := &DownstreamSession{}
	m.RegisterDownstream("u1", session, "")
	m.UnregisterDownstream("u1", session)

	m.mu.Lock()
	_, hasTask := m.pendingCloseTasks["u1"]
	m.mu.Unlock()
	if !hasTask {
		t.Fatalf("expected pending close task scheduled")
	}

	m.RegisterDownstream("u1", session, "")

	time.Sleep(100 * time.Millisecond)

	m.mu.Lock()
	_, existsAfter := m.upstreamClients["u1"]
	_, taskExists := m.pendingCloseTasks["u1"]
	m.mu.Unlock()
	if !existsAfter {
		t.Fatalf("expected upstream still alive after cancel")
	}
	if taskExists {
		t.Fatalf("expected pending close task canceled")
	}
}

func TestUpstreamWebSocketManager_RegisterDownstream_EvictsOldestIdentity(t *testing.T) {
	oldDelay := wsEvictionDelay
	wsEvictionDelay = 10 * time.Millisecond
	t.Cleanup(func() { wsEvictionDelay = oldDelay })

	tracker := &wsConnTracker{}
	srv := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		srv.Close()
	})

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil)
	t.Cleanup(m.CloseAllConnections)

	s1 := &DownstreamSession{}
	s2 := &DownstreamSession{}
	s3 := &DownstreamSession{}

	m.RegisterDownstream("u1", s1, "")
	m.RegisterDownstream("u2", s2, "")

	m.mu.Lock()
	m.connectionCreateMilli["u1"] = 1
	m.connectionCreateMilli["u2"] = 2
	m.mu.Unlock()

	m.RegisterDownstream("u3", s3, "")

	time.Sleep(50 * time.Millisecond)

	m.mu.Lock()
	_, hasU1 := m.upstreamClients["u1"]
	_, hasU2 := m.upstreamClients["u2"]
	_, hasU3 := m.upstreamClients["u3"]
	m.mu.Unlock()

	if hasU1 {
		t.Fatalf("expected u1 evicted")
	}
	if !hasU2 || !hasU3 {
		t.Fatalf("expected u2/u3 remain")
	}
}

type spyUserInfoCache struct {
	mu        sync.Mutex
	userInfos []CachedUserInfo
	messages  []CachedLastMessage
}

func (s *spyUserInfoCache) SaveUserInfo(info CachedUserInfo) {
	s.mu.Lock()
	s.userInfos = append(s.userInfos, info)
	s.mu.Unlock()
}

func (s *spyUserInfoCache) GetUserInfo(string) *CachedUserInfo { return nil }

func (s *spyUserInfoCache) EnrichUserInfo(_ string, originalData map[string]any) map[string]any {
	return originalData
}

func (s *spyUserInfoCache) BatchEnrichUserInfo(userList []map[string]any, _ string) []map[string]any {
	return userList
}

func (s *spyUserInfoCache) SaveLastMessage(message CachedLastMessage) {
	s.mu.Lock()
	s.messages = append(s.messages, message)
	s.mu.Unlock()
}

func (s *spyUserInfoCache) GetLastMessage(string, string) *CachedLastMessage { return nil }

func (s *spyUserInfoCache) BatchEnrichWithLastMessage(userList []map[string]any, _ string) []map[string]any {
	return userList
}

func TestUpstreamWebSocketClient_OnMessage_CachesUserInfoAndLastMessage(t *testing.T) {
	cache := &spyUserInfoCache{}
	m := NewUpstreamWebSocketManager(nil, "ws://unused", nil, cache)

	c := NewUpstreamWebSocketClient("u0", "ws://unused", m)

	c.onMessage(`{"code":15,"sel_userid":"u9","sel_userNikename":"N","sel_userSex":"M","sel_userAge":"18","sel_userAddress":"CN"}`)

	cache.mu.Lock()
	if len(cache.userInfos) != 1 {
		cache.mu.Unlock()
		t.Fatalf("userInfos=%d, want 1", len(cache.userInfos))
	}
	info := cache.userInfos[0]
	cache.mu.Unlock()

	if info.UserID != "u9" || info.Nickname != "N" || info.Gender != "M" || info.Age != "18" || info.Address != "CN" {
		b, _ := json.Marshal(info)
		t.Fatalf("unexpected user info: %s", string(b))
	}

	msg := `{"code":7,"fromuser":{"id":"u1","content":"hi","time":"t1","type":"text"},"touser":{"id":"u2"}}`
	c.onMessage(msg)

	cache.mu.Lock()
	got := len(cache.messages)
	cache.mu.Unlock()
	if got != 3 {
		t.Fatalf("messages=%d, want 3 (compat writes)", got)
	}
}

func TestUpstreamWebSocketClient_OnMessage_ForceoutMarksForbidden(t *testing.T) {
	oldDelay := wsForceoutDelay
	wsForceoutDelay = 0
	t.Cleanup(func() { wsForceoutDelay = oldDelay })

	forceout := NewForceoutManager()
	m := NewUpstreamWebSocketManager(nil, "ws://unused", forceout, nil)
	c := NewUpstreamWebSocketClient("u1", "ws://unused", m)

	c.onMessage(`{"code":-3,"forceout":true,"content":"x"}`)

	if !forceout.IsForbidden("u1") {
		t.Fatalf("expected forceout marked forbidden")
	}
}

func TestUpstreamWebSocketManager_BroadcastToDownstream_RemovesZombieSession(t *testing.T) {
	tracker := &wsConnTracker{}
	srv := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		srv.Close()
	})

	conn, _, err := websocket.DefaultDialer.Dial(toWSURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	_ = conn.Close()

	session := &DownstreamSession{conn: conn}
	manager := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil)
	t.Cleanup(manager.CloseAllConnections)

	manager.mu.Lock()
	manager.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{session: {}}
	manager.mu.Unlock()

	manager.BroadcastToDownstream("u1", `{"code":1}`)

	manager.mu.Lock()
	sessions := manager.downstreamSessions["u1"]
	manager.mu.Unlock()
	if sessions != nil && len(sessions) != 0 {
		t.Fatalf("expected zombie session removed, got %d", len(sessions))
	}
}
