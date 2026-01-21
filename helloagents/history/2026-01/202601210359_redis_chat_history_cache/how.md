# 技术设计: Redis 聊天记录缓存与历史合并

## 技术方案

### 核心技术
- Go 1.25（`github.com/redis/go-redis/v9`）
- 单元测试：`github.com/alicebob/miniredis/v2`（Redis 行为模拟）

### 实现要点
- 新增 `ChatHistoryCacheService` 接口与 Redis 实现，缓存消息以适配上游 `contents_list` 格式。
- 新增配置：
  - `CACHE_REDIS_CHAT_HISTORY_PREFIX`：Redis key 前缀（默认 `user:chathistory:`）
  - `CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS`：聊天记录缓存 TTL（默认 30 天）
- 缓存写入来源：
  1. WebSocket 私信消息（`code=7`）：解析消息字段并写入 Redis（best-effort，不影响转发）。
  2. `/api/getMessageHistory` 上游返回：解析 `contents_list` 后批量写入 Redis，用于回填历史。
- 历史查询合并：
  - `/api/getMessageHistory` 并发请求上游与 Redis
  - 以 `Tid/tid` 作为主去重键（缺失时回退 `id+toid+time+content`）
  - 返回顺序保持“上游语义”（`contents_list` 默认 newest-first）
  - Redis 失败 → 仅上游；上游失败且 Redis 有数据 → 返回 Redis；两者皆失败 → 返回错误

## 架构决策 ADR
### ADR-001: Redis 聊天记录采用「ZSET index + 单消息 key」模型
**上下文:** 需要支持按 `firstTid`（向前翻页）查询，且希望消息 TTL 为 30 天可配置。  
**决策:** 每个会话维护一个 ZSET 索引（score=tid，member=tid），每条消息存为独立 string key 并设置 TTL。  
**理由:**
- 支持按 tid 范围查询（`ZREVRANGEBYSCORE`）
- 单消息 TTL 易于实现（`SETEX`），无需额外定时清理任务
- 读取可按 tid 批量 GET，并在缺失时清理索引项
**替代方案:**
- 方案A：单 key 存整个列表（JSON）→ 写入/合并成本高，易出现并发覆盖
- 方案B：Hash + ZSET → 无法对 Hash field 设 TTL，需要额外清理机制
**影响:** Redis key 数量随消息量增长，需要依赖 TTL 与按需清理索引来控制规模。

## API 设计
### [POST] /api/getMessageHistory
- **请求:** 不变（沿用 `myUserID/UserToID/isFirst/firstTid/...`）
- **响应:** 维持上游新格式（`code` + `contents_list`）；在上游异常但 Redis 命中时，返回同结构的缓存结果。

## 数据模型（Redis）
以 `conversationKey = min(userId1,userId2) + "_" + max(userId1,userId2)` 为会话 key。

- 索引 ZSET：`{CACHE_REDIS_CHAT_HISTORY_PREFIX}{conversationKey}:index`
  - score: `tid`（可解析为 int64）
  - member: `tid`
- 单消息：`{CACHE_REDIS_CHAT_HISTORY_PREFIX}{conversationKey}:msg:{tid}`
  - value: 上游 `contents_list` 的单条消息 JSON
  - ttl: `CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS`

## 安全与性能
- 缓存写入为 best-effort：Redis 异常不影响主流程。
- 读取合并使用并发，避免串行等待上游 + Redis。
- 避免在日志中输出敏感配置值；Redis URL 解析失败时不回显原始连接串。

## 测试与部署
- 测试：
  - Redis 缓存保存/查询/索引清理（miniredis）
  - Handler：上游失败/为空时 Redis 补齐、上游正常时合并去重
- 部署：
  - `CACHE_TYPE=redis` 时启用聊天记录缓存；按需配置 Redis 连接与 TTL 环境变量。

