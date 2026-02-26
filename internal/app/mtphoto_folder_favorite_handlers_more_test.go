package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseMtPhotoFolderFavoriteListOptions_NilAndValid(t *testing.T) {
	got := parseMtPhotoFolderFavoriteListOptions(nil)
	if got.TagMode != "" || got.SortBy != "" || got.SortOrder != "" || got.GroupBy != "" {
		t.Fatalf("nil request options=%+v", got)
	}

	req := httptest.NewRequest(http.MethodGet,
		"http://api.local/api/getMtPhotoFolderFavorites?tagKeyword=k&tagMode=all&sortBy=name&sortOrder=asc&groupBy=tag",
		nil,
	)
	got = parseMtPhotoFolderFavoriteListOptions(req)
	if got.TagKeyword != "k" || got.TagMode != "all" || got.SortBy != "name" || got.SortOrder != "asc" || got.GroupBy != "tag" {
		t.Fatalf("valid options=%+v", got)
	}
}

func TestSanitizeMtPhotoFolderFavoriteUpsertInput_MoreBranches(t *testing.T) {
	if err := sanitizeMtPhotoFolderFavoriteUpsertInput(nil); err == nil {
		t.Fatalf("expected nil input error")
	}

	cases := []struct {
		name string
		in   MtPhotoFolderFavoriteUpsertInput
		want string
	}{
		{
			name: "empty folderName",
			in: MtPhotoFolderFavoriteUpsertInput{
				FolderID:   1,
				FolderName: " ",
				FolderPath: "/a",
			},
			want: "folderName 不能为空",
		},
		{
			name: "empty folderPath",
			in: MtPhotoFolderFavoriteUpsertInput{
				FolderID:   1,
				FolderName: "a",
				FolderPath: " ",
			},
			want: "folderPath 不能为空",
		},
		{
			name: "invalid cover md5",
			in: MtPhotoFolderFavoriteUpsertInput{
				FolderID:   1,
				FolderName: "a",
				FolderPath: "/a",
				CoverMD5:   "xyz",
			},
			want: "coverMd5 参数非法",
		},
		{
			name: "too many tags",
			in: MtPhotoFolderFavoriteUpsertInput{
				FolderID:   1,
				FolderName: "a",
				FolderPath: "/a",
				Tags:       make([]string, mtPhotoFolderFavoriteMaxTags+1),
			},
			want: "标签数量不能超过",
		},
		{
			name: "note too long",
			in: MtPhotoFolderFavoriteUpsertInput{
				FolderID:   1,
				FolderName: "a",
				FolderPath: "/a",
				Note:       strings.Repeat("中", mtPhotoFolderFavoriteMaxNoteRunes+1),
			},
			want: "note 长度不能超过",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			if in.Tags != nil {
				for i := range in.Tags {
					in.Tags[i] = fmt.Sprintf("t%d", i)
				}
			}
			err := sanitizeMtPhotoFolderFavoriteUpsertInput(&in)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want contains %q", err, tc.want)
			}
		})
	}
}

func TestIsBadRequestError_Branches(t *testing.T) {
	if isBadRequestError(nil) {
		t.Fatalf("nil should be false")
	}
	if !isBadRequestError(errBadRequest("x")) {
		t.Fatalf("badRequest should be true")
	}
	if isBadRequestError(http.ErrBodyNotAllowed) {
		t.Fatalf("normal error should be false")
	}
}

func TestMtPhotoFolderFavoriteHandlers_MoreErrorBranches(t *testing.T) {
	t.Run("list service error", func(t *testing.T) {
		app := &App{mtPhotoFolderFavorite: NewMtPhotoFolderFavoriteService(nil)}
		req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoFolderFavorites", nil)
		rr := httptest.NewRecorder()
		app.handleGetMtPhotoFolderFavorites(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("upsert not initialized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/upsertMtPhotoFolderFavorite", strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		(&App{}).handleUpsertMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("upsert internal error from service", func(t *testing.T) {
		app := &App{mtPhotoFolderFavorite: NewMtPhotoFolderFavoriteService(nil)}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/upsertMtPhotoFolderFavorite",
			strings.NewReader(`{"folderId":1,"folderName":"a","folderPath":"/a"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		app.handleUpsertMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("remove not initialized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/removeMtPhotoFolderFavorite",
			strings.NewReader(`{"folderId":1}`))
		rr := httptest.NewRecorder()
		(&App{}).handleRemoveMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d", rr.Code)
		}
	})

	t.Run("remove service error", func(t *testing.T) {
		app := &App{mtPhotoFolderFavorite: NewMtPhotoFolderFavoriteService(nil)}
		req := httptest.NewRequest(http.MethodPost, "http://api.local/api/removeMtPhotoFolderFavorite",
			strings.NewReader(`{"folderId":1}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		app.handleRemoveMtPhotoFolderFavorite(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}
