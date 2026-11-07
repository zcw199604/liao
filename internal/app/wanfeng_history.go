package app

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"liao/internal/upstream/wanfeng"
)

// 三期：让「历史会话 / 联系人 / 历史消息」在老接口上按线路分流。
//
// 老接口（/api/getHistoryUserList 等）是前端既有的调用方式，返回结构不能变；
// 晚风线路的差异全部在这里翻译，前端零改动。

const (
	// 单次历史消息接口最多返回多少条（与 chat2019 默认分页对齐）。
	wanfengHistoryPageSize = 20
	// 历史接口超时：比聊天本身宽松，避免首屏被对方慢响应拖住。
	wanfengHistoryTimeout = 20 * time.Second
)

// wanfengIdentityUID 返回该身份在晚风站点的 uid。
//
// 判定依据是 identity_provider_binding：一个 liao 身份只会绑定一条线路
// （绑过晚风的身份，历史数据就在晚风侧），因此不需要前端把当前线路传上来。
func (a *App) wanfengIdentityUID(ctx context.Context, identityID string) (string, bool) {
	identityID = strings.TrimSpace(identityID)
	if a == nil || a.wanfeng == nil || identityID == "" {
		return "", false
	}
	binding, err := a.wanfeng.BindingOf(ctx, identityID)
	if err != nil {
		slog.Warn("读取晚风身份绑定失败，回退 chat2019 历史接口", "identityID", identityID, "error", err)
		return "", false
	}
	if binding == nil || strings.TrimSpace(binding.UpstreamUID) == "" {
		return "", false
	}
	return binding.UpstreamUID, true
}

// wanfengHistoryUser 是「历史/收藏列表」里的一项（兼容前端 normalizeUser 的字段）。
type wanfengHistoryUser struct {
	ID            string `json:"id"`
	Nickname      string `json:"nickname"`
	Sex           string `json:"sex"`
	Age           string `json:"age"`
	Address       string `json:"address"`
	IP            string `json:"ip"`
	LastMsg       string `json:"lastMsg"`
	LastTime      string `json:"lastTime"`
	UnreadCount   int    `json:"unreadCount"`
	LocalArchived bool   `json:"localArchived,omitempty"`
	// 供前端排序/时间线使用
	LastMessageAt string `json:"lastMessageAt,omitempty"`
	Pinned        bool   `json:"pinned,omitempty"`
}

// buildWanfengHistoryUsers 把晚风 relations 映射成前端「历史用户列表」结构。
func buildWanfengHistoryUsers(relations []wanfeng.Relation, selfUID, selfName string) []wanfengHistoryUser {
	out := make([]wanfengHistoryUser, 0, len(relations))
	for _, rel := range relations {
		if rel.TargetUID == "" || rel.TargetUID == selfUID {
			continue
		}
		nickname := strings.TrimSpace(rel.TargetNickname)
		if nickname == "" {
			nickname = "匿名"
		}
		out = append(out, wanfengHistoryUser{
			ID:            rel.TargetUID,
			Nickname:      nickname,
			Sex:           wanfeng.SexFromUpstream(rel.TargetGender),
			Age:           formatOptionalAge(rel.TargetAge),
			Address:       "未知",
			LastMsg:       "暂无消息",
			LastTime:      formatHistoryTime(rel.LastMessageAt),
			UnreadCount:   rel.UnreadCount,
			LastMessageAt: rel.LastMessageAt,
			Pinned:        rel.Pinned,
		})
	}
	_ = selfName
	return out
}

// buildWanfengFavoriteUsers 把晚风收藏映射成前端「收藏用户列表」结构。
func buildWanfengFavoriteUsers(favorites []wanfeng.Favorite) []wanfengHistoryUser {
	out := make([]wanfengHistoryUser, 0, len(favorites))
	for _, fav := range favorites {
		if fav.UID == "" {
			continue
		}
		nickname := strings.TrimSpace(fav.Nickname)
		if nickname == "" {
			nickname = "匿名"
		}
		out = append(out, wanfengHistoryUser{
			ID:            fav.UID,
			Nickname:      nickname,
			Sex:           wanfeng.SexFromUpstream(fav.Gender),
			Age:           formatOptionalAge(fav.Age),
			Address:       "未知",
			LastMsg:       "暂无消息",
			LastTime:      formatHistoryTime(fav.LastMessageAt),
			LastMessageAt: fav.LastMessageAt,
			Pinned:        fav.Pinned,
		})
	}
	return out
}

// formatOptionalAge 0 岁显示为 0（前端按字符串显示，缺省用 "0"）。
func formatOptionalAge(age int) string {
	if age <= 0 {
		return "0"
	}
	return strconv.Itoa(age)
}

// formatHistoryTime 把对方的时间串转成前端列表用的「刚刚 / 时间」文案。
//
// 前端只在 lastTime 上做展示，不参与逻辑，因此这里保持简单：
// 能解析成时间就按「今天/昨天/日期」显示，否则原样回传。
func formatHistoryTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "刚刚"
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return humanizeHistoryTime(t)
		}
	}
	return raw
}

// humanizeHistoryTime 把时间转成「刚刚 / N分钟前 / 今天 HH:MM / 昨天 HH:MM / MM-DD」。
func humanizeHistoryTime(t time.Time) string {
	local := t.Local()
	now := time.Now()
	diff := now.Sub(local)
	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return strconv.Itoa(int(diff.Minutes())) + "分钟前"
	}

	sameDay := local.Year() == now.Year() && local.YearDay() == now.YearDay()
	if sameDay {
		return "今天 " + local.Format("15:04")
	}
	yesterday := now.AddDate(0, 0, -1)
	if local.Year() == yesterday.Year() && local.YearDay() == yesterday.YearDay() {
		return "昨天 " + local.Format("15:04")
	}
	return local.Format("01-02")
}

// wanfengHistoryContents 是前端 getMessageHistory 期望的结构。
type wanfengHistoryContents struct {
	Code         int                     `json:"code"`
	ContentsList []wanfengHistoryContent `json:"contents_list"`
	HasMore      bool                    `json:"hasMore"`
}

// wanfengHistoryContent 是历史消息里的一条（字段名对齐 chat2019 上游返回）。
type wanfengHistoryContent struct {
	// Content 是消息正文：文本是文字，媒体是 "[远程文件名]"。
	Content  string `json:"content"`
	Tid      string `json:"Tid"`
	Time     string `json:"time"`
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Type     string `json:"type,omitempty"`
	LocalTid string `json:"localTid,omitempty"`
}

// buildWanfengHistoryContents 把晚风历史消息翻译成前端结构。
//
// selfUID 是自己在晚风的 uid，selfIDMD5 是前端认的「自己」标识（md5(身份ID)）；
// 前端判定 isSelf 的依据是 content.id 是否等于对方 id，所以自己发的消息要填 selfIDMD5。
func (a *App) buildWanfengHistoryContents(
	ctx context.Context,
	identityID, selfUID string,
	peerUID string,
	page *wanfeng.ChatHistoryPage,
) wanfengHistoryContents {
	contents := wanfengHistoryContents{
		Code:         0,
		ContentsList: make([]wanfengHistoryContent, 0, len(page.Messages)),
		HasMore:      page.HasMore,
	}
	if page == nil {
		return contents
	}

	selfIDMD5 := md5HexLower(identityID)
	for _, msg := range page.Messages {
		content := strings.TrimSpace(msg.Payload)
		if content == "" {
			continue
		}

		// 媒体消息：对方给的是 URL，这里搬回我方媒体服务器换成 "[文件名]"，
		// 这样前端的历史渲染与实时消息走同一条链路。
		if msg.Type == wanfeng.MediaKindImage || msg.Type == wanfeng.MediaKindVideo || msg.Type == wanfeng.MediaKindAudio {
			if bridge := a.wanfengMedia; bridge != nil {
				if name, err := bridge.MirrorInbound(identityID, msg.Type, content); err == nil && strings.TrimSpace(name) != "" {
					content = wanfeng.MediaContent(name)
				} else {
					if err != nil {
						slog.Warn("历史消息媒体转存失败", "identityID", identityID, "error", err)
					}
					// 转存失败就跳过该条，避免前端渲染出空图/裂图
					continue
				}
			} else {
				continue
			}
		} else {
			// 文本消息：对方直接给文字
			content = msg.Payload
		}

		isSelf := strings.EqualFold(msg.FromCid, "self") ||
			(selfUID != "" && strings.EqualFold(msg.FromUID, selfUID))

		senderID := peerUID
		nickname := msg.Nickname
		if isSelf {
			senderID = selfIDMD5
			nickname = ""
		} else if senderID == "" {
			senderID = msg.FromUID
		}
		if nickname == "" && !isSelf {
			nickname = "匿名"
		}

		contents.ContentsList = append(contents.ContentsList, wanfengHistoryContent{
			Content:  content,
			Tid:      msg.Tid,
			Time:     formatWanfengMessageTime(msg.TS),
			ID:       senderID,
			Nickname: nickname,
			Type:     msg.Type,
		})
	}
	return contents
}

// formatWanfengMessageTime 把对方毫秒/秒时间戳格式化成前端用的 "2006-01-02 15:04:05.000"。
func formatWanfengMessageTime(ts int) string {
	if ts <= 0 {
		return time.UnixMilli(0).Format("2006-01-02 15:04:05.000")
	}
	v := int64(ts)
	if v < 1e12 {
		v *= 1000
	}
	return time.UnixMilli(v).Format("2006-01-02 15:04:05.000")
}

// serveWanfengHistoryUsers 在晚风线路上直接写出「历史/收藏用户列表」响应。
//
// 返回 true 表示已处理（调用方应直接 return）；false 表示该身份不属于晚风线路，
// 继续走原有 chat2019 上游，保证两条线路互不影响。
func (a *App) serveWanfengHistoryUsers(w http.ResponseWriter, r *http.Request, identityID string, favoritesOnly bool) bool {
	ctx, cancel := context.WithTimeout(r.Context(), wanfengHistoryTimeout)
	defer cancel()

	selfUID, ok := a.wanfengIdentityUID(ctx, identityID)
	if !ok {
		return false
	}

	api := a.wanfeng.API()
	if api == nil {
		return false
	}

	if favoritesOnly {
		list, _, err := api.Favorites(ctx, selfUID, 1, wanfengHistoryPageSize)
		if err != nil {
			slog.Warn("晚风收藏列表获取失败", "identityID", identityID, "error", err)
			// 失败时返回空列表而不是 5xx，避免前端整块报错
			writeJSON(w, http.StatusOK, []wanfengHistoryUser{})
			return true
		}
		writeJSON(w, http.StatusOK, buildWanfengFavoriteUsers(list))
		return true
	}

	home, err := api.HomeData(ctx, selfUID, wanfengHistoryPageSize)
	if err != nil {
		slog.Warn("晚风历史会话获取失败", "identityID", identityID, "error", err)
		writeJSON(w, http.StatusOK, []wanfengHistoryUser{})
		return true
	}
	writeJSON(w, http.StatusOK, buildWanfengHistoryUsers(home.Relations, selfUID, ""))
	return true
}

// serveWanfengMessageHistory 在晚风线路上直接写出聊天记录响应。
//
// 返回 true 表示已处理；false 表示该身份不属于晚风线路。
func (a *App) serveWanfengMessageHistory(w http.ResponseWriter, r *http.Request, identityID, peerUID, firstTid string) bool {
	ctx, cancel := context.WithTimeout(r.Context(), wanfengHistoryTimeout)
	defer cancel()

	selfUID, ok := a.wanfengIdentityUID(ctx, identityID)
	if !ok {
		return false
	}

	api := a.wanfeng.API()
	if api == nil {
		return false
	}

	// firstTid 是翻页游标：前端传 "0"/"" 表示最新一页。
	before := strings.TrimSpace(firstTid)
	if before == "0" {
		before = ""
	}

	page, err := api.ChatMessages(ctx, selfUID, strings.TrimSpace(peerUID), wanfengHistoryPageSize, before)
	if err != nil {
		slog.Warn("晚风聊天记录获取失败", "identityID", identityID, "peer", peerUID, "error", err)
		writeJSON(w, http.StatusOK, wanfengHistoryContents{Code: 0, ContentsList: []wanfengHistoryContent{}})
		return true
	}

	contents := a.buildWanfengHistoryContents(ctx, identityID, selfUID, strings.TrimSpace(peerUID), page)

	// 同步一份到聊天记录缓存，保证与 chat2019 线路一致的「离线可见」体验
	a.archiveWanfengHistory(ctx, identityID, peerUID, contents)

	writeJSON(w, http.StatusOK, contents)
	return true
}

// archiveWanfengHistory 把晚风历史消息写进聊天记录缓存（失败只记日志，不影响响应）。
func (a *App) archiveWanfengHistory(ctx context.Context, identityID, peerUID string, contents wanfengHistoryContents) {
	if a.chatHistoryCache == nil || len(contents.ContentsList) == 0 {
		return
	}
	conversationKey := generateConversationKey(strings.TrimSpace(identityID), strings.TrimSpace(peerUID))
	if conversationKey == "" {
		return
	}

	msgs := make([]map[string]any, 0, len(contents.ContentsList))
	for _, item := range contents.ContentsList {
		msgs = append(msgs, map[string]any{
			"content":  item.Content,
			"Tid":      item.Tid,
			"tid":      item.Tid,
			"time":     item.Time,
			"id":       item.ID,
			"nickname": item.Nickname,
		})
	}
	a.chatHistoryCache.SaveMessages(ctx, conversationKey, msgs)

	// 最后一条消息用于会话列表的「最近消息」展示
	if a.userArchive != nil && len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		a.userArchive.SaveLastMessage(ctx, identityID, peerUID,
			toString(last["content"]), toString(last["time"]))
	}
}

// serveWanfengFavorite 在晚风线路上处理「收藏/取消收藏」。
//
// 返回 true 表示已处理。收藏在对方是 WS 动作（add_favorite / remove_favorite），
// 同时本地也写一条，保证「收藏列表」立即可见（对方列表有缓存延迟）。
func (a *App) serveWanfengFavorite(w http.ResponseWriter, r *http.Request, identityID, targetUID string, add bool) bool {
	ctx, cancel := context.WithTimeout(r.Context(), wanfengHistoryTimeout)
	defer cancel()

	if _, ok := a.wanfengIdentityUID(ctx, identityID); !ok {
		return false
	}

	proto := a.wanfeng.Protocol()
	action := proto.ActionAddFavorite
	if !add {
		action = proto.ActionRemoveFavorite
	}

	if err := a.wanfeng.SendAction(identityID, action, map[string]any{"targetUid": targetUID}); err != nil {
		slog.Warn("晚风收藏动作下发失败", "identityID", identityID, "action", action, "error", err)
		// 对方连接没就绪时先落本地，用户不会看到「收藏失败」
	}

	// 本地收藏表同步（前端刷新收藏列表会立刻看到变化）
	if a.favoriteService == nil {
		writeJSON(w, http.StatusOK, map[string]any{"code": "0", "status": "true", "msg": "success"})
		return true
	}
	if add {
		if _, err := a.favoriteService.Add(ctx, identityID, targetUID, ""); err != nil {
			slog.Debug("本地收藏写入失败", "identityID", identityID, "error", err)
		}
	} else {
		if err := a.favoriteService.Remove(ctx, identityID, targetUID); err != nil {
			slog.Debug("本地收藏删除失败", "identityID", identityID, "error", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": "0", "status": "true", "msg": "success"})
	return true
}
