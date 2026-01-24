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
	"sync/atomic"
	"testing"
	"time"
)

type errReadCloserMtPhoto struct{}

func (errReadCloserMtPhoto) Read(p []byte) (int, error) { return 0, errors.New("read err") }
func (errReadCloserMtPhoto) Close() error               { return nil }

func TestMtPhotoService_ensureLogin_NotConfigured(t *testing.T) {
	svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
	if _, _, err := svc.ensureLogin(context.Background(), false); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMtPhotoService_ensureLogin_LoginLockedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	if _, _, err := svc.ensureLogin(context.Background(), false); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMtPhotoService_refreshLocked_Errors(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{Timeout: time.Second})
	svc.refreshToken = ""
	if err := svc.refreshLocked(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("bad baseURL", func(t *testing.T) {
		svc := NewMtPhotoService("http://[::1", "u", "p", "", "/lsp", &http.Client{Timeout: time.Second})
		svc.refreshToken = "rt"
		if err := svc.refreshLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("do error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("net err")
			}),
		})
		svc.refreshToken = "rt"
		if err := svc.refreshLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("read error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       errReadCloserMtPhoto{},
					Header:     make(http.Header),
				}, nil
			}),
		})
		svc.refreshToken = "rt"
		if err := svc.refreshLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("bad json", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("{")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		svc.refreshToken = "rt"
		if err := svc.refreshLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"","auth_code":""}`)),
					Header:     make(http.Header),
				}, nil
			}),
		})
		svc.refreshToken = "rt"
		if err := svc.refreshLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestMtPhotoService_loginLocked_Errors(t *testing.T) {
	t.Run("bad baseURL", func(t *testing.T) {
		svc := NewMtPhotoService("http://[::1", "u", "p", "", "/lsp", &http.Client{Timeout: time.Second})
		if err := svc.loginLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("do error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("net err")
			}),
		})
		if err := svc.loginLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("read error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       errReadCloserMtPhoto{},
					Header:     make(http.Header),
				}, nil
			}),
		})
		if err := svc.loginLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("non-2xx", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Status:     "401 Unauthorized",
					Body:       io.NopCloser(strings.NewReader("x")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		if err := svc.loginLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("bad json", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("{")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		if err := svc.loginLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"t"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		})
		if err := svc.loginLocked(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestParseMtPhotoExpiresIn_CoversBranches(t *testing.T) {
	if !parseMtPhotoExpiresIn(0).IsZero() {
		t.Fatalf("expected zero time")
	}

	start := time.Now()
	got := parseMtPhotoExpiresIn(2) // seconds TTL
	if got.Before(start.Add(1 * time.Second)) {
		t.Fatalf("got=%v, want after %v", got, start.Add(1*time.Second))
	}

	epoch := time.Now().Add(10 * time.Minute).UnixMilli()
	got = parseMtPhotoExpiresIn(epoch)
	if got.UnixMilli() != epoch {
		t.Fatalf("got=%v, want unixms=%d", got, epoch)
	}
}

func TestMtPhotoService_doRequest_Errors(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{Timeout: time.Second})
	svc.token = "t"
	svc.authCode = "ac"
	svc.tokenExp = time.Now().Add(1 * time.Hour)

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

	var loginCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			atomic.AddInt32(&loginCalls, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		default:
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	if _, err := svc.doRequest(context.Background(), http.MethodGet, srv.URL+"/api-album", nil, nil, true, false); err == nil {
		t.Fatalf("expected error")
	}
	if atomic.LoadInt32(&loginCalls) == 0 {
		t.Fatalf("expected login called")
	}
}

func TestMtPhotoService_doRequest_CoversBodyAndCookieAndEnsureLoginError(t *testing.T) {
	t.Run("ensureLogin error", func(t *testing.T) {
		svc := NewMtPhotoService("", "", "", "", "/lsp", &http.Client{Timeout: time.Second})
		if _, err := svc.doRequest(context.Background(), http.MethodGet, "http://example.com", nil, nil, true, true); err == nil {
			t.Fatalf("expected error")
		}
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ok" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("jwt"); got != "t" {
			t.Fatalf("jwt=%q, want %q", got, "t")
		}
		if got := r.Header.Get("Cookie"); got != "auth_code=ac" {
			t.Fatalf("cookie=%q, want %q", got, "auth_code=ac")
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

	svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	svc.token = "t"
	svc.authCode = "ac"
	svc.tokenExp = time.Now().Add(1 * time.Hour)

	resp, err := svc.doRequest(context.Background(), http.MethodPost, srv.URL+"/ok", map[string]string{
		"X-Test": "1",
	}, []byte("payload"), true, true)
	if err != nil {
		t.Fatalf("doRequest error: %v", err)
	}
	_ = resp.Body.Close()
}

func TestMtPhotoService_buildURL_Errors(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
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
	svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
	if _, err := svc.GetAlbums(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
	svc.albumsCache = []MtPhotoAlbum{{ID: 1}}
	svc.albumsCacheExpire = time.Now().Add(1 * time.Second)
	albums, err := svc.GetAlbums(context.Background())
	if err != nil || len(albums) != 1 || albums[0].ID != 1 {
		t.Fatalf("albums=%v err=%v", albums, err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
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

	svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	if _, err := svc.GetAlbums(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
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

	svc = NewMtPhotoService(badJSON.URL, "u", "p", "", "/lsp", badJSON.Client())
	if _, err := svc.GetAlbums(context.Background()); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("build url error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
		svc.baseURL = "http://[::1"
		if _, err := svc.GetAlbums(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("doRequest error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.Path, "/auth/login") {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Status:     "401 Unauthorized",
						Body:       io.NopCloser(strings.NewReader("x")),
						Header:     make(http.Header),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("[]")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		if _, err := svc.GetAlbums(context.Background()); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
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

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		albums, err := svc.GetAlbums(context.Background())
		if err != nil || len(albums) != 1 || albums[0].ID != 1 {
			t.Fatalf("albums=%v err=%v", albums, err)
		}
	})
}

func TestMtPhotoService_GetAlbumFilesPage_CoversBranches(t *testing.T) {
	svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
	if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 1, 1, 1); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
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
		case "/auth/login":
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

	svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	items, total, pages, err := svc.GetAlbumFilesPage(context.Background(), 2, 1, 60)
	if err != nil || total != 1 || pages != 1 || len(items) != 1 || items[0].MD5 != "m2" {
		t.Fatalf("items=%v total/pages=%d/%d err=%v", items, total, pages, err)
	}

	// non-2xx and bad json
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
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

	svc = NewMtPhotoService(bad.URL, "u", "p", "", "/lsp", bad.Client())
	if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 3, 1, 60); err == nil {
		t.Fatalf("expected error")
	}

	bad = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
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

	svc = NewMtPhotoService(bad.URL, "u", "p", "", "/lsp", bad.Client())
	if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 4, 1, 60); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("build url error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
		svc.baseURL = "http://[::1"
		if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 1, 1, 60); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("doRequest error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.Path, "/auth/login") {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Status:     "401 Unauthorized",
						Body:       io.NopCloser(strings.NewReader("x")),
						Header:     make(http.Header),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("{}")),
					Header:     make(http.Header),
				}, nil
			}),
		})
		if _, _, _, err := svc.GetAlbumFilesPage(context.Background(), 1, 1, 60); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("pageSize default when <=0 (cache)", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
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
			case "/auth/login":
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

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
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
	svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
	if _, err := svc.ResolveFilePath(context.Background(), " "); err == nil {
		t.Fatalf("expected error")
	}

	svc.baseURL = "http://[::1"
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
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

	svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	// bad json / empty result
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
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

	svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
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

	svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
		t.Fatalf("expected error")
	}

	t.Run("doRequest error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
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
			case "/auth/login":
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

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		if _, err := svc.ResolveFilePath(context.Background(), "m"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
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

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		item, err := svc.ResolveFilePath(context.Background(), "m")
		if err != nil || item == nil || item.ID != 9 || item.FilePath != "/lsp/a/b.jpg" {
			t.Fatalf("item=%v err=%v", item, err)
		}
	})
}

func TestMtPhotoService_GatewayFileDownload_ParamValidation(t *testing.T) {
	svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
	if _, err := svc.GatewayFileDownload(context.Background(), 1, "m"); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
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
	svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
	svc.baseURL = "http://[::1"
	if _, err := svc.GatewayGet(context.Background(), "s260", "m"); err == nil {
		t.Fatalf("expected error")
	}

	svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{
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
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		case "/gateway/s260/m":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("ok"))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	resp, err := svc.GatewayGet(context.Background(), "s260", "m")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	_ = resp.Body.Close()
}
