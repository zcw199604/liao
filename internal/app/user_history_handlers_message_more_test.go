package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleGetMessageHistory_CoversRemainingBranches(t *testing.T) {
	t.Run("redis get error disables cache", func(t *testing.T) {
		upstreamBody := `{"code":0,"contents_list":[]}`
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamMsgURL {
					return newTextResponse(http.StatusOK, upstreamBody), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		app := &App{httpClient: client, chatHistoryCache: &errChatHistoryCache{}}
		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		form.Set("isFirst", "1")
		form.Set("firstTid", "0")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("upstream fail without cache returns 500 and cookie set", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if got := r.Header.Get("Cookie"); got != "c=1" {
					t.Fatalf("cookie=%q", got)
				}
				return newTextResponse(http.StatusBadGateway, "bad"), nil
			}),
		}
		app := &App{httpClient: client}
		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		form.Set("isFirst", "1")
		form.Set("firstTid", "0")
		form.Set("cookieData", "c=1")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("upstream fail with cachedMessages adjusts cacheTo when userToID==fromUserID", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamMsgURL {
					return newTextResponse(http.StatusBadGateway, "bad"), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		historyCache := &stubChatHistoryCache{
			messages: []map[string]any{
				{"Tid": "9", "id": "u2", "toid": "u3", "content": "cached", "time": "t9"},
			},
		}
		app := &App{httpClient: client, userInfoCache: cache, chatHistoryCache: historyCache}

		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		form.Set("isFirst", "1")
		form.Set("firstTid", "0")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)

		msg := cache.GetLastMessage("me", "u2")
		if msg == nil || msg.FromUserID != "u2" || msg.ToUserID != "me" {
			t.Fatalf("msg=%v", msg)
		}
	})

	t.Run("upstream fail with cachedMessages adjusts cacheFrom when userToID==toUserID", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamMsgURL {
					return newTextResponse(http.StatusBadGateway, "bad"), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		historyCache := &stubChatHistoryCache{
			messages: []map[string]any{
				{"Tid": "9", "id": "u2", "toid": "u3", "content": "cached", "time": "t9"},
			},
		}
		app := &App{httpClient: client, userInfoCache: cache, chatHistoryCache: historyCache}

		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u3")
		form.Set("isFirst", "1")
		form.Set("firstTid", "0")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)

		msg := cache.GetLastMessage("me", "u3")
		if msg == nil || msg.FromUserID != "me" || msg.ToUserID != "u3" {
			t.Fatalf("msg=%v", msg)
		}
	})

	t.Run("upstream ok but empty body", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, ""), nil
			}),
		}
		app := &App{httpClient: client}
		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("needParse false passthrough", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, `{"code":0}`), nil
			}),
		}
		app := &App{httpClient: client}
		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != `{"code":0}` {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})

	t.Run("json unmarshal error", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, "{"), nil
			}),
		}
		app := &App{httpClient: client, userInfoCache: NewMemoryUserInfoCacheService()}
		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "{" {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})

	t.Run("contents_list not array fallback", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, `{"code":0,"contents_list":"x"}`), nil
			}),
		}
		app := &App{httpClient: client, userInfoCache: NewMemoryUserInfoCacheService()}
		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) == "" {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})

	t.Run("old format array enriched", func(t *testing.T) {
		upstreamBody := `[{"userid":"u2"}]`
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		cache.SaveUserInfo(CachedUserInfo{UserID: "u2", Nickname: "Bob"})
		app := &App{httpClient: client, userInfoCache: cache}

		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)

		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal: %v body=%q", err, rr.Body.String())
		}
		if len(list) != 1 || toString(list[0]["nickname"]) != "Bob" {
			t.Fatalf("list=%v", list)
		}
	})

	t.Run("fallback when no contents_list", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, `{"foo":1}`), nil
			}),
		}
		app := &App{httpClient: client, userInfoCache: NewMemoryUserInfoCacheService()}
		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != `{"foo":1}` {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})

	t.Run("save last message adjusts cacheTo when userToID==fromUserID", func(t *testing.T) {
		body := `{"code":0,"contents_list":[{"Tid":"1","id":"u2","toid":"u3","content":"hi","time":"t1"}]}`
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, body), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		app := &App{httpClient: client, userInfoCache: cache}

		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u2")
		form.Set("isFirst", "1")
		form.Set("firstTid", "0")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)

		msg := cache.GetLastMessage("me", "u2")
		if msg == nil || msg.ToUserID != "me" || msg.FromUserID != "u2" {
			t.Fatalf("msg=%v", msg)
		}
	})

	t.Run("save last message adjusts cacheFrom when userToID==toUserID", func(t *testing.T) {
		body := `{"code":0,"contents_list":[{"Tid":"1","id":"u2","toid":"u3","content":"hi","time":"t1"}]}`
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusOK, body), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		app := &App{httpClient: client, userInfoCache: cache}

		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("UserToID", "u3")
		form.Set("isFirst", "1")
		form.Set("firstTid", "0")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
		rr := httptest.NewRecorder()
		app.handleGetMessageHistory(rr, req)

		msg := cache.GetLastMessage("me", "u3")
		if msg == nil || msg.FromUserID != "me" || msg.ToUserID != "u3" {
			t.Fatalf("msg=%v", msg)
		}
	})
}

func TestInferMessageType_CoversRemainingBranches(t *testing.T) {
	if inferMessageType("") != "text" {
		t.Fatalf("expected text")
	}
	if inferMessageType("[notclosed") != "text" {
		t.Fatalf("expected text")
	}
}
