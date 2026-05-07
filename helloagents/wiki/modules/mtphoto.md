# mtPhoto

## 目的
接入 mtPhoto 相册系统，提供相册/文件夹浏览、缩略图、原图下载、相同媒体查询和本地导入。

## 模块概述
- **职责:** mtPhoto 登录、相册列表、文件列表、文件夹树、缩略图代理、原图下载、MD5 路径解析、同图查询、文件夹收藏。
- **状态:** 稳定
- **最后更新:** 2026-05-07

## 规范

### 需求: mtPhoto 配置可选
**模块:** mtPhoto  
依赖 `MTPHOTO_BASE_URL`、登录用户名、密码和可选 OTP。未配置时相关接口应明确返回错误，不影响主应用其他功能。

#### 场景: 获取缩略图
- `/api/getMtPhotoThumb` 为 `<img>` 直连场景放行。
- Handler 内部需做参数和 size 白名单校验。

### 需求: 本地文件路径安全
**模块:** mtPhoto  
`LSP_ROOT` 映射 `/lsp/*` 本地目录，导入和同图查询必须防止路径遍历。

#### 场景: 导入媒体
- `importMtPhotoMedia` 将 mtPhoto 文件复制到本地 `./upload`。
- 按 MD5 全局去重，命中时刷新排序时间。

### 需求: 文件夹收藏
**模块:** mtPhoto  
文件夹收藏落库到 `mtphoto_folder_favorite`，按 `folder_id` 唯一。

## API接口
- `GET /api/getMtPhotoAlbums`
- `GET /api/getMtPhotoAlbumFiles`
- `GET /api/getMtPhotoFolderRoot`
- `GET /api/getMtPhotoFolderContent`
- `GET /api/getMtPhotoFolderBreadcrumbs`
- `GET /api/getMtPhotoFolderFavorites`
- `POST /api/upsertMtPhotoFolderFavorite`
- `POST /api/removeMtPhotoFolderFavorite`
- `GET /api/getMtPhotoThumb`
- `GET /api/downloadMtPhotoOriginal`
- `GET /api/resolveMtPhotoFilePath`
- `GET /api/getMtPhotoSameMedia`
- `POST /api/importMtPhotoMedia`

## 数据模型
- `mtphoto_folder_favorite`
- `media_file`

## 依赖
- `internal/app/mtphoto_client.go`
- `internal/app/mtphoto_handlers.go`
- `internal/app/mtphoto_folder_favorite.go`
- `frontend/src/api/mtphoto.ts`
- `frontend/src/stores/mtphoto.ts`
