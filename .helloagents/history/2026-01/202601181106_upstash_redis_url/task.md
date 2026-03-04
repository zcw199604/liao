# 轻量迭代：支持 Upstash Redis（rediss://）

> 目标：在保持现有 `REDIS_HOST/REDIS_PORT/REDIS_PASSWORD/REDIS_DB` 兼容的前提下，新增支持通过 `rediss://` URL 连接 Upstash Redis（TLS）。
> 注意：Redis URL 可能包含敏感凭据，文档与日志中必须使用占位符，禁止明文写入仓库。

## 任务清单

- [√] 后端配置：支持从环境变量读取 `UPSTASH_REDIS_URL` / `REDIS_URL`（优先级：UPSTASH_REDIS_URL > REDIS_URL > 传统四元组）
- [√] Redis 客户端：当提供 `redis://` 或 `rediss://` URL 时，使用 URL 解析（支持 TLS），并保留原有 host/port/password/db 连接方式
- [√] 文档同步：更新知识库中缓存配置说明（补充 `REDIS_URL` 与 `rediss://` 用法，示例使用占位符）
- [√] 变更记录：更新 `helloagents/CHANGELOG.md`
- [√] 质量验证：通过 Docker 运行 `go test ./...`（本机未安装 Go 时的替代方案）
- [√] 迁移方案包：移动到 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
