package app

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileStorageService 负责本地文件保存/删除/读取与 MD5 查询（兼容 Java 侧行为）。
type FileStorageService struct {
	db            *sql.DB
	baseUploadAbs string
	baseTempAbs   string
}

const tempVideoExtractInputsDir = "tmp/video_extract_inputs"

func NewFileStorageService(db *sql.DB) *FileStorageService {
	wd, err := os.Getwd()
	base := "upload"
	if err == nil && wd != "" {
		base = filepath.Join(wd, "upload")
	}
	tempBase := filepath.Join(os.TempDir(), "video_extract_inputs")
	return &FileStorageService{db: db, baseUploadAbs: base, baseTempAbs: tempBase}
}

var supportedMediaTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/gif":  {},
	"image/webp": {},
	"video/mp4":  {},
}

var mediaTypeCategory = map[string]string{
	"image/jpeg": "image",
	"image/png":  "image",
	"image/gif":  "image",
	"image/webp": "image",
	"video/mp4":  "video",
}

var openMultipartFileHeaderFn = func(file *multipart.FileHeader) (multipart.File, error) {
	return file.Open()
}

func (s *FileStorageService) IsValidMediaType(contentType string) bool {
	if contentType == "" {
		return false
	}
	_, ok := supportedMediaTypes[strings.ToLower(contentType)]
	return ok
}

func (s *FileStorageService) CategoryFromContentType(contentType string) string {
	if contentType == "" {
		return "file"
	}
	if cat, ok := mediaTypeCategory[strings.ToLower(contentType)]; ok {
		return cat
	}
	return "file"
}

func (s *FileStorageService) FileExtension(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return ""
	}
	dot := strings.LastIndex(filename, ".")
	if dot > 0 && dot < len(filename)-1 {
		return strings.ToLower(filename[dot+1:])
	}
	return ""
}

func (s *FileStorageService) CalculateMD5(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", fmt.Errorf("文件为空")
	}
	src, err := openMultipartFileHeaderFn(file)
	if err != nil {
		return "", err
	}
	defer src.Close()

	hasher := md5.New()
	if _, err := io.CopyBuffer(hasher, src, make([]byte, 8192)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *FileStorageService) CalculateMD5FromLocalPath(localPath string) (string, error) {
	data, err := s.ReadLocalFile(localPath)
	if err != nil {
		return "", err
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:]), nil
}

func (s *FileStorageService) SaveFile(file *multipart.FileHeader, fileType string) (string, error) {
	if file == nil || file.Size == 0 {
		return "", fmt.Errorf("文件为空")
	}

	originalFilename := file.Filename
	unique := s.generateUniqueFilename(originalFilename)
	storageDir := s.storageDirectory(fileType)

	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return "", fmt.Errorf("无法创建存储目录: %w", err)
	}

	src, err := openMultipartFileHeaderFn(file)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dstPath := filepath.Join(storageDir, unique)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	rel, err := filepathRelFn(s.baseUploadAbs, dstPath)
	if err != nil {
		return "", err
	}

	// 返回形如：/images/2025/12/19/xxx.jpg
	rel = filepath.ToSlash(rel)
	return "/" + rel, nil
}

// SaveTempVideoExtractInput 将“抽帧任务输入视频”保存到系统临时目录（默认 os.TempDir()/video_extract_inputs），避免写入挂载的 upload 目录。
// 返回可对外访问的 localPath（形如 /tmp/video_extract_inputs/yyyy/MM/dd/xxx.mp4，URL 访问路径为 /upload{localPath}）。
func (s *FileStorageService) SaveTempVideoExtractInput(file *multipart.FileHeader) (string, error) {
	if file == nil || file.Size == 0 {
		return "", fmt.Errorf("文件为空")
	}

	originalFilename := file.Filename
	unique := s.generateUniqueFilename(originalFilename)

	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")

	baseTempAbs := strings.TrimSpace(s.baseTempAbs)
	if baseTempAbs == "" {
		baseTempAbs = filepath.Join(os.TempDir(), "video_extract_inputs")
	}
	storageDir := filepath.Join(baseTempAbs, year, month, day)
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return "", fmt.Errorf("无法创建存储目录: %w", err)
	}

	src, err := openMultipartFileHeaderFn(file)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dstPath := filepath.Join(storageDir, unique)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	localPath := filepath.ToSlash(filepath.Join("/", tempVideoExtractInputsDir, year, month, day, unique))
	return localPath, nil
}

// SaveFileFromReader 将 reader 内容保存到本地 upload 目录，并返回可对外访问的 localPath。
// 说明：用于“导入外部系统本地文件”场景，避免必须构造 multipart.FileHeader。
func (s *FileStorageService) SaveFileFromReader(originalFilename, contentType string, src io.Reader) (localPath string, fileSize int64, md5Value string, err error) {
	storageDir := s.storageDirectory(s.CategoryFromContentType(contentType))
	return s.saveFileFromReaderToDir(originalFilename, contentType, src, storageDir)
}

// SaveFileFromReaderInSubdir 将 reader 内容保存到本地 upload/{subdir} 目录，并返回可对外访问的 localPath。
// 说明：用于区分不同来源的落盘目录（例如抖音导入：upload/douyin/...）。
func (s *FileStorageService) SaveFileFromReaderInSubdir(originalFilename, contentType string, src io.Reader, subdir string) (localPath string, fileSize int64, md5Value string, err error) {
	subdir = strings.TrimSpace(subdir)
	if subdir == "" {
		return s.SaveFileFromReader(originalFilename, contentType, src)
	}

	storageDir := s.storageDirectoryWithSubdir(s.CategoryFromContentType(contentType), subdir)
	return s.saveFileFromReaderToDir(originalFilename, contentType, src, storageDir)
}

func (s *FileStorageService) saveFileFromReaderToDir(originalFilename, contentType string, src io.Reader, storageDir string) (localPath string, fileSize int64, md5Value string, err error) {
	if src == nil {
		return "", 0, "", fmt.Errorf("文件为空")
	}

	originalFilename = strings.TrimSpace(originalFilename)
	if originalFilename == "" {
		originalFilename = "imported"
	}

	if strings.TrimSpace(contentType) == "" {
		return "", 0, "", fmt.Errorf("contentType 为空")
	}

	unique := s.generateUniqueFilename(originalFilename)

	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return "", 0, "", fmt.Errorf("无法创建存储目录: %w", err)
	}

	dstPath := filepath.Join(storageDir, unique)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", 0, "", err
	}
	defer func() { _ = dst.Close() }()

	var hasher hash.Hash = md5.New()
	w := io.MultiWriter(dst, hasher)
	n, err := io.Copy(w, src)
	if err != nil {
		return "", 0, "", err
	}

	rel, err := filepathRelFn(s.baseUploadAbs, dstPath)
	if err != nil {
		return "", 0, "", err
	}
	rel = filepath.ToSlash(rel)

	return "/" + rel, n, hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *FileStorageService) storageDirectoryWithSubdir(fileType, subdir string) string {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")

	dir := filepath.Join(s.baseUploadAbs, subdir, fileType+"s", year, month, day)
	return dir
}

func (s *FileStorageService) DeleteFile(localPath string) bool {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return false
	}

	clean := strings.TrimPrefix(localPath, "/")
	full := filepath.Join(s.baseUploadAbs, filepath.FromSlash(clean))

	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		return false
	}
	return os.Remove(full) == nil
}

func (s *FileStorageService) ReadLocalFile(localPath string) ([]byte, error) {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return nil, fmt.Errorf("文件路径为空")
	}

	clean := strings.TrimPrefix(localPath, "/")
	full := filepath.Join(s.baseUploadAbs, filepath.FromSlash(clean))

	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		return nil, fmt.Errorf("文件不存在: %s", localPath)
	}

	return os.ReadFile(full)
}

// FindLocalPathByMD5 兼容 Java 行为：只查询遗留表 media_upload_history，并验证文件仍存在。
func (s *FileStorageService) FindLocalPathByMD5(ctx context.Context, md5Value string) (string, error) {
	md5Value = strings.TrimSpace(md5Value)
	if md5Value == "" {
		return "", nil
	}

	var localPath string
	err := s.db.QueryRowContext(ctx, "SELECT local_path FROM media_upload_history WHERE file_md5 = ? LIMIT 1", md5Value).Scan(&localPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	clean := strings.TrimPrefix(localPath, "/")
	full := filepath.Join(s.baseUploadAbs, filepath.FromSlash(clean))
	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		return "", nil
	}
	return localPath, nil
}

func (s *FileStorageService) generateUniqueFilename(originalFilename string) string {
	ext := s.FileExtension(originalFilename)
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	ts := time.Now().UnixMilli()
	if ext == "" {
		return fmt.Sprintf("%s_%d", id, ts)
	}
	return fmt.Sprintf("%s_%d.%s", id, ts, ext)
}

func (s *FileStorageService) storageDirectory(fileType string) string {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")

	// ./upload/images/yyyy/MM/dd/ 或 ./upload/videos/...
	dir := filepath.Join(s.baseUploadAbs, fileType+"s", year, month, day)
	return dir
}
