# 任务清单: MediaPreview 视频交互微调（手势减敏、快进/快退 1 秒、双击全屏、全屏抓帧/抽帧按钮）

目录: `helloagents/plan/202601210515_media_preview_video_gesture_tune/`

---

## 1. 前端：MediaPreview 视频交互
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 降低左右滑动 seek 灵敏度并提升触发阈值，减少误触
- [√] 1.2 在 `frontend/src/components/media/MediaPreview.vue` 将浮层“快进/快退”步长改为 1 秒
- [√] 1.3 在 `frontend/src/components/media/MediaPreview.vue` 支持双击/双击触控触发全屏，并确保自定义浮层/按钮在全屏中可见
- [√] 1.4 在 `frontend/src/components/media/MediaPreview.vue` 全屏右侧增加“抓帧/抽帧”快捷按钮（复用现有逻辑）

## 2. Gemini 方案复核
- [√] 2.1 使用 Gemini CLI 复核点击/双击/滑动手势互斥与全屏容器选择的可行性，记录关键风险点
  > 备注: 建议延迟单击以识别双击；全屏应作用于包含自定义 UI 的父容器；iOS 音量/全屏需降级。

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/media.md` 同步交互行为（快进步长/双击全屏/全屏按钮）
- [√] 4.2 更新 `helloagents/CHANGELOG.md` 记录 MediaPreview 视频交互微调

## 5. 测试与验证
- [√] 5.1 前端：执行 `cd frontend && npm run build` 验证编译通过
- [?] 5.2 手动验证：滑动不易误触；浮层±1秒；双击全屏；全屏右侧抓帧/抽帧可用（含移动端）
  > 备注: 需要在浏览器真机/桌面端手动验证（本环境已完成构建校验）。
