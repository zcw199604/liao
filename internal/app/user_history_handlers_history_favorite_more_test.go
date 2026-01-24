package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleGetHistoryUserList_CoversRemainingBranches(t *testing.T) {
	t.Run("cookie and postForm error", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.Host != "v1.chat2019.cn" {
					t.Fatalf("host=%q", r.Host)
				}
				if got := r.Header.Get("Cookie"); got != "c=1" {
					t.Fatalf("cookie=%q", got)
				}
				return nil, errors.New("boom")
			}),
		}
		app := &App{httpClient: client}

		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("cookieData", "c=1")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", form)
		rr := httptest.NewRecorder()
		app.handleGetHistoryUserList(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("json unmarshal error", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamHistoryURL {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader("not-json")),
						Header:     make(http.Header),
						Request:    r,
					}, nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		app := &App{httpClient: client, userInfoCache: NewMemoryUserInfoCacheService()}
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetHistoryUserList(rr, req)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "not-json" {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})

	t.Run("idKey=UserID", func(t *testing.T) {
		upstreamBody := `[{"UserID":"u2"}]`
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamHistoryURL {
					return newTextResponse(http.StatusOK, upstreamBody), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		cache.SaveUserInfo(CachedUserInfo{UserID: "u2", Nickname: "Bob"})
		app := &App{httpClient: client, userInfoCache: cache}

		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetHistoryUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(list) != 1 || toString(list[0]["nickname"]) != "Bob" {
			t.Fatalf("list=%v", list)
		}
	})

	t.Run("idKey=userid", func(t *testing.T) {
		upstreamBody := `[{"userid":"u2"}]`
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamHistoryURL {
					return newTextResponse(http.StatusOK, upstreamBody), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		cache.SaveUserInfo(CachedUserInfo{UserID: "u2", Nickname: "Bob"})
		app := &App{httpClient: client, userInfoCache: cache}

		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetHistoryUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(list) != 1 || toString(list[0]["nickname"]) != "Bob" {
			t.Fatalf("list=%v", list)
		}
	})
}

func TestHandleGetFavoriteUserList_CoversRemainingBranches(t *testing.T) {
	t.Run("cookie and postForm error", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if got := r.Header.Get("Cookie"); got != "c=1" {
					t.Fatalf("cookie=%q", got)
				}
				return nil, errors.New("boom")
			}),
		}
		app := &App{httpClient: client}

		form := url.Values{}
		form.Set("myUserID", "me")
		form.Set("cookieData", "c=1")
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", form)
		rr := httptest.NewRecorder()
		app.handleGetFavoriteUserList(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("upstream non-OK status", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return newTextResponse(http.StatusBadGateway, "bad"), nil
			}),
		}
		app := &App{httpClient: client}
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetFavoriteUserList(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("cache enabled enrich idKey=userid", func(t *testing.T) {
		upstreamBody := `[{"userid":"u2"}]`
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamFavoriteURL {
					return newTextResponse(http.StatusOK, upstreamBody), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		cache.SaveUserInfo(CachedUserInfo{UserID: "u2", Nickname: "Bob"})
		app := &App{httpClient: client, userInfoCache: cache}

		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetFavoriteUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(list) != 1 || toString(list[0]["nickname"]) != "Bob" {
			t.Fatalf("list=%v", list)
		}
	})

	t.Run("cache enabled enrich idKey=UserID", func(t *testing.T) {
		upstreamBody := `[{"UserID":"u2"}]`
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamFavoriteURL {
					return newTextResponse(http.StatusOK, upstreamBody), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		cache := NewMemoryUserInfoCacheService()
		cache.SaveUserInfo(CachedUserInfo{UserID: "u2", Nickname: "Bob"})
		app := &App{httpClient: client, userInfoCache: cache}

		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetFavoriteUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}
		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(list) != 1 || toString(list[0]["nickname"]) != "Bob" {
			t.Fatalf("list=%v", list)
		}
	})

	t.Run("json unmarshal error", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamFavoriteURL {
					return newTextResponse(http.StatusOK, "not-json"), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		app := &App{httpClient: client, userInfoCache: NewMemoryUserInfoCacheService()}
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetFavoriteUserList(rr, req)
		if rr.Code != http.StatusOK || strings.TrimSpace(rr.Body.String()) != "not-json" {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})
}

