package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
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
					"id":           "123456",
					"desc":         "测试标题",
					"type":         "视频",
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
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
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

func TestHandleDouyinDetail_Image(t *testing.T) {
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
					"id":           "123456",
					"desc":         "测试图片",
					"type":         "图片",
					"downloads":    []any{upstream.URL + "/img_noext"},
					"static_cover": upstream.URL + "/cover.jpg",
				},
				"params": map[string]any{},
				"time":   "2026-01-01",
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}

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
	if len(detailResp.Items) != 1 {
		t.Fatalf("items len=%d, want 1", len(detailResp.Items))
	}
	if detailResp.Items[0].Type != "image" {
		t.Fatalf("items[0].type=%q, want %q", detailResp.Items[0].Type, "image")
	}
	if !strings.HasSuffix(detailResp.Items[0].OriginalFilename, ".jpg") {
		t.Fatalf("items[0].originalFilename=%q should end with .jpg", detailResp.Items[0].OriginalFilename)
	}
}

func TestHandleDouyinDetail_LivePhotoMixed(t *testing.T) {
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
					"id":           "123456",
					"desc":         "测试实况",
					"type":         "实况",
					"downloads":    []any{upstream.URL + "/live_img_noext", upstream.URL + "/aweme/v1/play/?video_id=live1"},
					"static_cover": upstream.URL + "/cover.jpg",
				},
				"params": map[string]any{},
				"time":   "2026-01-01",
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}

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
	if len(detailResp.Items) != 2 {
		t.Fatalf("items len=%d, want 2", len(detailResp.Items))
	}
	if detailResp.Items[0].Type != "image" {
		t.Fatalf("items[0].type=%q, want %q", detailResp.Items[0].Type, "image")
	}
	if detailResp.Items[1].Type != "video" {
		t.Fatalf("items[1].type=%q, want %q", detailResp.Items[1].Type, "video")
	}
	if !strings.HasSuffix(detailResp.Items[1].OriginalFilename, ".mp4") {
		t.Fatalf("items[1].originalFilename=%q should end with .mp4", detailResp.Items[1].OriginalFilename)
	}
}

func TestHandleDouyinDownload_PassthroughRange(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/media.mp4" {
			http.NotFound(w, r)
			return
		}

		// 简化：只验证 Range 被透传 + 下游返回 206/Content-Range
		if strings.TrimSpace(r.Header.Get("Range")) == "bytes=0-3" {
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", "bytes 0-3/10")
			w.Header().Set("Content-Length", "4")
			w.WriteHeader(http.StatusPartialContent)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte("abcd"))
			}
			return
		}

		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte("0123456789"))
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}

	key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "123456",
		Title:     "测试标题",
		Type:      "视频",
		Downloads: []string{upstream.URL + "/media.mp4"},
	})
	if key == "" {
		t.Fatalf("missing key")
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key="+key+"&index=0", nil)
	downloadReq.Header.Set("Range", "bytes=0-3")
	downloadRR := httptest.NewRecorder()
	a.handleDouyinDownload(downloadRR, downloadReq)

	if downloadRR.Code != http.StatusPartialContent {
		t.Fatalf("download status=%d, want %d", downloadRR.Code, http.StatusPartialContent)
	}
	if got := strings.TrimSpace(downloadRR.Header().Get("Content-Range")); got != "bytes 0-3/10" {
		t.Fatalf("content-range=%q, want %q", got, "bytes 0-3/10")
	}
	if got := strings.TrimSpace(downloadRR.Header().Get("Accept-Ranges")); got != "bytes" {
		t.Fatalf("accept-ranges=%q, want %q", got, "bytes")
	}
	if got := strings.TrimSpace(downloadRR.Header().Get("Content-Length")); got != "4" {
		t.Fatalf("content-length=%q, want %q", got, "4")
	}
	if got := downloadRR.Body.String(); got != "abcd" {
		t.Fatalf("body=%q, want %q", got, "abcd")
	}
}

func TestHandleDouyinDownload_ExpiredKey(t *testing.T) {
	a := &App{
		douyinDownloader: NewDouyinDownloaderService("http://127.0.0.1:5555", "", "", "", 60*time.Second),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/download?key=missing&index=0", nil)
	rr := httptest.NewRecorder()
	a.handleDouyinDownload(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDouyinCover_Get(t *testing.T) {
	coverBytes := []byte("cover-bytes")

	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cover.jpg" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(coverBytes)))
		if r.Method != http.MethodHead {
			_, _ = w.Write(coverBytes)
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}
	key := a.douyinDownloader.CacheDetail(&douyinCachedDetail{
		DetailID:  "123456",
		Title:     "测试标题",
		Type:      "视频",
		CoverURL:  upstream.URL + "/cover.jpg",
		Downloads: []string{upstream.URL + "/media.mp4"},
	})
	if key == "" {
		t.Fatalf("missing key")
	}

	coverReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/douyin/cover?key="+key, nil)
	coverRR := httptest.NewRecorder()
	a.handleDouyinCover(coverRR, coverReq)

	if coverRR.Code != http.StatusOK {
		t.Fatalf("cover status=%d, body=%s", coverRR.Code, coverRR.Body.String())
	}
	if got := strings.TrimSpace(coverRR.Header().Get("Content-Type")); got != "image/jpeg" {
		t.Fatalf("content-type=%q, want %q", got, "image/jpeg")
	}
	if !bytes.Equal(coverRR.Body.Bytes(), coverBytes) {
		t.Fatalf("cover body mismatch: got=%q", coverRR.Body.Bytes())
	}
}

func TestHandleDouyinAccount_Posts(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/douyin/share":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"请求链接成功！","url":"https://www.douyin.com/user/MS4wLjABAAAA_test_secuid","params":{},"time":"2026-01-01"}`))
		case "/douyin/account":
			w.Header().Set("Content-Type", "application/json")
			payload := map[string]any{
				"message": "获取数据成功！",
				"data": map[string]any{
					"cursor":   123,
					"has_more": 1,
					"user": map[string]any{
						"nickname": "测试用户",
					},
					"aweme_list": []any{
						map[string]any{
							"aweme_id": "111",
							"desc":     "作品1",
							"video": map[string]any{
								"cover": map[string]any{
									"url_list": []any{upstream.URL + "/cover1.jpg"},
								},
								"play_addr": map[string]any{
									"url_list": []any{upstream.URL + "/media1.mp4"},
								},
							},
						},
						map[string]any{
							"aweme_id": "222",
							"desc":     "作品2",
							"images": []any{
								map[string]any{
									"url_list": []any{upstream.URL + "/cover2.jpg"},
								},
								map[string]any{
									"url_list": []any{upstream.URL + "/img2.jpg"},
								},
							},
						},
					},
				},
				"params": map[string]any{},
				"time":   "2026-01-01",
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}

	body := bytes.NewBufferString(`{"input":"https://v.douyin.com/xxxxxx/","cookie":"","tab":"post","cursor":0,"count":18}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", body)
	rr := httptest.NewRecorder()
	a.handleDouyinAccount(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("account status=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp douyinAccountResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal account response failed: %v", err)
	}
	if resp.SecUserID != "MS4wLjABAAAA_test_secuid" {
		t.Fatalf("secUserId=%q, want %q", resp.SecUserID, "MS4wLjABAAAA_test_secuid")
	}
	if resp.DisplayName != "测试用户" {
		t.Fatalf("displayName=%q, want %q", resp.DisplayName, "测试用户")
	}
	if resp.Tab != "post" {
		t.Fatalf("tab=%q, want %q", resp.Tab, "post")
	}
	if resp.Cursor != 123 {
		t.Fatalf("cursor=%d, want %d", resp.Cursor, 123)
	}
	if !resp.HasMore {
		t.Fatalf("hasMore=false, want true")
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items len=%d, want 2", len(resp.Items))
	}
	if resp.Items[0].DetailID != "111" {
		t.Fatalf("items[0].detailId=%q, want %q", resp.Items[0].DetailID, "111")
	}
	if resp.Items[0].Type != "video" {
		t.Fatalf("items[0].type=%q, want %q", resp.Items[0].Type, "video")
	}
	if strings.TrimSpace(resp.Items[0].Key) == "" {
		t.Fatalf("items[0].key should not be empty")
	}
	if !strings.Contains(resp.Items[0].CoverDownloadURL, "/api/douyin/cover?key=") {
		t.Fatalf("items[0].coverDownloadUrl=%q", resp.Items[0].CoverDownloadURL)
	}
	if len(resp.Items[0].Items) != 1 {
		t.Fatalf("items[0].items len=%d, want 1", len(resp.Items[0].Items))
	}
	if !strings.Contains(resp.Items[0].Items[0].DownloadURL, "/api/douyin/download?key=") {
		t.Fatalf("items[0].items[0].downloadUrl=%q", resp.Items[0].Items[0].DownloadURL)
	}
	if resp.Items[1].DetailID != "222" {
		t.Fatalf("items[1].detailId=%q, want %q", resp.Items[1].DetailID, "222")
	}
	if resp.Items[1].Type != "image" {
		t.Fatalf("items[1].type=%q, want %q", resp.Items[1].Type, "image")
	}
	if strings.TrimSpace(resp.Items[1].Key) == "" {
		t.Fatalf("items[1].key should not be empty")
	}
	if len(resp.Items[1].Items) != 2 {
		t.Fatalf("items[1].items len=%d, want 2", len(resp.Items[1].Items))
	}
}

func TestHandleDouyinAccount_Posts_FlatDataArray(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/douyin/share":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"请求链接成功！","url":"https://www.douyin.com/user/MS4wLjABAAAA_test_secuid","params":{},"time":"2026-01-01"}`))
		case "/douyin/account":
			w.Header().Set("Content-Type", "application/json")
			payload := map[string]any{
				"message": "获取数据成功！",
				"data": []any{
					map[string]any{
						"id":           "111",
						"desc":         "作品1",
						"type":         "视频",
						"downloads":    upstream.URL + "/aweme/v1/play/?video_id=v1",
						"static_cover": upstream.URL + "/cover1.jpg",
					},
					map[string]any{
						"id":        "222",
						"desc":      "作品2",
						"type":      "图集",
						"downloads": []any{upstream.URL + "/img1.jpeg", upstream.URL + "/img2.jpeg"},
					},
					map[string]any{
						"id":        "333",
						"desc":      "作品3",
						"type":      "实况",
						"downloads": []any{upstream.URL + "/live_img.jpeg", upstream.URL + "/aweme/v1/play/?video_id=live1"},
					},
				},
				"params": map[string]any{},
				"time":   "2026-01-01",
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}

	body := bytes.NewBufferString(`{"input":"https://v.douyin.com/xxxxxx/","cookie":"","tab":"post","cursor":0,"count":18}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", body)
	rr := httptest.NewRecorder()
	a.handleDouyinAccount(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("account status=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp douyinAccountResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal account response failed: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("items len=%d, want 3", len(resp.Items))
	}

	// video
	if resp.Items[0].DetailID != "111" {
		t.Fatalf("items[0].detailId=%q, want %q", resp.Items[0].DetailID, "111")
	}
	if resp.Items[0].Type != "video" {
		t.Fatalf("items[0].type=%q, want %q", resp.Items[0].Type, "video")
	}
	if strings.TrimSpace(resp.Items[0].Key) == "" {
		t.Fatalf("items[0].key should not be empty")
	}
	if !strings.Contains(resp.Items[0].CoverDownloadURL, "/api/douyin/cover?key=") {
		t.Fatalf("items[0].coverDownloadUrl=%q", resp.Items[0].CoverDownloadURL)
	}
	if len(resp.Items[0].Items) != 1 {
		t.Fatalf("items[0].items len=%d, want 1", len(resp.Items[0].Items))
	}
	if resp.Items[0].Items[0].Type != "video" {
		t.Fatalf("items[0].items[0].type=%q, want %q", resp.Items[0].Items[0].Type, "video")
	}
	if !strings.Contains(resp.Items[0].Items[0].DownloadURL, "/api/douyin/download?key=") {
		t.Fatalf("items[0].items[0].downloadUrl=%q", resp.Items[0].Items[0].DownloadURL)
	}

	// image
	if resp.Items[1].DetailID != "222" {
		t.Fatalf("items[1].detailId=%q, want %q", resp.Items[1].DetailID, "222")
	}
	if resp.Items[1].Type != "image" {
		t.Fatalf("items[1].type=%q, want %q", resp.Items[1].Type, "image")
	}
	if strings.TrimSpace(resp.Items[1].Key) == "" {
		t.Fatalf("items[1].key should not be empty")
	}
	if !strings.Contains(resp.Items[1].CoverDownloadURL, "/api/douyin/cover?key=") {
		t.Fatalf("items[1].coverDownloadUrl=%q", resp.Items[1].CoverDownloadURL)
	}
	if len(resp.Items[1].Items) != 2 {
		t.Fatalf("items[1].items len=%d, want 2", len(resp.Items[1].Items))
	}
	if resp.Items[1].Items[0].Type != "image" {
		t.Fatalf("items[1].items[0].type=%q, want %q", resp.Items[1].Items[0].Type, "image")
	}

	// live photo (mixed)
	if resp.Items[2].DetailID != "333" {
		t.Fatalf("items[2].detailId=%q, want %q", resp.Items[2].DetailID, "333")
	}
	if resp.Items[2].Type != "video" {
		t.Fatalf("items[2].type=%q, want %q", resp.Items[2].Type, "video")
	}
	if strings.TrimSpace(resp.Items[2].Key) == "" {
		t.Fatalf("items[2].key should not be empty")
	}
	if !strings.Contains(resp.Items[2].CoverDownloadURL, "/api/douyin/cover?key=") {
		t.Fatalf("items[2].coverDownloadUrl=%q", resp.Items[2].CoverDownloadURL)
	}
	if len(resp.Items[2].Items) != 2 {
		t.Fatalf("items[2].items len=%d, want 2", len(resp.Items[2].Items))
	}
	if resp.Items[2].Items[0].Type != "image" {
		t.Fatalf("items[2].items[0].type=%q, want %q", resp.Items[2].Items[0].Type, "image")
	}
	if resp.Items[2].Items[1].Type != "video" {
		t.Fatalf("items[2].items[1].type=%q, want %q", resp.Items[2].Items[1].Type, "video")
	}
}

func TestHandleDouyinAccount_NoPosts_DataArray(t *testing.T) {
	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/douyin/share":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"请求链接成功！","url":"https://www.douyin.com/user/MS4wLjABAAAA_test_secuid","params":{},"time":"2026-01-01"}`))
		case "/douyin/account":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"暂无作品","data":[],"params":{},"time":"2026-01-01"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	a := &App{
		douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second),
	}

	body := bytes.NewBufferString(`{"input":"https://v.douyin.com/xxxxxx/","cookie":"","tab":"post","cursor":0,"count":18}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/account", body)
	rr := httptest.NewRecorder()
	a.handleDouyinAccount(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("account status=%d, body=%s", rr.Code, rr.Body.String())
	}

	var resp douyinAccountResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal account response failed: %v", err)
	}
	if resp.Items == nil {
		t.Fatalf("items should be [], got null (raw=%s)", rr.Body.String())
	}
	if len(resp.Items) != 0 {
		t.Fatalf("items len=%d, want 0", len(resp.Items))
	}
	if resp.HasMore {
		t.Fatalf("hasMore=true, want false")
	}
}

func TestParseContentRangeTotal(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int64
	}{
		{name: "empty", in: "", want: 0},
		{name: "spaces", in: "   ", want: 0},
		{name: "noSlash", in: "bytes 0-1", want: 0},
		{name: "wildcardTotal", in: "bytes 0-1/*", want: 0},
		{name: "wildcardRange", in: "bytes */1048576", want: 1048576},
		{name: "valid", in: "bytes 0-1/10", want: 10},
		{name: "nonNumber", in: "bytes 0-1/abc", want: 0},
		{name: "zero", in: "bytes 0-1/0", want: 0},
		{name: "negative", in: "bytes 0-1/-1", want: 0},
		{name: "overflow", in: "bytes 0-1/9223372036854775808", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseContentRangeTotal(tc.in); got != tc.want {
				t.Fatalf("parseContentRangeTotal(%q)=%d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
