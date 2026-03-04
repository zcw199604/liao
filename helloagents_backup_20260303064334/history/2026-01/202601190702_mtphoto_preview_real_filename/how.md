# 技术设计: mtPhoto 预览详情展示真实文件名

## 技术方案

### 核心技术
- Vue 3 + TypeScript（前端组件扩展）
- 复用既有 mtPhoto API：`GET /api/resolveMtPhotoFilePath`

### 实现要点
- 为通用预览组件 `MediaPreview` 增加可选回调 `resolveOriginalFilename`：
  - 触发时机：用户点击顶部“查看详情/信息”按钮（打开详情面板前）
  - 条件：当前媒体缺少 `originalFilename` 且存在 `md5`
  - 行为：调用回调获取文件名并写入当前媒体对象的 `originalFilename`（仅写入文件名，不写入路径）
- 在 `MtPhotoAlbumModal` 中实现 resolver：
  - 调用 `mtphotoApi.resolveMtPhotoFilePath(md5)` 获取 `filePath`
  - 从 `filePath` 提取 basename（兼容 `/` 与 `\\` 分隔符）
  - 以 `md5 -> filename` 做本地缓存，避免重复请求
- 视频预览场景已调用 `resolveMtPhotoFilePath` 获取播放路径，可在同一次解析时补齐 `originalFilename`，减少再次请求

## 安全与性能
- **安全:** 仅展示 basename；详情面板不展示 `filePath`；仅在用户主动打开详情时解析
- **性能:** 按需请求 + `md5` 缓存，避免对相册列表批量解析

## 测试与部署
- **测试:** `cd frontend && npm run build`（类型检查 + 构建）
- **部署:** 无额外步骤（前端构建产物随现有流程发布）

