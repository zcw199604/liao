package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type errReadCloserMtPhoto struct{}

func (errReadCloserMtPhoto) Read(p []byte) (int, error) { return 0, errors.New("read err") }
func (errReadCloserMtPhoto) Close() error               { return nil }

func TestMtPhotoService_ensureAuthCode_NotConfigured(t *testing.T) {
	svc := NewMtPhotoService("", "", "/lsp", nil)
	if _, err := svc.ensureAuthCode(context.Background(), false); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMtPhotoService_ensureAuthCode_RefreshError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/auth_code" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	if _, err := svc.ensureAuthCode(context.Background(), false); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMtPhotoService_refreshAuthCodeLocked_Errors(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{Timeout: time.Second})
	if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("bad baseURL", func(t *testing.T) {
		svc := NewMtPhotoService("http://[::1", "u", "/lsp", &http.Client{Timeout: time.Second})
		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("do error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("net err")
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("read error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       errReadCloserMtPhoto{},
					Header:     make(http.Header),
				}, nil
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("bad json", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("{")),
					Header:     make(http.Header),
				}, nil
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"","auth_code":""}`)),
					Header:     make(http.Header),
				}, nil
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestMtPhotoService_refreshAuthCodeLocked_MoreErrors(t *testing.T) {
	t.Run("bad baseURL", func(t *testing.T) {
		svc := NewMtPhotoService("http://[::1", "u", "/lsp", &http.Client{Timeout: time.Second})
		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("do error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("net err")
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("read error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       errReadCloserMtPhoto{},
					Header:     make(http.Header),
				}, nil
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("non-2xx", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Status:     "401 Unauthorized",
					Body:       io.NopCloser(strings.NewReader("x")),
					Header:     make(http.Header),
				}, nil
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("bad json", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("{")),
					Header:     make(http.Header),
				}, nil
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"t"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		})

		if err := svc.refreshAuthCodeLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestParseMtPhotoAuthCodeExpire_CoversBranches(t *testing.T) {
	start := time.Now()
	got := parseMtPhotoAuthCodeExpire(0)
	if got.Before(start.Add(22 * time.Hour)) {
		t.Fatalf("got=%v, want default auth_code ttl", got)
	}

	got = parseMtPhotoAuthCodeExpire(2) // seconds TTL
	if got.Before(start.Add(1 * time.Second)) {
		t.Fatalf("got=%v, want after %v", got, start.Add(1*time.Second))
	}

	epoch := time.Now().Add(10 * time.Minute).UnixMilli()
	got = parseMtPhotoAuthCodeExpire(epoch)
	if got.UnixMilli() != epoch {
		t.Fatalf("got=%v, want unixms=%d", got, epoch)
	}
}

func TestMtPhotoService_doRequest_Errors(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{Timeout: time.Second})
	svc.authCode = "ac"

	_, err := svc.doRequest(context.Background(), http.MethodGet, "http://[::1", nil, nil, true, false)
	if err == nil {
		t.Fatalf("expected error")
	}

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(okSrv.Close)
	svc.httpClient = okSrv.Client()

	resp, err := svc.doRequest(context.Background(), http.MethodGet, okSrv.URL, map[string]string{" ": "x"}, nil, true, false)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	_ = resp.Body.Close()

	svc.httpClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("net err")
		}),
	}
	_, err = svc.doRequest(context.Background(), http.MethodGet, "http://example.com", nil, nil, true, false)
	if err == nil {
		t.Fatalf("expected error")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	resp, err = svc.doRequest(context.Background(), http.MethodGet, srv.URL+"/api-album", nil, nil, true, false)
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", resp.StatusCode)
	}
}

func TestMtPhotoService_doRequest_CoversBodyAndAPIKey(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		svc := NewMtPhotoService("", "", "/lsp", &http.Client{Timeout: time.Second})
		if _, err := svc.doRequest(context.Background(), http.MethodGet, "http://example.com", nil, nil, true, true); err == nil {
			t.Fatalf("expected error")
		}
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ok" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("x-api-key"); got != "u" {
			t.Fatalf("x-api-key=%q, want %q", got, "u")
		}
		if got := r.Header.Get("jwt"); got != "" {
			t.Fatalf("jwt header should be empty, got %q", got)
		}
		if got := r.Header.Get("Cookie"); got != "" {
			t.Fatalf("Cookie header should be empty, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "payload" {
			t.Fatalf("body=%q", string(body))
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	resp, err := svc.doRequest(context.Background(), http.MethodPost, srv.URL+"/ok", map[string]string{
		"X-Test": "1",
	}, []byte("payload"), true, true)
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	_ = resp.Body.Close()
}

func TestMtPhotoService_buildURL_Errors(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "/lsp", nil)
	if _, err := svc.buildURL("", nil); err == nil {
		t.Fatalf("expected error")
	}

	svc.baseURL = "http://[::1"
	if _, err := svc.buildURL("/x", nil); err == nil {
		t.Fatalf("expected error")
	}

	svc.baseURL = "http://example.com"
	u, err := svc.buildURL("/x", url.Values{"a": []string{"b"}})
	if err != nil || !strings.Contains(u, "a=b") {
		t.Fatalf("u=%q err=%v", u, err)
	}
}

func TestMtPhotoService_GetAlbums_CoversBranches(t *testing.T) {
	svc := NewMtPhotoService("", "", "/lsp", nil)
	if _, err := svc.GetAlbums(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "/lsp", nil)
	svc.albumsCache = []MtPhotoAlbum{{ID: 1}}
	svc.albumsCacheExpire = time.Now().Add(1 * time.Second)
	albums, err := svc.GetAlbums(context.Background())
	if err != nil || len(albums) != 1 || albums[0].ID != 1 {
		t.Fatalf("albums=%v err=%v", albums, err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/api-album":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	if _, err := svc.GetAlbums(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/api-album":
			_, _ = w.Write([]byte("{"))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(badJSON.Close)

	svc = NewMtPhotoService(badJSON.URL, "u", "/lsp", badJSON.Client())
	if _, err := svc.GetAlbums(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("build url error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", nil)
		svc.baseURL = "http://[::1"
		if _, err := svc.GetAlbums(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("doRequest error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("net err")
			}),
		})

		if _, err := svc.GetAlbums(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/auth_code":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
				return
			case "/api-album":
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"id": 1, "name": "a", "cover": "c", "count": 2},
				})
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
		albums, err := svc.GetAlbums(context.Background())
		if err != nil || len(albums) != 1 || albums[0].ID != 1 {
			t.Fatalf("albums=%v err=%v", albums, err)
		}
	})
}

func TestMtPhotoService_GetAlbumFilesPage_CoversBranches(t *testing.T) {
	svc := NewMtPhotoService("", "", "/lsp", nil)
	if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 1, 1, 1); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "/lsp", nil)
	if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 0, 1, 1); err == nil {
		t.Fatalf("expected error")
	}

	// cover page/pageSize normalization and cache branch
	svc.albumFilesCache[1] = mtPhotoAlbumFilesCacheEntry{
		expireAt: time.Now().Add(1 * time.Minute),
		total:    1,
		items:    []MtPhotoMediaItem{{ID: 1, MD5: "m1"}},
	}
	if items, _, pages, err := svc.GetAlbumFilesPage(context.Background(), 1, 0, 500); err != nil || len(items) != 1 || pages != 1 {
		t.Fatalf("items=%v pages=%d err=%v", items, pages, err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/api-album/filesV2/2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{
					{"day": "d", "list": []map[string]any{
						{"id": 1, "fileType": "JPEG", "MD5": ""},
						{"id": 2, "fileType": "JPEG", "MD5": "m2"},
					}},
				},
				"totalCount": 0,
				"ver":        1,
			})
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	items, total, pages, err := svc.GetAlbumFilesPage(context.Background(), 2, 1, 60)
	if err != nil || total != 1 || pages != 1 || len(items) != 1 || items[0].MD5 != "m2" {
		t.Fatalf("items=%v total/pages=%d/%d err=%v", items, total, pages, err)
	}

	// non-2xx and bad json
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/api-album/filesV2/3":
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(bad.Close)

	svc = NewMtPhotoService(bad.URL, "u", "/lsp", bad.Client())
	if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 3, 1, 60); err == nil {
		t.Fatalf("expected error")
	}

	bad = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/api-album/filesV2/4":
			_, _ = w.Write([]byte("{"))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(bad.Close)

	svc = NewMtPhotoService(bad.URL, "u", "/lsp", bad.Client())
	if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 4, 1, 60); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("build url error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", nil)
		svc.baseURL = "http://[::1"
		if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 1, 1, 60); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("doRequest error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("net err")
			}),
		})

		if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 1, 1, 60); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("pageSize default when <=0 (cache)", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", nil)
		items := make([]MtPhotoMediaItem, 0, 61)
		for i := 0; i < 61; i++ {
			items = append(items, MtPhotoMediaItem{ID: int64(i + 1), MD5: "m"})
		}
		svc.albumFilesCache[9] = mtPhotoAlbumFilesCacheEntry{
			expireAt: time.Now().Add(1 * time.Minute),
			total:    61,
			items:    items,
		}

		pageItems, total, pages, err := svc.GetAlbumFilesPage(context.Background(), 9, 1, 0)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if total != 61 || pages != 2 || len(pageItems) != 60 {
			t.Fatalf("total/pages/len=%d/%d/%d", total, pages, len(pageItems))
		}
	})

	t.Run("totalCount > 0 branch", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/auth_code":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
				return
			case "/api-album/filesV2/10":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"result": []map[string]any{
						{"day": "d", "list": []map[string]any{
							{"id": 1, "fileType": "MP4", "MD5": "m1"},
						}},
					},
					"totalCount": 2,
					"ver":        1,
				})
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
		pageItems, total, pages, err := svc.GetAlbumFilesPage(context.Background(), 10, 1, 60)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if total != 2 || pages != 1 || len(pageItems) != 1 || pageItems[0].Type != "video" {
			t.Fatalf("total/pages/items=%d/%d/%v", total, pages, pageItems)
		}
	})
}

func TestMtPhotoPaginationHelpers(t *testing.T) {
	if calcTotalPages(1, 0) != 0 {
		t.Fatalf("expected 0")
	}
	if calcTotalPages(0, 10) != 0 {
		t.Fatalf("expected 0")
	}

	if out := paginateMtPhotoItems(nil, 0, 1, 10); len(out) != 0 {
		t.Fatalf("out=%v", out)
	}
	if out := paginateMtPhotoItems([]MtPhotoMediaItem{{ID: 1}}, 1, 10, 1); len(out) != 0 {
		t.Fatalf("out=%v", out)
	}
}

func TestMtPhotoService_ResolveFilePath_CoversBranches(t *testing.T) {
	svc := NewMtPhotoService("", "", "/lsp", nil)
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "/lsp", nil)
	if _, err := svc.ResolveFilePath(context.Background(), " "); err == nil {
		t.Fatalf("expected error")
	}

	svc.baseURL = "http://[::1"
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/filesInMD5":
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	// bad json / empty result
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/filesInMD5":
			_, _ = w.Write([]byte("{"))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/filesInMD5":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("doRequest error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Status:     "401 Unauthorized",
					Body:       io.NopCloser(strings.NewReader("x")),
					Header:     make(http.Header),
				}, nil
			}),
		})

		if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("empty filepath", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/auth_code":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
				return
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"id": 1, "filePath": " "},
				})
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
		if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/auth_code":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
				return
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"id": 9, "filePath": " /lsp/a/b.jpg "},
				})
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
		item, err := svc.ResolveFilePath(context.Background(), "m")
		if err != nil || item == nil || item.ID != 9 || item.FilePath != "/lsp/a/b.jpg" {
			t.Fatalf("item=%v err=%v", item, err)
		}
	})
}

func TestMtPhotoService_ListSameMediaByMD5(t *testing.T) {
	t.Run("empty when upstream empty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/auth_code":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
				return
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
		items, err := svc.ListSameMediaByMD5(context.Background(), "m1")
		if err != nil {
			t.Fatalf("ListSameMediaByMD5 error: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("items=%v, want empty", items)
		}
	})

	t.Run("maps and sorts items", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/auth_code":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
				return
			case "/gateway/filesInMD5":
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{
						"id":       2,
						"MD5":      "m1",
						"filePath": "/lsp/a/old.jpg",
						"tokenAt":  "2026-01-01T08:00:00.000Z",
					},
					{
						"id":         1,
						"md5":        "m1",
						"filePath":   "/lsp/a/new.jpg",
						"tokenAt":    "2026-02-01T09:00:00.000Z",
						"folderId":   644,
						"folderPath": "/photo/新目录",
						"folderName": "新目录",
					},
				})
				return
			case "/gateway/fileInfo/2/m1":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":       2,
					"MD5":      "m1",
					"filePath": "/lsp/tg/dir-a/old.jpg",
					"folderId": 1952,
				})
				return
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
		items, err := svc.ListSameMediaByMD5(context.Background(), "m1")
		if err != nil {
			t.Fatalf("ListSameMediaByMD5 error: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("len(items)=%d, want 2", len(items))
		}
		if items[0].ID != 1 || items[0].FileName != "new.jpg" || !items[0].CanOpenFolder {
			t.Fatalf("first item=%+v", items[0])
		}
		if items[1].ID != 2 || items[1].FolderID != 1952 || items[1].Directory != "/lsp/a" || !items[1].CanOpenFolder {
			t.Fatalf("second item=%+v", items[1])
		}
	})
}

func TestMtPhotoService_GatewayFileDownload_ParamValidation(t *testing.T) {
	svc := NewMtPhotoService("", "", "/lsp", nil)
	if _, err := svc.GatewayFileDownload(context.Background(), 1, "m"); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "/lsp", nil)
	if _, err := svc.GatewayFileDownload(context.Background(), 0, "m"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := svc.GatewayFileDownload(context.Background(), 1, " "); err == nil {
		t.Fatalf("expected error")
	}

	svc.baseURL = "http://[::1"
	if _, err := svc.GatewayFileDownload(context.Background(), 1, "m"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMtPhotoService_GatewayGet_CoversBranches(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "/lsp", nil)
	svc.baseURL = "http://[::1"
	if _, err := svc.GatewayGet(context.Background(), "s260", "m"); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "/lsp", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Status:     "401 Unauthorized",
				Body:       io.NopCloser(strings.NewReader("x")),
				Header:     make(http.Header),
			}, nil
		}),
	})

	if _, err := svc.GatewayGet(context.Background(), "s260", "m"); err == nil {
		t.Fatalf("expected error")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/auth_code":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"auth_code":  "ac",
				"expires_in": time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/s260/m":
			if got := r.URL.Query().Get("auth_code"); got != "ac" {
				t.Fatalf("auth_code=%q, want ac", got)
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("ok"))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "/lsp", srv.Client())
	resp, err := svc.GatewayGet(context.Background(), "s260", "m")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	_ = resp.Body.Close()
}
