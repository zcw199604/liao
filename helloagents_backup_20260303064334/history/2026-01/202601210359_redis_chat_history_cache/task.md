# 任务清单: Redis 聊天记录缓存与历史合并

目录: `helloagents/history/2026-01/202601210359_redis_chat_history_cache/`

---

## 1. 配置与缓存模块
- [√] 1.1 在 `internal/config/config.go` 增加聊天记录 Redis 配置（prefix/expireDays）
- [√] 1.2 在 `internal/app/chat_history_cache.go` 定义 `ChatHistoryCacheService` 与合并/去重辅助函数
- [√] 1.3 在 `internal/app/chat_history_cache_redis.go` 实现 Redis 聊天记录缓存（ZSET+单消息 key+TTL）

## 2. WebSocket 消息入库
- [√] 2.1 在 `internal/app/websocket_manager.go` 捕获 `code=7` 私信并写入聊天记录缓存（含 id/toid/Tid/time/content 归一化）

## 3. 历史接口并发合并
- [√] 3.1 在 `internal/app/user_history_handlers.go` 的 `/api/getMessageHistory` 并发请求上游+Redis，合并去重后返回
- [√] 3.2 在 `/api/getMessageHistory` 中将上游 `contents_list` 回填到 Redis

## 4. 安全检查
- [√] 4.1 执行安全检查（按G9：敏感信息不落库/不泄露日志、输入校验、Redis 连接串不回显）

## 5. 测试
- [√] 5.1 在 `internal/app/chat_history_cache_redis_test.go` 覆盖 Redis 缓存保存/分页查询/索引清理
- [√] 5.2 在 `internal/app/user_history_handlers_test.go` 覆盖上游失败/为空时 Redis 补齐、上游正常时合并去重

## 6. 文档更新
- [√] 6.1 更新 `README.md`/`CLAUDE.md`：补充聊天记录缓存与环境变量说明
- [√] 6.2 更新知识库：`helloagents/wiki/api.md`、`helloagents/wiki/data.md`、`helloagents/wiki/arch.md`
- [√] 6.3 更新 `helloagents/CHANGELOG.md` 记录本次新增

