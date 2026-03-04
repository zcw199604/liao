# 任务清单: 聊天消息图文混排渲染

目录: `helloagents/plan/202601180621_chat_inline_media/`

---

## 1. 前端: 混排解析与数据结构
- [√] 1.1 在 `frontend/src/types/message.ts` 增加 `MessageSegment` 类型与 `ChatMessage.segments` 字段，确保对现有类型引用无破坏性影响，验证 why.md#需求-图文混排消息渲染-场景-websocket-实时消息混排
- [√] 1.2 新增 `frontend/src/utils/messageSegments.ts`（或同等位置）实现“混排解析 + URL 拼接”的复用函数（支持 emojiMap 排除、扩展名识别、无协议头约束），验证 why.md#需求-兼容表情与旧格式-场景-表情文本不应被识别为文件

## 2. 前端: WebSocket 收消息与会话列表预览
- [√] 2.1 在 `frontend/src/composables/useWebSocket.ts` 中接入混排解析，生成 `segments/isImage/isVideo/isFile/imageUrl/videoUrl/fileUrl` 并保持旧格式兼容，验证 why.md#需求-图文混排消息渲染-场景-websocket-实时消息混排
- [√] 2.2 在 `frontend/src/composables/useWebSocket.ts` 中调整 `lastMsg` 生成逻辑为基于 `segments` 的摘要（`文本(截断) + 标签`），验证 why.md#需求-lastmsg-摘要与类型推断增强-场景-后端最后消息摘要（混排）

## 3. 前端: 历史消息解析与去重一致性
- [√] 3.1 在 `frontend/src/stores/message.ts` 的 `loadHistory` 映射逻辑中复用混排解析，保证历史与 WS 渲染一致，验证 why.md#需求-图文混排消息渲染-场景-历史消息混排
- [√] 3.2 在 `frontend/src/stores/message.ts` 中增强媒体去重 key（支持从 `segments` 提取媒体 path），验证 why.md#需求-图文混排消息渲染-场景-历史消息混排，依赖任务3.1

## 4. 前端: 渲染组件支持图文同显
- [√] 4.1 在 `frontend/src/components/chat/MessageList.vue` 中支持按 `segments` 渲染（文本段 + 图片段/视频段/文件段），并保留 `segments` 缺失时的旧分支回退，验证 why.md#需求-图文混排消息渲染-场景-websocket-实时消息混排
- [√] 4.2 在 `frontend/src/components/chat/ChatHistoryPreview.vue` 中支持按 `segments` 渲染（与 MessageList 行为一致），验证 why.md#需求-图文混排消息渲染-场景-历史消息混排
- [√] 4.3 在 `frontend/src/components/chat/MessageBubble.vue` 中支持按 `segments` 渲染（用于组件复用与单测覆盖），验证 why.md#需求-图文混排消息渲染-场景-websocket-实时消息混排

## 5. 后端(Go): 类型推断与最后消息摘要增强
- [√] 5.1 在 `internal/app/user_history_handlers.go` 中增强 `inferMessageType`：支持从混排内容中扫描 `[...]` 媒体片段并推断 image/video/audio/file，验证 why.md#需求-lastmsg-摘要与类型推断增强-场景-后端最后消息摘要（混排）
- [√] 5.2 在 `internal/app/user_info_cache.go` 中增强 `formatLastMessage`：混排时输出 `文本(截断) + 标签`，emoji 不误判（同时覆盖 Memory/Redis 缓存实现），验证 why.md#需求-lastmsg-摘要与类型推断增强-场景-后端最后消息摘要（混排）
- [-] 5.3 Go 端 Redis 缓存无需单独修改（复用 `formatLastMessage`），已由 5.2 覆盖

## 6. 安全检查
- [√] 6.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避），重点关注“媒体识别条件”与“URL 拼接”不引入协议注入风险

## 7. 文档更新
- [√] 7.1 更新 `helloagents/wiki/modules/chat-ui.md`：补充“混排媒体消息”解析与预览摘要规则，并列出受影响接入点
- [√] 7.2 更新 `helloagents/CHANGELOG.md` 记录本次变更（fix/feat 归类按最终实现确认）

## 8. 测试
- [√] 8.1 在 `internal/app/user_info_cache_test.go` / `internal/app/user_history_handlers_test.go` 增加混排消息用例（如 `喜欢吗[20260104/image.jpg]` -> `我: 喜欢吗 [图片]`），并确保 `[doge]` 仍不误判
- [√] 8.2 新增/补充前端 Vitest 用例覆盖混排解析（至少覆盖：混排图片、纯图片、emoji、误判保护），验证 why.md#需求-兼容表情与旧格式

---

## 执行备注
- 已使用临时 Go 1.22 工具链运行 `go test ./...`，测试通过。
