package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"liao/internal/config"
	"liao/internal/cookiecloud"
)

// DouyinCookieProvider provides a cookie string used for TikTokDownloader's "cookie" JSON field.
// It intentionally returns the "Cookie header value" format: "a=1; b=2".
type DouyinCookieProvider interface {
	GetCookie(ctx context.Context) (string, error)
}

// DouyinCookieCloudProvider lazily fetches Douyin cookie from CookieCloud, and caches it in Redis or memory.
//
// Cache strategy:
// - If CACHE_TYPE=redis: Redis is used as the durable cache (TTL applied); an in-memory L1 cache is also kept.
// - Otherwise: only in-memory cache (TTL applied).
//
// Note: we never send password to the CookieCloud server (client-side decrypt only).
type DouyinCookieCloudProvider struct {
	cookieCloud *cookiecloud.Client
	httpClient  *http.Client

	uuid      string
	password  string
	domain    string
	cryptoTyp cookiecloud.CryptoType

	ttl time.Duration

	redis        *redis.Client
	redisTimeout time.Duration
	redisKey     string

	local *lruCache

	fetchMu sync.Mutex
}

const (
	douyinCookieCloudLocalKey = "douyin-cookie"
	douyinCookieCloudRedisKey = "cookiecloud:douyin:cookie"
)

func NewDouyinCookieCloudProvider(cfg config.Config, httpClient *http.Client) (*DouyinCookieCloudProvider, error) {
	baseURL := strings.TrimSpace(cfg.CookieCloudBaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("CookieCloud 未配置（请设置 COOKIECLOUD_BASE_URL）")
	}
	uuid := strings.TrimSpace(cfg.CookieCloudUUID)
	if uuid == "" {
		return nil, fmt.Errorf("CookieCloud 未配置（请设置 COOKIECLOUD_UUID）")
	}
	password := strings.TrimSpace(cfg.CookieCloudPassword)
	if password == "" {
		return nil, fmt.Errorf("CookieCloud 未配置（请设置 COOKIECLOUD_PASSWORD）")
	}

	domain := strings.TrimSpace(cfg.CookieCloudDomain)
	if domain == "" {
		domain = "douyin.com"
	}

	ttlHours := cfg.CookieCloudCookieExpireHours
	if ttlHours <= 0 {
		ttlHours = 72
	}
	ttl := time.Duration(ttlHours) * time.Hour

	cc, err := cookiecloud.NewClient(baseURL)
	if err != nil {
		return nil, err
	}
	cc.WithHTTPClient(httpClient)

	cryptoTyp := cookiecloud.CryptoType(strings.TrimSpace(cfg.CookieCloudCryptoType))

	p := &DouyinCookieCloudProvider{
		cookieCloud:  cc,
		httpClient:   httpClient,
		uuid:         uuid,
		password:     password,
		domain:       domain,
		cryptoTyp:    cryptoTyp,
		ttl:          ttl,
		redisTimeout: time.Duration(cfg.RedisTimeoutSeconds) * time.Second,
		redisKey:     buildDouyinCookieRedisKey(cfg, domain),
		local:        newLRUCache(4, ttl),
	}
	if p.redisTimeout <= 0 {
		p.redisTimeout = 15 * time.Second
	}

	if cfg.CacheType == "redis" {
		opts, err := buildRedisOptions(cfg.RedisURL, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, cfg.RedisDB, p.redisTimeout)
		if err != nil {
			return nil, err
		}
		client := redis.NewClient(opts)
		ctx, cancel := context.WithTimeout(context.Background(), p.redisTimeout)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("连接 Redis 失败: %w", err)
		}
		p.redis = client
	}

	return p, nil
}

func buildDouyinCookieRedisKey(cfg config.Config, domain string) string {
	base := strings.TrimSpace(cfg.CookieCloudBaseURL)
	base = strings.TrimRight(base, "/")
	uuid := strings.TrimSpace(cfg.CookieCloudUUID)
	domain = strings.TrimSpace(strings.TrimPrefix(domain, "."))
	// Include baseURL to avoid collisions when multiple CookieCloud servers share one Redis.
	return douyinCookieCloudRedisKey + ":" + base + ":" + uuid + ":" + domain
}

func (p *DouyinCookieCloudProvider) Close() error {
	if p == nil || p.redis == nil {
		return nil
	}
	return p.redis.Close()
}

func (p *DouyinCookieCloudProvider) GetCookie(ctx context.Context) (string, error) {
	if p == nil {
		return "", fmt.Errorf("CookieCloud provider 未初始化")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// L1 cache hit.
	if p.local != nil {
		if v, ok := p.local.Get(douyinCookieCloudLocalKey); ok {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s, nil
				}
			}
			p.local.Delete(douyinCookieCloudLocalKey)
		}
	}

	// L2 Redis cache hit.
	if p.redis != nil {
		val, err := p.redisGet(ctx)
		if err == nil {
			val = strings.TrimSpace(val)
			if val != "" {
				if p.local != nil {
					p.local.Set(douyinCookieCloudLocalKey, val)
				}
				return val, nil
			}
		} else if err != redis.Nil {
			// Best-effort: if Redis is temporarily unavailable, still try CookieCloud.
			slog.Warn("CookieCloud Redis 读取失败，尝试直接拉取", "error", err)
		}
	}

	// Cache miss: ensure only one goroutine fetches from CookieCloud per process.
	p.fetchMu.Lock()
	defer p.fetchMu.Unlock()

	// Double-check after lock (another goroutine may have already fetched it).
	if p.local != nil {
		if v, ok := p.local.Get(douyinCookieCloudLocalKey); ok {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s, nil
				}
			}
			p.local.Delete(douyinCookieCloudLocalKey)
		}
	}
	if p.redis != nil {
		val, err := p.redisGet(ctx)
		if err == nil {
			val = strings.TrimSpace(val)
			if val != "" {
				if p.local != nil {
					p.local.Set(douyinCookieCloudLocalKey, val)
				}
				return val, nil
			}
		} else if err != redis.Nil {
			slog.Warn("CookieCloud Redis 读取失败（加锁后）", "error", err)
		}
	}

	// Fetch from CookieCloud and cache.
	val, err := p.cookieCloud.GetCookieHeader(ctx, p.uuid, p.password, p.domain, p.cryptoTyp)
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", fmt.Errorf("CookieCloud 返回空 Cookie（domain=%s）", p.domain)
	}

	// Avoid logging cookie content (sensitive). Log only high-level fetch info.
	slog.Info("CookieCloud 拉取抖音 Cookie 成功", "domain", p.domain, "cookie_len", len(val), "redis", p.redis != nil)

	if p.local != nil {
		p.local.Set(douyinCookieCloudLocalKey, val)
	}
	if p.redis != nil {
		if err := p.redisSet(ctx, val); err != nil {
			// Best-effort: local cache still works for this process.
			slog.Warn("CookieCloud Redis 写入失败", "error", err)
		}
	}
	return val, nil
}

func (p *DouyinCookieCloudProvider) redisGet(ctx context.Context) (string, error) {
	if p == nil || p.redis == nil {
		return "", redis.Nil
	}
	ctx2, cancel := context.WithTimeout(ctx, p.redisTimeout)
	defer cancel()
	return p.redis.Get(ctx2, p.redisKey).Result()
}

func (p *DouyinCookieCloudProvider) redisSet(ctx context.Context, value string) error {
	if p == nil || p.redis == nil {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, p.redisTimeout)
	defer cancel()
	return p.redis.Set(ctx2, p.redisKey, value, p.ttl).Err()
}
