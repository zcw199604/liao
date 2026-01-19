# Media（媒体上传与媒体库）

## 目的
提供媒体上传、历史记录、媒体库查询与删除等能力，并保持对上游接口的代理兼容与本地数据可追溯。

## 模块概述
- **职责:** 上传/重传媒体；记录发送日志；分页查询上传/发送历史；全站媒体库分页；删除与批量删除；历史数据修复（repair）
- **状态:** ✅稳定
- **最后更新:** 2026-01-19

## 规范

### 需求: 上传与本地落盘
**模块:** Media
上传接口需将文件落盘到 `./upload` 并写入本地表（详见 `data.md`），同时对上游上传接口保持兼容。

### 需求: 媒体历史与消息关联
**模块:** Media
通过 `recordImageSend` 记录 “fromUserId → toUserId” 的媒体发送关系，用于：
- 查询两人双向历史图片（`getChatImages`）
- 查询用户发送历史（`getUserSentImages`）

### 需求: 媒体端口策略（全局配置）
**模块:** Media
为解决上游媒体服务端口不固定/可达不等于可用的问题，系统提供全局媒体端口策略（DB 配置，所有用户共用）：
- `fixed`：固定媒体端口（默认 `9006`）
- `probe`：可用端口探测（并发 TCP 竞速，仅检测端口是否可连通；用于快速选出“能响应”的端口）
- `real`：真实媒体请求（并发竞速：同时对候选端口请求 `http://{host}:{port}/img/Upload/{path}`，按最小字节阈值校验内容；首个有效响应（HTTP 200/206）即返回并取消其它请求；若全部失败则降级到 `probe` 或最终回退 `fixed`）

前端在解析 WS/历史消息中的 `[path]` 媒体消息时，通过 `/api/resolveImagePort` 解析端口并拼接 URL（图片/视频共用该策略）。

### 需求: 媒体库分页与端口字段
**模块:** Media
“全站图片库”接口返回分页数据，并包含 `port` 字段（用于前端在缺少样本路径时的兜底展示/兼容旧逻辑）。当需要高可靠性时，前端应优先走 `/api/resolveImagePort` 的解析结果。

### 需求: 已上传图片浏览（布局切换/无限滚动）
**模块:** Media
“已上传的图片”（`AllUploadImageModal`）用于从“全站图片库”分页浏览素材并发送，前端要求：
- 支持瀑布流（masonry）与网格（grid）布局切换，并将用户选择持久化到 localStorage（`media_layout_mode`）
- 支持无限滚动加载更多，复用 `InfiniteMediaGrid` 组件统一滚动/加载/空态/结束态逻辑
- 缩略图使用 `LazyImage` 进行懒加载与错误兜底，并提供选中态动效；预览背景采用毛玻璃（`backdrop-blur`）提升沉浸感

### 需求: 删除兼容性（localPath 归一化）
**模块:** Media
删除接口需兼容多种 `localPath` 形式（含/不含前导 `/`、带 `/upload` 前缀、完整 URL、含 `%2F` 编码等），并在错误场景返回与现状兼容的 HTTP 状态与 body。

### 需求: 历史媒体数据修复（repairMediaHistory）
**模块:** Media
提供历史表 `media_upload_history` 的修复与去重能力（默认 dry-run，需显式开启写入/删除）。

### 需求: 媒体查重（image_hash）
**模块:** Media
提供“上传文件后查重”的能力：
- MD5 精确命中即返回重复文件信息
- 无 MD5 命中时，按 pHash 相似度阈值查询相似图片并返回相似度（视频/不可解码文件仅做 MD5 查重）

### 需求: 媒体预览（画廊模式）
**模块:** Media
前端 `MediaPreview` 支持传入 `mediaList` 进行左右切换/滑动切换浏览；当预览内部切换当前项时会触发 `media-change` 事件。父组件如提供“上传/重传/导入”等动作，应监听该事件并同步当前目标，避免“切换后仍对首张执行操作”。
预览顶部支持打开“详细信息”面板：仅当媒体对象携带任一元信息字段（如 `md5/fileSize/pHash/similarity` 等）时显示入口。
当业务场景无法在列表阶段拿到真实文件名（如仅有 `md5`），可向 `MediaPreview` 传入可选的 `resolveOriginalFilename` 回调，在用户打开详情面板前按需解析并补齐 `originalFilename`（仅展示 basename，避免泄露目录结构）。
预览顶部“下载”按钮在下载 `/api/*` 资源时会使用带 Authorization 的 blob 下载，并解析 `Content-Disposition` 的 `filename*`（RFC 5987）与 `filename`；当 `filename` 为 URL 编码时会在保存前解码，避免中文文件名被编码。
“全站图片库/已上传图片”场景下，预览弹层背景使用毛玻璃（`backdrop-blur`），缩略图选中提供轻微缩小 + 弹性动画反馈以增强可感知性。

### 需求: 测试覆盖（Go）
**模块:** Media
为文件功能补齐可重复执行的测试用例，覆盖：
- `FileStorageService`（落盘/读取/删除/MD5/复用查询）
- `ImageCacheService`（写入/读取/过期/重建/清空）
- `ImageHashService`（pHash 计算、阈值换算）
- `MediaUploadService`（localPath 归一化、本地 URL 转换、删除行为）
- handler：`/api/uploadMedia`、`/api/checkDuplicateMedia`

## API接口
### [POST] /api/uploadMedia
**描述:** 上传媒体（代理上游 + 本地落盘/记录）

### [POST] /api/checkDuplicateMedia
**描述:** 上传文件后按 `image_hash` 进行查重/相似检索（返回 similarity）

### [GET] /api/getAllUploadImages
**描述:** 全站媒体库分页（返回 `data/total/page/pageSize/totalPages/port`）

### [POST] /api/deleteMedia
**描述:** 删除单个媒体（DB 记录 + 可能的物理文件删除）

### [POST] /api/batchDeleteMedia
**描述:** 批量删除媒体（最多 50 个）

### [POST] /api/repairMediaHistory
**描述:** 修复/去重历史媒体数据（默认 dry-run）

## 数据模型
详见 `helloagents/wiki/data.md`：
- `media_file`
- `media_send_log`
- `media_upload_history`（历史遗留）
- `image_hash`（图片哈希索引）
- `system_config`（系统全局配置）

## 依赖
- 后端：
  - `internal/app/media_upload.go`
  - `internal/app/media_history_handlers.go`
  - `internal/app/media_repair.go`
  - `internal/app/media_repair_handlers.go`
  - `internal/app/file_storage.go`
  - `internal/app/image_cache.go`
  - `internal/app/image_server.go`
  - `internal/app/image_hash.go`
  - `internal/app/image_hash_handlers.go`
  - `internal/app/system_config.go`
  - `internal/app/system_config_handlers.go`
  - `internal/app/image_port_strategy.go`
  - `internal/app/image_port_resolver.go`
  - `internal/app/port_detect.go`
  - `internal/app/schema.go`
- 前端：
  - `frontend/src/components/media/AllUploadImageModal.vue`
  - `frontend/src/components/media/MediaPreview.vue`
  - `frontend/src/components/common/InfiniteMediaGrid.vue`
  - `frontend/src/components/common/LazyImage.vue`

## 测试
- 运行：`go test ./...`
- 相关测试文件：
  - `internal/app/file_storage_test.go`
  - `internal/app/image_cache_test.go`
  - `internal/app/image_hash_test.go`
  - `internal/app/media_history_handlers_test.go`
  - `internal/app/media_repair_handlers_test.go`
  - `internal/app/media_upload_test.go`
  - `internal/app/media_handlers_test.go`
  - `internal/app/static_file_server_test.go`
  - `internal/app/test_helpers_test.go`
  - `internal/app/user_history_media_handlers_test.go`

## 变更历史
- [202601181549_mtphoto_preview_gallery](../../history/2026-01/202601181549_mtphoto_preview_gallery/) - 媒体预览画廊切换时对外同步当前媒体（用于 mtPhoto/全站图片库等场景）
- [202601190109_mtphoto_preview_detail](../../history/2026-01/202601190109_mtphoto_preview_detail/) - 媒体预览支持按元信息显示“查看详情”入口并展示详情面板
- [202601190702_mtphoto_preview_real_filename](../../history/2026-01/202601190702_mtphoto_preview_real_filename/) - mtPhoto 预览“查看详情”按需解析并展示真实文件名
- [202601190728_mtphoto_download_filename_cn](../../history/2026-01/202601190728_mtphoto_download_filename_cn/) - 修复预览下载中文文件名被编码的问题（下载时解码）
- [202601072058_fix_delete_media_403](../../history/2026-01/202601072058_fix_delete_media_403/) - 修复删除接口对多种 localPath 形式的兼容性（待复验）
- [202601071248_go_backend_rewrite](../../history/2026-01/202601071248_go_backend_rewrite/) - Go 后端重构并实现媒体上传/记录/媒体库
- [202601101607_image_hash_duplicate_check](../../history/2026-01/202601101607_image_hash_duplicate_check/) - 新增媒体查重接口（MD5 + pHash 相似度）
- [202601102011_go_file_tests](../../history/2026-01/202601102011_go_file_tests/) - Go 文件功能测试补齐（FileStorage/MediaUpload/ImageHash/handlers）
- [202601102319_image_port_strategy](../../history/2026-01/202601102319_image_port_strategy/) - 新增全局图片端口策略配置（fixed/probe/real）与前端 Settings 切换
- [202601110511_image_port_race](../../history/2026-01/202601110511_image_port_race/) - 图片端口解析优化：real 并发竞速与全失败降级兜底（回退 probe/fixed）
