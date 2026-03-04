# 轻量迭代：历史/收藏用户列表本地缓存 + 并发缓存增强

> 目标：
> 1) 减少 `/api/getHistoryUserList`、`/api/getFavoriteUserList` 对上游的调用次数（本地缓存默认 1 小时）。
> 2) 缓存增强阶段（用户信息/最后消息）并发获取，降低总耗时。
> 3) 在对 Redis 写入（如保存 lastMsg）时，同步更新本地缓存，尽量保持列表预览新鲜。

## 任务清单

- [√] 本地缓存：为历史/收藏用户列表增加 L1 内存缓存（默认 TTL=1 小时）
- [√] 上游调用：命中本地缓存时直接返回，跳过上游请求
- [√] 并发优化：用户信息与最后消息的批量读取并发执行（避免并发写 map）
- [√] 缓存同步：`SaveLastMessage` 写入时同步更新本地用户列表缓存（若命中对应用户）
- [√] 文档同步：更新 README / Wiki（说明本地缓存与 TTL）
- [√] 变更记录：更新 `helloagents/CHANGELOG.md`
- [√] 质量验证：通过 Docker 运行 `go test ./...`
- [√] 迁移方案包：移动到 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
