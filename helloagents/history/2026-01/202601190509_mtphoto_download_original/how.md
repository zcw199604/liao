# 技术设计: mtPhoto 图片下载改为原图

## 技术方案

### 核心技术
- 后端：Go + chi router；`MtPhotoService` 统一处理 mtPhoto 登录/续期与 gateway 访问
- 前端：Vue 3 + TypeScript；预览组件 `MediaPreview`；mtPhoto 弹窗 `MtPhotoAlbumModal`

### 实现要点

#### 1) 后端：新增原图下载代理接口（JWT 保护）
- 新增 `MtPhotoService.GatewayFileDownload(ctx, fileID, md5)`：
  - 访问 `GET {MTPHOTO_BASE_URL}/gateway/fileDownload/{fileID}/{md5}`
  - 复用现有 `doRequest(..., useJWT=true, useCookie=true)`，确保携带 mtPhoto 所需的 `jwt` 与 `auth_code` cookie
- 新增 handler：`GET /api/downloadMtPhotoOriginal?id=&md5=`：
  - 参数校验：`id > 0`；`md5` 非空且为合理格式（建议仅允许 32 位 hex）
  - 调用 `GatewayFileDownload` 获取上游响应
  - 仅透传必要响应头（例如 `Content-Type/Cache-Control/Content-Length/Accept-Ranges/Content-Disposition`）
  - 如上游未返回 `Content-Disposition`，可通过 `ResolveFilePath(md5)` 获取文件名（取 `filepath.Base`），并补充 `Content-Disposition: attachment; filename="..."` 以改善下载文件命名体验
  - 使用 `io.Copy` 流式转发，避免将原图读入内存

#### 2) 前端：预览下载 URL 与展示 URL 解耦
- 扩展 `UploadedMedia`：新增可选字段 `downloadUrl?: string`（必要时可补充 `downloadFilename?: string`）
- 在 `MtPhotoAlbumModal` 构造预览媒体列表时：
  - `url` 继续使用缩略图（`/api/getMtPhotoThumb?...`）作为展示/预览图
  - 对图片项补充 `downloadUrl: /api/downloadMtPhotoOriginal?id=<id>&md5=<md5>`，供下载使用
- 在 `MediaPreview` 的下载按钮逻辑中：
  - 优先使用 `currentMedia.downloadUrl`；否则回退 `currentMedia.url`
  - 对 `/api/*` 的下载地址使用带 Authorization 的请求获取 `blob` 并触发浏览器保存（保证 JWT 生效）
  - 非 `/api/*` 地址保持原有 `<a download>` 直链下载行为，避免引入跨域/权限问题

## 架构决策 ADR

### ADR-001: mtPhoto 原图下载采用“后端代理 + JWT 鉴权 + 前端 Blob 下载”
**上下文:** mtPhoto 原图下载需要后端持有的登录态（`jwt/auth_code`），且前端直接 `<a href>` 无法附带 Authorization 头。  
**决策:** 新增后端下载代理接口并保持 JWT 鉴权；前端对该接口使用 Blob 下载。  
**替代方案:** 放行下载接口（不需要 JWT）并用 `<a href>` 直链下载 → 拒绝原因: 存在开放下载滥用风险。  
**影响:** 前端需要新增下载处理逻辑；后端需确保流式转发与参数校验。

## API设计

### [GET] /api/downloadMtPhotoOriginal
- **请求（query）**
  - `id`: mtPhoto 文件 ID
  - `md5`: mtPhoto 文件 MD5
- **响应**
  - 二进制内容（原图）
  - `Content-Type`: 透传上游或推断
  - `Content-Disposition`: `attachment; filename="..."`（尽量提供）

## 安全与性能
- **安全:**
  - 保持 JWT 鉴权，不加入 `/api/getMtPhotoThumb` 的放行名单
  - `id/md5` 参数严格校验，避免成为开放代理
  - 仅透传必要响应头，避免 `Set-Cookie` 等敏感头下发
- **性能:**
  - 后端流式转发（不缓存整文件）
  - 前端仅在点击下载时拉取原图，预览仍使用缩略图

## 测试与部署
- **后端测试:** `go test ./...`（覆盖 fileDownload 代理与参数校验）
- **前端构建:** `cd frontend && npm run build`
- **手工验证:** mtPhoto 相册 → 预览 → 点击下载，确认保存为原图且文件大小明显大于缩略图
