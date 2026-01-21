package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTextResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type stubChatHistoryCache struct {
	mu       sync.Mutex
	messages []map[string]any
	saved    []map[string]any
}

func (s *stubChatHistoryCache) SaveMessages(ctx context.Context, conversationKey string, messages []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = append(s.saved, messages...)
}

func (s *stubChatHistoryCache) GetMessages(ctx context.Context, conversationKey string, beforeTid string, limit int) ([]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]map[string]any, 0, len(s.messages))
	for _, m := range s.messages {
		out = append(out, m)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func TestHandleGetHistoryUserList_PassthroughWhenCacheDisabled(t *testing.T) {
	upstreamBody := `[{"id":"u2"}]`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamHistoryURL {
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	app := &App{httpClient: client}

	form := url.Values{}
	form.Set("myUserID", "me")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", form)
	rr := httptest.NewRecorder()

	app.handleGetHistoryUserList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != upstreamBody {
		t.Fatalf("body=%q, want %q", got, upstreamBody)
	}
}

func TestHandleGetHistoryUserList_EnrichesWhenCacheEnabled(t *testing.T) {
	upstreamBody := `[{"id":"u2"}]`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamHistoryURL {
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	cache := NewMemoryUserInfoCacheService()
	cache.SaveUserInfo(CachedUserInfo{
		UserID:   "u2",
		Nickname: "Bob",
		Gender:   "男",
		Age:      "20",
		Address:  "BJ",
	})
	cache.SaveLastMessage(CachedLastMessage{
		FromUserID: "me",
		ToUserID:   "u2",
		Content:    "hello",
		Type:       "text",
		Time:       "t1",
	})

	app := &App{httpClient: client, userInfoCache: cache}

	form := url.Values{}
	form.Set("myUserID", "me")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", form)
	rr := httptest.NewRecorder()

	app.handleGetHistoryUserList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var list []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len=%d, want 1", len(list))
	}
	item := list[0]
	if got, _ := item["nickname"].(string); got != "Bob" {
		t.Fatalf("nickname=%q, want %q", got, "Bob")
	}
	if got, _ := item["lastMsg"].(string); got != "我: hello" {
		t.Fatalf("lastMsg=%q, want %q", got, "我: hello")
	}
	if got, _ := item["lastTime"].(string); got != "t1" {
		t.Fatalf("lastTime=%q, want %q", got, "t1")
	}
}

func TestHandleGetHistoryUserList_InlineMediaLastMsgPreview(t *testing.T) {
	upstreamBody := `[{"id":"u2"}]`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamHistoryURL {
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	cache := NewMemoryUserInfoCacheService()
	cache.SaveUserInfo(CachedUserInfo{
		UserID:   "u2",
		Nickname: "Bob",
		Gender:   "男",
		Age:      "20",
		Address:  "BJ",
	})
	cache.SaveLastMessage(CachedLastMessage{
		FromUserID: "me",
		ToUserID:   "u2",
		Content:    "喜欢吗[20260104/image.jpg]",
		Type:       "text",
		Time:       "t1",
	})

	app := &App{httpClient: client, userInfoCache: cache}

	form := url.Values{}
	form.Set("myUserID", "me")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", form)
	rr := httptest.NewRecorder()

	app.handleGetHistoryUserList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var list []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len=%d, want 1", len(list))
	}
	item := list[0]
	if got, _ := item["lastMsg"].(string); got != "我: 喜欢吗 [图片]" {
		t.Fatalf("lastMsg=%q, want %q", got, "我: 喜欢吗 [图片]")
	}
}

func TestHandleGetHistoryUserList_Returns500WhenUpstreamNonOK(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamHistoryURL {
				return newTextResponse(http.StatusBadGateway, "bad"), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	app := &App{httpClient: client}

	form := url.Values{}
	form.Set("myUserID", "me")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", form)
	rr := httptest.NewRecorder()

	app.handleGetHistoryUserList(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rr.Body.String(), "upstream status") {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestHandleGetFavoriteUserList_PassthroughWhenCacheDisabled(t *testing.T) {
	upstreamBody := `[{"id":"u3"}]`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamFavoriteURL {
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	app := &App{httpClient: client}

	form := url.Values{}
	form.Set("myUserID", "me")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", form)
	rr := httptest.NewRecorder()

	app.handleGetFavoriteUserList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != upstreamBody {
		t.Fatalf("body=%q, want %q", got, upstreamBody)
	}
}

func TestHandleGetMessageHistory_MergesRedisWhenUpstreamOK(t *testing.T) {
	upstreamBody := `{"code":0,"contents_list":[{"Tid":"2","id":"me","toid":"u2","content":"hi","time":"t2"}]}`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamMsgURL {
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	historyCache := &stubChatHistoryCache{
		messages: []map[string]any{
			{"Tid": "1", "id": "u2", "toid": "me", "content": "old", "time": "t1"},
		},
	}

	app := &App{
		httpClient:       client,
		userInfoCache:    NewMemoryUserInfoCacheService(),
		chatHistoryCache: historyCache,
	}

	form := url.Values{}
	form.Set("myUserID", "me")
	form.Set("UserToID", "u2")
	form.Set("isFirst", "1")
	form.Set("firstTid", "0")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
	rr := httptest.NewRecorder()

	app.handleGetMessageHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var obj map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &obj); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if code := toInt(obj["code"]); code != 0 {
		t.Fatalf("code=%d, want 0", code)
	}
	contents, ok := obj["contents_list"].([]any)
	if !ok {
		t.Fatalf("contents_list missing or invalid")
	}
	if len(contents) != 2 {
		t.Fatalf("contents_list len=%d, want 2", len(contents))
	}
	first, _ := contents[0].(map[string]any)
	second, _ := contents[1].(map[string]any)
	if tid := extractHistoryMessageTid(first); tid != "2" {
		t.Fatalf("first tid=%q, want %q", tid, "2")
	}
	if tid := extractHistoryMessageTid(second); tid != "1" {
		t.Fatalf("second tid=%q, want %q", tid, "1")
	}
}

func TestHandleGetMessageHistory_ReturnsRedisWhenUpstreamFails(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamMsgURL {
				return newTextResponse(http.StatusBadGateway, `{"error":"bad"}`), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	historyCache := &stubChatHistoryCache{
		messages: []map[string]any{
			{"Tid": "9", "id": "me", "toid": "u2", "content": "cached", "time": "t9"},
		},
	}

	app := &App{
		httpClient:       client,
		userInfoCache:    NewMemoryUserInfoCacheService(),
		chatHistoryCache: historyCache,
	}

	form := url.Values{}
	form.Set("myUserID", "me")
	form.Set("UserToID", "u2")
	form.Set("isFirst", "1")
	form.Set("firstTid", "0")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
	rr := httptest.NewRecorder()

	app.handleGetMessageHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var obj map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &obj); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if code := toInt(obj["code"]); code != 0 {
		t.Fatalf("code=%d, want 0", code)
	}
	contents, ok := obj["contents_list"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents_list len mismatch: %v", obj["contents_list"])
	}
	first, _ := contents[0].(map[string]any)
	if tid := extractHistoryMessageTid(first); tid != "9" {
		t.Fatalf("tid=%q, want %q", tid, "9")
	}
}

func TestHandleGetMessageHistory_SkipsUpstreamWhenRedisHasEnough(t *testing.T) {
	called := 0
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamMsgURL {
				called++
				return newTextResponse(http.StatusBadGateway, `{"error":"should-not-call"}`), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	msgs := make([]map[string]any, 0, 20)
	for i := 20; i >= 1; i-- {
		msgs = append(msgs, map[string]any{
			"Tid":     toString(i),
			"id":      "me",
			"toid":    "u2",
			"content": "m",
			"time":    "t",
		})
	}

	historyCache := &stubChatHistoryCache{messages: msgs}
	app := &App{
		httpClient:       client,
		userInfoCache:    NewMemoryUserInfoCacheService(),
		chatHistoryCache: historyCache,
	}

	form := url.Values{}
	form.Set("myUserID", "me")
	form.Set("UserToID", "u2")
	form.Set("isFirst", "0")
	form.Set("firstTid", "21")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
	rr := httptest.NewRecorder()

	app.handleGetMessageHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if called != 0 {
		t.Fatalf("upstream called=%d, want 0", called)
	}

	var obj map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &obj); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if code := toInt(obj["code"]); code != 0 {
		t.Fatalf("code=%d, want 0", code)
	}
	contents, ok := obj["contents_list"].([]any)
	if !ok || len(contents) != 20 {
		t.Fatalf("contents_list len mismatch: %v", obj["contents_list"])
	}
}

func TestHandleGetMessageHistory_CallsUpstreamWhenLatestEvenIfRedisHasEnough(t *testing.T) {
	called := 0
	upstreamBody := `{"code":0,"contents_list":[{"Tid":"21","id":"me","toid":"u2","content":"new","time":"t21"}]}`

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamMsgURL {
				called++
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}
			return newTextResponse(http.StatusNotFound, `{"error":"unexpected"}`), nil
		}),
	}

	msgs := make([]map[string]any, 0, 20)
	for i := 20; i >= 1; i-- {
		msgs = append(msgs, map[string]any{
			"Tid":     toString(i),
			"id":      "me",
			"toid":    "u2",
			"content": "m",
			"time":    "t",
		})
	}

	historyCache := &stubChatHistoryCache{messages: msgs}
	app := &App{
		httpClient:       client,
		userInfoCache:    NewMemoryUserInfoCacheService(),
		chatHistoryCache: historyCache,
	}

	form := url.Values{}
	form.Set("myUserID", "me")
	form.Set("UserToID", "u2")
	form.Set("isFirst", "1")
	form.Set("firstTid", "0")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
	rr := httptest.NewRecorder()

	app.handleGetMessageHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if called != 1 {
		t.Fatalf("upstream called=%d, want 1", called)
	}

	var obj map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &obj); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if code := toInt(obj["code"]); code != 0 {
		t.Fatalf("code=%d, want 0", code)
	}
	contents, ok := obj["contents_list"].([]any)
	if !ok || len(contents) != 20 {
		t.Fatalf("contents_list len mismatch: %v", obj["contents_list"])
	}
	first, _ := contents[0].(map[string]any)
	if tid := extractHistoryMessageTid(first); tid != "21" {
		t.Fatalf("first tid=%q, want %q", tid, "21")
	}
}
