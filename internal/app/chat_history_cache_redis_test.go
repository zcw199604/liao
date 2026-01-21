package app

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisChatHistoryCacheService_SaveAndQuery(t *testing.T) {
	mr := miniredis.RunT(t)

	svc, err := NewRedisChatHistoryCacheService(
		"redis://"+mr.Addr(),
		"",
		0,
		"",
		0,
		"test:ch:",
		1,
		0, // 立即写入，方便测试
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService failed: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	conv := "a_b"
	svc.SaveMessages(context.Background(), conv, []map[string]any{
		{"Tid": "1", "id": "a", "toid": "b", "content": "m1", "time": "t1"},
		{"Tid": "2", "id": "b", "toid": "a", "content": "m2", "time": "t2"},
		{"Tid": "3", "id": "a", "toid": "b", "content": "m3", "time": "t3"},
	})

	got, err := svc.GetMessages(context.Background(), conv, "0", 2)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if tid := extractHistoryMessageTid(got[0]); tid != "3" {
		t.Fatalf("tid[0]=%q, want %q", tid, "3")
	}
	if tid := extractHistoryMessageTid(got[1]); tid != "2" {
		t.Fatalf("tid[1]=%q, want %q", tid, "2")
	}

	got, err = svc.GetMessages(context.Background(), conv, "3", 10)
	if err != nil {
		t.Fatalf("GetMessages(beforeTid) failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(beforeTid)=%d, want 2", len(got))
	}
	if tid := extractHistoryMessageTid(got[0]); tid != "2" {
		t.Fatalf("tid(beforeTid)[0]=%q, want %q", tid, "2")
	}
	if tid := extractHistoryMessageTid(got[1]); tid != "1" {
		t.Fatalf("tid(beforeTid)[1]=%q, want %q", tid, "1")
	}
}

func TestRedisChatHistoryCacheService_DedupByTid(t *testing.T) {
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
	)
	if err != nil {
		t.Fatalf("NewRedisChatHistoryCacheService failed: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	conv := "a_b"
	svc.SaveMessages(context.Background(), conv, []map[string]any{
		{"Tid": "1", "id": "a", "toid": "b", "content": "m1", "time": "t1"},
		{"Tid": "2", "id": "a", "toid": "b", "content": "m2", "time": "t2"},
	})

	// 重复写入相同 Tid：应覆盖旧值而不是产生重复
	svc.SaveMessages(context.Background(), conv, []map[string]any{
		{"Tid": "2", "id": "a", "toid": "b", "content": "m2-new", "time": "t2"},
	})

	got, err := svc.GetMessages(context.Background(), conv, "0", 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if tid := extractHistoryMessageTid(got[0]); tid != "2" {
		t.Fatalf("tid[0]=%q, want %q", tid, "2")
	}
	if content := toString(got[0]["content"]); content != "m2-new" {
		t.Fatalf("content[0]=%q, want %q", content, "m2-new")
	}
	if tid := extractHistoryMessageTid(got[1]); tid != "1" {
		t.Fatalf("tid[1]=%q, want %q", tid, "1")
	}
}
