package app

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newDiskMultipartFileHeader(t *testing.T, fieldName, filename, contentType string, content []byte, maxMemory int64) (*http.Request, *multipart.FileHeader) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="`+fieldName+`"; filename="`+filename+`"`)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(maxMemory); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}
	files := req.MultipartForm.File[fieldName]
	if len(files) == 0 {
		t.Fatalf("no file in field %q", fieldName)
	}
	return req, files[0]
}

func TestFileStorageService_CalculateMD5_Errors(t *testing.T) {
	svc := &FileStorageService{}
	if _, err := svc.CalculateMD5(nil); err == nil {
		t.Fatalf("expected error")
	}

	// Open error: 删除 multipart 临时文件后再计算
	{
		req, fh := newDiskMultipartFileHeader(t, "file", "a.png", "image/png", []byte("hello"), 1)
		req.MultipartForm.RemoveAll()

		if _, err := svc.CalculateMD5(fh); err == nil {
			t.Fatalf("expected error")
		}
	}

	// Copy error: 将 multipart 临时文件替换为目录
	{
		req, fh := newDiskMultipartFileHeader(t, "file", "a.png", "image/png", []byte("hello"), 1)
		f, err := fh.Open()
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		osFile, ok := f.(*os.File)
		if !ok {
			f.Close()
			req.MultipartForm.RemoveAll()
			t.Fatalf("expected *os.File")
		}
		tmp := osFile.Name()
		_ = osFile.Close()
		req.MultipartForm.RemoveAll()

		if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.Mkdir(tmp, 0o755); err != nil {
			t.Fatalf("Mkdir tmp: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(tmp) })

		if _, err := svc.CalculateMD5(fh); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestFileStorageService_CalculateMD5FromLocalPath_Error(t *testing.T) {
	svc := &FileStorageService{baseUploadAbs: t.TempDir()}
	if _, err := svc.CalculateMD5FromLocalPath(""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFileStorageService_SaveFile_Errors(t *testing.T) {
	svc := &FileStorageService{baseUploadAbs: t.TempDir()}
	if _, err := svc.SaveFile(nil, "image"); err == nil {
		t.Fatalf("expected error")
	}

	_, emptyFH := newMultipartRequest(
		t,
		http.MethodPost,
		"http://example.com/upload",
		"file",
		"empty.png",
		"image/png",
		nil,
		map[string]string{},
	)
	if _, err := svc.SaveFile(emptyFH, "image"); err == nil {
		t.Fatalf("expected error")
	}

	// MkdirAll error: baseUploadAbs 指向文件
	{
		tempDir := t.TempDir()
		baseFile := filepath.Join(tempDir, "upload_file")
		if err := os.WriteFile(baseFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		svc2 := &FileStorageService{baseUploadAbs: baseFile}
		_, fh := newMultipartRequest(
			t,
			http.MethodPost,
			"http://example.com/upload",
			"file",
			"a.png",
			"image/png",
			[]byte("x"),
			map[string]string{},
		)
		if _, err := svc2.SaveFile(fh, "image"); err == nil {
			t.Fatalf("expected error")
		}
	}

	// file.Open error
	{
		tempDir := t.TempDir()
		svc2 := &FileStorageService{baseUploadAbs: tempDir}
		req, fh := newDiskMultipartFileHeader(t, "file", "a.png", "image/png", []byte("hello"), 1)
		req.MultipartForm.RemoveAll()

		if _, err := svc2.SaveFile(fh, "image"); err == nil {
			t.Fatalf("expected error")
		}
	}

	// os.Create error: 预创建只读目录
	{
		tempDir := t.TempDir()
		svc2 := &FileStorageService{baseUploadAbs: tempDir}
		_, fh := newMultipartRequest(
			t,
			http.MethodPost,
			"http://example.com/upload",
			"file",
			"a.png",
			"image/png",
			[]byte("hello"),
			map[string]string{},
		)

		storageDir := svc2.storageDirectory("image")
		if err := os.MkdirAll(storageDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.Chmod(storageDir, 0o555); err != nil {
			t.Fatalf("Chmod: %v", err)
		}
		if _, err := svc2.SaveFile(fh, "image"); err == nil {
			t.Fatalf("expected error")
		}
		if err := os.Chmod(storageDir, 0o755); err != nil {
			t.Fatalf("Chmod restore: %v", err)
		}

		_ = storageDir
	}

	// io.Copy error: 将上传临时文件替换为目录
	{
		tempDir := t.TempDir()
		svc2 := &FileStorageService{baseUploadAbs: tempDir}
		req, fh := newDiskMultipartFileHeader(t, "file", "a.png", "image/png", []byte("hello"), 1)
		f, err := fh.Open()
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		osFile, ok := f.(*os.File)
		if !ok {
			_ = f.Close()
			req.MultipartForm.RemoveAll()
			t.Fatalf("expected *os.File")
		}
		tmp := osFile.Name()
		_ = osFile.Close()
		req.MultipartForm.RemoveAll()

		if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.Mkdir(tmp, 0o755); err != nil {
			t.Fatalf("Mkdir tmp: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(tmp) })

		if _, err := svc2.SaveFile(fh, "image"); err == nil {
			t.Fatalf("expected error")
		}
	}
}

type alwaysErrorReader struct{}

func (alwaysErrorReader) Read(p []byte) (int, error) { return 0, errors.New("read fail") }

func TestFileStorageService_SaveFileFromReader_ErrorsAndDefaults(t *testing.T) {
	svc := &FileStorageService{baseUploadAbs: t.TempDir()}

	if _, _, _, err := svc.SaveFileFromReader("a.png", "image/png", nil); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, _, err := svc.SaveFileFromReader("a.png", "", strings.NewReader("x")); err == nil {
		t.Fatalf("expected error")
	}

	// originalFilename 为空 -> "imported"（无扩展名）
	localPath, n, md5Value, err := svc.SaveFileFromReader("", "image/png", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("SaveFileFromReader: %v", err)
	}
	if localPath == "" || n != 5 || md5Value == "" {
		t.Fatalf("localPath=%q n=%d md5=%q", localPath, n, md5Value)
	}

	// os.Create error
	{
		svc2 := &FileStorageService{baseUploadAbs: t.TempDir()}
		storageDir := svc2.storageDirectory("image")
		if err := os.MkdirAll(storageDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.Chmod(storageDir, 0o555); err != nil {
			t.Fatalf("Chmod: %v", err)
		}

		if _, _, _, err := svc2.SaveFileFromReader("a.png", "image/png", strings.NewReader("x")); err == nil {
			t.Fatalf("expected error")
		}
		if err := os.Chmod(storageDir, 0o755); err != nil {
			t.Fatalf("Chmod restore: %v", err)
		}
	}

	// io.Copy error
	{
		svc2 := &FileStorageService{baseUploadAbs: t.TempDir()}
		if _, _, _, err := svc2.SaveFileFromReader("a.png", "image/png", alwaysErrorReader{}); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestFileStorageService_DeleteAndRead_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	svc := &FileStorageService{baseUploadAbs: tempDir}

	if ok := svc.DeleteFile(" "); ok {
		t.Fatalf("expected false")
	}
	if _, err := svc.ReadLocalFile(" "); err == nil {
		t.Fatalf("expected error")
	}

	dirLocal := "/images/dir"
	fullDir := filepath.Join(tempDir, "images", "dir")
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if ok := svc.DeleteFile(dirLocal); ok {
		t.Fatalf("expected false for dir")
	}
	if _, err := svc.ReadLocalFile(dirLocal); err == nil {
		t.Fatalf("expected error for dir")
	}
}

func TestFileStorageService_FindLocalPathByMD5_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	svc := &FileStorageService{db: db, baseUploadAbs: tempDir}
	if got, err := svc.FindLocalPathByMD5(context.Background(), " "); err != nil || got != "" {
		t.Fatalf("got=%q err=%v", got, err)
	}

	mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}))

	if got, err := svc.FindLocalPathByMD5(context.Background(), "missing"); err != nil || got != "" {
		t.Fatalf("got=%q err=%v", got, err)
	}

	mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
		WithArgs("boom").
		WillReturnError(errors.New("query fail"))
	if _, err := svc.FindLocalPathByMD5(context.Background(), "boom"); err == nil {
		t.Fatalf("expected error")
	}

	localPath := "/images/2026/01/10/dir.png"
	full := filepath.Join(tempDir, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	mock.ExpectQuery(`SELECT local_path FROM media_upload_history WHERE file_md5 = \? LIMIT 1`).
		WithArgs("md5dir").
		WillReturnRows(sqlmock.NewRows([]string{"local_path"}).AddRow(localPath))

	if got, err := svc.FindLocalPathByMD5(context.Background(), "md5dir"); err != nil || got != "" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestFileStorageService_GenerateUniqueFilename_NoExtension(t *testing.T) {
	svc := &FileStorageService{}
	name := svc.generateUniqueFilename("noext")
	if strings.Contains(name, ".") {
		t.Fatalf("unexpected dot: %q", name)
	}
	if !strings.Contains(name, "_") {
		t.Fatalf("unexpected format: %q", name)
	}
}

func TestFileStorageService_FilepathRel_Errors(t *testing.T) {
	old := filepathRelFn
	t.Cleanup(func() { filepathRelFn = old })
	filepathRelFn = func(basepath, targpath string) (string, error) {
		return "", errors.New("rel fail")
	}

	{
		svc := &FileStorageService{baseUploadAbs: t.TempDir()}
		_, fh := newMultipartRequest(
			t,
			http.MethodPost,
			"http://example.com/upload",
			"file",
			"a.png",
			"image/png",
			[]byte("hello"),
			map[string]string{},
		)
		if _, err := svc.SaveFile(fh, "image"); err == nil {
			t.Fatalf("expected error")
		}
	}

	{
		svc := &FileStorageService{baseUploadAbs: t.TempDir()}
		if _, _, _, err := svc.SaveFileFromReader("a.png", "image/png", strings.NewReader("hello")); err == nil {
			t.Fatalf("expected error")
		}
	}
}

func TestFileStorageService_SaveTempVideoExtractInput_ErrorsAndFallbackBase(t *testing.T) {
	// nil/empty
	{
		svc := &FileStorageService{}
		if _, err := svc.SaveTempVideoExtractInput(nil); err == nil {
			t.Fatalf("expected error")
		}
		_, fh := newMultipartRequest(
			t,
			http.MethodPost,
			"http://example.com/upload",
			"file",
			"empty.mp4",
			"video/mp4",
			nil,
			map[string]string{},
		)
		if _, err := svc.SaveTempVideoExtractInput(fh); err == nil {
			t.Fatalf("expected error")
		}
	}

	// baseTempAbs 为空 -> 回落到 os.TempDir()/video_extract_inputs（用 TMPDIR 隔离）
	{
		tmpRoot := t.TempDir()
		t.Setenv("TMPDIR", tmpRoot)
		svc := &FileStorageService{baseTempAbs: " "}

		_, fh := newMultipartRequest(
			t,
			http.MethodPost,
			"http://example.com/upload",
			"file",
			"a.mp4",
			"video/mp4",
			[]byte("hello"),
			map[string]string{},
		)

		localPath, err := svc.SaveTempVideoExtractInput(fh)
		if err != nil {
			t.Fatalf("SaveTempVideoExtractInput: %v", err)
		}
		if !strings.HasPrefix(localPath, "/tmp/video_extract_inputs/") {
			t.Fatalf("localPath=%q", localPath)
		}

		inner := strings.TrimPrefix(localPath, "/tmp/video_extract_inputs/")
		full := filepath.Join(tmpRoot, "video_extract_inputs", filepath.FromSlash(inner))
		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			t.Fatalf("saved file not found: %v", err)
		}
	}

	// MkdirAll error: baseTempAbs 指向文件
	{
		tempDir := t.TempDir()
		baseFile := filepath.Join(tempDir, "tmpfile")
		if err := os.WriteFile(baseFile, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		svc := &FileStorageService{baseTempAbs: baseFile}
		_, fh := newMultipartRequest(
			t,
			http.MethodPost,
			"http://example.com/upload",
			"file",
			"a.mp4",
			"video/mp4",
			[]byte("hello"),
			map[string]string{},
		)
		if _, err := svc.SaveTempVideoExtractInput(fh); err == nil {
			t.Fatalf("expected error")
		}
	}

	// file.Open error
	{
		svc := &FileStorageService{baseTempAbs: t.TempDir()}
		req, fh := newDiskMultipartFileHeader(t, "file", "a.mp4", "video/mp4", []byte("hello"), 1)
		req.MultipartForm.RemoveAll()
		if _, err := svc.SaveTempVideoExtractInput(fh); err == nil {
			t.Fatalf("expected error")
		}
	}

	// os.Create error: 预创建只读目录
	{
		tempDir := t.TempDir()
		svc := &FileStorageService{baseTempAbs: tempDir}

		_, fh := newMultipartRequest(
			t,
			http.MethodPost,
			"http://example.com/upload",
			"file",
			"a.mp4",
			"video/mp4",
			[]byte("hello"),
			map[string]string{},
		)

		now := time.Now()
		storageDir := filepath.Join(tempDir, now.Format("2006"), now.Format("01"), now.Format("02"))
		if err := os.MkdirAll(storageDir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.Chmod(storageDir, 0o555); err != nil {
			t.Fatalf("Chmod: %v", err)
		}
		if _, err := svc.SaveTempVideoExtractInput(fh); err == nil {
			t.Fatalf("expected error")
		}
		_ = os.Chmod(storageDir, 0o755)
	}

	// io.Copy error: 将上传临时文件替换为目录
	{
		svc := &FileStorageService{baseTempAbs: t.TempDir()}
		req, fh := newDiskMultipartFileHeader(t, "file", "a.mp4", "video/mp4", []byte("hello"), 1)
		f, err := fh.Open()
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		osFile, ok := f.(*os.File)
		if !ok {
			_ = f.Close()
			req.MultipartForm.RemoveAll()
			t.Fatalf("expected *os.File")
		}
		tmp := osFile.Name()
		_ = osFile.Close()
		req.MultipartForm.RemoveAll()

		if err := os.MkdirAll(filepath.Dir(tmp), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.Mkdir(tmp, 0o755); err != nil {
			t.Fatalf("Mkdir tmp: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(tmp) })

		if _, err := svc.SaveTempVideoExtractInput(fh); err == nil {
			t.Fatalf("expected error")
		}
	}
}
