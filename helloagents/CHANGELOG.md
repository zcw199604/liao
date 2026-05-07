# Changelog

本文件记录项目所有重要变更。
格式基于 Keep a Changelog，版本号遵循语义化版本。

## [Unreleased]

### 新增
- 初始化当前 `helloagents/` 知识库核心文档，覆盖项目概览、架构、API、数据模型、模块规范与历史索引。

### 变更
- 知识库以当前 Go 后端、Vue 前端、Android 客户端、SQL 迁移脚本和 Docker 构建为准；旧备份目录仅作为历史参考。

## [1.0.0] - 2026-01-07

### 新增
- Go 后端重构版本完成，保留 HTTP API、WebSocket 代理、媒体上传、缓存和数据库迁移能力。
- GitHub Release 工作流用于打包发布。
