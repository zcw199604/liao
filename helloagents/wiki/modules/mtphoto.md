# mtPhoto（相册接入与导入上传）

## 目的
将外部 mtPhoto 相册系统接入当前应用，使用户可以按“相册 → 媒体（图片/视频）”方式浏览素材，并一键导入到本地媒体库后上传到上游，最终出现在聊天页“已上传的文件”中用于发送。

## 模块概述
- **职责:** mtPhoto 登录/续期（优先 refresh_token，失败回退登录）；相册列表；相册媒体分页（后端切片）；gateway 缩略图代理；按 MD5 解析本地文件路径；导入上传（落盘到 `./upload` + 上传到上游 + 写入 `media_file` + 写入缓存）
- **状态:** ✅稳定
- **最后更新:** 2026-01-19

## 入口与交互
- **聊天页上传菜单:** 新增“mtPhoto 相册”入口，打开相册弹窗
- **系统设置（System）:** 新增“mtPhoto 相册”入口，打开相册弹窗
- **相册弹窗:** 相册列表（首项“收藏夹”，封面预览为空）→ 相册媒体（网格/瀑布流可切换，无限滚动）→ 预览（图片支持左右切换浏览；预览顶部可查看详情并展示真实文件名等元信息；下载按钮下载原图）→ 点击“上传”触发导入（以当前预览图片为准）

## 核心流程

### 1) 浏览相册
1. 前端调用 `GET /api/getMtPhotoAlbums`
2. 后端 `MtPhotoService` 自动登录（必要时）并请求 mtPhoto 的 `/api-album`
3. 返回相册数组（包含 `id/name/cover/count` 等）
4. 前端在相册列表顶部注入“收藏夹”入口（`albumId=1`），封面预览暂为空；同时异步请求 `GET /api/getMtPhotoAlbumFiles?albumId=1&page=1&pageSize=1` 读取 `total` 作为数量展示；点击后复用相册媒体分页接口加载收藏夹内容

### 2) 浏览相册媒体（懒加载）
1. 前端滚动触底触发 `GET /api/getMtPhotoAlbumFiles?albumId=...&page=...&pageSize=...`
2. 后端请求 mtPhoto 的 `/api-album/filesV2/{id}?listVer=v2`，将结果扁平化并做分页切片
3. 前端使用 `InfiniteMediaGrid` 展示相册媒体（支持瀑布流/网格切换，localStorage: `media_layout_mode`），并使用 `loading="lazy"` 加载缩略图（`/api/getMtPhotoThumb`）

### 3) 导入上传（与“全站图片库”工作流一致）
1. 前端在预览中点击“上传”，调用 `POST /api/importMtPhotoMedia`（携带 `userid/md5` 与上游所需 headers 字段）
2. 后端通过 mtPhoto 的 `/gateway/filesInMD5` 获取 `filePath`（形如 `/lsp/...`）
3. 后端将文件保存到 `./upload`（生成 `/images/...` 或 `/videos/...`），即便上游失败也可在“全站图片库”重试
4. 后端将该文件上传到上游（与 `/api/uploadMedia` 同协议）
5. 成功后写入 `media_file` 并加入 `imageCache`，前端将其加入“已上传的文件”

### 4) 下载原图（mtPhoto 图片下载）
1. 预览展示继续使用缩略图（`/api/getMtPhotoThumb`），避免首屏加载过慢
2. 用户点击预览顶部“下载”按钮时，前端优先使用 `downloadUrl`（`/api/downloadMtPhotoOriginal?id=<id>&md5=<md5>`）
3. 后端代理 mtPhoto `gateway/fileDownload/{id}/{md5}` 流式透传原图内容并返回 `Content-Disposition: attachment` 以便浏览器保存
4. 前端下载时解析 `Content-Disposition` 的 `filename*`（RFC 5987）与 `filename`：当 `filename` 为 URL 编码（如 `%E4%B8%AD%E6%96%87.jpg`）时会在保存前解码，确保中文文件名不被编码

### 5) 查看详情（真实文件名）
1. 用户在预览中点击顶部“查看详情/信息”按钮打开详情面板
2. 当前媒体缺少 `originalFilename` 时，前端按需调用 `GET /api/resolveMtPhotoFilePath?md5=<md5>` 获取 `filePath`
3. 前端仅取 `filePath` 的 basename 作为“原始文件名”展示，并按 `md5` 缓存解析结果（不在 UI 中展示完整路径）

## 鉴权与续期策略（mtPhoto 上游）

- 登录：`POST {MTPHOTO_BASE_URL}/auth/login` → `access_token/auth_code/refresh_token/expires_in`
- 续期：`POST {MTPHOTO_BASE_URL}/auth/refresh`（`{"token":"<refresh_token>"}`）→ 返回新的 `access_token/auth_code/refresh_token/expires_in`
- 后端策略：提前 60 秒续期；优先 refresh，失败回退 login；401/403 自动续期并重试一次；续期成功后清空相册/媒体缓存

## 安全与约束
- **凭证安全:** mtPhoto 登录凭证仅通过环境变量注入；后端日志禁止输出 token/auth_code/refresh_token/password
- **路径安全:** mtPhoto 返回的 `filePath` 必须以 `/lsp/` 开头，并通过 `LSP_ROOT` 映射到本地目录；禁止 `..` 路径遍历
- **开放代理防护:** `GET /api/getMtPhotoThumb` 仅允许 `size=s260|h220`，且该接口为前端 `<img>` 加载所需而放行（不要求 JWT）

## API接口
详见 `helloagents/wiki/api.md` 的 mtPhoto 小节：
- `GET /api/getMtPhotoAlbums`
- `GET /api/getMtPhotoAlbumFiles`
- `GET /api/getMtPhotoThumb`
- `GET /api/downloadMtPhotoOriginal`
- `GET /api/resolveMtPhotoFilePath`
- `POST /api/importMtPhotoMedia`

## 配置（环境变量）
- `MTPHOTO_BASE_URL`: mtPhoto 服务地址（例如 `http://pc.zcw.work:38064`）
- `MTPHOTO_LOGIN_USERNAME`: 登录请求 username
- `MTPHOTO_LOGIN_PASSWORD`: 登录请求 password
- `MTPHOTO_LOGIN_OTP`: 可选
- `LSP_ROOT`: 本地 `/lsp/*` 映射根目录（默认 `/lsp`）

## 依赖
- 后端：
  - `internal/app/mtphoto_client.go`
  - `internal/app/mtphoto_handlers.go`
  - `internal/app/static.go`（`/lsp/*` 安全映射）
  - `internal/config/config.go`
- 前端：
  - `frontend/src/api/mtphoto.ts`
  - `frontend/src/stores/mtphoto.ts`
  - `frontend/src/components/media/MtPhotoAlbumModal.vue`
  - `frontend/src/components/common/InfiniteMediaGrid.vue`
  - `frontend/src/components/chat/UploadMenu.vue`
  - `frontend/src/views/ChatRoomView.vue`
  - `frontend/src/components/settings/SettingsDrawer.vue`
  - `frontend/src/App.vue`

## 测试
- 后端：`go test ./...`（重点覆盖 mtPhoto 登录/refresh 续期/401 重登、分页切片、导入路径校验）
- 前端：`npm run build`（类型检查 + 构建门禁）

## 变更历史
- [202601181444_mtphoto_album](../../history/2026-01/202601181444_mtphoto_album/) - 接入 mtPhoto 相册并支持按相册浏览与一键导入上传
- [202601181549_mtphoto_preview_gallery](../../history/2026-01/202601181549_mtphoto_preview_gallery/) - 相册图片预览支持左右切换浏览（切换后导入目标对齐）
- [202601190055_mtphoto_refresh_token](../../history/2026-01/202601190055_mtphoto_refresh_token/) - mtPhoto 续期支持 refresh_token（优先 `/auth/refresh`，失败回退 `/auth/login`）
- [202601190109_mtphoto_preview_detail](../../history/2026-01/202601190109_mtphoto_preview_detail/) - mtPhoto 相册预览支持查看详情（信息按钮 + 详情面板）
- [202601190552_mtphoto_favorites_album](../../history/2026-01/202601190552_mtphoto_favorites_album/) - mtPhoto 相册列表置顶新增收藏夹入口（封面预览为空）
- [202601190613_mtphoto_favorites_count](../../history/2026-01/202601190613_mtphoto_favorites_count/) - mtPhoto 相册列表展示收藏夹数量
- [202601190702_mtphoto_preview_real_filename](../../history/2026-01/202601190702_mtphoto_preview_real_filename/) - mtPhoto 相册预览“查看详情”展示真实文件名（按需解析）
- [202601190728_mtphoto_download_filename_cn](../../history/2026-01/202601190728_mtphoto_download_filename_cn/) - 修复 mtPhoto 相册预览下载中文文件名被编码的问题（下载时解码）
