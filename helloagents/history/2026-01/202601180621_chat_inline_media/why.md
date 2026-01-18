# 变更提案: 聊天消息图文混排渲染

## 需求背景
当前聊天消息的媒体渲染规则为“整条消息内容等于 `[path/to/file.ext]`”时才识别为图片/视频/文件。若用户发送类似 `喜欢吗[2026/01/18/Random/xxx.jpg]` 的混合内容，前端会按普通文本展示，图片不会渲染，导致体验不一致。

## 变更内容
1. 支持在同一条消息内解析并渲染“文字 + `[媒体路径]`”混排内容（可包含多个媒体片段），同时保留文字展示。
2. 兼容现有行为：纯媒体消息（`[path]`）仍按图片/视频/文件渲染；表情文本（如 `[doge]`）不应被误识别为文件。
3. 会话列表的最后消息预览（lastMsg）与后端最后消息缓存的格式化逻辑支持识别混排媒体，并输出可读摘要（如 `喜欢吗 [图片]`）。

## 影响范围
- **模块:**
  - 前端 Chat UI（消息解析/渲染、会话列表预览）
  - 后端 用户历史/最后消息缓存（类型推断、lastMsg 格式化）
- **文件（预估）:**
  - `frontend/src/composables/useWebSocket.ts`
  - `frontend/src/stores/message.ts`
  - `frontend/src/components/chat/MessageList.vue`
  - `frontend/src/components/chat/ChatHistoryPreview.vue`
  - `frontend/src/components/chat/MessageBubble.vue`
  - `frontend/src/types/message.ts`
  - `frontend/src/utils/*`（新增解析工具）
  - `internal/app/user_history_handlers.go`
  - `internal/app/user_info_cache.go`
  - `internal/app/user_history_handlers_test.go`
  - `internal/app/user_info_cache_test.go`

## 核心场景

### 需求: 图文混排消息渲染
**模块:** Chat UI
在消息文本中出现形如 `[2026/01/18/Random/xxx.jpg]` 的媒体片段时，同一条消息内同时显示文字与图片。

#### 场景: WebSocket 实时消息混排
收到消息内容：`喜欢吗[2026/01/18/Random/xxx.jpg]`
- 预期结果: 气泡内先显示“喜欢吗”，再显示对应图片
- 预期结果: 会话列表 lastMsg 显示为 `喜欢吗 [图片]`（方向前缀规则保持不变）

#### 场景: 历史消息混排
历史记录返回内容：`喜欢吗[2026/01/18/Random/xxx.jpg]`
- 预期结果: 渲染效果与 WebSocket 实时一致（图文同显）

### 需求: 兼容表情与旧格式
**模块:** Chat UI / 最后消息缓存
确保不会把表情文本误判为文件，且旧格式不回归。

#### 场景: 表情文本不应被识别为文件
收到消息内容：`[doge]`
- 预期结果: 按表情/文本显示，不触发媒体 URL 拼接

#### 场景: 纯媒体消息保持原行为
收到消息内容：`[2026/01/18/Random/xxx.jpg]`
- 预期结果: 仍按图片消息渲染，预览标签为 `[图片]`

### 需求: lastMsg 摘要与类型推断增强
**模块:** 后端缓存 / Chat UI
当消息为混排时，后端推断类型与 lastMsg 格式化应体现媒体类型，避免列表直接展示原始路径。

#### 场景: 后端最后消息摘要（混排）
缓存内容：`喜欢吗[2026/01/18/Random/xxx.jpg]`
- 预期结果: lastMsg 格式化输出为 `喜欢吗 [图片]`（如为我发出则带 `我: ` 前缀）

## 风险评估
- **风险:** 混排解析误判（如用户输入类似 `[v1.0]`）导致错误渲染。
- **缓解:** 仅当 `[]` 内内容满足“非表情 + 具备可识别扩展名 + 不包含协议头(如 `://`)”时才按媒体处理；否则按文本保留。
