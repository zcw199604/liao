# 任务清单: 媒体图库弹窗扩大显示区域

目录: `helloagents/plan/202601191522_media_gallery_expand/`

---

## 1. 前端布局调整（Media / mtPhoto）
- [√] 1.1 在 `frontend/src/components/media/AllUploadImageModal.vue` 中放宽弹窗容器宽高限制，验证 why.md#需求-瀑布流网格区域更大-场景-大屏浏览图片库
- [√] 1.2 在 `frontend/src/components/media/MtPhotoAlbumModal.vue` 中放宽弹窗容器宽高限制，验证 why.md#需求-瀑布流网格区域更大-场景-大屏浏览图片库
- [√] 1.3 在 `frontend/src/components/common/InfiniteMediaGrid.vue` 中调整滚动容器内边距，减少留白并保持滚动加载行为，验证 why.md#需求-瀑布流网格区域更大-场景-大屏浏览图片库

## 2. 质量验证
- [√] 2.1 执行 `npm -C frontend run build`，确保类型检查与构建通过
- [?] 2.2 手工验证：两处弹窗在瀑布流/网格模式下切换正常、滚动加载正常、空态/结束态正常
  > 备注: 需要在浏览器中手工验收（本次仅完成构建验证）
- [√] 2.3 使用本地 `gemini` 对变更做审查（基于 diff），记录需修正点并处理

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9：无敏感信息引入、无危险命令/路径处理变更、无权限逻辑变更）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/media.md`（同步弹窗展示区域调整说明）
- [√] 4.2 更新 `helloagents/wiki/modules/mtphoto.md`（同步相册弹窗展示区域调整说明）
- [√] 4.3 更新 `helloagents/CHANGELOG.md`
