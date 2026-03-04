# 任务清单: tests-boundary-cases

> **@status:** completed | 2026-01-22 00:46

目录: `helloagents/archive/2026-01/202601220044_tests-boundary-cases/`

---

## 任务状态符号说明

| 符号 | 状态 | 说明 |
|------|------|------|
| `[ ]` | pending | 待执行 |
| `[√]` | completed | 已完成 |
| `[X]` | failed | 执行失败 |
| `[-]` | skipped | 已跳过 |
| `[?]` | uncertain | 待确认 |

---

## 执行状态
```yaml
总任务: 7
已完成: 7
完成率: 100%
```

---

## 任务列表

### 1. 后端（Go）边界测试

- [√] 1.1 补齐 Identity handlers 边界/异常用例
  - 文件: `internal/app/identity_handlers_test.go`
  - 验证: `go test ./...`

- [√] 1.2 补齐 Favorite 删除接口边界/错误忽略用例
  - 文件: `internal/app/favorite_handlers_test.go`
  - 验证: `go test ./...`

- [√] 1.3 补齐抖音下载 Content-Range total 解析边界用例
  - 文件: `internal/app/douyin_handlers_test.go`
  - 验证: `go test ./...`

- [√] 1.4 补齐上传 URL 文件名/远程路径提取边界用例
  - 文件: `internal/app/media_upload_test.go`
  - 验证: `go test ./...`

- [√] 1.5 格式化并跑后端测试
  - 验证: `gofmt -w ...` + `go test ./...`

### 2. 前端（Vue/Vitest）边界测试

- [√] 2.1 补齐 messageSegments 解析/回退/预览的边界用例
  - 文件: `frontend/src/__tests__/messageSegments.test.ts`
  - 验证: `cd frontend && npm test`

- [√] 2.2 跑前端单测与构建
  - 验证: `cd frontend && npm test` + `npm run build`

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
| 1.x | completed | 采用 sqlmock/table-driven 测试，避免真实 DB/网络依赖 |
| 2.x | completed | 补齐异常 token 输入与预览兜底，防止解析出错 |
