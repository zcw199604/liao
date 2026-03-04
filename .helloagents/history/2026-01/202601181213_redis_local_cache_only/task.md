# 轻量迭代：仅缓存 Redis 数据（不缓存上游列表）+ 并发增强

> 背景：`upstreamMs=0` 说明历史/收藏列表接口命中“本地缓存最终响应”并跳过上游调用；此行为不符合预期。
>
> 目标：
> - 历史/收藏列表接口仍每次请求上游，不缓存上游列表数据
> - 仅对 Redis 读写相关数据做本地缓存（默认 1 小时，单位秒可配置）
> - 用户信息/最后消息的增强读取保持并发执行，降低总耗时

## 任务清单

- [√] 移除上游列表本地缓存：删除 `userListCache`，历史/收藏列表每次仍请求上游
- [√] 本地缓存 Redis 数据：调整 Redis L1 本地缓存 TTL 默认 1 小时，并增加配置 `CACHE_REDIS_LOCAL_TTL_SECONDS`
- [√] 并发增强保留：用户信息/最后消息批量读取并发执行
- [√] 文档同步：更新 README / Wiki / application.yml / CHANGELOG（删除列表本地缓存描述，补充 Redis 本地缓存 TTL 配置）
- [√] 质量验证：通过 Docker 运行 `go test ./...`
- [√] 迁移方案包：移动到 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
