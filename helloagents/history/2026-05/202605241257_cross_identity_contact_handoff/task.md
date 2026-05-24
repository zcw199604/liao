# 任务清单: 跨身份联系人接入

目录: `helloagents/history/2026-05/202605241257_cross_identity_contact_handoff/`

---

## 1. 后端候选接口
- [√] 1.1 在 `internal/app/user_archive.go` 中新增归档候选查询能力，返回 `owner_user_id` 下的 `chat_user_archive` 用户快照，验证 why.md#需求-候选数据包含数据库归档-场景-上游不可用或目标已被上游列表删除
- [√] 1.2 在 `internal/app/user_history_handlers.go` 中实现联系人候选合并逻辑，合并上游历史、上游收藏和本地归档，并按 `targetUserId` 去重，验证 why.md#需求-从其他身份选择联系人-场景-来源身份有上游历史或收藏对象，依赖任务1.1
- [√] 1.3 在 `internal/app/router.go` 注册 `GET /api/chat/contactCandidates`，并补充参数校验、limit 上限、上游失败 warnings，验证 why.md#需求-候选数据包含数据库归档-场景-上游不可用或目标已被上游列表删除，依赖任务1.2

## 2. 前端候选数据与临时会话状态
- [√] 2.1 在 `frontend/src/api/chat.ts` 中新增 `getContactCandidates` API 封装和响应类型，验证 why.md#需求-从其他身份选择联系人-场景-来源身份有本地归档匹配对象
- [√] 2.2 在 `frontend/src/stores/chat.ts` 中新增临时接入会话状态（例如 `temporaryConversationByKey` 或 `currentChatUser.localTemporary`），验证 why.md#需求-当前身份临时接入会话-场景-首条消息发送前
- [√] 2.3 在 `frontend/src/composables/useChat.ts` 中新增 `enterTemporaryChatFromCandidate` 方法，基于候选快照创建 `User` 并进入聊天，不立即写入正式历史列表，验证 why.md#需求-当前身份临时接入会话-场景-首条消息发送前，依赖任务2.1、2.2

## 3. 前端按钮与选择器 UI
- [√] 3.1 新增 `frontend/src/components/chat/CrossIdentityContactPicker.vue`，实现来源身份选择、候选加载、搜索、来源标记和空状态，验证 why.md#需求-从其他身份选择联系人-场景-来源身份有本地归档匹配对象
- [√] 3.2 在 `frontend/src/views/ChatListView.vue` 或 `frontend/src/components/chat/ChatSidebar.vue` 增加“从其他身份接入”按钮并打开选择器，验证 why.md#需求-从其他身份选择联系人-场景-来源身份有上游历史或收藏对象，依赖任务3.1
- [√] 3.3 在 `frontend/src/views/ChatRoomView.vue` 支持临时会话标识展示和发送失败时保留重试入口，验证 why.md#需求-当前身份临时接入会话-场景-首条消息发送前，依赖任务2.3

## 4. 消息缓存身份隔离
- [√] 4.1 在 `frontend/src/stores/message.ts` 中新增本地会话 key 规范，将 `chatHistory`、`firstTidMap`、`clearHistory` 统一改为 `currentIdentityId:targetUserId` 维度，验证 why.md#需求-消息缓存按身份隔离-场景-A-与-B-分别联系同一目标
- [√] 4.2 调整 `frontend/src/composables/useChat.ts`、`frontend/src/composables/useMessage.ts`、`frontend/src/composables/useWebSocket.ts`、`frontend/src/views/ChatRoomView.vue` 中所有 message store 调用点，传入当前身份 ID 或会话 key，验证 why.md#需求-消息缓存按身份隔离-场景-A-与-B-分别联系同一目标，依赖任务4.1
- [√] 4.3 调整乐观发送超时与回显合并逻辑，使 B:test 的发送状态不会影响 A:test，验证 why.md#需求-消息缓存按身份隔离-场景-A-与-B-分别联系同一目标，依赖任务4.2

## 5. 首发后转正与历史刷新
- [√] 5.1 在 `frontend/src/composables/useMessage.ts` 中为发送成功/回显确认增加回调或事件，当前会话为临时接入时触发 B 身份历史列表刷新，验证 why.md#需求-当前身份临时接入会话-场景-首条消息发送成功
- [√] 5.2 在 `frontend/src/stores/chat.ts` 或 `useChat.ts` 中实现临时会话转正式状态：刷新历史命中目标后移除临时标记，验证 why.md#需求-当前身份临时接入会话-场景-首条消息发送成功，依赖任务5.1

## 6. 安全检查
- [√] 6.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避），重点确认 `sourceIdentityId` 仅用于候选读取、发送消息仍使用当前身份
- [√] 6.2 检查候选接口响应不包含 cookie、JWT、访问码、上游请求头等敏感信息

## 7. 测试
- [√] 7.1 新增或扩展后端测试，覆盖 `GET /api/chat/contactCandidates` 的本地归档、上游合并去重、上游失败降级、参数非法场景
- [√] 7.2 新增或扩展前端测试，覆盖选择器打开、身份选择、候选加载、选中进入临时会话、首发后刷新历史
- [√] 7.3 新增或扩展 message store 测试，覆盖 `A:test` 与 `B:test` 缓存隔离、回显合并隔离、加载更多隔离
- [√] 7.4 运行 `go test ./...` 和 `cd frontend && npm run build`

## 8. 知识库同步
- [√] 8.1 更新 `helloagents/wiki/api.md`，记录 `GET /api/chat/contactCandidates`
- [√] 8.2 更新 `helloagents/wiki/modules/user-history.md`，记录跨身份联系人候选和归档数据源
- [√] 8.3 更新 `helloagents/wiki/modules/chat-ui.md`，记录“从其他身份接入”按钮、临时会话和首发后转正流程
- [√] 8.4 更新 `helloagents/wiki/data.md`，说明本功能复用 `chat_user_archive` 且不新增表
- [√] 8.5 更新 `helloagents/CHANGELOG.md`，记录用户可见能力变化
