package app

import (
	"strings"
	"testing"
	"time"
)

func TestDouyinHelperBranches_MissingPaths(t *testing.T) {
	if pinned, rank, pinnedAt := extractDouyinAccountPinned(nil); pinned || rank != nil || pinnedAt != "" {
		t.Fatalf("unexpected pinned result: pinned=%v rank=%v pinnedAt=%q", pinned, rank, pinnedAt)
	}

	if uid, name := extractDouyinAccountAuthorMeta(nil); uid != "" || name != "" {
		t.Fatalf("unexpected author meta uid=%q name=%q", uid, name)
	}

	if got := pickPreferredURLFromSlice([]string{" ", "\t"}, []string{".jpg"}); got != "" {
		t.Fatalf("expected empty preferred url, got=%q", got)
	}
	if got := pickPreferredDouyinImageURL(nil, true); got != "" {
		t.Fatalf("expected empty image url, got=%q", got)
	}
	if got := extractDouyinVideoPlayURLFromVideoMap(nil); got != "" {
		t.Fatalf("expected empty video url, got=%q", got)
	}

	live := extractDouyinAccountLivePhotoVideoPlayURLs(map[string]any{"images": []any{"not-a-map"}})
	if len(live) != 0 {
		t.Fatalf("expected empty live list, got=%v", live)
	}

	if got := extractDouyinAvatarURL(nil); got != "" {
		t.Fatalf("expected empty avatar for nil user, got=%q", got)
	}
	if got := extractDouyinAvatarURL(map[string]any{"avatar_larger": map[string]any{"urlList": []any{" https://img.example.com/a1.jpg "}}}); got != "https://img.example.com/a1.jpg" {
		t.Fatalf("avatar from urlList=%q", got)
	}
	if got := extractDouyinAvatarURL(map[string]any{"avatar_larger": map[string]any{"url": " https://img.example.com/a2.jpg "}}); got != "https://img.example.com/a2.jpg" {
		t.Fatalf("avatar from url=%q", got)
	}
	if got := extractDouyinAvatarURL(map[string]any{"avatar": " https://img.example.com/a3.jpg "}); got != "https://img.example.com/a3.jpg" {
		t.Fatalf("avatar from raw value=%q", got)
	}

	if got := extractDouyinDisplayName(nil); got != "" {
		t.Fatalf("expected empty display name for nil user, got=%q", got)
	}
	if got := extractDouyinDisplayName(map[string]any{"nickname": " "}); got != "" {
		t.Fatalf("expected empty display name for blank values, got=%q", got)
	}

	if got := extractDouyinSignature(nil); got != "" {
		t.Fatalf("expected empty signature for nil user, got=%q", got)
	}
	if got := extractDouyinSignature(map[string]any{"signature": " "}); got != "" {
		t.Fatalf("expected empty signature for blank values, got=%q", got)
	}

	if got := pickInt64Ptr(nil, []string{"a"}); got != nil {
		t.Fatalf("expected nil int pointer, got=%v", got)
	}
	fc, fg, ac, tf := extractDouyinUserStats(nil)
	if fc != nil || fg != nil || ac != nil || tf != nil {
		t.Fatalf("expected nil stats for nil user, got fc=%v fg=%v ac=%v tf=%v", fc, fg, ac, tf)
	}

	displayName, signature, avatarURL, profileURL, fc2, fg2, ac2, tf2 := extractDouyinAccountUserMeta(" sec-user-id ", nil)
	if displayName != "" || signature != "" || avatarURL != "" {
		t.Fatalf("unexpected user meta display=%q signature=%q avatar=%q", displayName, signature, avatarURL)
	}
	if !strings.Contains(profileURL, "/user/") {
		t.Fatalf("expected profile url, got=%q", profileURL)
	}
	if fc2 != nil || fg2 != nil || ac2 != nil || tf2 != nil {
		t.Fatalf("expected nil stats for nil data, got fc=%v fg=%v ac=%v tf=%v", fc2, fg2, ac2, tf2)
	}
}

func TestExtractDouyinAccountItems_LivePhotoDuplicateBranches(t *testing.T) {
	svc := NewDouyinDownloaderService("", "", "", "", time.Second)
	data := map[string]any{
		"aweme_list": []any{
			map[string]any{
				"aweme_id": "dup-1",
				"type":     "实况",
				"desc":     "live",
				"images": []any{
					map[string]any{
						"url_list": []any{"http://media.example.com/live.mp4"},
						"video": map[string]any{
							"play_addr": map[string]any{"url_list": []any{"http://media.example.com/live.mp4"}},
						},
					},
				},
				"video": map[string]any{
					"play_addr": map[string]any{"url_list": []any{"http://media.example.com/live2.mp4"}},
				},
			},
		},
	}

	items := extractDouyinAccountItems(svc, "sec-user", data)
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	if strings.TrimSpace(items[0].Key) == "" {
		t.Fatalf("expected cached key, got empty")
	}
	if len(items[0].Items) != 2 {
		t.Fatalf("expected 2 preview items, got=%v", items[0].Items)
	}
	if got := items[0].Items[0].URL; got != "http://media.example.com/live.mp4" {
		t.Fatalf("first preview url=%q", got)
	}
	if got := items[0].Items[1].URL; got != "http://media.example.com/live2.mp4" {
		t.Fatalf("second preview url=%q", got)
	}
}

func TestExtractDouyinAccountItems_LivePhotoSameVideoSkipped(t *testing.T) {
	svc := NewDouyinDownloaderService("", "", "", "", time.Second)
	data := map[string]any{
		"aweme_list": []any{
			map[string]any{
				"aweme_id": "dup-2",
				"type":     "实况",
				"desc":     "live",
				"images": []any{
					map[string]any{
						"url_list": []any{"http://media.example.com/live.mp4"},
						"video": map[string]any{
							"play_addr": map[string]any{"url_list": []any{"http://media.example.com/live.mp4"}},
						},
					},
				},
				"video": map[string]any{
					"play_addr": map[string]any{"url_list": []any{"http://media.example.com/live.mp4"}},
				},
			},
		},
	}

	items := extractDouyinAccountItems(svc, "sec-user", data)
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	if strings.TrimSpace(items[0].Key) == "" {
		t.Fatalf("expected cached key, got empty")
	}
	if len(items[0].Items) != 1 {
		t.Fatalf("expected duplicate live video to be skipped, got=%v", items[0].Items)
	}
	if got := items[0].Items[0].URL; got != "http://media.example.com/live.mp4" {
		t.Fatalf("preview url=%q", got)
	}
}

func TestExtractDouyinAccountItems_MediaMetaFields(t *testing.T) {
	data := map[string]any{
		"aweme_list": []any{
			map[string]any{
				"aweme_id": "video-1",
				"type":     "视频",
				"video": map[string]any{
					"duration": 12000,
					"play_addr": map[string]any{
						"url_list": []any{"http://media.example.com/v1.mp4"},
					},
				},
			},
			map[string]any{
				"aweme_id": "album-1",
				"type":     "图集",
				"images": []any{
					map[string]any{"url_list": []any{"http://media.example.com/a1.jpg"}},
					map[string]any{"url_list": []any{"http://media.example.com/a2.jpg"}},
				},
			},
			map[string]any{
				"aweme_id": "live-1",
				"type":     "实况",
				"images": []any{
					map[string]any{
						"url_list": []any{"http://media.example.com/l1.jpg"},
						"video": map[string]any{
							"play_addr": map[string]any{"url_list": []any{"http://media.example.com/l1.mp4"}},
						},
					},
					map[string]any{
						"url_list": []any{"http://media.example.com/l2.jpg"},
						"video": map[string]any{
							"play_addr": map[string]any{"url_list": []any{"http://media.example.com/l2.mp4"}},
						},
					},
				},
			},
		},
	}

	items := extractDouyinAccountItems(nil, "", data)
	if len(items) != 3 {
		t.Fatalf("items len=%d", len(items))
	}

	if items[0].MediaType != "video" {
		t.Fatalf("video mediaType=%q", items[0].MediaType)
	}
	if items[0].ImageCount != 0 {
		t.Fatalf("video imageCount=%d", items[0].ImageCount)
	}
	if items[0].VideoDuration < 11.9 || items[0].VideoDuration > 12.1 {
		t.Fatalf("video duration=%v", items[0].VideoDuration)
	}

	if items[1].MediaType != "imageAlbum" {
		t.Fatalf("album mediaType=%q", items[1].MediaType)
	}
	if items[1].ImageCount != 2 {
		t.Fatalf("album imageCount=%d", items[1].ImageCount)
	}
	if items[1].IsLivePhoto {
		t.Fatalf("album should not be live photo")
	}

	if items[2].MediaType != "livePhoto" {
		t.Fatalf("live mediaType=%q", items[2].MediaType)
	}
	if !items[2].IsLivePhoto {
		t.Fatalf("live should be marked as live photo")
	}
	if items[2].ImageCount != 2 {
		t.Fatalf("live imageCount=%d", items[2].ImageCount)
	}
	if items[2].LivePhotoPairs != 2 {
		t.Fatalf("live pairs=%d", items[2].LivePhotoPairs)
	}
}
