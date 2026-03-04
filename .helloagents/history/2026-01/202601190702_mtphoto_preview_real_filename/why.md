# 变更提案: mtPhoto 预览详情展示真实文件名

## 需求背景
当前 mtPhoto 相册预览的“查看详情”面板依赖 `media.originalFilename` 展示原始文件名，但 mtPhoto 预览数据未填充该字段，导致详情面板无法显示真实文件名，影响素材识别与下载命名体验。

## 变更内容
1. 在 mtPhoto 相册预览中，用户打开“查看详情”时按需解析真实文件名并展示。
2. 仅展示文件名（basename），不展示完整路径，避免泄露目录结构。
3. 对解析结果按 `md5` 做轻量缓存，减少重复请求。

## 影响范围
- **模块:** mtPhoto、Media（预览组件）
- **文件:**
  - `frontend/src/components/media/MediaPreview.vue`
  - `frontend/src/components/media/MtPhotoAlbumModal.vue`
- **API:** 复用既有 `GET /api/resolveMtPhotoFilePath`
- **数据:** 无

## 核心场景

### 需求: mtPhoto 预览详情展示真实文件名
**模块:** mtPhoto
mtPhoto 相册网格中点击图片进入预览后，打开详情面板应能看到真实文件名，便于识别与下载保存。

#### 场景: 在 mtPhoto 相册预览中查看真实文件名
条件：用户在 mtPhoto 相册中点击图片进入预览，并点击顶部“查看详情/信息”按钮打开详情面板。
- 预期结果：详情面板展示“原始文件名”为真实文件名（从 mtPhoto `filesInMD5` 解析得到的 `filePath` 的 basename）。

## 风险评估
- **风险:** 解析接口返回的 `filePath` 含目录信息，存在泄露风险
- **缓解:** 前端只取 basename 并展示；不在 UI 中展示完整路径；仅在用户主动打开详情时请求并按 `md5` 缓存

