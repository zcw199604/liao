package app

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserListCache_UpdateLastMessage_UpdatesBothSides(t *testing.T) {
	cache := newUserListCache(10, time.Hour)

	meList := []map[string]any{{"id": "u2"}}
	meBody, err := json.Marshal(meList)
	if err != nil {
		t.Fatalf("marshal me: %v", err)
	}
	cache.SetHistory("me", meList, meBody)

	u2List := []map[string]any{{"id": "me"}}
	u2Body, err := json.Marshal(u2List)
	if err != nil {
		t.Fatalf("marshal u2: %v", err)
	}
	cache.SetHistory("u2", u2List, u2Body)

	cache.UpdateLastMessage(CachedLastMessage{
		FromUserID: "me",
		ToUserID:   "u2",
		Content:    "hello",
		Type:       "text",
		Time:       "t1",
	})

	raw, _, ok := cache.GetHistory("me")
	if !ok {
		t.Fatalf("expected cache hit for me")
	}
	var gotMe []map[string]any
	if err := json.Unmarshal(raw, &gotMe); err != nil {
		t.Fatalf("unmarshal me: %v", err)
	}
	if msg, _ := gotMe[0]["lastMsg"].(string); msg != "我: hello" {
		t.Fatalf("me.lastMsg=%q, want %q", msg, "我: hello")
	}

	raw, _, ok = cache.GetHistory("u2")
	if !ok {
		t.Fatalf("expected cache hit for u2")
	}
	var gotU2 []map[string]any
	if err := json.Unmarshal(raw, &gotU2); err != nil {
		t.Fatalf("unmarshal u2: %v", err)
	}
	if msg, _ := gotU2[0]["lastMsg"].(string); msg != "hello" {
		t.Fatalf("u2.lastMsg=%q, want %q", msg, "hello")
	}
}
