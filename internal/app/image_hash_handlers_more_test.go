package app

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleCheckDuplicateMedia_ParseMultipartFormError(t *testing.T) {
	a := &App{}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/checkDuplicateMedia", strings.NewReader("x"))
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleCheckDuplicateMedia_MissingFile(t *testing.T) {
	a := &App{}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("x", "y"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/checkDuplicateMedia", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCheckDuplicateMedia_InvalidSimilarityThreshold(t *testing.T) {
	a := &App{}

	// similarityThreshold 非法应直接返回 400（不依赖 fileStorage/imageHash）
	req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/checkDuplicateMedia?similarityThreshold=abc", "file", "a.png", "image/png", []byte("x"), map[string]string{
		"similarityThreshold": "abc",
	})
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCheckDuplicateMedia_MD5CalculateError(t *testing.T) {
	oldOpen := openMultipartFileHeaderFn
	openMultipartFileHeaderFn = func(*multipart.FileHeader) (multipart.File, error) {
		return nil, errors.New("open fail")
	}
	t.Cleanup(func() { openMultipartFileHeaderFn = oldOpen })

	a := &App{fileStorage: &FileStorageService{}}
	req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", []byte("x"), nil)
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleCheckDuplicateMedia_FindByMD5HashError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	content := []byte("png-bytes")
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs(md5Hex, 20).
		WillReturnError(errors.New("query fail"))

	a := &App{
		fileStorage: &FileStorageService{},
		imageHash:   NewImageHashService(db),
	}

	req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", content, nil)
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleCheckDuplicateMedia_FindSimilarByPHashError(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 3), G: uint8(y * 3), B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	content := buf.Bytes()
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs(md5Hex, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at",
		}))

	mock.ExpectQuery(`(?s)BIT_COUNT.*FROM image_hash.*<= \?.*LIMIT \?`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10, 20).
		WillReturnError(errors.New("query fail"))

	a := &App{
		fileStorage: &FileStorageService{},
		imageHash:   NewImageHashService(db),
	}

	req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", content, nil)
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleCheckDuplicateMedia_PHashNoMatches(t *testing.T) {
	db, mock, cleanup := newSQLMock(t)
	defer cleanup()

	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	img.SetRGBA(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	content := buf.Bytes()
	sum := md5.Sum(content)
	md5Hex := hex.EncodeToString(sum[:])

	mock.ExpectQuery(`(?s)FROM image_hash.*WHERE md5_hash = \?.*LIMIT \?`).
		WithArgs(md5Hex, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at",
		}))

	mock.ExpectQuery(`(?s)BIT_COUNT.*FROM image_hash.*<= \?.*LIMIT \?`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_path", "file_name", "file_dir", "md5_hash", "phash", "file_size", "created_at", "distance",
		}))

	a := &App{
		fileStorage: &FileStorageService{},
		imageHash:   NewImageHashService(db),
	}

	req, _ := newMultipartRequest(t, http.MethodPost, "http://example.com/api/checkDuplicateMedia", "file", "a.png", "image/png", content, nil)
	rr := httptest.NewRecorder()
	a.handleCheckDuplicateMedia(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data: %v", resp)
	}
	if got, _ := data["matchType"].(string); got != "none" {
		t.Fatalf("matchType=%q, want %q", got, "none")
	}
	if got, _ := data["pHash"].(string); got == "" {
		t.Fatalf("expected pHash")
	}
}

func TestParseSimilarityThreshold_ClampAndError(t *testing.T) {
	if _, err := parseSimilarityThreshold("bad", 0.5); err == nil {
		t.Fatalf("expected error")
	}
	if v, err := parseSimilarityThreshold("", 2); err != nil || v != 1 {
		t.Fatalf("v=%v err=%v, want 1", v, err)
	}
	if v, err := parseSimilarityThreshold("101", 0.5); err != nil || v != 1 {
		t.Fatalf("v=%v err=%v, want 1", v, err)
	}
	if v, err := parseSimilarityThreshold("-1", 0.5); err != nil || v != 0 {
		t.Fatalf("v=%v err=%v, want 0", v, err)
	}
}

func TestResolvePHashThreshold_InvalidSimilarityAndThresholdFallback(t *testing.T) {
	if _, _, _, err := resolvePHashThreshold("bad", "", ""); err == nil {
		t.Fatalf("expected error")
	}

	thresholdType, _, distance, err := resolvePHashThreshold("", "", "999")
	if err != nil {
		t.Fatalf("resolvePHashThreshold: %v", err)
	}
	if thresholdType != "distance" || distance != phashBitLength {
		t.Fatalf("thresholdType=%q distance=%d, want distance=%d", thresholdType, distance, phashBitLength)
	}
}

func TestParseIntOrDefault_AndClampFloat(t *testing.T) {
	if got := parseIntOrDefault("", 7); got != 7 {
		t.Fatalf("got=%d, want 7", got)
	}
	if got := parseIntOrDefault("bad", 7); got != 7 {
		t.Fatalf("got=%d, want 7", got)
	}
	if got := parseIntOrDefault("8", 7); got != 8 {
		t.Fatalf("got=%d, want 8", got)
	}

	if got := clampFloat(-1, 0, 1); got != 0 {
		t.Fatalf("got=%v, want 0", got)
	}
	if got := clampFloat(2, 0, 1); got != 1 {
		t.Fatalf("got=%v, want 1", got)
	}
	if got := clampFloat(0.5, 0, 1); got != 0.5 {
		t.Fatalf("got=%v, want 0.5", got)
	}
}

func TestSimilarityThresholdToDistance_Clamp(t *testing.T) {
	if got := similarityThresholdToDistance(-1); got != phashBitLength {
		t.Fatalf("got=%d, want %d", got, phashBitLength)
	}
	if got := similarityThresholdToDistance(2); got != 0 {
		t.Fatalf("got=%d, want 0", got)
	}
}
