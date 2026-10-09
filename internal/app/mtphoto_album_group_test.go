package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetMtPhotoAlbums_IncludesGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "k" {
			t.Fatalf("x-api-key=%q", got)
		}
		switch r.URL.Path {
		case "/api-album":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "name": "a", "count": 1}})
		case "/api-group/album":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 10, "name": "旅行", "weights": 5, "albumIds": []int{2}},
				{"id": 11, "name": "空组", "weights": 1, "albumIds": nil},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	app := &App{mtPhoto: NewMtPhotoService(srv.URL, "k", "/lsp", srv.Client())}
	rr := httptest.NewRecorder()
	app.handleGetMtPhotoAlbums(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbums", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}

	var body struct {
		Data   []MtPhotoAlbum      `json:"data"`
		Groups []MtPhotoAlbumGroup `json:"groups"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Data) != 1 || len(body.Groups) != 2 {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if g := body.Groups[0]; g.ID != 10 || g.Name != "旅行" || g.Weights != 5 || len(g.AlbumIDs) != 1 || g.AlbumIDs[0] != 2 {
		t.Fatalf("group0=%+v", g)
	}
	if body.Groups[1].AlbumIDs == nil {
		t.Fatalf("albumIds 应规范化为空数组")
	}
}

func TestHandleGetMtPhotoAlbums_GroupsUnsupportedDegrades(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api-album" {
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "name": "a"}})
			return
		}
		http.NotFound(w, r) // 旧版上游无 /api-group/album
	}))
	t.Cleanup(srv.Close)

	app := &App{mtPhoto: NewMtPhotoService(srv.URL, "k", "/lsp", srv.Client())}
	rr := httptest.NewRecorder()
	app.handleGetMtPhotoAlbums(rr, httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoAlbums", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var body struct {
		Data   []MtPhotoAlbum      `json:"data"`
		Groups []MtPhotoAlbumGroup `json:"groups"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Data) != 1 || body.Groups == nil || len(body.Groups) != 0 {
		t.Fatalf("body=%s", rr.Body.String())
	}
}
