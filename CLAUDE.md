# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个匿名匹配聊天应用，采用**前后端分离架构**：
- **后端**：Go 1.25.6 + WebSocket 代理 + MySQL（可选 Redis）
- **前端**：Vue 3 + Vite + TypeScript + Pinia + Tailwind CSS

核心功能是作为WebSocket代理服务器，连接上游聊天服务，并为多个客户端提供身份管理、消息转发、媒体上传等功能。

## 架构设计

### 后端架构（Go）

入口 `cmd/liao/main.go`，核心实现位于 `internal/`。

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
- `DB_URL` - 数据库连接地址（仅通过 scheme 决定 `mysql`/`postgres`/`postgresql`；兼容 `jdbc:` 前缀）
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
- `CACHE_REDIS_EXPIRE_DAYS` - 用户信息/最后消息缓存 TTL（天，默认7；`CACHE_TYPE=redis`）
- `CACHE_REDIS_CHAT_HISTORY_PREFIX` - 聊天记录缓存 key 前缀（默认 `user:chathistory:`；`CACHE_TYPE=redis`）
- `CACHE_REDIS_CHAT_HISTORY_EXPIRE_DAYS` - 聊天记录缓存 TTL（天，默认30；`CACHE_TYPE=redis`）
- `CACHE_REDIS_FLUSH_INTERVAL_SECONDS` - Redis 写入批量 flush 间隔（秒，默认60；用于降低写入频率/成本）
- `CACHE_REDIS_LOCAL_TTL_SECONDS` - Redis L1 本地缓存 TTL（秒，默认3600；用于降低 Redis 读频率/提升响应速度）
- `WANFENG_ENABLED` - 是否启用「晚风线路」（与 chat2019 线路并列可选，默认 `false`）
- `WANFENG_BASE_URL` - 晚风站点根地址（默认 `https://wanfengniming.store`）
- `WANFENG_WS_URL` - 晚风 WebSocket 地址（可选；留空按 `WANFENG_BASE_URL + /ws` 推导）
- `WANFENG_TIMEOUT_SECONDS` - 调用晚风接口超时（秒，默认30）
- `WANFENG_PROTOCOL_OVERRIDES` - 晚风协议参数覆盖（JSON，用于对方改版时热修）
- `WANFENG_MEDIA_ENABLED` - 是否在晚风线路转发图片/视频（默认 `true`）

**说明**：
- `src/main/resources/application.yml` 为配置约定参考文件（Go 运行时不读取该文件），用于记录环境变量名与默认值对齐规则。

### 数据库初始化

创建数据库：`hot_img`（MySQL 或 PostgreSQL 均可；服务启动时会自动建表）

需要的表（服务启动时自动创建）：
- `identities` - 身份表
- `media_upload_history` - 媒体上传历史
- `user_chat_history` - 用户聊天历史
- `identity_location_setting` - 身份位置公开设置（默认不公开）

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

### 多线路（Provider）适配层
- 前端 `/ws?token=...&provider=<id>` 选择线路：缺省/`chat2019` 走原有通路，`wanfeng` 走 `internal/app/websocket_wanfeng.go` → `WanfengManager`
- 线路列表由 `/api/provider/list` 下发，前端存在 `localStorage['liao-ws-provider']`，切换线路会断开并重连
- 晚风线路协议差异（加密信封、帧动作名、字段名）**全部集中在** `internal/upstream/wanfeng/protocol.go` + `adapter.go`；对方改版时优先改这里或通过 `WANFENG_PROTOCOL_OVERRIDES` 覆盖，不要散落到业务代码
- 晚风身份（`uid` + `identityToken` + 独立 `deviceId`）必须落库到 `identity_provider_binding`，否则重连会变成新用户/触发对方同设备风控
- 图片/视频：前端照旧上传到 `/api/upload`（chat2019 媒体服务器），后端 `internal/app/wanfeng_media.go` 在两条线路之间搬运字节；对前端下发仍是 `code=7` 的 `[文件名]`，渲染链路零改动
- **媒体两个必踩坑**（详见 `docs/wanfeng-reverse-engineering.md` 6.6.1）：① 上传接口回参 `data` 是**旧格式 RSA 分段密文**（不是 `v3.` 信封），不解密就会把密文当 `filePath` → 我方发的图对方看不到（解密在 `internal/upstream/wanfeng/legacyrsa.go`）；② 对方媒体地址 `.../niming-file/view/<id>` **必须带 `thumb=1`**，裸地址一律 403「禁止访问」→ 对方发的图我方加载失败（`DownloadMedia` 会自动补 `thumb` 重试）。视频地址不需要 `thumb`，所以视频一直是好的；③ **对方媒体只有约 30 分钟有效期**（上传回参 `expireAt`），过期后 `404 文件不存在或已失效` 且**救不回来**：我方出站上传后立刻调 `refresh-expiry` 续期（POST-only，缺 `userId` 会报「参数不完整」），历史里的过期媒体不再整条丢弃，而是替换成 `[图片已过期]`/`[视频已过期]` 文字占位 + 10 分钟失败记忆；④ **「晚风文件ID → 我方文件名」必须持久化**（表 `wanfeng_media_map`）：前端刷新历史是整体替换消息列表，映射只放内存缓存时，缓存一过（30 分钟）+ 对方文件已过期 → 聊天区里的图/视频会「看几秒突然消失」，所以搬运成功即落库、失败一律给占位文案（`[图片已过期]`/`[图片加载失败]`）
- 历史会话/联系人/聊天记录：老接口（`/api/getHistoryUserList`、`/api/getFavoriteUserList`、`/api/getMessageHistory`）按「**当前线路**」分流，线路由前端请求头 `X-Liao-Provider` 声明（`internal/app/provider_context.go` + `frontend/src/utils/providerHeader.ts`，由 provider store 的 `currentId` 同步）——当前线路是 `wanfeng` 且身份有绑定时走 `internal/app/wanfeng_history.go` → 对方 `home-data`/`favorites`/`chat/messages`，其余走 chat2019；**不能只按身份绑定分流**：`identity_provider_binding` 只增不删，用户从晚风切回 chat2019 后若仍按绑定分流，刷新页面会被晚风侧列表整体替换（chat2019 的会话含刚匹配的人都消失）。前端未带该请求头（老客户端）时才回落到「按绑定分流」的旧行为；匹配偏好/位置设置的 `supported` 判定同理。晚风侧取列表失败一律返回 502（不再 200+空数组），前端 `loadHistoryUsers`/`loadFavoriteUsers` 失败时保留现有列表而不是清空
- **身份 Cookie 是历史接口的前置条件**：对方这几个接口按 Cookie `wf_identity` 鉴权（`uid` 只是业务主键），Cookie 缺失/错配一律 `HTTP 403 + code 40300「请求错误，请刷新页面」`。所以历史类调用必须用 `internal/app/wanfeng_client_cache.go` 里「按身份克隆、独立 Cookie Jar、已绑定身份 Cookie」的客户端，收到 40300 时重建 Cookie 重试一次；另外 `account/identity-cookie` 是 POST-only 接口，**不能带 `X-Get-Via-Post`**（带了会 404，身份 Cookie 永远换不回来）
- 拉黑/举报/撤回/已读：前端 act 在 `adapter.go` 的 `BuildCommand` 里翻译成 `block_user`/`unblock_user`/`report_user`/`recall_message`/`read_messages`；对方消息会自动回 `read_messages`（带 `tids`，对方界面才精确变已读）。
- **已读回执显示**：`messages_read` → `Protocol.ReadReceiptMessage` → 下游 **`code=31 {tids, readAt, targetUid}`** → 前端 `messageStore.markMessagesReadByPeer` 按 tid 标记自己的消息，气泡下显示「已读/未读」；自己消息的 `readByRecipient` 随 `code=7`/历史接口透传（chat2019 没有该字段，所以不显示）。对方站点是`按视口可见`发已读、我方是`收到即回`，这是既有差异（见 docs §5.5/5.6）
- **表情按线路分两套**：晚风用对方站点那 120 个 Unicode 表情（`frontend/src/constants/emoji.ts` 的 `wanfengEmojiList`，`EmojiPanel.vue` 按线路二选一），chat2019 保持原有图片表情 `emojiMap`；两边互不混用
- **掷骰子（晚风独有）**：对方没有独立消息类型，就是发一条普通文本 `🎲 掷出了 N`（N=1~100 随机），**5 秒冷却纯前端**；我方 `ChatInput.vue`（按钮+倒计时）+ `ChatRoomView.vue` 发送，按钮只在晚风线路显示
- **匹配状态机在服务端**：房间还在时 `join_match` 会被拒绝（`error` 「当前状态不允许匹配」），而 `cancel_match` 只作用于匹配队列（不离开房间）。所以 `internal/upstream/wanfeng/client.go` 的 `deliver()` 会在房间内重新匹配前先补发 `exit_chat`，房间内的 `randomOut` 也改发 `exit_chat`；收到该错误时自愈一次（`exit_chat` → 重发 `join_match`，10 秒冷却）
- 匹配偏好：本地表 `identity_match_preference`，随 `login` 与 `update_profile` 的 `matchPreference` 字段下发；接口 `/api/getMatchPreference`、`/api/updateMatchPreference`，`supported=false` 表示该线路不支持（chat2019）
- 位置（城市）公开：**默认不公开**，按身份存在 `identity_location_setting`（对方按我方出口 IP 给身份打城市，匹配成功会以 `match_result.partner.city` 显示给对面）。接口 `/api/getLocationSetting`、`/api/updateLocationSetting`，前端在「身份信息」面板里改；晚风线路登录后由 `client.PushShowCity` 用 `update_profile` **增量 patch `{showCity}`** 下发（login 帧没有该字段），改设置时若已连接立即生效。`supported=false` 表示该线路不生效（chat2019 前端写死 `address_show=false`，等同固定不公开）
- 明确不支持：输入状态（对方站点没有该事件）、在线状态实时查询（只有 `favorite_online` 推送）
- 未接入（纯装饰/独立功能线）：聊天特效 `type:effect`（`config/chat-effects` 清单，18 个，按 `minLevel` 分 VIP/SVIP）、话题按钮（`tools/public/topics/active`，本质是发 `【文案】` 文本）、闪图、故事卡片、漂流瓶、分享名片
- 语音仍未做：前端缺音频消息类型，晚风语音会按「文件」渲染（对方录制细节已逆向，见 `docs/wanfeng-reverse-engineering.md` 5.7）
- **逆向知识库**：`docs/wanfeng-reverse-engineering.md`（协议/API/字段/风控全量，含未验证项清单）+ `docs/js-artifacts.md`（原始 JS 片段，改版时逐字对比）。对方改版后的定位流程、重新抓取脚本都在里面，改动前先看它

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
- 确认数据库已启动（MySQL 或 PostgreSQL）
- 检查`DB_URL`中的主机、端口、数据库名
- 确认数据库用户有CREATE TABLE权限

### 前端代理不生效
- 确保后端运行在8080端口
- 检查`vite.config.ts`中的proxy配置
- 开发模式下前端运行在3000端口
