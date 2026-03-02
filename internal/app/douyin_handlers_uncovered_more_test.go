package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDouyinNumericAndDurationHelpers_UncoveredBranches(t *testing.T) {
	if got := asFloat64(float64(1.5)); got != 1.5 {
		t.Fatalf("float64 got=%v", got)
	}
	if got := asFloat64(float32(2.5)); got != 2.5 {
		t.Fatalf("float32 got=%v", got)
	}
	if got := asFloat64(int(3)); got != 3 {
		t.Fatalf("int got=%v", got)
	}
	if got := asFloat64(int64(4)); got != 4 {
		t.Fatalf("int64 got=%v", got)
	}
	if got := asFloat64("5.5"); got != 5.5 {
		t.Fatalf("string got=%v", got)
	}
	if got := asFloat64(struct{}{}); got != 0 {
		t.Fatalf("default got=%v", got)
	}

	if got := extractDouyinAccountVideoDurationSeconds(nil); got != 0 {
		t.Fatalf("nil duration=%v", got)
	}
}

func TestExtractDouyinAccountItems_TypeInferenceBranches(t *testing.T) {
	data := map[string]any{
		"aweme_list": []any{
			map[string]any{ // typeLabel contains 实况 but normalize result empty
				"aweme_id":  "a1",
				"type":      "我的实况作品",
				"downloads": []any{"https://example.com/a1.jpg", "https://example.com/aweme/v1/play/?video_id=v1"},
			},
			map[string]any{ // itemType=image but hasImage=false, should hit mediaType=itemType branch
				"aweme_id":  "a2",
				"type":      "图集",
				"downloads": []any{"https://example.com/aweme/v1/play/?video_id=v2"},
			},
		},
	}
	items := extractDouyinAccountItems(nil, "", data)
	if len(items) != 2 {
		t.Fatalf("items=%v", items)
	}
	if items[0].MediaType != "livePhoto" || !items[0].IsLivePhoto {
		t.Fatalf("item0=%+v", items[0])
	}
	if items[1].MediaType != "imageAlbum" {
		t.Fatalf("item1=%+v", items[1])
	}
}

func TestExtractDouyinAccountItems_UncoveredMediaTypeBranches(t *testing.T) {
	data := map[string]any{
		"aweme_list": []any{
			map[string]any{
				"aweme_id": "b1",
				// itemType=image (because images exists), but no image URL and no nested live video;
				// downloads fallback provides only video => hit `case itemType == "image"`.
				"images":    []any{map[string]any{}},
				"downloads": []any{"https://example.com/aweme/v1/play/?video_id=v1"},
			},
			map[string]any{
				"aweme_id": "b2",
				"type":     "视频", // typeHint=video
				"images": []any{
					map[string]any{
						"url_list": []any{"https://example.com/i1.jpg"},
						"video":    map[string]any{"play_addr": map[string]any{"url_list": []any{"https://example.com/aweme/v1/play/?video_id=v2"}}},
					},
				},
			},
			map[string]any{
				"aweme_id": "b3",
				// typeLabel empty + nestedVideos(2) + imageCount(1) => cover typeLabel default and livePhotoPairs min(imageCount, videos)
				"images": []any{
					map[string]any{
						"url_list": []any{"https://example.com/i2.jpg"},
						"video":    map[string]any{"play_addr": map[string]any{"url_list": []any{"https://example.com/aweme/v1/play/?video_id=v3"}}},
					},
					map[string]any{
						"video": map[string]any{"play_addr": map[string]any{"url_list": []any{"https://example.com/aweme/v1/play/?video_id=v4"}}},
					},
				},
			},
		},
	}

	items := extractDouyinAccountItems(nil, "", data)
	if len(items) != 3 {
		t.Fatalf("items=%v", items)
	}
	if items[0].MediaType != "imageAlbum" {
		t.Fatalf("item0=%+v", items[0])
	}
	if items[1].MediaType != "livePhoto" {
		t.Fatalf("item1=%+v", items[1])
	}
	if items[2].MediaType != "livePhoto" || items[2].LivePhotoPairs != 1 {
		t.Fatalf("item2=%+v", items[2])
	}
}

func TestHandleDouyinDetail_MediaTypeAndLivePhotoPairBranches(t *testing.T) {
	type testCase struct {
		name      string
		detail    map[string]any
		wantType  string
		wantPairs int
	}

	cases := []testCase{
		{
			name: "mixed downloads infer livePhoto and min pair",
			detail: map[string]any{
				"id":        "d1",
				"desc":      "t1",
				"type":      "",
				"downloads": []any{"https://example.com/a.jpg", "https://example.com/b.jpg", "https://example.com/aweme/v1/play/?video_id=1"},
			},
			wantType:  "livePhoto",
			wantPairs: 1,
		},
		{
			name: "contains 实况 forces livePhoto",
			detail: map[string]any{
				"id":        "d2",
				"desc":      "t2",
				"type":      "我的实况作品",
				"downloads": []any{"https://example.com/a.jpg"},
			},
			wantType:  "livePhoto",
			wantPairs: 1,
		},
		{
			name: "video fallback type",
			detail: map[string]any{
				"id":        "d3",
				"desc":      "t3",
				"type":      "",
				"downloads": []any{"https://example.com/aweme/v1/play/?video_id=3"},
			},
			wantType:  "video",
			wantPairs: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var upstream *httptest.Server
			upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/douyin/share":
					_ = json.NewEncoder(w).Encode(map[string]any{"message": "请求链接成功！", "url": "https://www.douyin.com/video/123"})
				case "/douyin/detail":
					_ = json.NewEncoder(w).Encode(map[string]any{"message": "获取数据成功！", "data": tc.detail})
				default:
					http.NotFound(w, r)
				}
			}))
			defer upstream.Close()

			app := &App{douyinDownloader: NewDouyinDownloaderService(upstream.URL, "", "", "", 60*time.Second)}
			req := httptest.NewRequest(http.MethodPost, "http://example.com/api/douyin/detail", bytes.NewBufferString(`{"input":"https://v.douyin.com/abc/"}`))
			rr := httptest.NewRecorder()
			app.handleDouyinDetail(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			var resp douyinDetailResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal err=%v", err)
			}
			if resp.MediaType != tc.wantType {
				t.Fatalf("mediaType=%q want=%q resp=%+v", resp.MediaType, tc.wantType, resp)
			}
			if resp.LivePhotoPairs != tc.wantPairs {
				t.Fatalf("pairs=%d want=%d resp=%+v", resp.LivePhotoPairs, tc.wantPairs, resp)
			}
		})
	}
}
