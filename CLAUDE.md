# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个匿名匹配聊天应用，采用**前后端分离架构**：
- **后端**：Go 1.25.6 + WebSocket 代理 + MySQL（可选 Redis）
- **前端**：Vue 3 + Vite + TypeScript + Pinia + Tailwind CSS

核心功能是作为WebSocket代理服务器，连接上游聊天服务，并为多个客户端提供身份管理、消息转发、媒体上传等功能。

## 架构设计

> 重要说明：
> - `src/main/java/` 与 `pom.xml` 为历史 Java(Spring Boot) 实现，**已弃用，仅供参考**（详见 `src/README.md`）。
> - 当前可运行后端以 Go 版本为准：入口 `cmd/liao/main.go`，核心实现位于 `internal/`。

### 后端架构（Go）

**目录结构**：
- `cmd/liao/` - Go 服务入口（加载配置、启动 HTTP Server、优雅关闭）
- `internal/app/` - 业务实现（HTTP API + `/ws` 代理 + MySQL/缓存/上传/静态托管）
- `internal/config/` - 配置加载（以环境变量为主；变量名/默认值对齐 `src/main/resources/application.yml` 的约定）

**关键技术点**：
1. **WebSocket 双向代理**：下游 `/ws` ↔ 上游 WS，按 `userId` 复用连接池
2. **Forceout 防重连**：80 秒内禁止同 `userId` 重连上游（避免 IP 被封）
3. **JWT 认证**：访问码登录换取 Token；除登录/校验外其余 `/api/**` 需 Bearer Token
4. **媒体上传**：上传落盘到 `./upload`，并维护本地记录/缓存

### 前端架构（Vue 3）

**目录结构**：`frontend/src/`
- `types/` - TypeScript类型定义
- `utils/` - 工具函数
- `constants/` - 常量配置
- `api/` - Axios接口封装
- `stores/` - Pinia状态管理
- `composables/` - 组合式函数（业务逻辑）
- `components/` - UI组件
  - `common/` - 通用组件
  - `chat/` - 聊天相关
  - `identity/` - 身份管理
  - `settings/` - 设置面板
- `views/` - 路由页面

**核心设计模式**：
- Composition API + `<script setup>`
- Pinia管理全局状态（auth、identity、websocket）
- Composables抽离业务逻辑（useWebSocket、useChat等）

## 开发命令

### 后端（Go）

```bash
# 开发启动（默认端口 8080）
go run ./cmd/liao

# 运行测试
go test ./...

# 构建二进制
go build ./cmd/liao
```

### 前端（Vue 3 + Vite）

```bash
cd frontend

# 安装依赖
npm install

# 开发模式（需要后端运行在8080端口）
npm run dev
# 访问 http://localhost:3000

# 构建生产版本
npm run build

# 预览生产构建
npm preview
```

### Docker构建

```bash
# 构建镜像（包含前后端）
docker build -t liao:latest .

# 运行容器（需配置环境变量）
docker run -d -p 8080:8080 \
  -e DB_URL="jdbc:mysql://host:3306/hot_img?..." \
  -e DB_USERNAME=root \
  -e DB_PASSWORD=yourpassword \
  -e WEBSOCKET_UPSTREAM_URL=ws://upstream-host:9999 \
  -e AUTH_ACCESS_CODE=your-access-code \
  -e JWT_SECRET=your-jwt-secret-at-least-256-bits \
  liao:latest
```

## 配置说明

### 环境变量

**必须配置**：
- `DB_URL` - MySQL数据库连接地址
- `DB_USERNAME` / `DB_PASSWORD` - 数据库凭证
- `WEBSOCKET_UPSTREAM_URL` - 上游WebSocket服务地址（如`ws://example.com:9999`）
- `AUTH_ACCESS_CODE` - 访问码（用于登录）
- `JWT_SECRET` - JWT密钥（至少256位）

**可选配置**：
- `SERVER_PORT` - 服务端口（默认8080）
- `TOKEN_EXPIRE_HOURS` - Token过期时间（默认24小时）
- `CACHE_TYPE` - `memory` / `redis`（默认 `memory`）
- `REDIS_URL` / `UPSTASH_REDIS_URL` - Redis 连接串（支持 `redis://` / `rediss://`；适合 Upstash，优先级最高）
- `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` / `REDIS_DB` - Redis 连接参数（当未设置 `REDIS_URL` 时生效；`CACHE_TYPE=redis`）
- `CACHE_REDIS_FLUSH_INTERVAL_SECONDS` - Redis 写入批量 flush 间隔（秒，默认60；用于降低写入频率/成本）
- `CACHE_REDIS_LOCAL_TTL_SECONDS` - Redis L1 本地缓存 TTL（秒，默认3600；用于降低 Redis 读频率/提升响应速度）

**说明**：
- `src/main/resources/application.yml` 为历史配置约定文件（Go 运行时不读取该文件），用于记录环境变量名与默认值对齐规则。

### 数据库初始化

创建数据库：`hot_img`

需要的表（服务启动时自动创建）：
- `identities` - 身份表
- `media_upload_history` - 媒体上传历史
- `user_chat_history` - 用户聊天历史

## 开发规范

### 代码注释
- **尽量使用中文注释**
- 接口、类、复杂逻辑必须添加注释

### API接口命名
- **使用驼峰命名**
- 示例：`/api/getUserList` 而非 `/api/get_user_list`

### 前端开发流程
- **写完前端代码后必须执行编译验证**
- 执行 `npm run build` 确保编译成功无错误
- 只有编译成功后才能认为任务完成
- 如有编译错误，必须修复后重新验证

## 关键业务逻辑

### ForceoutManager机制
- 当上游返回forceout消息时，记录该用户ID和时间戳
- 80秒内禁止该用户重新连接上游（避免IP被封）
- 定时清理过期记录

### 上游连接池（UpstreamWebSocketManager）
- 同一userId只创建一个上游连接
- 支持多个客户端共享同一上游连接
- 最后一个客户端断开后延迟80秒关闭上游连接（允许快速重连）

### JWT认证流程
1. 客户端POST `/api/auth/login` 提交访问码
2. 服务端验证后返回JWT Token
3. WebSocket连接时在URL参数传递Token：`/ws?token=xxx`
4. 后端中间件校验 Token（HTTP Bearer Token / WS token query）

### 静态资源托管
- 默认静态目录候选：`src/main/resources/static/` 或 `static/`（见 `internal/app/app.go`）
- `/upload/**` 映射到本地 `./upload` 目录

## CI/CD

GitHub Actions自动构建：
- 触发：推送到master分支或打tag（`v*`）
- 流程：
  1. 构建前端（npm run build）
  2. 构建 Go 后端
  3. 构建Docker镜像
  4. 推送到Docker Hub（`a7413498/liao`）

## 常见问题

### WebSocket连接失败
- 检查JWT Token是否有效
- 检查`WEBSOCKET_UPSTREAM_URL`配置
- 查看后端日志中的连接错误

### 数据库连接失败
- 确认MySQL已启动
- 检查`DB_URL`中的主机、端口、数据库名
- 确认数据库用户有CREATE TABLE权限

### 前端代理不生效
- 确保后端运行在8080端口
- 检查`vite.config.ts`中的proxy配置
- 开发模式下前端运行在3000端口
