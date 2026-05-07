# Media

## 目的
管理本地媒体上传、上游上传、媒体库、查重、发送日志和修复任务。

## 模块概述
- **职责:** 图片/视频落盘、上游上传、媒体记录、全站媒体库、聊天媒体查询、MD5/pHash 查重、视频海报生成与修复。
- **状态:** 稳定
- **最后更新:** 2026-05-07

## 规范

### 需求: 上传媒体
**模块:** Media  
`/api/uploadMedia` 接收 multipart 文件，保存到 `./upload/{images|videos}/yyyy/MM/dd/`，上传上游图片服务器，并写入本地数据库。

#### 场景: 上传成功
- 返回上游响应增强字段，如本地文件名、端口、视频海报路径。
- 写入 `media_file` 和 `ImageCacheService`。

#### 场景: 上游失败
- 本地文件保留，返回错误和 `localPath` 供重试。

### 需求: 查重
**模块:** Media  
`/api/checkDuplicateMedia` 先按 MD5 精确匹配；图片可继续计算 pHash 并按汉明距离判断相似。

#### 场景: 非图片文件
- 仍计算 MD5；pHash 不可计算时返回降级结果。

### 需求: 媒体删除
**模块:** Media  
删除应同步处理数据库记录和本地文件，批量删除走 `batchDeleteMedia`。

## API接口
- `POST /api/uploadMedia`
- `POST /api/uploadImage`
- `POST /api/checkDuplicateMedia`
- `GET /api/getCachedImages`
- `GET /api/getAllUploadImages`
- `POST /api/deleteMedia`
- `POST /api/batchDeleteMedia`
- `POST /api/recordImageSend`
- `GET /api/getChatImages`
- `POST /api/reuploadHistoryImage`
- `POST /api/repairMediaHistory`
- `POST /api/repairVideoPosters`

## 数据模型
- `media_file`
- `douyin_media_file`
- `media_send_log`
- `media_upload_history`
- `image_hash`

## 依赖
- `internal/app/media_upload.go`
- `internal/app/media_history_handlers.go`
- `internal/app/media_repair.go`
- `internal/app/video_poster.go`
- `internal/app/file_storage.go`
- `frontend/src/api/media.ts`
- `frontend/src/stores/media.ts`
