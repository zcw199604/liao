package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type archiveFallbackStub struct {
	merged []map[string]any
}

func (s *archiveFallbackStub) PersistUserList(context.Context, string, []map[string]any, UserArchiveListSource) {
}
func (s *archiveFallbackStub) MergeArchivedUsers(_ context.Context, _ string, _ []map[string]any, _ UserArchiveListSource) []map[string]any {
	return s.merged
}
func (s *archiveFallbackStub) TouchConversation(context.Context, string, string) {}
func (s *archiveFallbackStub) SaveLastMessage(context.Context, string, string, string, string) {
}
func (s *archiveFallbackStub) DeleteConversation(context.Context, string, string) {}

func TestUserHistoryHandlers_ArchiveFallbackOnUpstreamNonOK(t *testing.T) {
	archive := &archiveFallbackStub{merged: []map[string]any{{"id": "u1", "nickname": "archived"}}}
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return newTextResponse(http.StatusBadGateway, "bad gateway"), nil
	})}

	t.Run("history fallback", func(t *testing.T) {
		app := &App{httpClient: client, userArchive: archive}
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetHistoryUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil || len(list) != 1 || toString(list[0]["id"]) != "u1" {
			t.Fatalf("fallback list=%v err=%v", list, err)
		}
	})

	t.Run("favorite fallback", func(t *testing.T) {
		app := &App{httpClient: client, userArchive: archive}
		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetFavoriteUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil || len(list) != 1 || toString(list[0]["id"]) != "u1" {
			t.Fatalf("fallback list=%v err=%v", list, err)
		}
	})
}

func TestUserHistoryHelpers_UncoveredBranches(t *testing.T) {
	if got := detectUserListIDKey(nil); got != "id" {
		t.Fatalf("detect nil list=%q", got)
	}
	if got := detectUserListIDKey([]map[string]any{{"name": "x"}}); got != "id" {
		t.Fatalf("detect unknown key=%q", got)
	}

	app := &App{httpClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return newTextResponse(http.StatusOK, "ok"), nil
	})}}
	app.persistHistoryLastMessage("me", "u1", nil)
	app.persistHistoryLastMessage("me", "u1", map[string]any{"id": "u1"})

	_, err := app.uploadBytesToUpstream(context.Background(), "http://example.com/upload", "example.com:9003", "bad\r\nname.png", []byte("x"), "", "r", "ua")
	if err == nil {
		// fallback branch: ensure we still cover NewRequestWithContext error path when URL is bad.
		_, err = app.uploadBytesToUpstream(context.Background(), "http://[::1", "example.com:9003", "a.png", []byte("x"), "", "r", "ua")
	}
	if err == nil {
		t.Fatalf("expected uploadBytesToUpstream to return error branch")
	}

	// Ensure error propagation branch also works when request build is ok but transport fails.
	app.httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}
	if _, err := app.uploadBytesToUpstream(context.Background(), "http://example.com/upload", "example.com:9003", "a.jpg", []byte("x"), "", "r", "ua"); err == nil {
		t.Fatalf("expected transport error")
	}
}
