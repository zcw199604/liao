# 任务清单（轻量迭代）

> **目标**：在既有《TikTokDownloader Web API（FastAPI）整理》基础上，补充可复用的调用封装（SDK草稿）、错误处理约定、字段/默认值映射说明，完善本地知识库可用性。

## 任务

- [√] 补充调用指南：统一 base_url、token Header、超时/重试建议
- [√] 输出 SDK 草稿（Python/Node 任选其一为主，另一个给最小示例）
- [√] 补充错误处理约定（HTTP 状态码 / 业务失败 / 422 校验失败）
- [√] 增补字段说明：默认值/校验/“传空=不覆盖”行为
- [√] 更新知识库变更记录：`helloagents/CHANGELOG.md`
- [√] 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`

## 验证

- [√] 文档可直接指导外部程序调用（至少包含 1 个完整端到端示例：share→detail）
