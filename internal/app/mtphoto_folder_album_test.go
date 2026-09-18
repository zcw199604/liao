package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCreateMtPhotoFolderAlbum(t *testing.T) {
	for _, tc := range []struct {
		name, body, failPath string
		wantStatus           int
		wantPaths            []string
	}{
		{"multiple folders", `{"name":" Trips ","folders":[{"id":12,"path":"/photos/a"},{"id":34,"path":"/photos/b"},{"id":12,"path":"/photos/a"}]}`, "", 200, []string{"/api-album", "/api-album/link/99", "/api-album/link/99", "/api-album/linkSyncFiles/99"}},
		{"empty selection", `{"name":"Trips","folders":[]}`, "", 400, nil},
		{"empty name", `{"name":"  ","folders":[{"id":12,"path":"/a"}]}`, "", 400, nil},
		{"invalid folder", `{"name":"Trips","folders":[{"id":0,"path":"/a"}]}`, "", 400, nil},
		{"invalid json", `{`, "", 400, nil},
		{"partial failure", `{"name":"Trips","folders":[{"id":12,"path":"/a"}]}`, "/api-album/link/99", 502, []string{"/api-album", "/api-album/link/99"}},
		{"sync failure", `{"name":"Trips","folders":[{"id":12,"path":"/a"}]}`, "/api-album/linkSyncFiles/99", 502, []string{"/api-album", "/api-album/link/99", "/api-album/linkSyncFiles/99"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				if r.Method != http.MethodPost || r.Header.Get("x-api-key") != "test-key" {
					t.Errorf("incorrect method/auth")
				}
				if r.URL.Path == tc.failPath {
					w.WriteHeader(403)
					return
				}
				switch r.URL.Path {
				case "/api-album":
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["name"] != "Trips" {
						t.Errorf("body=%v err=%v", body, err)
					}
					w.WriteHeader(201)
					_, _ = w.Write([]byte(`{"id":99,"name":"Trips","count":0}`))
				case "/api-album/link/99":
					var body struct {
						Type    string
						Value   string
						Exclude bool
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					var value struct {
						ID    int
						Label string
					}
					if err := json.Unmarshal([]byte(body.Value), &value); err != nil {
						t.Fatal(err)
					}
					if body.Type != "folder" || body.Exclude || value.ID <= 0 || !strings.HasPrefix(value.Label, "/") {
						t.Errorf("bad link: %+v %+v", body, value)
					}
					_, _ = w.Write([]byte(`{"n":1}`))
				default:
					_, _ = w.Write([]byte(`{"n":1}`))
				}
			}))
			defer srv.Close()
			svc := NewMtPhotoService(srv.URL, "test-key", "", srv.Client())
			svc.albumsCache = []MtPhotoAlbum{{ID: 1}}
			svc.albumsCacheExpire = time.Now().Add(time.Minute)
			a := &App{mtPhoto: svc}
			w := httptest.NewRecorder()
			a.handleCreateMtPhotoFolderAlbum(w, httptest.NewRequest(http.MethodPost, "/api/createMtPhotoFolderAlbum", strings.NewReader(tc.body)))
			if w.Code != tc.wantStatus {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if !reflect.DeepEqual(paths, tc.wantPaths) {
				t.Fatalf("paths=%v want=%v", paths, tc.wantPaths)
			}
			if len(paths) > 0 {
				if len(svc.albumsCache) != 0 {
					t.Error("album cache not invalidated")
				}
				var result struct {
					Album   MtPhotoAlbum
					Success bool
					Error   string
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Album.ID != 99 || result.Success != (tc.wantStatus == 200) {
					t.Errorf("result=%+v", result)
				}
				if tc.wantStatus != 200 && !strings.Contains(result.Error, "99") {
					t.Error("partial creation must identify the existing album")
				}
			}
		})
	}
}
