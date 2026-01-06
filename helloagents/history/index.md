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
| 202601051001 | vue-component-tests | 测试 | ✅已完成 | [链接](2026-01/202601051001_vue-component-tests/) |
| 202601051025 | vue-view-tests | 测试 | ✅已完成 | [链接](2026-01/202601051025_vue-view-tests/) |
| 202601051105 | frontend-test-coverage | 测试 | ✅已完成 | [链接](2026-01/202601051105_frontend-test-coverage/) |
| 202601051213 | perf_userlist_timing_logs | 优化 | ⚠️待复验 | [链接](2026-01/202601051213_perf_userlist_timing_logs/) |
| 202601060948 | chat_gesture_ux | 功能 | ✅已完成 | [链接](2026-01/202601060948_chat_gesture_ux/) |

---

## 按月归档

### 2026-01

- [202601041818_fix_history_userlist_lastmsg](2026-01/202601041818_fix_history_userlist_lastmsg/) - 修复历史用户列表 lastMsg/lastTime 增强对 `UserID/userid` 的兼容性
- [202601041854_fix_lastmsg_key_normalize](2026-01/202601041854_fix_lastmsg_key_normalize/) - 修复消息id/toid与myUserID不一致导致 lastMsg/lastTime 无法命中
- [202601042052_frontend-tests](2026-01/202601042052_frontend-tests/) - 前端接入 Vitest 并补充核心模块单元测试
- [202601042237_fix_list_time_and_favorite_enrich](2026-01/202601042237_fix_list_time_and_favorite_enrich/) - 修复聊天列表时间格式化不一致，并对齐收藏列表缓存增强（待JDK17环境复验）
- [202601051001_vue-component-tests](2026-01/202601051001_vue-component-tests/) - 前端补充 Vue 组件级测试（SFC 渲染/交互）
- [202601051025_vue-view-tests](2026-01/202601051025_vue-view-tests/) - 前端补充视图级页面测试（LoginPage/IdentityPicker/ChatListView/ChatRoomView）
- [202601051105_frontend-test-coverage](2026-01/202601051105_frontend-test-coverage/) - 前端补充核心聊天业务与关键组件测试覆盖（composables/store/components/utils）
- [202601051213_perf_userlist_timing_logs](2026-01/202601051213_perf_userlist_timing_logs/) - 为历史/收藏用户列表增加分段耗时日志（上游/补充用户信息/最后消息/总耗时）
- [202601060948_chat_gesture_ux](2026-01/202601060948_chat_gesture_ux/) - 聊天手势与弹层交互增强（列表左右滑切换/边缘右滑返回/抽屉左滑关闭/点击外关闭）
