# 变更提案: 视频上传缩略图

## 需求背景
当前“全站图片库”在展示视频条目时会直接加载 `<video>`，列表滚动时会触发较大的资源请求与解码开销，也难以快速识别视频内容。

## 变更内容
1. 上传视频落盘后，自动生成一张缩略图（JPG）。
2. “全站图片库”列表接口返回视频缩略图 URL，前端优先展示缩略图。
3. 点击视频条目进入预览后仍使用原视频 URL 播放。

## 影响范围
- **模块:**
  - Media（媒体上传与媒体库）
- **文件:**
  - 后端：`internal/app/user_history_handlers.go`、`internal/app/media_upload.go`、`internal/app/schema.go`
  - 前端：`frontend/src/stores/media.ts`、`frontend/src/types/media.ts`、`frontend/src/components/media/AllUploadImageModal.vue`
  - 文档：`helloagents/wiki/modules/media.md`、`helloagents/wiki/api.md`、`helloagents/wiki/data.md`、`helloagents/CHANGELOG.md`
- **API:**
  - `GET /api/getAllUploadImages`（响应 DTO 增加 `thumbUrl`）
- **数据:**
  - `media_file` 新增 `thumb_local_path` 字段（可空）

## 核心场景

### 需求: 上传视频生成缩略图
**模块:** Media
上传视频成功后，服务端在本地生成缩略图文件并记录到媒体库。

#### 场景: 上传 mp4
用户通过 `/api/uploadMedia` 上传 `video/mp4`。
- 预期结果：本地 `./upload/videos/...` 下生成 `*_thumb.jpg`，并在 `media_file.thumb_local_path` 记录路径。

### 需求: 列表展示缩略图并可播放
**模块:** Media
列表接口返回缩略图 URL；前端列表展示缩略图，点击后打开预览并播放视频。

#### 场景: 浏览全站媒体库
用户打开“所有上传图片（全站媒体库）”弹窗浏览视频。
- 预期结果：列表使用缩略图展示，点击进入 `MediaPreview` 后可正常播放视频。

## 风险评估
- **风险:** 依赖运行环境可执行 `ffmpeg`，且生成缩略图会增加上传耗时与 CPU 开销。
- **缓解:** 缩略图生成采用超时控制与 best-effort 策略；失败不影响上传主流程与列表展示（前端回退到现有视频封面逻辑）。
