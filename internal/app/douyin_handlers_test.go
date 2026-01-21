package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestHandleDouyinDetailAndDownload_Video(t *testing.T) {
	videoBytes := []byte("video-bytes")

	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/douyin/share":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"请求链接成功！","url":"https://www.douyin.com/video/123456","params":{},"time":"2026-01-01"}`))
		case "/douyin/detail":
			w.Header().Set("Content-Type", "application/json")
			payload := map[string]any{
				"message": "获取数据成功！",
				"data": map[string]any{
					"id":          "123456",
					"desc":        "测试标题",
					"type":        "视频",
					"downloads":    upstream.URL + "/media.mp4",
					"static_cover": upstream.URL + "/cover.jpg",
				},
				"params": map[string]any{},
				"time":   "2026-01-01",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/media.mp4":
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Content-Length", strconv.Itoa(len(videoBytes)))
			if r.Method != http.MethodHead {
				_, _ = w.Write(videoBytes)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", ""),
	}

	// 1) detail（走 share -> detail）
	body := bytes.NewBufferString(`{"input":"https://v.douyin.com/xxxxxx/","cookie":"","proxy":""}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", body)
	rr := httptest.NewRecorder()
	a.handleDouyinDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("detail status=%d, body=%s", rr.Code, rr.Body.String())
	}

	var detailResp douyinDetailResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("unmarshal detail response failed: %v", err)
	}
	if detailResp.Key == "" {
		t.Fatalf("missing key: %+v", detailResp)
	}
	if detailResp.DetailID != "123456" {
		t.Fatalf("detailId=%q, want %q", detailResp.DetailID, "123456")
	}
	if detailResp.Title != "测试标题" {
		t.Fatalf("title=%q, want %q", detailResp.Title, "测试标题")
	}
	if len(detailResp.Items) != 1 {
		t.Fatalf("items len=%d, want 1", len(detailResp.Items))
	}
	if got := detailResp.Items[0].URL; got != upstream.URL+"/media.mp4" {
		t.Fatalf("item url=%q, want %q", got, upstream.URL+"/media.mp4")
	}

	// 2) download（透传媒体 + Content-Disposition）
	downloadReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+detailResp.Key+"&index=0", nil)
	downloadRR := httptest.NewRecorder()
	a.handleDouyinDownload(downloadRR, downloadReq)
	if downloadRR.Code != http.StatusOK {
		t.Fatalf("download status=%d, body=%s", downloadRR.Code, downloadRR.Body.String())
	}
	if !bytes.Equal(downloadRR.Body.Bytes(), videoBytes) {
		t.Fatalf("download body mismatch: got=%q", downloadRR.Body.Bytes())
	}
	cd := downloadRR.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "filename*=") {
		t.Fatalf("missing filename*: %q", cd)
	}
	if !strings.Contains(cd, "%E6%B5%8B%E8%AF%95%E6%A0%87%E9%A2%98") {
		t.Fatalf("content-disposition should include escaped title: %q", cd)
	}

	// 3) head（最佳努力返回 Content-Length/Disposition）
	headReq := httptest.NewRequest(http.MethodHead, "http://example.com/api/douyin/download?key="+detailResp.Key+"&index=0", nil)
	headRR := httptest.NewRecorder()
	a.handleDouyinDownload(headRR, headReq)
	if headRR.Code != http.StatusOK {
		t.Fatalf("head status=%d", headRR.Code)
	}
	if headRR.Body.Len() != 0 {
		t.Fatalf("head body should be empty, got=%q", headRR.Body.Bytes())
	}
	if got := headRR.Header().Get("Content-Length"); got != strconv.Itoa(len(videoBytes)) {
		t.Fatalf("content-length=%q, want %q", got, strconv.Itoa(len(videoBytes)))
	}
	if cd2 := headRR.Header().Get("Content-Disposition"); !strings.Contains(cd2, "filename*=") {
		t.Fatalf("missing filename* in head: %q", cd2)
	}
}

func TestHandleDouyinDownload_ExpiredKey(t *testing.T) {
	a := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:5555", "", "", ""),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key=missing&index=0", nil)
	rr := httptest.NewRecorder()
	a.handleDouyinDownload(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
	}
}
