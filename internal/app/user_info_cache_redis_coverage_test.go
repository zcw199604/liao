package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestNewRedisUserInfoCacheService_Defaults(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService(
		"",
		host,
		port,
		"",
		0,
		"", // default keyPrefix
		"", // default lastMessagePrefix
		0,  // default expireDays
		0,  // no flush loop
		0,  // default localTTL
		0,  // default timeout
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if svc.keyPrefix != "user:info:" {
		t.Fatalf("keyPrefix=%q", svc.keyPrefix)
	}
	if svc.lastMessagePrefix != "user:lastmsg:" {
		t.Fatalf("lastMessagePrefix=%q", svc.lastMessagePrefix)
	}
}

func TestNewRedisUserInfoCacheService_BuildOptionsError(t *testing.T) {
	if _, err := NewRedisUserInfoCacheService(
		"redis://%zz",
		"",
		0,
		"",
		0,
		"user:info:",
		"user:lastmsg:",
		7,
		0,
		3600,
		1,
	); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewRedisUserInfoCacheService_PingFailure(t *testing.T) {
	if _, err := NewRedisUserInfoCacheService(
		"",
		"127.0.0.1",
		0,
		"",
		0,
		"user:info:",
		"user:lastmsg:",
		7,
		0,
		3600,
		1,
	); err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildRedisOptions_CoversBranches(t *testing.T) {
	mr := miniredis.RunT(t)

	opts, err := buildRedisOptions("", "redis://"+mr.Addr(), 0, "", 0, time.Second)
	if err != nil || !strings.Contains(opts.Addr, ":") {
		t.Fatalf("opts=%v err=%v", opts, err)
	}

	opts, err = buildRedisOptions("", "", 6379, "", 0, time.Second)
	if err != nil || opts.Addr != "localhost:6379" {
		t.Fatalf("opts=%v err=%v", opts, err)
	}

	opts, err = buildRedisOptions("redis://"+mr.Addr(), "", 0, "", 0, 0)
	if err != nil || opts.DialTimeout <= 0 {
		t.Fatalf("opts=%v err=%v", opts, err)
	}
}

func TestRedisUserInfoCacheService_NilSafety(t *testing.T) {
	var svc *RedisUserInfoCacheService
	_ = svc.flushPending(nil)
	svc.flushOnce()
	if err := svc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	svc.SaveUserInfo(CachedUserInfo{UserID: "u1"})
	if got := svc.GetUserInfo("u1"); got != nil {
		t.Fatalf("expected nil")
	}
	svc.SaveLastMessage(CachedLastMessage{FromUserID: "u1", ToUserID: "u2"})
	if got := svc.GetLastMessage("u1", "u2"); got != nil {
		t.Fatalf("expected nil")
	}
}

func TestRedisUserInfoCacheService_flushOnce_DefaultTimeout(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	svc := &RedisUserInfoCacheService{
		client:  client,
		timeout: 0,
	}
	svc.flushOnce()
}

func TestRedisUserInfoCacheService_flushLoop_TickerFlushes(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService(
		"",
		host,
		port,
		"",
		0,
		"user:info:",
		"user:lastmsg:",
		7,
		1,    // ticker flush
		3600, // local TTL
		1,    // timeout
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	svc.SaveUserInfo(CachedUserInfo{UserID: "u1", Nickname: "n1"})

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := rdb.Get(ctx, "user:info:u1").Result(); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected user:info:u1 flushed by ticker")
}

func TestRedisUserInfoCacheService_SaveAndGetUserInfo_EdgeCases(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService("", host, port, "", 0, "user:info:", "user:lastmsg:", 7, 0, 3600, 1)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	// empty userId
	svc.SaveUserInfo(CachedUserInfo{UserID: " "})
	if got := svc.GetUserInfo(" "); got != nil {
		t.Fatalf("expected nil")
	}

	// local cache wrong type -> fall back to redis.
	svc.local.Set("u1", "bad")
	raw, _ := json.Marshal(CachedUserInfo{UserID: "u1", Nickname: "n1"})
	if err := svc.client.Set(context.Background(), "user:info:u1", raw, 0).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := svc.GetUserInfo("u1"); got == nil || got.Nickname != "n1" {
		t.Fatalf("got=%v", got)
	}

	// redis GET error -> nil
	if got := svc.GetUserInfo("missing"); got != nil {
		t.Fatalf("expected nil")
	}

	// invalid JSON -> nil
	if err := svc.client.Set(context.Background(), "user:info:bad", "{", 0).Err(); err != nil {
		t.Fatalf("set bad: %v", err)
	}
	if got := svc.GetUserInfo("bad"); got != nil {
		t.Fatalf("expected nil")
	}

	m := map[string]any{"nickname": "keep"}
	out := svc.EnrichUserInfo("missing", m)
	if out["nickname"] != "keep" {
		t.Fatalf("out=%v", out)
	}
}

func TestRedisUserInfoCacheService_BatchEnrichUserInfo_EdgeCases(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService("", host, port, "", 0, "user:info:", "user:lastmsg:", 7, 0, 3600, 1)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if out := svc.BatchEnrichUserInfo(nil, "id"); out != nil {
		t.Fatalf("expected nil")
	}

	users := []map[string]any{
		nil,
		{"id": " "},
		{"x": "u1"},
	}
	out := svc.BatchEnrichUserInfo(users, "id")
	if len(out) != 3 {
		t.Fatalf("out=%v", out)
	}

	svc.SaveUserInfo(CachedUserInfo{UserID: "u1", Nickname: "n1"})
	users = []map[string]any{
		{"id": "u1"},
		{"id": "missing"},
		{"id": " "},
	}
	out = svc.BatchEnrichUserInfo(users, "id")
	if out[0]["nickname"] != "n1" {
		t.Fatalf("out[0]=%v", out[0])
	}
	if _, ok := out[1]["nickname"]; ok {
		t.Fatalf("out[1]=%v", out[1])
	}
}

func TestRedisUserInfoCacheService_multiGetUserInfo_CoversBranches(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService("", host, port, "", 0, "user:info:", "user:lastmsg:", 7, 0, 3600, 1)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	// local hit: no MGET
	svc.SaveUserInfo(CachedUserInfo{UserID: "u0", Nickname: "n0"})
	if got := svc.multiGetUserInfo([]string{"u0"}); got["u0"].Nickname != "n0" {
		t.Fatalf("got=%v", got)
	}

	// prepare MGET values: valid / invalid / empty / missing.
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	raw, _ := json.Marshal(CachedUserInfo{UserID: "u1", Nickname: "n1"})
	if err := rdb.Set(context.Background(), "user:info:u1", raw, 0).Err(); err != nil {
		t.Fatalf("set u1: %v", err)
	}
	if err := rdb.Set(context.Background(), "user:info:u2", "{", 0).Err(); err != nil {
		t.Fatalf("set u2: %v", err)
	}
	if err := rdb.Set(context.Background(), "user:info:u3", "", 0).Err(); err != nil {
		t.Fatalf("set u3: %v", err)
	}

	got := svc.multiGetUserInfo([]string{"u1", "u2", "u3", "missing"})
	if got["u1"].Nickname != "n1" || got["u2"].UserID != "" {
		t.Fatalf("got=%v", got)
	}

	// MGET error branch: client closed.
	_ = svc.client.Close()
	_ = svc.multiGetUserInfo([]string{"u1", "missing"})
}

func TestRedisUserInfoCacheService_SaveAndGetLastMessage_EdgeCases(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService("", host, port, "", 0, "user:info:", "user:lastmsg:", 7, 0, 3600, 1)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	// empty key after generation -> return
	svc.SaveLastMessage(CachedLastMessage{})

	if got := svc.GetLastMessage(" ", "u2"); got != nil {
		t.Fatalf("expected nil")
	}

	svc.SaveLastMessage(CachedLastMessage{FromUserID: "u1", ToUserID: "u2", Content: "hi", Type: "text", Time: "t"})
	if got := svc.GetLastMessage("u1", "u2"); got == nil || got.Content != "hi" {
		t.Fatalf("got=%v", got)
	}

	// redis GET error
	if got := svc.GetLastMessage("u1", "missing"); got != nil {
		t.Fatalf("expected nil")
	}

	// invalid JSON
	if err := svc.client.Set(context.Background(), "user:lastmsg:u1_u3", "{", 0).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := svc.GetLastMessage("u1", "u3"); got != nil {
		t.Fatalf("expected nil")
	}
}

func TestRedisUserInfoCacheService_BatchEnrichWithLastMessage_EdgeCases(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService("", host, port, "", 0, "user:info:", "user:lastmsg:", 7, 0, 3600, 1)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if out := svc.BatchEnrichWithLastMessage(nil, "u1"); out != nil {
		t.Fatalf("expected nil")
	}

	users := []map[string]any{
		{"id": " "},
		{"x": "u2"},
		nil,
	}
	out := svc.BatchEnrichWithLastMessage(users, "u1")
	if len(out) != 3 {
		t.Fatalf("out=%v", out)
	}

	svc.SaveLastMessage(CachedLastMessage{FromUserID: "u1", ToUserID: "u2", Content: "hi", Type: "text", Time: "t"})
	users = []map[string]any{
		{"id": "u2"},
		{"id": "missing"},
		nil,
	}
	out = svc.BatchEnrichWithLastMessage(users, "u1")
	if out[0]["lastMsg"] == "" || out[0]["lastTime"] != "t" {
		t.Fatalf("out[0]=%v", out[0])
	}
	if _, ok := out[1]["lastMsg"]; ok {
		t.Fatalf("out[1]=%v", out[1])
	}
}

func TestRedisUserInfoCacheService_multiGetLastMessages_CoversBranches(t *testing.T) {
	mr := miniredis.RunT(t)
	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("splitHostPort: %v", err)
	}

	svc, err := NewRedisUserInfoCacheService("", host, port, "", 0, "user:info:", "user:lastmsg:", 7, 0, 3600, 1)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	// local hit: no MGET
	svc.SaveLastMessage(CachedLastMessage{ConversationKey: "u1_u2", FromUserID: "u1", ToUserID: "u2", Content: "hi", Type: "text", Time: "t"})
	if got := svc.multiGetLastMessages([]string{"u1_u2"}); got["u1_u2"].Content != "hi" {
		t.Fatalf("got=%v", got)
	}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	raw, _ := json.Marshal(CachedLastMessage{ConversationKey: "u1_u5", FromUserID: "u1", ToUserID: "u5", Content: "hi2", Type: "text", Time: "t2"})
	if err := rdb.Set(context.Background(), "user:lastmsg:u1_u5", raw, 0).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := rdb.Set(context.Background(), "user:lastmsg:u1_u3", "{", 0).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := rdb.Set(context.Background(), "user:lastmsg:u1_u4", "", 0).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := svc.multiGetLastMessages([]string{"u1_u5", "u1_u3", "u1_u4", "missing"})
	if got["u1_u5"].Time != "t2" || got["u1_u3"].Time != "" {
		t.Fatalf("got=%v", got)
	}

	// MGET error branch: client closed.
	_ = svc.client.Close()
	_ = svc.multiGetLastMessages([]string{"u1_u2", "missing"})
}
