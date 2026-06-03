# 为什么要做

## 背景
- 当前 `GET /api/chat/contactCandidates` 以 `sourceIdentityId` 为中心，适合“从某个身份接入联系人”，不适合全局归档检索。
- 现在需要新增一种搜索：不依赖 `owner_user_id`，按 `target_user_id` 的模糊匹配，以及 `snapshot_json` 内的用户信息字段命中。
- 现有侧边栏本地搜索只过滤当前已加载列表，无法覆盖 `chat_user_archive` 里的全部归档记录。

## 目标
- 支持跨身份查询 `chat_user_archive`。
- 支持 `target_user_id LIKE`。
- 支持从 `snapshot_json` 提取 `nickname`、`name`、`targetUserName` 等字段参与搜索。
- 返回结果时保留 `owner_user_id`，用于定位命中的来源身份。

## 非目标
- 不修改上游历史/收藏代理语义。
- 不把现有 `contactCandidates` 改造成万能接口。
- 不新增独立归档搜索表。

## 成功标准
- 不传 `owner_user_id` 也能查到命中的归档记录。
- `target_user_id` 模糊匹配和 `snapshot_json` 命中都能返回。
- MySQL 与 PostgreSQL 行为一致。
- 有测试覆盖正常、空值、坏 JSON、无命中、跨身份多结果等场景。

