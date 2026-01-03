# Liao - 匿名匹配聊天应用

一个基于WebSocket的匿名聊天代理服务器，支持多客户端身份管理、消息转发和媒体上传功能。

## 技术栈

### 后端技术
- **Spring Boot** 3.2.1 - 应用框架
- **Java** 17 - 开发语言
- **Spring Boot Web** - REST API支持
- **Spring Boot WebSocket** - 服务端WebSocket
- **Java-WebSocket** 1.5.6 - 客户端WebSocket（连接上游服务）
- **Spring Boot Data JPA** - 数据持久化
- **MySQL Connector** - 数据库驱动
- **JWT (jjwt)** 0.11.5 - 身份认证
- **Jackson** - JSON处理
- **Lombok** - 代码简化

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

- **WebSocket双向代理** - 客户端通过Spring WebSocket连接后端，后端通过Java-WebSocket连接上游服务
- **连接池管理** - 同一用户ID的多个客户端共享一个上游连接
- **身份管理** - 支持多身份CRUD操作
- **消息转发** - 实时消息双向转发
- **媒体上传** - 图片、视频、文件上传和历史记录
- **JWT认证** - 访问码登录和Token鉴权
- **防重连机制** - ForceoutManager防止IP被封（80秒防重连）

## 快速开始

### 环境要求

- **JDK 17** (推荐使用 Amazon Corretto 17)
- **Node.js** 18+ 和 npm
- **MySQL** 8.0+
- **Maven** 3.6+

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
# 设置JDK环境（如需要）
export JAVA_HOME="C:\Users\MyPC\.jdks\corretto-17.0.16"
export PATH="$JAVA_HOME/bin:$PATH"

# 运行Spring Boot
mvn spring-boot:run
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
# 输出到 ../src/main/resources/static/
```

**打包后端**：
```bash
mvn clean package -DskipTests
```

**运行JAR**：
```bash
java -jar target/liao-1.0-SNAPSHOT.jar
```

## 项目结构

### 后端结构
```
src/main/java/com/zcw/
├── config/              # 配置类（WebSocket、JWT、CORS）
├── controller/          # REST API控制器
│   ├── AuthController           # 访问码认证
│   ├── IdentityController       # 身份管理
│   ├── MediaHistoryController   # 媒体历史
│   └── SystemController         # 系统管理
├── websocket/           # WebSocket核心逻辑
│   ├── UpstreamWebSocketManager # 上游连接池管理
│   ├── UpstreamWebSocketClient  # 上游客户端
│   ├── ProxyWebSocketHandler    # 下游处理器
│   └── ForceoutManager          # 防重连机制
├── service/             # 业务逻辑
├── model/               # 数据模型
└── util/                # 工具类
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
  -e WEBSOCKET_UPSTREAM_URL=ws://upstream-host:9999 \
  -e AUTH_ACCESS_CODE=your-access-code \
  -e JWT_SECRET=your-jwt-secret-at-least-256-bits \
  liao:latest
```

## 配置说明

### 必需环境变量
- `DB_URL` - MySQL数据库连接地址
- `DB_USERNAME` - 数据库用户名
- `DB_PASSWORD` - 数据库密码
- `WEBSOCKET_UPSTREAM_URL` - 上游WebSocket服务地址
- `AUTH_ACCESS_CODE` - 访问码（用于登录）
- `JWT_SECRET` - JWT密钥（至少256位）

### 可选环境变量
- `SERVER_PORT` - 服务端口（默认8080）
- `TOKEN_EXPIRE_HOURS` - Token过期时间（默认24小时）

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
  2. Maven打包后端（含前端静态文件）
  3. 构建Docker镜像
  4. 推送到Docker Hub（`a7413498/liao`）

## 许可证

[添加许可证信息]

## 贡献

欢迎提交Issue和Pull Request！
