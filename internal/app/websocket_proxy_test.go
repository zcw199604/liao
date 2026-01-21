package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHandleWebSocket_UnauthorizedWithoutToken(t *testing.T) {
	a := &App{jwt: NewJWTService("secret-1", 1)}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/ws", nil)
	rr := httptest.NewRecorder()

	a.handleWebSocket(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rr.Body.String(), "WebSocket连接Token无效") {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestHandleWebSocket_ProxiesMessagesAfterSign(t *testing.T) {
	received := make(chan string, 10)
	tracker := &wsConnTracker{}
	upstream := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			received <- string(data)
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		upstream.Close()
	})

	wsManager := NewUpstreamWebSocketManager(nil, toWSURL(upstream.URL), nil, nil, nil)
	t.Cleanup(wsManager.CloseAllConnections)

	jwtService := NewJWTService("secret-1", 1)
	token, err := jwtService.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	app := &App{
		jwt:       jwtService,
		wsManager: wsManager,
	}

	backend := httptest.NewServer(http.HandlerFunc(app.handleWebSocket))
	t.Cleanup(backend.Close)

	wsURL := toWSURL(backend.URL) + "/ws?token=" + url.QueryEscape(token)
	downstream, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial downstream failed: %v", err)
	}
	t.Cleanup(func() { _ = downstream.Close() })

	// before sign: should be ignored
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(`{"act":"noop","id":"u1"}`)); err != nil {
		t.Fatalf("send pre-sign failed: %v", err)
	}

	sign := `{"act":"sign","id":"u1"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(sign)); err != nil {
		t.Fatalf("send sign failed: %v", err)
	}

	select {
	case got := <-received:
		if got != sign {
			t.Fatalf("sign=%q, want %q", got, sign)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream sign")
	}

	payload := `{"act":"say","id":"u1","content":"hi"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		t.Fatalf("send payload failed: %v", err)
	}

	select {
	case got := <-received:
		if got != payload {
			t.Fatalf("payload=%q, want %q", got, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream payload")
	}
}

func TestHandleWebSocket_IgnoresBinaryAndInvalidJSON(t *testing.T) {
	received := make(chan string, 10)
	tracker := &wsConnTracker{}
	upstream := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			received <- string(data)
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		upstream.Close()
	})

	wsManager := NewUpstreamWebSocketManager(nil, toWSURL(upstream.URL), nil, nil, nil)
	t.Cleanup(wsManager.CloseAllConnections)

	jwtService := NewJWTService("secret-1", 1)
	token, err := jwtService.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	app := &App{
		jwt:       jwtService,
		wsManager: wsManager,
	}

	backend := httptest.NewServer(http.HandlerFunc(app.handleWebSocket))
	t.Cleanup(backend.Close)

	wsURL := toWSURL(backend.URL) + "/ws?token=" + url.QueryEscape(token)
	downstream, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial downstream failed: %v", err)
	}
	t.Cleanup(func() { _ = downstream.Close() })

	if err := downstream.WriteMessage(websocket.BinaryMessage, []byte("bin")); err != nil {
		t.Fatalf("send binary failed: %v", err)
	}
	if err := downstream.WriteMessage(websocket.TextMessage, []byte("{")); err != nil {
		t.Fatalf("send invalid json failed: %v", err)
	}

	sign := `{"act":"sign","id":"u1"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(sign)); err != nil {
		t.Fatalf("send sign failed: %v", err)
	}

	select {
	case got := <-received:
		if got != sign {
			t.Fatalf("sign=%q, want %q", got, sign)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream sign")
	}
}

func TestHandleWebSocket_IgnoresMismatchedUserIDAfterSign(t *testing.T) {
	received := make(chan string, 10)
	tracker := &wsConnTracker{}
	upstream := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			received <- string(data)
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		upstream.Close()
	})

	wsManager := NewUpstreamWebSocketManager(nil, toWSURL(upstream.URL), nil, nil, nil)
	t.Cleanup(wsManager.CloseAllConnections)

	jwtService := NewJWTService("secret-1", 1)
	token, err := jwtService.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	app := &App{
		jwt:       jwtService,
		wsManager: wsManager,
	}

	backend := httptest.NewServer(http.HandlerFunc(app.handleWebSocket))
	t.Cleanup(backend.Close)

	wsURL := toWSURL(backend.URL) + "/ws?token=" + url.QueryEscape(token)
	downstream, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial downstream failed: %v", err)
	}
	t.Cleanup(func() { _ = downstream.Close() })

	sign := `{"act":"sign","id":"u1"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(sign)); err != nil {
		t.Fatalf("send sign failed: %v", err)
	}
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream sign")
	}

	mismatch := `{"act":"say","id":"u2","content":"bad"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(mismatch)); err != nil {
		t.Fatalf("send mismatch failed: %v", err)
	}

	select {
	case got := <-received:
		t.Fatalf("unexpected upstream message: %q", got)
	case <-time.After(200 * time.Millisecond):
		// ok
	}

	okMessage := `{"act":"say","id":"u1","content":"ok"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(okMessage)); err != nil {
		t.Fatalf("send ok failed: %v", err)
	}
	select {
	case got := <-received:
		if got != okMessage {
			t.Fatalf("message=%q, want %q", got, okMessage)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream message")
	}
}

func TestHandleWebSocket_SignSwitch_RebindsSession(t *testing.T) {
	received := make(chan string, 10)
	tracker := &wsConnTracker{}
	upstream := newUpstreamWSServer(t, func(conn *websocket.Conn) {
		tracker.add(conn)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			received <- string(data)
		}
	})
	t.Cleanup(func() {
		tracker.closeAll()
		upstream.Close()
	})

	wsManager := NewUpstreamWebSocketManager(nil, toWSURL(upstream.URL), nil, nil, nil)
	t.Cleanup(wsManager.CloseAllConnections)

	jwtService := NewJWTService("secret-1", 1)
	token, err := jwtService.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	app := &App{
		jwt:       jwtService,
		wsManager: wsManager,
	}

	backend := httptest.NewServer(http.HandlerFunc(app.handleWebSocket))
	t.Cleanup(backend.Close)

	wsURL := toWSURL(backend.URL) + "/ws?token=" + url.QueryEscape(token)
	downstream, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial downstream failed: %v", err)
	}
	t.Cleanup(func() { _ = downstream.Close() })

	signU1 := `{"act":"sign","id":"u1"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(signU1)); err != nil {
		t.Fatalf("send sign u1 failed: %v", err)
	}
	select {
	case got := <-received:
		if got != signU1 {
			t.Fatalf("sign=%q, want %q", got, signU1)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream sign")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		wsManager.mu.Lock()
		got := len(wsManager.downstreamSessions["u1"])
		wsManager.mu.Unlock()
		if got == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	wsManager.mu.Lock()
	gotU1 := len(wsManager.downstreamSessions["u1"])
	wsManager.mu.Unlock()
	if gotU1 != 1 {
		t.Fatalf("downstream sessions for u1=%d, want 1", gotU1)
	}

	signU2 := `{"act":"sign","id":"u2"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(signU2)); err != nil {
		t.Fatalf("send sign u2 failed: %v", err)
	}
	select {
	case got := <-received:
		if got != signU2 {
			t.Fatalf("sign=%q, want %q", got, signU2)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting upstream sign u2")
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		wsManager.mu.Lock()
		u1 := len(wsManager.downstreamSessions["u1"])
		u2 := len(wsManager.downstreamSessions["u2"])
		wsManager.mu.Unlock()
		if u1 == 0 && u2 == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	wsManager.mu.Lock()
	gotU1 = len(wsManager.downstreamSessions["u1"])
	gotU2 := len(wsManager.downstreamSessions["u2"])
	wsManager.mu.Unlock()
	if gotU1 != 0 || gotU2 != 1 {
		t.Fatalf("u1 sessions=%d, u2 sessions=%d, want 0 and 1", gotU1, gotU2)
	}
}

func TestHandleWebSocket_RejectsForbiddenUserOnSign(t *testing.T) {
	forceout := NewForceoutManager()
	forceout.AddForceoutUser("u1")

	wsManager := NewUpstreamWebSocketManager(nil, "ws://unused", forceout, nil, nil)
	t.Cleanup(wsManager.CloseAllConnections)

	jwtService := NewJWTService("secret-1", 1)
	token, err := jwtService.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	app := &App{
		jwt:       jwtService,
		wsManager: wsManager,
	}

	backend := httptest.NewServer(http.HandlerFunc(app.handleWebSocket))
	t.Cleanup(backend.Close)

	wsURL := toWSURL(backend.URL) + "/ws?token=" + url.QueryEscape(token)
	downstream, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial downstream failed: %v", err)
	}
	t.Cleanup(func() { _ = downstream.Close() })

	sign := `{"act":"sign","id":"u1"}`
	if err := downstream.WriteMessage(websocket.TextMessage, []byte(sign)); err != nil {
		t.Fatalf("send sign failed: %v", err)
	}

	_ = downstream.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := downstream.ReadMessage()
	if err != nil {
		t.Fatalf("read reject failed: %v", err)
	}

	var node map[string]any
	if err := json.Unmarshal(payload, &node); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := node["code"].(float64); got != -4 {
		t.Fatalf("code=%v, want -4", node["code"])
	}
	if got, _ := node["forceout"].(bool); !got {
		t.Fatalf("forceout=%v, want true", node["forceout"])
	}
	if got, _ := node["content"].(string); !strings.Contains(got, "暂时禁止") {
		t.Fatalf("content=%q, want contains 暂时禁止", got)
	}

	_ = downstream.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	_, _, err = downstream.ReadMessage()
	if err == nil {
		t.Fatalf("expected downstream closed")
	}
}
