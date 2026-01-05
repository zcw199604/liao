# 任务清单（轻量迭代）

- [√] 前端：聊天列表（侧边栏）会话时间统一使用 `formatTime`
- [√] 后端：`/api/getFavoriteUserList` 对齐 `/api/getHistoryUserList` 的缓存增强逻辑（用户信息、lastMsg、lastTime）
- [X] 后端验证：运行 `mvn test`
  > 备注: 当前环境使用 JDK 8（`--release` 不支持）导致 Maven 编译失败；需在 JDK 17 环境复验。
- [√] 知识库：更新 `helloagents/wiki/modules/user-history.md` 与 `helloagents/CHANGELOG.md`
- [√] 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
