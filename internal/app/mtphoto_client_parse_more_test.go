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

func TestMtPhotoParseHelperFunctions(t *testing.T) {
	t.Run("parseMtPhotoAnyInt64 covers value kinds", func(t *testing.T) {
		tests := []struct {
			name string
			raw  map[string]any
			want int64
		}{
			{name: "int", raw: map[string]any{"k": int(1)}, want: 1},
			{name: "int8", raw: map[string]any{"k": int8(2)}, want: 2},
			{name: "int16", raw: map[string]any{"k": int16(3)}, want: 3},
			{name: "int32", raw: map[string]any{"k": int32(4)}, want: 4},
			{name: "int64", raw: map[string]any{"k": int64(5)}, want: 5},
			{name: "uint", raw: map[string]any{"k": uint(6)}, want: 6},
			{name: "uint8", raw: map[string]any{"k": uint8(7)}, want: 7},
			{name: "uint16", raw: map[string]any{"k": uint16(8)}, want: 8},
			{name: "uint32", raw: map[string]any{"k": uint32(9)}, want: 9},
			{name: "uint64", raw: map[string]any{"k": uint64(10)}, want: 10},
			{name: "float32", raw: map[string]any{"k": float32(11.9)}, want: 11},
			{name: "float64", raw: map[string]any{"k": float64(12.9)}, want: 12},
			{name: "json number int", raw: map[string]any{"k": json.Number("13")}, want: 13},
			{name: "json number float", raw: map[string]any{"k": json.Number("14.8")}, want: 14},
			{name: "string", raw: map[string]any{"k": " 15 "}, want: 15},
			{name: "trimmed empty string", raw: map[string]any{"k": "  "}, want: 0},
			{name: "invalid string", raw: map[string]any{"k": "x"}, want: 0},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := parseMtPhotoAnyInt64(tc.raw, "k"); got != tc.want {
					t.Fatalf("got=%d want=%d", got, tc.want)
				}
			})
		}

		if got := parseMtPhotoAnyInt64(map[string]any{"k": uint64(^uint64(0))}, "k"); got != 0 {
			t.Fatalf("overflow uint64 should be skipped, got=%d", got)
		}
		if got := parseMtPhotoAnyInt64(nil, "k"); got != 0 {
			t.Fatalf("nil raw should be 0")
		}
		if got := parseMtPhotoAnyInt64(map[string]any{"k": 1}); got != 0 {
			t.Fatalf("empty keys should be 0")
		}
	})

	t.Run("parseMtPhotoAnyString covers value kinds", func(t *testing.T) {
		if got := parseMtPhotoAnyString(map[string]any{"k": "  abc  "}, "k"); got != "abc" {
			t.Fatalf("string got=%q", got)
		}
		if got := parseMtPhotoAnyString(map[string]any{"k": json.Number("123")}, "k"); got != "123" {
			t.Fatalf("json number got=%q", got)
		}
		if got := parseMtPhotoAnyString(map[string]any{"k": 456}, "k"); got != "456" {
			t.Fatalf("default fmt got=%q", got)
		}
		if got := parseMtPhotoAnyString(map[string]any{"k": "  "}, "k"); got != "" {
			t.Fatalf("blank should be empty")
		}
		if got := parseMtPhotoAnyString(nil, "k"); got != "" {
			t.Fatalf("nil raw should be empty")
		}
	})

	t.Run("normalize folder path and parse time", func(t *testing.T) {
		if got := normalizeMtPhotoFolderPath(" "); got != "" {
			t.Fatalf("empty folder path should be empty")
		}
		if got := normalizeMtPhotoFolderPath("."); got != "" {
			t.Fatalf("dot folder path should be empty")
		}
		if got := normalizeMtPhotoFolderPath("a\\b"); got != "a\\b" {
			t.Fatalf("path normalize got=%q", got)
		}

		if parseMtPhotoTimeValue("") != 0 {
			t.Fatalf("blank time should be 0")
		}
		if got := parseMtPhotoTimeValue("2026-03-01T12:00:00Z"); got <= 0 {
			t.Fatalf("rfc3339 parse failed: %d", got)
		}
		if got := parseMtPhotoTimeValue("2026-03-01 12:00:00"); got <= 0 {
			t.Fatalf("datetime parse failed: %d", got)
		}
		if got := parseMtPhotoTimeValue("2026-03-01"); got <= 0 {
			t.Fatalf("date parse failed: %d", got)
		}
		if got := parseMtPhotoTimeValue("1700000000001"); got != 1700000000001 {
			t.Fatalf("millis parse failed: %d", got)
		}
		if got := parseMtPhotoTimeValue("1700000000"); got != 1700000000000 {
			t.Fatalf("seconds parse failed: %d", got)
		}
		if got := parseMtPhotoTimeValue("bad"); got != 0 {
			t.Fatalf("invalid time should be 0")
		}
	})
}

func TestMtPhotoService_GetFileInfoAndListSameMedia_ErrorAndFallbackBranches(t *testing.T) {
	t.Run("GetFileInfo validation and transport errors", func(t *testing.T) {
		svc := NewMtPhotoService("", "u", "p", "", "/lsp", nil)
		if _, err := svc.GetFileInfo(context.Background(), 1, "m"); err == nil {
			t.Fatalf("not configured should fail")
		}

		svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
		if _, err := svc.GetFileInfo(context.Background(), 0, "m"); err == nil {
			t.Fatalf("invalid id should fail")
		}
		if _, err := svc.GetFileInfo(context.Background(), 1, " "); err == nil {
			t.Fatalf("blank md5 should fail")
		}

		svc.baseURL = "http://[::1"
		if _, err := svc.GetFileInfo(context.Background(), 1, "m"); err == nil {
			t.Fatalf("bad base url should fail")
		}

		svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		})})
		if _, err := svc.GetFileInfo(context.Background(), 1, "m"); err == nil {
			t.Fatalf("doRequest error should fail")
		}
	})

	t.Run("GetFileInfo non-2xx, decode error and success fallback md5", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "auth_code": "ac", "expires_in": time.Now().Add(time.Hour).UnixMilli()})
			case "/gateway/fileInfo/1/m1":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("bad"))
			case "/gateway/fileInfo/2/m2":
				_, _ = w.Write([]byte("{"))
			case "/gateway/fileInfo/3/m3":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ID":        "3",
					"file_path": " /lsp/a/b.jpg ",
					"folder_id": json.Number("9"),
					"token_at":  " 2026-03-01T01:02:03Z ",
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		svc := NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())
		if _, err := svc.GetFileInfo(context.Background(), 1, "m1"); err == nil {
			t.Fatalf("non-2xx should fail")
		}
		if _, err := svc.GetFileInfo(context.Background(), 2, "m2"); err == nil {
			t.Fatalf("decode error should fail")
		}
		info, err := svc.GetFileInfo(context.Background(), 3, "m3")
		if err != nil {
			t.Fatalf("GetFileInfo success err=%v", err)
		}
		if info.ID != 3 || info.FolderID != 9 || info.MD5 != "m3" || info.FilePath != "/lsp/a/b.jpg" {
			t.Fatalf("info unexpected: %+v", info)
		}
	})

	t.Run("ListSameMediaByMD5 errors and enrich fallback", func(t *testing.T) {
		svc := NewMtPhotoService("", "u", "p", "", "/lsp", nil)
		if _, err := svc.ListSameMediaByMD5(context.Background(), "m"); err == nil {
			t.Fatalf("not configured should fail")
		}

		svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", nil)
		if _, err := svc.ListSameMediaByMD5(context.Background(), " "); err == nil {
			t.Fatalf("blank md5 should fail")
		}
		svc.baseURL = "http://[::1"
		if _, err := svc.ListSameMediaByMD5(context.Background(), "m"); err == nil {
			t.Fatalf("bad base url should fail")
		}

		svc = NewMtPhotoService("http://example.com", "u", "p", "", "/lsp", &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		})})
		if _, err := svc.ListSameMediaByMD5(context.Background(), "m"); err == nil {
			t.Fatalf("doRequest error should fail")
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth/login":
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "auth_code": "ac", "expires_in": time.Now().Add(time.Hour).UnixMilli()})
			case "/gateway/filesInMD5":
				var reqBody map[string]any
				_ = json.NewDecoder(r.Body).Decode(&reqBody)
				md5Value := strings.TrimSpace(toString(reqBody["MD5"]))
				if md5Value == "bad-status" {
					w.WriteHeader(http.StatusBadGateway)
					_, _ = w.Write([]byte("upstream bad"))
					return
				}
				if md5Value == "bad-json" {
					_, _ = w.Write([]byte("{"))
					return
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"id": 2, "filePath": "/root/b.jpg", "tokenAt": "", "md5": "", "folderId": 0},
					{"id": 1, "filePath": " /root/a.jpg ", "day": "2025-01-01", "folderId": 8, "folderPath": "", "folderName": ""},
					{"id": 3, "filePath": "   "},
					{"id": 4, "filePath": "/root/c.jpg", "tokenAt": "bad", "folderId": 0, "md5": "m4"},
				})
			case "/gateway/fileInfo/2/m-main":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":       2,
					"filePath": "/extra/dir/b.jpg",
					"folderId": 22,
					"tokenAt":  "2026-03-01T10:00:00Z",
				})
			case "/gateway/fileInfo/4/m4":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("x"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		svc = NewMtPhotoService(srv.URL, "u", "p", "", "/lsp", srv.Client())

		if _, err := svc.ListSameMediaByMD5(context.Background(), "bad-status"); err == nil {
			t.Fatalf("bad status should fail")
		}
		if _, err := svc.ListSameMediaByMD5(context.Background(), "bad-json"); err == nil {
			t.Fatalf("bad json should fail")
		}

		items, err := svc.ListSameMediaByMD5(context.Background(), "m-main")
		if err != nil {
			t.Fatalf("ListSameMediaByMD5 err=%v", err)
		}
		if len(items) != 3 {
			t.Fatalf("len(items)=%d, want 3", len(items))
		}
		if items[0].ID != 2 || items[0].FolderID != 22 || items[0].FolderPath != "/root" || items[0].FolderName != "root" || items[0].MD5 != "m-main" {
			t.Fatalf("item0 unexpected: %+v", items[0])
		}
		if items[1].ID != 1 || items[1].FolderName != "root" || !items[1].CanOpenFolder {
			t.Fatalf("item1 unexpected: %+v", items[1])
		}
		if items[2].ID != 4 || items[2].FolderID != 0 {
			t.Fatalf("item2 unexpected: %+v", items[2])
		}
	})
}
