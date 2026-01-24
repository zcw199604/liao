package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestUpstreamWebSocketManager_RegisterAndUnregister_EdgeBranches(t *testing.T) {
	m := NewUpstreamWebSocketManager(nil, "ws://unused", NewForceoutManager(), nil, nil)

	m.RegisterDownstream("", &DownstreamSession{}, "")
	m.RegisterDownstream("u1", nil, "")

	m.forceout.AddForceoutUser("u2")
	m.RegisterDownstream("u2", &DownstreamSession{}, "")

	m.mu.Lock()
	_, has := m.downstreamSessions["u2"]
	m.mu.Unlock()
	if has {
		t.Fatalf("expected forbidden user not registered")
	}

	// existing upstream should not create new one
	existing := &UpstreamWebSocketClient{done: make(chan struct{})}
	m.mu.Lock()
	m.upstreamClients["u3"] = existing
	m.mu.Unlock()
	s := &DownstreamSession{}
	m.RegisterDownstream("u3", s, "")
	m.mu.Lock()
	if m.upstreamClients["u3"] != existing {
		t.Fatalf("expected existing upstream kept")
	}
	m.mu.Unlock()

	// unregister when multiple sessions remain should not schedule close
	m2 := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil, nil)
	s1 := &DownstreamSession{}
	s2 := &DownstreamSession{}
	m2.mu.Lock()
	m2.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{s1: {}, s2: {}}
	m2.mu.Unlock()
	m2.UnregisterDownstream("u1", s1)
	m2.mu.Lock()
	if _, ok := m2.pendingCloseTasks["u1"]; ok {
		m2.mu.Unlock()
		t.Fatalf("should not schedule close")
	}
	if len(m2.downstreamSessions["u1"]) != 1 {
		m2.mu.Unlock()
		t.Fatalf("sessions=%v", m2.downstreamSessions["u1"])
	}
	m2.mu.Unlock()

	// unregister with empty input
	m2.UnregisterDownstream("", s2)
	m2.UnregisterDownstream("u1", nil)

	m.HandleUpstreamDisconnect(" ")
}

func TestUpstreamWebSocketManager_SendToUpstream_Branches(t *testing.T) {
	tracker := &wsConnTracker{}
	received := make(chan string, 1)
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

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil, nil)
	t.Cleanup(m.CloseAllConnections)

	m.SendToUpstream("", `{"x":1}`)

	// no downstream should be noop
	m.SendToUpstream("u0", `{"x":1}`)
	m.mu.Lock()
	_, exists := m.upstreamClients["u0"]
	m.mu.Unlock()
	if exists {
		t.Fatalf("unexpected upstream created without downstream")
	}

	// has downstream but no upstream: should create
	m.mu.Lock()
	m.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{&DownstreamSession{}: {}}
	m.mu.Unlock()
	m.SendToUpstream("u1", `{"x":1}`)

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream message")
	}

	// existing but not open + no downstream should return
	m2 := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil, nil)
	t.Cleanup(m2.CloseAllConnections)
	m2.mu.Lock()
	m2.upstreamClients["u2"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m2.mu.Unlock()
	m2.SendToUpstream("u2", `{"x":2}`)

	// open client path: send directly
	conn, _, err := websocket.DefaultDialer.Dial(toWSURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	openClient := NewUpstreamWebSocketClient("u3", toWSURL(srv.URL), nil)
	openClient.conn = conn
	openClient.connected = true

	m3 := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil, nil)
	t.Cleanup(m3.CloseAllConnections)
	m3.mu.Lock()
	m3.upstreamClients["u3"] = openClient
	m3.downstreamSessions["u3"] = map[*DownstreamSession]struct{}{&DownstreamSession{}: {}}
	m3.mu.Unlock()
	m3.SendToUpstream("u3", `{"x":3}`)

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream message2")
	}
	openClient.CloseExpected()
}

func TestUpstreamWebSocketManager_RegisterDownstream_EvictionStopsPendingTask(t *testing.T) {
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

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil, nil)
	t.Cleanup(m.CloseAllConnections)

	s1 := &DownstreamSession{}
	s2 := &DownstreamSession{}
	s3 := &DownstreamSession{}

	m.RegisterDownstream("u1", s1, "")
	m.RegisterDownstream("u2", s2, "")

	m.mu.Lock()
	m.connectionCreateMilli["u1"] = 1
	m.connectionCreateMilli["u2"] = 2
	m.pendingCloseTasks["u1"] = time.AfterFunc(time.Hour, func() {})
	m.mu.Unlock()

	m.RegisterDownstream("u3", s3, "")

	time.Sleep(50 * time.Millisecond)

	m.mu.Lock()
	_, hasTask := m.pendingCloseTasks["u1"]
	m.mu.Unlock()
	if hasTask {
		t.Fatalf("expected pending task removed for evicted user")
	}
}

func TestUpstreamWebSocketManager_ScheduleCloseUpstreamLocked_Branches(t *testing.T) {
	oldDelay := wsCloseDelay
	wsCloseDelay = 20 * time.Millisecond
	t.Cleanup(func() { wsCloseDelay = oldDelay })

	m := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil, nil)

	// replace existing timer
	m.mu.Lock()
	m.pendingCloseTasks["u1"] = time.AfterFunc(time.Hour, func() {})
	m.scheduleCloseUpstreamLocked("u1")
	m.mu.Unlock()

	// when sessions exist, timer should self-cancel without closing
	m.mu.Lock()
	m.downstreamSessions["u2"] = map[*DownstreamSession]struct{}{&DownstreamSession{}: {}}
	m.upstreamClients["u2"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m.connectionCreateMilli["u2"] = 1
	m.scheduleCloseUpstreamLocked("u2")
	m.mu.Unlock()

	time.Sleep(50 * time.Millisecond)

	m.mu.Lock()
	_, stillUp := m.upstreamClients["u2"]
	_, hasTask := m.pendingCloseTasks["u2"]
	m.mu.Unlock()
	if !stillUp {
		t.Fatalf("expected upstream not closed when sessions exist")
	}
	if hasTask {
		t.Fatalf("expected pending task cleared")
	}

	// CloseUpstreamConnection empty input
	m.CloseUpstreamConnection("")
}

func TestUpstreamWebSocketManager_BroadcastAndForceout_Branches(t *testing.T) {
	oldDelay := wsForceoutDelay
	wsForceoutDelay = 0
	t.Cleanup(func() { wsForceoutDelay = oldDelay })

	m := NewUpstreamWebSocketManager(nil, "ws://unused", NewForceoutManager(), nil, nil)

	// broadcast with nil session key
	session := &DownstreamSession{}
	m.mu.Lock()
	m.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{nil: {}, session: {}}
	m.mu.Unlock()
	m.BroadcastToDownstream("u1", `{"x":1}`)

	m.HandleForceout("", "x")
	m.HandleForceout("u1", `{"code":-3,"forceout":true}`)
	time.Sleep(10 * time.Millisecond)

	if !m.forceout.IsForbidden("u1") {
		t.Fatalf("expected forceout recorded")
	}
	m.mu.Lock()
	_, has := m.downstreamSessions["u1"]
	m.mu.Unlock()
	if has {
		t.Fatalf("expected downstream cleared")
	}
}

func TestUpstreamWebSocketManager_GetConnectionStats_NonZero(t *testing.T) {
	m := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil, nil)
	m.mu.Lock()
	m.upstreamClients["u1"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{&DownstreamSession{}: {}, &DownstreamSession{}: {}}
	m.mu.Unlock()

	stats := m.GetConnectionStats()
	if stats["active"].(int) == 0 || stats["upstream"].(int) != 1 || stats["downstream"].(int) != 2 {
		t.Fatalf("stats=%v", stats)
	}
}

func TestUpstreamWebSocketManager_CloseAllConnections_StopsTimersAndCloses(t *testing.T) {
	m := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil, nil)

	client := &UpstreamWebSocketClient{done: make(chan struct{})}
	timer := time.AfterFunc(time.Hour, func() {})

	m.mu.Lock()
	m.upstreamClients["u1"] = client
	m.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{nil: {}, &DownstreamSession{}: {}}
	m.pendingCloseTasks["u1"] = timer
	m.mu.Unlock()

	m.CloseAllConnections()

	select {
	case <-client.done:
	default:
		t.Fatalf("expected upstream done closed")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.upstreamClients) != 0 || len(m.downstreamSessions) != 0 || len(m.pendingCloseTasks) != 0 || len(m.connectionCreateMilli) != 0 {
		t.Fatalf("unexpected state: %+v", m)
	}
}

func TestUpstreamWebSocketManager_RegisterDownstream_StopsPendingCloseTask(t *testing.T) {
	m := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil, nil)

	timer := time.AfterFunc(time.Hour, func() {})
	m.mu.Lock()
	m.pendingCloseTasks["u1"] = timer
	m.upstreamClients["u1"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m.mu.Unlock()

	m.RegisterDownstream("u1", &DownstreamSession{}, "")

	m.mu.Lock()
	_, hasTask := m.pendingCloseTasks["u1"]
	m.mu.Unlock()
	if hasTask {
		t.Fatalf("expected pending task removed")
	}
}

func TestUpstreamWebSocketManager_RegisterDownstream_SkipsEvictingSelfInCreateTimeLoop(t *testing.T) {
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

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil, nil)
	t.Cleanup(m.CloseAllConnections)

	m.mu.Lock()
	m.upstreamClients["u1"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m.upstreamClients["u2"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	// Stale timestamp for the incoming identity to exercise the "uid == userID" continue branch.
	m.connectionCreateMilli["u3"] = 1
	m.mu.Unlock()

	m.RegisterDownstream("u3", &DownstreamSession{}, "")
}

func TestUpstreamWebSocketManager_CreateUpstreamConnection_FastPathAndRacyPath(t *testing.T) {
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

	m := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil, nil)
	t.Cleanup(m.CloseAllConnections)

	// fast path existing
	existing := &UpstreamWebSocketClient{done: make(chan struct{})}
	m.mu.Lock()
	m.upstreamClients["u1"] = existing
	m.mu.Unlock()
	if got := m.createUpstreamConnection("u1", ""); got != existing {
		t.Fatalf("expected existing returned")
	}

	// race-ish: concurrent calls should converge to one client
	const loops = 10
	for i := 0; i < loops; i++ {
		userID := "u" + string(rune('a'+i))
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = m.createUpstreamConnection(userID, `{"act":"sign"}`)
		}()
		go func() {
			defer wg.Done()
			_ = m.createUpstreamConnection(userID, "")
		}()
		wg.Wait()
		m.CloseUpstreamConnection(userID)
	}
}

func TestUpstreamWebSocketManager_CreateUpstreamConnection_SecondCheckClosesNew(t *testing.T) {
	blocked := make(chan struct{})
	release := make(chan struct{})

	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		close(blocked)
		<-release
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"state":"OK","msg":{"server":"ws://unused"}}`)),
		}, nil
	})}

	m := NewUpstreamWebSocketManager(client, "ws://fallback", nil, nil, nil)

	resultCh := make(chan *UpstreamWebSocketClient, 1)
	go func() {
		resultCh <- m.createUpstreamConnection("u1", "")
	}()

	<-blocked
	existing := &UpstreamWebSocketClient{done: make(chan struct{})}
	m.mu.Lock()
	m.upstreamClients["u1"] = existing
	m.mu.Unlock()
	close(release)

	got := <-resultCh
	if got != existing {
		t.Fatalf("got=%v want=%v", got, existing)
	}
}

func TestUpstreamWebSocketClient_connect_UnexpectedCloseCallsDisconnect(t *testing.T) {
	tracker := &wsConnTracker{}
	srv := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		_, _, _ = conn.ReadMessage() // wait for flushPending
		_ = conn.WriteControl(websocket.PongMessage, []byte("p"), time.Now().Add(time.Second))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"code":0}`))
		_ = conn.Close()
	})
	t.Cleanup(func() {
		tracker.closeAll()
		srv.Close()
	})

	m := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil, nil)
	session := &DownstreamSession{}
	c := NewUpstreamWebSocketClient("u1", toWSURL(srv.URL), m)
	c.pending = []string{"hello"}

	m.mu.Lock()
	m.upstreamClients["u1"] = c
	m.connectionCreateMilli["u1"] = 1
	m.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{session: {}}
	m.mu.Unlock()

	c.connect()
	c.CloseUnexpected()

	m.mu.Lock()
	_, still := m.upstreamClients["u1"]
	m.mu.Unlock()
	if still {
		t.Fatalf("expected disconnect cleanup")
	}
}

func TestUpstreamWebSocketClient_ConnectAndMessaging_Branches(t *testing.T) {
	// dial error should trigger disconnect cleanup
	m := NewUpstreamWebSocketManager(nil, "ws://unused", nil, nil, nil)
	session := &DownstreamSession{}
	m.mu.Lock()
	m.upstreamClients["u1"] = &UpstreamWebSocketClient{done: make(chan struct{})}
	m.connectionCreateMilli["u1"] = 1
	m.downstreamSessions["u1"] = map[*DownstreamSession]struct{}{session: {}}
	m.mu.Unlock()

	c := NewUpstreamWebSocketClient("u1", "ws://127.0.0.1:0", m)
	c.connect()
	m.mu.Lock()
	_, still := m.upstreamClients["u1"]
	m.mu.Unlock()
	if still {
		t.Fatalf("expected disconnect cleanup")
	}

	// expectedClose should skip disconnect callback
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

	m2 := NewUpstreamWebSocketManager(nil, toWSURL(srv.URL), nil, nil, nil)
	c2 := NewUpstreamWebSocketClient("u2", toWSURL(srv.URL), m2)
	go c2.connect()
	deadline := time.Now().Add(2 * time.Second)
	for !c2.IsOpen() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	c2.CloseExpected()
}

func TestUpstreamWebSocketClient_FlushPending_ConnNilAndWriteError(t *testing.T) {
	t.Run("conn nil clears queue", func(t *testing.T) {
		c := NewUpstreamWebSocketClient("u", "ws://unused", nil)
		c.pending = []string{"x"}
		c.flushPending()
		if len(c.pending) != 0 {
			t.Fatalf("pending=%v", c.pending)
		}
	})

	t.Run("write error triggers CloseUnexpected", func(t *testing.T) {
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
			t.Fatalf("dial: %v", err)
		}
		c := NewUpstreamWebSocketClient("u", toWSURL(srv.URL), nil)
		c.conn = conn
		c.connected = true
		c.pending = []string{"x"}

		_ = conn.Close()
		c.flushPending()

		select {
		case <-c.done:
		default:
			t.Fatalf("expected done closed")
		}
	})
}

func TestUpstreamWebSocketClient_SendMessageAndFlushPending_Branches(t *testing.T) {
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

	c := NewUpstreamWebSocketClient("u", toWSURL(srv.URL), nil)
	c.SendMessage(" ")
	c.SendMessage("x")
	if len(c.pending) != 1 {
		t.Fatalf("pending=%v", c.pending)
	}

	conn, _, err := websocket.DefaultDialer.Dial(toWSURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c.conn = conn
	c.connected = true
	c.flushPending()

	// force write error path
	_ = conn.Close()
	c.SendMessage("y")
}

func TestUpstreamWebSocketClient_OnMessage_HistoryAndBranches(t *testing.T) {
	history := &stubChatHistoryCache{}
	cache := &spyUserInfoCache{}
	m := NewUpstreamWebSocketManager(nil, "ws://unused", NewForceoutManager(), cache, history)
	c := NewUpstreamWebSocketClient("u1", "ws://unused", m)

	// JSON parse error should still broadcast (no panic)
	c.onMessage("not-json")

	// code 15 with empty target no-op
	c.onMessage(`{"code":15,"sel_userid":" "}`)

	// code 7: msgType default + no compat writes when c.userID matches fromUserID
	c.onMessage(`{"code":7,"fromuser":{"id":"u1","content":"hi","time":"t1"},"touser":{"id":"u2"},"Tid":"tid1"}`)

	cache.mu.Lock()
	gotMsgs := len(cache.messages)
	cache.mu.Unlock()
	if gotMsgs != 1 {
		t.Fatalf("messages=%d want=1", gotMsgs)
	}

	// history save with md5 normalization + nickname
	hashed := md5HexLower("u1")
	payload := map[string]any{
		"code": 7,
		"fromuser": map[string]any{
			"id":       hashed,
			"content":  "x",
			"time":     "t2",
			"type":     "text",
			"nickname": "n",
			"Tid":      "tid2",
		},
		"touser": map[string]any{
			"id": "u3",
		},
	}
	b, _ := json.Marshal(payload)
	c.onMessage(string(b))

	history.mu.Lock()
	if len(history.saved) == 0 {
		history.mu.Unlock()
		t.Fatalf("expected history saved")
	}
	msg := history.saved[len(history.saved)-1]
	history.mu.Unlock()
	if msg["id"] != "u1" || msg["toid"] != "u3" || msg["nickname"] != "n" {
		t.Fatalf("msg=%v", msg)
	}
}
