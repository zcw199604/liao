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
| legacy 归档索引（旧结构） | [history/index.md](history/index.md) |

## 知识库状态

```yaml
最后更新: 2026-02-02 04:08
模块数量: 11
待执行方案: 1
```

## 读取指引

```yaml
启动任务:
  1. 读取本文件获取导航
  2. 读取 context.md 获取项目上下文
  3. 读取 modules/_index.md 定位相关模块文档

任务相关:
  - 涉及特定模块: 读取 modules/{模块名}.md
  - 需要历史决策: 搜索 CHANGELOG.md → 读取对应 archive/{YYYY-MM}/{方案包}/proposal.md
  - 继续之前任务: 读取 plan/{方案包}/*
```

## legacy 说明

- `helloagents/wiki/` 为兼容旧链接保留；新文档统一落在 `helloagents/modules/`。
- `helloagents/history/` 为旧归档结构，后续可按需迁移到 `helloagents/archive/`。
