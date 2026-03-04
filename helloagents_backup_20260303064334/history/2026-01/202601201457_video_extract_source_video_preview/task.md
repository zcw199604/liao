# 任务清单: 抽帧任务上传视频支持预览与单帧抓取

目录: `helloagents/plan/202601201457_video_extract_source_video_preview/`

---

## 1. 前端：抽帧任务上传视频预览/抓帧入口
- [√] 1.1 在 `frontend/src/components/media/VideoExtractCreateModal.vue` 增加“预览/抓帧”入口（打开 `MediaPreview` 预览源视频），验证：上传视频后可预览并使用“抓帧”
- [√] 1.2 在 `frontend/src/components/media/VideoExtractTaskModal.vue` 增加“源视频预览/抓帧”入口（upload 类型任务），验证：进入任务详情可打开源视频并抓帧

## 2. 安全检查
- [√] 2.1 执行安全检查（按G9：避免泄露本地路径/Token；跨域失败提示；未选择身份时上传降级）

## 3. 文档更新
- [√] 3.1 更新 `helloagents/CHANGELOG.md` 记录：抽帧任务上传视频支持预览/抓帧入口

## 4. 测试与验证
- [√] 4.1 前端：执行 `cd frontend && npm run build` 验证编译通过
- [?] 4.2 手动验证：抽帧任务中心上传视频→创建弹窗预览→抓帧（下载+上传）；任务详情打开源视频→抓帧
  > 备注: 需要在浏览器中手动验证（本环境仅完成构建校验）。
