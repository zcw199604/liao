package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMtPhotoService_GetFolderRoot_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
		case "/gateway/folders/root":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path": "",
				"folderList": []map[string]any{
					{
						"id":           518,
						"name":         "photo",
						"path":         "/photo",
						"subFolderNum": 16,
						"subFileNum":   0,
						"cover":        "",
						"s_cover":      nil,
						"fileType":     "folder",
					},
				},
				"fileList": []any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	content, err := svc.GetFolderRoot(ctx)
	if err != nil {
		t.Fatalf("GetFolderRoot error: %v", err)
	}
	if content == nil {
		t.Fatalf("content should not be nil")
	}
	if len(content.FolderList) != 1 {
		t.Fatalf("folderList=%d, want 1", len(content.FolderList))
	}
	if content.FolderList[0].ID != 518 {
		t.Fatalf("folder id=%d, want 518", content.FolderList[0].ID)
	}
	if content.FolderList[0].SCover != nil {
		t.Fatalf("s_cover should be nil when upstream returns null")
	}
}

func TestMtPhotoService_GetFolderContentPage_Paginate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
		case "/gateway/foldersV2/644":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":       "/photo/我的照片",
				"folderList": []any{},
				"fileList": []map[string]any{
					{
						"id":       1,
						"fileName": "a.jpg",
						"fileType": "JPEG",
						"size":     "1",
						"tokenAt":  "2024-09-16T18:06:47.000Z",
						"MD5":      "m1",
						"width":    100,
						"height":   200,
						"status":   2,
					},
					{
						"id":       2,
						"fileName": "b.mp4",
						"fileType": "MP4",
						"size":     "2",
						"tokenAt":  "2024-09-16T18:06:46.000Z",
						"MD5":      "m2",
						"status":   2,
					},
					{
						"id":       3,
						"fileName": "c.jpg",
						"fileType": "JPEG",
						"size":     "3",
						"tokenAt":  "2024-09-16T18:06:45.000Z",
						"MD5":      "m3",
						"status":   2,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	content1, total1, pages1, err := svc.GetFolderContentPage(ctx, 644, 1, 2)
	if err != nil {
		t.Fatalf("GetFolderContentPage(page1) error: %v", err)
	}
	if total1 != 3 || pages1 != 2 {
		t.Fatalf("total/pages=%d/%d, want 3/2", total1, pages1)
	}
	if len(content1.FileList) != 2 {
		t.Fatalf("page1 fileList len=%d, want 2", len(content1.FileList))
	}
	if content1.FileList[1].Type != "video" {
		t.Fatalf("page1 second type=%q, want video", content1.FileList[1].Type)
	}

	content2, total2, pages2, err := svc.GetFolderContentPage(ctx, 644, 2, 2)
	if err != nil {
		t.Fatalf("GetFolderContentPage(page2) error: %v", err)
	}
	if total2 != 3 || pages2 != 2 {
		t.Fatalf("total/pages=%d/%d, want 3/2", total2, pages2)
	}
	if len(content2.FileList) != 1 || content2.FileList[0].MD5 != "m3" {
		t.Fatalf("page2 fileList=%v, want [m3]", content2.FileList)
	}
}

func TestMtPhotoService_GetFolderContentPage_StatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
			})
		case "/gateway/foldersV2/999":
			w.WriteHeader(http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	_, _, _, err := svc.GetFolderContentPage(ctx, 999, 1, 60)
	if err == nil {
		t.Fatalf("expected error")
	}
	var statusErr *mtPhotoStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err=%v, want mtPhotoStatusError", err)
	}
	if statusErr.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", statusErr.StatusCode, http.StatusForbidden)
	}
}
