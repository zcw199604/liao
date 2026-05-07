# 任务清单: Go 后端重构（API/WS 100%兼容）

目录: `helloagents/plan/202601071248_go_backend_rewrite/`

---

## 1. Go 工程骨架
- [√] 1.1 初始化 Go 模块与目录结构（cmd/internal），实现 HTTP 服务启动与优雅退出，验证 why.md#需求-api-ws-100兼容-场景-鉴权失败返回-401-且前端跳转登录
- [√] 1.2 实现配置加载（env 默认值对齐 `application.yml`），含 JDBC `DB_URL` 解析为 DSN，验证 why.md#需求-api-ws-100兼容
- [√] 1.3 实现中间件：CORS（全放开）、JWT 拦截（仅放行 `/api/auth/*`）、统一错误 JSON 输出，验证 why.md#需求-api-ws-100兼容-场景-鉴权失败返回-401-且前端跳转登录
- [√] 1.4 实现静态资源托管 + SPA 回退 + `/upload/**` 文件服务，验证 why.md#需求-api-ws-100兼容

## 2. 数据层（MySQL）
- [√] 2.1 MySQL 连接池与健康检查，完成 `identity` 表的 `CREATE TABLE IF NOT EXISTS`，验证 why.md#需求-api-ws-100兼容
- [√] 2.2 实现 `identity` DAO（CRUD + last_used_at 排序），验证 why.md#需求-api-ws-100兼容
- [√] 2.3 实现 `chat_favorites` DAO（去重、列表、删除），验证 why.md#需求-api-ws-100兼容
- [√] 2.4 实现 `media_file` / `media_send_log` DAO（查询/分页/更新/删除），验证 why.md#需求-媒体上传-图片库行为一致
- [√] 2.5 保留 `media_upload_history` 的读取能力（MD5→local_path），验证 why.md#需求-媒体上传-图片库行为一致

## 3. Auth（认证）
- [√] 3.1 实现 `POST /api/auth/login`（访问码校验 + JWT 签发），对齐状态码与返回体，验证 why.md#需求-api-ws-100兼容
- [√] 3.2 实现 `GET /api/auth/verify`（Bearer 校验），对齐返回体，验证 why.md#需求-api-ws-100兼容

## 4. Identity（身份管理）
- [√] 4.1 实现 `GET /api/getIdentityList`，验证 why.md#需求-api-ws-100兼容
- [√] 4.2 实现 `POST /api/createIdentity`、`/quickCreateIdentity`，验证 why.md#需求-api-ws-100兼容
- [√] 4.3 实现 `POST /api/updateIdentity`、`/updateIdentityId`、`/deleteIdentity`、`/selectIdentity`，验证 why.md#需求-api-ws-100兼容

## 5. Favorite（本地收藏）
- [√] 5.1 实现 `POST /api/favorite/add`、`/remove`、`/removeById`、`GET /api/favorite/listAll`、`GET /api/favorite/check`，验证 why.md#需求-api-ws-100兼容

## 6. 缓存子系统（内存/Redis）
- [√] 6.1 实现内存缓存：用户信息/最后消息/图片缓存/forceout 禁止列表，验证 why.md#需求-上游历史-收藏列表缓存增强一致
- [√] 6.2 实现可选 Redis 缓存（含 L1 本地缓存与批量 multi-get），验证 why.md#需求-上游历史-收藏列表缓存增强一致

## 7. 上游 HTTP 代理（v1.chat2019.cn）
- [√] 7.1 实现上游 HTTP 客户端（15s 超时，Header/Cookie 规则一致），验证 why.md#需求-上游历史-收藏列表缓存增强一致
- [√] 7.2 实现 `/api/getHistoryUserList`、`/api/getFavoriteUserList`（含列表增强与 timing 日志），验证 why.md#需求-上游历史-收藏列表缓存增强一致
- [√] 7.3 实现 `/api/reportReferrer`、`/api/getMessageHistory`（含最后消息缓存写入），验证 why.md#需求-上游历史-收藏列表缓存增强一致
- [√] 7.4 实现 `/api/getImgServer`、`/api/deleteUpstreamUser`、`/api/toggleFavorite`、`/api/cancelFavorite`，验证 why.md#需求-api-ws-100兼容

## 8. 媒体上传与图片库（本地文件 + 上游上传）
- [√] 8.1 实现文件存储：MIME 校验、MD5、保存/删除/读取、路径规范化，验证 why.md#需求-媒体上传-图片库行为一致
- [√] 8.2 实现图片服务器地址管理与端口探测（800ms 超时，端口优先级一致），验证 why.md#需求-媒体上传-图片库行为一致
- [√] 8.3 实现 `/api/uploadMedia`、`/api/uploadImage`、`/api/getCachedImages`，对齐成功/失败返回体，验证 why.md#需求-媒体上传-图片库行为一致

## 9. Media History（本地媒体历史 API）
- [√] 9.1 实现 `/api/recordImageSend`、`/api/getUserUploadHistory`、`/api/getUserSentImages`、`/api/getUserUploadStats`，验证 why.md#需求-媒体上传-图片库行为一致
- [√] 9.2 实现 `/api/getChatImages`、`/api/reuploadHistoryImage`、`/api/getAllUploadImages`，验证 why.md#需求-媒体上传-图片库行为一致
- [√] 9.3 实现 `/api/deleteMedia`、`/api/batchDeleteMedia`（状态码与错误码一致），验证 why.md#需求-媒体上传-图片库行为一致

## 10. WebSocket 代理（/ws）
- [√] 10.1 实现 `/ws` 握手 token 校验 + session 生命周期管理，验证 why.md#需求-api-ws-100兼容-场景-websocket-握手-token-校验
- [√] 10.2 实现下游消息解析与 `act=sign` 注册逻辑，验证 why.md#需求-websocket-上游连接池与-forceout-行为一致
- [√] 10.3 实现上游连接池（最大 2 身份、FIFO 淘汰、80s 延迟关闭、断线处理），验证 why.md#需求-websocket-上游连接池与-forceout-行为一致
- [√] 10.4 实现 forceout（-3/-4）全链路处理与上游消息缓存增强（code=15/7），验证 why.md#需求-websocket-上游连接池与-forceout-行为一致
- [√] 10.5 实现系统接口对 WS 管理器的控制能力（stats/closeAll），验证 why.md#需求-api-ws-100兼容

## 11. Docker 与验证
- [√] 11.1 更新 Dockerfile 为 Go 单进程容器（构建前端静态资源并由 Go 托管），验证 why.md#需求-api-ws-100兼容
- [√] 11.2 执行 `go test ./...` 与关键接口手工验证（含 WebSocket 代理），记录结果（`go test ./...` ✅通过；`go vet ./...` ✅通过）

## 12. 安全检查与文档同步
- [√] 12.1 执行安全检查（输入验证、敏感信息处理、权限控制、文件路径安全、外部请求超时），按 G9
- [√] 12.2 更新知识库（`helloagents/wiki/*`、`helloagents/project.md`、`helloagents/CHANGELOG.md`），对齐 Go 实现现状
