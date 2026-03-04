# 技术设计: mtPhoto 预览支持查看图片详情

## 技术方案

### 核心技术
- Vue 3 + TypeScript
- 复用组件：`frontend/src/components/media/MediaPreview.vue`、`frontend/src/components/media/MediaDetailPanel.vue`

### 实现要点
- `MediaPreview` 内部基于 `currentMedia` 计算是否展示详情入口（信息按钮）。
- 将 “hasMediaDetails” 判定从仅依赖 `fileSize` 改为：只要存在 `md5/originalFilename/localFilename/fileExtension/fileType/uploadTime/updateTime/pHash/similarity` 等任一字段即认为具备详情能力。
- mtPhoto 相册预览已在 `mediaList` 中提供 `md5`（画廊模式），因此无需新增后端接口即可展示详情入口并打开面板。

## 安全与性能
- **安全:** 不新增任何后端透传/代理能力；详情面板只展示前端已有字段，不引入敏感信息（token、cookie 等）。
- **性能:** 仅调整前端条件渲染，不新增网络请求与额外计算。

## 测试与部署
- **构建验证:** `cd frontend && npm run build`
- **手工验证:** 打开 mtPhoto 相册 → 点击图片进入预览 → 点击信息按钮 → 详情面板可打开并展示 MD5

