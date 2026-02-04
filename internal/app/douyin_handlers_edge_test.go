package app

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDouyinHelpers_asBool_asInt(t *testing.T) {
	if !asBool(true) {
		t.Fatalf("expected true")
	}
	if asBool(float64(0)) {
		t.Fatalf("expected false")
	}
	if !asBool(float64(1)) {
		t.Fatalf("expected true")
	}
	if asBool(0) {
		t.Fatalf("expected false")
	}
	if !asBool(int64(2)) {
		t.Fatalf("expected true")
	}
	if !asBool(" YES ") {
		t.Fatalf("expected true")
	}
	if asBool("no") {
		t.Fatalf("expected false")
	}

	if asInt(nil) != 0 {
		t.Fatalf("expected 0")
	}
	if asInt(float64(3)) != 3 {
		t.Fatalf("expected 3")
	}
	if asInt(int64(4)) != 4 {
		t.Fatalf("expected 4")
	}
	if asInt(" 5 ") != 5 {
		t.Fatalf("expected 5")
	}
	if asInt("x") != 0 {
		t.Fatalf("expected 0")
	}
}

func TestHandleDouyinDownloadHead_FallbackRangeContentRangeTotal(t *testing.T) {
	mediaBytes := []byte("x")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodGet:
			if strings.TrimSpace(r.Header.Get("Range")) != "bytes=0-0" {
				http.Error(w, "missing range", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", "bytes 0-0/1234")
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(mediaBytes)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(upstream.Close)

	a := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second)}

	cached := &douyinCachedDetail{
		DetailID:  "d1",
		Title:     "t",
		Type:      "视频",
		Downloads: []string{upstream.URL + "/media.mp4"},
	}

	req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key=x&index=0", nil)
	rec := httptest.NewRecorder()
	a.handleDouyinDownloadHead(rec, req, "x", cached, 0, cached.Downloads[0])

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Length"); got != "1234" {
		t.Fatalf("content-length=%q", got)
	}
	if got := strings.TrimSpace(rec.Header().Get("Accept-Ranges")); got != "bytes" {
		t.Fatalf("accept-ranges=%q", got)
	}
	if got := strings.TrimSpace(rec.Header().Get("Content-Type")); got != "video/mp4" {
		t.Fatalf("content-type=%q", got)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "filename*=") {
		t.Fatalf("content-disposition=%q", cd)
	}
}

func TestHandleDouyinDownloadHead_InvalidURL(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", 60*time.Second)}
	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		t.Fatalf("should not call transport")
		return nil, nil
	})}

	req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key=x&index=0", nil)
	rec := httptest.NewRecorder()
	a.handleDouyinDownloadHead(rec, req, "x", &douyinCachedDetail{Title: "t", DetailID: "d1", Downloads: []string{"x"}}, 0, "http://[::1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
	got := decodeJSONBody(t, rec.Body)
	if got["error"].(string) != "下载链接非法" {
		t.Fatalf("got=%v", got)
	}
}

func TestHandleDouyinDownloadHead_RangeErrors(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", 60*time.Second)}
	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodHead {
			return nil, errors.New("head fail")
		}
		if r.Method == http.MethodGet {
			return nil, errors.New("get fail")
		}
		return nil, errors.New("unexpected")
	})}

	req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key=x&index=0", nil)
	rec := httptest.NewRecorder()
	a.handleDouyinDownloadHead(rec, req, "x", &douyinCachedDetail{Title: "t", DetailID: "d1", Downloads: []string{"http://example.com/media.mp4"}}, 0, "http://example.com/media.mp4")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
	got := decodeJSONBody(t, rec.Body)
	if !strings.Contains(got["error"].(string), "下载失败") {
		t.Fatalf("got=%v", got)
	}
}

func TestHandleDouyinDownloadHead_RangeStatusError(t *testing.T) {
	a := &App{douyinDownloader: NewDouyinDownloaderService("http://example.com", "", "", "", 60*time.Second)}
	a.douyinDownloader.api.httpClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodHead {
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: ioNopCloser("boom")}, nil
		}
		if r.Method == http.MethodGet {
			return &http.Response{StatusCode: http.StatusInternalServerError, Status: "500", Header: make(http.Header), Body: ioNopCloser("abcd")}, nil
		}
		return nil, errors.New("unexpected")
	})}

	req := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key=x&index=0", nil)
	rec := httptest.NewRecorder()
	a.handleDouyinDownloadHead(rec, req, "x", &douyinCachedDetail{Title: "t", DetailID: "d1", Downloads: []string{"http://example.com/media.mp4"}}, 0, "http://example.com/media.mp4")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
	got := decodeJSONBody(t, rec.Body)
	if !strings.Contains(got["error"].(string), "下载失败") {
		t.Fatalf("got=%v", got)
	}
}

func TestDouyinFilenameHelpers(t *testing.T) {
	if got := guessExtFromURL(" "); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := guessExtFromURL("http://example.com/a"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := guessExtFromURL("http://example.com/a.mp4?x=1"); got != ".mp4" {
		t.Fatalf("got=%q", got)
	}
	if got := guessExtFromURL("not a url.mp4"); got != ".mp4" {
		t.Fatalf("got=%q", got)
	}
	if got := guessExtFromURL("http://[::1/a.mp4"); got != ".mp4" {
		t.Fatalf("got=%q", got)
	}
	if got := guessExtFromURL("http://[::1/a"); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := guessExtFromURL("http://example.com/a." + strings.Repeat("x", 20)); got != "" {
		t.Fatalf("got=%q", got)
	}

	if got := buildDouyinOriginalFilename("", " id ", 0, 1, ""); got != "id.bin" {
		t.Fatalf("got=%q", got)
	}
	if got := buildDouyinOriginalFilename("t", "id", 1, 2, ".mp4"); got != "t_02.mp4" {
		t.Fatalf("got=%q", got)
	}

	if got := buildDouyinFallbackFilename("", 0, 1, ""); got != "douyin_unknown.bin" {
		t.Fatalf("got=%q", got)
	}
	if got := buildDouyinFallbackFilename("id", 0, 2, ".jpg"); got != "douyin_id_01.jpg" {
		t.Fatalf("got=%q", got)
	}

	cd := buildAttachmentContentDisposition("", "")
	if !strings.Contains(cd, "filename=") || !strings.Contains(cd, "filename*=") {
		t.Fatalf("cd=%q", cd)
	}
}

func TestSanitizeFilename(t *testing.T) {
	if got := sanitizeFilename(" "); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := sanitizeFilename("a/b:c*?\"<>\n\t|"); !strings.Contains(got, "_") {
		t.Fatalf("got=%q", got)
	}
	long := strings.Repeat("中", 200)
	if got := sanitizeFilename(long); len([]rune(got)) != 100 {
		t.Fatalf("len=%d got=%q", len([]rune(got)), got)
	}
}

func TestWriteDouyinDownloadHeaders_ContentLengthFallback(t *testing.T) {
	a := &App{}
	h := make(http.Header)
	h.Set("Content-Type", "")
	h.Set("Content-Length", "10")
	h.Set("Accept-Ranges", "bytes")

	rec := httptest.NewRecorder()
	a.writeDouyinDownloadHeaders(rec, &douyinCachedDetail{Title: "t", DetailID: "d1", Downloads: []string{"x"}}, 0, "http://example.com/x", h, false)

	if got := rec.Header().Get("Content-Length"); got != "10" {
		t.Fatalf("len=%q", got)
	}
	if got := rec.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("ar=%q", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("ct=%q", ct)
	}
}

func ioNopCloser(body string) *nopCloser {
	return &nopCloser{r: bytes.NewBufferString(body)}
}

type nopCloser struct{ r *bytes.Buffer }

func (n *nopCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n *nopCloser) Close() error               { return nil }

func TestGuessDouyinMediaTypeFromURL(t *testing.T) {
	if got := guessDouyinMediaTypeFromURL(""); got != "image" {
		t.Fatalf("got=%q", got)
	}
	if got := guessDouyinMediaTypeFromURL("http://x/a.mp4"); got != "video" {
		t.Fatalf("got=%q", got)
	}
	if got := guessDouyinMediaTypeFromURL("http://x/a.jpg"); got != "image" {
		t.Fatalf("got=%q", got)
	}
	if got := guessDouyinMediaTypeFromURL("http://x/noext"); got != "image" {
		t.Fatalf("got=%q", got)
	}
	if got := guessDouyinMediaTypeFromURL("http://x/aweme/v1/play/?video_id=1"); got != "video" {
		t.Fatalf("got=%q", got)
	}
}

func TestFirstStringFromURLList(t *testing.T) {
	if got := firstStringFromURLList(nil); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := firstStringFromURLList([]any{" x "}); got != "x" {
		t.Fatalf("got=%q", got)
	}
}

func TestExtractDouyinAccountVideoPlayURL(t *testing.T) {
	if got := extractDouyinAccountVideoPlayURL(nil); got != "" {
		t.Fatalf("got=%q", got)
	}
	if got := extractDouyinAccountVideoPlayURL(map[string]any{}); got != "" {
		t.Fatalf("got=%q", got)
	}
	item := map[string]any{
		"video": map[string]any{
			"play_addr": map[string]any{
				"url_list": []any{" https://v "},
			},
		},
	}
	if got := extractDouyinAccountVideoPlayURL(item); got != "https://v" {
		t.Fatalf("got=%q", got)
	}
	item2 := map[string]any{
		"video": map[string]any{
			"downloadAddr": map[string]any{"url": " https://u "},
		},
	}
	if got := extractDouyinAccountVideoPlayURL(item2); got != "https://u" {
		t.Fatalf("got=%q", got)
	}
	item2b := map[string]any{
		"video": map[string]any{
			"playAddr": map[string]any{"urlList": []any{" https://u2 "}},
		},
	}
	if got := extractDouyinAccountVideoPlayURL(item2b); got != "https://u2" {
		t.Fatalf("got=%q", got)
	}
	item3 := map[string]any{
		"video": map[string]any{
			"download_addr": " https://s ",
		},
	}
	if got := extractDouyinAccountVideoPlayURL(item3); got != "https://s" {
		t.Fatalf("got=%q", got)
	}

	item4 := map[string]any{
		"video": map[string]any{
			"play_addr":    nil,
			"downloadAddr": map[string]any{"urlList": []any{" https://u3 "}},
		},
	}
	if got := extractDouyinAccountVideoPlayURL(item4); got != "https://u3" {
		t.Fatalf("got=%q", got)
	}

	item5 := map[string]any{
		"video": map[string]any{
			"play_addr": map[string]any{"url_list": []any{}},
			"playAddr":  1,
		},
	}
	if got := extractDouyinAccountVideoPlayURL(item5); got != "" {
		t.Fatalf("got=%q", got)
	}
}

func TestExtractDouyinAccountImageURLsAndDownloads(t *testing.T) {
	if got := extractDouyinAccountImageURLs(map[string]any{}, true); got != nil {
		t.Fatalf("got=%v", got)
	}
	item := map[string]any{
		"images": []any{
			map[string]any{"url_list": []any{" https://a "}},
			"bad",
			map[string]any{"urlList": []any{" https://a2 "}},
			map[string]any{"download_url_list": []any{" https://b1 "}},
			map[string]any{"downloadUrlList": []any{" https://b "}},
		},
	}
	got := extractDouyinAccountImageURLs(item, true)
	if len(got) != 4 || got[0] != "https://a" || got[1] != "https://a2" || got[2] != "https://b1" || got[3] != "https://b" {
		t.Fatalf("got=%v", got)
	}
	if d := extractDouyinAccountFlatDownloads(nil); d != nil {
		t.Fatalf("d=%v", d)
	}
	if d := extractDouyinAccountFlatDownloads(map[string]any{"downloads": []any{" https://x "}}); len(d) != 1 || strings.TrimSpace(d[0]) != "https://x" {
		t.Fatalf("d=%v", d)
	}
	if d := extractDouyinAccountFlatDownloads(map[string]any{"downloads": " ", "download": " https://y "}); len(d) != 1 || strings.TrimSpace(d[0]) != "https://y" {
		t.Fatalf("d=%v", d)
	}
	if d := extractDouyinAccountFlatDownloads(map[string]any{}); d != nil {
		t.Fatalf("d=%v", d)
	}
	if d := extractDouyinAccountFlatDownloads(map[string]any{"downloadUrl": " https://d "}); len(d) != 1 || strings.TrimSpace(d[0]) != "https://d" {
		t.Fatalf("d=%v", d)
	}
}

func TestExtractDouyinAccountItems(t *testing.T) {
	s := NewDouyinDownloaderService("", "", "", "", time.Second)

	data := map[string]any{
		"awemeList": []any{
			map[string]any{"desc": "skip"},
			map[string]any{
				"awemeId": "1",
				"type":    "图集",
				"desc":    "m1",
				"downloads": []any{
					"http://example.com/img",
					"http://example.com/aweme/v1/play/?video_id=1",
					" ",
				},
			},
			map[string]any{
				"aweme_id": "2",
				"desc":     "m2",
				"video": map[string]any{
					"cover": map[string]any{"url_list": []any{"http://example.com/c2.jpg"}},
					"play_addr": map[string]any{
						"url_list": []any{"http://example.com/v2.mp4"},
					},
				},
			},
			map[string]any{
				"id":   "3",
				"desc": "m3",
				"images": []any{
					map[string]any{"url_list": []any{"http://example.com/i1.jpg"}},
				},
			},
			map[string]any{
				"aweme_id": "4",
				"desc":     "m4",
				"video": map[string]any{
					"play_addr": map[string]any{"url_list": []any{"http://example.com/v4.mp4"}},
				},
			},
		},
	}

	items := extractDouyinAccountItems(s, "sec1", data)
	if len(items) != 4 {
		t.Fatalf("len=%d items=%v", len(items), items)
	}
	if strings.TrimSpace(items[0].Key) == "" || len(items[0].Items) == 0 {
		t.Fatalf("item0=%v", items[0])
	}
	if items[0].CoverURL == "" || items[0].CoverDownloadURL == "" {
		t.Fatalf("item0=%v", items[0])
	}
	if items[1].DetailID != "2" || items[2].DetailID != "3" {
		t.Fatalf("items=%v", items)
	}
	if items[3].DetailID != "4" || items[3].CoverDownloadURL != "" {
		t.Fatalf("item3=%v", items[3])
	}

	noCache := extractDouyinAccountItems(nil, "", data)
	if len(noCache) != 4 || noCache[0].Key != "" || len(noCache[0].Items) != 0 {
		t.Fatalf("noCache=%v", noCache[0])
	}
}

func TestExtractDouyinAccountItems_EarlyReturns(t *testing.T) {
	if got := extractDouyinAccountItems(nil, "", nil); got != nil {
		t.Fatalf("expected nil")
	}
	if got := extractDouyinAccountItems(nil, "", map[string]any{"aweme_list": "x"}); got != nil {
		t.Fatalf("expected nil")
	}
	if got := extractDouyinAccountItems(nil, "", map[string]any{"aweme_list": []any{}}); got != nil {
		t.Fatalf("expected nil")
	}

	items := extractDouyinAccountItems(nil, "", map[string]any{"aweme_list": []any{"not-a-map"}})
	if items == nil || len(items) != 0 {
		t.Fatalf("items=%v", items)
	}
}
