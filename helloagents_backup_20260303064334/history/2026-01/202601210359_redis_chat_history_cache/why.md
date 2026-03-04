# 变更提案: Redis 聊天记录缓存与历史合并

## 需求背景
上游服务器对用户聊天记录的可查询时间较短，导致用户在“加载更多历史消息”时可能拿不到完整历史。本变更引入 Redis 作为长期缓存（默认 1 个月），在获取历史消息时并发读取“上游 + Redis”，合并去重后返回，以延长可回溯时间并保持接口兼容性。

## 变更内容
1. 新增 Redis 聊天记录缓存：按会话维度存储消息，默认缓存 30 天并支持环境变量配置。
2. `/api/getMessageHistory`：并发获取上游与 Redis 数据，合并去重后返回（保持上游返回格式）。
3. WebSocket 私信消息（`code=7`）在服务端写入 Redis，保证实时消息可被后续历史查询命中。

## 影响范围
- **模块:**
  - Go 后端：`internal/app`（UserHistory/WebSocket）与 `internal/config`
  - 知识库：`helloagents/wiki/*`、`helloagents/CHANGELOG.md`
- **文件:**
  - `internal/config/config.go`
  - `internal/app/app.go`
  - `internal/app/user_history_handlers.go`
  - `internal/app/websocket_manager.go`
  - `internal/app/chat_history_cache*.go`
  - `README.md`、`CLAUDE.md`
  - `helloagents/wiki/api.md`、`helloagents/wiki/data.md`、`helloagents/wiki/arch.md`
- **API:**
  - `POST /api/getMessageHistory`：在上游历史不足/失败时可返回 Redis 缓存补齐结果；在正常情况下合并去重不改变体验。
- **数据:**
  - Redis 新增聊天记录相关 key（见 how.md）。

## 核心场景

### 需求: 聊天记录长期可回溯（默认 1 个月）
**模块:** UserHistory / WebSocket Proxy

#### 场景: 上游历史不足时仍可加载更多
当上游接口返回空或返回条数不足时，服务端从 Redis 读取同一会话的历史记录并合并补齐，前端“加载更多”仍可继续。

#### 场景: 上游正常时保持一致性
上游返回正常时，服务端合并去重（以 `Tid/tid` 为主键）后返回，避免重复消息，保持 `contents_list` 的排序语义不变。

#### 场景: 缓存过期可配置
默认缓存 30 天；支持通过环境变量调整 TTL，以平衡成本与可回溯周期。

## 风险评估
- **风险:** Redis 存储与读写成本上升（消息量可能较大）。
  - **缓解:** 默认 TTL 30 天；按会话拆 key；Redis 读写失败时降级为仅上游。
- **风险:** WebSocket/上游历史消息字段存在差异导致前端解析异常。
  - **缓解:** 缓存以“上游 `contents_list` 消息对象结构”为目标格式；合并逻辑兼容 `Tid/tid`、`id/toid` 等字段大小写差异。

