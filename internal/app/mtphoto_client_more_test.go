package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMtPhotoStatusError_ErrorBranches(t *testing.T) {
	var nilErr *mtPhotoStatusError
	if got := nilErr.Error(); got != "mtPhoto 请求失败" {
		t.Fatalf("nil error=%q", got)
	}

	if got := (&mtPhotoStatusError{StatusCode: 401}).Error(); got != "mtPhoto 请求失败: HTTP 401" {
		t.Fatalf("statusCode only error=%q", got)
	}

	if got := (&mtPhotoStatusError{Status: " 403 Forbidden ", Action: " "}).Error(); got != "mtPhoto 请求失败: 403 Forbidden" {
		t.Fatalf("status only error=%q", got)
	}

	if got := (&mtPhotoStatusError{Action: "获取目录失败"}).Error(); got != "获取目录失败" {
		t.Fatalf("action only error=%q", got)
	}

	if got := (&mtPhotoStatusError{Status: "500 Internal Server Error", Action: "获取目录失败"}).Error(); got != "获取目录失败: 500 Internal Server Error" {
		t.Fatalf("full error=%q", got)
	}
}

func TestNormalizeMtPhotoSize_MoreBranches(t *testing.T) {
	if got := normalizeMtPhotoSize(json.Number("123")); got != "123" {
		t.Fatalf("json.Number=%q", got)
	}
	if got := normalizeMtPhotoSize(float64(-1)); got != "" {
		t.Fatalf("negative float=%q", got)
	}
	if got := normalizeMtPhotoSize(float64(9.9)); got != "9" {
		t.Fatalf("float convert=%q", got)
	}
	if got := normalizeMtPhotoSize(struct{ A int }{A: 7}); !strings.Contains(got, "{7}") {
		t.Fatalf("default branch=%q", got)
	}
}

func TestMapMtPhotoFolderFiles_MD5FallbackAndSkip(t *testing.T) {
	got := mapMtPhotoFolderFiles([]mtPhotoFolderFileItemRaw{
		{
			ID:       1,
			FileName: "a.jpg",
			FileType: "JPEG",
			MD5:      "",
			MD5Lower: "m1",
			TokenAt:  "2026-01-02 01:02:03",
			Size:     json.Number("12"),
		},
		{
			ID:       2,
			FileName: "skip.jpg",
			FileType: "JPEG",
			MD5:      " ",
			MD5Lower: " ",
		},
	})
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}
	if got[0].MD5 != "m1" || got[0].Day != "2026-01-02" || got[0].Size != "12" {
		t.Fatalf("mapped item=%+v", got[0])
	}
}

func TestGetFolderData_Branches(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
		if _, err := svc.getFolderData(context.Background(), "/gateway/folders/root", "x"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("buildURL error", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
		svc.baseURL = "http://[::1"
		if _, err := svc.getFolderData(context.Background(), "/gateway/folders/root", "x"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("decode error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
			case "/gateway/folders/root":
				_, _ = w.Write([]byte("{"))
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		if _, err := svc.getFolderData(context.Background(), "/gateway/folders/root", "x"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("folderList nil and file mapping", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
			case "/gateway/folders/root":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"path":       " /root ",
					"folderList": nil,
					"fileList": []map[string]any{
						{
							"id":       1,
							"fileName": "a.jpg",
							"fileType": "JPEG",
							"size":     12,
							"tokenAt":  "2026-01-02 01:02:03",
							"MD5":      "",
							"md5":      "m1",
						},
					},
					"trashNum": 2,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		content, err := svc.getFolderData(context.Background(), "/gateway/folders/root", "x")
		if err != nil {
			t.Fatalf("getFolderData error: %v", err)
		}
		if content.Path != "/root" || len(content.FolderList) != 0 || len(content.FileList) != 1 {
			t.Fatalf("content=%+v", content)
		}
	})
}

func TestGetFolderBreadcrumbs_InvalidFolderID(t *testing.T) {
	svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
	if _, err := svc.GetFolderBreadcrumbs(context.Background(), 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPaginateMtPhotoFolderItems_Branches(t *testing.T) {
	if got := paginateMtPhotoFolderItems(nil, 1, 10); len(got) != 0 {
		t.Fatalf("nil input should return empty: %+v", got)
	}
	if got := paginateMtPhotoFolderItems([]MtPhotoFolderFileItem{{ID: 1}}, 1, 0); len(got) != 0 {
		t.Fatalf("pageSize<=0 should return empty: %+v", got)
	}
	if got := paginateMtPhotoFolderItems([]MtPhotoFolderFileItem{{ID: 1}}, 2, 1); len(got) != 0 {
		t.Fatalf("start>=len should return empty: %+v", got)
	}
}

func TestMapMtPhotoTimelineFiles_AndMergeBranches(t *testing.T) {
	if out := mapMtPhotoTimelineFiles(nil); len(out) != 0 {
		t.Fatalf("expected empty timeline")
	}

	timeline := mapMtPhotoTimelineFiles([]mtPhotoFolderFilesGroup{
		{
			Day: "",
			List: []mtPhotoFileItem{
				{ID: 11, FileType: "", MD5: "", VMD5: "m-vmd5", Status: 0},
				{ID: 12, FileType: "MP4", MD5: "m-video", Duration: 2.5, Status: 1},
			},
		},
		{
			Day: "2026-01-03 10:20:30",
			List: []mtPhotoFileItem{
				{ID: 21, FileType: "JPEG", MD5: "m-image", Status: 2},
			},
		},
	})
	if len(timeline) != 3 {
		t.Fatalf("timeline len=%d", len(timeline))
	}
	if timeline[0].Day != "2026-01-03" {
		t.Fatalf("sort day expected 2026-01-03, got %q", timeline[0].Day)
	}

	if out := mergeFolderTimelineWithDetail(nil, []MtPhotoFolderFileItem{{ID: 1}}); len(out) != 0 {
		t.Fatalf("empty timeline should stay empty")
	}

	onlyTimeline := []MtPhotoFolderFileItem{{ID: 1, MD5: "m1"}}
	if out := mergeFolderTimelineWithDetail(onlyTimeline, nil); len(out) != 1 || out[0].ID != 1 {
		t.Fatalf("empty detail should keep timeline: %+v", out)
	}

	merged := mergeFolderTimelineWithDetail(
		[]MtPhotoFolderFileItem{
			{ID: 999, MD5: "m-vmd5", FileType: "", Day: "", Type: "", Status: 0},
			{ID: 22, MD5: "m-miss", FileType: "", Day: "", Type: "", Status: 0},
		},
		[]MtPhotoFolderFileItem{
			{
				ID:       1,
				MD5:      "m-vmd5",
				FileName: "a.jpg",
				Size:     "12",
				TokenAt:  "2026-01-03 10:00:00",
				Day:      "2026-01-03",
				FileType: "JPEG",
				Width:    100,
				Height:   200,
				Duration: func() *float64 { v := 1.5; return &v }(),
				Status:   2,
			},
		},
	)
	if len(merged) != 2 {
		t.Fatalf("merged len=%d", len(merged))
	}
	if merged[0].FileName != "a.jpg" || merged[0].FileType != "JPEG" || merged[0].Type == "" || merged[0].Duration == nil || merged[0].Status != 2 {
		t.Fatalf("merged[0]=%+v", merged[0])
	}
}

func TestGetFolderTimelineFiles_AndGetFolderContentPage_Branches(t *testing.T) {
	t.Run("getFolderTimelineFiles basic errors", func(t *testing.T) {
		svc := NewMtPhotoService("", "", "", "", "/lsp", nil)
		if _, _, err := svc.getFolderTimelineFiles(context.Background(), 1); err == nil {
			t.Fatalf("expected not configured error")
		}

		svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
		if _, _, err := svc.getFolderTimelineFiles(context.Background(), 0); err == nil {
			t.Fatalf("expected folderId error")
		}

		svc.baseURL = "http://[::1"
		if _, _, err := svc.getFolderTimelineFiles(context.Background(), 1); err == nil {
			t.Fatalf("expected buildURL error")
		}
	})

	t.Run("getFolderTimelineFiles status/decode/total fallback", func(t *testing.T) {
		badStatus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
			case "/gateway/folderFiles/1":
				w.WriteHeader(http.StatusBadGateway)
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(badStatus.Close)
		svc := NewMtPhotoService(badStatus.URL, "u", "p", "", "/lsp", badStatus.Client())
		if _, _, err := svc.getFolderTimelineFiles(context.Background(), 1); err == nil {
			t.Fatalf("expected status error")
		}

		badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
			case "/gateway/folderFiles/2":
				_, _ = w.Write([]byte("{"))
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(badJSON.Close)
		svc = NewMtPhotoService(badJSON.URL, "u", "p", "", "/lsp", badJSON.Client())
		if _, _, err := svc.getFolderTimelineFiles(context.Background(), 2); err == nil {
			t.Fatalf("expected decode error")
		}

		ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
			case "/gateway/folderFiles/3":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"result": []map[string]any{
						{
							"day": "2026-01-03",
							"list": []map[string]any{
								{"id": 1, "fileType": "JPEG", "MD5": "m1"},
							},
						},
					},
					"totalCount": 0,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(ok.Close)
		svc = NewMtPhotoService(ok.URL, "u", "p", "", "/lsp", ok.Client())
		items, total, err := svc.getFolderTimelineFiles(context.Background(), 3)
		if err != nil || len(items) != 1 || total != 1 {
			t.Fatalf("items=%v total=%d err=%v", items, total, err)
		}
	})

	t.Run("GetFolderContentPage normalization and timeline merge", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "t",
					"auth_code":    "ac",
					"expires_in":   time.Now().Add(1 * time.Hour).UnixMilli(),
				})
			case "/gateway/foldersV2/7":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"path": "/p",
					"fileList": []map[string]any{
						{
							"id":       7,
							"fileName": "detail.jpg",
							"fileType": "JPEG",
							"size":     "12",
							"tokenAt":  "2026-01-03 11:22:33",
							"MD5":      "m1",
							"status":   2,
						},
					},
				})
			case "/gateway/folderFiles/7":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"result": []map[string]any{
						{
							"day": "bad-day",
							"list": []map[string]any{
								{"id": 700, "fileType": "", "MD5": "m1", "status": 0},
							},
						},
					},
					"totalCount": 1,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		content, total, pages, err := svc.GetFolderContentPage(context.Background(), 7, 0, 500, true)
		if err != nil {
			t.Fatalf("GetFolderContentPage error: %v", err)
		}
		if total != 1 || pages != 1 || len(content.FileList) != 1 {
			t.Fatalf("unexpected result total/pages/len=%d/%d/%d", total, pages, len(content.FileList))
		}
		if content.FileList[0].FileName != "detail.jpg" || content.FileList[0].Type == "" {
			t.Fatalf("merged item=%+v", content.FileList[0])
		}
	})

	t.Run("GetFolderContentPage invalid folder", func(t *testing.T) {
		svc := NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
		if _, _, _, err := svc.GetFolderContentPage(context.Background(), 0, 1, 60, false); err == nil {
			t.Fatalf("expected error")
		}
	})
}

