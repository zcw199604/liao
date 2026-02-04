package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type douyinDetailRequest struct {
	Input  string `json:"input"`
	Cookie string `json:"cookie,omitempty"`
	Proxy  string `json:"proxy,omitempty"`
}

type douyinAccountRequest struct {
	Input  string `json:"input"`
	Cookie string `json:"cookie,omitempty"`
	Tab    string `json:"tab,omitempty"`    // post|favorite
	Cursor int    `json:"cursor,omitempty"` // 游标
	Count  int    `json:"count,omitempty"`  // 每页数量（>0）
	Proxy  string `json:"proxy,omitempty"`  // 可选（前端不暴露）
}

type douyinAccountItem struct {
	DetailID         string            `json:"detailId"`
	Type             string            `json:"type,omitempty"` // image|video（best-effort）
	Desc             string            `json:"desc,omitempty"`
	CoverURL         string            `json:"coverUrl,omitempty"`
	CoverDownloadURL string            `json:"coverDownloadUrl,omitempty"`
	IsPinned         bool              `json:"isPinned,omitempty"`
	PinnedRank       *int              `json:"pinnedRank,omitempty"`
	PinnedAt         string            `json:"pinnedAt,omitempty"`
	PublishAt        string            `json:"publishAt,omitempty"`
	CrawledAt        string            `json:"crawledAt,omitempty"`
	LastSeenAt       string            `json:"lastSeenAt,omitempty"`
	Status           string            `json:"status,omitempty"`
	AuthorUniqueID   string            `json:"authorUniqueId,omitempty"`
	AuthorName       string            `json:"authorName,omitempty"`
	Key              string            `json:"key,omitempty"`
	Items            []douyinMediaItem `json:"items,omitempty"` // 预览/下载列表（best-effort；为空时前端可回退到 /api/douyin/detail）
}

type douyinAccountResponse struct {
	SecUserID      string              `json:"secUserId"`
	DisplayName    string              `json:"displayName,omitempty"`
	Signature      string              `json:"signature,omitempty"`
	AvatarURL      string              `json:"avatarUrl,omitempty"`
	ProfileURL     string              `json:"profileUrl,omitempty"`
	FollowerCount  *int64              `json:"followerCount,omitempty"`
	FollowingCount *int64              `json:"followingCount,omitempty"`
	AwemeCount     *int64              `json:"awemeCount,omitempty"`
	TotalFavorited *int64              `json:"totalFavorited,omitempty"`
	Tab            string              `json:"tab"`
	Cursor         int                 `json:"cursor"`
	HasMore        bool                `json:"hasMore"`
	Items          []douyinAccountItem `json:"items"`
}

type douyinDetailResponse struct {
	Key      string            `json:"key"`
	DetailID string            `json:"detailId"`
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	CoverURL string            `json:"coverUrl,omitempty"`
	Items    []douyinMediaItem `json:"items"`
}

type douyinMediaItem struct {
	Index            int    `json:"index"`
	Type             string `json:"type"` // image|video
	URL              string `json:"url"`
	DownloadURL      string `json:"downloadUrl"`
	OriginalFilename string `json:"originalFilename,omitempty"`
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		return s == "1" || s == "true" || s == "yes"
	default:
		return false
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(t))
		return i
	default:
		return 0
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return i
	default:
		return int64(asInt(v))
	}
}

func asInt64Ptr(v any) *int64 {
	n := int64(asInt(v))
	if n <= 0 {
		return nil
	}
	return &n
}

func formatUnixTimestampISO(v int64) string {
	if v <= 0 {
		return ""
	}
	// Handle milliseconds timestamps (common in some upstream variants).
	if v > 1_000_000_000_000 {
		v = v / 1000
	}
	if v <= 0 {
		return ""
	}
	return formatLocalDateTimeISO(time.Unix(v, 0).In(time.Local))
}

func extractDouyinAccountPublishAt(item map[string]any) string {
	if item == nil {
		return ""
	}
	keys := []string{
		"create_time",
		"createTime",
		"publish_time",
		"publishTime",
		"published_at",
		"publishedAt",
	}
	for _, k := range keys {
		if ts := asInt64(item[k]); ts > 0 {
			if s := formatUnixTimestampISO(ts); s != "" {
				return s
			}
		}
	}
	return ""
}

func extractDouyinAccountPinned(item map[string]any) (bool, *int, string) {
	if item == nil {
		return false, nil, ""
	}

	isPinned := false
	pinnedKeys := []string{"is_top", "isTop", "is_pinned", "isPinned", "top", "pinned"}
	for _, k := range pinnedKeys {
		if item[k] == nil {
			continue
		}
		isPinned = asBool(item[k])
		break
	}

	var rank *int
	rankKeys := []string{"top_rank", "topRank", "pinned_rank", "pinnedRank", "top_index", "topIndex", "pin_index", "pinIndex"}
	for _, k := range rankKeys {
		n := asInt(item[k])
		// Rank can be 0-based in some upstreams; treat any non-negative as valid.
		if item[k] != nil && n >= 0 {
			v := n
			rank = &v
			break
		}
	}

	pinnedAt := ""
	timeKeys := []string{"top_time", "topTime", "pinned_time", "pinnedTime", "pin_time", "pinTime"}
	for _, k := range timeKeys {
		if ts := asInt64(item[k]); ts > 0 {
			pinnedAt = formatUnixTimestampISO(ts)
			if pinnedAt != "" {
				break
			}
		}
	}

	return isPinned, rank, pinnedAt
}

func extractDouyinAccountStatus(item map[string]any) string {
	if item == nil {
		return "normal"
	}
	if v := strings.TrimSpace(asString(item["status"])); v != "" {
		return v
	}
	if asBool(item["is_delete"]) || asBool(item["isDelete"]) || asBool(item["deleted"]) {
		return "deleted"
	}
	if asBool(item["is_private"]) || asBool(item["isPrivate"]) || asBool(item["private"]) {
		return "private"
	}
	return "normal"
}

func extractDouyinAccountAuthorMeta(item map[string]any) (uniqueID string, name string) {
	if item == nil {
		return "", ""
	}

	candidates := []any{
		item["author"],
		item["author_info"],
		item["authorInfo"],
		item["user"],
		item["user_info"],
		item["userInfo"],
	}
	for _, raw := range candidates {
		m, ok := raw.(map[string]any)
		if !ok || m == nil {
			continue
		}
		if name == "" {
			nameKeys := []string{"nickname", "nick_name", "display_name", "displayName", "name"}
			for _, k := range nameKeys {
				if v := strings.TrimSpace(asString(m[k])); v != "" {
					name = v
					break
				}
			}
		}
		if uniqueID == "" {
			idKeys := []string{"unique_id", "uniqueId", "short_id", "shortId", "douyin_id", "douyinId"}
			for _, k := range idKeys {
				if v := strings.TrimSpace(asString(m[k])); v != "" {
					uniqueID = v
					break
				}
			}
		}
		if uniqueID != "" && name != "" {
			break
		}
	}

	return uniqueID, name
}

func extractDouyinAccountCoverURL(item map[string]any) string {
	// 1) video.cover.url_list[0]
	if v, ok := item["video"].(map[string]any); ok {
		if cover, ok := v["cover"].(map[string]any); ok {
			preferredExts := preferredDouyinImageExts(true)
			urls := append(extractStringSlice(cover["url_list"]), extractStringSlice(cover["urlList"])...)
			if u := pickPreferredURLFromSlice(urls, preferredExts); strings.TrimSpace(u) != "" {
				return u
			}
		}
	}

	// 2) images[0].url_list[0]
	if imgs, ok := item["images"].([]any); ok && len(imgs) > 0 {
		if img0, ok := imgs[0].(map[string]any); ok {
			if u := strings.TrimSpace(pickPreferredDouyinImageURL(img0, true)); u != "" {
				return u
			}
		}
	}

	// 3) 兼容上游扁平字段（部分版本可能直接返回 static/dynamic cover）
	flatKeys := []string{
		"static_cover",
		"staticCover",
		"dynamic_cover",
		"dynamicCover",
		"cover_url",
		"coverUrl",
	}
	for _, k := range flatKeys {
		if u := strings.TrimSpace(asString(item[k])); u != "" {
			return u
		}
	}

	return ""
}

func firstStringFromURLList(v any) string {
	list := extractStringSlice(v)
	if len(list) == 0 {
		return ""
	}
	return strings.TrimSpace(list[0])
}

func preferredDouyinImageExts(preferWebP bool) []string {
	if preferWebP {
		return []string{".webp", ".jpeg", ".jpg"}
	}
	return []string{".jpeg", ".jpg", ".webp"}
}

func pickPreferredURLFromSlice(urls []string, preferredExts []string) string {
	if len(urls) == 0 {
		return ""
	}

	for _, ext := range preferredExts {
		for _, raw := range urls {
			u := strings.TrimSpace(raw)
			if u == "" {
				continue
			}
			if guessExtFromURL(u) == ext {
				return u
			}
		}
	}

	for _, raw := range urls {
		if u := strings.TrimSpace(raw); u != "" {
			return u
		}
	}
	return ""
}

func pickPreferredDouyinImageURL(image map[string]any, preferWebP bool) string {
	if image == nil {
		return ""
	}

	preferredExts := preferredDouyinImageExts(preferWebP)

	primary := append(extractStringSlice(image["url_list"]), extractStringSlice(image["urlList"])...)
	if u := pickPreferredURLFromSlice(primary, preferredExts); strings.TrimSpace(u) != "" {
		return u
	}

	secondary := append(extractStringSlice(image["download_url_list"]), extractStringSlice(image["downloadUrlList"])...)
	return pickPreferredURLFromSlice(secondary, preferredExts)
}

func extractDouyinAccountVideoPlayURL(item map[string]any) string {
	v, ok := item["video"].(map[string]any)
	if !ok || v == nil {
		return ""
	}

	return extractDouyinVideoPlayURLFromVideoMap(v)
}

func extractDouyinVideoPlayURLFromVideoMap(v map[string]any) string {
	if v == nil {
		return ""
	}

	addrKeys := []string{
		"play_addr",
		"playAddr",
		"play_addr_h264",
		"playAddrH264",
		"play_addr_bytevc1",
		"playAddrBytevc1",
		"play_addr_lowbr",
		"playAddrLowbr",
		"download_addr",
		"downloadAddr",
	}

	for _, k := range addrKeys {
		raw := v[k]
		if raw == nil {
			continue
		}
		if m, ok := raw.(map[string]any); ok && m != nil {
			if u := strings.TrimSpace(firstStringFromURLList(m["url_list"])); u != "" {
				return u
			}
			if u := strings.TrimSpace(firstStringFromURLList(m["urlList"])); u != "" {
				return u
			}
			if u := strings.TrimSpace(asString(m["url"])); u != "" {
				return u
			}
		}
		if u := strings.TrimSpace(asString(raw)); u != "" {
			return u
		}
	}

	return ""
}

func extractDouyinAccountLivePhotoVideoPlayURLs(item map[string]any) []string {
	imgsAny, ok := item["images"].([]any)
	if !ok || len(imgsAny) == 0 {
		return nil
	}

	out := make([]string, 0, len(imgsAny))
	for _, it := range imgsAny {
		m, ok := it.(map[string]any)
		if !ok || m == nil {
			continue
		}
		v, ok := m["video"].(map[string]any)
		if !ok || v == nil {
			continue
		}
		u := strings.TrimSpace(extractDouyinVideoPlayURLFromVideoMap(v))
		if u == "" || guessDouyinMediaTypeFromURL(u) != "video" {
			continue
		}
		has := false
		for _, existed := range out {
			if strings.TrimSpace(existed) == u {
				has = true
				break
			}
		}
		if !has {
			out = append(out, u)
		}
	}
	return out
}

func extractDouyinAccountImageURLs(item map[string]any, preferWebP bool) []string {
	imgsAny, ok := item["images"].([]any)
	if !ok || len(imgsAny) == 0 {
		return nil
	}

	out := make([]string, 0, len(imgsAny))
	for _, it := range imgsAny {
		m, ok := it.(map[string]any)
		if !ok || m == nil {
			continue
		}

		if u := strings.TrimSpace(pickPreferredDouyinImageURL(m, preferWebP)); u != "" {
			out = append(out, u)
			continue
		}
	}

	return out
}

func extractDouyinAccountFlatDownloads(item map[string]any) []string {
	if item == nil {
		return nil
	}
	// 兼容上游扁平字段：downloads 可能是 string 或 []string/[]any
	candidates := []any{item["downloads"], item["download"], item["download_url"], item["downloadUrl"]}
	for _, raw := range candidates {
		list := extractStringSlice(raw)
		if len(list) > 0 {
			return list
		}
	}
	return nil
}

func guessDouyinMediaTypeFromURL(raw string) string {
	u := strings.ToLower(strings.TrimSpace(raw))
	if u == "" {
		return "image"
	}

	switch strings.ToLower(guessExtFromURL(u)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".bmp":
		return "image"
	case ".mp4", ".m3u8", ".mov", ".webm", ".m4v":
		return "video"
	}

	// 抖音播放直链常见形态：/aweme/v1/play/?video_id=...
	if strings.Contains(u, "/aweme/v1/play/") || strings.Contains(u, "video_id=") {
		return "video"
	}
	// 部分直链无扩展名，但域名/路径能体现视频属性（best-effort）
	if strings.Contains(u, "douyinvod") || strings.Contains(u, "bytevod") || strings.Contains(u, "/video/") {
		return "video"
	}
	return "image"
}

func extractDouyinAvatarURL(user map[string]any) string {
	if user == nil {
		return ""
	}

	avatarKeys := []string{
		"avatar_larger",
		"avatarLarger",
		"avatar_thumb",
		"avatarThumb",
		"avatar_medium",
		"avatarMedium",
		"avatar",
	}

	for _, k := range avatarKeys {
		raw := user[k]
		if raw == nil {
			continue
		}

		if m, ok := raw.(map[string]any); ok && m != nil {
			if u := strings.TrimSpace(firstStringFromURLList(m["url_list"])); u != "" {
				return u
			}
			if u := strings.TrimSpace(firstStringFromURLList(m["urlList"])); u != "" {
				return u
			}
			if u := strings.TrimSpace(asString(m["url"])); u != "" {
				return u
			}
		}

		if u := strings.TrimSpace(asString(raw)); u != "" {
			return u
		}
	}

	flatKeys := []string{"avatar_url", "avatarUrl", "avatarURL"}
	for _, k := range flatKeys {
		if u := strings.TrimSpace(asString(user[k])); u != "" {
			return u
		}
	}

	return ""
}

func extractDouyinDisplayName(user map[string]any) string {
	if user == nil {
		return ""
	}

	candidates := []string{"nickname", "nick_name", "nickName", "name", "user_name", "userName"}
	for _, k := range candidates {
		if v := strings.TrimSpace(asString(user[k])); v != "" {
			return v
		}
	}
	return ""
}

func extractDouyinSignature(user map[string]any) string {
	if user == nil {
		return ""
	}

	candidates := []string{
		"signature",
		"user_signature",
		"userSignature",
		"bio",
		"description",
	}
	for _, k := range candidates {
		if v := strings.TrimSpace(asString(user[k])); v != "" {
			return v
		}
	}
	return ""
}

func pickInt64Ptr(m map[string]any, keys []string) *int64 {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if v := asInt64Ptr(m[k]); v != nil {
			return v
		}
	}
	return nil
}

func extractDouyinUserStats(user map[string]any) (followerCount, followingCount, awemeCount, totalFavorited *int64) {
	if user == nil {
		return nil, nil, nil, nil
	}

	followerKeys := []string{"follower_count", "followerCount", "fans_count", "fansCount"}
	followingKeys := []string{"following_count", "followingCount", "followings_count", "followingsCount"}
	awemeKeys := []string{"aweme_count", "awemeCount", "work_count", "workCount", "video_count", "videoCount"}
	favorKeys := []string{"total_favorited", "totalFavorited", "total_favorite", "totalFavorite", "liked_count", "likedCount", "favorited_count", "favoritedCount"}

	followerCount = pickInt64Ptr(user, followerKeys)
	followingCount = pickInt64Ptr(user, followingKeys)
	awemeCount = pickInt64Ptr(user, awemeKeys)
	totalFavorited = pickInt64Ptr(user, favorKeys)

	statsKeys := []string{"statistics", "statistic", "stats", "user_statistics", "userStats"}
	for _, k := range statsKeys {
		raw := user[k]
		m, ok := raw.(map[string]any)
		if !ok || m == nil {
			continue
		}
		if followerCount == nil {
			followerCount = pickInt64Ptr(m, followerKeys)
		}
		if followingCount == nil {
			followingCount = pickInt64Ptr(m, followingKeys)
		}
		if awemeCount == nil {
			awemeCount = pickInt64Ptr(m, awemeKeys)
		}
		if totalFavorited == nil {
			totalFavorited = pickInt64Ptr(m, favorKeys)
		}
	}

	return followerCount, followingCount, awemeCount, totalFavorited
}

func extractDouyinAccountUserMeta(secUserID string, data map[string]any) (displayName, signature, avatarURL, profileURL string, followerCount, followingCount, awemeCount, totalFavorited *int64) {
	secUserID = strings.TrimSpace(secUserID)
	if secUserID != "" {
		profileURL = "https://www.douyin.com/user/" + url.PathEscape(secUserID)
	}
	if data == nil {
		return "", "", "", profileURL, nil, nil, nil, nil
	}

	updateFrom := func(m map[string]any) {
		if m == nil {
			return
		}
		if displayName == "" {
			displayName = extractDouyinDisplayName(m)
		}
		if signature == "" {
			signature = extractDouyinSignature(m)
		}
		if avatarURL == "" {
			avatarURL = extractDouyinAvatarURL(m)
		}

		fc, fg, ac, tf := extractDouyinUserStats(m)
		if followerCount == nil {
			followerCount = fc
		}
		if followingCount == nil {
			followingCount = fg
		}
		if awemeCount == nil {
			awemeCount = ac
		}
		if totalFavorited == nil {
			totalFavorited = tf
		}
	}

	userCandidates := []any{
		data["user"],
		data["user_info"],
		data["userInfo"],
		data["author"],
		data["account"],
		data["profile"],
	}
	for _, raw := range userCandidates {
		m, ok := raw.(map[string]any)
		if !ok || m == nil {
			continue
		}
		updateFrom(m)
	}

	// Fallback: 从第一条作品的 author 提取昵称/头像/简介/统计（best-effort）
	listAny := data["aweme_list"]
	if listAny == nil {
		listAny = data["awemeList"]
	}
	if list, ok := listAny.([]any); ok && len(list) > 0 {
		if aweme0, ok := list[0].(map[string]any); ok && aweme0 != nil {
			authorCandidates := []any{
				aweme0["author"],
				aweme0["authorInfo"],
				aweme0["user"],
				aweme0["user_info"],
			}
			for _, raw := range authorCandidates {
				m, ok := raw.(map[string]any)
				if !ok || m == nil {
					continue
				}
				updateFrom(m)
			}
		}
	}

	return displayName, signature, avatarURL, profileURL, followerCount, followingCount, awemeCount, totalFavorited
}

func extractDouyinAccountItems(s *DouyinDownloaderService, secUserID string, data map[string]any) []douyinAccountItem {
	if data == nil {
		return nil
	}

	listAny := data["aweme_list"]
	if listAny == nil {
		listAny = data["awemeList"]
	}
	list, ok := listAny.([]any)
	if !ok || len(list) == 0 {
		return nil
	}

	items := make([]douyinAccountItem, 0, len(list))
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok || m == nil {
			continue
		}

		id := strings.TrimSpace(asString(m["aweme_id"]))
		if id == "" {
			id = strings.TrimSpace(asString(m["awemeId"]))
		}
		if id == "" {
			id = strings.TrimSpace(asString(m["id"]))
		}
		if id == "" {
			continue
		}

		typeLabel := strings.TrimSpace(asString(m["type"])) // 视频/图集/实况（部分上游版本）

		itemType := "video"
		if imgs, ok := m["images"].([]any); ok && len(imgs) > 0 {
			itemType = "image"
		} else if typeLabel != "" && (strings.Contains(typeLabel, "图集") || strings.Contains(typeLabel, "实况") || strings.Contains(typeLabel, "图片")) {
			itemType = "image"
		}

		desc := strings.TrimSpace(asString(m["desc"]))
		cover := extractDouyinAccountCoverURL(m)
		isPinned, pinnedRank, pinnedAt := extractDouyinAccountPinned(m)
		publishAt := extractDouyinAccountPublishAt(m)
		status := extractDouyinAccountStatus(m)
		authorUniqueID, authorName := extractDouyinAccountAuthorMeta(m)

		// best-effort：直接从 account 返回中抽取可预览资源，避免点击后再 /detail N 次。
		downloads := []string(nil)
		if itemType == "image" {
			nestedVideos := extractDouyinAccountLivePhotoVideoPlayURLs(m)
			isLivePhoto := (typeLabel != "" && strings.Contains(typeLabel, "实况")) || len(nestedVideos) > 0

			downloads = extractDouyinAccountImageURLs(m, !isLivePhoto)

			if len(nestedVideos) > 0 {
				for _, u := range nestedVideos {
					has := false
					for _, existed := range downloads {
						if strings.TrimSpace(existed) == strings.TrimSpace(u) {
							has = true
							break
						}
					}
					if !has {
						downloads = append(downloads, u)
					}
				}
			}

			if isLivePhoto {
				if typeLabel == "" {
					typeLabel = "实况"
				}
				if u := extractDouyinAccountVideoPlayURL(m); u != "" {
					has := false
					for _, existed := range downloads {
						if strings.TrimSpace(existed) == strings.TrimSpace(u) {
							has = true
							break
						}
					}
					if !has {
						downloads = append(downloads, u)
					}
				}
			}
		} else {
			if u := extractDouyinAccountVideoPlayURL(m); u != "" {
				downloads = []string{u}
			}
		}
		if len(downloads) == 0 {
			downloads = extractDouyinAccountFlatDownloads(m)
		}
		if strings.TrimSpace(cover) == "" && len(downloads) > 0 {
			for _, u := range downloads {
				if guessDouyinMediaTypeFromURL(u) == "image" {
					cover = u
					break
				}
			}
		}

		// Top-level type best-effort（用于列表元数据；预览项的 type 会按 URL 逐个判断）
		displayType := itemType
		if len(downloads) > 0 {
			hasImage := false
			hasVideo := false
			for _, u := range downloads {
				if guessDouyinMediaTypeFromURL(u) == "video" {
					hasVideo = true
				} else {
					hasImage = true
				}
			}
			if hasImage && !hasVideo {
				displayType = "image"
			} else if hasVideo && !hasImage {
				displayType = "video"
			} else if hasImage && hasVideo {
				displayType = "video"
			}
		}

		key := ""
		var previewItems []douyinMediaItem
		coverDownloadURL := ""
		if s != nil && len(downloads) > 0 {
			cached := &douyinCachedDetail{
				SecUserID: secUserID,
				DetailID:  id,
				Title:     desc,
				Type:      defaultString(typeLabel, map[string]string{"video": "视频", "image": "图集"}[displayType]),
				CoverURL:  cover,
				Downloads: downloads,
			}
			key = s.CacheDetail(cached)
			if key != "" {
				if strings.TrimSpace(cover) != "" {
					coverDownloadURL = fmt.Sprintf("/api/douyin/cover?key=%s", url.QueryEscape(key))
				}

				previewItems = make([]douyinMediaItem, 0, len(downloads))
				for i, u := range downloads {
					u = strings.TrimSpace(u)

					mediaType := guessDouyinMediaTypeFromURL(u)
					ext := guessExtFromURL(u)
					if ext == "" {
						if mediaType == "video" {
							ext = ".mp4"
						} else {
							ext = ".jpg"
						}
					}

					previewItems = append(previewItems, douyinMediaItem{
						Index:            i,
						Type:             mediaType,
						URL:              u,
						DownloadURL:      fmt.Sprintf("/api/douyin/download?key=%s&index=%d", url.QueryEscape(key), i),
						OriginalFilename: buildDouyinOriginalFilename(desc, id, i, len(downloads), ext),
					})
				}
			}
		}

		items = append(items, douyinAccountItem{
			DetailID:         id,
			Type:             displayType,
			Desc:             desc,
			CoverURL:         cover,
			IsPinned:         isPinned,
			PinnedRank:       pinnedRank,
			PinnedAt:         pinnedAt,
			PublishAt:        publishAt,
			Status:           status,
			AuthorUniqueID:   authorUniqueID,
			AuthorName:       authorName,
			Key:              key,
			Items:            previewItems,
			CoverDownloadURL: coverDownloadURL,
		})
	}
	return items
}

func (a *App) handleDouyinAccount(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	var req douyinAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	input := strings.TrimSpace(req.Input)
	if input == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "input 不能为空"})
		return
	}

	tab := strings.TrimSpace(req.Tab)
	if tab == "" {
		tab = "post"
	}
	cursor := req.Cursor
	if cursor < 0 {
		cursor = 0
	}
	count := req.Count
	if count <= 0 {
		count = 18
	}

	secUserID, resolvedURL, err := a.douyinDownloader.ResolveSecUserID(r.Context(), input, req.Proxy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	data, err := a.douyinDownloader.FetchAccount(r.Context(), secUserID, tab, req.Cookie, req.Proxy, cursor, count)
	if err != nil {
		msg := err.Error()
		if resolvedURL != "" {
			msg = msg + "（resolved=" + resolvedURL + "）"
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": msg})
		return
	}

	displayName, signature, avatarURL, profileURL, followerCount, followingCount, awemeCount, totalFavorited := extractDouyinAccountUserMeta(secUserID, data)

	nextCursor := asInt(data["cursor"])
	if nextCursor == 0 {
		if v := asInt(data["max_cursor"]); v > 0 {
			nextCursor = v
		}
	}

	hasMore := asBool(data["has_more"])
	if !hasMore {
		hasMore = asBool(data["hasMore"])
	}

	items := extractDouyinAccountItems(a.douyinDownloader, secUserID, data)
	if items == nil {
		items = []douyinAccountItem{}
	}

	writeJSON(w, http.StatusOK, douyinAccountResponse{
		SecUserID:      secUserID,
		DisplayName:    displayName,
		Signature:      signature,
		AvatarURL:      avatarURL,
		ProfileURL:     profileURL,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		AwemeCount:     awemeCount,
		TotalFavorited: totalFavorited,
		Tab:            tab,
		Cursor:         nextCursor,
		HasMore:        hasMore,
		Items:          items,
	})
}

func (a *App) handleDouyinDetail(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	var req douyinDetailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	input := strings.TrimSpace(req.Input)
	if input == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "input 不能为空"})
		return
	}

	detailID, resolvedURL, err := a.douyinDownloader.ResolveDetailID(r.Context(), input, req.Proxy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	detail, err := a.douyinDownloader.FetchDetail(r.Context(), detailID, req.Cookie, req.Proxy)
	if err != nil {
		msg := err.Error()
		if resolvedURL != "" {
			msg = msg + "（resolved=" + resolvedURL + "）"
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": msg})
		return
	}

	key := a.douyinDownloader.CacheDetail(detail)
	if key == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "缓存失败"})
		return
	}

	defaultType := "video"
	if strings.Contains(detail.Type, "图集") || strings.Contains(detail.Type, "实况") || strings.Contains(detail.Type, "图片") {
		defaultType = "image"
	}

	items := make([]douyinMediaItem, 0, len(detail.Downloads))
	for i, u := range detail.Downloads {
		u = strings.TrimSpace(u)

		mediaType := defaultType
		switch strings.ToLower(guessExtFromURL(u)) {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".bmp":
			mediaType = "image"
		case ".mp4", ".m3u8", ".mov", ".webm", ".m4v":
			mediaType = "video"
		default:
			// 抖音播放直链常见形态：/aweme/v1/play/?video_id=...
			if strings.Contains(strings.ToLower(u), "/aweme/v1/play/") || strings.Contains(strings.ToLower(u), "video_id=") {
				mediaType = "video"
			}
		}

		ext := guessExtFromURL(u)
		if ext == "" {
			if mediaType == "video" {
				ext = ".mp4"
			} else {
				ext = ".jpg"
			}
		}

		items = append(items, douyinMediaItem{
			Index:            i,
			Type:             mediaType,
			URL:              u,
			DownloadURL:      fmt.Sprintf("/api/douyin/download?key=%s&index=%d", url.QueryEscape(key), i),
			OriginalFilename: buildDouyinOriginalFilename(detail.Title, detail.DetailID, i, len(detail.Downloads), ext),
		})
	}

	writeJSON(w, http.StatusOK, douyinDetailResponse{
		Key:      key,
		DetailID: detail.DetailID,
		Type:     detail.Type,
		Title:    detail.Title,
		CoverURL: detail.CoverURL,
		Items:    items,
	})
}

func (a *App) handleDouyinDownload(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	indexValue := strings.TrimSpace(r.URL.Query().Get("index"))
	index, err := strconv.Atoi(indexValue)
	if key == "" || indexValue == "" || err != nil || index < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key/index 非法"})
		return
	}

	cached, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || cached == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "解析已过期，请重新解析"})
		return
	}
	if index >= len(cached.Downloads) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "index 越界"})
		return
	}

	remoteURL := strings.TrimSpace(cached.Downloads[index])
	if remoteURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接为空"})
		return
	}

	if r.Method == http.MethodHead {
		a.handleDouyinDownloadHead(w, r, key, cached, index, remoteURL)
		return
	}

	rangeValue := strings.TrimSpace(r.Header.Get("Range"))

	doDownload := func(urlValue string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, urlValue, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://www.douyin.com/")
		if rangeValue != "" {
			req.Header.Set("Range", rangeValue)
		}
		return a.douyinDownloader.api.httpClient.Do(req)
	}

	resp, err := doDownload(remoteURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + err.Error()})
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusText := resp.Status
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()

		// 兼容：收藏用户作品列表中的 downloads 可能是历史快照，CDN 链接会过期导致 403。
		// 这里 best-effort 自动刷新一次（调用上游 /douyin/detail 获取新的 downloads），避免用户手动“重新解析”。
		if resp.StatusCode == http.StatusForbidden {
			refreshed, refreshErr := a.douyinDownloader.RefreshDetailBestEffort(cached.DetailID)
			if refreshErr == nil && refreshed != nil && index < len(refreshed.Downloads) {
				newURL := strings.TrimSpace(refreshed.Downloads[index])
				if newURL != "" {
					// Update cache under the same key so subsequent range requests reuse the refreshed URL set.
					if a.douyinDownloader.cache != nil {
						a.douyinDownloader.cache.Set(key, *refreshed)
					}

					remoteURL = newURL
					cached = refreshed

					resp2, err2 := doDownload(remoteURL)
					if err2 != nil {
						writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + err2.Error()})
						return
					}
					if resp2.StatusCode >= 200 && resp2.StatusCode < 300 {
						resp = resp2
						defer resp.Body.Close()
						goto writeBody
					}

					body2, _ := io.ReadAll(io.LimitReader(resp2.Body, 1024))
					_ = resp2.Body.Close()
					writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", resp2.Status, strings.TrimSpace(string(body2)))})
					return
				}
			}
		}

		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", statusText, strings.TrimSpace(string(body)))})
		return
	}

	defer resp.Body.Close()

writeBody:
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	ext := guessExtFromContentType(contentType)
	if ext == "" {
		ext = guessExtFromURL(remoteURL)
	}
	if ext == "" {
		ext = ".bin"
	}

	originalFilename := buildDouyinOriginalFilename(cached.Title, cached.DetailID, index, len(cached.Downloads), ext)
	fallback := buildDouyinFallbackFilename(cached.DetailID, index, len(cached.Downloads), ext)

	w.Header().Set("Cache-Control", "no-store")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", buildAttachmentContentDisposition(fallback, originalFilename))

	if ar := strings.TrimSpace(resp.Header.Get("Accept-Ranges")); ar != "" {
		w.Header().Set("Accept-Ranges", ar)
	}
	if cr := strings.TrimSpace(resp.Header.Get("Content-Range")); cr != "" {
		w.Header().Set("Content-Range", cr)
	}
	if cl := strings.TrimSpace(resp.Header.Get("Content-Length")); cl != "" {
		w.Header().Set("Content-Length", cl)
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) handleDouyinCover(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key 非法"})
		return
	}

	cached, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || cached == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "解析已过期，请重新解析"})
		return
	}
	remoteURL := strings.TrimSpace(cached.CoverURL)
	if remoteURL == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "封面不存在"})
		return
	}

	method := r.Method
	if method != http.MethodGet && method != http.MethodHead {
		method = http.MethodGet
	}

	doCover := func(urlValue string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), method, urlValue, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://www.douyin.com/")
		return a.douyinDownloader.api.httpClient.Do(req)
	}

	resp, err := doCover(remoteURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "获取封面失败: " + err.Error()})
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusText := resp.Status
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()

		// Best-effort: cover URL might also expire (same root cause as downloads).
		if resp.StatusCode == http.StatusForbidden {
			refreshed, refreshErr := a.douyinDownloader.RefreshDetailBestEffort(cached.DetailID)
			if refreshErr == nil && refreshed != nil && strings.TrimSpace(refreshed.CoverURL) != "" {
				if a.douyinDownloader.cache != nil {
					a.douyinDownloader.cache.Set(key, *refreshed)
				}
				remoteURL = strings.TrimSpace(refreshed.CoverURL)
				cached = refreshed

				resp2, err2 := doCover(remoteURL)
				if err2 != nil {
					writeJSON(w, http.StatusBadRequest, map[string]any{"error": "获取封面失败: " + err2.Error()})
					return
				}
				if resp2.StatusCode >= 200 && resp2.StatusCode < 300 {
					resp = resp2
					defer resp.Body.Close()
					goto writeCover
				}
				body2, _ := io.ReadAll(io.LimitReader(resp2.Body, 1024))
				_ = resp2.Body.Close()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("获取封面失败: %s %s", resp2.Status, strings.TrimSpace(string(body2)))})
				return
			}
		}

		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("获取封面失败: %s %s", statusText, strings.TrimSpace(string(body)))})
		return
	}

	defer resp.Body.Close()

writeCover:
	w.Header().Set("Cache-Control", "no-store")
	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if cl := strings.TrimSpace(resp.Header.Get("Content-Length")); cl != "" {
		w.Header().Set("Content-Length", cl)
	}

	w.WriteHeader(resp.StatusCode)
	if method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, resp.Body)
}

func (a *App) handleDouyinDownloadHead(w http.ResponseWriter, r *http.Request, key string, cached *douyinCachedDetail, index int, remoteURL string) {
	triedRefresh := false

	// Keep backward-compatible error message for malformed URLs (e.g. "http://[::1").
	if _, err := http.NewRequestWithContext(r.Context(), http.MethodGet, remoteURL, nil); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接非法"})
		return
	}

	doHead := func(urlValue string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodHead, urlValue, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://www.douyin.com/")
		return a.douyinDownloader.api.httpClient.Do(req)
	}

	doRange := func(urlValue string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, urlValue, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://www.douyin.com/")
		req.Header.Set("Range", "bytes=0-0")
		return a.douyinDownloader.api.httpClient.Do(req)
	}

retryHead:
	resp, err := doHead(remoteURL)
	if err == nil && resp != nil && resp.Body != nil {
		// HEAD response body is typically empty, but still needs closing.
		defer resp.Body.Close()
	}
	if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		a.writeDouyinDownloadHeaders(w, cached, index, remoteURL, resp.Header, false)
		w.WriteHeader(http.StatusOK)
		return
	}
	if !triedRefresh && resp != nil && resp.StatusCode == http.StatusForbidden {
		triedRefresh = true
		refreshed, refreshErr := a.douyinDownloader.RefreshDetailBestEffort(cached.DetailID)
		if refreshErr == nil && refreshed != nil && index < len(refreshed.Downloads) {
			newURL := strings.TrimSpace(refreshed.Downloads[index])
			if newURL != "" {
				if a.douyinDownloader.cache != nil && strings.TrimSpace(key) != "" {
					a.douyinDownloader.cache.Set(strings.TrimSpace(key), *refreshed)
				}
				cached = refreshed
				remoteURL = newURL
				goto retryHead
			}
		}
	}

	// 部分 CDN 不支持 HEAD：fallback 到 Range=0-0 的 GET，最佳努力拿到 Content-Length。
retryRange:
	rangeResp, rangeErr := doRange(remoteURL)
	if rangeErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + rangeErr.Error()})
		return
	}
	defer rangeResp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(rangeResp.Body, 1))

	if rangeResp.StatusCode < 200 || rangeResp.StatusCode >= 300 {
		if !triedRefresh && rangeResp.StatusCode == http.StatusForbidden {
			triedRefresh = true
			refreshed, refreshErr := a.douyinDownloader.RefreshDetailBestEffort(cached.DetailID)
			if refreshErr == nil && refreshed != nil && index < len(refreshed.Downloads) {
				newURL := strings.TrimSpace(refreshed.Downloads[index])
				if newURL != "" {
					if a.douyinDownloader.cache != nil && strings.TrimSpace(key) != "" {
						a.douyinDownloader.cache.Set(strings.TrimSpace(key), *refreshed)
					}
					cached = refreshed
					remoteURL = newURL
					goto retryRange
				}
			}
		}

		body, _ := io.ReadAll(io.LimitReader(rangeResp.Body, 1024))
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", rangeResp.Status, strings.TrimSpace(string(body)))})
		return
	}

	a.writeDouyinDownloadHeaders(w, cached, index, remoteURL, rangeResp.Header, true)
	w.WriteHeader(http.StatusOK)
}

func (a *App) writeDouyinDownloadHeaders(w http.ResponseWriter, cached *douyinCachedDetail, index int, remoteURL string, hdr http.Header, ranged bool) {
	contentType := strings.TrimSpace(hdr.Get("Content-Type"))
	ext := guessExtFromContentType(contentType)
	if ext == "" {
		ext = guessExtFromURL(remoteURL)
	}
	if ext == "" {
		ext = ".bin"
	}

	originalFilename := buildDouyinOriginalFilename(cached.Title, cached.DetailID, index, len(cached.Downloads), ext)
	fallback := buildDouyinFallbackFilename(cached.DetailID, index, len(cached.Downloads), ext)

	w.Header().Set("Cache-Control", "no-store")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Content-Disposition", buildAttachmentContentDisposition(fallback, originalFilename))

	acceptRanges := strings.TrimSpace(hdr.Get("Accept-Ranges"))
	if acceptRanges != "" {
		w.Header().Set("Accept-Ranges", acceptRanges)
	}

	// ranged=true 时优先从 Content-Range 获取总大小，避免 Content-Length=1 的情况。
	var lengthValue string
	if ranged {
		if total := parseContentRangeTotal(hdr.Get("Content-Range")); total > 0 {
			lengthValue = fmt.Sprintf("%d", total)
		}
	}
	if lengthValue == "" {
		lengthValue = strings.TrimSpace(hdr.Get("Content-Length"))
	}
	if lengthValue != "" {
		w.Header().Set("Content-Length", lengthValue)
	}
}

func (a *App) handleDouyinImport(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.douyinDownloader == nil || a.fileStorage == nil || a.mediaUpload == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "服务未初始化"})
		return
	}
	if !a.douyinDownloader.configured() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）"})
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "请求解析失败"})
		return
	}

	userID := strings.TrimSpace(r.FormValue("userid"))
	if userID == "" {
		userID = "pre_identity"
	}
	key := strings.TrimSpace(r.FormValue("key"))
	indexValue := strings.TrimSpace(r.FormValue("index"))
	index, err := strconv.Atoi(indexValue)
	if key == "" || indexValue == "" || err != nil || index < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "key/index 不能为空或非法"})
		return
	}

	cached, ok := a.douyinDownloader.GetCachedDetail(key)
	if !ok || cached == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "解析已过期，请重新解析"})
		return
	}
	if index >= len(cached.Downloads) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "index 越界"})
		return
	}

	remoteURL := strings.TrimSpace(cached.Downloads[index])
	if remoteURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载链接为空"})
		return
	}

	doImportDownload := func(urlValue string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, urlValue, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://www.douyin.com/")
		return a.douyinDownloader.api.httpClient.Do(req)
	}

	downloadResp, err := doImportDownload(remoteURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + err.Error()})
		return
	}
	if downloadResp.StatusCode < 200 || downloadResp.StatusCode >= 300 {
		statusText := downloadResp.Status
		body, _ := io.ReadAll(io.LimitReader(downloadResp.Body, 1024))
		_ = downloadResp.Body.Close()

		// Best-effort: cached downloads may expire; refresh once on 403 and retry.
		if downloadResp.StatusCode == http.StatusForbidden {
			refreshed, refreshErr := a.douyinDownloader.RefreshDetailBestEffort(cached.DetailID)
			if refreshErr == nil && refreshed != nil && index < len(refreshed.Downloads) {
				newURL := strings.TrimSpace(refreshed.Downloads[index])
				if newURL != "" {
					if a.douyinDownloader.cache != nil && strings.TrimSpace(key) != "" {
						a.douyinDownloader.cache.Set(strings.TrimSpace(key), *refreshed)
					}
					remoteURL = newURL
					cached = refreshed

					downloadResp2, err2 := doImportDownload(remoteURL)
					if err2 != nil {
						writeJSON(w, http.StatusBadRequest, map[string]any{"error": "下载失败: " + err2.Error()})
						return
					}
					if downloadResp2.StatusCode >= 200 && downloadResp2.StatusCode < 300 {
						downloadResp = downloadResp2
						defer downloadResp.Body.Close()
						goto importContinue
					}
					body2, _ := io.ReadAll(io.LimitReader(downloadResp2.Body, 1024))
					_ = downloadResp2.Body.Close()
					writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", downloadResp2.Status, strings.TrimSpace(string(body2)))})
					return
				}
			}
		}

		writeJSON(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("下载失败: %s %s", statusText, strings.TrimSpace(string(body)))})
		return
	}
	defer downloadResp.Body.Close()

importContinue:
	contentType := strings.TrimSpace(downloadResp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = inferContentTypeFromFilename("x" + guessExtFromURL(remoteURL))
	}
	if strings.TrimSpace(contentType) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "无法识别文件类型"})
		return
	}
	if !a.fileStorage.IsValidMediaType(contentType) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "不支持的文件类型"})
		return
	}

	ext := guessExtFromContentType(contentType)

	originalFilename := buildDouyinOriginalFilename(cached.Title, cached.DetailID, index, len(cached.Downloads), ext)
	localPath, fileSize, md5Value, err := a.fileStorage.SaveFileFromReaderInSubdir(originalFilename, contentType, downloadResp.Body, "douyin")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "本地存储失败: " + err.Error()})
		return
	}

	// 已导入过则直接复用：避免重复写文件导致孤儿文件
	dedup := false
	localFilename := filepath.Base(strings.TrimPrefix(localPath, "/"))
	fileExtension := a.fileStorage.FileExtension(originalFilename)
	if md5Value != "" {
		if existing, err := a.mediaUpload.findStoredMediaFileByMD5(r.Context(), md5Value); err == nil && existing != nil && existing.File != nil {
			_ = a.fileStorage.DeleteFile(localPath)
			dedup = true
			localPath = existing.File.LocalPath
			if strings.TrimSpace(existing.File.LocalFilename) != "" {
				localFilename = existing.File.LocalFilename
			}
			fileSize = existing.File.FileSize
			contentType = existing.File.FileType
			if strings.TrimSpace(existing.File.FileExtension) != "" {
				fileExtension = existing.File.FileExtension
			}
		}
	}

	_, _ = a.mediaUpload.SaveDouyinUploadRecord(r.Context(), DouyinUploadRecord{
		UserID:           userID,
		SecUserID:        strings.TrimSpace(cached.SecUserID),
		DetailID:         strings.TrimSpace(cached.DetailID),
		OriginalFilename: originalFilename,
		LocalFilename:    localFilename,
		RemoteFilename:   "",
		RemoteURL:        "",
		LocalPath:        localPath,
		FileSize:         fileSize,
		FileType:         contentType,
		FileExtension:    fileExtension,
		FileMD5:          md5Value,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"state":         "OK",
		"dedup":         dedup,
		"uploaded":      false,
		"localPath":     localPath,
		"localFilename": localFilename,
	})
}

func guessExtFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err == nil && u != nil {
		ext := strings.ToLower(filepath.Ext(u.Path))
		if ext != "" && len(ext) <= 10 {
			return ext
		}
		return ""
	}
	ext := strings.ToLower(filepath.Ext(raw))
	if ext != "" && len(ext) <= 10 {
		return ext
	}
	return ""
}

func buildDouyinOriginalFilename(title, detailID string, index, total int, ext string) string {
	base := sanitizeFilename(title)
	if base == "" {
		base = strings.TrimSpace(detailID)
	}
	if strings.TrimSpace(ext) == "" {
		ext = ".bin"
	}
	if total > 1 {
		return fmt.Sprintf("%s_%02d%s", base, index+1, ext)
	}
	return base + ext
}

func buildDouyinFallbackFilename(detailID string, index, total int, ext string) string {
	id := strings.TrimSpace(detailID)
	if id == "" {
		id = "unknown"
	}
	if strings.TrimSpace(ext) == "" {
		ext = ".bin"
	}
	if total > 1 {
		return fmt.Sprintf("douyin_%s_%02d%s", id, index+1, ext)
	}
	return fmt.Sprintf("douyin_%s%s", id, ext)
}

func buildAttachmentContentDisposition(fallback, originalFilename string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "download"
	}
	originalFilename = strings.TrimSpace(originalFilename)
	if originalFilename == "" {
		originalFilename = fallback
	}
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, url.PathEscape(originalFilename))
}

func sanitizeFilename(input string) string {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return ""
	}

	// 替换常见非法字符（Windows/URL/控制字符）
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", " ",
		"\t", " ",
	)
	raw = replacer.Replace(raw)
	raw = strings.TrimSpace(strings.Join(strings.Fields(raw), " "))

	// 控制长度（按 rune 避免截断中文）
	rs := []rune(raw)
	if len(rs) > 100 {
		raw = string(rs[:100])
	}
	return strings.TrimSpace(raw)
}

func parseContentRangeTotal(contentRange string) int64 {
	raw := strings.TrimSpace(contentRange)
	if raw == "" {
		return 0
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return 0
	}
	totalStr := strings.TrimSpace(parts[1])
	if totalStr == "" || totalStr == "*" {
		return 0
	}
	n, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
