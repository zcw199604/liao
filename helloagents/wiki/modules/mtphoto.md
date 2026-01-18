# mtPhoto（相册接入与导入上传）

## 目的
将外部 mtPhoto 相册系统接入当前应用，使用户可以按“相册 → 媒体（图片/视频）”方式浏览素材，并一键导入到本地媒体库后上传到上游，最终出现在聊天页“已上传的文件”中用于发送。

## 模块概述
- **职责:** mtPhoto 登录/续期；相册列表；相册媒体分页（后端切片）；gateway 缩略图代理；按 MD5 解析本地文件路径；导入上传（落盘到 `./upload` + 上传到上游 + 写入 `media_file` + 写入缓存）
- **状态:** ✅稳定
- **最后更新:** 2026-01-18

## 入口与交互
- **聊天页上传菜单:** 新增“mtPhoto 相册”入口，打开相册弹窗
- **系统设置（System）:** 新增“mtPhoto 相册”入口，打开相册弹窗
- **相册弹窗:** 相册列表 → 相册媒体网格（无限滚动）→ 预览 → 点击“上传”触发导入

## 核心流程

### 1) 浏览相册
1. 前端调用 `GET /api/getMtPhotoAlbums`
2. 后端 `MtPhotoService` 自动登录（必要时）并请求 mtPhoto 的 `/api-album`
3. 返回相册数组（包含 `id/name/cover/count` 等）

### 2) 浏览相册媒体（懒加载）
1. 前端滚动触底触发 `GET /api/getMtPhotoAlbumFiles?albumId=...&page=...&pageSize=...`
2. 后端请求 mtPhoto 的 `/api-album/filesV2/{id}?listVer=v2`，将结果扁平化并做分页切片
3. 前端使用 `loading="lazy"` 加载缩略图（`/api/getMtPhotoThumb`）

### 3) 导入上传（与“全站图片库”工作流一致）
1. 前端在预览中点击“上传”，调用 `POST /api/importMtPhotoMedia`（携带 `userid/md5` 与上游所需 headers 字段）
2. 后端通过 mtPhoto 的 `/gateway/filesInMD5` 获取 `filePath`（形如 `/lsp/...`）
3. 后端将文件保存到 `./upload`（生成 `/images/...` 或 `/videos/...`），即便上游失败也可在“全站图片库”重试
4. 后端将该文件上传到上游（与 `/api/uploadMedia` 同协议）
5. 成功后写入 `media_file` 并加入 `imageCache`，前端将其加入“已上传的文件”

## 安全与约束
- **凭证安全:** mtPhoto 登录凭证仅通过环境变量注入；后端日志禁止输出 token/auth_code/password
- **路径安全:** mtPhoto 返回的 `filePath` 必须以 `/lsp/` 开头，并通过 `LSP_ROOT` 映射到本地目录；禁止 `..` 路径遍历
- **开放代理防护:** `GET /api/getMtPhotoThumb` 仅允许 `size=s260|h220`，且该接口为前端 `<img>` 加载所需而放行（不要求 JWT）

## API接口
详见 `helloagents/wiki/api.md` 的 mtPhoto 小节：
- `GET /api/getMtPhotoAlbums`
- `GET /api/getMtPhotoAlbumFiles`
- `GET /api/getMtPhotoThumb`
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
  - `frontend/src/components/chat/UploadMenu.vue`
  - `frontend/src/views/ChatRoomView.vue`
  - `frontend/src/components/settings/SettingsDrawer.vue`
  - `frontend/src/App.vue`

## 测试
- 后端：`go test ./...`（重点覆盖 mtPhoto 登录/401 重登、分页切片、导入路径校验）
- 前端：`npm run build`（类型检查 + 构建门禁）
