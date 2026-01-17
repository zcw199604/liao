# 任务清单: 修复 SPA 刷新 404（轻量迭代）

目录: `helloagents/history/2026-01/202601171351_fix_spa_refresh_404/`

---

## 1. 后端路由回退
- [√] 1.1 调整 `internal/app/static.go` 的 SPA 回退逻辑：非静态文件路径回退 `index.html`
- [√] 1.2 补充 Go 单测覆盖刷新与静态资源 404 行为（`internal/app/spa_handler_test.go`）

## 2. 文档更新
- [√] 2.1 更新 `helloagents/wiki/api.md`：补充 Go 版 SPA 回退说明
- [√] 2.2 更新 `helloagents/CHANGELOG.md`：记录修复项

## 3. 测试
- [√] 3.1 运行 `go test ./...`（至少覆盖 `internal/app` 的新增用例）

## 4. 方案包归档
- [√] 4.1 迁移方案包到 `helloagents/history/2026-01/202601171351_fix_spa_refresh_404/` 并更新 `helloagents/history/index.md`
