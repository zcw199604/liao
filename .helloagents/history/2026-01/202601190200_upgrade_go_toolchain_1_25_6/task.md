# 轻量迭代任务清单

- [√] 升级 Go 模块版本：`go 1.22` → `go 1.25`，并指定 `toolchain go1.25.6`
- [√] 更新 GitHub Actions（Release 工作流）Go 版本来源：从 `go.mod` 读取（不固定 patch）
- [√] 更新 Docker 构建镜像：固定为 `golang:1.25.6-alpine`
- [√] 运行 `go test ./...` 验证通过
- [√] 同步知识库版本信息（`helloagents/project.md`、`helloagents/wiki/arch.md`）
- [√] 记录变更到 `helloagents/CHANGELOG.md`
