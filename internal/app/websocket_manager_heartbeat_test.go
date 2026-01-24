package app

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestUpstreamWebSocketClient_Heartbeat(t *testing.T) {
	oldInterval := wsUpstreamPingInterval
	wsUpstreamPingInterval = 5 * time.Millisecond
	t.Cleanup(func() { wsUpstreamPingInterval = oldInterval })

	t.Run("conn nil continues", func(t *testing.T) {
		c := NewUpstreamWebSocketClient("u", "ws://unused", nil)
		go c.heartbeat()
		time.Sleep(15 * time.Millisecond)
		c.CloseExpected()
	})

	t.Run("conn not nil writes ping", func(t *testing.T) {
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

		go c.heartbeat()
		time.Sleep(15 * time.Millisecond)
		c.CloseExpected()
	})
}
