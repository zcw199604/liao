# 任务清单: 修复未聊天筛选导致的用户列表排序异常

目录: `helloagents/plan/202605050920_fix-not-chatted-sort/`

---

## 1. 定位与排序策略实现
- [ ] 1.1 在 `frontend/src/stores/chat.ts` 中调整 `loadHistoryUsers()` 合并逻辑，保留未聊天用户但取消无条件前置，验证 why.md#需求-默认消息列表按最近会话排序-场景-已有未聊天用户且后端返回较新的真实会话
- [ ] 1.2 在 `frontend/src/stores/chat.ts` 中实现稳定排序兜底：按 `lastMessageTime/lastTime` 降序，时间缺失或相等时保留合并前相对顺序，依赖任务1.1

## 2. 回归测试
- [ ] 2.1 在 `frontend/src/__tests__/chat-store-load.test.ts` 中新增测试：刷新后本地未聊天用户仍保留，但后端较新真实会话排在其前面，验证 why.md#需求-默认消息列表按最近会话排序-场景-已有未聊天用户且后端返回较新的真实会话
- [ ] 2.2 在 `frontend/src/__tests__/chat-store-load.test.ts` 中补充测试：本地未聊天用户时间较新时可正常排在前面，避免简单追加末尾造成反向问题，依赖任务2.1

## 3. 安全检查
- [ ] 3.1 执行安全检查（按G9: 确认无生产服务连接、无敏感信息写入、无破坏性操作、无权限/支付相关变更）

## 4. 文档更新
- [ ] 4.1 更新 `helloagents/wiki/modules/chat-ui.md`，记录未聊天筛选与默认列表排序规则
- [ ] 4.2 更新 `helloagents/CHANGELOG.md` 的 Unreleased 修复项

## 5. 验证
- [ ] 5.1 执行 `cd frontend && npm run build`，验证 TypeScript 与生产构建通过
- [ ] 5.2 如测试脚本可用，执行相关 Vitest 用例或现有前端测试命令，验证新增回归测试通过
