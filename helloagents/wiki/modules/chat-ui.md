# Chat UI

## 目的
提供 Web 端匿名聊天体验，包括列表、聊天室、消息输入、媒体预览、未读计数和身份切换。

## 模块概述
- **职责:** Vue 路由守卫、聊天列表、聊天室、WebSocket 消息解析、消息 stores、媒体弹窗、设置抽屉。
- **状态:** 稳定
- **最后更新:** 2026-05-10

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
