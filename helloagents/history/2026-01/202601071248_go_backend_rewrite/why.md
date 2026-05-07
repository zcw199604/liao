# 变更提案: Go 后端重构（API/WS 100%兼容）

## 需求背景

当前后端为 Java 17 / Spring Boot 3，包含：
- REST API（鉴权、身份管理、收藏、历史/消息代理、媒体上传与图片库、系统管理）
- WebSocket 双向代理（下游 `/ws` → 上游动态 WS 服务），包含连接池、forceout 处理、缓存增强
- MySQL 持久化 + 可选 Redis 缓存 + 本地 `./upload` 文件存储 + SPA 静态资源托管

现状问题与驱动：
- **内存占用偏高**（启动后常驻约 700MB+），并且部署以 JVM 为核心运行时。
- 用户目标：**整体重构为 Go**，并要求：
  1. **100% 接口/逻辑兼容**（HTTP + WebSocket，含状态码与返回结构）
  2. 继续使用 **MySQL + 可选 Redis**
  3. **单容器运行前后端**（Go 托管 SPA 静态资源 + `/upload`）

## 变更内容

1. 新增 Go 后端实现，完整覆盖现有后端接口与 WebSocket 行为（以 `helloagents/wiki/api.md` 为 SSOT）。
2. 保持前端调用方式不变：`API_BASE=/api`、`WS_URL=/ws` 不改动。
3. 保持数据与运行目录兼容：继续使用 MySQL 表结构、`./upload` 文件结构与 `/upload/**` 访问方式。
4. 保持可选缓存模式：默认内存缓存；可通过配置启用 Redis 缓存。
5. 更新 Docker 构建为 **单进程 Go 容器**（内存目标显著低于现状）。

## 影响范围

- **模块：**
  - 认证与鉴权（JWT + 访问码）
  - 身份管理（identity）
  - 本地收藏（chat_favorites）
  - 上游历史/收藏/消息代理（v1.chat2019.cn）
  - 媒体上传与图片库（media_file/media_send_log + `./upload`）
  - WebSocket 代理（/ws，下游→上游）
  - 系统管理（连接统计、断开连接、forceout 管理、上游删用户）
- **API：** 全量对齐 `helloagents/wiki/api.md`
- **数据：** 全量对齐 `helloagents/wiki/data.md`

## 核心场景

### 需求: API/WS 100%兼容
**模块:** 全局
以 `helloagents/wiki/api.md` 为唯一真实来源，Go 版本必须保证：
- 路径、方法、参数名、默认值一致
- HTTP 状态码与响应 JSON 结构一致
- “返回原始上游字符串”与“增强返回 JSON”场景一致
- JWT 拦截规则一致（除登录/验证外均需 Bearer）

#### 场景: 鉴权失败返回 401 且前端跳转登录
前端在收到 HTTP 401 时会清除 Token 并跳转 `/login`。
- 预期结果: Go 返回与现状一致的 401 body（`code=401` + msg）

#### 场景: WebSocket 握手 token 校验
前端通过 `/ws?token=...` 连接。
- 预期结果: token 缺失/无效时握手失败；有效时允许连接并转发消息

### 需求: WebSocket 上游连接池与 forceout 行为一致
**模块:** WebSocket Proxy
- 预期结果: 同一 userId 多下游共享一条上游连接
- 预期结果: 最大 2 个身份并发（FIFO 淘汰 + code=-6 通知 + 延迟关闭）
- 预期结果: forceout（code=-3,forceout=true）触发 5 分钟禁用 + code=-4 拒绝连接

### 需求: 上游历史/收藏列表缓存增强一致
**模块:** User History
`/api/getHistoryUserList`、`/api/getFavoriteUserList` 需在可用时补齐 `nickname/sex/age/address` 以及 `lastMsg/lastTime`。

#### 场景: 上游用户ID字段不固定
上游返回用户ID字段可能为 `id` / `UserID` / `userid` 等。
- 预期结果: 能正确识别对方 userId，并命中会话缓存，填充 `lastMsg/lastTime`

### 需求: 媒体上传/图片库行为一致
**模块:** Media
- `/api/uploadMedia`：本地落盘 + 上游上传 + DB 记录 + 缓存写入；成功返回需包含 `port` 与 `localFilename`
- `/api/getAllUploadImages`：返回 `{port,data,total,page,pageSize,totalPages}`，data 为对象数组（含 `uploadTime/updateTime`）
- `/api/deleteMedia`/`/api/batchDeleteMedia`：状态码与错误码一致

## 风险评估

- **风险: 兼容性边界细节多**
  - 缓解: 以 `helloagents/wiki/api.md` / `helloagents/wiki/data.md` 为 SSOT；实现过程中为关键接口补充契约测试（httptest + WebSocket）。
- **风险: 外部上游依赖不稳定**
  - 缓解: 代码层抽象 upstream client，测试使用可控的 mock server；生产运行保持超时与降级策略一致。
- **风险: 时间字段序列化格式差异**
  - 缓解: 明确对齐 Spring Boot 当前输出格式（特别是 `LocalDateTime`），在 Go 中统一格式化。

