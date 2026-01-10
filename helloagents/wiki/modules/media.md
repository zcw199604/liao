# Media（媒体上传与媒体库）

## 目的
提供媒体上传、历史记录、媒体库查询与删除等能力，并保持对上游接口的代理兼容与本地数据可追溯。

## 模块概述
- **职责:** 上传/重传媒体；记录发送日志；分页查询上传/发送历史；全站媒体库分页；删除与批量删除；历史数据修复（repair）
- **状态:** ✅稳定
- **最后更新:** 2026-01-10

## 规范

### 需求: 上传与本地落盘
**模块:** Media
上传接口需将文件落盘到 `./upload` 并写入本地表（详见 `data.md`），同时对上游上传接口保持兼容。

### 需求: 媒体历史与消息关联
**模块:** Media
通过 `recordImageSend` 记录 “fromUserId → toUserId” 的媒体发送关系，用于：
- 查询两人双向历史图片（`getChatImages`）
- 查询用户发送历史（`getUserSentImages`）

### 需求: 媒体库分页与端口探测
**模块:** Media
“全站图片库”接口返回分页数据，并包含 `port` 字段（用于前端探测可用图片服务端口）。

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

## 依赖
- `internal/app/media_upload.go`
- `internal/app/media_history_handlers.go`
- `internal/app/media_repair.go`
- `internal/app/media_repair_handlers.go`
- `internal/app/file_storage.go`
- `internal/app/image_cache.go`
- `internal/app/image_server.go`
- `internal/app/image_hash.go`
- `internal/app/image_hash_handlers.go`
- `internal/app/port_detect.go`
- `internal/app/schema.go`

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
- [202601072058_fix_delete_media_403](../../history/2026-01/202601072058_fix_delete_media_403/) - 修复删除接口对多种 localPath 形式的兼容性（待复验）
- [202601071248_go_backend_rewrite](../../history/2026-01/202601071248_go_backend_rewrite/) - Go 后端重构并实现媒体上传/记录/媒体库
- [202601101607_image_hash_duplicate_check](../../history/2026-01/202601101607_image_hash_duplicate_check/) - 新增媒体查重接口（MD5 + pHash 相似度）
- [202601102011_go_file_tests](../../history/2026-01/202601102011_go_file_tests/) - Go 文件功能测试补齐（FileStorage/MediaUpload/ImageHash/handlers）
