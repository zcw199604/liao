package app

import (
	"strconv"
	"strings"
	"sync"
	"time"
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
	mu              sync.RWMutex
	userInfo        map[string]CachedUserInfo
	lastMessageByKey map[string]CachedLastMessage
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

	if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
		path := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(content, "["), "]"))
		// 兼容表情文本（如 [doge]）：无路径分隔符且无扩展名时，按普通文本显示
		if !strings.Contains(path, "/") && !strings.Contains(path, "\\") && !strings.Contains(path, ".") {
			return prefix + content
		}
		switch {
		case strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".gif") || strings.HasSuffix(path, ".bmp"):
			return prefix + "[图片]"
		case strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".avi") || strings.HasSuffix(path, ".mov") || strings.HasSuffix(path, ".wmv") || strings.HasSuffix(path, ".flv"):
			return prefix + "[视频]"
		case strings.HasSuffix(path, ".mp3") || strings.HasSuffix(path, ".wav") || strings.HasSuffix(path, ".aac") || strings.HasSuffix(path, ".flac"):
			return prefix + "[音频]"
		default:
			return prefix + "[文件]"
		}
	}

	if len([]rune(content)) > 30 {
		runes := []rune(content)
		return prefix + string(runes[:30]) + "..."
	}
	return prefix + content
}

func formatTime(timeStr string) string {
	if strings.TrimSpace(timeStr) == "" {
		return "刚刚"
	}
	return timeStr
}
