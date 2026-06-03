# 任务清单

- [√] RED - 后端测试
  - 覆盖 `target_user_id LIKE` 命中。
  - 覆盖 `snapshot_json` 中 `nickname/name/targetUserName` 命中。
  - 覆盖不传 `owner_user_id` 也能查到结果。
  - 覆盖坏 JSON、空快照、无命中、跨身份多结果。

- [√] GREEN - 后端实现
  - 新增全局归档搜索 service / handler / route。
  - SQL 按 `target_user_id` 和 `snapshot_json` 做粗筛。
  - Go 里解析 `snapshot_json` 并补充字段匹配。
  - 返回结果中带上 `ownerUserId`。

- [√] RED - 前端测试
  - 覆盖搜索参数下发。
  - 覆盖结果渲染与空态。
  - 覆盖清空搜索词后的回退行为。

- [√] GREEN - 前端接入
  - 在聊天侧边栏或搜索弹窗中接入新接口。
  - 支持输入 `target_user_id` 片段或名字关键词。
  - 高亮命中内容。

- [√] VERIFY - 回归验证
  - `go test ./...`
  - `cd frontend && npm run build`
  - 如前端新增交互测试，再补对应 `vitest` 命令。

## 执行总结

- RED 证据: `go test ./internal/app -run TestDBUserArchiveService_SearchArchive` 初始失败，原因是 `SearchArchive` 方法不存在；前端 `vitest` 初始失败，原因是 `searchChatArchive` 和 `ChatArchiveSearchPicker.vue` 不存在。
- GREEN 证据: 新增 `/api/chat/archiveSearch`、归档搜索 service、前端 API wrapper 和 `ChatArchiveSearchPicker.vue` 后，相关测试通过。
- VERIFY 证据: `go test ./...`、`cd frontend && npm run build`、相关 Vitest 测试均通过。
