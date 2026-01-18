# User History（用户聊天历史）

## 目的
提供历史用户列表与消息历史代理，并在本地补充缓存信息（头像/昵称/最后消息）。

## 模块概述
- **职责:** 代理上游历史/收藏接口；对返回列表进行缓存增强（用户信息、lastMsg/lastTime）
- **状态:** ✅稳定
- **最后更新:** 2026-01-18

## 规范

### 需求: 历史/收藏用户列表补充最后消息
**模块:** User History
历史/收藏用户列表 `/api/getHistoryUserList`、`/api/getFavoriteUserList` 需要在响应中包含 `lastMsg` 和 `lastTime`，优先使用缓存数据。

#### lastMsg 格式化规则（缓存增强）
- 文本消息：原文（超长截断）
- 媒体消息：`[path/to/file.ext]` → `[图片]` / `[视频]` / `[音频]` / `[文件]`（按扩展名识别）
- 图文混排：`喜欢吗[path/to/file.ext]` → `喜欢吗 [图片]`（文本截断 + 媒体标签，优先级：图片 > 视频 > 音频 > 文件）
- 表情文本：`[doge]` 等无路径分隔符且无扩展名的 `[...]` 片段，按普通文本返回（避免误识别为 `[文件]`）

#### 场景: 上游用户ID字段不固定
上游返回用户ID字段可能为 `id` / `UserID` / `userid` 等。
- 预期结果1: 能正确识别对方用户ID并查找会话缓存
- 预期结果2: 在响应中填充 `lastMsg` / `lastTime`

### 需求: 历史/收藏列表分段耗时日志
**模块:** User History
为定位接口耗时来源，历史/收藏用户列表接口在后端输出分段耗时日志（便于按字段聚合/对比）。

#### 场景: 定位耗时热点
- 预期结果1: 记录上游请求耗时（`upstreamMs`）
- 预期结果2: 记录补充用户基本信息耗时（`enrichUserInfoMs`）
- 预期结果3: 记录补充最后消息耗时（`lastMsgMs`）
- 预期结果4: 记录总耗时（`totalMs`）并输出列表大小（`size`）与缓存开关（`cacheEnabled`）

## API接口

### [POST] /api/getHistoryUserList
**描述:** 获取历史用户列表，并对列表进行缓存增强
**输入:** `myUserID`, `vipcode`, `serverPort`（表单参数）
**输出:** 用户列表（JSON），元素包含 `lastMsg`, `lastTime`（如缓存命中）

### [POST] /api/getFavoriteUserList
**描述:** 获取收藏用户列表，并对列表进行缓存增强
**输入:** `myUserID`, `vipcode`, `serverPort`（表单参数）
**输出:** 用户列表（JSON），元素包含 `lastMsg`, `lastTime`（如缓存命中）

## 依赖
- `internal/app/user_history_handlers.go`
- `internal/app/user_info_cache.go`（接口定义）
- `internal/app/user_info_cache_redis.go` / `internal/app/user_info_cache.go`（Redis/Memory 实现）

## 变更历史
- [202601041818_fix_history_userlist_lastmsg](../../history/2026-01/202601041818_fix_history_userlist_lastmsg/) - 修复 lastMsg/lastTime 增强在 `UserID/userid` 场景失效
- [202601041854_fix_lastmsg_key_normalize](../../history/2026-01/202601041854_fix_lastmsg_key_normalize/) - 修复消息id/toid与myUserID不一致导致 lastMsg/lastTime 无法命中
- [202601051213_perf_userlist_timing_logs](../../history/2026-01/202601051213_perf_userlist_timing_logs/) - 为历史/收藏用户列表增加分段耗时日志（上游/补充用户信息/最后消息/总耗时）
- [202601120630_fix_lastmsg_emoji_preview](../../history/2026-01/202601120630_fix_lastmsg_emoji_preview/) - 修复 lastMsg 格式化将表情文本（如 `[doge]`）误识别为 `[文件]`
