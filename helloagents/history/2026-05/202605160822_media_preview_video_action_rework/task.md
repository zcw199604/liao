# 媒体预览视频操作重构任务清单

## 任务状态说明

- `[ ]` 待执行
- `[√]` 已完成
- `[X]` 执行失败
- `[-]` 已跳过
- `[?]` 待确认

## 任务清单

### 阶段 1: 组件接口与文案调整

- [√] 在 `frontend/src/components/media/MediaPreview.vue` 增加视频工具显示相关 props。
  - 涉及文件: `frontend/src/components/media/MediaPreview.vue`
  - 验收: 默认行为兼容现有入口，调用方不传 props 时仍支持保存当前帧和创建抽帧任务。

- [√] 将 `MediaPreview` 中所有“抓帧”用户可见文案改为“保存当前帧”。
  - 涉及文件: `frontend/src/components/media/MediaPreview.vue`
  - 验收: 按钮、title、toast 中不再使用“抓帧”描述即时单帧保存。

- [√] 将 `MediaPreview` 中任务型“抽帧”文案改为“创建抽帧任务”。
  - 涉及文件: `frontend/src/components/media/MediaPreview.vue`
  - 验收: 用户可明确知道点击后会进入任务创建流程。

### 阶段 2: 视频工具菜单实现

- [√] 在 `MediaPreview` 非全屏视频预览中新增 `视频工具` 菜单入口。
  - 涉及文件: `frontend/src/components/media/MediaPreview.vue`
  - 验收: 菜单内显示 `保存当前帧` 和按门禁显示的 `创建抽帧任务`。

- [√] 在 `MediaPreview` 全屏浮层中将原右侧独立按钮替换为视频工具菜单或统一工具组。
  - 涉及文件: `frontend/src/components/media/MediaPreview.vue`
  - 验收: 全屏时仍能保存当前帧和创建抽帧任务，且不遮挡倍速与播放控制。

- [√] 调整菜单关闭和状态清理逻辑。
  - 涉及文件: `frontend/src/components/media/MediaPreview.vue`
  - 验收: 切换媒体、关闭预览、点击外部和执行菜单项后菜单关闭。

### 阶段 3: 来源入口场景化

- [√] 调整 `VideoExtractCreateModal` 源视频预览按钮文案为 `预览源视频`。
  - 涉及文件: `frontend/src/components/media/VideoExtractCreateModal.vue`
  - 验收: 头部按钮不再显示 `预览/抓帧`。

- [√] 从抽帧创建弹窗打开源视频预览时禁用再次创建抽帧任务入口。
  - 涉及文件: `frontend/src/components/media/VideoExtractCreateModal.vue`
  - 验收: 源视频预览中不出现递归打开创建任务的入口。

- [√] 评估并调整全站媒体库标题与提示文案，将“图片”语义收敛为“媒体”。
  - 涉及文件: `frontend/src/components/media/AllUploadImageModal.vue`
  - 验收: 已包含视频的界面不再主要称为“所有上传图片”；管理提示仍清晰。

- [√] 校准 mtPhoto 视频预览主按钮文案。
  - 涉及文件: `frontend/src/components/media/MtPhotoAlbumModal.vue`
  - 验收: 视频场景显示导入视频语义，抽帧任务不抢占主按钮。

- [√] 校准抖音下载视频预览主按钮和工具显示。
  - 涉及文件: `frontend/src/components/media/DouyinDownloadModal.vue`
  - 验收: 抖音视频预览优先导入/下载，视频处理动作作为次级工具。

### 阶段 4: 测试与验证

- [√] 更新 `MediaPreview` 相关前端测试。
  - 涉及文件: `frontend/src/__tests__/media-preview-more.test.ts` 或相邻测试文件
  - 验收: 覆盖视频工具菜单、保存当前帧文案、抽帧入口门禁。

- [√] 更新抽帧弹窗相关测试。
  - 涉及文件: `frontend/src/__tests__/settings-media-components.test.ts` 或相邻测试文件
  - 验收: `预览源视频` 文案和 `showExtractTask=false` 行为有测试覆盖。

- [√] 执行前端测试。
  - 命令: `cd frontend && npm test`
  - 验收: 测试通过，或记录失败原因和后续处理。

- [√] 执行前端构建。
  - 命令: `cd frontend && npm run build`
  - 验收: 构建通过。

### 阶段 5: 知识库同步

- [√] 更新 `helloagents/wiki/modules/media.md`。
  - 验收: 媒体预览视频操作规范反映“保存当前帧”和“视频工具菜单”。

- [√] 更新 `helloagents/wiki/modules/video-extract.md`。
  - 验收: 从媒体预览创建抽帧任务的入口语义更新为“创建抽帧任务”。

- [√] 更新 `helloagents/CHANGELOG.md`。
  - 验收: `Unreleased` 记录本次交互重构。

## 回滚策略

如新菜单交互影响视频预览稳定性，可按以下顺序回滚：

1. 保留文案更新，恢复原独立按钮显示。
2. 移除新增 props 调用方配置，使用默认显示规则。
3. 若仍有问题，回退 `MediaPreview.vue` 的工具菜单改动，保留测试中对既有行为的覆盖。

## 执行总结

- 18/18 个任务已完成。
- 前端完整测试通过：`cd frontend && npm test`。
- 前端类型检查与生产构建通过：`cd frontend && npm run build`。
