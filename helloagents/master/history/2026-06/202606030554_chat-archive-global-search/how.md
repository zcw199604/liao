# 怎么做

## 方案决策
新增一个独立的全局归档搜索能力，优先保持现有 `contactCandidates` 的职责不变。

推荐接口：`GET /api/chat/archiveSearch`

原因：
- `contactCandidates` 语义是“来源身份候选聚合”，必须保留 `sourceIdentityId`。
- 本需求是“跨身份归档搜索”，语义不同，单独接口更清晰。

## 数据流
前端搜索框 -> 新接口 -> 后端 handler -> 归档搜索 service -> `chat_user_archive` -> 解析 `snapshot_json` -> 返回结果。

## 检索策略
1. SQL 先做粗筛：
   - `target_user_id LIKE ?`
   - `COALESCE(snapshot_json, '') LIKE ?`
2. Go 里再对 `snapshot_json` 做 `json.Unmarshal`，补充匹配：
   - `nickname`
   - `name`
   - `targetUserName`
   - `area/address`
   - `lastMsg`
3. 返回时统一去敏感字段，避免把 cookie/token 一类内容暴露出去。

## 跨库兼容
- 继续使用 `?` 占位，由 `database.DB` 的 `Rebind` 处理 MySQL / PostgreSQL 差异。
- 不依赖 MySQL/PostgreSQL 原生 JSON 函数，避免双库语义分叉。
- `LIKE` 采用统一的小写比较策略，保证大小写体验一致。

## 数据契约
返回结果建议包含：
- `ownerUserId`
- `targetUserId`
- `nickname` / `name` / `targetUserName`
- `lastMsg`
- `lastTime`
- `snapshot`
- `localArchived`

## 风险与取舍
- `LIKE '%keyword%'` 会降低索引收益，先用小范围分页和 limit 控制。
- `snapshot_json` 为空或坏 JSON 时，不能让整批查询失败，只跳过该条或退回粗筛结果。
- 不按 `owner_user_id` 过滤会扩大结果集，必须保留 `limit`，后续如数据量增大再考虑全文索引或冗余搜索列。

## 备选方案
1. 复用 `contactCandidates` 并把 `owner_user_id` 设为可选。  
   不推荐，职责会混乱。
2. 直接用数据库原生 JSON 查询。  
   不推荐，MySQL / PostgreSQL 兼容成本高。
3. 新增独立搜索接口。  
   推荐，语义最清晰，回归风险最低。

