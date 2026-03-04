# Liao 知识库

> 本目录用于沉淀项目知识、变更记录与方案包归档。

## 快速导航

| 需要了解 | 读取文件 |
|---------|---------|
| 项目概况、技术栈、开发约定 | [context.md](context.md) |
| 模块索引与通用文档索引 | [modules/_index.md](modules/_index.md) |
| 项目变更历史 | [CHANGELOG.md](CHANGELOG.md) |
| 方案归档索引 | [archive/_index.md](archive/_index.md) |
| 当前待执行的方案 | [plan/](plan/) |
| 历史会话记录 | [sessions/](sessions/) |
| legacy 归档索引（旧结构） | [history/index.md](history/index.md) |

## 模块关键词索引

> AI 可优先通过关键词定位模块文档，再按需深读。

| 模块 | 关键词 | 摘要 |
|------|--------|------|
| auth | access code, JWT, 登录鉴权 | 登录与令牌校验 |
| chat-ui | Vue, 聊天界面, 交互 | 聊天前端体验与交互规范 |
| db | MySQL, PostgreSQL, 迁移 | 数据库双栈与迁移策略 |
| douyin-downloader | 抖音下载, 导入, 任务 | 抖音链接下载与导入流程 |
| douyin-livephoto | Live Photo, 实况 | 抖音实况资源处理 |
| identity | 身份, 用户切换, 绑定 | 多身份管理与切换 |
| media | 上传, 预览, 画廊 | 媒体上传与预览能力 |
| mtphoto | 相册, 收藏夹, 预览 | mtPhoto 集成与相册能力 |
| user-history | 历史会话, 最后消息, 缓存 | 用户列表与历史消息数据 |
| websocket-proxy | /ws, 上游转发, 连接池 | WebSocket 代理链路 |
| frontend-theme | 主题, 样式, Tailwind | 前端主题与排版约定 |

## 知识库状态

```yaml
kb_version: 2.3.0
最后更新: 2026-03-03 06:43
模块数量: 11
待执行方案: 3
```

## 读取指引

```yaml
启动任务:
  1. 读取本文件获取导航
  2. 读取 context.md 获取项目上下文
  3. 检查 plan/ 是否有进行中方案包

任务相关:
  - 涉及特定模块: 读取 modules/{模块名}.md
  - 需要历史决策: 搜索 CHANGELOG.md → 读取对应 archive/{YYYY-MM}/{方案包}/proposal.md
  - 继续之前任务: 读取 plan/{方案包}/*
```

## legacy 说明

- `.helloagents/wiki/` 为兼容旧链接保留；新文档统一落在 `.helloagents/modules/`。
- `.helloagents/history/` 为旧归档结构，后续可按需迁移到 `.helloagents/archive/`。
