# 任务清单: 视频上传缩略图

目录: `helloagents/plan/202601210431_video_thumbnail/`

---

## 1. 后端（Media）
- [√] 1.1 在 `internal/app/schema.go` 中为 `media_file` 增加 `thumb_local_path` 字段并提供增量迁移，验证 why.md#需求-上传视频生成缩略图-场景-上传-mp4
- [√] 1.2 在 `internal/app/media_upload.go` 中扩展 `MediaFileDTO` 返回 `thumbUrl` 并在查询中读取 `thumb_local_path`，验证 why.md#需求-列表展示缩略图并可播放-场景-浏览全站媒体库
- [√] 1.3 在 `internal/app/media_upload.go` 中实现视频缩略图生成（ffmpeg 抽帧 + 超时控制），验证 why.md#需求-上传视频生成缩略图-场景-上传-mp4
- [√] 1.4 在 `internal/app/user_history_handlers.go` 的 `/api/uploadMedia` 流程中对视频调用缩略图生成并写入上传记录，验证 why.md#需求-上传视频生成缩略图-场景-上传-mp4

## 2. 前端（Media）
- [√] 2.1 在 `frontend/src/types/media.ts` 为 `UploadedMedia` 增加 `thumbUrl` 字段并在 `frontend/src/stores/media.ts` 映射后端字段，验证 why.md#需求-列表展示缩略图并可播放-场景-浏览全站媒体库
- [√] 2.2 在 `frontend/src/components/media/AllUploadImageModal.vue` 中对视频优先展示 `thumbUrl`，点击后在 `MediaPreview` 播放 `url`，验证 why.md#需求-列表展示缩略图并可播放-场景-浏览全站媒体库

## 3. 安全检查
- [√] 3.1 执行安全检查（输入路径归一化、禁止 shell 拼接、`ffmpeg` 超时控制、失败降级不影响主流程）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/media.md`、`helloagents/wiki/api.md`、`helloagents/wiki/data.md` 说明 `thumbUrl/thumb_local_path`
- [√] 4.2 更新 `helloagents/CHANGELOG.md`

## 5. 测试
- [√] 5.1 运行 `go test ./...` 并修复与 `media_file` 字段变更相关的测试用例
- [√] 5.2 运行 `cd frontend && npm run build` 验证前端编译通过
