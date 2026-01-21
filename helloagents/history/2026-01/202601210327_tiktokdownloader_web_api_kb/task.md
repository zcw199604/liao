# 任务清单（轻量迭代）

> **目标**：拉取 `JoeanAmier/TikTokDownloader`，整理其 Web API（接口清单/参数/示例），沉淀到本仓库知识库（`helloagents/wiki/`）。

## 任务

- [√] 拉取 TikTokDownloader 到本地临时目录（记录 commit）
- [√] 枚举 Web API 路由与请求/响应模型（FastAPI）
- [√] 生成知识库文档：`helloagents/wiki/external/tiktokdownloader-web-api.md`
- [√] 更新知识库变更记录：`helloagents/CHANGELOG.md`
- [√] 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`
- [√] 清理临时目录

## 验证

- [√] 确认文档覆盖 `src/application/main_server.py` 中全部路由
- [√] 确认示例请求可被 FastAPI 文档识别（JSON body + 可选 `token` Header）
