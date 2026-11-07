# Liao - 匿名匹配聊天应用

一个基于WebSocket的匿名聊天代理服务器，支持多客户端身份管理、消息转发和媒体上传功能。

## 技术栈

### 后端技术
- **Go** 1.25.6 - 开发语言
- **chi** - HTTP 路由
- **gorilla/websocket** - WebSocket（下游服务端 + 上游客户端）
- **MySQL (go-sql-driver/mysql) / PostgreSQL (pgx)** - 数据库驱动（由 `DB_URL` 的 scheme 决定）
- **Redis (可选：go-redis)** - 用户信息/最后消息/聊天记录缓存
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
- **防重连机制** - Forceout：`code=-3` + `forceout=true` 时 5 分钟内禁止重新 sign；下游断开后上游连接延迟 80 秒关闭

## 快速开始

### 环境要求

- **Node.js** 18+ 和 npm
- **MySQL** 8.0+ 或 **PostgreSQL** 13+（二选一）
- **Go** 1.25.6（本地开发需要；生产建议 Docker）

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd liao
```

2. **配置数据库**

服务支持 MySQL + PostgreSQL（双栈），并且**仅通过 `DB_URL` 的 scheme 选择数据库类型**：
- MySQL: `mysql://` / `jdbc:mysql://`
- PostgreSQL: `postgres://` / `postgresql://` / `jdbc:postgresql://`

```sql
CREATE DATABASE hot_img CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

3. **配置环境变量**

Go 服务通过环境变量读取配置；`src/main/resources/application.yml` 仅作为 env 变量名与默认值对照表保留（运行时不读取）。

```bash
# MySQL 示例（scheme=mysql）
export DB_URL="jdbc:mysql://localhost:3306/hot_img?useSSL=false&serverTimezone=Asia/Shanghai"

# PostgreSQL 示例（scheme=postgres/postgresql）
export DB_URL="postgres://localhost:5432/hot_img?sslmode=disable"

export DB_USERNAME="root"
export DB_PASSWORD="yourpassword"
export AUTH_ACCESS_CODE="your-access-code"
export JWT_SECRET="your-jwt-secret-at-least-256-bits-long-please-change-this-to-random-string"
# 可选：上游 WS 地址降级值（未配置时默认 ws://localhost:9999；正常情况会动态获取）
export WEBSOCKET_UPSTREAM_URL="ws://localhost:9999"
```

可选：接入 Selenium_Crawler 的 CF Worker 媒体预览，在 **图片管理 → X 媒体预览** 中打开：

```bash
export WORKER_PREVIEW_URL="https://your-worker.example.com"
export WORKER_PREVIEW_USERNAME="admin"
# WORKER_PREVIEW_PASSWORD 通过部署平台的 Secret 或本地环境变量注入，勿提交到仓库。
```

`WORKER_PREVIEW_URL` 填 Worker 根地址，不带 `/preview`。配置密码后，liao 后端自动登录并代理页面、查询与媒体下载；账号密码和 Worker Token 不下发浏览器，不要求两个服务使用同一主域。未配置时其他功能照常使用。当前对接 Worker 自有账号密码登录；若另外启用了 Cloudflare Access，还需单独处理 Access 认证。

预览 Cookie 仅可访问 `/worker-preview/`，不能用于普通 liao API，且有效期不超过当前 liao 登录。Worker 页面及其加载的脚本与 liao 共享浏览器源，只应接入自己信任的 Worker。媒体经过 liao 服务转发，视频流量也会经过 liao 部署节点。

4. **安装前端依赖**
```bash
cd frontend
npm install
```

### 页面更新与手机缓存

- HTML 使用内容 hash 作为 ETag，允许浏览器缓存，但每次打开或刷新时校验版本：未变化返回 304，不重复传输 HTML；有变化返回新版内容。`/version.json` 不缓存；带内容 hash 的 JS/CSS 可长期缓存，重新构建后使用新的资源地址。
- 已打开的页面会在回到前台及每分钟检测更新，发现新版本后显示“刷新”按钮；不会自动打断聊天，也不会清除登录、收藏等本地数据。旧的懒加载资源失效时也会提示刷新。
- **仅替换 Go 二进制不会更新 Vue 页面。** 前端有改动时，需要执行 `cd frontend && npm run build`，并部署生成的静态资源；重建 Docker 镜像会自动包含前端构建。
- 反向代理或 CDN 不应强制缓存 HTML、`/version.json` 和 `/worker-preview/*`。历史边缘缓存若已存在，需在部署时清除对应 HTML 缓存；新策略生效后，手机正常刷新即可，无需清理浏览器数据。

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
└── app/                  # 业务实现（HTTP API + /ws 代理 + MySQL/PostgreSQL/缓存/上传）
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
  -e UPSTREAM_HTTP_TIMEOUT_SECONDS=60 \
  -e TIKTOKDOWNLOADER_TIMEOUT_SECONDS=60 \
  -e REDIS_TIMEOUT_SECONDS=15 \
  liao:latest
```

## 配置说明

### 必需环境变量
- `DB_URL` - 数据库连接地址（仅通过 scheme 决定 `mysql`/`postgres`/`postgresql`；兼容 `jdbc:` 前缀）
- `DB_USERNAME` - 数据库用户名
- `DB_PASSWORD` - 数据库密码
- `AUTH_ACCESS_CODE` - 访问码（用于登录）
- `JWT_SECRET` - JWT密钥（至少256位）

### 可选环境变量
- `SERVER_PORT` - 服务端口（默认8080）
- `TOKEN_EXPIRE_HOURS` - Token过期时间（默认24小时）
- `UPSTREAM_HTTP_TIMEOUT_SECONDS` - 调用上游 HTTP 接口超时（秒，默认60）
- `WEBSOCKET_UPSTREAM_URL` - 上游 WebSocket 地址降级值（默认 `ws://localhost:9999`；正常情况会动态获取）
- `TIKTOKDOWNLOADER_BASE_URL` - TikTokDownloader Web API 地址（启用抖音下载功能；未配置则该功能返回“未启用”错误）
- `TIKTOKDOWNLOADER_TOKEN` - TikTokDownloader Web API 的 `token` Header（默认上游不校验，可不配）
- `TIKTOKDOWNLOADER_TIMEOUT_SECONDS` - 调用 TikTokDownloader Web API 超时（秒，默认跟随 `UPSTREAM_HTTP_TIMEOUT_SECONDS`）
- `DOUYIN_COOKIE` - 抖音抓取默认 Cookie（可选；页面填写优先；敏感信息建议仅在运行环境中配置）
- `DOUYIN_PROXY` - 抖音抓取默认代理（可选；前端不提供输入）
- `CACHE_TYPE` - `memory` 或 `redis`（默认 `memory`）
- `REDIS_URL` / `UPSTASH_REDIS_URL` - Redis 连接串（支持 `redis://` / `rediss://`，优先级高于传统四元组；适合 Upstash）
- `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` / `REDIS_DB` - Redis 连接参数（当未设置 `REDIS_URL` 时生效；`CACHE_TYPE=redis`）
- `REDIS_TIMEOUT_SECONDS` - Redis 连接/读写超时（秒，默认15；`CACHE_TYPE=redis`）
- `CACHE_REDIS_EXPIRE_DAYS` - 用户信息/最后消息缓存 TTL（天，默认7；`CACHE_TYPE=redis`）
- `CACHE_REDIS_CHAT_HISTORY_PREFIX` - 聊天记录缓存 key 前缀（默认 `user:chathistory:`；`CACHE_TYPE=redis`）
- `CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS` - 聊天记录缓存 TTL（天，默认30；`CACHE_TYPE=redis`）
- `CACHE_REDIS_FLUSH_INTERVAL_SECONDS` - Redis 写入批量 flush 间隔（秒，默认60；用于降低写入频率/成本）
- `CACHE_REDIS_LOCAL_TTL_SECONDS` - Redis L1 本地缓存 TTL（秒，默认3600；用于降低 Redis 读频率/提升响应速度）
- `WANFENG_ENABLED` - 是否启用「晚风线路」（与现有线路并列可选，默认 `false`）
- `WANFENG_BASE_URL` - 晚风站点根地址（默认 `https://wanfengniming.store`；HTTP 接口与 WS 同域）
- `WANFENG_WS_URL` - 晚风 WebSocket 地址（可选；留空按 `WANFENG_BASE_URL + /ws` 推导）
- `WANFENG_TIMEOUT_SECONDS` - 调用晚风接口超时（秒，默认30）
- `WANFENG_PROTOCOL_OVERRIDES` - 晚风协议参数覆盖（JSON，用于对方改版时热修，详见 `docs/wanfeng-niming-integration-plan.md` 与 `docs/wanfeng-reverse-engineering.md`）
- `WANFENG_MEDIA_ENABLED` - 是否在晚风线路转发图片/视频（默认 `true`；关闭后媒体消息降级为系统提示）
- `WANFENG_PROXY_ENABLED` - 晚风 HTTP API、媒体上传/下载及 WS 统一使用 SCF 代理（默认 `false`）
- `SCFPROXY_POOL_FILE` - scfProxy 固定 IP 池 `pool.json` 路径，与 `SCFPROXY_ENDPOINTS_FILE` 二选一
- `SCFPROXY_ENDPOINTS_FILE` - scfProxy endpoint 列表文件路径，每行一个 `wss://` 地址
- `SCFPROXY_TOKEN` - SCF 隧道共享令牌，必须与代理服务端一致，仅通过环境变量提供

晚风代理使用 [`scfproxy`](https://github.com/zcw199604/scfProxy) v0.1.0。固定 IP 池可由 Liao 自动管理，也可复用已有池。仅复用已有池时：

```bash
export WANFENG_ENABLED=true
export WANFENG_PROXY_ENABLED=true
export SCFPROXY_POOL_FILE=/config/pool.json
# 在运行环境中另行注入 SCFPROXY_TOKEN，勿写入仓库。
```

Docker 部署需将池文件所在目录可写挂载到容器，例如 `-v /host/scf-config:/config:rw`，并传入上述环境变量。自动切换需要在该目录写入锁文件并原子更新池文件，不能只挂载单个文件或使用只读挂载。
代理只影响晚风线路，涵盖握手、身份、历史/关系接口、图片/视频上传下载及 WebSocket；按身份克隆的客户端仍独立保存 Cookie，共享代理出口。
启用后配置错误会阻止启动，连接失败会返回错误，不回退直连。HTTPS/WSS 保持目标站点的 TLS 证书校验。
固定池模式下，HTTP 错误响应或 WS 错误中包含 **“您的 IP 已被封禁”**（忽略空白和 IP 大小写），或错误码为 `IP_BANNED` 时，Liao 自动调用 scfProxy 切换到已有备用出口，并记录旧 IP 封禁（库默认 24 小时）。关键字命中不受其他错误码影响；不含该关键字的普通 403、会话/身份失效和超时不触发切换。
切换关闭旧隧道，WS 保留身份并重新连接，后续 HTTP 请求使用新出口；业务请求不自动重放。同一旧出口的并发封禁报告只切换一次。池中至少需要一个备用节点；无备用或写入失败时保留错误、停止该 WS 的重连，不回退直连，服务日志记录切换失败原因。
自动切换只修改本地池文件，不需要腾讯云 SecretId/SecretKey。希望由 Liao 创建节点、清理退役云函数和自动补位时，启用下方托管模式；未启用时补位仍需自行执行 scfProxy CLI。
库仍每约 2 秒热加载外部池变更，因此也支持通过 scfProxy CLI 的 `pool switch` 或 `pool rotate` 手动切换。
使用 endpoint 列表时，改为配置 `SCFPROXY_ENDPOINTS_FILE` 并取消 `SCFPROXY_POOL_FILE`；同一目标主机优先固定到同一 endpoint。列表模式不支持上述业务封禁自动切换，自动切换应使用固定池模式。

### 由 Liao 自动管理 SCF 代理池

Docker 镜像内置 `scfproxy` CLI（v0.1.0）和 `/app/scf-server.zip`（linux/amd64 云函数代码包）。启用 `SCFPROXY_MANAGED=true` 后，Liao 启动时执行 `scfproxy pool replenish ... --yes`，空目录可直接创建主用/备用，无需手工准备 `pool.json`。已有池保留当前主用，并恢复中断的创建/删除操作。

IP 封禁后先切换现有备用，再后台调用 CLI 清理退役函数、补足节点；外部池切换也会触发补位。维护任务串行执行，默认每 60 秒再次维护，失败会记录日志并重试。没有待清理节点且数量充足时 CLI 不创建新资源。启动维护失败会阻止服务启动；运行期补位失败不影响当前可用出口。服务退出会取消 CLI 子进程，保留池文件和已有云函数供下次启动恢复。

在现有 Docker Compose 服务中追加（凭证通过运行环境或受保护的 `.env` 提供，勿提交仓库）：

```yaml
environment:
  WANFENG_ENABLED: "true"
  WANFENG_PROXY_ENABLED: "true"
  SCFPROXY_MANAGED: "true"
  SCFPROXY_POOL_FILE: /config/pool.json
  SCFPROXY_TOKEN: ${SCFPROXY_TOKEN:?请设置隧道共享令牌}
  TENCENTCLOUD_SECRET_ID: ${TENCENTCLOUD_SECRET_ID:?请设置腾讯云SecretId}
  TENCENTCLOUD_SECRET_KEY: ${TENCENTCLOUD_SECRET_KEY:?请设置腾讯云SecretKey}
  SCFPROXY_REGIONS: ap-singapore,ap-tokyo,ap-seoul
  SCFPROXY_POOL_SIZE: "2"
volumes:
  - ./scf-config:/config:rw
```

| 可选环境变量 | 默认值 / 作用 |
| --- | --- |
| `TENCENTCLOUD_SESSION_TOKEN` | 临时腾讯云凭证的会话 Token；长期凭证无需设置 |
| `SCFPROXY_CLI_PATH` | `scfproxy`，从 PATH 查找；也可设置绝对路径 |
| `SCFPROXY_PACKAGE_PATH` | `/app/scf-server.zip`；非 Docker 运行需自行准备部署包 |
| `SCFPROXY_REGIONS` | `ap-singapore,ap-tokyo,ap-seoul`；候选地域至少有 `SCFPROXY_POOL_SIZE` 个不同地域，建议多留一个供轮换 |
| `SCFPROXY_POOL_SIZE` | `2`，包含主用和备用，至少为 2；设为 3 可扩大补位期间的备用余量 |
| `SCFPROXY_POOL_PREFIX` | `scfproxy`；新节点命名前缀，CLI 只删除此前缀且标记为 scfproxy 的池内退役函数；复用旧池应保持原前缀 |
| `SCFPROXY_COMMAND_TIMEOUT_SECONDS` | `600`，单次 CLI 操作超时；创建多个节点或地域较慢时可调高 |
| `SCFPROXY_RECONCILE_INTERVAL_SECONDS` | `60`，周期维护及失败重试间隔 |

该模式会通过腾讯云 API 创建和删除计费云函数，需要相应的 SCF 管理权限。凭证只经环境传入子进程，不进入命令行参数、池文件或日志。持久化 `/config` 目录可使容器重建后继续使用原池；停止服务不销毁云资源。启动时可能需要等待数分钟完成首次建池，部署探活需留出时间。代理节点补位完成前若再遇封禁且没有备用，该连接仍会返回错误；可增大池大小留出余量。

### 怎么切换聊天线路

线路选择入口在**前端聊天页**里，不在环境变量里：

1. 左侧聊天列表左上角，点 **☰（三条杠）菜单**；
2. 选 **「身份信息」**（不是「系统设置」）；
3. 抽屉**最底部**有「聊天线路」区块，点「晚风线路」即可，选中后自动断开重连。

排查「看不到线路选择」：该区块只有**两条以上线路**时才渲染，而后端默认关闭晚风线路。

```bash
# 看当前后端开了几条线路（1 条 = 线路选择块不会出现）
curl -s -H "Authorization: Bearer <你的token>" http://localhost:8080/api/provider/list
```

返回 `providers` 数组里只有「经典线路」，说明没开：

```bash
WANFENG_ENABLED=true   # 加到 .env / docker -e / 部署环境变量后重启服务
```

前端的选择保存在 `localStorage['liao-ws-provider']`（按浏览器记，切换身份不影响），
实际连接地址是 `/ws?token=...&provider=wanfeng`。

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
  5. 企业微信通知（可选）

发布（GitHub Release）：
- 在 Actions 中运行 `Release` 工作流：默认创建并推送 `v1.0.0` Tag，编译打包产物并创建对应 Release。

### 企业微信通知配置

GitHub Actions 支持通过企业微信应用推送构建/发布通知。

**配置步骤**：

1. 在企业微信管理后台创建应用，获取以下参数：
   - 企业 ID（corpid）
   - 应用 Secret（corpsecret）
   - 应用 AgentId
   - 接收人 UserID（或 `@all` 发给所有人）

2. 在 GitHub 仓库 Settings → Secrets and variables → Actions 中添加：
   - `WECHAT_CORP_ID` - 企业 ID
   - `WECHAT_CORP_SECRET` - 应用 Secret
   - `WECHAT_AGENT_ID` - 应用 AgentId
   - `WECHAT_TO_USER` - 接收人 UserID
   - `WECHAT_API_URL`（可选）- 企业微信 API 地址（默认 `https://qyapi.weixin.qq.com`）

配置后，构建成功/失败时会自动推送通知到企业微信。未配置时会自动跳过通知步骤。

## 许可证

[添加许可证信息]

## 贡献

欢迎提交Issue和Pull Request！
