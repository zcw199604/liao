package app

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// handleWanfengWebSocket 处理选择「晚风线路」的下游连接。
// 前端协议保持不变（act 帧），由 WanfengManager 负责与对方站点做协议翻译。
func (a *App) handleWanfengWebSocket(w http.ResponseWriter, r *http.Request) {
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
			if registeredUserID != "" && registeredUserID != userID {
				a.wanfeng.UnregisterDownstream(registeredUserID, session)
			}
			registeredUserID = userID
			a.wanfeng.RegisterDownstream(userID, session, raw)
			continue
		}

		if registeredUserID == "" || userID != registeredUserID {
			continue
		}
		a.wanfeng.SendToUpstream(registeredUserID, raw)
	}

	if registeredUserID != "" {
		a.wanfeng.UnregisterDownstream(registeredUserID, session)
	}
	_ = session.Close()
}
