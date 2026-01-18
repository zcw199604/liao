package app

// userInfoCacheWithUserList 将 UserInfoCacheService 的写入同步到用户列表本地缓存。
// 用途：减少上游调用后，仍尽量保持列表 lastMsg/lastTime 的新鲜度。
type userInfoCacheWithUserList struct {
	inner         UserInfoCacheService
	userListCache *userListCache
}

func wrapUserInfoCacheWithUserList(inner UserInfoCacheService, userListCache *userListCache) UserInfoCacheService {
	if inner == nil || userListCache == nil {
		return inner
	}
	return &userInfoCacheWithUserList{
		inner:         inner,
		userListCache: userListCache,
	}
}

func (c *userInfoCacheWithUserList) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	if closer, ok := c.inner.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

func (c *userInfoCacheWithUserList) SaveUserInfo(info CachedUserInfo) {
	if c == nil || c.inner == nil {
		return
	}
	c.inner.SaveUserInfo(info)
	// 用户信息更新涉及的列表归属不明确（可能出现在多个 myUserID 的列表中），
	// 为避免 O(N) 扫描，这里不主动刷新用户列表缓存；列表接口缓存 miss 时会重新增强并更新本地缓存。
}

func (c *userInfoCacheWithUserList) GetUserInfo(userID string) *CachedUserInfo {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.GetUserInfo(userID)
}

func (c *userInfoCacheWithUserList) EnrichUserInfo(userID string, originalData map[string]any) map[string]any {
	if c == nil || c.inner == nil {
		return originalData
	}
	return c.inner.EnrichUserInfo(userID, originalData)
}

func (c *userInfoCacheWithUserList) BatchEnrichUserInfo(userList []map[string]any, userIDKey string) []map[string]any {
	if c == nil || c.inner == nil {
		return userList
	}
	return c.inner.BatchEnrichUserInfo(userList, userIDKey)
}

func (c *userInfoCacheWithUserList) SaveLastMessage(message CachedLastMessage) {
	if c == nil || c.inner == nil {
		return
	}
	c.inner.SaveLastMessage(message)
	if c.userListCache != nil {
		c.userListCache.UpdateLastMessage(message)
	}
}

func (c *userInfoCacheWithUserList) GetLastMessage(myUserID, otherUserID string) *CachedLastMessage {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.GetLastMessage(myUserID, otherUserID)
}

func (c *userInfoCacheWithUserList) BatchEnrichWithLastMessage(userList []map[string]any, myUserID string) []map[string]any {
	if c == nil || c.inner == nil {
		return userList
	}
	return c.inner.BatchEnrichWithLastMessage(userList, myUserID)
}

func (c *userInfoCacheWithUserList) batchGetUserInfo(userIDs []string) map[string]CachedUserInfo {
	if c == nil || c.inner == nil {
		return map[string]CachedUserInfo{}
	}
	if batch, ok := c.inner.(interface {
		batchGetUserInfo([]string) map[string]CachedUserInfo
	}); ok {
		return batch.batchGetUserInfo(userIDs)
	}
	result := make(map[string]CachedUserInfo, len(userIDs))
	for _, uid := range userIDs {
		if info := c.inner.GetUserInfo(uid); info != nil {
			result[uid] = *info
		}
	}
	return result
}

func (c *userInfoCacheWithUserList) batchGetLastMessages(conversationKeys []string) map[string]CachedLastMessage {
	if c == nil || c.inner == nil {
		return map[string]CachedLastMessage{}
	}
	if batch, ok := c.inner.(interface {
		batchGetLastMessages([]string) map[string]CachedLastMessage
	}); ok {
		return batch.batchGetLastMessages(conversationKeys)
	}
	// 无法从 conversationKey 反推出 GetLastMessage 所需的 my/other 维度，这里兜底返回空。
	return map[string]CachedLastMessage{}
}
