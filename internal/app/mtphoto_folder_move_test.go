package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestMtPhotoMoveFolderFiles(t *testing.T) {
	for _, tc := range []struct {
		name, body, code string
		status           int
		ids              []int64
	}{
		{"direct includes files without thumbnails", `{"sourceId":1,"targetId":2}`, "", 200, []int64{11, 12}},
		{"recursive", `{"sourceId":1,"targetId":2,"includeSubfolders":true}`, "", 200, []int64{11, 12, 31}},
		{"same folder", `{"sourceId":1,"targetId":1}`, "", 400, nil},
		{"target inside source", `{"sourceId":1,"targetId":3,"includeSubfolders":true}`, "", 400, nil},
		{"upstream conflict", `{"sourceId":1,"targetId":2}`, "fileConflict", 502, []int64{11, 12}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var moved []int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-api-key") != "key" {
					t.Error("missing API key")
				}
				switch r.URL.Path {
				case "/gateway/foldersV2/1":
					_, _ = w.Write([]byte(`{"path":"/source","folderList":[{"id":3}],"fileList":[{"id":11},{"id":12},{"id":11}]}`))
				case "/gateway/foldersV2/2":
					_, _ = w.Write([]byte(`{"path":"/target","folderList":[],"fileList":[]}`))
				case "/gateway/foldersV2/3":
					_, _ = w.Write([]byte(`{"path":"/source/child","folderList":[],"fileList":[{"id":31}]}`))
				case "/gateway/filePathEdit":
					var body struct {
						Type      string
						FileIDs   []int64 `json:"fileIds"`
						DistID    int     `json:"distId"`
						Overwrite int
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					moved = body.FileIDs
					if body.Type != "move" || body.DistID != 2 || body.Overwrite != 2 {
						t.Errorf("body=%+v", body)
					}
					if tc.code != "" {
						_ = json.NewEncoder(w).Encode(map[string]any{"code": tc.code})
						return
					}
					_, _ = w.Write([]byte(`{"n":1}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			svc := NewMtPhotoService(srv.URL, "key", "", srv.Client())
			svc.albumsCache = []MtPhotoAlbum{{ID: 4}}
			a := &App{mtPhoto: svc}
			w := httptest.NewRecorder()
			a.handleMoveMtPhotoFolderFiles(w, httptest.NewRequest(http.MethodPost, "/api/moveMtPhotoFolderFiles", strings.NewReader(tc.body)))
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if !reflect.DeepEqual(moved, tc.ids) {
				t.Errorf("moved=%v want=%v", moved, tc.ids)
			}
			if len(moved) > 0 && len(svc.albumsCache) > 0 {
				t.Error("stale cache")
			}
		})
	}
}
