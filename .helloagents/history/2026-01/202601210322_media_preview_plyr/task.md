# 任务清单: 媒体预览视频播放器升级为 Plyr

目录: `helloagents/plan/202601210322_media_preview_plyr/`

---

## 1. 知识库（功能基线整理）

- [√] 1.1 更新 `helloagents/wiki/modules/media.md`：整理图片/视频预览的已支持功能、快捷键与边界说明，作为回归基线，验证 why.md#需求-视频预览体验升级且不回退

## 2. 前端集成（保持功能一致）

- [√] 2.1 在 `frontend/package.json` 引入 `plyr` 并更新 `frontend/package-lock.json`，验证 why.md#需求-视频预览体验升级且不回退-场景-预览视频（播放/暂停/定位）
- [√] 2.2 在 `frontend/src/components/media/MediaPreview.vue` 集成 Plyr（仅主视频预览），并保证倍速/抓帧/抽帧入口/画廊切换/详情面板/下载逻辑行为不变，验证 why.md#需求-视频预览体验升级且不回退-场景-倍速与抓帧，依赖任务2.1

## 3. 安全检查

- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、EHRB风险规避）

## 4. 文档与历史

- [√] 4.1 更新 `helloagents/CHANGELOG.md`（Unreleased）
- [√] 4.2 迁移方案包至 `helloagents/history/2026-01/` 并更新 `helloagents/history/index.md`

## 5. 测试

- [√] 5.1 执行 `cd frontend && npm run build` 通过（或记录失败原因）
