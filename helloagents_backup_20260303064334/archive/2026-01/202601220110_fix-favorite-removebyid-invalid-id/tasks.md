# 任务清单: fix_favorite_removebyid_invalid_id

> **@status:** completed | 2026-01-22 01:11

目录: `helloagents/archive/2026-01/202601220110_fix-favorite-removebyid-invalid-id/`

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
总任务: 4
已完成: 4
完成率: 100%
```

---

## 任务列表

### 1. 后端（Go）

- [√] 1.1 `removeById` 参数校验：id 解析失败返回 400
  - 文件: `internal/app/favorite_handlers.go`
  - 验证: `go test ./...`

- [√] 1.2 更新对应单元测试（非法/空 id）
  - 文件: `internal/app/favorite_handlers_test.go`
  - 验证: `go test ./...`

### 2. 文档与记录

- [√] 2.1 更新 API 文档（Favorite removeById 错误语义）
  - 文件: `helloagents/wiki/api.md`

- [√] 2.2 更新变更记录（CHANGELOG + 归档）
  - 文件: `helloagents/CHANGELOG.md`

---

## 执行备注

> 执行过程中的重要记录

| 任务 | 状态 | 备注 |
|------|------|------|
| 1.1 | completed | id 为空/非法/<=0 统一返回 400 |
| 1.2 | completed | 删除仍忽略 DB 错误，保持现有语义 |
