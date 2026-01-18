# 任务清单: mtPhoto 相册预览支持左右切换浏览

目录: `helloagents/plan/202601181549_mtphoto_preview_gallery/`

---

## 1. 前端 - 媒体预览组件
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 中新增“当前媒体变更”事件（例如 `media-change`），在 next/prev/jumpTo 与初始化定位后触发，向父组件传递当前媒体（至少包含 url/type/md5）。

## 2. 前端 - mtPhoto 相册预览接入画廊模式
- [√] 2.1 在 `frontend/src/components/media/MtPhotoAlbumModal.vue` 中为“点击图片预览”构造并传入 `mediaList`（基于当前已加载的相册图片列表），启用左右切换浏览，验证 why.md#需求-mtphoto-图片预览支持左右切换#场景-点击图片进入预览并左右切换
- [√] 2.2 在 `frontend/src/components/media/MtPhotoAlbumModal.vue` 中监听预览组件的媒体变更事件，同步更新当前 md5/type，确保“上传”导入的是当前预览图片，验证 why.md#需求-mtphoto-图片预览支持左右切换#场景-预览切换后上传当前图片

## 3. 回归与一致性（可选优化）
- [√] 3.1 在 `frontend/src/components/media/AllUploadImageModal.vue` 中同步预览切换后的当前媒体（避免“切换后上传仍上传首张”的潜在不一致），与现有行为对齐。

## 4. 安全检查
- [√] 4.1 执行安全检查（按G9: 不扩大代理白名单、避免引入敏感信息、避免不安全的 URL 拼接）。

## 5. 文档更新
- [√] 5.1 更新 `helloagents/wiki/modules/mtphoto.md` 与 `helloagents/wiki/modules/media.md`（如需要），记录 mtPhoto 预览左右切换能力与交互说明。
- [√] 5.2 更新 `helloagents/CHANGELOG.md` 记录本次变更。

## 6. 测试
- [√] 6.1 执行 `cd frontend && npm run build`
- [?] 6.2 手工验证：mtPhoto 相册图片预览左右切换与“切换后上传当前图片”
  > 备注: 需要在浏览器中人工验证交互（左右按钮/方向键/滑动）与切换后导入目标是否正确
