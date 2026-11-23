package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	integrationThumbMaxEdge      = 12000
	integrationThumbMaxPixels    = 12 * 1000 * 1000
	integrationThumbMaxFileBytes = 20 << 20
	integrationThumbConcurrency  = 2
	integrationThumbMaxEntries   = 5000
)

var errIntegrationThumbUnsupported = errors.New("integration thumbnail unsupported")

// integrationThumbCache 将本地媒体缩略图缓存在独立目录，避免每次列表都解码原图。
type integrationThumbCache struct {
	dir         string
	sem         chan struct{}
	mu          sync.Mutex
	inflight    map[string]chan struct{}
	entries     int
	initialized bool
}

func newIntegrationThumbCache(dir string) *integrationThumbCache {
	return &integrationThumbCache{
		dir:      strings.TrimSpace(dir),
		sem:      make(chan struct{}, integrationThumbConcurrency),
		inflight: map[string]chan struct{}{},
	}
}

func (c *integrationThumbCache) thumbnail(ctx context.Context, rootDir, cacheIdentity, sourceAbs string, target int) ([]byte, string, time.Time, error) {
	if c == nil || c.dir == "" {
		return nil, "", time.Time{}, fmt.Errorf("缩略图缓存未初始化")
	}
	if target <= 0 {
		target = 320
	}
	info, err := os.Stat(sourceAbs)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if info.IsDir() || info.Size() <= 0 {
		return nil, "", time.Time{}, os.ErrNotExist
	}
	source, sourceInfo, err := openRegularUnder(rootDir, sourceAbs)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	defer source.Close()

	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d|%d", cacheIdentity, info.ModTime().UnixNano(), info.Size(), target)))
	hash := hex.EncodeToString(sum[:])
	cachePath := filepath.Join(c.dir, hash[:2], hash+".jpg")

	if data, ok := readThumbFile(cachePath); ok {
		return data, `"` + hash + `"`, info.ModTime(), nil
	}

	c.mu.Lock()
	if wait, ok := c.inflight[hash]; ok {
		c.mu.Unlock()
		select {
		case <-wait:
		case <-ctx.Done():
			return nil, "", time.Time{}, ctx.Err()
		}
		if data, ok := readThumbFile(cachePath); ok {
			return data, `"` + hash + `"`, info.ModTime(), nil
		}
		return nil, "", time.Time{}, errIntegrationThumbUnsupported
	}
	done := make(chan struct{})
	c.inflight[hash] = done
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.inflight, hash)
		close(done)
		c.mu.Unlock()
	}()

	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, "", time.Time{}, ctx.Err()
	}
	// 先拿到信号量后重查，避免并发请求重复解码同一张图。
	if data, ok := readThumbFile(cachePath); ok {
		return data, `"` + hash + `"`, info.ModTime(), nil
	}

	data, err := buildIntegrationThumbFromFile(source, sourceInfo.Size(), target)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if err := c.writeThumb(cachePath, data); err != nil {
		// 缓存写失败不影响本次响应，仅放弃复用。
		return data, `"` + hash + `"`, info.ModTime(), nil
	}
	return data, `"` + hash + `"`, info.ModTime(), nil
}

func (c *integrationThumbCache) writeThumb(path string, data []byte) error {
	if err := writeThumbFile(path, data); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.initialized {
		c.entries = countThumbFiles(c.dir)
		c.initialized = true
	}
	c.entries++
	if c.entries > integrationThumbMaxEntries {
		c.pruneLocked()
	}
	return nil
}

func countThumbFiles(dir string) int {
	count := 0
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jpg") {
			count++
		}
		return nil
	})
	return count
}

type thumbFileEntry struct {
	path    string
	modTime time.Time
}

func (c *integrationThumbCache) pruneLocked() {
	var files []thumbFileEntry
	_ = filepath.WalkDir(c.dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jpg") {
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil {
			return nil
		}
		files = append(files, thumbFileEntry{path: path, modTime: info.ModTime()})
		return nil
	})
	if len(files) <= integrationThumbMaxEntries {
		c.entries = len(files)
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.Before(files[j].modTime) })
	target := integrationThumbMaxEntries * 9 / 10
	remove := len(files) - target
	for i := 0; i < remove; i++ {
		_ = os.Remove(files[i].path)
	}
	c.entries = len(files) - remove
}

func readThumbFile(path string) ([]byte, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

func writeThumbFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".thumb-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func buildIntegrationThumb(sourceAbs string, target int) ([]byte, error) {
	file, err := os.Open(sourceAbs)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return buildIntegrationThumbFromFile(file, stat.Size(), target)
}

func buildIntegrationThumbFromFile(file *os.File, size int64, target int) ([]byte, error) {
	if size > integrationThumbMaxFileBytes {
		return nil, errIntegrationThumbUnsupported
	}

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, errIntegrationThumbUnsupported
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, errIntegrationThumbUnsupported
	}
	maxEdge := config.Width
	if config.Height > maxEdge {
		maxEdge = config.Height
	}
	if maxEdge > integrationThumbMaxEdge || int64(config.Width)*int64(config.Height) > integrationThumbMaxPixels {
		return nil, errIntegrationThumbUnsupported
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	src, _, err := image.Decode(file)
	if err != nil {
		return nil, errIntegrationThumbUnsupported
	}

	width, height := scaledSize(src.Bounds().Dx(), src.Bounds().Dy(), target)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func scaledSize(width, height, target int) (int, int) {
	if width <= 0 || height <= 0 {
		return 1, 1
	}
	if width <= target && height <= target {
		return width, height
	}
	if width >= height {
		return target, maxInt(1, height*target/width)
	}
	return maxInt(1, width*target/height), target
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
