package app

import (
	"strings"
	"sync"
	"time"
)

type userInfoBatchGetter interface {
	batchGetUserInfo(userIDs []string) map[string]CachedUserInfo
	batchGetLastMessages(conversationKeys []string) map[string]CachedLastMessage
}

func enrichUserListInPlace(cache UserInfoCacheService, userList []map[string]any, userIDKey string, myUserID string) (enrichUserInfoMs int64, lastMsgMs int64) {
	batch, ok := cache.(userInfoBatchGetter)
	if !ok || cache == nil || userList == nil || len(userList) == 0 {
		return 0, 0
	}

	userIDs := collectUserIDs(userList, userIDKey)
	conversationKeys := collectConversationKeys(userList, myUserID)

	var (
		infoMap    map[string]CachedUserInfo
		messageMap map[string]CachedLastMessage
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		start := time.Now()
		infoMap = batch.batchGetUserInfo(userIDs)
		enrichUserInfoMs = time.Since(start).Milliseconds()
		wg.Done()
	}()

	go func() {
		start := time.Now()
		messageMap = batch.batchGetLastMessages(conversationKeys)
		lastMsgMs = time.Since(start).Milliseconds()
		wg.Done()
	}()

	wg.Wait()

	applyUserInfo(userList, userIDKey, infoMap)
	applyLastMessage(userList, myUserID, messageMap)
	return enrichUserInfoMs, lastMsgMs
}

func collectUserIDs(userList []map[string]any, userIDKey string) []string {
	if userList == nil || len(userList) == 0 {
		return nil
	}
	userIDs := make([]string, 0, len(userList))
	seen := make(map[string]struct{}, len(userList))
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
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		userIDs = append(userIDs, uid)
	}
	return userIDs
}

func collectConversationKeys(userList []map[string]any, myUserID string) []string {
	if userList == nil || len(userList) == 0 {
		return nil
	}
	myUserID = strings.TrimSpace(myUserID)
	if myUserID == "" {
		return nil
	}
	keys := make([]string, 0, len(userList))
	seen := make(map[string]struct{}, len(userList))
	for _, user := range userList {
		otherUserID := strings.TrimSpace(extractUserID(user))
		if otherUserID == "" {
			continue
		}
		key := generateConversationKey(myUserID, otherUserID)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func applyUserInfo(userList []map[string]any, userIDKey string, infoMap map[string]CachedUserInfo) {
	if userList == nil || len(userList) == 0 || len(infoMap) == 0 {
		return
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
		info, ok := infoMap[uid]
		if !ok {
			continue
		}
		putIfAbsent(m, "nickname", info.Nickname)
		putIfAbsent(m, "sex", info.Gender)
		putIfAbsent(m, "age", info.Age)
		putIfAbsent(m, "address", info.Address)
	}
}

func applyLastMessage(userList []map[string]any, myUserID string, messageMap map[string]CachedLastMessage) {
	if userList == nil || len(userList) == 0 || len(messageMap) == 0 {
		return
	}
	myUserID = strings.TrimSpace(myUserID)
	if myUserID == "" {
		return
	}
	for _, user := range userList {
		otherUserID := strings.TrimSpace(extractUserID(user))
		if otherUserID == "" {
			continue
		}
		key := generateConversationKey(myUserID, otherUserID)
		if key == "" {
			continue
		}
		msg, ok := messageMap[key]
		if !ok {
			continue
		}
		user["lastMsg"] = formatLastMessage(msg, myUserID)
		user["lastTime"] = formatTime(msg.Time)
	}
}
