# User History（用户聊天历史）

## 目的
提供历史用户列表与消息历史代理，并在本地补充缓存信息（头像/昵称/最后消息）。

## 模块概述
- **职责:** 代理上游历史接口；对返回列表进行缓存增强（用户信息、lastMsg/lastTime）
- **状态:** ✅稳定
- **最后更新:** 2026-01-04

## 规范

### 需求: 历史用户列表补充最后消息
**模块:** User History
历史用户列表 `/api/getHistoryUserList` 需要在响应中包含 `lastMsg` 和 `lastTime`，优先使用缓存数据。

#### 场景: 上游用户ID字段不固定
上游返回用户ID字段可能为 `id` / `UserID` / `userid` 等。
- 预期结果1: 能正确识别对方用户ID并查找会话缓存
- 预期结果2: 在响应中填充 `lastMsg` / `lastTime`

## API接口

### [POST] /api/getHistoryUserList
**描述:** 获取历史用户列表，并对列表进行缓存增强
**输入:** `myUserID`, `vipcode`, `serverPort`（表单参数）
**输出:** 用户列表（JSON），元素包含 `lastMsg`, `lastTime`（如缓存命中）

## 依赖
- `com.zcw.controller.UserHistoryController`
- `com.zcw.service.UserInfoCacheService`（Redis/Memory 实现）

## 变更历史
- [202601041818_fix_history_userlist_lastmsg](../../history/2026-01/202601041818_fix_history_userlist_lastmsg/) - 修复 lastMsg/lastTime 增强在 `UserID/userid` 场景失效
