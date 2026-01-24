package app

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisFactoryFns_Coverage(t *testing.T) {
	mr := miniredis.RunT(t)
	redisURL := "redis://" + mr.Addr()

	uic, err := newRedisUserInfoCacheServiceFn(
		redisURL,
		"",
		0,
		"",
		0,
		"",
		"",
		1,
		0,
		1,
		1,
	)
	if err != nil {
		t.Fatalf("newRedisUserInfoCacheServiceFn: %v", err)
	}
	if closer, ok := uic.(interface{ Close() error }); ok {
		_ = closer.Close()
	}

	chc, err := newRedisChatHistoryCacheServiceFn(
		redisURL,
		"",
		0,
		"",
		0,
		"",
		1,
		0,
		1,
	)
	if err != nil {
		t.Fatalf("newRedisChatHistoryCacheServiceFn: %v", err)
	}
	if closer, ok := chc.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}
