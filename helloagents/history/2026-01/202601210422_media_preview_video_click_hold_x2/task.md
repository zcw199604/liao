# 任务清单: MediaPreview 视频交互增强（单击播放/暂停、点击浮现三按钮、长按2倍速、滑动快进/音量、按钮美化）

目录: `helloagents/plan/202601210422_media_preview_video_click_hold_x2/`

---

## 1. 前端：MediaPreview 视频交互与样式
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 实现视频区域单击播放/暂停，验证 why.md#需求-单击播放暂停-场景-单击切换
- [√] 1.2 在 `frontend/src/components/media/MediaPreview.vue` 实现倍速控件长按临时 x2（松开恢复），验证 why.md#需求-长按临时2x倍速-场景-长按加速并恢复
- [√] 1.3 在 `frontend/src/components/media/MediaPreview.vue` 实现单击浮现“倒退/播放暂停/快进”三按钮（1秒自动隐藏），验证 why.md#需求-点击浮现三按钮倒退播放暂停快进-场景-单击浮现并自动隐藏
- [√] 1.4 在 `frontend/src/components/media/MediaPreview.vue` 实现视频手势：左右滑动快进/倒退、上下滑动调音量，验证 why.md#需求-手势快进倒退与音量调整-场景-手势控制
- [√] 1.5 在 `frontend/src/components/media/MediaPreview.vue` 美化“抓帧/抽帧”按钮样式与布局，验证 why.md#需求-抓帧抽帧按钮美化-场景-预览底部操作

## 2. 安全检查
- [√] 2.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/media.md` 补充 MediaPreview 视频交互说明
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录 MediaPreview 视频交互增强

## 4. 测试与验证
- [√] 4.1 前端：执行 `cd frontend && npm run build` 验证编译通过
- [?] 4.2 手动验证：单击播放/暂停；长按 x2 加速与释放恢复；抓帧/抽帧按钮视觉与交互；移动端触摸场景
  > 备注: 需要在浏览器真机/桌面端手动验证（本环境已完成构建校验）。
