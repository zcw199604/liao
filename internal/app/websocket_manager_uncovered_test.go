package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpstreamWebSocketClient_OnMessage_LongLogBranches(t *testing.T) {
	t.Run("long raw message truncation", func(t *testing.T) {
		c := &UpstreamWebSocketClient{userID: "u1"}
		c.onMessage(strings.Repeat("x", 600))
	})

	t.Run("chat message long content truncation", func(t *testing.T) {
		c := &UpstreamWebSocketClient{userID: "u1"}
		node := map[string]any{
			"code": 7,
			"fromuser": map[string]any{
				"id":      "from",
				"Tid":     "tid-1",
				"type":    "text",
				"time":    "2026-03-02 10:00:00",
				"content": strings.Repeat("c", 120),
			},
			"touser": map[string]any{
				"id": "to",
			},
		}
		b, _ := json.Marshal(node)
		c.onMessage(string(b))
	})
}

