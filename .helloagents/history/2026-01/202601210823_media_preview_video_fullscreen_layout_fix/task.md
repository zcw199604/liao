# 任务清单: 修复 MediaPreview 视频真全屏布局偏移（居中/黑边）

目录: `helloagents/plan/202601210823_media_preview_video_fullscreen_layout_fix/`

---

## 1. 前端：MediaPreview 真全屏布局修复
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 针对 `.media-preview-video-wrapper` 增加 `:fullscreen/:-webkit-full-screen` 样式：全屏占满视口、flex 居中对齐
- [√] 1.2 在 `frontend/src/components/media/MediaPreview.vue` 全屏时覆盖 `.plyr` 与 `video` 的 `max-width/max-height(95%)`、圆角与阴影，避免出现“偏移 + 黑边”视觉

## 2. Gemini 方案复核
- [√] 2.1 使用 Gemini CLI 复核全屏偏移/黑边问题的最可能原因（`inline-flex` + 未显式居中）与 CSS 修复建议（`:fullscreen` 下强制 flex 居中 + 宽高/约束覆盖）

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 不涉及生产环境/PII/权限/支付；仅前端样式调整）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/media.md` 补充 MediaPreview 真全屏布局说明/注意事项
- [√] 4.2 更新 `helloagents/CHANGELOG.md` 记录本次修复
- [√] 4.3 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`

## 5. 测试与验证
- [√] 5.1 前端：执行 `cd frontend && npm run build` 验证编译通过
- [?] 5.2 手动验证：进入/退出真全屏后视频始终居中；不再出现明显偏移黑边；Plyr 控制栏/自定义浮层按钮可正常交互（桌面端 + 移动端/iOS 如适用）
