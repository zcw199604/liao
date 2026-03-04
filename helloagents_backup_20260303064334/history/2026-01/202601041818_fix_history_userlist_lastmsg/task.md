# 任务清单: 修复历史用户列表 lastMsg/lastTime（轻量迭代）

目录: `helloagents/history/2026-01/202601041818_fix_history_userlist_lastmsg/`

---

## 1. 最后消息增强
- [√] 1.1 兼容读取 `id/UserID/userid/userId` 作为对方用户ID（Redis实现）
- [√] 1.2 兼容读取 `id/UserID/userid/userId` 作为对方用户ID（内存实现）

## 2. 测试
- [√] 2.1 新增 `RedisUserInfoCacheServiceLastMessageTest` 用例覆盖 `UserID` 字段
- [√] 2.2 新增 `MemoryUserInfoCacheServiceLastMessageTest` 用例覆盖 `UserID` 字段
- [√] 2.3 运行 `mvn test`（JDK 17）
