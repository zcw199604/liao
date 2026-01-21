# 任务清单: 抽帧任务临时视频上传与退出清理

目录: `helloagents/plan/202601210135_video_extract_temp_input_cleanup/`

---

## 1. 后端（视频抽帧）
- [√] 1.1 在 `internal/app/file_storage.go` 增加临时视频落盘方法 `SaveTempVideoExtractInput`，将上传输入保存到 `upload/tmp/video_extract_inputs/` 并返回 localPath
- [√] 1.2 在 `internal/app/video_extract_handlers.go` 修改 `/api/uploadVideoExtractInput` 使用临时目录，并新增 `/api/cleanupVideoExtractInput` 删除临时视频（限制目录范围，防误删）
- [√] 1.3 在 `internal/app/router.go` 注册 `/api/cleanupVideoExtractInput` 路由

## 2. 前端（任务中心）
- [√] 2.1 在 `frontend/src/api/videoExtract.ts` 增加 `cleanupVideoExtractInput` API
- [√] 2.2 在 `frontend/src/components/media/VideoExtractTaskModal.vue` 追踪本次上传的临时 localPath，并在关闭任务中心时调用 cleanup 接口清理

## 3. 安全检查
- [√] 3.1 执行安全检查：清理接口仅允许删除 `tmp/video_extract_inputs` 下文件，防路径越界与误删

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/api.md`、`helloagents/wiki/modules/media.md` 与 `helloagents/CHANGELOG.md` 同步临时目录与退出清理行为

## 5. 测试
- [√] 5.1 在 `internal/app/file_storage_test.go` 补齐临时落盘测试
- [√] 5.2 新增 `internal/app/video_extract_handlers_test.go` 覆盖上传落盘与清理接口行为

