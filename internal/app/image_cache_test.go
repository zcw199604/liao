package app

import (
	"testing"
	"time"
)

func TestImageCacheService_AddGetRebuildClearAndExpire(t *testing.T) {
	svc := NewImageCacheService()

	if got := svc.GetCachedImages("u1"); got != nil {
		t.Fatalf("expected nil cache, got %+v", got)
	}

	svc.AddImageToCache("u1", "/images/a.png")
	svc.AddImageToCache("u1", "/images/b.png")

	got := svc.GetCachedImages("u1")
	if got == nil || len(got.ImageURLs) != 2 {
		t.Fatalf("expected 2 cached urls, got %+v", got)
	}
	if got.ImageURLs[0] != "/images/a.png" || got.ImageURLs[1] != "/images/b.png" {
		t.Fatalf("unexpected cache urls: %+v", got.ImageURLs)
	}

	svc.ClearCache("u1")
	if got := svc.GetCachedImages("u1"); got != nil {
		t.Fatalf("expected nil after ClearCache, got %+v", got)
	}

	svc.RebuildCache("u1", []string{"/images/c.png"})
	got = svc.GetCachedImages("u1")
	if got == nil || len(got.ImageURLs) != 1 || got.ImageURLs[0] != "/images/c.png" {
		t.Fatalf("unexpected rebuild cache: %+v", got)
	}

	// 过期应自动清理
	svc.cache["u1"].ExpireTime = time.Now().UnixMilli() - 1
	if got := svc.GetCachedImages("u1"); got != nil {
		t.Fatalf("expected nil after expired, got %+v", got)
	}
	if _, ok := svc.cache["u1"]; ok {
		t.Fatalf("expected cache entry removed after expired")
	}
}
