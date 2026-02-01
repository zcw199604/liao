package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewTikTokDownloaderClient_Trims(t *testing.T) {
	c := NewTikTokDownloaderClient(" http://example.com/ ", " t ", nil)
	if c.baseURL != "http://example.com" {
		t.Fatalf("baseURL=%q", c.baseURL)
	}
	if c.token != "t" {
		t.Fatalf("token=%q", c.token)
	}
	if c.httpClient == nil {
		t.Fatalf("expected httpClient")
	}
}

func TestTikTokDownloaderClient_postJSON(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		c := NewTikTokDownloaderClient("", "", &http.Client{})
		if err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, &map[string]any{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("new request error", func(t *testing.T) {
		c := NewTikTokDownloaderClient("http://[::1", "", &http.Client{})
		if err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, &map[string]any{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("do error", func(t *testing.T) {
		c := NewTikTokDownloaderClient("http://example.com", "", &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("do fail")
			}),
		})
		if err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, &map[string]any{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("read error", func(t *testing.T) {
		c := NewTikTokDownloaderClient("http://example.com", "", &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       errReadCloser{},
				}, nil
			}),
		})
		if err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, &map[string]any{}); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("status error and truncate msg", func(t *testing.T) {
		longBody := strings.Repeat("x", 400)
		c := NewTikTokDownloaderClient("http://example.com", "", &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Internal Server Error",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(longBody)),
				}, nil
			}),
		})
		err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, &map[string]any{})
		if err == nil || !strings.Contains(err.Error(), "TikTokDownloader 上游错误") || !strings.Contains(err.Error(), "...") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("out nil skips unmarshal", func(t *testing.T) {
		c := NewTikTokDownloaderClient("http://example.com", "", &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				}, nil
			}),
		})
		if err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, nil); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("unmarshal error", func(t *testing.T) {
		c := NewTikTokDownloaderClient("http://example.com", "", &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("not-json")),
				}, nil
			}),
		})
		var out map[string]any
		err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, &out)
		if err == nil || !strings.Contains(err.Error(), "响应解析失败") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		c := NewTikTokDownloaderClient("http://example.com", "tok", &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("token") != "tok" {
					t.Fatalf("token header=%q", r.Header.Get("token"))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"message":"ok","url":"u"}`)),
				}, nil
			}),
		})
		var out tikTokDownloaderURLResponse
		if err := c.postJSON(t.Context(), "/x", map[string]any{"a": 1}, &out); err != nil {
			t.Fatalf("err=%v", err)
		}
		if out.URL != "u" {
			t.Fatalf("out=%v", out)
		}
	})
}

func TestTikTokDownloaderClient_APIs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/douyin/share":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "ok",
				"url":     " https://www.douyin.com/video/123 ",
			})
		case "/douyin/detail":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "ok",
				"data": map[string]any{
					"desc":          " t ",
					"type":          " video ",
					"static_cover":  " ",
					"dynamic_cover": " https://c ",
					"downloads":     []any{" https://d1 ", " ", "https://d2"},
				},
			})
		case "/douyin/account/page":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "ok",
				"data": map[string]any{
					"items":       []any{},
					"next_cursor": 0,
					"has_more":    false,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewTikTokDownloaderClient(srv.URL, "", srv.Client())

	urlValue, err := client.DouyinShare(t.Context(), "x", "")
	if err != nil || urlValue != "https://www.douyin.com/video/123" {
		t.Fatalf("url=%q err=%v", urlValue, err)
	}

	detail, err := client.DouyinDetail(t.Context(), "123", "", "")
	if err != nil || strings.TrimSpace(asString(detail["desc"])) != "t" {
		t.Fatalf("detail=%v err=%v", detail, err)
	}

	account, err := client.DouyinAccount(t.Context(), "u", "post", 0, 18, "", "")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if _, ok := account["aweme_list"]; !ok {
		t.Fatalf("expected aweme_list, got %v", account)
	}
}

func TestTikTokDownloaderClient_DetailAndAccount_Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/douyin/detail":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "no data",
				"data":    nil,
			})
		case "/douyin/account/page":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "bad",
				"data":    "x",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewTikTokDownloaderClient(srv.URL, "", srv.Client())

	if _, err := client.DouyinDetail(t.Context(), "x", "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.DouyinAccount(t.Context(), "u", "post", 0, 18, "", ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestTikTokDownloaderClient_DouyinShare_Error(t *testing.T) {
	client := NewTikTokDownloaderClient("", "", &http.Client{})
	if _, err := client.DouyinShare(t.Context(), "x", ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestTikTokDownloaderClient_DetailAccount_MoreErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/douyin/detail":
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "x", "data": []any{}})
		case "/douyin/account/page":
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "x", "data": nil})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewTikTokDownloaderClient(srv.URL, "", srv.Client())
	if _, err := client.DouyinDetail(t.Context(), "x", "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.DouyinAccount(t.Context(), "u", "post", 0, 18, "", ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestTikTokDownloaderClient_DetailAccount_PostJSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)

	client := NewTikTokDownloaderClient(srv.URL, "", srv.Client())
	if _, err := client.DouyinDetail(t.Context(), "x", "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.DouyinAccount(t.Context(), "u", "post", 0, 18, "", ""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDouyinDownloaderService_ParsingHelpers(t *testing.T) {
	if got := extractDouyinDetailID(" 123 "); got != "123" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("/video/456"); got != "456" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("/note/789"); got != "789" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("https://www.douyin.com/video/456"); got != "456" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("https://www.douyin.com/note/789"); got != "789" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("x?modal_id=42"); got != "42" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("x?aweme_id=43"); got != "43" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("share https://www.douyin.com/video/44 end"); got != "44" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("share https://www.douyin.com/note/45 end"); got != "45" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("share https://example.com?modal_id=46 end"); got != "46" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("share https://example.com?aweme_id=47 end"); got != "47" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID("share https://example.com end"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinDetailID(" "); got != "" {
		t.Fatalf("got=%q", got)
	}

	if got := extractDouyinSecUserID(" "); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("MS4wLjABAAAAAAAAAAAAAAA"); got != "MS4wLjABAAAAAAAAAAAAAAA" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("/user/u1"); got != "u1" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("https://www.douyin.com/user/u1"); got != "u1" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("x?sec_uid=MS4wLjAB%2Bxx"); got != "MS4wLjAB+xx" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("x?sec_user_id=%ZZ"); got != "%ZZ" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("share https://example.com?sec_uid=MS4wLjAB%2Bzz end"); got != "MS4wLjAB+zz" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("share https://example.com?sec_user_id=%ZZ end"); got != "%ZZ" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinSecUserID("share https://www.douyin.com/user/u2 end"); got != "u2" {
		t.Fatalf("got=%q", got)
	}
}

func TestNewDouyinDownloaderService_DefaultTimeout(t *testing.T) {
	svc := NewDouyinDownloaderService("http://example.com", "", " c ", " p ", 0)
	if svc.upstreamTimeout != 60*time.Second {
		t.Fatalf("timeout=%v", svc.upstreamTimeout)
	}
	if svc.defaultCookie != "c" || svc.defaultProxy != "p" {
		t.Fatalf("cookie=%q proxy=%q", svc.defaultCookie, svc.defaultProxy)
	}
}

func TestDouyinDownloaderService_EffectiveDefaults(t *testing.T) {
	s := &DouyinDownloaderService{upstreamTimeout: -1, defaultCookie: "c", defaultProxy: "p"}
	if got := s.effectiveUpstreamTimeout(); got != 60*time.Second {
		t.Fatalf("got=%v", got)
	}
	if got := s.effectiveCookie(" "); got != "c" {
		t.Fatalf("got=%q", got)
	}
	if got := s.effectiveCookie(" x "); got != "x" {
		t.Fatalf("got=%q", got)
	}
	if got := s.effectiveProxy(" "); got != "p" {
		t.Fatalf("got=%q", got)
	}
	if got := s.effectiveProxy(" x "); got != "x" {
		t.Fatalf("got=%q", got)
	}
}

func TestDouyinDownloaderService_Flows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		switch r.URL.Path {
		case "/douyin/share":
			text := strings.TrimSpace(asString(payload["text"]))
			urlValue := "https://www.douyin.com/video/123"
			if text == "err" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("boom"))
				return
			}
			if text == "empty" {
				urlValue = " "
			}
			if text == "noid" {
				urlValue = "https://example.com"
			}
			if text == "ok2" {
				urlValue = "https://www.douyin.com/video/124"
			}
			if text == "secok" {
				urlValue = "https://www.douyin.com/user/u2"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": urlValue})
		case "/douyin/detail":
			id := strings.TrimSpace(asString(payload["detail_id"]))
			if id == "upstreamErr" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("boom"))
				return
			}
			var data any
			switch id {
			case "missing":
				data = map[string]any{"desc": "", "type": "", "static_cover": "", "downloads": []any{}}
			case "fallback":
				data = map[string]any{"desc": "", "type": "", "static_cover": "", "download": "https://d1"}
			case "titled":
				data = map[string]any{"desc": "title", "type": "video", "static_cover": "https://c", "downloads": "https://d1"}
			default:
				data = map[string]any{"desc": "", "type": "", "static_cover": "", "downloads": []any{"https://d1"}}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "data": data})
		case "/douyin/account/page":
			if !asBool(payload["source"]) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("expected source=true"))
				return
			}
			id := strings.TrimSpace(asString(payload["sec_user_id"]))
			if id == "nil" {
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "x", "data": nil})
				return
			}
			if id == "list" {
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "data": []any{}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "ok",
				"data": map[string]any{
					"items":       []any{},
					"next_cursor": payload["cursor"],
					"has_more":    false,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := &DouyinDownloaderService{
		api:             NewTikTokDownloaderClient(srv.URL, "", srv.Client()),
		cache:           newLRUCache(10, time.Second),
		upstreamTimeout: time.Second,
		defaultCookie:   "c",
		defaultProxy:    "p",
	}

	// ResolveDetailID direct
	if id, resolved, err := svc.ResolveDetailID(context.Background(), "https://www.douyin.com/video/999", ""); err != nil || id != "999" || resolved != "" {
		t.Fatalf("id=%q resolved=%q err=%v", id, resolved, err)
	}
	// ResolveDetailID via share
	if id, resolved, err := svc.ResolveDetailID(context.Background(), "noid", ""); err == nil || id != "" || resolved != "https://example.com" {
		t.Fatalf("id=%q resolved=%q err=%v", id, resolved, err)
	}
	if id, resolved, err := svc.ResolveDetailID(context.Background(), "ok2", ""); err != nil || id != "124" || resolved != "https://www.douyin.com/video/124" {
		t.Fatalf("id=%q resolved=%q err=%v", id, resolved, err)
	}
	if _, _, err := svc.ResolveDetailID(context.Background(), "err", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := svc.ResolveDetailID(context.Background(), "empty", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := svc.ResolveDetailID(context.Background(), "", ""); err == nil {
		t.Fatalf("expected error")
	}

	// ResolveSecUserID direct
	if id, resolved, err := svc.ResolveSecUserID(context.Background(), "https://www.douyin.com/user/u1", ""); err != nil || id != "u1" || resolved != "" {
		t.Fatalf("id=%q resolved=%q err=%v", id, resolved, err)
	}
	if id, resolved, err := svc.ResolveSecUserID(context.Background(), "secok", ""); err != nil || id != "u2" || resolved != "https://www.douyin.com/user/u2" {
		t.Fatalf("id=%q resolved=%q err=%v", id, resolved, err)
	}
	if _, _, err := svc.ResolveSecUserID(context.Background(), "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := svc.ResolveSecUserID(context.Background(), "empty", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := svc.ResolveSecUserID(context.Background(), "noid", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := svc.ResolveSecUserID(context.Background(), "err", ""); err == nil {
		t.Fatalf("expected error")
	}

	// FetchDetail errors + fallback
	if _, err := svc.FetchDetail(context.Background(), "", "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := svc.FetchDetail(context.Background(), "upstreamErr", "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := svc.FetchDetail(context.Background(), "missing", "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if got, err := svc.FetchDetail(context.Background(), "fallback", "", ""); err != nil || len(got.Downloads) != 1 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if got, err := svc.FetchDetail(context.Background(), "titled", "", ""); err != nil || got.Title != "title" || got.CoverURL != "https://c" {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if got, err := svc.FetchDetail(context.Background(), "ok", "", ""); err != nil || len(got.Downloads) != 1 {
		t.Fatalf("got=%v err=%v", got, err)
	}

	// FetchAccount defaults
	if _, err := svc.FetchAccount(context.Background(), "", "", "", "", -1, 0); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := svc.FetchAccount(context.Background(), "nil", "", "", "", 0, 18); err == nil {
		t.Fatalf("expected error")
	}
	if got, err := svc.FetchAccount(context.Background(), "list", "", "", "", 0, 18); err != nil || got == nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if got, err := svc.FetchAccount(context.Background(), "u1", "", "", "", -1, 0); err != nil || got == nil {
		t.Fatalf("got=%v err=%v", got, err)
	}

	key := svc.CacheDetail(&douyinCachedDetail{DetailID: "x", Downloads: []string{"d"}})
	if strings.TrimSpace(key) == "" {
		t.Fatalf("expected key")
	}
	if detail, ok := svc.GetCachedDetail(key); !ok || detail.DetailID != "x" {
		t.Fatalf("detail=%v ok=%v", detail, ok)
	}
	if _, ok := svc.GetCachedDetail(" "); ok {
		t.Fatalf("expected miss")
	}
}

func TestAsStringAndExtractStringSlice(t *testing.T) {
	if got := asString(nil); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := asString("x"); got != "x" {
		t.Fatalf("got=%q", got)
	}
	if got := asString(bytes.NewBufferString("x")); got != "x" {
		t.Fatalf("got=%q", got)
	}
	if got := asString(1); got != "" {
		t.Fatalf("got=%q", got)
	}

	if got := extractStringSlice(nil); got != nil {
		t.Fatalf("got=%v", got)
	}
	if got := extractStringSlice(" "); got != nil {
		t.Fatalf("got=%v", got)
	}
	if got := extractStringSlice(" x "); len(got) != 1 || got[0] != "x" {
		t.Fatalf("got=%v", got)
	}
	if got := extractStringSlice([]any{" x ", nil, " "}); len(got) != 1 || got[0] != "x" {
		t.Fatalf("got=%v", got)
	}
	if got := extractStringSlice([]string{" x ", " "}); len(got) != 1 || got[0] != "x" {
		t.Fatalf("got=%v", got)
	}
	if got := extractStringSlice(1); got != nil {
		t.Fatalf("got=%v", got)
	}
}

func TestTikTokDownloaderClient_postJSON_MarshalErrorIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if len(b) != 0 {
			t.Fatalf("expected empty body, got %q", string(b))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "ok", "url": "x"})
	}))
	t.Cleanup(srv.Close)

	c := NewTikTokDownloaderClient(srv.URL, "", srv.Client())
	var out tikTokDownloaderURLResponse
	if err := c.postJSON(t.Context(), "/douyin/share", func() {}, &out); err != nil {
		t.Fatalf("err=%v", err)
	}
	if out.URL != "x" {
		t.Fatalf("out=%v", out)
	}
}

func TestDouyinDownloaderService_ConfiguredGuard(t *testing.T) {
	s := &DouyinDownloaderService{api: NewTikTokDownloaderClient("", "", &http.Client{})}
	if _, _, err := s.ResolveDetailID(context.Background(), "x", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := s.ResolveSecUserID(context.Background(), "x", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := s.FetchDetail(context.Background(), "x", "", ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := s.FetchAccount(context.Background(), "x", "", "", "", 0, 0); err == nil {
		t.Fatalf("expected error")
	}
	if got := s.CacheDetail(nil); got != "" {
		t.Fatalf("got=%q", got)
	}
	if _, ok := s.GetCachedDetail("x"); ok {
		t.Fatalf("expected miss")
	}

	var nilSvc *DouyinDownloaderService
	if got := nilSvc.CacheDetail(&douyinCachedDetail{DetailID: "x"}); got != "" {
		t.Fatalf("got=%q", got)
	}
}

func TestDouyinDownloaderService_GetCachedDetail_TypeMismatch(t *testing.T) {
	s := &DouyinDownloaderService{cache: newLRUCache(10, time.Second)}
	s.cache.Set("k", "not-struct")
	if _, ok := s.GetCachedDetail("k"); ok {
		t.Fatalf("expected miss")
	}
}

func TestTikTokDownloaderClient_postJSON_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewTikTokDownloaderClient("http://example.com", "", &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			select {
			case <-r.Context().Done():
				return nil, r.Context().Err()
			default:
				return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(`{}`))}, nil
			}
		}),
	})

	if err := c.postJSON(ctx, "/x", map[string]any{"a": 1}, &map[string]any{}); err == nil {
		t.Fatalf("expected error")
	}
}
