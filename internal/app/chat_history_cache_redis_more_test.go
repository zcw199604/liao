package app

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestNewRedisChatHistoryCacheService_DefaultsAndErrors(t *testing.T) {
	mr := miniredis.RunT(t)

	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"", // default prefix
		0,  // default expireDays
		0,
		0, // default timeout
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if svc.keyPrefix != "user:chathistory:" {
		t.Fatalf("keyPrefix=%q", svc.keyPrefix)
	}
	if svc.expire != 30*24*time.Hour {
		t.Fatalf("expire=%v", svc.expire)
	}
	if svc.timeout != 15*time.Second {
		t.Fatalf("timeout=%v", svc.timeout)
	}

	if _, err := NewRedisChatHistoryCacheService("redis://%zz", "", 0, "", 0, "p:", 1, 0, 1); err == nil {
		t.Fatalf("expected error")
	}

	if _, err := NewRedisChatHistoryCacheService("redis://127.0.0.1:1", "", 0, "", 0, "p:", 1, 0, 1); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRedisChatHistoryCacheService_CloseAndFlushOnce_EdgeCases(t *testing.T) {
	var nilSvc *RedisChatHistoryCacheService
	if err := nilSvc.Close(); err != nil {
		t.Fatalf("err=%v", err)
	}
	nilSvc.flushOnce()
	(&RedisChatHistoryCacheService{}).flushOnce()

	mr := miniredis.RunT(t)
	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		0,
		1,
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	svc.timeout = 0
	svc.flushOnce()
}

func TestRedisChatHistoryCacheService_FlushPending_EdgeCases(t *testing.T) {
	var nilSvc *RedisChatHistoryCacheService
	if err := nilSvc.flushPending(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}

	mr := miniredis.RunT(t)
	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		1,
		1,
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if err := svc.flushPending(nil); err != nil {
		t.Fatalf("err=%v", err)
	}
	if err := svc.flushPending(context.Background()); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestRedisChatHistoryCacheService_SaveMessages_EdgeCases(t *testing.T) {
	var nilSvc *RedisChatHistoryCacheService
	nilSvc.SaveMessages(context.Background(), "a_b", []map[string]any{{"Tid": "1"}})

	mr := miniredis.RunT(t)
	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		0,
		1,
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	svc.SaveMessages(context.Background(), " ", []map[string]any{{"Tid": "1"}})
	svc.SaveMessages(context.Background(), "a_b", nil)
	svc.SaveMessages(context.Background(), "a_b", []map[string]any{nil, {"Tid": ""}, {"Tid": "not-int"}, {"Tid": "1", "x": make(chan int)}})

	// ctx=nil -> writeCtx=Background
	svc.SaveMessages(nil, "a_b", []map[string]any{{"Tid": "1", "content": "m1"}})

	// flushInterval>0 but pending=nil
	svc2, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		1,
		1,
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc2.Close() })
	svc2.pendingMu.Lock()
	svc2.pending = nil
	svc2.pendingMu.Unlock()
	svc2.SaveMessages(context.Background(), "a_b", []map[string]any{{"Tid": "1"}})
}

func TestRedisChatHistoryCacheService_WriteBatch_EdgeCases(t *testing.T) {
	var nilSvc *RedisChatHistoryCacheService
	if err := nilSvc.writeBatch(context.Background(), nil); err != nil {
		t.Fatalf("err=%v", err)
	}

	mr := miniredis.RunT(t)
	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		0,
		1,
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if err := svc.writeBatch(nil, []pendingChatHistoryMessage{{conversationKey: "a", tid: "1"}}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if err := svc.writeBatch(context.Background(), nil); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestRedisChatHistoryCacheService_GetMessages_EdgeCases(t *testing.T) {
	var nilSvc *RedisChatHistoryCacheService
	if got, err := nilSvc.GetMessages(context.Background(), "a_b", "0", 10); err != nil || len(got) != 0 {
		t.Fatalf("got=%v err=%v", got, err)
	}

	mr := miniredis.RunT(t)
	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		0,
		1,
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	if got, err := svc.GetMessages(context.Background(), " ", "0", 10); err != nil || len(got) != 0 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if got, err := svc.GetMessages(context.Background(), "a_b", "0", 0); err != nil || len(got) != 0 {
		t.Fatalf("got=%v err=%v", got, err)
	}

	conv := "a_b"
	key := svc.zsetKey(conv)
	mr.ZAdd(key, 1, "")
	mr.ZAdd(key, 2, "2|not-json")
	mr.ZAdd(key, 3, "3|null")
	mr.ZAdd(key, 4, "4|{\"Tid\":\"4\"}")
	mr.ZAdd(key, 5, "5|")

	got, err := svc.GetMessages(nil, conv, "bad", 10)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}

	// redis error
	_ = svc.Close()
	if _, err := svc.GetMessages(context.Background(), conv, "0", 10); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRedisChatHistoryCacheService_FlushPending_PipelineError(t *testing.T) {
	mr := miniredis.RunT(t)
	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		1,
		1,
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	svc.pendingMu.Lock()
	svc.pending["a_b|1"] = pendingChatHistoryMessage{conversationKey: "a_b", tid: "1", score: 1, member: "1|{}"}
	svc.pendingMu.Unlock()

	mr.Close()
	if err := svc.flushPending(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}
