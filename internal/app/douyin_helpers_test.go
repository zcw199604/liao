package app

import (
	"net/http"
	"strings"
	"testing"
)

func TestDouyinScalarHelpers(t *testing.T) {
	if got := asInt64(float64(12)); got != 12 {
		t.Fatalf("asInt64 float=%d", got)
	}
	if got := asInt64(13); got != 13 {
		t.Fatalf("asInt64 int=%d", got)
	}
	if got := asInt64(int64(14)); got != 14 {
		t.Fatalf("asInt64 int64=%d", got)
	}
	if got := asInt64(" 15 "); got != 15 {
		t.Fatalf("asInt64 string=%d", got)
	}
	if got := asInt64("bad"); got != 0 {
		t.Fatalf("asInt64 bad=%d", got)
	}

	if got := asInt64Ptr(0); got != nil {
		t.Fatalf("want nil, got=%v", got)
	}
	if got := asInt64Ptr("2"); got == nil || *got != 2 {
		t.Fatalf("want 2, got=%v", got)
	}

	if got := formatUnixTimestampISO(0); got != "" {
		t.Fatalf("want empty, got=%q", got)
	}
	if got := formatUnixTimestampISO(1700000000); got == "" {
		t.Fatalf("want non-empty for seconds")
	}
	if got := formatUnixTimestampISO(1700000000000); got == "" {
		t.Fatalf("want non-empty for milliseconds")
	}
}

func TestDouyinAccountExtractors(t *testing.T) {
	if got := extractDouyinAccountPublishAt(nil); got != "" {
		t.Fatalf("publish=nil got=%q", got)
	}
	if got := extractDouyinAccountPublishAt(map[string]any{"publish_time": int64(1700000000000)}); got == "" {
		t.Fatalf("publish should parse ms timestamp")
	}

	pinned, rank, pinnedAt := extractDouyinAccountPinned(map[string]any{
		"is_top":   true,
		"top_rank": 0,
		"top_time": int64(1700000000),
	})
	if !pinned || rank == nil || *rank != 0 || pinnedAt == "" {
		t.Fatalf("pinned=%v rank=%v pinnedAt=%q", pinned, rank, pinnedAt)
	}

	if got := extractDouyinAccountStatus(map[string]any{"status": "  blocked  "}); got != "blocked" {
		t.Fatalf("status explicit got=%q", got)
	}
	if got := extractDouyinAccountStatus(map[string]any{"is_delete": true}); got != "deleted" {
		t.Fatalf("status deleted got=%q", got)
	}
	if got := extractDouyinAccountStatus(map[string]any{"private": true}); got != "private" {
		t.Fatalf("status private got=%q", got)
	}
	if got := extractDouyinAccountStatus(nil); got != "normal" {
		t.Fatalf("status nil got=%q", got)
	}

	uid, name := extractDouyinAccountAuthorMeta(map[string]any{
		"author": map[string]any{
			"unique_id": "u-1",
			"nickname":  "n-1",
		},
	})
	if uid != "u-1" || name != "n-1" {
		t.Fatalf("author meta uid=%q name=%q", uid, name)
	}

	cover := extractDouyinAccountCoverURL(map[string]any{
		"video": map[string]any{
			"cover": map[string]any{
				"url_list": []any{"https://img.example.com/c.jpg", "https://img.example.com/c.webp"},
			},
		},
	})
	if cover != "https://img.example.com/c.webp" {
		t.Fatalf("cover=%q", cover)
	}
}

func TestDouyinURLPickHelpers(t *testing.T) {
	if got := firstStringFromURLList([]any{" a ", "b"}); got != "a" {
		t.Fatalf("first=%q", got)
	}

	picked := pickPreferredURLFromSlice([]string{"http://x/a.jpg", "http://x/a.webp"}, []string{".webp", ".jpg"})
	if picked != "http://x/a.webp" {
		t.Fatalf("picked=%q", picked)
	}
	picked = pickPreferredDouyinImageURL(map[string]any{
		"download_url_list": []any{"http://x/b.jpeg"},
	}, false)
	if picked != "http://x/b.jpeg" {
		t.Fatalf("picked image=%q", picked)
	}

	video := extractDouyinAccountVideoPlayURL(map[string]any{
		"video": map[string]any{
			"play_addr": map[string]any{"url_list": []any{"https://v.example.com/1.mp4"}},
		},
	})
	if video != "https://v.example.com/1.mp4" {
		t.Fatalf("video=%q", video)
	}

	liveVideos := extractDouyinAccountLivePhotoVideoPlayURLs(map[string]any{
		"images": []any{
			map[string]any{"video": map[string]any{"play_addr": map[string]any{"url_list": []any{"https://v.example.com/2.mp4"}}}},
			map[string]any{"video": map[string]any{"play_addr": map[string]any{"url_list": []any{"https://v.example.com/2.mp4"}}}},
			map[string]any{"video": map[string]any{"play_addr": map[string]any{"url_list": []any{"https://i.example.com/2.jpg"}}}},
		},
	})
	if len(liveVideos) != 1 || liveVideos[0] != "https://v.example.com/2.mp4" {
		t.Fatalf("liveVideos=%v", liveVideos)
	}

	images := extractDouyinAccountImageURLs(map[string]any{
		"images": []any{
			map[string]any{"url_list": []any{"https://i.example.com/a.jpg", "https://i.example.com/a.webp"}},
			map[string]any{"download_url_list": []any{"https://i.example.com/b.jpeg"}},
		},
	}, true)
	if len(images) != 2 || images[0] != "https://i.example.com/a.webp" || images[1] != "https://i.example.com/b.jpeg" {
		t.Fatalf("images=%v", images)
	}

	flat := extractDouyinAccountFlatDownloads(map[string]any{"download": " https://v.example.com/s.mp4 "})
	if len(flat) != 1 || flat[0] != "https://v.example.com/s.mp4" {
		t.Fatalf("flat=%v", flat)
	}
}

func TestDouyinUserMetaHelpers(t *testing.T) {
	avatar := extractDouyinAvatarURL(map[string]any{
		"avatar_larger": map[string]any{"url_list": []any{"https://img.example.com/avatar.jpg"}},
	})
	if avatar != "https://img.example.com/avatar.jpg" {
		t.Fatalf("avatar=%q", avatar)
	}
	if got := extractDouyinAvatarURL(map[string]any{"avatar_url": " https://img.example.com/a2.jpg "}); got != "https://img.example.com/a2.jpg" {
		t.Fatalf("avatar flat=%q", got)
	}

	if got := extractDouyinDisplayName(map[string]any{"nickname": "  Alice "}); got != "Alice" {
		t.Fatalf("display=%q", got)
	}
	if got := extractDouyinSignature(map[string]any{"bio": "  hello  "}); got != "hello" {
		t.Fatalf("signature=%q", got)
	}

	statsMap := map[string]any{
		"statistics": map[string]any{
			"follower_count":  10,
			"following_count": int64(20),
			"aweme_count":     "30",
			"liked_count":     float64(40),
		},
	}
	follower, following, aweme, favorited := extractDouyinUserStats(statsMap)
	if follower == nil || following == nil || aweme == nil || favorited == nil {
		t.Fatalf("stats nil follower=%v following=%v aweme=%v favorited=%v", follower, following, aweme, favorited)
	}
	if *follower != 10 || *following != 20 || *aweme != 30 || *favorited != 40 {
		t.Fatalf("stats values follower=%v following=%v aweme=%v favorited=%v", *follower, *following, *aweme, *favorited)
	}

	picked := pickInt64Ptr(map[string]any{"x": 0, "y": 5}, []string{"x", "y"})
	if picked == nil || *picked != 5 {
		t.Fatalf("pickInt64Ptr=%v", picked)
	}

	displayName, signature, avatarURL, profileURL, fc, fg, ac, tf := extractDouyinAccountUserMeta("MS4wLjABAAAA_x", map[string]any{
		"author": map[string]any{
			"nickname":      "A",
			"signature":     "S",
			"avatar_url":    "https://img.example.com/a.jpg",
			"followerCount": 1,
			"statistics": map[string]any{
				"followingCount": 2,
				"awemeCount":     3,
				"totalFavorited": 4,
			},
		},
	})
	if displayName != "A" || signature != "S" || avatarURL == "" || !strings.Contains(profileURL, "/user/") {
		t.Fatalf("meta display=%q signature=%q avatar=%q profile=%q", displayName, signature, avatarURL, profileURL)
	}
	if fc == nil || fg == nil || ac == nil || tf == nil || *fc != 1 || *fg != 2 || *ac != 3 || *tf != 4 {
		t.Fatalf("meta stats fc=%v fg=%v ac=%v tf=%v", fc, fg, ac, tf)
	}
}

func TestDouyinHTTPHelpers(t *testing.T) {
	if got := effectiveDouyinUserAgent(nil); got == "" {
		t.Fatalf("default ua empty")
	}
	req := &http.Request{Header: http.Header{"User-Agent": []string{" custom-ua "}}}
	if got := effectiveDouyinUserAgent(req); got != "custom-ua" {
		t.Fatalf("ua=%q", got)
	}

	if got := truncateForLog("  abcdef  ", 3); got != "abc..." {
		t.Fatalf("truncate=%q", got)
	}
	if got := truncateForLog(" abc ", 10); got != "abc" {
		t.Fatalf("truncate short=%q", got)
	}

	if got := douyinRefererForDetail("123", "video"); got != "https://www.douyin.com/video/123" {
		t.Fatalf("referer video=%q", got)
	}
	if got := douyinRefererForDetail("123", "image"); got != "https://www.douyin.com/note/123" {
		t.Fatalf("referer image=%q", got)
	}
	if got := douyinRefererForDetail("", "video"); got != douyinDefaultReferer {
		t.Fatalf("referer fallback=%q", got)
	}

	if !isDouyinHost("www.douyin.com") {
		t.Fatalf("expected douyin host")
	}
	if !isDouyinHost("www.douyin.com:443") {
		t.Fatalf("expected douyin host with port")
	}
	if !isDouyinHost("img.douyinpic.com") {
		t.Fatalf("expected douyinpic host")
	}
	if isDouyinHost("example.com") {
		t.Fatalf("unexpected non-douyin host")
	}

	info := buildURLLogInfo("https://www.douyin.com/video/123?x=1&y=2")
	if info.Scheme != "https" || info.Host != "www.douyin.com" || info.Path != "/video/123" || info.QueryKeys != 2 || info.Hash == "" {
		t.Fatalf("info=%+v", info)
	}
}
