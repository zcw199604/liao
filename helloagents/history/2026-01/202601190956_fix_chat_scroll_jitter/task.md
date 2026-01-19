# 任务清单: 修复聊天图片加载抖动

目录: `helloagents/history/2026-01/202601190956_fix_chat_scroll_jitter/`

---

## 1. 消息列表滚动稳定性
- [√] 1.1 调整 `frontend/src/components/chat/MessageList.vue`：系统自动贴底使用 `behavior: 'auto'`，仅用户点击“回到底部/新消息”使用 `smooth`
- [√] 1.2 在 `frontend/src/components/chat/MessageList.vue` 合并同一帧内的多次贴底滚动请求（`requestAnimationFrame` + `nextTick`）
- [√] 1.3 在媒体加载/失败后，仅当用户位于底部时才触发贴底滚动，避免打断阅读历史消息
- [√] 1.4 为滚动容器显式启用 `overflow-anchor: auto`，减少内容尺寸变化导致的视觉跳动

## 2. ChatMedia 预占位与事件
- [√] 2.1 调整 `frontend/src/components/chat/ChatMedia.vue`：为图片/视频提供默认占位比例（image=4/3, video=16/9），并在 `load/error/loadeddata` 时触发 `layout` 事件
- [√] 2.2 为图片增加 `decoding="async"`，降低解码对主线程的阻塞

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/chat-ui.md`：补充“滚动抖动治理”规范与注意事项
- [√] 3.2 更新 `helloagents/CHANGELOG.md`：记录本次修复

## 4. 测试
- [√] 4.1 运行前端单测：`cd frontend && npm test`
- [√] 4.2 运行前端构建：`cd frontend && npm run build`
