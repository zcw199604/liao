# 变更提案: mtPhoto API Key 接入方式迁移

## 需求背景
当前 Liao 的 mtPhoto 接入由 `MtPhotoService` 使用 `MTPHOTO_LOGIN_USERNAME`、`MTPHOTO_LOGIN_PASSWORD` 和可选 OTP 调用 `/auth/login`，后续请求通过非标准 `jwt` 请求头和 `Cookie: auth_code=...` 访问上游。刚整理的 MT Photos API 文档显示，当前推荐方式是通过 `x-api-key` 调用业务 API，并在访问 `/gateway/*` 媒体资源前调用 `/auth/auth_code` 获取 24 小时有效的 `auth_code`。

本次方案目标是把现有 mtPhoto 上游对接实现整体迁移到当前文档方式，同时保持 Liao 对前端暴露的 `/api/getMtPhoto*` 等本地接口尽量不变，降低 UI 与调用方改动面。

⚠️ 不确定因素: 用户要求中的“当前方式”基于上一轮整理出的 MT Photos OpenAPI 文档理解为 `x-api-key` + `/auth/auth_code`。如果实际希望改成 Bearer JWT 或完全透传 OpenAPI 原始接口，需要在执行前调整方案。

## 变更内容
1. 新增 `MTPHOTO_API_KEY` 配置，作为 MT Photos 上游主认证凭据。
2. 将 `MtPhotoService` 的上游认证从账号密码登录/refresh 替换为 API Key 请求头。
3. 将媒体资源访问从 cookie auth_code 改为请求 URL query 参数 `auth_code`，并集中缓存/续期 `/auth/auth_code` 返回值。
4. 保留 Liao 当前 mtPhoto 本地 HTTP API 形态，避免前端和外部调用方大范围迁移。
5. 更新测试、配置文档和知识库，标记旧账号密码配置为兼容/废弃路径。

## 范围边界
- **范围内:** 后端 mtPhoto 上游客户端认证、媒体网关请求、配置加载、相关单元测试、知识库同步。
- **范围外:** 不重写 mtPhoto 前端 UI；不新增 MT Photos API Key 管理界面；不代理全部 423 个上游 OpenAPI 操作；不迁移 `mtphoto_folder_favorite` 数据表；不改变本地 `/api/getMtPhoto*` 路由命名。
- **拆分说明:** 需求影响后端配置、上游客户端、媒体代理、视频抽帧依赖和文档，属于跨模块技术迁移。为控制风险，本方案按“配置与认证核心 → 媒体网关 → 测试与文档”分阶段实施。

## 影响范围
- **模块:** mtPhoto、Video Extract、System Config 文档、配置加载、后端测试。
- **文件:** `internal/config/config.go`、`internal/app/app.go`、`internal/app/mtphoto_client.go`、`internal/app/mtphoto_handlers.go`、`internal/app/video_extract*.go` 相关调用测试、`src/main/resources/application.yml`、`helloagents/wiki/modules/mtphoto.md`、`helloagents/wiki/api.md`。
- **API:** Liao 本地 `/api/getMtPhotoAlbums`、`/api/getMtPhotoAlbumFiles`、`/api/getMtPhotoFolderRoot`、`/api/getMtPhotoFolderContent`、`/api/getMtPhotoThumb`、`/api/downloadMtPhotoOriginal` 等保持路径不变；上游调用认证方式改变。
- **数据:** 无数据库迁移；`mtphoto_folder_favorite` 与本地 `media_file` 写入逻辑保持兼容。

## 核心场景

### 需求: API Key 认证替换账号密码登录
**模块:** mtPhoto
服务端配置 `MTPHOTO_BASE_URL` 和 `MTPHOTO_API_KEY` 后，mtPhoto 相关接口应使用 `x-api-key` 请求头访问 MT Photos 上游。

#### 场景: 查询相册列表
启动服务后调用 `GET /api/getMtPhotoAlbums`。
- 后端请求上游 `GET /api-album`。
- 请求包含 `x-api-key`，不再先调用 `/auth/login`。
- 上游 401/403 时返回明确的鉴权错误。

### 需求: 媒体资源使用 auth_code query 参数
**模块:** mtPhoto
访问缩略图、原图、视频源等 `/gateway/*` 资源前，后端应通过 `POST /auth/auth_code` 使用 API Key 换取 `auth_code`。

#### 场景: 获取缩略图
前端通过 `<img src="/api/getMtPhotoThumb?size=s260&md5=...">` 加载图片。
- 后端先确保存在未过期 `auth_code`。
- 后端请求上游 `GET /gateway/s260/{md5}?auth_code=...`。
- 本地接口继续只放行 `s260`、`h220`，避免开放代理。

#### 场景: 下载原文件
用户点击下载 mtPhoto 原文件。
- 本地 `GET /api/downloadMtPhotoOriginal?id=...&md5=...` 保持不变。
- 后端请求上游 `GET /gateway/fileDownload/{id}/{md5}?auth_code=...`。
- 响应头仍只透传必要下载头，不下发上游敏感头。

### 需求: 现有本地接口兼容
**模块:** mtPhoto
前端 mtPhoto store 和媒体预览流程不应因认证迁移而大范围修改。

#### 场景: 浏览文件夹与导入媒体
用户打开 mtPhoto 文件夹、查看时间线、导入本地媒体。
- 本地返回结构保持 `folderList`、`fileList`、`total`、`page`、`totalPages`。
- 导入仍通过 `POST /gateway/filesInMD5` 解析 `/lsp/*` 路径，再复制到 `./upload`。
- `LSP_ROOT` 路径越界保护保持不变。

## 风险评估
- **风险:** `MTPHOTO_API_KEY` 属于敏感凭据，误写日志或文档会泄露访问权限。
- **缓解:** 配置只读环境变量；错误消息不包含 API Key；测试使用假值；知识库只记录变量名不记录真实值。
- **风险:** 上游部分版本可能仍接受旧 `jwt` 头但对 `x-api-key` 行为存在差异。
- **缓解:** 以本地 OpenAPI 快照为准实现；为 401/403 做一次 auth_code 刷新重试；必要时保留短期兼容开关但默认走 API Key。
- **风险:** auth_code 过期会导致图片直连场景大量 404。
- **缓解:** 缓存设置 23 小时安全 TTL；401/403 时强制刷新并重试一次；并发刷新加锁。
