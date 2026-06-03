# Chat UI

## 目的
提供 Web 端匿名聊天体验，包括列表、聊天室、消息输入、媒体预览、未读计数、身份切换、跨身份联系人接入和全局归档搜索。

## 模块概述
- **职责:** Vue 路由守卫、聊天列表、聊天室、跨身份联系人选择、全局归档搜索、临时会话、WebSocket 消息解析、消息 stores、媒体弹窗、设置抽屉。
- **状态:** 稳定
- **最后更新:** 2026-06-03

## 规范

### 需求: 路由鉴权
**模块:** Chat UI
`/list` 与 `/chat/:userId?` 需要已登录且已选择身份；未登录跳转 `/login`，未选身份跳转 `/identity`。

#### 场景: 页面刷新
- Go SPA 回退必须支持 Vue Router `createWebHistory()` 路径刷新。

### 需求: 未读计数
**模块:** Chat UI
只有当前处于 `/chat` 路由且消息属于当前会话时，才把消息视为已读。

#### 场景: 列表页收到消息
- 即使 `currentChatUser` 残留，也应累加未读。

### 需求: WebSocket 消息展示
**模块:** Chat UI
前端解析上游 `code=7` 私信、`code=15` 匹配用户、正在输入事件、forceout 和系统消息。

#### 场景: 自己发送的消息回显
- 使用当前身份 ID 的 MD5 与 `fromuser.id` 对齐判断是否为自己。
- 尝试合并本地乐观消息，避免重复渲染。

#### 场景: 匹配成功未聊天
- `code=15` 匹配成功后，前端立即把用户加入当前身份的历史列表顶部，支持用户直接进入聊天。
- 后端会同步归档匹配用户；后续刷新列表时由历史列表接口补齐缺失用户。

### 需求: 列表身份隔离
**模块:** Chat UI
聊天列表状态必须绑定当前身份，避免切换身份后复用上一个身份的内存列表。

#### 场景: 身份切换后进入列表
- `chat` store 记录 `listOwnerUserId`。
- 加载历史/收藏列表前发现 owner 变化时，应清空旧身份的 `userMap`、历史 ID、收藏 ID 和当前聊天对象，再加载新身份列表。

### 需求: 全局收藏切换身份进入聊天
**模块:** Chat UI
全局收藏中的“切换身份并聊天”必须使用确定性的身份切换和会话准备流程，不能依赖固定延时等待列表页加载。

#### 场景: 当前 A 身份点击 B 身份收藏对象
- 断开旧身份 WebSocket，并清空旧身份聊天列表、当前聊天对象、消息缓存和连续匹配状态。
- 选择 B 身份时不自动跳转 `/list`，避免中间页面加载清空目标聊天对象。
- 在 B 身份 owner 下插入收藏目标用户，设置 `currentChatUser`，并按 `B:目标用户` 会话键加载历史。
- 最终跳转 `/chat/:targetUserId`，聊天窗口顶部应显示目标用户，历史列表按 B 身份上下文展示。

#### 场景: 历史加载失败
- 仍保留目标用户为 `currentChatUser`，并停留 `/chat/:targetUserId`。
- 历史加载失败只影响消息列表内容，不应回退 `/list` 或显示空聊天对象。

### 需求: 从其他身份接入联系人
**模块:** Chat UI
聊天侧边栏提供“从其他身份接入”入口，用户可选择非当前身份作为来源，再从来源身份的历史、收藏和本地归档候选中选择目标用户。

#### 场景: 选择候选进入聊天
- `CrossIdentityContactPicker.vue` 打开时加载身份列表，默认选中第一个非当前身份。
- 选择器通过 `getContactCandidates` 请求候选，展示来源标记：历史、收藏、归档，并保持后端返回顺序，不做二次排序。
- 候选顺序与来源身份实际列表顺序一致：历史原序优先、收藏原序追加、归档仅补充上游未返回联系人。
- 选中候选后调用 `enterTemporaryChatFromCandidate`，在当前身份下创建临时会话并跳转 `/chat/:targetUserId`。
- 选中目标不会立即写入当前身份正式历史列表。

#### 场景: 首条消息发送成功
- 文本、图片、视频和重试发送成功后，如果当前会话仍为临时会话，则刷新当前身份历史列表。
- 刷新后如果目标用户已出现在当前身份历史中，调用 `markConversationFormal` 移除临时标记。
- 收到自己消息 WebSocket 回显时也会触发同一刷新和转正逻辑。

#### 场景: 消息缓存按身份隔离
- `message` store 使用 `conversationKey(ownerUserId, targetUserId)` 作为本地消息缓存 key。
- `A:test` 与 `B:test` 的消息、乐观发送状态、超时和回显合并互不影响。
- 为兼容旧调用，单参数 `getMessages('test')` 在只有唯一 `*:test` 会话时可回读该会话，但新写入应传当前身份或显式会话 key。

#### 场景: 临时会话状态展示
- 聊天室顶部展示临时接入提示。
- 发送失败时仍保留本地乐观消息和重试入口，不强行把目标加入正式历史列表。

### 需求: 全局归档搜索接入
**模块:** Chat UI
聊天侧边栏提供“归档搜索”入口，用户可按目标用户 ID 或归档快照名称搜索所有身份下的本地归档会话。

#### 场景: 搜索归档用户
- `ChatArchiveSearchPicker.vue` 调用 `searchChatArchive` 请求 `/api/chat/archiveSearch`。
- 搜索结果展示目标用户名称、目标用户 ID、归档来源身份 `ownerUserId`、历史/收藏来源标签和最近消息。
- 搜索结果按后端返回顺序展示，不在前端二次排序。

#### 场景: 选择归档结果进入聊天
- 选中归档结果后，以结果中的 `ownerUserId` 作为来源身份，调用 `enterTemporaryChatFromCandidate`。
- 当前登录身份下创建临时会话并跳转 `/chat/:targetUserId`。
- 选中目标不会立即写入当前身份正式历史列表，仍由首条消息发送成功后的转正流程处理。

### 需求: 聊天媒体预览入口
**模块:** Chat UI
聊天消息中的图片和视频进入统一 `MediaPreview` 能力，以复用视频播放、快退/快进、倍速、抓帧和抽帧入口。

#### 场景: 消息列表视频预览
- `MessageList.vue` 的 segments 视频和 fallback 视频都通过 `ChatMedia` 默认预览能力派发 `preview-media`。
- 视频预览事件的 `detail.type` 为 `video`，由上层预览容器打开 `MediaPreview`。
- `ChatMedia.previewable` 默认保持兼容；只有调用方显式禁用时才不进入预览。

## API接口
通过 `frontend/src/api/*` 调用后端 `/api`，通过 `frontend/src/composables/useWebSocket.ts` 连接 `/ws`。

## 数据模型
前端状态由 Pinia stores 维护：`auth`、`user`、`chat`、`message`、`media`、`systemConfig` 等。

## 依赖
- `frontend/src/router/index.ts`
- `frontend/src/views/ChatListView.vue`
- `frontend/src/views/ChatRoomView.vue`
- `frontend/src/components/chat/*`
- `frontend/src/composables/useWebSocket.ts`
- `frontend/src/stores/chat.ts`
