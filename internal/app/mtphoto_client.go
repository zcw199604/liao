package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MtPhotoService 负责对接 mtPhoto 相册系统：
// - 统一在后端完成登录/续期（优先 refresh_token；过期或 401/403 自动续期并重试一次）
// - 提供相册/媒体查询与 gateway 代理能力
// - 提供轻量缓存（避免频繁拉取大相册列表）
type MtPhotoService struct {
	baseURL string

	username string
	password string
	otp      string

	lspRoot string

	httpClient *http.Client

	mu           sync.Mutex
	token        string
	tokenExp     time.Time
	authCode     string
	refreshToken string
	loginOnce    time.Time

	albumsCache       []MtPhotoAlbum
	albumsCacheExpire time.Time

	albumFilesCache map[int]mtPhotoAlbumFilesCacheEntry
}

type mtPhotoAlbumFilesCacheEntry struct {
	expireAt time.Time
	total    int
	items    []MtPhotoMediaItem
}

type mtPhotoStatusError struct {
	StatusCode int
	Status     string
	Action     string
}

func (e *mtPhotoStatusError) Error() string {
	if e == nil {
		return "mtPhoto 请求失败"
	}

	status := strings.TrimSpace(e.Status)
	if status == "" && e.StatusCode > 0 {
		status = fmt.Sprintf("HTTP %d", e.StatusCode)
	}

	action := strings.TrimSpace(e.Action)
	if action == "" {
		action = "mtPhoto 请求失败"
	}
	if status == "" {
		return action
	}
	return fmt.Sprintf("%s: %s", action, status)
}

func NewMtPhotoService(baseURL, username, password, otp, lspRoot string, httpClient *http.Client) *MtPhotoService {
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")

	lspRoot = strings.TrimSpace(lspRoot)
	if lspRoot == "" {
		lspRoot = "/lsp"
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	return &MtPhotoService{
		baseURL:         baseURL,
		username:        strings.TrimSpace(username),
		password:        strings.TrimSpace(password),
		otp:             strings.TrimSpace(otp),
		lspRoot:         lspRoot,
		httpClient:      httpClient,
		albumFilesCache: make(map[int]mtPhotoAlbumFilesCacheEntry),
	}
}

func (s *MtPhotoService) configured() bool {
	return s != nil && s.baseURL != "" && s.username != "" && s.password != ""
}

type mtPhotoLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	OTP      string `json:"otp"`
}

type mtPhotoLoginResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	AuthCode     string `json:"auth_code"`
	RefreshToken string `json:"refresh_token"`
}

func (s *MtPhotoService) ensureLogin(ctx context.Context, force bool) (token string, authCode string, err error) {
	if !s.configured() {
		return "", "", fmt.Errorf("mtPhoto 未配置（请设置 MTPHOTO_BASE_URL/MTPHOTO_LOGIN_USERNAME/MTPHOTO_LOGIN_PASSWORD）")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// force=true 时强制续期；否则只有在缺失/即将过期时才续期
	needRenew := force || strings.TrimSpace(s.token) == "" || strings.TrimSpace(s.authCode) == ""
	if !needRenew && !s.tokenExp.IsZero() {
		// 提前 60 秒续期，避免边界 401
		if time.Now().After(s.tokenExp.Add(-60 * time.Second)) {
			needRenew = true
		}
	}

	if !needRenew {
		return s.token, s.authCode, nil
	}

	// 限流：短时间内多请求并发 401 时避免瞬间风暴
	if !force && !s.loginOnce.IsZero() && time.Since(s.loginOnce) < 800*time.Millisecond {
		// 轻微等待，降低短时间内重复 refresh/login 的冲击
		time.Sleep(120 * time.Millisecond)
	}
	s.loginOnce = time.Now()

	// 续期策略：优先 refresh_token；失败则回退到账号登录
	if strings.TrimSpace(s.refreshToken) != "" {
		if err := s.refreshLocked(ctx); err == nil {
			return s.token, s.authCode, nil
		}
	}

	if err := s.loginLocked(ctx); err != nil {
		return "", "", err
	}
	return s.token, s.authCode, nil
}

type mtPhotoRefreshRequest struct {
	Token string `json:"token"`
}

func (s *MtPhotoService) refreshLocked(ctx context.Context) error {
	refreshToken := strings.TrimSpace(s.refreshToken)
	if refreshToken == "" {
		return fmt.Errorf("refresh_token 为空")
	}

	refreshURL := s.baseURL + "/auth/refresh"
	body, _ := json.Marshal(mtPhotoRefreshRequest{Token: refreshToken})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mtPhoto refresh 失败: %s", resp.Status)
	}

	var parsed mtPhotoLoginResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("mtPhoto refresh 响应解析失败: %w", err)
	}
	if strings.TrimSpace(parsed.AccessToken) == "" || strings.TrimSpace(parsed.AuthCode) == "" {
		return fmt.Errorf("mtPhoto refresh 响应缺少 access_token/auth_code")
	}

	s.token = strings.TrimSpace(parsed.AccessToken)
	s.authCode = strings.TrimSpace(parsed.AuthCode)
	if rt := strings.TrimSpace(parsed.RefreshToken); rt != "" {
		s.refreshToken = rt
	}
	s.tokenExp = parseMtPhotoExpiresIn(parsed.ExpiresIn)

	s.resetCachesLocked()
	return nil
}

func (s *MtPhotoService) loginLocked(ctx context.Context) error {
	loginURL := s.baseURL + "/auth/login"
	body, _ := json.Marshal(mtPhotoLoginRequest{
		Username: s.username,
		Password: s.password,
		OTP:      s.otp,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mtPhoto 登录失败: %s", resp.Status)
	}

	var parsed mtPhotoLoginResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("mtPhoto 登录响应解析失败: %w", err)
	}
	if strings.TrimSpace(parsed.AccessToken) == "" || strings.TrimSpace(parsed.AuthCode) == "" {
		return fmt.Errorf("mtPhoto 登录响应缺少 access_token/auth_code")
	}

	s.token = strings.TrimSpace(parsed.AccessToken)
	s.authCode = strings.TrimSpace(parsed.AuthCode)
	if rt := strings.TrimSpace(parsed.RefreshToken); rt != "" {
		s.refreshToken = rt
	}
	s.tokenExp = parseMtPhotoExpiresIn(parsed.ExpiresIn)

	s.resetCachesLocked()
	return nil
}

func parseMtPhotoExpiresIn(expiresIn int64) time.Time {
	if expiresIn <= 0 {
		return time.Time{}
	}
	// 文档示例为 epoch ms；但为兼容部分实现，支持 seconds TTL。
	if expiresIn < 1_000_000_000_000 {
		return time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	return time.UnixMilli(expiresIn)
}

func (s *MtPhotoService) resetCachesLocked() {
	// 登录态/续期后清空缓存，避免用户侧看到旧数据
	s.albumsCache = nil
	s.albumsCacheExpire = time.Time{}
	s.albumFilesCache = make(map[int]mtPhotoAlbumFilesCacheEntry)
}

func (s *MtPhotoService) doRequest(ctx context.Context, method, urlStr string, headers map[string]string, body []byte, useJWT, useCookie bool) (*http.Response, error) {
	// 尝试两次：首次使用现有 token；401/403 后强制续期再试一次
	lastAuthStatusCode := 0
	lastAuthStatus := ""
	for attempt := 0; attempt < 2; attempt++ {
		force := attempt == 1
		token, authCode, err := s.ensureLogin(ctx, force)
		if err != nil {
			return nil, err
		}

		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, urlStr, reader)
		if err != nil {
			return nil, err
		}

		for k, v := range headers {
			if strings.TrimSpace(k) == "" {
				continue
			}
			req.Header.Set(k, v)
		}
		if useJWT {
			req.Header.Set("jwt", token)
		}
		if useCookie {
			// mtPhoto 的 gateway 资源需要 auth_code cookie
			req.Header.Set("Cookie", "auth_code="+authCode)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			lastAuthStatusCode = resp.StatusCode
			lastAuthStatus = resp.Status
			_ = resp.Body.Close()
			continue
		}
		return resp, nil
	}

	if lastAuthStatusCode > 0 {
		return nil, &mtPhotoStatusError{
			StatusCode: lastAuthStatusCode,
			Status:     lastAuthStatus,
			Action:     "mtPhoto 请求鉴权失败",
		}
	}
	return nil, fmt.Errorf("mtPhoto 请求鉴权失败")
}

func (s *MtPhotoService) buildURL(pathname string, query url.Values) (string, error) {
	if strings.TrimSpace(pathname) == "" {
		return "", fmt.Errorf("pathname 为空")
	}
	u, err := url.Parse(s.baseURL + pathname)
	if err != nil {
		return "", err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

type MtPhotoAlbum struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Cover     string `json:"cover"`
	Count     int    `json:"count"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

func (s *MtPhotoService) GetAlbums(ctx context.Context) ([]MtPhotoAlbum, error) {
	if !s.configured() {
		return nil, fmt.Errorf("mtPhoto 未配置")
	}

	// 轻量缓存：避免频繁打开弹窗导致重复拉取
	s.mu.Lock()
	if len(s.albumsCache) > 0 && time.Now().Before(s.albumsCacheExpire) {
		out := make([]MtPhotoAlbum, len(s.albumsCache))
		copy(out, s.albumsCache)
		s.mu.Unlock()
		return out, nil
	}
	s.mu.Unlock()

	urlStr, err := s.buildURL("/api-album", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.doRequest(ctx, http.MethodGet, urlStr, map[string]string{
		"Accept": "application/json",
	}, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mtPhoto 获取相册失败: %s", resp.Status)
	}

	var parsed []MtPhotoAlbum
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.albumsCache = parsed
	s.albumsCacheExpire = time.Now().Add(30 * time.Second)
	s.mu.Unlock()

	return parsed, nil
}

type MtPhotoFolderItem struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Hide             bool    `json:"hide,omitempty"`
	GalleryName      string  `json:"galleryName,omitempty"`
	GalleryFolderNum int     `json:"galleryFolderNum,omitempty"`
	Path             string  `json:"path,omitempty"`
	Cover            string  `json:"cover,omitempty"`
	SCover           *string `json:"s_cover"`
	SubFolderNum     int     `json:"subFolderNum,omitempty"`
	SubFileNum       int     `json:"subFileNum,omitempty"`
	TrashNum         int     `json:"trashNum,omitempty"`
	FileType         string  `json:"fileType,omitempty"`
}

type mtPhotoFolderFileItemRaw struct {
	ID       int64    `json:"id"`
	FileName string   `json:"fileName"`
	FileType string   `json:"fileType"`
	Size     any      `json:"size"`
	TokenAt  string   `json:"tokenAt"`
	MD5      string   `json:"MD5"`
	MD5Lower string   `json:"md5"`
	Width    int      `json:"width,omitempty"`
	Height   int      `json:"height,omitempty"`
	Duration *float64 `json:"duration"`
	Status   int      `json:"status"`
}

type MtPhotoFolderFileItem struct {
	ID       int64    `json:"id"`
	FileName string   `json:"fileName"`
	FileType string   `json:"fileType"`
	Size     string   `json:"size"`
	TokenAt  string   `json:"tokenAt"`
	Day      string   `json:"day,omitempty"`
	MD5      string   `json:"md5"`
	Width    int      `json:"width,omitempty"`
	Height   int      `json:"height,omitempty"`
	Duration *float64 `json:"duration"`
	Status   int      `json:"status"`
	Type     string   `json:"type"`
}

type mtPhotoFolderResponse struct {
	Path       string                     `json:"path"`
	FolderList []MtPhotoFolderItem        `json:"folderList"`
	FileList   []mtPhotoFolderFileItemRaw `json:"fileList"`
	TrashNum   int                        `json:"trashNum"`
}

type MtPhotoFolderContent struct {
	Path       string                  `json:"path"`
	FolderList []MtPhotoFolderItem     `json:"folderList"`
	FileList   []MtPhotoFolderFileItem `json:"fileList"`
	TrashNum   int                     `json:"trashNum,omitempty"`
}

func normalizeMtPhotoSize(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		if v < 0 {
			return ""
		}
		return strconv.FormatInt(int64(v), 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func normalizeMtPhotoDay(value string) string {
	day := strings.TrimSpace(value)
	if day == "" {
		return ""
	}
	if len(day) >= 10 {
		day = day[:10]
	}
	if _, err := time.Parse("2006-01-02", day); err == nil {
		return day
	}
	return ""
}

func mapMtPhotoFolderFiles(items []mtPhotoFolderFileItemRaw) []MtPhotoFolderFileItem {
	if len(items) == 0 {
		return []MtPhotoFolderFileItem{}
	}

	out := make([]MtPhotoFolderFileItem, 0, len(items))
	for _, item := range items {
		md5Value := strings.TrimSpace(item.MD5)
		if md5Value == "" {
			md5Value = strings.TrimSpace(item.MD5Lower)
		}
		if md5Value == "" {
			continue
		}
		out = append(out, MtPhotoFolderFileItem{
			ID:       item.ID,
			FileName: strings.TrimSpace(item.FileName),
			FileType: strings.TrimSpace(item.FileType),
			Size:     normalizeMtPhotoSize(item.Size),
			TokenAt:  strings.TrimSpace(item.TokenAt),
			Day:      normalizeMtPhotoDay(item.TokenAt),
			MD5:      md5Value,
			Width:    item.Width,
			Height:   item.Height,
			Duration: item.Duration,
			Status:   item.Status,
			Type:     inferMtPhotoType(item.FileType),
		})
	}
	if out == nil {
		return []MtPhotoFolderFileItem{}
	}
	return out
}

func (s *MtPhotoService) getFolderData(ctx context.Context, pathname string, action string) (*MtPhotoFolderContent, error) {
	if !s.configured() {
		return nil, fmt.Errorf("mtPhoto 未配置")
	}

	urlStr, err := s.buildURL(pathname, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.doRequest(ctx, http.MethodGet, urlStr, map[string]string{
		"Accept": "application/json",
	}, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &mtPhotoStatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Action:     action,
		}
	}

	var parsed mtPhotoFolderResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%s 响应解析失败: %w", strings.TrimSpace(action), err)
	}

	folderList := parsed.FolderList
	if folderList == nil {
		folderList = []MtPhotoFolderItem{}
	}

	return &MtPhotoFolderContent{
		Path:       strings.TrimSpace(parsed.Path),
		FolderList: folderList,
		FileList:   mapMtPhotoFolderFiles(parsed.FileList),
		TrashNum:   parsed.TrashNum,
	}, nil
}

func (s *MtPhotoService) GetFolderRoot(ctx context.Context) (*MtPhotoFolderContent, error) {
	return s.getFolderData(ctx, "/gateway/folders/root", "mtPhoto 获取目录根节点失败")
}

func (s *MtPhotoService) GetFolderBreadcrumbs(ctx context.Context, folderID int64) (*MtPhotoFolderContent, error) {
	if folderID <= 0 {
		return nil, fmt.Errorf("folderId 非法")
	}
	return s.getFolderData(ctx, fmt.Sprintf("/gateway/folderBreadcrumbs/%d", folderID), "mtPhoto 获取目录面包屑失败")
}

func paginateMtPhotoFolderItems(all []MtPhotoFolderFileItem, page, pageSize int) []MtPhotoFolderFileItem {
	if len(all) == 0 || pageSize <= 0 {
		return []MtPhotoFolderFileItem{}
	}
	start := (page - 1) * pageSize
	if start >= len(all) {
		return []MtPhotoFolderFileItem{}
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	out := make([]MtPhotoFolderFileItem, 0, end-start)
	out = append(out, all[start:end]...)
	return out
}

func cloneDuration(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	duration := value
	return &duration
}

func mapMtPhotoTimelineFiles(groups []mtPhotoFolderFilesGroup) []MtPhotoFolderFileItem {
	type timelineItem struct {
		item MtPhotoFolderFileItem
		seq  int
	}

	if len(groups) == 0 {
		return []MtPhotoFolderFileItem{}
	}

	flattened := make([]timelineItem, 0)
	seq := 0
	for _, group := range groups {
		day := normalizeMtPhotoDay(group.Day)
		for _, file := range group.List {
			md5Value := strings.TrimSpace(file.MD5)
			if md5Value == "" {
				md5Value = strings.TrimSpace(file.VMD5)
			}
			if md5Value == "" {
				continue
			}

			fileType := strings.TrimSpace(file.FileType)
			flattened = append(flattened, timelineItem{
				item: MtPhotoFolderFileItem{
					ID:       file.ID,
					FileType: fileType,
					Day:      day,
					MD5:      md5Value,
					Width:    file.Width,
					Height:   file.Height,
					Duration: cloneDuration(file.Duration),
					Status:   file.Status,
					Type:     inferMtPhotoType(fileType),
				},
				seq: seq,
			})
			seq++
		}
	}

	sort.SliceStable(flattened, func(i, j int) bool {
		left := flattened[i].item.Day
		right := flattened[j].item.Day
		switch {
		case left == "" && right == "":
			return flattened[i].seq < flattened[j].seq
		case left == "":
			return false
		case right == "":
			return true
		case left == right:
			return flattened[i].seq < flattened[j].seq
		default:
			return left > right
		}
	})

	out := make([]MtPhotoFolderFileItem, 0, len(flattened))
	for _, item := range flattened {
		out = append(out, item.item)
	}
	return out
}

func mergeFolderTimelineWithDetail(timelineFiles, detailFiles []MtPhotoFolderFileItem) []MtPhotoFolderFileItem {
	if len(timelineFiles) == 0 {
		return []MtPhotoFolderFileItem{}
	}
	if len(detailFiles) == 0 {
		return timelineFiles
	}

	detailByID := make(map[int64]MtPhotoFolderFileItem, len(detailFiles))
	detailByMD5 := make(map[string]MtPhotoFolderFileItem, len(detailFiles))
	for _, item := range detailFiles {
		if item.ID > 0 {
			detailByID[item.ID] = item
		}
		md5Value := strings.TrimSpace(item.MD5)
		if md5Value != "" {
			detailByMD5[md5Value] = item
		}
	}

	out := make([]MtPhotoFolderFileItem, 0, len(timelineFiles))
	for _, timeline := range timelineFiles {
		merged := timeline
		var detail MtPhotoFolderFileItem
		var ok bool
		if merged.ID > 0 {
			detail, ok = detailByID[merged.ID]
		}
		if !ok {
			detail, ok = detailByMD5[strings.TrimSpace(merged.MD5)]
		}
		if ok {
			if merged.FileName == "" {
				merged.FileName = detail.FileName
			}
			if merged.Size == "" {
				merged.Size = detail.Size
			}
			if merged.TokenAt == "" {
				merged.TokenAt = detail.TokenAt
			}
			if merged.Day == "" {
				merged.Day = detail.Day
			}
			if merged.FileType == "" {
				merged.FileType = detail.FileType
			}
			if merged.Width <= 0 {
				merged.Width = detail.Width
			}
			if merged.Height <= 0 {
				merged.Height = detail.Height
			}
			if merged.Duration == nil && detail.Duration != nil {
				duration := *detail.Duration
				merged.Duration = &duration
			}
			if merged.Status == 0 {
				merged.Status = detail.Status
			}
		}
		if merged.Type == "" {
			merged.Type = inferMtPhotoType(merged.FileType)
		}
		out = append(out, merged)
	}
	return out
}

func (s *MtPhotoService) getFolderTimelineFiles(ctx context.Context, folderID int64) (items []MtPhotoFolderFileItem, total int, err error) {
	if !s.configured() {
		return nil, 0, fmt.Errorf("mtPhoto 未配置")
	}
	if folderID <= 0 {
		return nil, 0, fmt.Errorf("folderId 非法")
	}

	urlStr, err := s.buildURL(fmt.Sprintf("/gateway/folderFiles/%d", folderID), nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := s.doRequest(ctx, http.MethodGet, urlStr, map[string]string{
		"Accept": "application/json",
	}, nil, true, false)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, &mtPhotoStatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Action:     "mtPhoto 获取目录时间线内容失败",
		}
	}

	var parsed mtPhotoFolderFilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, 0, fmt.Errorf("mtPhoto 目录时间线响应解析失败: %w", err)
	}

	mapped := mapMtPhotoTimelineFiles(parsed.Result)
	total = parsed.TotalCount
	if total <= 0 {
		total = len(mapped)
	}
	return mapped, total, nil
}

func (s *MtPhotoService) GetFolderContentPage(
	ctx context.Context,
	folderID int64,
	page, pageSize int,
	includeTimeline bool,
) (content *MtPhotoFolderContent, total int, totalPages int, err error) {
	if folderID <= 0 {
		return nil, 0, 0, fmt.Errorf("folderId 非法")
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 60
	}
	if pageSize > 200 {
		pageSize = 200
	}

	content, err = s.getFolderData(ctx, fmt.Sprintf("/gateway/foldersV2/%d", folderID), "mtPhoto 获取目录内容失败")
	if err != nil {
		return nil, 0, 0, err
	}

	total = len(content.FileList)
	if includeTimeline {
		timelineItems, _, timelineErr := s.getFolderTimelineFiles(ctx, folderID)
		if timelineErr == nil && len(timelineItems) > 0 {
			content.FileList = mergeFolderTimelineWithDetail(timelineItems, content.FileList)
			total = len(content.FileList)
		}
	}

	totalPages = calcTotalPages(total, pageSize)
	content.FileList = paginateMtPhotoFolderItems(content.FileList, page, pageSize)
	return content, total, totalPages, nil
}

type mtPhotoAlbumFilesV2Response struct {
	Result     []mtPhotoAlbumFilesV2Group `json:"result"`
	TotalCount int                        `json:"totalCount"`
	Ver        int                        `json:"ver"`
}

type mtPhotoAlbumFilesV2Group struct {
	Day  string            `json:"day"`
	Addr string            `json:"addr"`
	List []mtPhotoFileItem `json:"list"`
}

type mtPhotoFileItem struct {
	ID       int64   `json:"id"`
	Status   int     `json:"status"`
	FileType string  `json:"fileType"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	MD5      string  `json:"MD5"`
	VMD5     string  `json:"VMD5"`
}

type MtPhotoMediaItem struct {
	ID       int64   `json:"id"`
	MD5      string  `json:"md5"`
	FileType string  `json:"fileType"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Day      string  `json:"day,omitempty"`
	Type     string  `json:"type"` // image/video
}

type mtPhotoFolderFilesResponse struct {
	Result     []mtPhotoFolderFilesGroup `json:"result"`
	TotalCount int                       `json:"totalCount"`
	Ver        int                       `json:"ver"`
}

type mtPhotoFolderFilesGroup struct {
	Day  string            `json:"day"`
	Addr string            `json:"addr"`
	List []mtPhotoFileItem `json:"list"`
}

func inferMtPhotoType(fileType string) string {
	switch strings.ToUpper(strings.TrimSpace(fileType)) {
	case "MP4":
		return "video"
	default:
		return "image"
	}
}

func (s *MtPhotoService) GetAlbumFilesPage(ctx context.Context, albumID, page, pageSize int) (items []MtPhotoMediaItem, total int, totalPages int, err error) {
	if !s.configured() {
		return nil, 0, 0, fmt.Errorf("mtPhoto 未配置")
	}
	if albumID <= 0 {
		return nil, 0, 0, fmt.Errorf("albumId 非法")
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 60
	}
	if pageSize > 200 {
		pageSize = 200
	}

	now := time.Now()
	s.mu.Lock()
	if cached, ok := s.albumFilesCache[albumID]; ok && now.Before(cached.expireAt) && len(cached.items) > 0 {
		total = cached.total
		all := cached.items
		s.mu.Unlock()
		return paginateMtPhotoItems(all, total, page, pageSize), total, calcTotalPages(total, pageSize), nil
	}
	s.mu.Unlock()

	query := url.Values{}
	query.Set("listVer", "v2")
	urlStr, err := s.buildURL(fmt.Sprintf("/api-album/filesV2/%d", albumID), query)
	if err != nil {
		return nil, 0, 0, err
	}

	resp, err := s.doRequest(ctx, http.MethodGet, urlStr, map[string]string{
		"Accept": "application/json",
	}, nil, true, false)
	if err != nil {
		return nil, 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, 0, fmt.Errorf("mtPhoto 获取相册媒体失败: %s", resp.Status)
	}

	var parsed mtPhotoAlbumFilesV2Response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, 0, 0, err
	}

	all := make([]MtPhotoMediaItem, 0, parsed.TotalCount)
	for _, g := range parsed.Result {
		day := strings.TrimSpace(g.Day)
		for _, f := range g.List {
			md5Val := strings.TrimSpace(f.MD5)
			if md5Val == "" {
				continue
			}
			all = append(all, MtPhotoMediaItem{
				ID:       f.ID,
				MD5:      md5Val,
				FileType: f.FileType,
				Width:    f.Width,
				Height:   f.Height,
				Duration: f.Duration,
				Day:      day,
				Type:     inferMtPhotoType(f.FileType),
			})
		}
	}
	total = parsed.TotalCount
	if total <= 0 {
		total = len(all)
	}

	s.mu.Lock()
	s.albumFilesCache[albumID] = mtPhotoAlbumFilesCacheEntry{
		expireAt: time.Now().Add(2 * time.Minute),
		total:    total,
		items:    all,
	}
	s.mu.Unlock()

	return paginateMtPhotoItems(all, total, page, pageSize), total, calcTotalPages(total, pageSize), nil
}

func calcTotalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	if total <= 0 {
		return 0
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	return pages
}

func paginateMtPhotoItems(all []MtPhotoMediaItem, total, page, pageSize int) []MtPhotoMediaItem {
	if len(all) == 0 || pageSize <= 0 {
		return []MtPhotoMediaItem{}
	}

	start := (page - 1) * pageSize
	if start >= len(all) {
		return []MtPhotoMediaItem{}
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}

	out := make([]MtPhotoMediaItem, 0, end-start)
	out = append(out, all[start:end]...)
	_ = total
	return out
}

type mtPhotoFilesInMD5Request struct {
	MD5 string `json:"MD5"`
}

type MtPhotoFilePath struct {
	ID       int64  `json:"id"`
	FilePath string `json:"filePath"`
}

func (s *MtPhotoService) ResolveFilePath(ctx context.Context, md5Value string) (*MtPhotoFilePath, error) {
	if !s.configured() {
		return nil, fmt.Errorf("mtPhoto 未配置")
	}

	md5Value = strings.TrimSpace(md5Value)
	if md5Value == "" {
		return nil, fmt.Errorf("md5 为空")
	}

	urlStr, err := s.buildURL("/gateway/filesInMD5", nil)
	if err != nil {
		return nil, err
	}

	body, _ := json.Marshal(mtPhotoFilesInMD5Request{MD5: md5Value})
	resp, err := s.doRequest(ctx, http.MethodPost, urlStr, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}, body, true, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mtPhoto 查询文件路径失败: %s", resp.Status)
	}

	var parsed []MtPhotoFilePath
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed) == 0 || strings.TrimSpace(parsed[0].FilePath) == "" {
		return nil, fmt.Errorf("mtPhoto 未返回文件路径")
	}

	item := parsed[0]
	item.FilePath = strings.TrimSpace(item.FilePath)
	return &item, nil
}

// GatewayGet 用于代理 mtPhoto /gateway/{size}/{md5}。
// 返回 resp 由调用方负责关闭。
func (s *MtPhotoService) GatewayGet(ctx context.Context, size, md5Value string) (*http.Response, error) {
	if !s.configured() {
		return nil, fmt.Errorf("mtPhoto 未配置")
	}

	size = strings.TrimSpace(size)
	md5Value = strings.TrimSpace(md5Value)
	if size == "" || md5Value == "" {
		return nil, fmt.Errorf("参数为空")
	}

	urlStr, err := s.buildURL(fmt.Sprintf("/gateway/%s/%s", size, md5Value), nil)
	if err != nil {
		return nil, err
	}

	return s.doRequest(ctx, http.MethodGet, urlStr, map[string]string{
		"Accept": "*/*",
	}, nil, true, true)
}

// GatewayFileDownload 用于代理 mtPhoto /gateway/fileDownload/{id}/{md5}（下载原图/原文件）。
// 返回 resp 由调用方负责关闭。
func (s *MtPhotoService) GatewayFileDownload(ctx context.Context, fileID int64, md5Value string) (*http.Response, error) {
	if !s.configured() {
		return nil, fmt.Errorf("mtPhoto 未配置")
	}

	if fileID <= 0 {
		return nil, fmt.Errorf("id 非法")
	}

	md5Value = strings.TrimSpace(md5Value)
	if md5Value == "" {
		return nil, fmt.Errorf("md5 为空")
	}

	urlStr, err := s.buildURL(fmt.Sprintf("/gateway/fileDownload/%d/%s", fileID, md5Value), nil)
	if err != nil {
		return nil, err
	}

	return s.doRequest(ctx, http.MethodGet, urlStr, map[string]string{
		"Accept": "*/*",
	}, nil, true, true)
}
