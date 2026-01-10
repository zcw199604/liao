package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

