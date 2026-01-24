package app

import (
	"testing"
)

func TestFormatLastMessage_InlineImageWithText(t *testing.T) {
	got := formatLastMessage(CachedLastMessage{
		FromUserID: "me",
		ToUserID:   "u2",
		Content:    "喜欢吗[20260104/image.jpg]",
		Type:       "text",
		Time:       "t",
	}, "me")
	if got != "我: 喜欢吗 [图片]" {
		t.Fatalf("got %q, want %q", got, "我: 喜欢吗 [图片]")
	}
}

func TestFormatLastMessage_EmojiTokenNotFile(t *testing.T) {
	got := formatLastMessage(CachedLastMessage{
		FromUserID: "me",
		ToUserID:   "u2",
		Content:    "[doge]",
		Type:       "text",
		Time:       "t",
	}, "me")
	if got != "我: [doge]" {
		t.Fatalf("got %q, want %q", got, "我: [doge]")
	}
}

func TestInferMessageType_InlineImage(t *testing.T) {
	if got := inferMessageType("喜欢吗[20260104/image.jpg]"); got != "image" {
		t.Fatalf("got %q, want %q", got, "image")
	}
}

func TestInferMessageType_EmojiIsText(t *testing.T) {
	if got := inferMessageType("[doge]"); got != "text" {
		t.Fatalf("got %q, want %q", got, "text")
	}
}

func TestMemoryUserInfoCacheService_UserInfoCRUD(t *testing.T) {
	svc := NewMemoryUserInfoCacheService()

	svc.SaveUserInfo(CachedUserInfo{UserID: " "})
	if got := svc.GetUserInfo("u1"); got != nil {
		t.Fatalf("expected nil, got=%+v", got)
	}

	svc.SaveUserInfo(CachedUserInfo{
		UserID:   "u1",
		Nickname: "Alice",
		Gender:   "女",
		Age:      "18",
		Address:  "SZ",
	})

	info := svc.GetUserInfo("u1")
	if info == nil || info.UserID != "u1" || info.Nickname != "Alice" {
		t.Fatalf("info=%+v", info)
	}

	// 返回的是副本
	info.Nickname = "X"
	info2 := svc.GetUserInfo("u1")
	if info2 == nil || info2.Nickname != "Alice" {
		t.Fatalf("info2=%+v", info2)
	}

	// 空 userID 返回 nil
	if got := svc.GetUserInfo(" "); got != nil {
		t.Fatalf("expected nil")
	}
}

func TestMemoryUserInfoCacheService_EnrichAndBatch(t *testing.T) {
	svc := NewMemoryUserInfoCacheService()
	svc.SaveUserInfo(CachedUserInfo{
		UserID:   "u1",
		Nickname: "Alice",
		Gender:   "女",
		Age:      "18",
		Address:  "SZ",
	})

	original := map[string]any{
		"nickname": "Keep",
	}
	got := svc.EnrichUserInfo("u1", original)
	if got["nickname"] != "Keep" {
		t.Fatalf("expected keep, got=%v", got["nickname"])
	}
	if got["sex"] != "女" || got["age"] != "18" || got["address"] != "SZ" {
		t.Fatalf("enriched=%v", got)
	}

	if out := svc.BatchEnrichUserInfo(nil, "id"); len(out) != 0 {
		t.Fatalf("expected empty")
	}

	users := []map[string]any{
		{"id": "u1"},
		{"id": "  "},
		{"x": "u1"},
		nil,
		{"id": []byte("u1")},
	}
	out := svc.BatchEnrichUserInfo(users, "id")
	if out[0]["nickname"] != "Alice" {
		t.Fatalf("out[0]=%v", out[0])
	}
	if out[4]["nickname"] != "Alice" {
		t.Fatalf("out[4]=%v", out[4])
	}
}

func TestMemoryUserInfoCacheService_LastMessageAndBatch(t *testing.T) {
	svc := NewMemoryUserInfoCacheService()

	svc.SaveLastMessage(CachedLastMessage{FromUserID: "", ToUserID: "u2"})
	if got := svc.GetLastMessage("u1", "u2"); got != nil {
		t.Fatalf("expected nil")
	}

	svc.SaveLastMessage(CachedLastMessage{
		FromUserID: "u2",
		ToUserID:   "u1",
		Content:    "hello",
		Type:       "text",
		Time:       "t",
	})
	last := svc.GetLastMessage("u1", "u2")
	if last == nil || last.ConversationKey != "u1_u2" || last.Content != "hello" {
		t.Fatalf("last=%+v", last)
	}

	// 返回的是副本
	last.Content = "X"
	last2 := svc.GetLastMessage("u1", "u2")
	if last2 == nil || last2.Content != "hello" {
		t.Fatalf("last2=%+v", last2)
	}

	users := []map[string]any{
		{"id": "u2"},
		{"id": "bad"},
		{"id": "u2", "lastMsg": "keep"},
	}
	out := svc.BatchEnrichWithLastMessage(users, "u1")
	if out[0]["lastMsg"] != "hello" || out[0]["lastTime"] != "t" {
		t.Fatalf("out[0]=%v", out[0])
	}
	if _, ok := out[1]["lastMsg"]; ok {
		t.Fatalf("out[1] should not have lastMsg: %v", out[1])
	}
	if out[2]["lastMsg"] != "keep" {
		t.Fatalf("out[2]=%v", out[2])
	}
}

func TestUserInfoCacheHelpers(t *testing.T) {
	if got := generateConversationKey("", "u2"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := generateConversationKey("u2", "u1"); got != "u1_u2" {
		t.Fatalf("got=%q", got)
	}

	if got := fmtAny([]byte("x")); got != "x" {
		t.Fatalf("got=%q", got)
	}
	if got := fmtAny(1); got != "1" {
		t.Fatalf("got=%q", got)
	}
	if got := fmtAny(int64(2)); got != "2" {
		t.Fatalf("got=%q", got)
	}
	if got := fmtAny(float64(3)); got != "3" {
		t.Fatalf("got=%q", got)
	}
	if got := fmtAny(true); got != "true" {
		t.Fatalf("got=%q", got)
	}
	if got := fmtAny(false); got != "false" {
		t.Fatalf("got=%q", got)
	}
	if got := fmtAny(struct{}{}); got != "" {
		t.Fatalf("got=%q", got)
	}

	// toString: string/[]byte/其它
	if got := toString(" x "); got != " x " {
		t.Fatalf("got=%q", got)
	}
	if got := toString([]byte("x")); got != "x" {
		t.Fatalf("got=%q", got)
	}
}

func TestInferMediaKindFromBracketBody(t *testing.T) {
	if got := inferMediaKindFromBracketBody(""); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("http://x/y.jpg"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("a b.jpg"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("a.JPG"); got != "image" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("v.mp4?x=1#y"); got != "video" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("a.mp3"); got != "audio" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("a.123"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("a.ab12"); got != "file" {
		t.Fatalf("got=%q", got)
	}
	if got := inferMediaKindFromBracketBody("a.abcdefghijklmnop"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := looksLikeFileExt("ab12"); !got {
		t.Fatalf("expected true")
	}
	if got := looksLikeFileExt("AB"); got {
		t.Fatalf("expected false")
	}
}

func TestFormatTimeAndTruncate(t *testing.T) {
	if got := formatTime(" "); got != "刚刚" {
		t.Fatalf("got=%q", got)
	}
	if got := formatTime("t"); got != "t" {
		t.Fatalf("got=%q", got)
	}
	if got := truncateRunes("abc", 0); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := truncateRunes("你好世界", 3); got != "你好世..." {
		t.Fatalf("got=%q", got)
	}
}

func TestMemoryUserInfoCacheService_BatchGetUserInfo_EmptyAndNilReceiver(t *testing.T) {
	var nilSvc *MemoryUserInfoCacheService
	got := nilSvc.batchGetUserInfo([]string{"u1"})
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}

	svc := NewMemoryUserInfoCacheService()
	svc.SaveUserInfo(CachedUserInfo{UserID: "u1", Nickname: "n1"})
	got = svc.batchGetUserInfo([]string{"u1", " ", "missing"})
	if got["u1"].Nickname != "n1" {
		t.Fatalf("got=%v", got)
	}
}

func TestMemoryUserInfoCacheService_BatchGetLastMessages_EmptyAndNilReceiver(t *testing.T) {
	var nilSvc *MemoryUserInfoCacheService
	got := nilSvc.batchGetLastMessages([]string{"u1_u2"})
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}

	svc := NewMemoryUserInfoCacheService()
	svc.SaveLastMessage(CachedLastMessage{FromUserID: "u1", ToUserID: "u2", Content: "hi", Type: "text", Time: "t"})
	got = svc.batchGetLastMessages([]string{"u1_u2", " "})
	if got["u1_u2"].Content != "hi" {
		t.Fatalf("got=%v", got)
	}
}

func TestMemoryUserInfoCacheService_EnrichUserInfo_MissingUser_NoChange(t *testing.T) {
	svc := NewMemoryUserInfoCacheService()
	original := map[string]any{"nickname": "keep"}
	out := svc.EnrichUserInfo("missing", original)
	if out["nickname"] != "keep" {
		t.Fatalf("out=%v", out)
	}
}

func TestMemoryUserInfoCacheService_GetLastMessage_InvalidKey(t *testing.T) {
	svc := NewMemoryUserInfoCacheService()
	if got := svc.GetLastMessage(" ", "u2"); got != nil {
		t.Fatalf("expected nil")
	}
}

func TestMemoryUserInfoCacheService_BatchEnrichWithLastMessage_NilListAndEmptyUser(t *testing.T) {
	svc := NewMemoryUserInfoCacheService()
	if out := svc.BatchEnrichWithLastMessage(nil, "u1"); len(out) != 0 {
		t.Fatalf("out=%v", out)
	}

	users := []map[string]any{
		{"x": "u2"},
		{"id": nil},
		nil,
	}
	out := svc.BatchEnrichWithLastMessage(users, "u1")
	if len(out) != 3 {
		t.Fatalf("out=%v", out)
	}
}

func TestExtractUserID_PutIfAbsent_EdgeCases(t *testing.T) {
	if got := extractUserID(nil); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := extractUserID(map[string]any{"UserID": "u1"}); got != "u1" {
		t.Fatalf("got=%q", got)
	}
	if got := extractUserID(map[string]any{"x": "u1"}); got != "" {
		t.Fatalf("got=%q", got)
	}

	putIfAbsent(nil, "k", "v")
	m := map[string]any{}
	putIfAbsent(m, "k", nil)
	if _, ok := m["k"]; ok {
		t.Fatalf("m=%v", m)
	}
}

func TestFormatLastMessage_EdgeCases(t *testing.T) {
	if got := formatLastMessage(CachedLastMessage{FromUserID: "me", Content: ""}, "me"); got != "我: [消息]" {
		t.Fatalf("got=%q", got)
	}
	if got := formatLastMessage(CachedLastMessage{FromUserID: "me", Content: "   "}, "me"); got != "我: [消息]" {
		t.Fatalf("got=%q", got)
	}
	if got := formatLastMessage(CachedLastMessage{FromUserID: "me", Content: "hi[20260104/image.jpg"}, "me"); got != "我: hi[20260104/image.jpg" {
		t.Fatalf("got=%q", got)
	}
	if got := formatLastMessage(CachedLastMessage{FromUserID: "me", Content: "[a.mp4]"}, "me"); got != "我: [视频]" {
		t.Fatalf("got=%q", got)
	}
	if got := formatLastMessage(CachedLastMessage{FromUserID: "me", Content: "[a.mp3]"}, "me"); got != "我: [音频]" {
		t.Fatalf("got=%q", got)
	}
	if got := formatLastMessage(CachedLastMessage{FromUserID: "me", Content: "[a.ab12]"}, "me"); got != "我: [文件]" {
		t.Fatalf("got=%q", got)
	}
}

func TestInferMediaKindFromBracketBody_HashAndLooksLikeFileExt_Empty(t *testing.T) {
	if got := inferMediaKindFromBracketBody("v.mp4#t=1"); got != "video" {
		t.Fatalf("got=%q", got)
	}
	if looksLikeFileExt("") {
		t.Fatalf("expected false")
	}
}
