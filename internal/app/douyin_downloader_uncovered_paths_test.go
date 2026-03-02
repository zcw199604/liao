package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"strings"
	"testing"
	"time"
)

func TestTikTokDownloaderClient_DouyinAccount_ItemsNilDefaultsToEmptySlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/douyin/account/page" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "ok",
			"data": map[string]any{
				"next_cursor": 9,
				"has_more":    true,
			},
		})
	}))
	defer srv.Close()

	client := NewTikTokDownloaderClient(srv.URL, "", srv.Client())
	got, err := client.DouyinAccount(t.Context(), "u", "post", 0, 18, "", "")
	if err != nil {
		t.Fatalf("DouyinAccount err=%v", err)
	}
	items, ok := got["aweme_list"].([]any)
	if !ok {
		t.Fatalf("aweme_list type=%T", got["aweme_list"])
	}
	if len(items) != 0 {
		t.Fatalf("aweme_list len=%d, want 0", len(items))
	}
}

func TestDouyinDownloaderService_FetchDetail_AuthorSecUIDFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/douyin/detail" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "ok",
			"data": map[string]any{
				"desc":      "title",
				"type":      "video",
				"downloads": []any{"https://example.com/v.mp4"},
				"author": map[string]any{
					"secUid": "sec-from-author-secuId",
				},
			},
		})
	}))
	defer srv.Close()

	svc := NewDouyinDownloaderService(srv.URL, "", "", "", time.Second)
	detail, err := svc.FetchDetail(context.Background(), "d1", "", "")
	if err != nil {
		t.Fatalf("FetchDetail err=%v", err)
	}
	if detail.SecUserID != "sec-from-author-secuId" {
		t.Fatalf("SecUserID=%q", detail.SecUserID)
	}
}

func TestDouyinDownloaderService_RefreshAndFetchAccount_GuardErrors(t *testing.T) {
	t.Run("RefreshDetailBestEffort not configured", func(t *testing.T) {
		svc := NewDouyinDownloaderService("", "", "", "", time.Second)
		if _, err := svc.RefreshDetailBestEffort("d1"); err == nil {
			t.Fatalf("expected not configured error")
		}
	})

	t.Run("RefreshDetailBestEffort empty detail id", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)
		if _, err := svc.RefreshDetailBestEffort(" "); err == nil {
			t.Fatalf("expected empty detail_id error")
		}
	})

	t.Run("FetchAccount effectiveCookie error", func(t *testing.T) {
		svc := &DouyinDownloaderService{
			api:             NewTikTokDownloaderClient("http://example.com", "", &http.Client{}),
			cache:           newLRUCache(10, time.Second),
			upstreamTimeout: time.Second,
			cookieProvider: cookieProviderFunc(func(context.Context) (string, error) {
				return "", errors.New("cookie provider failed")
			}),
		}

		_, err := svc.FetchAccount(context.Background(), "sec-user", "post", "", "", 0, 18)
		if err == nil || !strings.Contains(err.Error(), "cookie provider failed") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("RefreshDetailBestEffort type assertion fallback", func(t *testing.T) {
		svc := NewDouyinDownloaderService("http://example.com", "", "", "", time.Second)

		var wg sync.WaitGroup
		started := make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			close(started)
			_, _, _ = svc.refreshDetailGroup.Do("detail-1", func() (any, error) {
				time.Sleep(100 * time.Millisecond)
				return "not-detail-struct", nil
			})
		}()
		<-started

		_, err := svc.RefreshDetailBestEffort("detail-1")
		wg.Wait()
		if err == nil || !strings.Contains(err.Error(), "刷新作品直链失败") {
			t.Fatalf("err=%v", err)
		}
	})
}
