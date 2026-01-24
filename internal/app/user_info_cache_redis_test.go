package app

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisUserInfoCacheService_GetUserInfoAndEnrich(t *testing.T) {
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
		0, // 立即写入
		3600,
		15,
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	svc.SaveUserInfo(CachedUserInfo{UserID: "u1", Nickname: "Alice", Gender: "女"})
	if got := svc.GetUserInfo("u1"); got == nil || got.Nickname != "Alice" {
		t.Fatalf("got=%+v", got)
	}

	// 覆盖：local miss -> redis hit
	svc2, err := NewRedisUserInfoCacheService(
		"",
		host,
		port,
		"",
		0,
		"user:info:",
		"user:lastmsg:",
		7,
		0,
		3600,
		15,
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc2.Close() })

	if got := svc2.GetUserInfo("u1"); got == nil || got.Nickname != "Alice" {
		t.Fatalf("got=%+v", got)
	}

	// 无效 JSON 不应 panic
	ctx := context.Background()
	if err := svc2.client.Set(ctx, "user:info:bad", "{", 0).Err(); err != nil {
		t.Fatalf("set bad: %v", err)
	}
	if got := svc2.GetUserInfo("bad"); got != nil {
		t.Fatalf("expected nil")
	}

	m := map[string]any{"nickname": "keep"}
	out := svc2.EnrichUserInfo("u1", m)
	if out["nickname"] != "keep" || out["sex"] != "女" {
		t.Fatalf("out=%v", out)
	}
}

func TestRedisUserInfoCacheService_BatchEnrichUserInfo(t *testing.T) {
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
		0,
		3600,
		15,
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	svc.SaveUserInfo(CachedUserInfo{UserID: "u1", Nickname: "A"})
	svc.SaveUserInfo(CachedUserInfo{UserID: "u2", Nickname: "B"})

	users := []map[string]any{
		{"id": "u1"},
		{"id": "u2"},
		{"id": "u2"}, // duplicate
		{"x": "u1"},
		nil,
	}
	out := svc.BatchEnrichUserInfo(users, "id")
	if out[0]["nickname"] != "A" || out[1]["nickname"] != "B" {
		t.Fatalf("out=%v", out)
	}

	// 覆盖 batchGetUserInfo 的转发
	mm := svc.batchGetUserInfo([]string{"u1", "u2", "missing"})
	if mm["u1"].Nickname != "A" || mm["u2"].Nickname != "B" {
		t.Fatalf("mm=%v", mm)
	}
}

func TestRedisUserInfoCacheService_LastMessageAndBatch(t *testing.T) {
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
		0,
		3600,
		15,
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	svc.SaveLastMessage(CachedLastMessage{FromUserID: "u1", ToUserID: "u2", Content: "hi", Type: "text", Time: "t"})
	if got := svc.GetLastMessage("u1", "u2"); got == nil || got.Content != "hi" {
		t.Fatalf("got=%+v", got)
	}

	// 覆盖：local miss -> redis hit
	svc2, err := NewRedisUserInfoCacheService(
		"",
		host,
		port,
		"",
		0,
		"user:info:",
		"user:lastmsg:",
		7,
		0,
		3600,
		15,
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc2.Close() })

	if got := svc2.GetLastMessage("u1", "u2"); got == nil || got.Content != "hi" {
		t.Fatalf("got=%+v", got)
	}

	users := []map[string]any{
		{"id": "u2"},
		{"id": "missing"},
	}
	out := svc2.BatchEnrichWithLastMessage(users, "u1")
	if out[0]["lastMsg"] == "" || out[0]["lastTime"] != "t" {
		t.Fatalf("out[0]=%v", out[0])
	}

	// 覆盖 batchGetLastMessages 的转发
	msgs := svc2.batchGetLastMessages([]string{"u1_u2"})
	if msgs["u1_u2"].Content != "hi" {
		t.Fatalf("msgs=%v", msgs)
	}

	// 边界：空 myUserID 或 nil list
	_ = svc2.BatchEnrichWithLastMessage(nil, "u1")
	_ = svc2.BatchEnrichWithLastMessage(users, " ")
	if got := svc2.GetLastMessage(" ", "u2"); got != nil {
		t.Fatalf("expected nil")
	}
}

func TestBuildRedisOptions_ParseURLFailure_DoesNotLeak(t *testing.T) {
	if _, err := buildRedisOptions("redis://%zz", "", 0, "", 0, 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRedisUserInfoCacheService_MGetWrongType_IsBestEffort(t *testing.T) {
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
		0,
		3600,
		15,
	)
	if err != nil {
		t.Fatalf("NewRedisUserInfoCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	// 在 Redis 中把 key 写成 list，MGET 会触发 WRONGTYPE（multiGetUserInfo 应直接返回已知结果）
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	ctx := context.Background()
	if err := rdb.RPush(ctx, "user:info:u1", "x").Err(); err != nil {
		t.Fatalf("rpush: %v", err)
	}

	got := svc.multiGetUserInfo([]string{"u1"})
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}
