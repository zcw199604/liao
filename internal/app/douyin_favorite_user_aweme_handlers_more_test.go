package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleDouyinFavoriteUserAwemeUpsert_ErrorBranches(t *testing.T) {
	t.Run("service not initialized", func(t *testing.T) {
		req := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{})
		rr := httptest.NewRecorder()
		(*App)(nil).handleDouyinFavoriteUserAwemeUpsert(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		app := &App{douyinFavorite: &DouyinFavoriteService{}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("{"))
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemeUpsert(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("empty secUserId", func(t *testing.T) {
		app := &App{douyinFavorite: &DouyinFavoriteService{}}
		req := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{
			"secUserId": " ",
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemeUpsert(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("empty awemeId is skipped and returns added=0", func(t *testing.T) {
		db, _, cleanup := newSQLMock(t)
		defer cleanup()

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		req := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{
			"secUserId": "MS4wLjABAAAA_x",
			"items": []any{
				map[string]any{"awemeId": " "},
			},
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemeUpsert(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
		if got, _ := resp["added"].(float64); got != 0 {
			t.Fatalf("added=%v", resp["added"])
		}
	})

	t.Run("save error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("MS4wLjABAAAA_x", "111").
			WillReturnError(errors.New("boom"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		req := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{
			"secUserId": "MS4wLjABAAAA_x",
			"items": []any{
				map[string]any{"awemeId": "111"},
			},
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemeUpsert(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleDouyinFavoriteUserAwemeList_MoreBranches(t *testing.T) {
	t.Run("service not initialized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api?secUserId=x", nil)
		rr := httptest.NewRecorder()
		(*App)(nil).handleDouyinFavoriteUserAwemeList(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("missing secUserId", func(t *testing.T) {
		app := &App{douyinFavorite: &DouyinFavoriteService{}}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemeList(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("list db error", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		mock.ExpectQuery(`FROM douyin_favorite_user_aweme`).
			WithArgs("MS4wLjABAAAA_x", 21, 0).
			WillReturnError(errors.New("boom"))

		app := &App{douyinFavorite: NewDouyinFavoriteService(wrapMySQLDB(db))}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/api?secUserId=MS4wLjABAAAA_x", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemeList(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("media inference and preview branches", func(t *testing.T) {
		db, mock, cleanup := newSQLMock(t)
		defer cleanup()

		now := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
		rows := sqlmock.NewRows([]string{
			"aweme_id",
			"type",
			"description",
			"cover_url",
			"downloads",
			"is_pinned",
			"pinned_rank",
			"pinned_at",
			"publish_at",
			"crawled_at",
			"last_seen_at",
			"status",
			"author_unique_id",
			"author_name",
			"created_at",
			"updated_at",
		}).AddRow(
			"infer-live",
			sql.NullString{String: "", Valid: true},
			sql.NullString{String: "作品A", Valid: true},
			sql.NullString{String: "https://example.com/c1.jpg", Valid: true},
			sql.NullString{String: `["https://example.com/a.jpg","https://example.com/b.jpg","https://example.com/aweme/v1/play/?video_id=v1"]`, Valid: true},
			false, nil, nil,
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullString{String: "normal", Valid: true},
			sql.NullString{String: "u1", Valid: true},
			sql.NullString{String: "n1", Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
		).AddRow(
			"explicit-live-image-only",
			sql.NullString{String: "livePhoto", Valid: true},
			sql.NullString{String: "作品B", Valid: true},
			sql.NullString{String: "https://example.com/c2.jpg", Valid: true},
			sql.NullString{String: `["https://example.com/c.jpg",""]`, Valid: true},
			false, nil, nil,
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullString{String: "normal", Valid: true},
			sql.NullString{String: "u2", Valid: true},
			sql.NullString{String: "n2", Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
		).AddRow(
			"infer-video-empty-downloads",
			sql.NullString{String: "", Valid: true},
			sql.NullString{String: "作品C", Valid: true},
			sql.NullString{String: "", Valid: true},
			sql.NullString{},
			false, nil, nil,
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullString{String: "normal", Valid: true},
			sql.NullString{String: "u3", Valid: true},
			sql.NullString{String: "n3", Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
		).AddRow(
			"type-video-cn",
			sql.NullString{String: "视频", Valid: true},
			sql.NullString{String: "作品D", Valid: true},
			sql.NullString{String: "https://example.com/c4.jpg", Valid: true},
			sql.NullString{String: `["https://example.com/aweme/v1/play/?video_id=v2","https://example.com/noext",""]`, Valid: true},
			false, nil, nil,
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullString{String: "normal", Valid: true},
			sql.NullString{String: "u4", Valid: true},
			sql.NullString{String: "n4", Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
		).AddRow(
			"infer-image-only",
			sql.NullString{String: "", Valid: true},
			sql.NullString{String: "作品E", Valid: true},
			sql.NullString{String: "https://example.com/c5.jpg", Valid: true},
			sql.NullString{String: `["https://example.com/e.jpg"]`, Valid: true},
			false, nil, nil,
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullString{String: "normal", Valid: true},
			sql.NullString{String: "u5", Valid: true},
			sql.NullString{String: "n5", Valid: true},
			sql.NullTime{Time: now, Valid: true},
			sql.NullTime{Time: now, Valid: true},
		)

		mock.ExpectQuery(`FROM douyin_favorite_user_aweme`).
			WithArgs("MS4wLjABAAAA_x", 11, 0).
			WillReturnRows(rows)

		app := &App{
			douyinFavorite:   NewDouyinFavoriteService(wrapMySQLDB(db)),
			douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", 30*time.Second),
		}

		req := httptest.NewRequest(http.MethodGet, "http://example.com/api?secUserId=MS4wLjABAAAA_x&count=10", nil)
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemeList(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}

		var resp struct {
			Items []struct {
				DetailID       string `json:"detailId"`
				MediaType      string `json:"mediaType"`
				IsLivePhoto    bool   `json:"isLivePhoto"`
				ImageCount     int    `json:"imageCount"`
				LivePhotoPairs int    `json:"livePhotoPairs"`
				Key            string `json:"key"`
				Items          []struct {
					Type             string `json:"type"`
					OriginalFilename string `json:"originalFilename"`
				} `json:"items"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if len(resp.Items) != 5 {
			t.Fatalf("items len=%d", len(resp.Items))
		}

		index := map[string]int{}
		for i, it := range resp.Items {
			index[it.DetailID] = i
		}

		live := resp.Items[index["infer-live"]]
		if live.MediaType != "livePhoto" || !live.IsLivePhoto || live.LivePhotoPairs != 1 || strings.TrimSpace(live.Key) == "" {
			t.Fatalf("infer-live=%+v", live)
		}

		explicitLive := resp.Items[index["explicit-live-image-only"]]
		if explicitLive.MediaType != "livePhoto" || explicitLive.LivePhotoPairs != 1 || len(explicitLive.Items) != 1 {
			t.Fatalf("explicit-live=%+v", explicitLive)
		}

		inferVideo := resp.Items[index["infer-video-empty-downloads"]]
		if inferVideo.MediaType != "video" || inferVideo.ImageCount != 0 || inferVideo.Key != "" {
			t.Fatalf("infer-video=%+v", inferVideo)
		}

		typeVideoCN := resp.Items[index["type-video-cn"]]
		if typeVideoCN.MediaType != "video" || len(typeVideoCN.Items) != 2 {
			t.Fatalf("type-video-cn=%+v", typeVideoCN)
		}
		if typeVideoCN.Items[0].Type != "video" || !strings.HasSuffix(typeVideoCN.Items[0].OriginalFilename, ".mp4") {
			t.Fatalf("video preview=%+v", typeVideoCN.Items[0])
		}
		if typeVideoCN.Items[1].Type != "image" || !strings.HasSuffix(typeVideoCN.Items[1].OriginalFilename, ".jpg") {
			t.Fatalf("image preview=%+v", typeVideoCN.Items[1])
		}

		imageOnly := resp.Items[index["infer-image-only"]]
		if imageOnly.MediaType != "imageAlbum" || imageOnly.ImageCount != 1 {
			t.Fatalf("image-only=%+v", imageOnly)
		}
	})
}

func TestHandleDouyinFavoriteUserAwemePullLatest_MoreBranches(t *testing.T) {
	t.Run("service not initialized / not configured / bad json / empty secUserId", func(t *testing.T) {
		req := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{})
		rr := httptest.NewRecorder()
		(*App)(nil).handleDouyinFavoriteUserAwemePullLatest(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("nil app status=%d", rr.Code)
		}

		app := &App{
			douyinFavorite:   &DouyinFavoriteService{},
			douyinDownloader: NewDouyinDownloaderService("", "", "", "", 3*time.Second),
		}
		rr = httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemePullLatest(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("not configured status=%d", rr.Code)
		}

		reqBad := httptest.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("{"))
		rr = httptest.NewRecorder()
		app2 := &App{
			douyinFavorite:   &DouyinFavoriteService{},
			douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", 3*time.Second),
		}
		app2.handleDouyinFavoriteUserAwemePullLatest(rr, reqBad)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("bad json status=%d", rr.Code)
		}

		reqEmpty := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{
			"secUserId": " ",
		})
		rr = httptest.NewRecorder()
		app2.handleDouyinFavoriteUserAwemePullLatest(rr, reqEmpty)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("empty secUserId status=%d", rr.Code)
		}
	})

	t.Run("count<=0 defaults to 50 and fetch error", func(t *testing.T) {
		var gotCount float64
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/account/page":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if v, ok := body["count"].(float64); ok {
					gotCount = v
				}
				w.WriteHeader(http.StatusBadGateway)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "boom"})
			default:
				http.NotFound(w, r)
			}
		}))
		defer upstream.Close()

		app := &App{
			douyinFavorite:   &DouyinFavoriteService{},
			douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 3*time.Second),
		}
		req := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{
			"secUserId": "MS4wLjABAAAA_x",
			"count":     0,
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemePullLatest(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if gotCount != 50 {
			t.Fatalf("count=%v, want 50", gotCount)
		}
	})

	t.Run("count>200 clamp and save error", func(t *testing.T) {
		var gotCount float64
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/douyin/account/page":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if v, ok := body["count"].(float64); ok {
					gotCount = v
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": "ok",
					"data": map[string]any{
						"next_cursor": 0,
						"has_more":    false,
						"items": []any{
							map[string]any{
								"aweme_id": "",
								"desc":     "skip me",
							},
							map[string]any{
								"aweme_id": "111",
								"desc":     "ok",
								"video": map[string]any{
									"play_addr": map[string]any{
										"url_list": []any{"https://example.com/aweme/v1/play/?video_id=v1"},
									},
								},
							},
						},
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer upstream.Close()

		db, mock, cleanup := newSQLMock(t)
		defer cleanup()
		mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
			WithArgs("MS4wLjABAAAA_x", "111").
			WillReturnError(errors.New("boom"))

		app := &App{
			douyinFavorite:   NewDouyinFavoriteService(wrapMySQLDB(db)),
			douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 3*time.Second),
		}
		req := newJSONRequest(t, http.MethodPost, "http://example.com", map[string]any{
			"secUserId": "MS4wLjABAAAA_x",
			"count":     999,
		})
		rr := httptest.NewRecorder()
		app.handleDouyinFavoriteUserAwemePullLatest(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if gotCount != 200 {
			t.Fatalf("count=%v, want 200", gotCount)
		}
	})
}

