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
	listCalls    []archiveListCandidatesCall
	touchCalls   [][2]string
	saveCalls    []archiveSaveCall
	deleteCalls  [][2]string
	mergeFn      func(ownerUserID string, upstream []map[string]any, source UserArchiveListSource) []map[string]any
	listFn       func(ownerUserID string, limit int) ([]ContactCandidate, error)
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

type archiveListCandidatesCall struct {
	ownerUserID string
	limit       int
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

func (s *archiveSpy) ListContactCandidates(_ context.Context, ownerUserID string, limit int) ([]ContactCandidate, error) {
	s.listCalls = append(s.listCalls, archiveListCandidatesCall{ownerUserID: ownerUserID, limit: limit})
	if s.listFn != nil {
		return s.listFn(ownerUserID, limit)
	}
	return nil, nil
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

func TestHandleGetContactCandidates(t *testing.T) {
	t.Run("archive only returns local candidates", func(t *testing.T) {
		spy := &archiveSpy{
			listFn: func(ownerUserID string, limit int) ([]ContactCandidate, error) {
				if ownerUserID != "source-a" {
					t.Fatalf("ownerUserID=%q", ownerUserID)
				}
				if limit != 50 {
					t.Fatalf("limit=%d", limit)
				}
				return []ContactCandidate{{
					TargetUserID:  "target-1",
					Nickname:      "Local Target",
					Sources:       []string{"archive"},
					LocalArchived: true,
				}}, nil
			},
		}
		app := &App{httpClient: http.DefaultClient, userArchive: spy}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/chat/contactCandidates?sourceIdentityId=source-a&includeUpstream=0&limit=50", nil)
		rr := httptest.NewRecorder()
		app.handleGetContactCandidates(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		data := resp["data"].(map[string]any)
		items := data["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("items=%v", items)
		}
		first := items[0].(map[string]any)
		if first["targetUserId"] != "target-1" || first["localArchived"] != true {
			t.Fatalf("first=%v", first)
		}
		if len(spy.persistCalls) != 0 {
			t.Fatalf("persistCalls=%v", spy.persistCalls)
		}
	})

	t.Run("merge upstream and archive dedup by target", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				switch r.URL.String() {
				case upstreamHistoryURL:
					return newTextResponse(http.StatusOK, `[{"id":"target-1","nickname":"History One","cookieData":"secret"}]`), nil
				case upstreamFavoriteURL:
					return newTextResponse(http.StatusOK, `[{"id":"target-1","nickname":"Favorite One"},{"id":"target-2","nickname":"Favorite Two"}]`), nil
				default:
					return newTextResponse(http.StatusNotFound, "no"), nil
				}
			}),
		}
		spy := &archiveSpy{
			listFn: func(string, int) ([]ContactCandidate, error) {
				return []ContactCandidate{{
					TargetUserID:  "target-3",
					Nickname:      "Archived Three",
					Sources:       []string{"archive"},
					LocalArchived: true,
				}}, nil
			},
		}
		app := &App{httpClient: client, userArchive: spy}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/chat/contactCandidates?sourceIdentityId=source-a&limit=10", nil)
		rr := httptest.NewRecorder()
		app.handleGetContactCandidates(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}

		var resp struct {
			Code int `json:"code"`
			Data struct {
				Items []ContactCandidate `json:"items"`
			} `json:"data"`
			Warnings []string `json:"warnings"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Code != 0 {
			t.Fatalf("code=%d", resp.Code)
		}
		if len(resp.Warnings) != 0 {
			t.Fatalf("warnings=%v", resp.Warnings)
		}
		if len(resp.Data.Items) != 3 {
			t.Fatalf("items=%+v", resp.Data.Items)
		}
		if resp.Data.Items[0].TargetUserID != "target-1" {
			t.Fatalf("first=%+v", resp.Data.Items[0])
		}
		if len(resp.Data.Items[0].Sources) != 2 {
			t.Fatalf("sources=%v", resp.Data.Items[0].Sources)
		}
		if _, ok := resp.Data.Items[0].Snapshot["cookieData"]; ok {
			t.Fatalf("sensitive snapshot leaked: %v", resp.Data.Items[0].Snapshot)
		}
		if len(spy.persistCalls) != 2 {
			t.Fatalf("persistCalls=%v", spy.persistCalls)
		}
	})

	t.Run("upstream failure degrades to archive with warnings", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("boom")
			}),
		}
		spy := &archiveSpy{
			listFn: func(string, int) ([]ContactCandidate, error) {
				return []ContactCandidate{{TargetUserID: "target-9", Nickname: "Archived", Sources: []string{"archive"}, LocalArchived: true}}, nil
			},
		}
		app := &App{httpClient: client, userArchive: spy}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/chat/contactCandidates?sourceIdentityId=source-a", nil)
		rr := httptest.NewRecorder()
		app.handleGetContactCandidates(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}

		var resp struct {
			Data struct {
				Items []ContactCandidate `json:"items"`
			} `json:"data"`
			Warnings []string `json:"warnings"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(resp.Data.Items) != 1 || resp.Data.Items[0].TargetUserID != "target-9" {
			t.Fatalf("items=%+v", resp.Data.Items)
		}
		if len(resp.Warnings) != 2 {
			t.Fatalf("warnings=%v", resp.Warnings)
		}
	})

	t.Run("invalid source identity", func(t *testing.T) {
		app := &App{httpClient: http.DefaultClient}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api/chat/contactCandidates", nil)
		rr := httptest.NewRecorder()
		app.handleGetContactCandidates(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
		}
	})
}
