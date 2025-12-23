# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个匿名匹配聊天应用，采用**前后端分离架构**：
- **后端**：Spring Boot 3.2.1 + Java 17 + WebSocket代理 + MySQL
- **前端**：Vue 3 + Vite + TypeScript + Pinia + Tailwind CSS

核心功能是作为WebSocket代理服务器，连接上游聊天服务，并为多个客户端提供身份管理、消息转发、媒体上传等功能。

## 架构设计

### 后端架构（Spring Boot）

**包结构**：`src/main/java/com/zcw/`
- `config/` - 配置类（WebSocket、JWT、CORS、服务器配置）
- `controller/` - REST API控制器
  - `AuthController` - 访问码认证
  - `IdentityController` - 身份CRUD
  - `MediaHistoryController` - 媒体上传历史
  - `UserHistoryController` - 用户聊天历史
  - `SystemController` - 系统管理（强制断开连接等）
- `websocket/` - WebSocket核心逻辑
  - `UpstreamWebSocketManager` - 管理上游连接池（按userId复用连接）
  - `UpstreamWebSocketClient` - 上游WebSocket客户端（使用Java-WebSocket库）
  - `ProxyWebSocketHandler` - 下游客户端处理器（Spring WebSocket）
  - `ForceoutManager` - 防止重复登录导致IP封禁（80秒防重连机制）
- `service/` - 业务逻辑
  - `IdentityService` - 身份管理（数据库CRUD）
  - `MediaUploadService` - 媒体上传历史记录
  - `ImageServerService` - 图片服务器接口调用
- `model/` - 数据模型
- `util/` - 工具类（JWT等）

**关键技术点**：
1. **WebSocket双向代理**：客户端通过Spring WebSocket连接后端，后端通过Java-WebSocket连接上游服务
2. **连接池管理**：同一用户ID的多个客户端共享一个上游连接，最后一个客户端断开后延迟80秒关闭上游连接
3. **ForceoutManager**：检测forceout消息（被踢下线），防止在80秒内重连导致IP被封
4. **JWT认证**：访问码登录后颁发Token，WebSocket连接使用JWT鉴权

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

### 后端（Spring Boot）

```bash
# 编译项目
mvn clean compile

# 运行测试
mvn test

# 打包（跳过测试）
mvn clean package -DskipTests

# 运行应用（需要先启动MySQL）
mvn spring-boot:run

# 或直接运行JAR
java -jar target/liao-1.0-SNAPSHOT.jar
```

### 前端（Vue 3 + Vite）

```bash
cd frontend

# 安装依赖
npm install

# 开发模式（需要后端运行在8080端口）
npm run dev
# 访问 http://localhost:3000

# 构建生产版本（输出到 ../src/main/resources/static/）
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

### 环境变量（application.yml）

**必须配置**：
- `DB_URL` - MySQL数据库连接地址
- `DB_USERNAME` / `DB_PASSWORD` - 数据库凭证
- `WEBSOCKET_UPSTREAM_URL` - 上游WebSocket服务地址（如`ws://example.com:9999`）
- `AUTH_ACCESS_CODE` - 访问码（用于登录）
- `JWT_SECRET` - JWT密钥（至少256位）

**可选配置**：
- `SERVER_PORT` - 服务端口（默认8080）
- `TOKEN_EXPIRE_HOURS` - Token过期时间（默认24小时）

### 数据库初始化

创建数据库：`hot_img`

需要的表（服务启动时自动创建）：
- `identities` - 身份表
- `media_upload_history` - 媒体上传历史
- `user_chat_history` - 用户聊天历史

## 开发规范

### 代码注释
- **必须使用中文注释**（CLAUDE.md全局规则）
- 接口、类、复杂逻辑必须添加注释

### API接口命名
- **使用驼峰命名**（CLAUDE.md全局规则）
- 示例：`/api/getUserList` 而非 `/api/get_user_list`

### 文件路径
- **必须使用完整绝对路径**，包含盘符和反斜杠（Windows）
- 示例：`D:\workspace-idea\liao\src\main\java\...`

## 关键业务逻辑

### ForceoutManager机制
- 当上游返回forceout消息时，记录该用户ID和时间戳
- 80秒内禁止该用户重新连接上游（避免IP被封）
- 定时任务每5分钟清理过期记录

### 上游连接池（UpstreamWebSocketManager）
- 同一userId只创建一个上游连接
- 支持多个客户端共享同一上游连接
- 最后一个客户端断开后延迟80秒关闭上游连接（允许快速重连）

### JWT认证流程
1. 客户端POST `/api/auth/login` 提交访问码
2. 服务端验证后返回JWT Token
3. WebSocket连接时在URL参数传递Token：`/ws?token=xxx`
4. `JwtWebSocketInterceptor` 拦截并验证Token

### 前端构建集成
- 前端构建输出到 `src/main/resources/static/`
- Spring Boot静态资源服务自动托管前端
- `SpaForwardController` 处理前端路由（SPA历史模式）

## CI/CD

GitHub Actions自动构建：
- 触发：推送到master分支或打tag（`v*`）
- 流程：
  1. 构建前端（npm run build）
  2. Maven打包后端（含前端静态文件）
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
