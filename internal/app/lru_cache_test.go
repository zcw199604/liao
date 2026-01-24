package app

import (
	"testing"
	"time"
)

func TestLRUCache_BasicsAndDelete(t *testing.T) {
	c := newLRUCache(1, 10*time.Millisecond)

	c.Set("", "x")
	c.Set("k", nil)
	if _, ok := c.Get("k"); ok {
		t.Fatalf("expected miss")
	}

	c.Set("k1", "v1")
	if v, ok := c.Get("k1"); !ok || v.(string) != "v1" {
		t.Fatalf("get=%v ok=%v", v, ok)
	}

	// exceed maxEntries -> evict back
	c.Set("k2", "v2")
	if _, ok := c.Get("k1"); ok {
		t.Fatalf("expected evicted")
	}

	// delete no-op cases
	c.Delete("")
	c.Delete("not-exist")

	// delete existing
	c.Delete("k2")
	if _, ok := c.Get("k2"); ok {
		t.Fatalf("expected deleted")
	}
}

func TestLRUCache_Get_CleansCorruptedOrExpired(t *testing.T) {
	c := newLRUCache(10, time.Second)
	c.Set("k", "v")

	// corrupt entry type
	el := c.data["k"]
	el.Value = "bad"
	if _, ok := c.Get("k"); ok {
		t.Fatalf("expected miss")
	}

	c.Set("k2", "v2")
	el2 := c.data["k2"]
	el2.Value = lruCacheEntry{key: "k2", value: "v2", expiresAt: time.Now().Add(-1 * time.Second)}
	if _, ok := c.Get("k2"); ok {
		t.Fatalf("expected expired")
	}
}

func TestNewLRUCache_Defaults(t *testing.T) {
	c := newLRUCache(0, 0)
	if c.maxEntries != 10000 {
		t.Fatalf("maxEntries=%d", c.maxEntries)
	}
	if c.ttl <= 0 {
		t.Fatalf("ttl=%v", c.ttl)
	}
}

func TestLRUCache_Set_UpdatesExistingEntry(t *testing.T) {
	c := newLRUCache(10, time.Second)
	c.Set("k", "v1")
	c.Set("k", "v2")

	v, ok := c.Get("k")
	if !ok || v.(string) != "v2" {
		t.Fatalf("get=%v ok=%v", v, ok)
	}
}
