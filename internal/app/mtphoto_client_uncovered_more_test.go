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

func TestMtPhotoMappingHelpers_UncoveredBranches(t *testing.T) {
	if got := mapMtPhotoFolderFiles([]mtPhotoFolderFileItemRaw{{ID: 1}}); len(got) != 0 {
		t.Fatalf("expected empty mapped files, got=%v", got)
	}

	groups := []mtPhotoFolderFilesGroup{
		{Day: "", List: []mtPhotoFileItem{{ID: 1, MD5: "", VMD5: ""}}},
		{Day: "2026-01-01", List: []mtPhotoFileItem{{ID: 2, MD5: "m2", FileType: "JPEG"}}},
		{Day: "", List: []mtPhotoFileItem{{ID: 3, MD5: "m3", FileType: "JPEG"}}},
	}
	mapped := mapMtPhotoTimelineFiles(groups)
	if len(mapped) != 2 {
		t.Fatalf("mapped len=%d, want 2", len(mapped))
	}
	if mapped[0].ID != 2 {
		t.Fatalf("expected dated item first, got=%+v", mapped)
	}
}

func TestMtPhotoService_getFolderTimelineFiles_AndContentPageBranches(t *testing.T) {
	t.Run("timeline doRequest error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network fail")
		})})
		svc.token = "t"
		svc.authCode = "ac"
		svc.tokenExp = time.Now().Add(time.Hour)
		if _, _, err := svc.getFolderTimelineFiles(context.Background(), 1); err == nil {
			t.Fatalf("expected timeline error")
		}
	})

	t.Run("content page pageSize<=0 fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "auth_code": "ac", "expires_in": time.Now().Add(time.Hour).UnixMilli()})
			case "/gateway/foldersV2/1":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"path":       "/",
					"folderList": []map[string]any{},
					"fileList":   []map[string]any{{"id": 1, "md5": "m1", "fileName": "a.jpg", "fileType": "JPEG"}},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		content, total, totalPages, err := svc.GetFolderContentPage(context.Background(), 1, 1, 0, false)
		if err != nil {
			t.Fatalf("GetFolderContentPage err=%v", err)
		}
		if content == nil || total != 1 || totalPages != 1 {
			t.Fatalf("content=%v total=%d totalPages=%d", content, total, totalPages)
		}
	})
}

func TestMtPhotoService_ListSameMediaByMD5_AdditionalSortAndEnrichBranches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "auth_code": "ac", "expires_in": time.Now().Add(time.Hour).UnixMilli()})
		case "/gateway/filesInMD5":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "filePath": "a.jpg", "tokenAt": "2026-03-01T10:00:00Z", "md5": "m1", "folderId": 0},
				{"id": 2, "filePath": "/b.jpg", "tokenAt": "2026-03-01T10:00:00Z", "md5": "m1", "folderPath": "/bdir"},
				{"id": 3, "filePath": "/adir/c.jpg", "tokenAt": "2026-03-01T10:00:00Z", "md5": "m1", "folderPath": "/adir"},
			})
		case "/gateway/fileInfo/1/m1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "filePath": "/x/aa.jpg", "folderId": 11})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	items, err := svc.ListSameMediaByMD5(context.Background(), "m1")
	if err != nil {
		t.Fatalf("ListSameMediaByMD5 err=%v", err)
	}
	if len(items) != 3 {
		t.Fatalf("items=%v", items)
	}
	if items[0].ID != 3 {
		t.Fatalf("sorted first should be /adir item, got=%+v", items[0])
	}
	if items[1].ID != 2 {
		t.Fatalf("sorted second should be /bdir item, got=%+v", items[1])
	}
	if items[2].ID != 1 || items[2].FolderPath != "/x" || items[2].FolderName != "x" {
		t.Fatalf("enriched item unexpected: %+v", items[2])
	}
}
