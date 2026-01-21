# 任务清单: 媒体预览暂停遮罩图标隐藏

目录: `helloagents/plan/202601210243_media_preview_hide_pause_icon/`

---

备注: 实际实现已随 `202601210322_media_preview_plyr` 一并完成（含引入 Plyr 美化控制栏），本方案包迁移入历史用于追溯。

## 1. 前端媒体预览

- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 为主预览 `<video>` 增加专用 class，并在 scoped CSS 中隐藏原生中央遮罩按钮，验证 why.md#需求-暂停查看时无遮挡-场景-视频暂停
- [√] 1.2 在 `frontend/src/components/media/MediaPreview.vue` 回归验证倍速/抓帧/抽帧按钮等交互不受影响，验证 why.md#需求-暂停查看时无遮挡-场景-视频暂停，依赖任务1.1

## 2. 安全检查

- [√] 2.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 3. 文档更新

- [√] 3.1 更新 `helloagents/wiki/modules/media.md` 记录该交互差异与适用范围
- [√] 3.2 更新 `helloagents/CHANGELOG.md`（Unreleased → 修复/变更）

## 4. 测试

- [√] 4.1 执行 `cd frontend && npm run build` 通过（或记录失败原因）
