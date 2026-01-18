# Liao - 匿名匹配聊天应用

一个基于WebSocket的匿名聊天代理服务器，支持多客户端身份管理、消息转发和媒体上传功能。

## 技术栈

### 后端技术
- **Go** 1.22 - 开发语言
- **chi** - HTTP 路由
- **gorilla/websocket** - WebSocket（下游服务端 + 上游客户端）
- **MySQL (go-sql-driver/mysql)** - 数据库驱动
- **Redis (可选：go-redis)** - 用户信息/最后消息缓存
- **JWT (golang-jwt/jwt)** - 身份认证

### 前端技术
- **Vue** 3.5.24 - 前端框架
- **Vite** 7.2.4 - 构建工具
- **TypeScript** 5.9.3 - 类型系统
- **Vue Router** 4.6.4 - 路由管理
- **Pinia** 3.0.4 - 状态管理
- **Axios** 1.13.2 - HTTP客户端
- **Tailwind CSS** 3.4.19 - 样式框架
- **VueUse** 14.1.0 - Vue组合式工具库

## 核心功能

- **WebSocket双向代理** - 客户端通过 `/ws` 连接后端，后端连接上游服务并转发消息
- **连接池管理** - 同一用户ID的多个客户端共享一个上游连接
- **身份管理** - 支持多身份CRUD操作
- **消息转发** - 实时消息双向转发
- **媒体上传** - 图片、视频、文件上传和历史记录
- **JWT认证** - 访问码登录和Token鉴权
- **防重连机制** - ForceoutManager防止IP被封（80秒防重连）

## 快速开始

### 环境要求

- **Node.js** 18+ 和 npm
- **MySQL** 8.0+
- **Go** 1.22（本地开发需要；生产建议 Docker）

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd liao
```

2. **配置数据库**
```sql
CREATE DATABASE hot_img CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

3. **配置环境变量**

创建 `application.yml` 或设置环境变量：
```yaml
spring:
  datasource:
    url: ${DB_URL:jdbc:mysql://localhost:3306/hot_img?useSSL=false&serverTimezone=Asia/Shanghai}
    username: ${DB_USERNAME:root}
    password: ${DB_PASSWORD:yourpassword}

websocket:
  upstream:
    url: ${WEBSOCKET_UPSTREAM_URL:ws://upstream-host:9999}

auth:
  access-code: ${AUTH_ACCESS_CODE:your-access-code}

jwt:
  secret: ${JWT_SECRET:your-jwt-secret-at-least-256-bits}
```

4. **安装前端依赖**
```bash
cd frontend
npm install
```

### 运行项目

#### 开发模式

**后端**（端口8080）：
```bash
go run ./cmd/liao
```

**前端**（端口3000）：
```bash
cd frontend
npm run dev
```

访问 http://localhost:3000

#### 生产构建

**构建前端**：
```bash
cd frontend
npm run build
```

**构建后端**：
```bash
go build ./cmd/liao
```

**运行**：
```bash
./liao
```

## 项目结构

### 后端结构
```
cmd/liao/                 # Go 服务入口
internal/
└── app/                  # 业务实现（HTTP API + /ws 代理 + MySQL/缓存/上传）
```

### 前端结构
```
frontend/src/
├── api/                 # API接口封装
├── components/          # UI组件
│   ├── common/          # 通用组件
│   ├── chat/            # 聊天相关
│   ├── identity/        # 身份管理
│   └── settings/        # 设置面板
├── composables/         # 组合式函数（业务逻辑）
├── stores/              # Pinia状态管理
├── types/               # TypeScript类型
├── utils/               # 工具函数
├── constants/           # 常量配置
└── views/               # 路由页面
```

## Docker部署

### 构建镜像
```bash
docker build -t liao:latest .
```

### 运行容器
```bash
docker run -d -p 8080:8080 \
  -e DB_URL="jdbc:mysql://host:3306/hot_img?useSSL=false&serverTimezone=Asia/Shanghai" \
  -e DB_USERNAME=root \
  -e DB_PASSWORD=yourpassword \
  -e AUTH_ACCESS_CODE=your-access-code \
  -e JWT_SECRET=your-jwt-secret-at-least-256-bits \
  liao:latest
```

## 配置说明

### 必需环境变量
- `DB_URL` - MySQL数据库连接地址
- `DB_USERNAME` - 数据库用户名
- `DB_PASSWORD` - 数据库密码
- `AUTH_ACCESS_CODE` - 访问码（用于登录）
- `JWT_SECRET` - JWT密钥（至少256位）

### 可选环境变量
- `SERVER_PORT` - 服务端口（默认8080）
- `TOKEN_EXPIRE_HOURS` - Token过期时间（默认24小时）
- `WEBSOCKET_UPSTREAM_URL` - 上游 WebSocket 地址降级值（默认 `ws://localhost:9999`；正常情况会动态获取）
- `CACHE_TYPE` - `memory` 或 `redis`（默认 `memory`）
- `REDIS_URL` / `UPSTASH_REDIS_URL` - Redis 连接串（支持 `redis://` / `rediss://`，优先级高于传统四元组；适合 Upstash）
- `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` / `REDIS_DB` - Redis 连接参数（当未设置 `REDIS_URL` 时生效；`CACHE_TYPE=redis`）
- `CACHE_REDIS_FLUSH_INTERVAL_SECONDS` - Redis 写入批量 flush 间隔（秒，默认60；用于降低写入频率/成本）
- `CACHE_USERLIST_TTL_SECONDS` - 历史/收藏用户列表本地缓存 TTL（秒，默认3600；用于减少上游调用）

## 开发规范

- 代码注释使用中文
- API接口使用驼峰命名（如 `/api/getUserList`）
- 前端代码完成后必须执行 `npm run build` 验证编译
- 文件路径使用完整绝对路径（Windows环境）

## CI/CD

项目使用GitHub Actions自动构建：
- 触发条件：推送到master分支或打tag（`v*`）
- 构建流程：
  1. 构建前端（npm run build）
  2. 构建 Go 后端
  3. 构建Docker镜像
  4. 推送到Docker Hub（`a7413498/liao`）

发布（GitHub Release）：
- 在 Actions 中运行 `Release` 工作流：默认创建并推送 `v1.0.0` Tag，编译打包产物并创建对应 Release。

## 许可证

[添加许可证信息]

## 贡献

欢迎提交Issue和Pull Request！
