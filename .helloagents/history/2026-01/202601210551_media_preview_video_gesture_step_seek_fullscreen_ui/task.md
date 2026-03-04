# 任务清单: MediaPreview 视频交互再微调（左右滑动步进、降低误触、全屏倍速布局避让）

目录: `helloagents/plan/202601210551_media_preview_video_gesture_step_seek_fullscreen_ui/`

---

## 1. 前端：MediaPreview 视频手势与控件
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 将左右滑动快进/倒退改为 1 秒步进（每步约 40~80px），避免轻微滑动跳跃过大
- [√] 1.2 在 `frontend/src/components/media/MediaPreview.vue` 优化滑动方向锁定（更保守的方向判定）并提高触发阈值，降低误触概率
- [√] 1.3 在 `frontend/src/components/media/MediaPreview.vue` 全屏模式下将倍速按钮移至左上（隐藏顶部栏倍速入口），避免与右侧抓帧/抽帧按钮重叠
- [√] 1.4 在 `frontend/src/components/media/MediaPreview.vue` 全屏模式隐藏底部抓帧/抽帧按钮，仅保留右侧快捷入口，减少 UI 冗余

## 2. Gemini 方案复核
- [√] 2.1 使用 Gemini CLI 复核“左右滑动步进 + 防误触阈值 + 全屏控件分区/互斥显隐”等 UX 建议
  > 备注: 建议采用 dead zone + 方向锁定；左右滑动按像素累积槽位映射到秒；全屏 UI 分区以避免控件重叠。

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/media.md` 同步左右滑动改为 1 秒步进与全屏倍速布局
- [√] 4.2 更新 `helloagents/CHANGELOG.md` 记录 MediaPreview 视频交互再微调
- [√] 4.3 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`

## 5. 测试与验证
- [√] 5.1 前端：执行 `cd frontend && npm run build` 验证编译通过
- [?] 5.2 手动验证：左右滑动不易误触且为 1 秒步进；全屏右侧抓帧/抽帧不与倍速重叠；点击浮层 1 秒自动隐藏；双击全屏（含移动端）
  > 备注: 需要在浏览器真机/桌面端手动验证（本环境已完成构建校验）。
