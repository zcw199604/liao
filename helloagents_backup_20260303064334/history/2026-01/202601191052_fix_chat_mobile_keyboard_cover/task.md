# 任务清单: 修复移动端键盘遮挡聊天最新消息

目录: `helloagents/history/2026-01/202601191052_fix_chat_mobile_keyboard_cover/`

---

## 1. 视口高度与键盘适配
- [√] 1.1 调整 `frontend/src/index.css`：`.page-container` 改用 `100dvh` 并支持 `--app-height` 覆盖
- [√] 1.2 在 `frontend/src/main.ts` 注入 `--app-height`（优先 `visualViewport.height`，回退 `innerHeight`）

## 2. 消息列表贴底滚动
- [√] 2.1 调整 `frontend/src/components/chat/MessageList.vue`：监听滚动容器尺寸变化（ResizeObserver），仅当用户位于底部时自动贴底

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/chat-ui.md`：补充“移动端键盘遮挡”治理规范
- [√] 3.2 更新 `helloagents/CHANGELOG.md`：记录本次修复
- [√] 3.3 更新 `helloagents/history/index.md`：追加变更记录

## 4. 测试
- [√] 4.1 运行前端单测：`cd frontend && npm test`
- [√] 4.2 运行前端构建：`cd frontend && npm run build`

## 5. 审查
- [√] 5.1 使用本地 `gemini` CLI 审查 diff：兼容性与副作用评估
