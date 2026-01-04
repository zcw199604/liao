# Changelog

本文件记录项目所有重要变更。
格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/),
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 修复
- 修复 `/api/getHistoryUserList` 在上游用户ID字段为 `UserID/userid` 时，未能填充 `lastMsg` / `lastTime` 的问题。
- 修复上游返回的消息 `id/toid` 与 `myUserID` 不一致时，最后消息缓存会话Key无法命中导致 `lastMsg` / `lastTime` 缺失的问题。
