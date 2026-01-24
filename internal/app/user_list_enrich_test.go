package app

import "testing"

func TestEnrichUserListInPlace_EarlyReturn(t *testing.T) {
	if a, b := enrichUserListInPlace(nil, nil, "id", "u1"); a != 0 || b != 0 {
		t.Fatalf("got=%d/%d, want 0/0", a, b)
	}
	if a, b := enrichUserListInPlace(NewMemoryUserInfoCacheService(), []map[string]any{}, "id", "u1"); a != 0 || b != 0 {
		t.Fatalf("got=%d/%d, want 0/0", a, b)
	}
}

func TestEnrichUserListInPlace_Success(t *testing.T) {
	cache := NewMemoryUserInfoCacheService()
	cache.SaveUserInfo(CachedUserInfo{UserID: "u2", Nickname: "n2", Gender: "男", Age: "10", Address: "a2"})
	cache.SaveUserInfo(CachedUserInfo{UserID: "u3", Nickname: "n3"})
	cache.SaveLastMessage(CachedLastMessage{FromUserID: "u1", ToUserID: "u2", Content: "hi", Type: "text", Time: "2026-01-01T00:00:00Z"})

	users := []map[string]any{
		{"id": "u2"},
		{"id": "u3", "nickname": "keep"},
		nil,
		{"id": " "},
	}

	enrichUserListInPlace(cache, users, "id", "u1")

	if users[0]["nickname"] != "n2" || users[0]["sex"] != "男" {
		t.Fatalf("user[0]=%v", users[0])
	}
	if users[0]["lastMsg"] == "" || users[0]["lastTime"] == "" {
		t.Fatalf("expected lastMsg/lastTime, user[0]=%v", users[0])
	}

	// putIfAbsent should not override existing nickname.
	if users[1]["nickname"] != "keep" {
		t.Fatalf("user[1]=%v", users[1])
	}
}

func TestCollectUserIDs(t *testing.T) {
	if got := collectUserIDs(nil, "id"); got != nil {
		t.Fatalf("got=%v, want nil", got)
	}
	if got := collectUserIDs([]map[string]any{}, "id"); got != nil {
		t.Fatalf("got=%v, want nil", got)
	}

	users := []map[string]any{
		nil,
		{"id": nil},
		{"x": "u1"},
		{"id": " u1 "},
		{"id": "u1"},
		{"id": " "},
	}
	got := collectUserIDs(users, "id")
	if len(got) != 1 || got[0] != "u1" {
		t.Fatalf("got=%v", got)
	}
}

func TestCollectConversationKeys(t *testing.T) {
	if got := collectConversationKeys(nil, "u1"); got != nil {
		t.Fatalf("got=%v, want nil", got)
	}

	users := []map[string]any{
		{"id": "u2"},
		{"userId": "u2"},
		{"id": " "},
		nil,
	}
	if got := collectConversationKeys(users, " "); got != nil {
		t.Fatalf("got=%v, want nil", got)
	}

	got := collectConversationKeys(users, "u1")
	if len(got) != 1 || got[0] != "u1_u2" {
		t.Fatalf("got=%v", got)
	}
}

func TestApplyUserInfo(t *testing.T) {
	users := []map[string]any{
		{"id": "u2"},
		{"id": "u3", "nickname": "keep"},
		{"x": "u2"},
		{"id": nil},
		nil,
		{"id": "missing"},
	}
	infoMap := map[string]CachedUserInfo{
		"u2": {UserID: "u2", Nickname: "n2", Gender: "女"},
		"u3": {UserID: "u3", Nickname: "n3", Gender: "男"},
	}
	applyUserInfo(users, "id", infoMap)

	if users[0]["nickname"] != "n2" || users[0]["sex"] != "女" {
		t.Fatalf("user[0]=%v", users[0])
	}
	if users[1]["nickname"] != "keep" {
		t.Fatalf("user[1]=%v", users[1])
	}

	applyUserInfo(users, "id", nil)
}

func TestApplyLastMessage(t *testing.T) {
	users := []map[string]any{{"id": "u2"}, {"id": "u3"}, nil}
	msgMap := map[string]CachedLastMessage{
		"u1_u2": {FromUserID: "u1", ToUserID: "u2", Content: "hi", Type: "text", Time: "2026-01-01T00:00:00Z"},
	}

	applyLastMessage(users, " ", msgMap)
	if _, ok := users[0]["lastMsg"]; ok {
		t.Fatalf("expected no change when myUserID empty")
	}

	applyLastMessage(users, "u1", nil)
	if _, ok := users[0]["lastMsg"]; ok {
		t.Fatalf("expected no change when messageMap empty")
	}

	applyLastMessage(users, "u1", msgMap)
	if users[0]["lastMsg"] == "" || users[0]["lastTime"] == "" || users[1]["lastMsg"] != nil {
		t.Fatalf("users=%v", users)
	}
}
