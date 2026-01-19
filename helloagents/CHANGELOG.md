# Changelog

本文件记录项目所有重要变更。
格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/),
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增
- 增强聊天交互：会话列表左右滑切换“消息/收藏”、聊天页边缘右滑返回、侧边栏左滑关闭、长按菜单点击外关闭
- 前端/后端：接入 mtPhoto 相册系统：上传菜单/系统设置新增“mtPhoto 相册”入口，按相册展示图片/视频并支持一键导入上传到上游（导入失败会先落盘到本地，可在“全站图片库”中重试）。
- 后端：mtPhoto 续期新增 refresh_token 支持（优先调用 `/auth/refresh`，失败回退 `/auth/login`），减少频繁重登。
- 前端/后端：支持聊天消息“文字 + `[path]` 媒体占位符”混排渲染（如 `喜欢吗[2026/...jpg]`），聊天气泡与历史预览可图文同显；会话列表 lastMsg 按“文本截断 + 媒体标签”展示。
- 后端/前端：新增全局图片端口策略配置（fixed/probe/real）并在 Settings 面板可视化切换；WS/历史聊天媒体（图片/视频）按策略解析端口。
- 后端：Redis 缓存支持 `UPSTASH_REDIS_URL`/`REDIS_URL`（支持 `rediss://` TLS），便于接入 Upstash Redis。
- 后端：Redis 写入改为队列批量 flush（默认 60 秒，可通过 `CACHE_REDIS_FLUSH_INTERVAL_SECONDS` 调整），降低 Upstash 按量计费成本。
- 后端：历史/收藏用户列表的用户信息/最后消息增强改为并发批量读取，并将 Redis L1 本地缓存 TTL 默认调为 1 小时（`CACHE_REDIS_LOCAL_TTL_SECONDS`），降低 Redis 读频率/提升响应速度。
- 前端：接入 Vitest + jsdom，并为核心模块补充单元测试（utils/time/string、useToast、request、auth store）。
- 前端：补充 Vue 组件级测试（Dialog/Toast/Loading/UserList/ChatSidebar）。
- 前端：补充视图级页面测试（LoginPage/IdentityPicker/ChatListView/ChatRoomView）。
- 前端：补充核心聊天业务测试（useChat/useMessage/useWebSocket、chat/message store）。
- 前端：补充核心聊天与设置/媒体组件测试（ChatInput/MessageList/MessageBubble/EmojiPanel/UploadMenu/MatchButton/SettingsDrawer/MediaPreview）。
- 前端：补充 utils 测试（cookie/file/media）。
- 后端：为 `/api/getHistoryUserList` 与 `/api/getFavoriteUserList` 增加分段耗时日志（上游/补充用户信息/最后消息/总耗时）。
- 后端：新增 Go 版单进程服务（API + `/ws` WebSocket 代理 100%兼容、MySQL + 可选 Redis、静态前端托管），支持单容器运行并降低运行内存。
- 后端：Go 服务日志支持 `LOG_LEVEL`（debug/info/warn/error）与 `LOG_FORMAT=text` 配置（默认 JSON）。
- 后端：新增 `/api/repairMediaHistory` 历史媒体数据修复接口（扫描遗留表 `media_upload_history`：补齐缺失 `file_md5`、按 MD5 全局去重/可选本地路径去重；默认 dry-run，需 `commit=true` 才会写入/删除）。
- 后端：新增 `/api/checkDuplicateMedia` 媒体查重接口：先按 `image_hash.md5_hash` 精确匹配；无命中则按 pHash 相似度阈值查询并返回 similarity/distance。
- 后端：补齐 Go 服务文件功能测试用例（FileStorage/MediaUpload/ImageHash/静态文件），覆盖 `/api/uploadMedia`、`/api/checkDuplicateMedia`、`/api/getAllUploadImages`、`/api/getUserUploadHistory`、`/api/getUserSentImages`、`/api/getUserUploadStats`、`/api/getChatImages`、`/api/recordImageSend`、`/api/reuploadHistoryImage`、`/api/deleteMedia`、`/api/batchDeleteMedia`、`/api/getCachedImages`、`/api/repairMediaHistory`。
- 后端：补齐 Go 服务认证/WebSocket 管理器与 WebSocket 代理（`/ws`）测试，覆盖登录/验签、握手鉴权、sign 代理、连接池/延迟关闭/淘汰、forceout、缓存写入与僵尸下游清理等关键路径。
- 后端：补齐 Go 服务 `favorite`（本地收藏CRUD）与 `user_history`（历史/收藏列表代理）测试用例，覆盖成功/失败与缓存增强分支。
- 前端：补充 `useWebSocket` 断线自动重连、forceout 禁止重连与手动断开不重连的测试用例。
- 前端：聊天发送支持乐观 UI（sending/failed 可重试），并在收到 WebSocket 回显时合并更新避免重复渲染。
- 前端：聊天消息列表新增骨架屏占位（历史加载/侧边栏/收藏列表）并引入虚拟滚动（vue-virtual-scroller）；媒体渲染抽取 `ChatMedia`（加载占位/懒加载/错误兜底）以减少布局抖动。
- CI：新增 `Release` GitHub Actions 工作流，用于创建 `v*` Tag 并生成 GitHub Release 产物。
- 知识库：补齐 Wiki 概览/架构文档，并补充关键模块文档（Auth/Identity/WebSocket Proxy/Media）。
- 文档：标记历史 Java(Spring Boot) 后端目录（`src/main/java/`）为已弃用，仅供参考。

### 修复
- 修复聊天侧边栏匹配按钮点击后误自动进入聊天的问题（恢复 `startContinuousMatch(1)` 行为）。
- 前端：mtPhoto 相册图片预览支持左右切换浏览，并确保切换后“上传/导入”作用于当前预览图片。
- 前端：mtPhoto 相册预览支持查看详情（信息按钮 + 详情面板）。
- 前端：全站图片库预览切换后同步当前媒体，避免“切换后仍对首张执行上传/重传”的不一致行为。
- 后端：图片端口策略优化：`real` 改为并发竞速（HTTP 200/206 即胜出并取消其余请求），并在全部失败时降级到 `probe`（并发 TCP 通断）或最终回退 `fixed`，避免串行探测导致卡顿。
- 后端：`/api/getFavoriteUserList` 对齐 `/api/getHistoryUserList` 的缓存增强逻辑，补全用户信息并补齐 `lastMsg/lastTime`。
- 后端：修复 `lastMsg` 格式化将表情文本（如 `[doge]`）误识别为 `[文件]`，导致会话列表预览显示不正确的问题。
- 前端：修复聊天列表（侧边栏）最后一条消息时间未使用统一格式化（`formatTime`）的问题。
- 前端：修复 ChatRoomView 预览事件监听清理的生命周期注册，避免 Vue 警告。
- 前端：修复在列表页收到新消息时，未读气泡可能不显示的问题（路由判定 + 会话状态清理双保险）。
- 前端：修复 WebSocket 私信回显时 `isSelf` 误判，改为对齐上游 `randomdeskry.js` 的 `md5(user_id)` 判定逻辑，避免自己消息显示在左侧。
- 前端：修复切换身份后 WebSocket 仍绑定旧用户，导致匹配无响应且仍收到旧身份消息。
- 前端：修复图片端口策略为 real/probe 时，视频 URL 仍使用固定端口（8006）的问题。
- 前端：修复聊天记录媒体消息偶发重复显示（WS 推送与历史拉取合并时按 remotePath + isSelf + 5s 时间窗口语义去重）。
- 前端：修复在聊天页自己发送消息回显时，会话未置顶到“消息/收藏”列表且 lastMsg 预览未加 `我: ` 前缀的问题。
- 修复 `/api/getHistoryUserList` 在上游用户ID字段为 `UserID/userid` 时，未能填充 `lastMsg` / `lastTime` 的问题。
- 修复上游返回的消息 `id/toid` 与 `myUserID` 不一致时，最后消息缓存会话Key无法命中导致 `lastMsg` / `lastTime` 缺失的问题。
- 后端：补齐 Go 版 User History/鉴权关键日志，避免容器运行时仅有启动日志。
- 后端：修复 Go 端错误读取 `Host` 请求头导致媒体 URL 回退 `localhost`，从而出现图片无法加载的问题（影响 `getAllUploadImages/getCachedImages` 等）。
- 后端：修复 Go 端删除媒体对历史 `localPath`（无前导 `/` 或携带 `/upload` 前缀/完整 URL）兼容不足，导致返回 403 的问题。
- 后端：修复 Go 端 `/api/deleteMedia` 兼容性：支持 `localPath` 含 `%2F` 编码、兼容 `local_path` 异常带 `/upload` 前缀/缺少前导 `/`，并取消按 `userId` 校验上传者归属以对齐“全站图片库”展示行为，避免误报 403。
- 后端：修复 Go 版 WebSocket 下游广播在存在僵尸连接时可能写阻塞，导致匹配/消息回包延迟或丢失的问题（增加下游写超时并在发送失败时清理会话）。
- 后端：修复 `/ws` 在 sign 绑定后仍可转发其他 `id` 消息的问题；现在仅转发与已绑定 `userId` 一致的消息，并在重复 sign（切换身份）时自动解绑旧身份。
- 后端：修复 `/api/checkDuplicateMedia` 的 pHash 计算与阈值语义，对齐 Python `imagehash.phash`（median/DCT 顺序/重采样）并支持 `distanceThreshold` 默认 10。
- 后端：`JWTService` 在密钥缺失时拒绝签发/校验 Token（与配置校验保持一致，避免误配导致隐患）。
- 后端：修复 SPA 路由在 `/list`、`/chat` 等页面刷新/直达时偶发 404（Go 静态托管回退从固定白名单改为通用判定）。
