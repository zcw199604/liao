package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestDouyinCookieCloudProvider_GetCookie_FetchErrorBranch(t *testing.T) {
	cfg := minimalCookieCloudConfig()
	cfg.CacheType = "memory"
	cfg.CookieCloudBaseURL = "http://127.0.0.1:1"
	cfg.CookieCloudUUID = "u1"
	cfg.CookieCloudPassword = "p1"
	cfg.CookieCloudDomain = "douyin.com"
	cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"

	p, err := NewDouyinCookieCloudProvider(cfg, &http.Client{Timeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("NewDouyinCookieCloudProvider err=%v", err)
	}
	defer func() { _ = p.Close() }()

	if _, err := p.GetCookie(context.Background()); err == nil {
		t.Fatalf("expected fetch error")
	}
}

func TestDouyinCookieCloudProvider_GetCookie_DoubleCheckRedisHitBranch(t *testing.T) {
	uuid := "u2"
	password := "p2"
	srv := newCookieCloudServerForProviderTest(t, uuid, password, []map[string]any{
		{"name": "server", "value": "1", "domain": ".douyin.com", "path": "/"},
	})
	defer srv.Close()

	cfg := minimalCookieCloudConfig()
	cfg.CacheType = "memory"
	cfg.CookieCloudBaseURL = srv.URL
	cfg.CookieCloudUUID = uuid
	cfg.CookieCloudPassword = password
	cfg.CookieCloudDomain = "douyin.com"
	cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"

	p, err := NewDouyinCookieCloudProvider(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewDouyinCookieCloudProvider err=%v", err)
	}
	defer func() { _ = p.Close() }()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis err=%v", err)
	}
	defer mr.Close()

	p.redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	p.redisKey = "k1"
	p.redisTimeout = time.Second

	// 锁住 fetchMu，让首次 Redis 检查先 miss，然后在解锁前写入 Redis，
	// 触发“加锁后二次 Redis 命中”分支。
	p.fetchMu.Lock()
	done := make(chan struct {
		val string
		err error
	}, 1)
	go func() {
		val, getErr := p.GetCookie(context.Background())
		done <- struct {
			val string
			err error
		}{val: val, err: getErr}
	}()
	time.Sleep(40 * time.Millisecond)
	// 初次 local 检查在加锁前已结束，这里注入非字符串值以覆盖“加锁后二次检查”删除分支。
	p.local.Set(douyinCookieCloudLocalKey, 123)
	mr.Set(p.redisKey, "redis=1")
	p.fetchMu.Unlock()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("GetCookie err=%v", got.err)
		}
		if got.val != "redis=1" {
			t.Fatalf("cookie=%q", got.val)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("GetCookie timed out")
	}
}
