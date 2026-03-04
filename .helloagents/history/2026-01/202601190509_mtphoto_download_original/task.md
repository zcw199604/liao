# 任务清单: mtPhoto 图片下载改为原图

目录: `helloagents/plan/202601190509_mtphoto_download_original/`

---

## 1. 后端 - mtPhoto 原图下载代理
- [√] 1.1 在 `internal/app/mtphoto_client.go` 中新增 `GatewayFileDownload`（调用 `/gateway/fileDownload/{id}/{md5}` 并复用登录/续期逻辑），验证 why.md#需求-mtphoto-图片下载应下载原图#场景-在-mtphoto-相册预览中点击下载下载原图
- [√] 1.2 在 `internal/app/mtphoto_handlers.go` 中新增 `handleDownloadMtPhotoOriginal`（参数校验 + 流式透传 + 仅透传必要响应头），验证 why.md#需求-mtphoto-图片下载应下载原图#场景-在-mtphoto-相册预览中点击下载下载原图
- [√] 1.3 在 `internal/app/router.go` 注册 `GET /api/downloadMtPhotoOriginal` 路由，并确认该接口不在 JWT 放行名单内，验证 why.md#需求-mtphoto-图片下载应下载原图#场景-在-mtphoto-相册预览中点击下载下载原图

## 2. 前端 - 预览下载改造
- [√] 2.1 在 `frontend/src/types/media.ts` 扩展 `UploadedMedia`，新增可选字段 `downloadUrl`（必要时可补充 `downloadFilename`），验证 why.md#需求-mtphoto-图片下载应下载原图#场景-在-mtphoto-相册预览中点击下载下载原图
- [√] 2.2 在 `frontend/src/components/media/MtPhotoAlbumModal.vue` 构造图片预览 `mediaList` 时补充 `downloadUrl=/api/downloadMtPhotoOriginal?id=<id>&md5=<md5>`，并确保画廊切换后下载 URL 对齐当前图片，验证 why.md#需求-mtphoto-图片下载应下载原图#场景-画廊切换后下载应对齐当前图片
- [√] 2.3 在 `frontend/src/components/media/MediaPreview.vue` 中改造下载按钮逻辑：优先 `downloadUrl`；当下载地址为 `/api/*` 时使用带 Authorization 的 blob 下载，否则回退原直链下载，验证 why.md#需求-非-mtphoto-下载行为保持不变#场景-非-mtphoto-图片文件仍可正常下载

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9：参数校验、禁止敏感头透传、下载接口保持 JWT 鉴权、防止开放代理滥用）。

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/api.md` 补充 `GET /api/downloadMtPhotoOriginal` 接口说明。
- [√] 4.2 更新 `helloagents/wiki/modules/mtphoto.md` 补充“mtPhoto 原图下载”与前端下载逻辑说明。
- [√] 4.3 更新 `helloagents/CHANGELOG.md` 记录本次变更。

## 5. 测试
- [√] 5.1 后端执行 `go test ./...`
- [√] 5.2 前端执行 `cd frontend && npm run build`
- [?] 5.3 手工验证：mtPhoto 相册预览点击“下载”保存为原图（对比文件大小/清晰度），且非 mtPhoto 预览下载不受影响
  > 备注: 需要在浏览器中人工验证（mtPhoto 实际数据环境）

## 6. 方案包归档（强制）
- [√] 6.1 开发实施完成后：更新本 `task.md` 状态并迁移方案包至 `helloagents/history/2026-01/202601190509_mtphoto_download_original/`
- [√] 6.2 更新 `helloagents/history/index.md` 增加本次变更索引
