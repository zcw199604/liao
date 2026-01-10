package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type DownstreamSession struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func (s *DownstreamSession) SendText(message string) error {
	if s == nil || s.conn == nil {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(wsDownstreamWriteDeadline))
	return s.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (s *DownstreamSession) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.Close()
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Spring 侧 setAllowedOrigins("*")，这里保持一致。
		return true
	},
}

func (a *App) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" || !a.jwt.ValidateToken(token) {
		http.Error(w, "WebSocket连接Token无效", http.StatusUnauthorized)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	session := &DownstreamSession{conn: conn}

	var registeredUserID string
	for {
		msgType, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			break
		}
		if msgType != websocket.TextMessage {
			continue
		}

		var node map[string]any
		if err := json.Unmarshal(payload, &node); err != nil {
			continue
		}

		act := strings.TrimSpace(toString(node["act"]))
		userID := strings.TrimSpace(toString(node["id"]))
		if userID == "" {
			continue
		}

		raw := string(payload)
		if act == "sign" {
			if registeredUserID != "" && registeredUserID != userID && a.wsManager != nil {
				a.wsManager.UnregisterDownstream(registeredUserID, session)
			}
			registeredUserID = userID
			if a.wsManager != nil {
				a.wsManager.RegisterDownstream(userID, session, raw)
			}
			continue
		}

		if registeredUserID == "" {
			continue
		}
		if userID != registeredUserID {
			continue
		}
		if a.wsManager != nil {
			a.wsManager.SendToUpstream(registeredUserID, raw)
		}
	}

	if registeredUserID != "" && a.wsManager != nil {
		a.wsManager.UnregisterDownstream(registeredUserID, session)
	}
	_ = session.Close()
}
