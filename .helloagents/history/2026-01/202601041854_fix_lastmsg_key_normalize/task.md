# 任务清单: 修复最后消息会话Key不一致（轻量迭代）

目录: `helloagents/plan/202601041854_fix_lastmsg_key_normalize/`

---

## 1. 缓存写入归一化
- [√] 1.1 WebSocket `code=7` 写入 lastmsg 时，若 from/to 均不等于本地 `userId`，则补写可被 `(userId <-> 对方)` 命中的会话Key
- [√] 1.2 `/api/getMessageHistory` 解析 `contents_list` 时，以请求参数 `(myUserID, UserToID)` 修正缓存写入的会话Key

## 2. 测试
- [√] 2.1 新增 `UpstreamWebSocketClientLastMessageCacheTest`
- [√] 2.2 扩展 `UserHistoryControllerTest` 覆盖会话Key修正
- [√] 2.3 运行 `mvn test`（JDK 17）
