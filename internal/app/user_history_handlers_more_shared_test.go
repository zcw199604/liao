package app

import (
	"bytes"
	"context"
	"errors"
)

type errReadCloserUserHistory struct{}

func (errReadCloserUserHistory) Read(p []byte) (int, error) { return 0, errors.New("read err") }
func (errReadCloserUserHistory) Close() error               { return nil }

// errMultipartFile implements multipart.File with read errors, used to force io.Copy failures.
type errMultipartFile struct {
	r *bytes.Reader
}

func (f *errMultipartFile) Read(p []byte) (int, error) { return 0, errors.New("read err") }
func (f *errMultipartFile) ReadAt(p []byte, off int64) (int, error) {
	return f.r.ReadAt(p, off)
}
func (f *errMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return f.r.Seek(offset, whence)
}
func (f *errMultipartFile) Close() error { return nil }

type errChatHistoryCache struct{}

func (e *errChatHistoryCache) SaveMessages(context.Context, string, []map[string]any) {}
func (e *errChatHistoryCache) GetMessages(context.Context, string, string, int) ([]map[string]any, error) {
	return nil, errors.New("redis err")
}
