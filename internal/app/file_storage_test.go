package app

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFileStorageService_IsValidMediaType(t *testing.T) {
	svc := &FileStorageService{}

	cases := []struct {
		name        string
		contentType string
		want        bool
	}{
		{name: "empty", contentType: "", want: false},
		{name: "jpeg", contentType: "image/jpeg", want: true},
		{name: "pngUpper", contentType: "IMAGE/PNG", want: true},
		{name: "mp4", contentType: "video/mp4", want: true},
		{name: "unsupported", contentType: "text/plain", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.IsValidMediaType(tc.contentType); got != tc.want {
				t.Fatalf("IsValidMediaType(%q)=%v, want %v", tc.contentType, got, tc.want)
			}
		})
	}
}

func TestFileStorageService_CategoryFromContentType(t *testing.T) {
	svc := &FileStorageService{}

	cases := []struct {
		name        string
		contentType string
		want        string
	}{
		{name: "empty", contentType: "", want: "file"},
		{name: "jpeg", contentType: "image/jpeg", want: "image"},
		{name: "mp4", contentType: "video/mp4", want: "video"},
		{name: "unsupported", contentType: "application/octet-stream", want: "file"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.CategoryFromContentType(tc.contentType); got != tc.want {
				t.Fatalf("CategoryFromContentType(%q)=%q, want %q", tc.contentType, got, tc.want)
			}
		})
	}
}

func TestFileStorageService_FileExtension(t *testing.T) {
	svc := &FileStorageService{}

	cases := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "empty", filename: "", want: ""},
		{name: "noDot", filename: "abc", want: ""},
		{name: "dotAtEnd", filename: "a.", want: ""},
		{name: "normal", filename: "a.png", want: "png"},
		{name: "upper", filename: "A.JPG", want: "jpg"},
		{name: "withSpaces", filename: "  a.webp  ", want: "webp"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.FileExtension(tc.filename); got != tc.want {
				t.Fatalf("FileExtension(%q)=%q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

func TestFileStorageService_SaveReadDeleteAndMD5(t *testing.T) {
	tempDir := t.TempDir()
	svc := &FileStorageService{baseUploadAbs: tempDir}

	content := []byte("hello-file-storage")
	req, fileHeader := newMultipartRequest(
		t,
		"POST",
		"http://example.com/upload",
		"file",
		"test.PNG",
		"image/png",
		content,
		map[string]string{},
	)
	_ = req

	localPath, err := svc.SaveFile(fileHeader, "image")
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	if !strings.HasPrefix(localPath, "/images/") {
		t.Fatalf("localPath=%q, want prefix %q", localPath, "/images/")
	}
	if !strings.HasSuffix(localPath, ".png") {
		t.Fatalf("localPath=%q, want suffix %q", localPath, ".png")
	}

	now := time.Now()
	year, month, day := now.Format("2006"), now.Format("01"), now.Format("02")
	wantDir := "/images/" + year + "/" + month + "/" + day + "/"
	if !strings.Contains(localPath, wantDir) {
		t.Fatalf("localPath=%q, want contain %q", localPath, wantDir)
	}

	gotBytes, err := svc.ReadLocalFile(localPath)
	if err != nil {
		t.Fatalf("ReadLocalFile failed: %v", err)
	}
	if string(gotBytes) != string(content) {
		t.Fatalf("ReadLocalFile content=%q, want %q", string(gotBytes), string(content))
	}

	wantMD5 := md5.Sum(content)
	gotMD5, err := svc.CalculateMD5FromLocalPath(localPath)
	if err != nil {
		t.Fatalf("CalculateMD5FromLocalPath failed: %v", err)
	}
	if gotMD5 != hex.EncodeToString(wantMD5[:]) {
		t.Fatalf("md5=%q, want %q", gotMD5, hex.EncodeToString(wantMD5[:]))
	}

	if ok := svc.DeleteFile(localPath); !ok {
		t.Fatalf("DeleteFile(%q)=false, want true", localPath)
	}
	if ok := svc.DeleteFile(localPath); ok {
		t.Fatalf("DeleteFile(%q)=true after deleted, want false", localPath)
	}
	if _, err := svc.ReadLocalFile(localPath); err == nil {
		t.Fatalf("ReadLocalFile(%q) expected error after delete", localPath)
	}
}

func TestFileStorageService_SaveTempVideoExtractInput(t *testing.T) {
	tempDir := t.TempDir()
	svc := &FileStorageService{baseUploadAbs: tempDir, baseTempAbs: tempDir}

	content := []byte("temp-video")
	_, fileHeader := newMultipartRequest(
		t,
		"POST",
		"http://example.com/upload",
		"file",
		"test.mp4",
		"video/mp4",
		content,
		map[string]string{},
	)

	localPath, err := svc.SaveTempVideoExtractInput(fileHeader)
	if err != nil {
		t.Fatalf("SaveTempVideoExtractInput failed: %v", err)
	}
	if !strings.HasPrefix(localPath, "/tmp/video_extract_inputs/") {
		t.Fatalf("localPath=%q, want prefix %q", localPath, "/tmp/video_extract_inputs/")
	}
	if !strings.HasSuffix(localPath, ".mp4") {
		t.Fatalf("localPath=%q, want suffix %q", localPath, ".mp4")
	}

	inner := strings.TrimPrefix(localPath, "/tmp/video_extract_inputs/")
	full := filepath.Join(tempDir, filepath.FromSlash(inner))
	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		t.Fatalf("saved file not found: %v", err)
	}
}

func TestFileStorageService_FindLocalPathByMD5(t *testing.T) {
	tempDir := t.TempDir()
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := &FileStorageService{db: wrapMySQLDB(db), baseUploadAbs: tempDir}

	localPath := "/images/2026/01/10/test.png"
	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
		WithArgs("md5value").
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow(localPath))

	got, err := svc.FindLocalPathByMD5(context.Background(), "md5value")
	if err != nil {
		t.Fatalf("FindLocalPathByMD5 failed: %v", err)
	}
	if got != localPath {
		t.Fatalf("FindLocalPathByMD5=%q, want %q", got, localPath)
	}

	// 文件不存在时应返回空字符串
	if err := os.Remove(full); err != nil {
		t.Fatalf("remove file failed: %v", err)
	}

	mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
		WithArgs("md5missing").
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow(localPath))

	got, err = svc.FindLocalPathByMD5(context.Background(), "md5missing")
	if err != nil {
		t.Fatalf("FindLocalPathByMD5 failed: %v", err)
	}
	if got != "" {
		t.Fatalf("FindLocalPathByMD5=%q, want empty", got)
	}
}
