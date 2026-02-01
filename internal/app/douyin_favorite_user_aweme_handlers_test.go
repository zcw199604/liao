package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleDouyinFavoriteUserAwemeUpsert_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
		WithArgs("MS4wLjABAAAA_x", "111", "222").
		WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}))

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO douyin_favorite_user_aweme`).
		WithArgs(
			"MS4wLjABAAAA_x",
			"111",
			"video",
			"作品1",
			"https://example.com/c1.jpg",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO douyin_favorite_user_aweme`).
		WithArgs(
			"MS4wLjABAAAA_x",
			"222",
			"image",
			"作品2",
			"https://example.com/c2.jpg",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{douyinFavorite: NewDouyinFavoriteService(db)}

	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/aweme/upsert", map[string]any{
		"secUserId": "MS4wLjABAAAA_x",
		"items": []any{
			map[string]any{
				"awemeId":   "222",
				"type":      "image",
				"desc":      "作品2",
				"coverUrl":  "https://example.com/c2.jpg",
				"downloads": []any{"https://example.com/img1.jpg", "https://example.com/img2.jpg"},
			},
			map[string]any{
				"awemeId":   "111",
				"type":      "video",
				"desc":      "作品1",
				"coverUrl":  "https://example.com/c1.jpg",
				"downloads": []any{"https://example.com/v1.mp4"},
			},
		},
	})
	rr := httptest.NewRecorder()

	app.handleDouyinFavoriteUserAwemeUpsert(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, ok := resp["success"].(bool); !ok || !got {
		t.Fatalf("success=%v, want true", resp["success"])
	}
	if got, _ := resp["added"].(float64); got != 2 {
		t.Fatalf("added=%v, want %v", resp["added"], 2)
	}
}

func TestHandleDouyinFavoriteUserAwemeList_Success(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	createTime := time.Date(2026, 1, 24, 1, 2, 3, 0, time.Local)
	updateTime := time.Date(2026, 1, 24, 1, 2, 4, 0, time.Local)
	mock.ExpectQuery(`SELECT aweme_id, type, description, cover_url, downloads, created_at, updated_at`).
		WithArgs("MS4wLjABAAAA_x", 3, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"aweme_id",
			"type",
			"description",
			"cover_url",
			"downloads",
			"created_at",
			"updated_at",
		}).AddRow(
			"111",
			sql.NullString{String: "video", Valid: true},
			sql.NullString{String: "作品1", Valid: true},
			sql.NullString{String: "https://example.com/c1.jpg", Valid: true},
			sql.NullString{String: `["https://example.com/v1.mp4"]`, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: updateTime, Valid: true},
		).AddRow(
			"222",
			sql.NullString{String: "image", Valid: true},
			sql.NullString{String: "作品2", Valid: true},
			sql.NullString{String: "https://example.com/c2.jpg", Valid: true},
			sql.NullString{String: `["https://example.com/img1.jpg","https://example.com/img2.jpg"]`, Valid: true},
			sql.NullTime{Time: createTime, Valid: true},
			sql.NullTime{Time: updateTime, Valid: true},
		))

	app := &App{
		douyinFavorite:   NewDouyinFavoriteService(db),
		douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", 60*time.Second),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/favoriteUser/aweme/list?secUserId=MS4wLjABAAAA_x&cursor=0&count=2", nil)
	rr := httptest.NewRecorder()
	app.handleDouyinFavoriteUserAwemeList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		SecUserID string `json:"secUserId"`
		Cursor    int    `json:"cursor"`
		HasMore   bool   `json:"hasMore"`
		Items     []struct {
			DetailID         string `json:"detailId"`
			Key              string `json:"key"`
			CoverDownloadURL string `json:"coverDownloadUrl"`
			Items            []struct {
				URL         string `json:"url"`
				DownloadURL string `json:"downloadUrl"`
			} `json:"items"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.SecUserID != "MS4wLjABAAAA_x" {
		t.Fatalf("secUserId=%q, want %q", resp.SecUserID, "MS4wLjABAAAA_x")
	}
	if resp.Cursor != 2 {
		t.Fatalf("cursor=%d, want %d", resp.Cursor, 2)
	}
	if resp.HasMore {
		t.Fatalf("hasMore=true, want false")
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items len=%d, want %d", len(resp.Items), 2)
	}
	if resp.Items[0].DetailID != "111" {
		t.Fatalf("items[0].detailId=%q, want %q", resp.Items[0].DetailID, "111")
	}
	if strings.TrimSpace(resp.Items[0].Key) == "" {
		t.Fatalf("items[0].key should not be empty")
	}
	if !strings.Contains(resp.Items[0].CoverDownloadURL, "/api/douyin/cover?key=") {
		t.Fatalf("items[0].coverDownloadUrl=%q", resp.Items[0].CoverDownloadURL)
	}
	if len(resp.Items[0].Items) != 1 {
		t.Fatalf("items[0].items len=%d, want %d", len(resp.Items[0].Items), 1)
	}
	if !strings.Contains(resp.Items[0].Items[0].DownloadURL, "/api/douyin/download?key=") {
		t.Fatalf("items[0].items[0].downloadUrl=%q", resp.Items[0].Items[0].DownloadURL)
	}
}

func TestHandleDouyinFavoriteUserAwemePullLatest_Success(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/douyin/account/page" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		payload := map[string]any{
			"message": "获取数据成功！",
			"data": map[string]any{
				"next_cursor": 0,
				"has_more":    false,
				"items": []any{
					map[string]any{
						"aweme_id": "111",
						"desc":     "作品1",
						"video": map[string]any{
							"cover": map[string]any{
								"url_list": []any{upstream.URL + "/cover1.jpg"},
							},
							"play_addr": map[string]any{
								"url_list": []any{upstream.URL + "/media1.mp4"},
							},
						},
					},
				},
			},
			"params": map[string]any{},
			"time":   "2026-01-01",
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer upstream.Close()

	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT aweme_id\s+FROM douyin_favorite_user_aweme`).
		WithArgs("MS4wLjABAAAA_x", "111").
		WillReturnRows(sqlmock.NewRows([]string{"aweme_id"}))

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO douyin_favorite_user_aweme`).
		WithArgs(
			"MS4wLjABAAAA_x",
			"111",
			"video",
			"作品1",
			upstream.URL+"/cover1.jpg",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	app := &App{
		douyinFavorite:   NewDouyinFavoriteService(db),
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}

	req := newJSONRequest(t, http.MethodPost, "http://example.com/api/douyin/favoriteUser/aweme/pullLatest", map[string]any{
		"secUserId": "MS4wLjABAAAA_x",
		"cookie":    "",
		"count":     1,
	})
	rr := httptest.NewRecorder()
	app.handleDouyinFavoriteUserAwemePullLatest(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, _ := resp["added"].(float64); got != 1 {
		t.Fatalf("added=%v, want %v", resp["added"], 1)
	}
	if got, _ := resp["fetched"].(float64); got != 1 {
		t.Fatalf("fetched=%v, want %v", resp["fetched"], 1)
	}
}
