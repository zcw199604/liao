package app

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"liao/internal/config"
)

func TestDouyinCookieCloudProvider_MemoryCache(t *testing.T) {
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

		_ = json.NewEncoder(w).Encode(map[string]any{
			"encrypted":   encrypted,
			"crypto_type": "aes-128-cbc-fixed",
		})
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

	got, err := p.GetCookie(context.Background())
	if err != nil {
		t.Fatalf("GetCookie: %v", err)
	}
	if got != "a=1; b=2" {
		t.Fatalf("cookie mismatch: %q", got)
	}

	got2, err := p.GetCookie(context.Background())
	if err != nil {
		t.Fatalf("GetCookie(2): %v", err)
	}
	if got2 != "a=1; b=2" {
		t.Fatalf("cookie mismatch (2): %q", got2)
	}

	if hits.Load() != 1 {
		t.Fatalf("expected 1 CookieCloud hit, got %d", hits.Load())
	}
}

func TestDouyinCookieCloudProvider_RedisCache(t *testing.T) {
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

		_ = json.NewEncoder(w).Encode(map[string]any{
			"encrypted":   encrypted,
			"crypto_type": "aes-128-cbc-fixed",
		})
	}))
	defer srv.Close()

	mr := miniredis.RunT(t)
	addr := mr.Addr()

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("Atoi: %v", err)
	}

	cfg := minimalCookieCloudConfig()
	cfg.CacheType = "redis"
	cfg.RedisURL = "redis://" + addr
	cfg.RedisHost = host
	cfg.RedisPort = port
	cfg.RedisTimeoutSeconds = 2
	cfg.CookieCloudBaseURL = srv.URL
	cfg.CookieCloudUUID = uuid
	cfg.CookieCloudPassword = password
	cfg.CookieCloudDomain = "douyin.com"
	cfg.CookieCloudCryptoType = "aes-128-cbc-fixed"
	cfg.CookieCloudCookieExpireHours = 1

	p1, err := NewDouyinCookieCloudProvider(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewDouyinCookieCloudProvider: %v", err)
	}
	t.Cleanup(func() { _ = p1.Close() })

	got, err := p1.GetCookie(context.Background())
	if err != nil {
		t.Fatalf("GetCookie: %v", err)
	}
	if got != "a=1; b=2" {
		t.Fatalf("cookie mismatch: %q", got)
	}

	key := buildDouyinCookieRedisKey(cfg, "douyin.com")
	if !mr.Exists(key) {
		t.Fatalf("expected redis key %q to exist", key)
	}
	if ttl := mr.TTL(key); ttl <= 0 || ttl > time.Hour {
		t.Fatalf("unexpected ttl: %v", ttl)
	}

	// New provider simulating process restart: should hit Redis, not CookieCloud.
	p2, err := NewDouyinCookieCloudProvider(cfg, srv.Client())
	if err != nil {
		t.Fatalf("NewDouyinCookieCloudProvider(2): %v", err)
	}
	t.Cleanup(func() { _ = p2.Close() })

	got2, err := p2.GetCookie(context.Background())
	if err != nil {
		t.Fatalf("GetCookie(2): %v", err)
	}
	if got2 != "a=1; b=2" {
		t.Fatalf("cookie mismatch (2): %q", got2)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 CookieCloud hit, got %d", hits.Load())
	}
}

func minimalCookieCloudConfig() config.Config {
	// Keep this helper here to avoid depending on config.Load env state in tests.
	return config.Config{
		CacheType:           "memory",
		RedisTimeoutSeconds: 1,
	}
}

func encryptCookieCloudFixed(uuid, password string, plain []byte) string {
	key := deriveCookieCloudPassphrase(uuid, password)

	block, _ := aes.NewCipher(key)
	iv := make([]byte, aes.BlockSize) // 16 bytes of 0x00
	mode := cipher.NewCBCEncrypter(block, iv)

	padded := pkcs7Pad(plain, aes.BlockSize)
	out := make([]byte, len(padded))
	mode.CryptBlocks(out, padded)

	return base64.StdEncoding.EncodeToString(out)
}

func deriveCookieCloudPassphrase(uuid, password string) []byte {
	sum := md5.Sum([]byte(uuid + "-" + password))
	hexStr := hex.EncodeToString(sum[:])
	return []byte(hexStr[:16])
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padLen)
	}
	return out
}
