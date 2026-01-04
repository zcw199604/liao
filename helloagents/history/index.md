# 变更历史索引

本文件记录所有已完成变更的索引，便于追溯和查询。

---

## 索引

| 时间戳 | 功能名称 | 类型 | 状态 | 方案包路径 |
|--------|----------|------|------|------------|
| 202601041818 | fix_history_userlist_lastmsg | 修复 | ✅已完成 | [链接](2026-01/202601041818_fix_history_userlist_lastmsg/) |
| 202601041854 | fix_lastmsg_key_normalize | 修复 | ✅已完成 | [链接](2026-01/202601041854_fix_lastmsg_key_normalize/) |
| 202601042052 | frontend-tests | 测试 | ✅已完成 | [链接](2026-01/202601042052_frontend-tests/) |
| 202601042237 | fix_list_time_and_favorite_enrich | 修复 | ⚠️待复验 | [链接](2026-01/202601042237_fix_list_time_and_favorite_enrich/) |

---

## 按月归档

### 2026-01

- [202601041818_fix_history_userlist_lastmsg](2026-01/202601041818_fix_history_userlist_lastmsg/) - 修复历史用户列表 lastMsg/lastTime 增强对 `UserID/userid` 的兼容性
- [202601041854_fix_lastmsg_key_normalize](2026-01/202601041854_fix_lastmsg_key_normalize/) - 修复消息id/toid与myUserID不一致导致 lastMsg/lastTime 无法命中
- [202601042052_frontend-tests](2026-01/202601042052_frontend-tests/) - 前端接入 Vitest 并补充核心模块单元测试
- [202601042237_fix_list_time_and_favorite_enrich](2026-01/202601042237_fix_list_time_and_favorite_enrich/) - 修复聊天列表时间格式化不一致，并对齐收藏列表缓存增强（待JDK17环境复验）
