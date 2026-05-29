# Media

## 目的
管理本地媒体上传、上游上传、媒体库、查重、发送日志和修复任务。

## 模块概述
- **职责:** 图片/视频落盘、上游上传、媒体记录、全站媒体库、聊天媒体查询、MD5/pHash 查重、媒体尺寸回填、视频海报生成与修复。
- **状态:** 稳定
- **最后更新:** 2026-05-27

## 规范

### 需求: 上传媒体
**模块:** Media
`/api/uploadMedia` 接收 multipart 文件，保存到 `./upload/{images|videos}/yyyy/MM/dd/`，上传上游图片服务器，并写入本地数据库。

#### 场景: 上传成功
- 返回上游响应增强字段，如本地文件名、端口、视频海报路径。
- 写入 `media_file` 和 `ImageCacheService`。
- 图片保存到本地后会解析宽高并写入 `media_width`、`media_height`，供全站媒体库瀑布流使用。

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

### 需求: 全站媒体库瀑布流
**模块:** Media
`GET /api/getAllUploadImages` 返回 `media_file` 与 `douyin_media_file` 中的媒体记录，并透出 `width`、`height` 供前端 masonry 布局计算。

#### 场景: 移动端展示混合比例图片
- 前端 `InfiniteMediaGrid` 优先使用后端返回的 `width`、`height` 分列。
- `AllUploadImageModal` 在 masonry 模式下向 `MediaTile` 传入 `aspectRatio`，减少图片加载后的高度跳变。
- 少量缺尺寸记录按保守默认比例和移动端分散策略展示，不阻断分页。

### 需求: 历史媒体尺寸回填
**模块:** Media
`POST /api/repairMediaDimensions` 按批次处理 `media_file` 或 `douyin_media_file` 中缺少尺寸的历史记录。

#### 场景: 文件存在且可解析
- `commit=false` 只统计待更新数量。
- `commit=true` 写入 `media_width`、`media_height`。
- 返回 `nextAfterId` 和 `hasMore`，调用方可按游标继续处理。

#### 场景: 历史文件不存在
- 记录计入 `fileMissing`，不写入宽高。
- warnings 中最多返回 200 条样例。
- 批处理继续执行后续记录，不因单条缺失中断。

#### 场景: 路径非法或图片解析失败
- 路径为空/越界计入 `invalidPath`。
- 图片头不可解析计入 `decodeFailed`。
- 非图片记录计入 `unsupported`。

### 需求: 媒体预览视频操作
**模块:** Media
`frontend/src/components/media/MediaPreview.vue` 是图片、视频和文件的统一预览层。视频预览由 Plyr 增强原生 `video`，并提供播放/暂停、前后 1 秒、横滑 seek、竖滑音量、倍速/慢放、长按临时 2x，以及通过“视频工具”菜单触发的保存当前帧和创建抽帧任务能力。

#### 场景: 保存当前帧
- “保存当前帧”位于视频工具菜单中，用于即时截取当前播放位置的一张图片。
- 保存前先暂停当前视频并使用 canvas 绘制当前帧。
- 保存开始会清理待处理单击/双击 tap，暂停后主动同步播放状态，避免保存过程中再次触发播放/暂停翻转。
- 全屏保存 loading 期间保持浮层工具栏可见，保存结束后恢复自动隐藏。
- 保存结果会下载为 PNG；存在当前身份时会继续上传到图片库。
- 跨域视频无法 canvas 保存时，应提示用户改用本地库或创建抽帧任务。

#### 场景: 视频工具菜单
- 预览底部主操作区不再并列展示“保存当前帧 / 创建抽帧任务 / 上传”等多个处理按钮。
- `showVideoTools`、`showCaptureFrame`、`showExtractTask` 控制视频工具入口和菜单项显示，默认保持兼容。
- 从抽帧创建/任务中心预览源视频时会禁用 `showExtractTask`，避免在抽帧流程内递归打开新的抽帧创建流程。

#### 场景: 播放器状态同步
- 倍速、当前时间和音量写入同时同步 Plyr 与原生 `video`，避免保存当前帧、横滑 seek、慢放和长按 2x 看到的状态不一致。
- 松开横滑/竖滑手势时会补一次最终帧应用，避免 pointerup 早于动画帧导致最后一次 seek/音量调整丢失。
- Plyr 初始化失败并回退原生 controls 时，外层手势不抢占原生控制栏操作。

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
- `POST /api/repairMediaDimensions`

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
- `internal/app/media_dimensions_repair.go`
- `internal/app/video_poster.go`
- `internal/app/file_storage.go`
- `frontend/src/api/media.ts`
- `frontend/src/stores/media.ts`
