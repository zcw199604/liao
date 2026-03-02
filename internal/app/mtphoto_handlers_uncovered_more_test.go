package app

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleGetMtPhotoSameMedia_ListErrorBranch(t *testing.T) {
	app := &App{mtPhoto: NewMtPhotoService("", "", "", "", "/lsp", nil)}
	req := httptest.NewRequest(http.MethodGet, "http://api.local/api/getMtPhotoSameMedia?md5=0123456789abcdef0123456789abcdef", nil)
	rr := httptest.NewRecorder()
	app.handleGetMtPhotoSameMedia(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMtPhotoPathHelpers_UncoveredBranches(t *testing.T) {
	if got := normalizeMtPhotoLookupLocalPath("/upload/images/%E4%BD%A0%E5%A5%BD.jpg?x=1#frag"); got != "/upload/images/你好.jpg" {
		t.Fatalf("decoded path=%q", got)
	}
	if got := normalizeMtPhotoLookupLocalPath("?x=1"); got != "" {
		t.Fatalf("query-only path should normalize to empty, got=%q", got)
	}
}

func TestCalculateMD5FromAbsPath_ReadErrorBranch(t *testing.T) {
	tmp := t.TempDir()
	sockPath := filepath.Join(tmp, "sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	defer ln.Close()

	if _, err := calculateMD5FromAbsPath(sockPath); err == nil {
		t.Fatalf("expected read error")
	}
}

func TestCalculateMD5FromSupportedLocalPath_UncoveredErrorBranches(t *testing.T) {
	app := &App{fileStorage: &FileStorageService{baseUploadAbs: t.TempDir()}}

	if _, err := app.calculateMD5FromSupportedLocalPath("/upload/"); err == nil || !strings.Contains(err.Error(), "仅支持本地 /upload/images 或 /upload/videos 文件") {
		t.Fatalf("err=%v", err)
	}

	if _, err := app.calculateMD5FromSupportedLocalPath("/upload/images/not-exists.jpg"); err == nil || !strings.Contains(err.Error(), "读取本地文件失败") {
		t.Fatalf("err=%v", err)
	}
}
