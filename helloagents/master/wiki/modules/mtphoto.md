# mtPhoto

## 目的
接入 mtPhoto 相册系统，提供相册/文件夹浏览、缩略图、原图下载、相同媒体查询和本地导入。

## 模块概述
- **职责:** mtPhoto API Key 接入、相册列表、文件列表、文件夹树、缩略图代理、原图下载、MD5 路径解析、同图查询、文件夹收藏。
- **状态:** 稳定
- **最后更新:** 2026-09-18

## 规范

### 需求: mtPhoto 配置可选
**模块:** mtPhoto
依赖 `MTPHOTO_BASE_URL` 和 `MTPHOTO_API_KEY`。未配置时相关接口应明确返回错误，不影响主应用其他功能。`MTPHOTO_LOGIN_USERNAME`、`MTPHOTO_LOGIN_PASSWORD`、`MTPHOTO_LOGIN_OTP` 仅保留历史配置对照，不再作为主链路认证方式。

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

### 文件夹相册创建

- Web 文件夹浏览支持勾选当前目录或多个子目录，可跨目录保留选择、按 ID 去重、全选搜索结果，并将所选目录合并为一个命名相册。
- `POST /api/createMtPhotoFolderAlbum` 接收 `{"name":"相册名","folders":[{"id":1063,"path":"/upload/root/test"}]}`，沿用本地 API 登录鉴权和服务端 `MTPHOTO_API_KEY`。
- 上游流程为 `POST /api-album` → 逐个 `POST /api-album/link/{id}` → `POST /api-album/linkSyncFiles/{id}`。关联参数是 `type:"folder"`、`value: JSON.stringify({id,label:完整路径})`、`exclude:false`；保留自动关联，而非一次性拷贝文件列表。
- 2026-09-18 核对官网 OpenAPI 及官方网页实现：OpenAPI 的 `exclude` 字段描述与示例有矛盾，官方网页自动加入使用 `false`，排除使用 `true`。
- 成功返回 `{"success":true,"album":{...}}`。已创建但关联或同步失败时返回 HTTP 502、`success:false`、已创建的 `album` 和包含恢复建议的 `error`；保留相册，不自动删除或重复创建。创建后清除相册列表缓存，前端刷新列表；部分成功也清空选择并展示错误。

### 需求: 上游 API 快照可追溯
**模块:** mtPhoto
mtPhoto 上游文档已整理为本地快照，后续接口变更应优先更新快照再同步本模块说明。

#### 场景: 查阅上游接口
- 整理快照: [mtphotos-upstream-api.md](mtphotos-upstream-api.md)
- 原始快照: [mtphotos-openapi.json](mtphotos-openapi.json)、[mtphotos-api-info.json](mtphotos-api-info.json)
- 来源入口: `https://mtmt.tech/api/`
- OpenAPI JSON: `https://demo.mtmt.tech/api-json`
- 服务端信息: `https://demo.mtmt.tech/api-info`

#### 场景: 登录与媒体访问
- 普通上游 JSON API 使用 `x-api-key: <MTPHOTO_API_KEY>`。
- 缩略图、原文件下载等媒体资源访问前，后端调用 `POST /auth/auth_code`，body 为 `{"api_key":"..."}`。
- 媒体资源请求使用 query 参数 `auth_code`，例如 `/gateway/s260/{md5}?auth_code=...`，不再发送 `jwt` 头或 `Cookie: auth_code=...`。

## API接口
- `GET /api/getMtPhotoAlbums`
- `POST /api/createMtPhotoFolderAlbum`
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

## 本项目与上游映射

| 本地接口 | 上游接口 | 上游认证 | 说明 |
|----------|----------|----------|------|
| `GET /api/getMtPhotoAlbums` | `GET /api-album` | `x-api-key` | 相册列表 |
| `GET /api/getMtPhotoAlbumFiles` | `GET /api-album/filesV2/{id}?listVer=v2` | `x-api-key` | 相册时间线文件 |
| `GET /api/getMtPhotoFolderRoot` | `GET /gateway/folders/root` | `x-api-key` | 根目录 |
| `GET /api/getMtPhotoFolderContent` | `GET /gateway/foldersV2/{id}` + `GET /gateway/folderFiles/{id}` | `x-api-key` | 目录内容与时间线合并 |
| `GET /api/getMtPhotoFolderBreadcrumbs` | `GET /gateway/folderBreadcrumbs/{id}` | `x-api-key` | 面包屑 |
| `GET /api/getMtPhotoThumb` | `GET /gateway/{size}/{md5}?auth_code=...` | `auth_code` | 仅放行 `s260`、`h220` |
| `GET /api/downloadMtPhotoOriginal` | `GET /gateway/fileDownload/{id}/{md5}?auth_code=...` | `auth_code` | 原图/原文件下载 |
| `GET /api/resolveMtPhotoFilePath` | `POST /gateway/filesInMD5` | `x-api-key` | 通过 MD5 解析本地文件路径 |
| `GET /api/getMtPhotoSameMedia` | `POST /gateway/filesInMD5` + `GET /gateway/fileInfo/{id}/{md5}` | `x-api-key` | 相同媒体查询 |
| `POST /api/importMtPhotoMedia` | 先解析 `/lsp/*` 再导入本地 `./upload` | 本地文件系统 | 仅落本地，不回写上游 |

## 数据模型
- `mtphoto_folder_favorite`
- `media_file`

## 依赖
- `internal/app/mtphoto_client.go`
- `internal/app/mtphoto_handlers.go`
- `internal/app/mtphoto_folder_favorite.go`
- `frontend/src/api/mtphoto.ts`
- `frontend/src/stores/mtphoto.ts`
