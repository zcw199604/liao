package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type tikTokDownloaderURLResponse struct {
	Message string `json:"message"`
	URL     string `json:"url"`
	Params  any    `json:"params,omitempty"`
	Time    string `json:"time,omitempty"`
}

type tikTokDownloaderDataResponse struct {
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
	Params  any            `json:"params,omitempty"`
	Time    string         `json:"time,omitempty"`
}

type TikTokDownloaderClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewTikTokDownloaderClient(baseURL, token string, httpClient *http.Client) *TikTokDownloaderClient {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &TikTokDownloaderClient{
		baseURL:    baseURL,
		token:      strings.TrimSpace(token),
		httpClient: httpClient,
	}
}

func (c *TikTokDownloaderClient) configured() bool {
	return c != nil && strings.TrimSpace(c.baseURL) != ""
}

func (c *TikTokDownloaderClient) postJSON(ctx context.Context, path string, payload any, out any) error {
	if !c.configured() {
		return fmt.Errorf("TikTokDownloader 未配置（请设置 TIKTOKDOWNLOADER_BASE_URL）")
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// 上游默认不校验 token；但路由依赖会读取 Header("token")，这里统一带上（可为空）。
	req.Header.Set("token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 300 {
			msg = msg[:300] + "..."
		}
		return fmt.Errorf("TikTokDownloader 上游错误: %s %s", resp.Status, msg)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("TikTokDownloader 响应解析失败: %w", err)
	}
	return nil
}

func (c *TikTokDownloaderClient) DouyinShare(ctx context.Context, text, proxy string) (string, error) {
	var resp tikTokDownloaderURLResponse
	if err := c.postJSON(ctx, "/douyin/share", map[string]any{
		"text":  text,
		"proxy": proxy,
	}, &resp); err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.URL), nil
}

func (c *TikTokDownloaderClient) DouyinDetail(ctx context.Context, detailID, cookie, proxy string) (map[string]any, error) {
	var resp tikTokDownloaderDataResponse
	if err := c.postJSON(ctx, "/douyin/detail", map[string]any{
		"detail_id": detailID,
		"cookie":    cookie,
		"proxy":     proxy,
		"source":    false,
	}, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("TikTokDownloader 获取数据失败: %s", strings.TrimSpace(resp.Message))
	}
	return resp.Data, nil
}

func (c *TikTokDownloaderClient) DouyinAccount(ctx context.Context, secUserID, tab string, cursor, count int, cookie, proxy string) (map[string]any, error) {
	var resp tikTokDownloaderDataResponse
	payload := map[string]any{
		"sec_user_id": secUserID,
		"tab":         tab,
		"cursor":      cursor,
		"count":       count,
		"cookie":      cookie,
		"proxy":       proxy,
		"source":      false,
		"pages":       1, // 每次只请求一页，方便前端按 cursor 分页加载
	}
	if err := c.postJSON(ctx, "/douyin/account", payload, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("TikTokDownloader 获取数据失败: %s", strings.TrimSpace(resp.Message))
	}
	return resp.Data, nil
}

type DouyinDownloaderService struct {
	api *TikTokDownloaderClient

	cache *lruCache

	upstreamTimeout time.Duration

	defaultCookie string
	defaultProxy  string
}

type douyinCachedDetail struct {
	DetailID  string
	Title     string
	Type      string
	CoverURL  string
	Downloads []string
}

func NewDouyinDownloaderService(baseURL, token, defaultCookie, defaultProxy string, upstreamTimeout time.Duration) *DouyinDownloaderService {
	if upstreamTimeout <= 0 {
		upstreamTimeout = 60 * time.Second
	}
	httpClient := &http.Client{Timeout: upstreamTimeout}
	return &DouyinDownloaderService{
		api:             NewTikTokDownloaderClient(baseURL, token, httpClient),
		cache:           newLRUCache(2000, 15*time.Minute),
		upstreamTimeout: upstreamTimeout,
		defaultCookie:   strings.TrimSpace(defaultCookie),
		defaultProxy:    strings.TrimSpace(defaultProxy),
	}
}

func (s *DouyinDownloaderService) configured() bool {
	return s != nil && s.api != nil && s.api.configured()
}

func (s *DouyinDownloaderService) effectiveUpstreamTimeout() time.Duration {
	if s == nil || s.upstreamTimeout <= 0 {
		return 60 * time.Second
	}
	return s.upstreamTimeout
}

var (
	reDouyinIDOnly   = regexp.MustCompile(`^[0-9]+$`)
	reDouyinVideoID  = regexp.MustCompile(`/video/([0-9]+)`)
	reDouyinNoteID   = regexp.MustCompile(`/note/([0-9]+)`)
	reDouyinModalID  = regexp.MustCompile(`(?i)[?&]modal_id=([0-9]+)`)
	reDouyinAwemeID  = regexp.MustCompile(`(?i)[?&]aweme_id=([0-9]+)`)
	reDouyinShareURL = regexp.MustCompile(`https?://[^\s]+`)
	reDouyinUserPath = regexp.MustCompile(`/user/([^/?\s]+)`)
	reDouyinSecUID   = regexp.MustCompile(`(?i)[?&](sec_uid|sec_user_id)=([^&]+)`)
)

func extractDouyinDetailID(text string) string {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return ""
	}
	if reDouyinIDOnly.MatchString(raw) {
		return raw
	}
	if m := reDouyinVideoID.FindStringSubmatch(raw); len(m) == 2 {
		return m[1]
	}
	if m := reDouyinNoteID.FindStringSubmatch(raw); len(m) == 2 {
		return m[1]
	}
	if m := reDouyinModalID.FindStringSubmatch(raw); len(m) == 2 {
		return m[1]
	}
	if m := reDouyinAwemeID.FindStringSubmatch(raw); len(m) == 2 {
		return m[1]
	}

	// 兼容分享文本：先从文本中抽取 URL 再尝试匹配
	if m := reDouyinShareURL.FindStringSubmatch(raw); len(m) >= 1 {
		u := m[0]
		if m2 := reDouyinVideoID.FindStringSubmatch(u); len(m2) == 2 {
			return m2[1]
		}
		if m2 := reDouyinNoteID.FindStringSubmatch(u); len(m2) == 2 {
			return m2[1]
		}
		if m2 := reDouyinModalID.FindStringSubmatch(u); len(m2) == 2 {
			return m2[1]
		}
		if m2 := reDouyinAwemeID.FindStringSubmatch(u); len(m2) == 2 {
			return m2[1]
		}
	}

	return ""
}

func extractDouyinSecUserID(text string) string {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return ""
	}

	// 直接粘贴 sec_user_id（常见前缀 MS4wLjAB...），避免误判：要求“无空白 + 长度足够”
	if !strings.ContainsAny(raw, " \n\t") && len(raw) >= 16 && strings.HasPrefix(raw, "MS4wLjAB") {
		return raw
	}

	if m := reDouyinUserPath.FindStringSubmatch(raw); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	if m := reDouyinSecUID.FindStringSubmatch(raw); len(m) == 3 {
		if decoded, err := url.QueryUnescape(m[2]); err == nil && strings.TrimSpace(decoded) != "" {
			return strings.TrimSpace(decoded)
		}
		return strings.TrimSpace(m[2])
	}

	// 兼容分享文本：先从文本中抽取 URL 再尝试匹配
	if m := reDouyinShareURL.FindStringSubmatch(raw); len(m) >= 1 {
		u := m[0]
		if m2 := reDouyinUserPath.FindStringSubmatch(u); len(m2) == 2 {
			return strings.TrimSpace(m2[1])
		}
		if m2 := reDouyinSecUID.FindStringSubmatch(u); len(m2) == 3 {
			if decoded, err := url.QueryUnescape(m2[2]); err == nil && strings.TrimSpace(decoded) != "" {
				return strings.TrimSpace(decoded)
			}
			return strings.TrimSpace(m2[2])
		}
	}

	return ""
}

func (s *DouyinDownloaderService) effectiveCookie(cookie string) string {
	if v := strings.TrimSpace(cookie); v != "" {
		return v
	}
	return s.defaultCookie
}

func (s *DouyinDownloaderService) effectiveProxy(proxy string) string {
	if v := strings.TrimSpace(proxy); v != "" {
		return v
	}
	return s.defaultProxy
}

func (s *DouyinDownloaderService) ResolveDetailID(ctx context.Context, input, proxy string) (detailID string, resolvedURL string, err error) {
	if !s.configured() {
		return "", "", fmt.Errorf("抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）")
	}

	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", "", fmt.Errorf("input 不能为空")
	}

	if id := extractDouyinDetailID(raw); id != "" {
		return id, "", nil
	}

	// 短链/分享文本：调用 /douyin/share 获取重定向后的完整链接，再提取 detail_id
	ctx2, cancel := context.WithTimeout(ctx, s.effectiveUpstreamTimeout())
	defer cancel()

	urlValue, err := s.api.DouyinShare(ctx2, raw, s.effectiveProxy(proxy))
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(urlValue) == "" {
		return "", "", fmt.Errorf("无法解析分享链接：share 返回为空")
	}
	if id := extractDouyinDetailID(urlValue); id != "" {
		return id, urlValue, nil
	}

	return "", urlValue, fmt.Errorf("无法从链接提取作品ID: %s", urlValue)
}

func (s *DouyinDownloaderService) ResolveSecUserID(ctx context.Context, input, proxy string) (secUserID string, resolvedURL string, err error) {
	if !s.configured() {
		return "", "", fmt.Errorf("抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）")
	}

	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", "", fmt.Errorf("input 不能为空")
	}

	if id := extractDouyinSecUserID(raw); id != "" {
		return id, "", nil
	}

	// 短链/分享文本：调用 /douyin/share 获取重定向后的完整链接，再提取 sec_user_id
	ctx2, cancel := context.WithTimeout(ctx, s.effectiveUpstreamTimeout())
	defer cancel()

	urlValue, err := s.api.DouyinShare(ctx2, raw, s.effectiveProxy(proxy))
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(urlValue) == "" {
		return "", "", fmt.Errorf("无法解析分享链接：share 返回为空")
	}
	if id := extractDouyinSecUserID(urlValue); id != "" {
		return id, urlValue, nil
	}

	return "", urlValue, fmt.Errorf("无法从链接提取 sec_user_id: %s", urlValue)
}

func (s *DouyinDownloaderService) FetchDetail(ctx context.Context, detailID, cookie, proxy string) (*douyinCachedDetail, error) {
	if !s.configured() {
		return nil, fmt.Errorf("抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）")
	}
	detailID = strings.TrimSpace(detailID)
	if detailID == "" {
		return nil, fmt.Errorf("detail_id 不能为空")
	}

	ctx2, cancel := context.WithTimeout(ctx, s.effectiveUpstreamTimeout())
	defer cancel()

	data, err := s.api.DouyinDetail(ctx2, detailID, s.effectiveCookie(cookie), s.effectiveProxy(proxy))
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(asString(data["desc"]))
	if title == "" {
		title = detailID
	}
	typeValue := strings.TrimSpace(asString(data["type"]))
	cover := strings.TrimSpace(asString(data["static_cover"]))
	if cover == "" {
		cover = strings.TrimSpace(asString(data["dynamic_cover"]))
	}

	downloads := extractStringSlice(data["downloads"])
	if len(downloads) == 0 {
		// 兼容：上游字段异常时尝试兜底
		if v := strings.TrimSpace(asString(data["download"])); v != "" {
			downloads = []string{v}
		}
	}
	if len(downloads) == 0 {
		return nil, fmt.Errorf("上游返回缺少 downloads 字段")
	}

	return &douyinCachedDetail{
		DetailID:  detailID,
		Title:     title,
		Type:      typeValue,
		CoverURL:  cover,
		Downloads: downloads,
	}, nil
}

func (s *DouyinDownloaderService) FetchAccount(ctx context.Context, secUserID, tab, cookie, proxy string, cursor, count int) (map[string]any, error) {
	if !s.configured() {
		return nil, fmt.Errorf("抖音下载未启用（请配置 TIKTOKDOWNLOADER_BASE_URL）")
	}
	secUserID = strings.TrimSpace(secUserID)
	if secUserID == "" {
		return nil, fmt.Errorf("sec_user_id 不能为空")
	}
	if strings.TrimSpace(tab) == "" {
		tab = "post"
	}
	if count <= 0 {
		count = 18
	}
	if cursor < 0 {
		cursor = 0
	}

	ctx2, cancel := context.WithTimeout(ctx, s.effectiveUpstreamTimeout())
	defer cancel()

	data, err := s.api.DouyinAccount(ctx2, secUserID, tab, cursor, count, s.effectiveCookie(cookie), s.effectiveProxy(proxy))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *DouyinDownloaderService) CacheDetail(detail *douyinCachedDetail) string {
	if s == nil || s.cache == nil || detail == nil {
		return ""
	}
	key := strings.ReplaceAll(uuid.NewString(), "-", "")
	s.cache.Set(key, *detail)
	return key
}

func (s *DouyinDownloaderService) GetCachedDetail(key string) (*douyinCachedDetail, bool) {
	if s == nil || s.cache == nil {
		return nil, false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false
	}
	v, ok := s.cache.Get(key)
	if !ok {
		return nil, false
	}
	typed, ok := v.(douyinCachedDetail)
	if !ok {
		return nil, false
	}
	return &typed, true
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return ""
	}
}

func extractStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{strings.TrimSpace(t)}
	case []any:
		out := make([]string, 0, len(t))
		for _, it := range t {
			s := strings.TrimSpace(asString(it))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(t))
		for _, it := range t {
			s := strings.TrimSpace(it)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
