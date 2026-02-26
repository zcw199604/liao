package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newCookieCloudServerForProviderTest(t *testing.T, uuid, password string, cookies []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get/"+uuid {
			http.NotFound(w, r)
			return
		}
		plain := map[string]any{
			"cookie_data": map[string]any{
				".douyin.com": cookies,
			},
			"local_storage_data": map[string]any{},
			"update_time":        "2026-02-01T00:00:00Z",
		}
		plainBytes, _ := json.Marshal(plain)
		encrypted := encryptCookieCloudFixed(uuid, password, plainBytes)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"encrypted":   encrypted,
			"crypto_type": "aes-128-cbc-fixed",
		})
	}))
}

func TestNewDouyinCookieCloudProvider_MoreBranches(t *testing.T) {
	t.Run("cookiecloud client init error", func(t *testing.T) {
		cfg := minimalCookieCloudConfig()
		cfg.CacheType = "memory"
		cfg.CookieCloudBaseURL = "bad://"
		cfg.CookieCloudUUID = "u"
		cfg.CookieCloudPassword = "p"
		if _, err := NewDouyinCookieCloudProvider(cfg, &http.Client{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("redis timeout default when <=0", func(t *testing.T) {
		uuid := "u1"
		password := "p1"
		srv := newCookieCloudServerForProviderTest(t, uuid, password, []map[string]any{
			{"name": "a", "value": "1", "domain": ".douyin.com", "path": "/"},
		})
		defer srv.Close()

		cfg := minimalCookieCloudConfig()
		cfg.CacheType = "memory"
		cfg.RedisTimeoutSeconds = 0
		cfg.CookieCloudBaseURL = srv.URL
		cfg.CookieCloudUUID = uuid
		cfg.CookieCloudPassword = password
		cfg.CookieCloudDomain = "douyin.com"
		cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"

		p, err := NewDouyinCookieCloudProvider(cfg, srv.Client())
		if err != nil {
			t.Fatalf("NewDouyinCookieCloudProvider: %v", err)
		}
		defer func() { _ = p.Close() }()
		if p.redisTimeout != 15*time.Second {
			t.Fatalf("redisTimeout=%v", p.redisTimeout)
		}
	})

	t.Run("redis options parse error", func(t *testing.T) {
		uuid := "u1"
		password := "p1"
		srv := newCookieCloudServerForProviderTest(t, uuid, password, []map[string]any{
			{"name": "a", "value": "1", "domain": ".douyin.com", "path": "/"},
		})
		defer srv.Close()

		cfg := minimalCookieCloudConfig()
		cfg.CacheType = "redis"
		cfg.RedisURL = "://bad-redis-url"
		cfg.CookieCloudBaseURL = srv.URL
		cfg.CookieCloudUUID = uuid
		cfg.CookieCloudPassword = password
		cfg.CookieCloudDomain = "douyin.com"
		cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"
		if _, err := NewDouyinCookieCloudProvider(cfg, srv.Client()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("redis ping failure", func(t *testing.T) {
		uuid := "u1"
		password := "p1"
		srv := newCookieCloudServerForProviderTest(t, uuid, password, []map[string]any{
			{"name": "a", "value": "1", "domain": ".douyin.com", "path": "/"},
		})
		defer srv.Close()

		cfg := minimalCookieCloudConfig()
		cfg.CacheType = "redis"
		cfg.RedisURL = ""
		cfg.RedisHost = "127.0.0.1"
		cfg.RedisPort = 1
		cfg.RedisTimeoutSeconds = 1
		cfg.CookieCloudBaseURL = srv.URL
		cfg.CookieCloudUUID = uuid
		cfg.CookieCloudPassword = password
		cfg.CookieCloudDomain = "douyin.com"
		cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"
		if _, err := NewDouyinCookieCloudProvider(cfg, srv.Client()); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestDouyinCookieCloudProvider_GetCookie_MoreBranches(t *testing.T) {
	t.Run("nil provider", func(t *testing.T) {
		if _, err := (*DouyinCookieCloudProvider)(nil).GetCookie(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ctx nil + local non-string + redis read/write warnings", func(t *testing.T) {
		uuid := "u1"
		password := "p1"
		srv := newCookieCloudServerForProviderTest(t, uuid, password, []map[string]any{
			{"name": "a", "value": "1", "domain": ".douyin.com", "path": "/"},
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
			t.Fatalf("NewDouyinCookieCloudProvider: %v", err)
		}
		defer func() { _ = p.Close() }()

		// trigger local non-string delete path
		p.local.Set(douyinCookieCloudLocalKey, 123)

		// trigger redis read/write error branches: closed client returns non-redis.Nil errors
		rc := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 100 * time.Millisecond})
		_ = rc.Close()
		p.redis = rc

		got, err := p.GetCookie(nil)
		if err != nil {
			t.Fatalf("GetCookie: %v", err)
		}
		if got != "a=1" {
			t.Fatalf("cookie=%q", got)
		}
	})

	t.Run("cookiecloud returns empty cookie string", func(t *testing.T) {
		uuid := "u1"
		password := "p1"
		srv := newCookieCloudServerForProviderTest(t, uuid, password, []map[string]any{
			// cookie item exists but name empty -> BuildCookieHeader returns ""
			{"name": "", "value": "1", "domain": ".douyin.com", "path": "/"},
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
			t.Fatalf("NewDouyinCookieCloudProvider: %v", err)
		}
		defer func() { _ = p.Close() }()

		_, err = p.GetCookie(context.Background())
		if err == nil || !strings.Contains(err.Error(), "返回空 Cookie") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestDouyinCookieCloudProvider_RedisGetSet_NilBranches(t *testing.T) {
	p := &DouyinCookieCloudProvider{}
	if got, err := p.redisGet(context.Background()); err != redis.Nil || got != "" {
		t.Fatalf("redisGet got=%q err=%v", got, err)
	}
	if err := p.redisSet(context.Background(), "x"); err != nil {
		t.Fatalf("redisSet err=%v", err)
	}
}

