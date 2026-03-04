# 技术设计: mtPhoto 相册预览支持左右切换浏览

## 技术方案
### 核心技术
- Vue 3 + TypeScript
- 现有媒体预览组件：`frontend/src/components/media/MediaPreview.vue`
- mtPhoto 相册弹窗：`frontend/src/components/media/MtPhotoAlbumModal.vue`

### 实现要点
- 在 mtPhoto 相册内点击“图片”时，构造当前已加载的相册图片列表 `mediaList` 传给 `MediaPreview`，启用其内置画廊能力（左右按钮/方向键/滑动）。
- 为确保“上传”始终作用于当前正在预览的媒体，在 `MediaPreview` 中新增对外事件（例如 `media-change`），在 `next/prev/jumpTo` 与初始化定位后触发，携带当前媒体信息。
- `MtPhotoAlbumModal` 监听 `media-change`，同步更新当前 `md5/type`（必要时更新 url），使导入逻辑始终以当前媒体为准。
- 视频预览保持现状（仍按需解析 `/lsp/...` 路径播放）。本次变更优先满足“图片预览左右切换”，避免为视频批量解析路径带来额外请求与性能成本。

## 架构决策 ADR
本需求不涉及架构调整与新增依赖，省略。

## API设计
无新增/变更 API。

## 数据模型
无。

## 安全与性能
- **安全:** 不新增后端透传能力，不扩大现有 `/api/getMtPhotoThumb` 白名单；仅在前端同步状态，避免导入错图。
- **性能:** `mediaList` 基于“当前已加载”的相册图片列表构造，不做全量相册拉取；不引入批量 `resolveMtPhotoFilePath` 调用。

## 测试与部署
- **测试:**
  - 前端构建：`cd frontend && npm run build`
  - 手工验证：
    1) 打开 mtPhoto 相册 → 点击图片 → 预览中左右切换正常；
    2) 切换后点击上传 → 导入的是当前显示图片；
    3) 点击视频预览仍可播放（不受影响）。
- **部署:** 无额外部署步骤，随前端构建产物发布。

