package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var motionPhotoEOIGapBytes = []byte{
	0x05, 0xFA, 0x2F, 0x36, 0x00, 0x00, 0x00, 0x00,
	0x44, 0xDC, 0x1C, 0x24, 0x3F, 0x8A, 0x0F, 0x86,
	0x18, 0xB5, 0x81, 0xE1, 0x8F, 0x31, 0xBA, 0x62,
}

var motionPhotoJFIFAPP0Segment = []byte{
	0xFF, 0xE0, 0x00, 0x10,
	'J', 'F', 'I', 'F', 0x00,
	0x01, 0x01, 0x00,
	0x00, 0x01, 0x00, 0x01,
	0x00, 0x00,
}

func (a *App) handleDouyinLivePhoto(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key 不能为空"})
		return
	}

	cached, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || cached == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "解析已过期，请重新解析"})
		return
	}

	imageIndex, err := parseOptionalInt(r.URL.Query().Get("imageIndex"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "imageIndex 非法"})
		return
	}
	videoIndex, err := parseOptionalInt(r.URL.Query().Get("videoIndex"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "videoIndex 非法"})
		return
	}

	imgIdx, vidIdx, errMsg := selectDouyinLivePhotoPair(cached.Downloads, imageIndex, videoIndex)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": errMsg})
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "zip"
	}
	if format == "jpeg" {
		format = "jpg"
	}
	if format != "zip" && format != "jpg" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "format 仅支持 zip/jpg"})
		return
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务器缺少 ffmpeg，无法生成实况文件"})
		return
	}
	if format == "zip" {
		if _, err := exec.LookPath("exiftool"); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务器缺少 exiftool，无法生成 iOS 实况文件"})
			return
		}
	}

	remoteImageURL := strings.TrimSpace(cached.Downloads[imgIdx])
	remoteVideoURL := strings.TrimSpace(cached.Downloads[vidIdx])
	if remoteImageURL == "" || remoteVideoURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "实况资源链接为空"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "douyin-livephoto-")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "创建临时目录失败"})
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	rawImagePath := filepath.Join(tmpDir, "image.raw")
	rawVideoPath := filepath.Join(tmpDir, "video.raw")

	imgContentType, err := downloadDouyinResourceToFile(ctx, a.douyinDownloader.api.httpClient, remoteImageURL, rawImagePath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载图片失败: " + err.Error()})
		return
	}

	vidContentType, err := downloadDouyinResourceToFile(ctx, a.douyinDownloader.api.httpClient, remoteVideoURL, rawVideoPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载视频失败: " + err.Error()})
		return
	}

	stillPath := filepath.Join(tmpDir, "still.jpg")
	if err := normalizeLivePhotoStillImage(ctx, rawImagePath, imgContentType, stillPath); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "图片转码失败: " + err.Error()})
		return
	}

	// Live Photo 导出：文件名尽量与普通图片下载一致（标题 + 序号），额外追加 `_live`。
	baseJPGName := buildDouyinOriginalFilename(cached.Title, cached.DetailID, imgIdx, len(cached.Downloads), ".jpg")
	base := strings.TrimSuffix(baseJPGName, ".jpg")
	if strings.TrimSpace(base) == "" {
		base = "livephoto"
	}

	if format == "zip" {
		motionPath := filepath.Join(tmpDir, "motion.mov")
		if err := normalizeLivePhotoMotionVideo(ctx, rawVideoPath, vidContentType, motionPath); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "视频转码失败: " + err.Error()})
			return
		}

		assetID := strings.ToUpper(uuid.NewString())
		if err := tagLivePhotoAsset(ctx, stillPath, motionPath, assetID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "写入实况元数据失败: " + err.Error()})
			return
		}

		zipFilename := base + "_live.zip"
		zipFallback := buildDouyinFallbackFilename(cached.DetailID, imgIdx, len(cached.Downloads), ".zip")
		zipFallback = strings.TrimSuffix(zipFallback, ".zip") + "_live.zip"

		// 处理完成后再开始写入响应，避免中途失败导致已输出不可恢复
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", buildAttachmentContentDisposition(zipFallback, zipFilename))
		w.WriteHeader(http.StatusOK)

		zw := zip.NewWriter(w)
		defer func() { _ = zw.Close() }()

		if err := zipFile(zw, stillPath, base+".jpg"); err != nil {
			return
		}
		if err := zipFile(zw, motionPath, base+".mov"); err != nil {
			return
		}
		return
	}

	motionMP4Path := filepath.Join(tmpDir, "motion.mp4")
	if err := normalizeMotionPhotoMotionVideo(ctx, rawVideoPath, vidContentType, motionMP4Path); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "视频转码失败: " + err.Error()})
		return
	}

	outPath := filepath.Join(tmpDir, "live.jpg")
	if err := buildMotionPhotoJPG(stillPath, motionMP4Path, outPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "生成实况 JPG 失败: " + err.Error()})
		return
	}

	f, err := os.Open(outPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "读取生成文件失败"})
		return
	}
	defer f.Close()

	if st, err := f.Stat(); err == nil && st != nil && st.Size() > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(st.Size(), 10))
	}

	filename := base + "_live.jpg"
	fallback := buildDouyinFallbackFilename(cached.DetailID, imgIdx, len(cached.Downloads), ".jpg")
	fallback = strings.TrimSuffix(fallback, ".jpg") + "_live.jpg"
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", buildAttachmentContentDisposition(fallback, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

func parseOptionalInt(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func selectDouyinLivePhotoPair(downloads []string, imageIndex, videoIndex *int) (imgIdx int, vidIdx int, errMsg string) {
	if len(downloads) == 0 {
		return 0, 0, "downloads 为空"
	}
	if imageIndex != nil && (*imageIndex < 0 || *imageIndex >= len(downloads)) {
		return 0, 0, "imageIndex 越界"
	}
	if videoIndex != nil && (*videoIndex < 0 || *videoIndex >= len(downloads)) {
		return 0, 0, "videoIndex 越界"
	}

	imagePositions, videoPositions := collectDouyinMediaPositions(downloads)
	if len(imagePositions) == 0 || len(videoPositions) == 0 {
		return 0, 0, "未找到图片/视频资源，无法生成实况"
	}

	if imageIndex != nil {
		if guessDouyinMediaTypeFromURL(downloads[*imageIndex]) != "image" {
			return 0, 0, "imageIndex 不是图片资源"
		}
	}
	if videoIndex != nil {
		if guessDouyinMediaTypeFromURL(downloads[*videoIndex]) != "video" {
			return 0, 0, "videoIndex 不是视频资源"
		}
	}

	if imageIndex != nil && videoIndex != nil {
		return *imageIndex, *videoIndex, ""
	}

	if imageIndex != nil {
		imgIdx = *imageIndex
		rank := findDouyinMediaRank(imagePositions, imgIdx)
		if rank < 0 {
			return 0, 0, "imageIndex 不是图片资源"
		}

		if len(videoPositions) == 1 {
			// 兼容单视频场景：默认认为所有图片共享同一段动态视频。
			return imgIdx, videoPositions[0], ""
		}
		if rank < len(videoPositions) {
			return imgIdx, videoPositions[rank], ""
		}
		return 0, 0, "未找到与该图片对应的视频资源"
	}

	if videoIndex != nil {
		vidIdx = *videoIndex
		rank := findDouyinMediaRank(videoPositions, vidIdx)
		if rank < 0 {
			return 0, 0, "videoIndex 不是视频资源"
		}

		if len(imagePositions) == 1 {
			// 兼容单图片场景：默认认为所有视频共享同一张静态图。
			return imagePositions[0], vidIdx, ""
		}
		if rank < len(imagePositions) {
			return imagePositions[rank], vidIdx, ""
		}
		return 0, 0, "未找到与该视频对应的图片资源"
	}

	// 无显式索引时，默认返回第一组可用配对。
	return imagePositions[0], videoPositions[0], ""
}

func collectDouyinMediaPositions(downloads []string) (imagePositions []int, videoPositions []int) {
	imagePositions = make([]int, 0, len(downloads))
	videoPositions = make([]int, 0, len(downloads))
	for i, raw := range downloads {
		if guessDouyinMediaTypeFromURL(raw) == "video" {
			videoPositions = append(videoPositions, i)
			continue
		}
		imagePositions = append(imagePositions, i)
	}
	return imagePositions, videoPositions
}

func findDouyinMediaRank(positions []int, target int) int {
	for i, p := range positions {
		if p == target {
			return i
		}
	}
	return -1
}

func downloadDouyinResourceToFile(ctx context.Context, client *http.Client, remoteURL, dstPath string) (contentType string, err error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", fmt.Errorf("remoteURL 为空")
	}

	u, err := url.Parse(remoteURL)
	if err != nil || u == nil || strings.TrimSpace(u.Scheme) == "" {
		return "", fmt.Errorf("remoteURL 非法")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return "", fmt.Errorf("remoteURL 非法")
	}
	req.Header.Set("User-Agent", douyinDefaultUserAgent)
	req.Header.Set("Referer", douyinDefaultReferer)
	req.Header.Set("Origin", douyinDefaultOrigin)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("%s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	contentType = strings.TrimSpace(resp.Header.Get("Content-Type"))

	f, err := os.Create(dstPath)
	if err != nil {
		return contentType, err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return contentType, err
	}
	return contentType, nil
}

func normalizeLivePhotoStillImage(ctx context.Context, inputPath, contentType, outputJPGPath string) error {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "image/jpeg") || strings.HasSuffix(strings.ToLower(inputPath), ".jpg") || strings.HasSuffix(strings.ToLower(inputPath), ".jpeg") {
		return copyFile(inputPath, outputJPGPath)
	}
	// 统一转为 JPG：iOS Live Photo 静态图建议使用 JPG/HEIC，这里固定输出 JPG。
	return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-frames:v", "1", outputJPGPath})
}

func normalizeLivePhotoMotionVideo(ctx context.Context, inputPath, contentType, outputMOVPath string) error {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "video/quicktime") || strings.HasSuffix(strings.ToLower(inputPath), ".mov") {
		// 仍走一次 remux，保证 metadata atom 结构一致（best-effort）
		return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-c", "copy", "-f", "mov", outputMOVPath})
	}
	return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-c", "copy", "-f", "mov", outputMOVPath})
}

func tagLivePhotoAsset(ctx context.Context, stillPath, motionPath, assetID string) error {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return fmt.Errorf("assetID 为空")
	}

	// 图片：写入 Apple ContentIdentifier（MakerApple），并补齐 Make/Model 增加 iOS 识别概率。
	if err := runCommand(ctx, "exiftool", []string{
		"-overwrite_original",
		"-Make=Apple",
		"-Model=iPhone",
		"-ContentIdentifier=" + assetID,
		stillPath,
	}); err != nil {
		return err
	}

	// 视频：写入 QuickTime ContentIdentifier（用于与图片配对）。
	if err := runCommand(ctx, "exiftool", []string{
		"-overwrite_original",
		"-ContentIdentifier=" + assetID,
		motionPath,
	}); err != nil {
		return err
	}
	return nil
}

func normalizeMotionPhotoMotionVideo(ctx context.Context, inputPath, contentType, outputMP4Path string) error {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(contentType, "video/mp4") || strings.HasSuffix(strings.ToLower(inputPath), ".mp4") {
		return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-c", "copy", "-movflags", "+faststart", "-f", "mp4", outputMP4Path})
	}
	return runCommand(ctx, "ffmpeg", []string{"-y", "-i", inputPath, "-c", "copy", "-movflags", "+faststart", "-f", "mp4", outputMP4Path})
}

func runCommand(ctx context.Context, name string, args []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(out.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func zipFile(zw *zip.Writer, srcPath, zipName string) error {
	srcPath = strings.TrimSpace(srcPath)
	zipName = strings.TrimSpace(zipName)
	if srcPath == "" || zipName == "" {
		return fmt.Errorf("zip 参数为空")
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = zipName
	hdr.Method = zip.Deflate

	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

func buildMotionPhotoJPG(stillJPGPath, motionMP4Path, outputPath string) error {
	still, err := os.ReadFile(stillJPGPath)
	if err != nil {
		return err
	}
	mp4Info, err := os.Stat(motionMP4Path)
	if err != nil {
		return err
	}
	if mp4Info.Size() <= 0 {
		return fmt.Errorf("motion mp4 为空")
	}

	stillBodyOffset, err := findJPEGFirstDQTOffset(still)
	if err != nil {
		return err
	}
	stillBody := still[stillBodyOffset:]
	if len(stillBody) < 2 || !bytes.HasSuffix(stillBody, []byte{0xFF, 0xD9}) {
		return fmt.Errorf("still jpg 非法（缺少 EOI）")
	}

	width, height, err := parseJPEGDimensions(still)
	if err != nil {
		// best-effort：仍然输出可用文件
		width = 0
		height = 0
	}

	exifSeg, err := buildMotionPhotoExifAPP1Segment(width, height)
	if err != nil {
		return err
	}
	xmpSeg, err := buildMotionPhotoXMPAPP1Segment(mp4Info.Size())
	if err != nil {
		return err
	}

	// 为提升 MIUI 相册对“动态照片”的识别概率：
	// - JPEG 头部段顺序：APP1(Exif) → APP1(XMP) → APP0(JFIF)
	// - EOI 与 MP4 之间插入固定 24 字节 gap（对齐常见导出形态）
	jpegWithXMP := make([]byte, 0, 2+len(exifSeg)+len(xmpSeg)+len(motionPhotoJFIFAPP0Segment)+len(stillBody))
	jpegWithXMP = append(jpegWithXMP, 0xFF, 0xD8) // SOI
	jpegWithXMP = append(jpegWithXMP, exifSeg...)
	jpegWithXMP = append(jpegWithXMP, xmpSeg...)
	jpegWithXMP = append(jpegWithXMP, motionPhotoJFIFAPP0Segment...)
	jpegWithXMP = append(jpegWithXMP, stillBody...)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := out.Write(jpegWithXMP); err != nil {
		return err
	}

	if _, err := out.Write(motionPhotoEOIGapBytes); err != nil {
		return err
	}

	mp4, err := os.Open(motionMP4Path)
	if err != nil {
		return err
	}
	defer mp4.Close()
	if _, err := io.Copy(out, mp4); err != nil {
		return err
	}
	return out.Close()
}

func buildMotionPhotoXMPAPP1Segment(microVideoOffset int64) ([]byte, error) {
	if microVideoOffset <= 0 {
		return nil, fmt.Errorf("microVideoOffset 非法")
	}

	xmpXML := fmt.Sprintf(
		"<x:xmpmeta xmlns:x=\"adobe:ns:meta/\" x:xmptk=\"Adobe XMP Core 5.1.0-jc003\">\n"+
			"  <rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n"+
			"    <rdf:Description rdf:about=\"\"\n"+
			"        xmlns:GCamera=\"http://ns.google.com/photos/1.0/camera/\"\n"+
			"      GCamera:MicroVideoVersion=\"1\"\n"+
			"      GCamera:MicroVideo=\"1\"\n"+
			"      GCamera:MicroVideoOffset=\"%d\"\n"+
			"      GCamera:MicroVideoPresentationTimestampUs=\"0\"/>\n"+
			"  </rdf:RDF>\n"+
			"</x:xmpmeta>\n",
		microVideoOffset,
	)

	const xmpHeader = "http://ns.adobe.com/xap/1.0/\x00"
	payload := append([]byte(xmpHeader), []byte(xmpXML)...)
	if len(payload) > 0xFFFF-2 {
		return nil, fmt.Errorf("XMP 过大")
	}

	// APP1: marker(2) + length(2) + payload；length 字段包含 length 自身(2)，不包含 marker。
	length := len(payload) + 2
	seg := make([]byte, 0, 2+2+len(payload))
	seg = append(seg, 0xFF, 0xE1, byte(length>>8), byte(length))
	seg = append(seg, payload...)
	return seg, nil
}

func injectJPEGSegmentAfterAPP0(jpeg []byte, segment []byte) ([]byte, error) {
	if len(jpeg) < 2 || jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		return nil, fmt.Errorf("不是 JPEG")
	}

	insertAt := 2
	if len(jpeg) >= 6 && jpeg[2] == 0xFF && jpeg[3] == 0xE0 {
		n := int(binary.BigEndian.Uint16(jpeg[4:6]))
		if n < 2 || 2+2+n > len(jpeg) {
			return nil, fmt.Errorf("JPEG APP0 段非法")
		}
		insertAt = 2 + 2 + n
	}

	out := make([]byte, 0, len(jpeg)+len(segment))
	out = append(out, jpeg[:insertAt]...)
	out = append(out, segment...)
	out = append(out, jpeg[insertAt:]...)
	return out, nil
}

func buildMotionPhotoExifAPP1Segment(width, height int) ([]byte, error) {
	var w uint32
	var h uint32
	if width > 0 {
		w = uint32(width)
	}
	if height > 0 {
		h = uint32(height)
	}

	// 形态参考：常见 MIUI/微信导出。
	// - Exif APP1 length=0x006a（含 length 字段）
	// - TIFF big-endian
	// - IFD0 4 entries: ImageWidth/ImageLength/ExifIFDPointer/Orientation
	// - ExifIFD 2 entries: 0x9A01(BYTE=1), 0x9208(LONG=0)
	payload := make([]byte, 0, 104)
	payload = append(payload, []byte("Exif\x00\x00")...)

	// TIFF header
	payload = append(payload, 'M', 'M')               // big-endian
	payload = append(payload, 0x00, 0x2A)             // magic 42
	payload = append(payload, 0x00, 0x00, 0x00, 0x08) // IFD0 offset

	// IFD0: 4 entries
	payload = append(payload, 0x00, 0x04)
	// 0x0100 ImageWidth LONG 1
	payload = append(payload, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01)
	payload = append(payload, byte(w>>24), byte(w>>16), byte(w>>8), byte(w))
	// 0x0101 ImageLength LONG 1
	payload = append(payload, 0x01, 0x01, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01)
	payload = append(payload, byte(h>>24), byte(h>>16), byte(h>>8), byte(h))
	// 0x8769 ExifIFDPointer LONG 1 -> 0x3E (62)
	payload = append(payload, 0x87, 0x69, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x3E)
	// 0x0112 Orientation LONG 1 -> 0
	payload = append(payload, 0x01, 0x12, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00)
	// next IFD offset = 0
	payload = append(payload, 0x00, 0x00, 0x00, 0x00)

	// Exif IFD at offset 0x3E: 2 entries
	payload = append(payload, 0x00, 0x02)
	// 0x9A01 BYTE 1 -> 1
	payload = append(payload, 0x9A, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00)
	// 0x9208 LONG 1 -> 0
	payload = append(payload, 0x92, 0x08, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00)
	// next IFD offset = 0
	payload = append(payload, 0x00, 0x00, 0x00, 0x00)

	// padding to match common shape
	for len(payload) < 104 {
		payload = append(payload, 0x00)
	}
	if len(payload) != 104 {
		return nil, fmt.Errorf("exif payload size=%d, want 104", len(payload))
	}

	// APP1: marker(2) + length(2) + payload；length 字段包含 length 自身(2)，不包含 marker。
	length := len(payload) + 2
	seg := make([]byte, 0, 2+2+len(payload))
	seg = append(seg, 0xFF, 0xE1, byte(length>>8), byte(length))
	seg = append(seg, payload...)
	return seg, nil
}

func findJPEGFirstDQTOffset(jpeg []byte) (int, error) {
	if len(jpeg) < 2 || jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		return 0, fmt.Errorf("不是 JPEG")
	}

	i := 2
	for i+1 < len(jpeg) {
		if jpeg[i] != 0xFF {
			return 0, fmt.Errorf("JPEG 段解析失败")
		}
		markerStart := i
		for i < len(jpeg) && jpeg[i] == 0xFF {
			i++
		}
		if i >= len(jpeg) {
			return 0, fmt.Errorf("JPEG 段解析失败")
		}
		marker := jpeg[i]
		i++

		switch marker {
		case 0xDB: // DQT
			return markerStart, nil
		case 0xDA: // SOS
			return 0, fmt.Errorf("JPEG 未包含 DQT 段")
		case 0xD9: // EOI
			return 0, fmt.Errorf("JPEG 未包含 DQT 段")
		case 0x01: // TEM
			continue
		}
		if marker >= 0xD0 && marker <= 0xD7 { // RSTn
			continue
		}

		if i+2 > len(jpeg) {
			return 0, fmt.Errorf("JPEG 段解析失败")
		}
		n := int(binary.BigEndian.Uint16(jpeg[i : i+2]))
		if n < 2 || i+n > len(jpeg) {
			return 0, fmt.Errorf("JPEG 段解析失败")
		}
		i += n
	}

	return 0, fmt.Errorf("JPEG 未包含 DQT 段")
}

func parseJPEGDimensions(jpeg []byte) (width int, height int, err error) {
	if len(jpeg) < 2 || jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		return 0, 0, fmt.Errorf("不是 JPEG")
	}

	isSOF := func(marker byte) bool {
		switch marker {
		case 0xC0, 0xC1, 0xC2, 0xC3, 0xC5, 0xC6, 0xC7, 0xC9, 0xCA, 0xCB, 0xCD, 0xCE, 0xCF:
			return true
		default:
			return false
		}
	}

	i := 2
	for i+1 < len(jpeg) {
		if jpeg[i] != 0xFF {
			return 0, 0, fmt.Errorf("JPEG 段解析失败")
		}
		for i < len(jpeg) && jpeg[i] == 0xFF {
			i++
		}
		if i >= len(jpeg) {
			return 0, 0, fmt.Errorf("JPEG 段解析失败")
		}
		marker := jpeg[i]
		i++

		if marker == 0xDA { // SOS
			return 0, 0, fmt.Errorf("JPEG 缺少 SOF 段")
		}
		if marker == 0xD9 { // EOI
			return 0, 0, fmt.Errorf("JPEG 缺少 SOF 段")
		}
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}

		if i+2 > len(jpeg) {
			return 0, 0, fmt.Errorf("JPEG 段解析失败")
		}
		n := int(binary.BigEndian.Uint16(jpeg[i : i+2]))
		if n < 2 || i+n > len(jpeg) {
			return 0, 0, fmt.Errorf("JPEG 段解析失败")
		}
		if isSOF(marker) {
			if n < 7 {
				return 0, 0, fmt.Errorf("JPEG SOF 段非法")
			}
			// len(2) + precision(1) + height(2) + width(2)
			height = int(binary.BigEndian.Uint16(jpeg[i+3 : i+5]))
			width = int(binary.BigEndian.Uint16(jpeg[i+5 : i+7]))
			if width <= 0 || height <= 0 {
				return 0, 0, fmt.Errorf("JPEG 尺寸非法")
			}
			return width, height, nil
		}
		i += n
	}

	return 0, 0, fmt.Errorf("JPEG 缺少 SOF 段")
}
