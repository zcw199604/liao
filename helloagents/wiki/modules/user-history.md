# User History

## 目的
代理上游历史/收藏/消息接口，并使用本地缓存和归档增强列表可用性。

## 模块概述
- **职责:** 历史用户列表、收藏用户列表、消息历史、上游删除、最后消息缓存、用户信息缓存、本地归档。
- **状态:** 稳定
- **最后更新:** 2026-05-10

## 规范

### 需求: 列表增强
**模块:** User History  
`getHistoryUserList` 与 `getFavoriteUserList` 代理上游后，尽量补齐昵称、性别、年龄、地址、最近消息和最近时间。

#### 场景: 上游字段名不稳定
- 代码需要兼容 `id`、`UserID`、`userid` 等字段。
- 缓存命中时补齐展示字段。

### 需求: 本地归档兜底
**模块:** User History  
上游返回列表时，将用户快照和最近消息持久化到 `chat_user_archive`，用于上游删除后的恢复展示。

#### 场景: 上游用户被删除
- 列表接口可合并本地归档项。
- 删除上游用户成功后应同步清理本地归档，避免手动删除后回流。

#### 场景: 匹配成功但未聊天
- WebSocket 收到上游 `code=15` 匹配成功事件时，应以当前身份 ID 作为 owner，将匹配用户快照写入 `chat_user_archive` 的 history 来源。
- 刷新历史列表时，先保持上游返回用户顺序，再把上游缺失的本地归档用户追加到列表末尾。

### 需求: Redis 聊天记录缓存
**模块:** User History  
`getMessageHistory` 在 Redis 模式下可缓存并合并 `contents_list`。

#### 场景: 最新页
- 始终请求上游，保证最新消息。

#### 场景: 历史翻页
- Redis 命中足够时可跳过上游。

## API接口
- `POST /api/getHistoryUserList`
- `POST /api/getFavoriteUserList`
- `POST /api/reportReferrer`
- `POST /api/getMessageHistory`
- `POST /api/toggleFavorite`
- `POST /api/cancelFavorite`
- `POST /api/deleteUpstreamUser`
- `POST /api/batchDeleteUpstreamUsers`

## 数据模型
- `chat_user_archive`
- Redis `user:lastmsg:{conversationKey}`
- Redis `user:chathistory:{conversationKey}`

## 依赖
- `internal/app/user_history_handlers.go`
- `internal/app/user_archive.go`
- `internal/app/websocket_manager.go`
- `internal/app/user_info_cache*.go`
- `internal/app/chat_history_cache*.go`
