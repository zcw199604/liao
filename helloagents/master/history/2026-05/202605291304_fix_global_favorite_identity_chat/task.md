# 任务清单: 修复全局收藏切换身份进入聊天异常

目录: `helloagents/master/plan/202605291304_fix_global_favorite_identity_chat/`

---

## 0. 方案边界确认
- [√] 0.1 确认本次任务仅覆盖 why.md 的范围内切片，范围外的后端接口、数据库结构、跨身份联系人选择器重构不进入实现。
- [√] 0.2 确认 how.md 的设计边界完整，尤其是身份选择跳转契约、聊天 store owner 边界和消息缓存键。
- [√] 0.3 确认最小改动策略: 不做无关 UI 重构、依赖升级、路由体系重写或公共 API 重命名。

## 1. 身份选择流程参数化
- [√] 1.1 在 `frontend/src/composables/useIdentity.ts` 中为 `select()` 增加可选 `redirectTo?: string | false` 参数，默认保持跳转 `/list`，验证 why.md#需求-普通身份选择行为保持兼容-场景-在身份选择页选择身份。
- [√] 1.2 为 `redirectTo: false` 分支补充测试或可验证路径，确认只设置 `currentUser` 并调用 `identityStore.selectIdentity()`，不触发 `router.push()`，依赖任务1.1。

## 2. 聊天状态准备能力
- [√] 2.1 在 `frontend/src/stores/chat.ts` 中评估是否需要公开 `ensureListOwner()`；如公开，保持函数语义不变并补充类型导出，验证 how.md#设计边界。
- [√] 2.2 在 `frontend/src/composables/useChat.ts` 中新增或复用 helper，将全局收藏项构造为最小 `User`，并在当前身份 owner 下 `upsertUser` + `enterChat`，验证 why.md#需求-全局收藏跨身份切换后进入正确会话-场景-a-身份点击-b-身份全局收藏对象，依赖任务2.1。
- [√] 2.3 确认目标用户进入聊天后，`messageStore.getMessages(currentIdentityId, targetUserId)` 使用新身份会话键读取，不复用旧身份缓存，依赖任务2.2。

## 3. 全局收藏切换流程修复
- [√] 3.1 在 `frontend/src/components/settings/GlobalFavorites.vue` 中移除 `setTimeout` 跳转逻辑，改为 `async/await` 顺序流程，验证 why.md#需求-全局收藏进入聊天流程无固定延时竞态-场景-目标身份列表加载较慢。
- [√] 3.2 在 `GlobalFavorites.vue` 切换开始前执行旧状态清理: `disconnect(true)`、`messageStore.resetAll()`、`chatStore.clearAllUsers()`、取消匹配状态，保持与现有切换身份语义一致，依赖任务3.1。
- [√] 3.3 在 `GlobalFavorites.vue` 中调用 `select(identity, { redirectTo: false })` 后准备目标聊天对象、触发历史加载，并最终 `router.push(/chat/:targetUserId)`，验证 why.md#需求-全局收藏跨身份切换后进入正确会话-场景-a-身份点击-b-身份全局收藏对象，依赖任务3.2。
- [√] 3.4 确保预览弹窗入口和直接聊天按钮共用同一切换函数，避免双路径行为漂移，依赖任务3.3。
- [√] 3.5 如需要关闭抽屉，在 `frontend/src/components/settings/SettingsDrawer.vue` 与 `GlobalFavorites.vue` 之间增加明确事件，例如 `@open-chat` 后 `emit('update:visible', false)`；不得依赖路由后残留遮罩自然消失，依赖任务3.3。

## 4. 回归测试
- [√] 4.1 为 `useIdentity.select()` 增加测试: 默认跳 `/list`，`redirectTo: false` 不跳转，验证 why.md#需求-普通身份选择行为保持兼容-场景-在身份选择页选择身份。
- [√] 4.2 为 `GlobalFavorites.vue` 增加测试: 当前 A 身份点击 B 身份收藏对象 C 后，调用新身份选择、设置 `currentChatUser=C`、路由进入 `/chat/C`，验证 why.md#需求-全局收藏跨身份切换后进入正确会话-场景-a-身份点击-b-身份全局收藏对象。
- [√] 4.3 增加竞态回归测试: 模拟列表 owner 切换清理或慢加载，确认最终聊天对象不会被清空，验证 why.md#需求-全局收藏进入聊天流程无固定延时竞态-场景-目标身份列表加载较慢。
- [√] 4.4 增加异常分支测试: 历史加载失败时仍保留目标聊天对象并进入 `/chat/:targetUserId`，验证 why.md#风险评估。

## 5. 安全检查
- [√] 5.1 执行安全检查（按G9: 确认无生产服务连接、无敏感信息写入、无破坏性命令、无权限/支付相关变更、无新增第三方依赖）。

## 6. 文档更新
- [√] 6.1 更新 `helloagents/master/wiki/modules/chat-ui.md`，记录全局收藏跨身份进入聊天的确定性流程、owner 隔离和历史加载预期。
- [√] 6.2 更新 `helloagents/master/wiki/modules/identity.md`，记录 `select()` 默认跳 `/list` 与可选无跳转选择语义。
- [√] 6.3 更新 `helloagents/master/CHANGELOG.md` 的 Unreleased 修复项。

## 7. 验证
- [√] 7.1 运行相关前端单元测试或现有 Vitest 命令，记录结果。
- [√] 7.2 执行 `cd frontend && npm run build`，验证 TypeScript 与生产构建通过。
- [-] 7.3 手工验证: A 身份打开全局收藏，点击 B 身份收藏对象 C，确认最终身份为 B、路由为 `/chat/C`、顶部显示 C、历史记录按 B:C 加载。
> 备注: 当前执行环境未启动后端和浏览器手工会话；以单元测试和构建验证替代。
