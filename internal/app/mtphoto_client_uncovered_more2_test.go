package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestMtPhotoService_ListSameMediaByMD5_UncoveredSortAndFallbackBranches(t *testing.T) {
	md5Input := "0123456789abcdef0123456789abcdef"
	var hitPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitPaths = append(hitPaths, r.URL.Path)

		switch {
		case r.URL.Path == "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "t",
				"auth_code":    "ac",
				"expires_in":   time.Now().Add(time.Hour).UnixMilli(),
			})
		case r.URL.Path == "/gateway/filesInMD5":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 11, "filePath": "root.jpg", "folderPath": "/same", "tokenAt": "2026-03-01T10:00:00Z", "day": "2026-03-01"},
				{"id": 12, "filePath": "/same/b.jpg", "folderPath": "/same", "tokenAt": "2026-03-01T10:00:00Z", "day": "2026-03-01"},
				{"id": 13, "filePath": "/same/a.jpg", "folderPath": "/same", "tokenAt": "2026-03-01T10:00:00Z", "day": "2026-03-01"},
				{"id": 14, "filePath": "/same/a.jpg", "folderPath": "/same", "tokenAt": "2026-03-01T10:00:00Z", "day": "2026-03-01"},
			})
		case strings.HasPrefix(r.URL.Path, "/gateway/fileInfo/"):
			// 这里返回非 2xx，触发 GetFileInfo error 分支；本测试重点验证前置分支和排序分支。
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
	items, err := svc.ListSameMediaByMD5(context.Background(), md5Input)
	if err != nil {
		t.Fatalf("ListSameMediaByMD5 err=%v", err)
	}
	if len(items) != 4 {
		t.Fatalf("items len=%d", len(items))
	}

	var rootItem *MtPhotoSameMediaItem
	for i := range items {
		if items[i].ID == 11 {
			rootItem = &items[i]
			break
		}
	}
	if rootItem == nil || rootItem.Directory != "" {
		t.Fatalf("directory '.' branch not hit, item=%+v", rootItem)
	}

	// 同 token/day/folderPath 时按 FilePath 升序，FilePath 相同再按 ID 升序。
	orderedIDs := []int64{items[0].ID, items[1].ID, items[2].ID, items[3].ID}
	if !slices.Contains(orderedIDs, int64(13)) || !slices.Contains(orderedIDs, int64(14)) || !slices.Contains(orderedIDs, int64(12)) {
		t.Fatalf("orderedIDs=%v", orderedIDs)
	}
	idx12 := slices.Index(orderedIDs, int64(12))
	idx13 := slices.Index(orderedIDs, int64(13))
	idx14 := slices.Index(orderedIDs, int64(14))
	if !(idx13 < idx12 && idx13 < idx14 && idx14 < idx12) {
		t.Fatalf("unexpected sort order ids=%v", orderedIDs)
	}

	// row 的 md5 为空时应回退使用入参 md5 调用 /gateway/fileInfo/{id}/{md5}。
	wantInfoPath := "/gateway/fileInfo/12/" + md5Input
	if !slices.Contains(hitPaths, wantInfoPath) {
		t.Fatalf("expected hit %q, got paths=%v", wantInfoPath, hitPaths)
	}
}
