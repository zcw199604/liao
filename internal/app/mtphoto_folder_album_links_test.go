package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMtPhotoFolderAlbumLinks(t *testing.T) {
	for _, tc := range []struct {
		name, second string
		status       int
	}{
		{"multiple albums and exclusions", `[{"type":"folder","value":"{\"id\":12}","exclude":false}]`, 200},
		{"object values mixed with strings", `[{"type":"folder","value":{"id":12,"label":"/tg/a"},"exclude":false},{"type":"folder","value":"{\"id\":12}","exclude":false},{"type":"folder","value":{"id":34},"exclude":true},{"type":"tag","value":{"id":56},"exclude":false}]`, 200},
		{"null folder value", `[{"type":"folder","value":null,"exclude":false}]`, 502},
		{"object missing folder id", `[{"type":"folder","value":{"label":"/tg/a"},"exclude":false}]`, 502},
		{"invalid link must not report unlinked", `[{"type":"folder","value":"broken","exclude":false}]`, 502},
		{"upstream permission failure", "forbidden", 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-api-key") != "key" {
					t.Error("missing API key")
				}
				switch r.URL.Path {
				case "/api-album":
					_, _ = w.Write([]byte(`[{"id":1,"name":"旅行"},{"id":2,"name":"精选"}]`))
				case "/api-album/link/1":
					_, _ = w.Write([]byte(`[{"type":"folder","value":"{\"id\":12,\"label\":\"/tg/a\"}","exclude":false},{"type":"folder","value":"{\"id\":12}","exclude":false},{"type":"folder","value":"{\"id\":34}","exclude":true},{"type":"tag","value":"{\"id\":56}","exclude":false}]`))
				case "/api-album/link/2":
					if tc.second == "forbidden" {
						w.WriteHeader(403)
					} else {
						_, _ = w.Write([]byte(tc.second))
					}
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			a := &App{mtPhoto: NewMtPhotoService(srv.URL, "key", "", srv.Client())}
			w := httptest.NewRecorder()
			a.handleGetMtPhotoFolderAlbumLinks(w, httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderAlbumLinks", nil))
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.status == 200 {
				var result struct {
					Items map[string][]MtPhotoAlbum `json:"items"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if len(result.Items) != 1 || len(result.Items["12"]) != 2 || result.Items["12"][0].Name != "旅行" {
					t.Fatalf("result=%+v", result)
				}
			}
		})
	}
}
