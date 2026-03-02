package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHandleGetFavoriteUserList_ArchiveFallbackOnUpstreamError(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("upstream boom")
		}),
	}
	spy := &archiveSpy{
		mergeFn: func(_ string, upstream []map[string]any, source UserArchiveListSource) []map[string]any {
			if source != UserArchiveListSourceFavorite {
				t.Fatalf("source=%s", source)
			}
			if upstream != nil {
				t.Fatalf("upstream should be nil on fallback, got=%v", upstream)
			}
			return []map[string]any{{"id": "fallback-u1", "nickname": "fallback"}}
		},
	}
	app := &App{httpClient: client, userArchive: spy}

	req := newURLEncodedRequest(t, http.MethodPost, "http://example.com/api/getFavoriteUserList", url.Values{"myUserID": []string{"me"}})
	rr := httptest.NewRecorder()
	app.handleGetFavoriteUserList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}

	var list []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal err=%v", err)
	}
	if len(list) != 1 || toString(list[0]["id"]) != "fallback-u1" {
		t.Fatalf("list=%v", list)
	}
}
