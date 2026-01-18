package app

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisUserInfoCacheService_BatchFlush_DedupLastMessage(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("split addr: %v", err)
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
		3600, // 避免自动 ticker 干扰，手动 flush 验证即可
		3600,
	)
	if err != nil {
		t.Fatalf("new redis cache: %v", err)
	}
	defer svc.Close()

	// 同一会话 1 分钟内多次写入，应只保留最后一次（以降低写入频率）。
	svc.SaveLastMessage(CachedLastMessage{
		FromUserID: "u1",
		ToUserID:   "u2",
		Content:    "a",
		Type:       "text",
		Time:       "t1",
	})
	svc.SaveLastMessage(CachedLastMessage{
		FromUserID: "u1",
		ToUserID:   "u2",
		Content:    "b",
		Type:       "text",
		Time:       "t2",
	})

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	ctx := context.Background()
	key := "user:lastmsg:u1_u2"

	// 未 flush 前不应写入 Redis（只写入本地 LRU）。
	if _, err := rdb.Get(ctx, key).Result(); err != redis.Nil {
		t.Fatalf("expected redis key missing before flush, got err=%v", err)
	}

	if err := svc.flushPending(ctx); err != nil {
		t.Fatalf("flush pending: %v", err)
	}

	raw, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		t.Fatalf("get after flush: %v", err)
	}
	var msg CachedLastMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Content != "b" {
		t.Fatalf("content=%q, want %q", msg.Content, "b")
	}
}

func TestRedisUserInfoCacheService_CloseFlushesPendingWrites(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("split addr: %v", err)
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
		3600, // 避免自动 ticker 干扰
		3600,
	)
	if err != nil {
		t.Fatalf("new redis cache: %v", err)
	}

	svc.SaveUserInfo(CachedUserInfo{
		UserID:   "u1",
		Nickname: "n1",
	})

	// Close 需要尽量 flush 最后一批待写入数据。
	_ = svc.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	ctx := context.Background()
	if _, err := rdb.Get(ctx, "user:info:u1").Result(); err != nil {
		t.Fatalf("expected key after close flush, err=%v", err)
	}
}

func splitHostPort(addr string) (host string, port int, err error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

func TestSplitHostPort(t *testing.T) {
	host, port, err := splitHostPort("127.0.0.1:6379")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if host != "127.0.0.1" {
		t.Fatalf("host=%q", host)
	}
	if port != 6379 {
		t.Fatalf("port=%d", port)
	}
}

func TestRedisUserInfoCacheService_Defaults_NoPanicOnImmediateMode(t *testing.T) {
	// 仅覆盖边界：flushIntervalSeconds<=0 时走立即写入分支，不应 panic。
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	host, port, err := splitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("split addr: %v", err)
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
	)
	if err != nil {
		t.Fatalf("new redis cache: %v", err)
	}
	defer svc.Close()

	svc.SaveLastMessage(CachedLastMessage{
		FromUserID: "u1",
		ToUserID:   "u2",
		Content:    "x",
		Type:       "text",
		Time:       time.Now().Format(time.RFC3339),
	})
}
