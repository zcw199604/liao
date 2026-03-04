# 轻量迭代：Redis 写入降频（队列批量 flush）

> 目标：降低 Upstash Redis（按量计费）写请求数量，将频繁 `SET` 改为队列缓冲 + 定时批量 flush。
> 默认：每 60 秒批量写入一次；可通过环境变量（秒）调整。

## 任务清单

- [√] 配置：新增环境变量 `CACHE_REDIS_FLUSH_INTERVAL_SECONDS`（默认 60），用于控制 Redis 写队列 flush 间隔
- [√] Redis 写入：将 `SaveUserInfo` / `SaveLastMessage` 的 Redis `SET` 改为队列缓冲（仍保留本地 LRU 立即写入）
- [√] flush 机制：后台 goroutine 定时批量写入；关闭时尽量 flush 并优雅退出
- [√] 文档同步：更新 README / 知识库的缓存配置说明（强调默认 60s、单位秒、可调整）
- [√] 变更记录：更新 `helloagents/CHANGELOG.md`
- [√] 质量验证：通过 Docker 运行 `go test ./...`
- [√] 迁移方案包：移动到 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
