package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMtPhotoFolderFavoriteHandlers(t *testing.T) {
	rawDB, mock, cleanup := newSQLMock(t)
	t.Cleanup(cleanup)

	db := wrapMySQLDB(rawDB)
	app := &App{mtPhotoFolderFavorite: NewMtPhotoFolderFavoriteService(db)}

	t.Run("list not init", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderFavorites", nil)
		rr := httptest.NewRecorder()
		(&App{}).handleGetMtPhotoFolderFavorites(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
		}
	})

	t.Run("list ok", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
		}).AddRow(1, 644, "我的照片", "/photo/我的照片", "e38c3a4e832e7e66538002287d9663b5", `["常用"]`, "每周更新", now, now)
		mock.ExpectQuery(`SELECT id, folder_id, folder_name, folder_path, cover_md5, tags_json, note, created_at, updated_at\s+FROM mtphoto_folder_favorite\s+ORDER BY updated_at DESC, id DESC`).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/getMtPhotoFolderFavorites", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderFavorites(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
		if !strings.Contains(rr.Body.String(), `"folderId":644`) {
			t.Fatalf("body=%s", rr.Body.String())
		}
	})

	t.Run("upsert bad body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/upsertMtPhotoFolderFavorite", bytes.NewBufferString("{"))
		rr := httptest.NewRecorder()
		app.handleUpsertMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("upsert bad request", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"folderId": 0,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/upsertMtPhotoFolderFavorite", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		app.handleUpsertMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("upsert ok", func(t *testing.T) {
		now := time.Now()
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO mtphoto_folder_favorite")).WillReturnResult(sqlmock.NewResult(1, 1))
		rows := sqlmock.NewRows([]string{
			"id", "folder_id", "folder_name", "folder_path", "cover_md5", "tags_json", "note", "created_at", "updated_at",
		}).AddRow(1, 644, "我的照片", "/photo/我的照片", "e38c3a4e832e7e66538002287d9663b5", `["常用"]`, "每周更新", now, now)
		mock.ExpectQuery(`SELECT id, folder_id, folder_name, folder_path, cover_md5, tags_json, note, created_at, updated_at\s+FROM mtphoto_folder_favorite\s+WHERE folder_id = \?\s+LIMIT 1`).
			WithArgs(int64(644)).
			WillReturnRows(rows)

		body, _ := json.Marshal(map[string]any{
			"folderId":   644,
			"folderName": "我的照片",
			"folderPath": "/photo/我的照片",
			"coverMd5":   "e38c3a4e832e7e66538002287d9663b5",
			"tags":       []string{"常用"},
			"note":       "每周更新",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/upsertMtPhotoFolderFavorite", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		app.handleUpsertMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d, body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"success":true`) {
			t.Fatalf("body=%s", rr.Body.String())
		}
	})

	t.Run("remove bad body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/removeMtPhotoFolderFavorite", bytes.NewBufferString("{"))
		rr := httptest.NewRecorder()
		app.handleRemoveMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("remove invalid folderId", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"folderId": 0})
		req := httptest.NewRequest(http.MethodPost, "/api/removeMtPhotoFolderFavorite", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		app.handleRemoveMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("remove ok", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM mtphoto_folder_favorite WHERE folder_id = ?")).
			WithArgs(int64(644)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		body, _ := json.Marshal(map[string]any{"folderId": 644})
		req := httptest.NewRequest(http.MethodPost, "/api/removeMtPhotoFolderFavorite", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		app.handleRemoveMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
		}
	})
}
