package app

import (
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type CachedUserInfo struct {
	UserID     string `json:"userId"`
	Nickname   string `json:"nickname"`
	Gender     string `json:"gender"`
	Age        string `json:"age"`
	Address    string `json:"address"`
	UpdateTime int64  `json:"updateTime"`
}

type CachedLastMessage struct {
	ConversationKey string `json:"conversationKey"`
	FromUserID      string `json:"fromUserId"`
	ToUserID        string `json:"toUserId"`
	Content         string `json:"content"`
	Type            string `json:"type"`
	Time            string `json:"time"`
	UpdateTime      int64  `json:"updateTime"`
}

func generateConversationKey(userID1, userID2 string) string {
	if userID1 == "" || userID2 == "" {
		return ""
	}
	if userID1 < userID2 {
		return userID1 + "_" + userID2
	}
	return userID2 + "_" + userID1
}

type UserInfoCacheService interface {
	SaveUserInfo(info CachedUserInfo)
	GetUserInfo(userID string) *CachedUserInfo
	EnrichUserInfo(userID string, originalData map[string]any) map[string]any
	BatchEnrichUserInfo(userList []map[string]any, userIDKey string) []map[string]any

	SaveLastMessage(message CachedLastMessage)
	GetLastMessage(myUserID, otherUserID string) *CachedLastMessage
	BatchEnrichWithLastMessage(userList []map[string]any, myUserID string) []map[string]any
}

// MemoryUserInfoCacheService 对齐 Java 的 MemoryUserInfoCacheService 行为。
type MemoryUserInfoCacheService struct {
	mu               sync.RWMutex
	userInfo         map[string]CachedUserInfo
	lastMessageByKey map[string]CachedLastMessage
}

func (s *MemoryUserInfoCacheService) batchGetUserInfo(userIDs []string) map[string]CachedUserInfo {
	result := make(map[string]CachedUserInfo, len(userIDs))
	if s == nil || len(userIDs) == 0 {
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, uid := range userIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" {
			continue
		}
		if info, ok := s.userInfo[uid]; ok {
			result[uid] = info
		}
	}
	return result
}

func (s *MemoryUserInfoCacheService) batchGetLastMessages(conversationKeys []string) map[string]CachedLastMessage {
	result := make(map[string]CachedLastMessage, len(conversationKeys))
	if s == nil || len(conversationKeys) == 0 {
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, key := range conversationKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if msg, ok := s.lastMessageByKey[key]; ok {
			result[key] = msg
		}
	}
	return result
}

func NewMemoryUserInfoCacheService() *MemoryUserInfoCacheService {
	return &MemoryUserInfoCacheService{
		userInfo:         make(map[string]CachedUserInfo),
		lastMessageByKey: make(map[string]CachedLastMessage),
	}
}

func (s *MemoryUserInfoCacheService) SaveUserInfo(info CachedUserInfo) {
	if strings.TrimSpace(info.UserID) == "" {
		return
	}
	info.UpdateTime = time.Now().UnixMilli()
	s.mu.Lock()
	s.userInfo[info.UserID] = info
	s.mu.Unlock()
}

func (s *MemoryUserInfoCacheService) GetUserInfo(userID string) *CachedUserInfo {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	s.mu.RLock()
	info, ok := s.userInfo[userID]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	cp := info
	return &cp
}

func (s *MemoryUserInfoCacheService) EnrichUserInfo(userID string, originalData map[string]any) map[string]any {
	info := s.GetUserInfo(userID)
	if info == nil {
		return originalData
	}
	putIfAbsent(originalData, "nickname", info.Nickname)
	putIfAbsent(originalData, "sex", info.Gender)
	putIfAbsent(originalData, "age", info.Age)
	putIfAbsent(originalData, "address", info.Address)
	return originalData
}

func (s *MemoryUserInfoCacheService) BatchEnrichUserInfo(userList []map[string]any, userIDKey string) []map[string]any {
	if userList == nil {
		return []map[string]any{}
	}
	for _, m := range userList {
		if m == nil {
			continue
		}
		raw, ok := m[userIDKey]
		if !ok || raw == nil {
			continue
		}
		uid := strings.TrimSpace(toString(raw))
		if uid == "" {
			continue
		}
		s.EnrichUserInfo(uid, m)
	}
	return userList
}

func (s *MemoryUserInfoCacheService) SaveLastMessage(message CachedLastMessage) {
	key := generateConversationKey(message.FromUserID, message.ToUserID)
	if strings.TrimSpace(key) == "" {
		return
	}
	message.ConversationKey = key
	message.UpdateTime = time.Now().UnixMilli()

	s.mu.Lock()
	s.lastMessageByKey[key] = message
	s.mu.Unlock()
}

func (s *MemoryUserInfoCacheService) GetLastMessage(myUserID, otherUserID string) *CachedLastMessage {
	key := generateConversationKey(strings.TrimSpace(myUserID), strings.TrimSpace(otherUserID))
	if key == "" {
		return nil
	}
	s.mu.RLock()
	msg, ok := s.lastMessageByKey[key]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	cp := msg
	return &cp
}

func (s *MemoryUserInfoCacheService) BatchEnrichWithLastMessage(userList []map[string]any, myUserID string) []map[string]any {
	if userList == nil {
		return []map[string]any{}
	}
	for _, user := range userList {
		otherUserID := extractUserID(user)
		if strings.TrimSpace(otherUserID) == "" {
			continue
		}
		last := s.GetLastMessage(myUserID, otherUserID)
		if last == nil {
			continue
		}
		display := formatLastMessage(*last, myUserID)
		putIfAbsent(user, "lastMsg", display)
		putIfAbsent(user, "lastTime", formatTime(last.Time))
	}
	return userList
}

func extractUserID(user map[string]any) string {
	if user == nil {
		return ""
	}
	for _, key := range []string{"id", "UserID", "userid", "userId"} {
		if v, ok := user[key]; ok && v != nil {
			return toString(v)
		}
	}
	return ""
}

func putIfAbsent(m map[string]any, key string, value any) {
	if m == nil {
		return
	}
	if value == nil {
		return
	}
	existing, ok := m[key]
	if ok && existing != nil {
		return
	}
	m[key] = value
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(fmtAny(v)), "\u0000", ""))
	}
}

func fmtAny(v any) string {
	// 避免引入 fmt 导致大量格式化开销，这里仅覆盖常见类型。
	switch t := v.(type) {
	case []byte:
		return string(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		// JSON 数字默认 float64
		return strconv.FormatInt(int64(t), 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func formatLastMessage(msg CachedLastMessage, myUserID string) string {
	prefix := ""
	if msg.FromUserID == myUserID {
		prefix = "我: "
	}
	content := msg.Content
	if content == "" {
		return prefix + "[消息]"
	}

	hasImage := false
	hasVideo := false
	hasAudio := false
	hasFile := false

	var textBuilder strings.Builder
	idx := 0
	for idx < len(content) {
		open := strings.Index(content[idx:], "[")
		if open < 0 {
			textBuilder.WriteString(content[idx:])
			break
		}
		open += idx
		close := strings.Index(content[open+1:], "]")
		if close < 0 {
			textBuilder.WriteString(content[idx:])
			break
		}
		close += open + 1

		if open > idx {
			textBuilder.WriteString(content[idx:open])
		}

		token := content[open : close+1]
		kind := inferMediaKindFromBracketBody(content[open+1 : close])
		if kind == "" {
			// 非媒体（如 [doge] 或普通方括号文本）按原样保留
			textBuilder.WriteString(token)
		} else {
			switch kind {
			case "image":
				hasImage = true
			case "video":
				hasVideo = true
			case "audio":
				hasAudio = true
			default:
				hasFile = true
			}
		}

		idx = close + 1
	}

	label := ""
	switch {
	case hasImage:
		label = "[图片]"
	case hasVideo:
		label = "[视频]"
	case hasAudio:
		label = "[音频]"
	case hasFile:
		label = "[文件]"
	}

	text := strings.TrimSpace(textBuilder.String())
	if text != "" {
		text = truncateRunes(text, 30)
	}

	if text == "" {
		if label != "" {
			return prefix + label
		}
		return prefix + "[消息]"
	}
	if label == "" {
		return prefix + text
	}
	return prefix + text + " " + label
}

func truncateRunes(input string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(input)
	if len(runes) <= max {
		return input
	}
	return string(runes[:max]) + "..."
}

func inferMediaKindFromBracketBody(body string) string {
	path := strings.TrimSpace(body)
	if path == "" {
		return ""
	}

	// 避免把 URL 或带空白的内容误判为上传路径
	if strings.Contains(path, "://") {
		return ""
	}
	if strings.IndexFunc(path, unicode.IsSpace) >= 0 {
		return ""
	}

	lower := strings.ToLower(path)
	if i := strings.IndexByte(lower, '?'); i >= 0 {
		lower = lower[:i]
	}
	if i := strings.IndexByte(lower, '#'); i >= 0 {
		lower = lower[:i]
	}

	switch {
	case strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".gif") || strings.HasSuffix(lower, ".bmp") || strings.HasSuffix(lower, ".webp"):
		return "image"
	case strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".webm") || strings.HasSuffix(lower, ".ogg") ||
		strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".avi") || strings.HasSuffix(lower, ".mkv") ||
		strings.HasSuffix(lower, ".wmv") || strings.HasSuffix(lower, ".flv"):
		return "video"
	case strings.HasSuffix(lower, ".mp3") || strings.HasSuffix(lower, ".wav") || strings.HasSuffix(lower, ".aac") || strings.HasSuffix(lower, ".flac"):
		return "audio"
	}

	dot := strings.LastIndexByte(lower, '.')
	if dot < 0 {
		return ""
	}
	ext := lower[dot+1:]
	if !looksLikeFileExt(ext) {
		return ""
	}
	return "file"
}

func looksLikeFileExt(ext string) bool {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		return false
	}
	if len(ext) > 10 {
		return false
	}

	hasLetter := false
	for _, r := range ext {
		if r >= 'a' && r <= 'z' {
			hasLetter = true
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return hasLetter
}

func formatTime(timeStr string) string {
	if strings.TrimSpace(timeStr) == "" {
		return "刚刚"
	}
	return timeStr
}
