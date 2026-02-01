# 项目上下文

## 1. 基本信息

```yaml
名称: Liao
描述: 匿名匹配聊天应用（WebSocket 代理 + 多身份管理 + 媒体上传）
类型: Web应用
状态: 开发中
```

## 2. 技术上下文

```yaml
语言: Go + TypeScript
框架: chi / gorilla/websocket / Vue 3
包管理器: go modules + npm
构建工具: go build / Vite
```

### 主要依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | 1.25.6 | 后端开发语言 |
| chi | 见 go.mod | HTTP 路由 |
| gorilla/websocket | 见 go.mod | WebSocket |
| go-sql-driver/mysql | 见 go.mod | MySQL 驱动 |
| go-redis（可选） | 见 go.mod | 缓存 |
| Vue | 3.5.24 | 前端框架 |
| Vite | 7.2.4 | 构建工具 |
| TypeScript | 5.9.3 | 类型系统 |
| Tailwind CSS | 3.4.19 | 样式框架 |

## 3. 项目概述

### 核心功能

- WebSocket 双向代理（客户端连接 `/ws`，后端连接上游并转发消息）
- 身份管理（多身份 CRUD）
- 消息转发（实时双向转发）
- 媒体上传（图片/视频/文件上传与历史记录）
- JWT 认证（访问码登录 + Token 鉴权）
- Redis 缓存（可选：用户信息/最后消息/聊天记录）

### 项目边界

```yaml
范围内:
  - 代理上游 WebSocket 消息与媒体相关 API
  - 提供匿名匹配聊天 UI
范围外:
  - 上游服务的业务实现（本项目只做代理与兼容对齐）
```

## 4. 开发约定

### 代码规范

```yaml
Go:
  入口: cmd/liao/
  业务实现: internal/
接口路径: camelCase（项目约定）
前端:
  工程: frontend/
  构建产物: src/main/resources/static/（gitignored，运行时由 Go 服务托管）
```

### 常用命令

- 后端启动: `go run ./cmd/liao`
- 后端测试: `go test ./...`
- 后端构建: `go build ./cmd/liao`
- 前端开发: `cd frontend && npm run dev`
- 前端构建: `cd frontend && npm run build`

### 配置约定

- 运行时配置通过环境变量提供（Go 服务不读取 `src/main/resources/application.yml`，该文件仅作为 env 默认值对照表保留）
- 本地运行至少需要（按实际功能启用情况）:
  - `DB_URL`, `DB_USERNAME`, `DB_PASSWORD`
  - `WEBSOCKET_UPSTREAM_URL`
  - `AUTH_ACCESS_CODE`, `JWT_SECRET`

### 错误处理

```yaml
策略: 接口尽量对齐上游行为；失败时优先返回上游原始响应
日志: 关键路径 INFO；异常 ERROR
```

### 测试要求

```yaml
后端: go test ./...
前端: npm run build（CI gate）
```

### Git 规范

```yaml
提交格式: Conventional Commits（feat/fix/refactor/chore）
```

## 5. 当前约束（源自历史决策）

| 约束 | 原因 | 决策来源 |
|------|------|---------|
| Go 服务不读取 `src/main/resources/application.yml` | 仅作为 env 对照表保留 | 项目约定（未建立统一决策ID） |
| 前端构建产物不提交 | 静态产物目录 gitignored | 项目约定 |

## 6. 已知技术债务（可选）

| 债务描述 | 优先级 | 来源 | 建议处理时机 |
|---------|--------|------|-------------|
| legacy 文档路径 `helloagents/wiki/...` 的内部引用尚未全量迁移到 `helloagents/modules/...` | P2 | 本次知识库升级 | 后续新方案包优先使用新路径，并逐步修正模块文档内部链接 |
