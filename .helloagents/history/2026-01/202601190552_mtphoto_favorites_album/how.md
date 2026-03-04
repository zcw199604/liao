# 技术设计: mtPhoto 相册列表新增收藏夹入口

## 技术方案

### 核心技术
- 前端：Vue 3 + Pinia（现有 mtPhoto 弹窗与 store）
- 后端：复用既有 `/api/getMtPhotoAlbumFiles`（后端已代理 `filesV2/{id}?listVer=v2` 并做分页切片）

### 实现要点
- **在相册列表注入“收藏夹”卡片（置顶）**：
  - 在 `frontend/src/stores/mtphoto.ts` 的 `loadAlbums()` 中，将上游相册列表映射为本地模型，并在数组首位插入“收藏夹”项。
  - “收藏夹”封面字段保持为空，避免请求缩略图。
- **本地相册ID与上游 albumId 解耦**：
  - 为避免与上游相册 ID 冲突（例如上游同样返回 ID=1），本地使用独立的 `id`（用于 `v-for` key 与 UI 选择），并新增字段 `mtPhotoAlbumId`（用于请求媒体列表的上游 albumId）。
  - 普通相册：`id === mtPhotoAlbumId`；收藏夹：`id` 取负数或字符串化唯一值，`mtPhotoAlbumId=1`。
- **媒体加载与数量同步**：
  - 点击“收藏夹”后复用既有 `openAlbum()` → `loadAlbumPage()` 工作流，请求 `mtPhotoAlbumId` 对应的媒体分页。
  - `loadAlbumPage()` 成功后，将返回的 `total` 同步回 `selectedAlbum.count`，保证标题统计准确（同时避免打开相册列表时额外请求）。
- **重复项过滤策略**：
  - 若上游相册列表中存在疑似收藏夹条目（例如 `id===1`），优先过滤，避免列表重复。

## 架构设计
无架构变更，沿用既有：
- `MtPhotoAlbumModal` 负责相册列表展示与相册媒体展示
- `mtphoto store` 负责数据拉取与状态管理
- 后端 `GetAlbumFilesPage` 负责上游 `filesV2` 扁平化与分页切片

## API设计
无新增/变更接口，复用现有接口：
- `GET /api/getMtPhotoAlbums`
- `GET /api/getMtPhotoAlbumFiles?albumId=...&page=...&pageSize=...`（收藏夹固定 `albumId=1`）

## 数据模型
无数据变更。

## 安全与性能
- **安全:** 不引入新的开放代理与鉴权放行；收藏夹仅改变前端展示与请求参数。
- **性能:** 不在打开相册列表时拉取收藏夹媒体；仅在用户点击收藏夹后按现有分页逻辑加载，避免额外开销。

## 测试与部署
- **前端验证:**
  - `npm run build`（类型检查 + 构建）
  - 手动验证：相册列表置顶“收藏夹”；进入后可加载/预览/导入上传
- **后端验证（可选）:** 无后端改动时可跳过；如需回归可执行 `go test ./...`
