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

type archiveSpy struct {
	persistCalls []archivePersistCall
	mergeCalls   []archiveMergeCall
	touchCalls   [][2]string
	saveCalls    []archiveSaveCall
	deleteCalls  [][2]string
	mergeFn      func(ownerUserID string, upstream []map[string]any, source UserArchiveListSource) []map[string]any
}

type archivePersistCall struct {
	ownerUserID string
	source      UserArchiveListSource
	users       []map[string]any
}

type archiveMergeCall struct {
	ownerUserID string
	source      UserArchiveListSource
	upstream    []map[string]any
}

type archiveSaveCall struct {
	ownerUserID  string
	targetUserID string
	content      string
	messageTime  string
}

func (s *archiveSpy) PersistUserList(_ context.Context, ownerUserID string, users []map[string]any, source UserArchiveListSource) {
	copied := make([]map[string]any, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		item := make(map[string]any, len(user))
		for k, v := range user {
			item[k] = v
		}
		copied = append(copied, item)
	}
	s.persistCalls = append(s.persistCalls, archivePersistCall{
		ownerUserID: ownerUserID,
		source:      source,
		users:       copied,
	})
}

func (s *archiveSpy) MergeArchivedUsers(_ context.Context, ownerUserID string, upstream []map[string]any, source UserArchiveListSource) []map[string]any {
	s.mergeCalls = append(s.mergeCalls, archiveMergeCall{
		ownerUserID: ownerUserID,
		source:      source,
		upstream:    upstream,
	})
	if s.mergeFn != nil {
		return s.mergeFn(ownerUserID, upstream, source)
	}
	return upstream
}

func (s *archiveSpy) TouchConversation(_ context.Context, ownerUserID, targetUserID string) {
	s.touchCalls = append(s.touchCalls, [2]string{ownerUserID, targetUserID})
}

func (s *archiveSpy) SaveLastMessage(_ context.Context, ownerUserID, targetUserID, content, messageTime string) {
	s.saveCalls = append(s.saveCalls, archiveSaveCall{
		ownerUserID:  ownerUserID,
		targetUserID: targetUserID,
		content:      content,
		messageTime:  messageTime,
	})
}

func (s *archiveSpy) DeleteConversation(_ context.Context, ownerUserID, targetUserID string) {
	s.deleteCalls = append(s.deleteCalls, [2]string{ownerUserID, targetUserID})
}

func TestHandleGetHistoryUserList_ArchiveFallbackAndMerge(t *testing.T) {
	t.Run("merge archived users into upstream list", func(t *testing.T) {
		upstreamBody := `[{"id":"u1","nickname":"A"}]`
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == upstreamHistoryURL {
					return newTextResponse(http.StatusOK, upstreamBody), nil
				}
				return newTextResponse(http.StatusNotFound, "no"), nil
			}),
		}
		spy := &archiveSpy{
			mergeFn: func(_ string, upstream []map[string]any, _ UserArchiveListSource) []map[string]any {
				return append(upstream, map[string]any{"id": "u2", "nickname": "B", "localArchived": true})
			},
		}
		app := &App{httpClient: client, userArchive: spy}

		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetHistoryUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d", rr.Code)
		}

		if len(spy.persistCalls) != 1 {
			t.Fatalf("persist calls=%d", len(spy.persistCalls))
		}
		if spy.persistCalls[0].source != UserArchiveListSourceHistory {
			t.Fatalf("persist source=%s", spy.persistCalls[0].source)
		}

		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("len=%d list=%v", len(list), list)
		}
		if toString(list[1]["id"]) != "u2" || list[1]["localArchived"] != true {
			t.Fatalf("list[1]=%v", list[1])
		}
	})

	t.Run("upstream fail returns archived list", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("boom")
			}),
		}
		spy := &archiveSpy{
			mergeFn: func(_ string, upstream []map[string]any, _ UserArchiveListSource) []map[string]any {
				if len(upstream) != 0 {
					t.Fatalf("upstream should be empty on fallback, got %d", len(upstream))
				}
				return []map[string]any{{"id": "u9", "localArchived": true}}
			},
		}
		app := &App{httpClient: client, userArchive: spy}

		req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getHistoryUserList", url.Values{"myUserID": []string{"me"}})
		rr := httptest.NewRecorder()
		app.handleGetHistoryUserList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}

		var list []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(list) != 1 || toString(list[0]["id"]) != "u9" {
			t.Fatalf("list=%v", list)
		}
	})
}

func TestHandleGetFavoriteUserList_ArchiveMerge(t *testing.T) {
	upstreamBody := `[{"id":"u1"}]`
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == upstreamFavoriteURL {
				return newTextResponse(http.StatusOK, upstreamBody), nil
			}
			return newTextResponse(http.StatusNotFound, "no"), nil
		}),
	}
	spy := &archiveSpy{
		mergeFn: func(_ string, upstream []map[string]any, source UserArchiveListSource) []map[string]any {
			if source != UserArchiveListSourceFavorite {
				t.Fatalf("source=%s", source)
			}
			return append(upstream, map[string]any{"id": "f2", "localArchived": true})
		},
	}
	app := &App{httpClient: client, userArchive: spy}

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
	if len(list) != 2 || toString(list[1]["id"]) != "f2" {
		t.Fatalf("list=%v", list)
	}
	if len(spy.persistCalls) != 1 || spy.persistCalls[0].source != UserArchiveListSourceFavorite {
		t.Fatalf("persistCalls=%v", spy.persistCalls)
	}
}

func TestHandleGetMessageHistory_SyncsArchiveLastMessage(t *testing.T) {
	body := `{"code":0,"contents_list":[{"Tid":"1","id":"u2","toid":"u3","content":"hi","time":"t1"}]}`
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return newTextResponse(http.StatusOK, body), nil
		}),
	}
	spy := &archiveSpy{}
	app := &App{httpClient: client, userArchive: spy, userInfoCache: NewMemoryUserInfoCacheService()}

	form := url.Values{}
	form.Set("myUserID", "me")
	form.Set("UserToID", "u2")
	form.Set("isFirst", "1")
	form.Set("firstTid", "0")
	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getMessageHistory", form)
	rr := httptest.NewRecorder()
	app.handleGetMessageHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}

	if len(spy.touchCalls) == 0 {
		t.Fatalf("expected touch call")
	}
	if len(spy.saveCalls) != 1 {
		t.Fatalf("saveCalls=%d", len(spy.saveCalls))
	}
	if spy.saveCalls[0].ownerUserID != "me" || spy.saveCalls[0].targetUserID != "u2" || spy.saveCalls[0].content != "hi" || spy.saveCalls[0].messageTime != "t1" {
		t.Fatalf("save=%+v", spy.saveCalls[0])
	}
}
