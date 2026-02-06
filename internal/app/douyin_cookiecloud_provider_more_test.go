package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"liao/internal/config"
)

func TestNewDouyinCookieCloudProvider_ValidationAndDefaults(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{name: "missing baseURL", cfg: config.Config{CookieCloudUUID: "u", CookieCloudPassword: "p"}, want: "COOKIECLOUD_BASE_URL"},
		{name: "missing uuid", cfg: config.Config{CookieCloudBaseURL: "https://example.com", CookieCloudPassword: "p"}, want: "COOKIECLOUD_UUID"},
		{name: "missing password", cfg: config.Config{CookieCloudBaseURL: "https://example.com", CookieCloudUUID: "u"}, want: "COOKIECLOUD_PASSWORD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewDouyinCookieCloudProvider(tc.cfg, &http.Client{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v", err)
			}
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get/u1" {
			http.NotFound(w, r)
			return
		}
		plain := map[string]any{
			"cookie_data": map[string]any{
				".douyin.com": []any{map[string]any{"name": "a", "value": "1"}},
			},
			"local_storage_data": map[string]any{},
			"update_time":        "2026-02-01T00:00:00Z",
		}
		plainBytes, _ := json.Marshal(plain)
		encrypted := encryptCookieCloudFixed("u1", "p1", plainBytes)
		_ = json.NewEncoder(w).Encode(map[string]any{"encrypted": encrypted, "crypto_type": "aes-128-cbc-fixed"})
	}))
	defer srv.Close()

	cfg := minimalCookieCloudConfig()
	cfg.CacheType = "memory"
	cfg.CookieCloudBaseURL = srv.URL
	cfg.CookieCloudUUID = "u1"
	cfg.CookieCloudPassword = "p1"
	cfg.CookieCloudDomain = "" // should default to douyin.com
	cfg.CookieCloudCookieExpireHours = 0
	cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"

	p, err := NewDouyinCookieCloudProvider(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewDouyinCookieCloudProvider: %v", err)
	}
	defer func() { _ = p.Close() }()

	if p.domain != "douyin.com" {
		t.Fatalf("domain=%q", p.domain)
	}
	if p.ttl != 72*time.Hour {
		t.Fatalf("ttl=%v", p.ttl)
	}
}

func TestDouyinCookieCloudProvider_ConcurrentGetCookieSingleFetch(t *testing.T) {
	uuid := "u1"
	password := "p1"

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get/"+uuid {
			http.NotFound(w, r)
			return
		}
		hits.Add(1)

		plain := map[string]any{
			"cookie_data": map[string]any{
				".douyin.com": []any{
					map[string]any{"name": "a", "value": "1", "domain": ".douyin.com", "path": "/"},
					map[string]any{"name": "b", "value": "2", "domain": ".douyin.com", "path": "/"},
				},
			},
			"local_storage_data": map[string]any{},
			"update_time":        "2026-02-01T00:00:00Z",
		}
		plainBytes, _ := json.Marshal(plain)
		encrypted := encryptCookieCloudFixed(uuid, password, plainBytes)
		_ = json.NewEncoder(w).Encode(map[string]any{"encrypted": encrypted, "crypto_type": "aes-128-cbc-fixed"})
	}))
	defer srv.Close()

	cfg := minimalCookieCloudConfig()
	cfg.CacheType = "memory"
	cfg.CookieCloudBaseURL = srv.URL
	cfg.CookieCloudUUID = uuid
	cfg.CookieCloudPassword = password
	cfg.CookieCloudDomain = "douyin.com"
	cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"
	cfg.CookieCloudCookieExpireHours = 72

	p, err := NewDouyinCookieCloudProvider(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewDouyinCookieCloudProvider: %v", err)
	}
	defer func() { _ = p.Close() }()

	const workers = 20
	results := make([]string, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		idx := i
		go func() {
			defer wg.Done()
			results[idx], errs[idx] = p.GetCookie(context.Background())
		}()
	}
	wg.Wait()

	for i := 0; i < workers; i++ {
		if errs[i] != nil {
			t.Fatalf("err[%d]=%v", i, errs[i])
		}
		if results[i] != "a=1; b=2" {
			t.Fatalf("result[%d]=%q", i, results[i])
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 upstream hit, got %d", hits.Load())
	}
}
