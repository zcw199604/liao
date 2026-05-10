# API 手册

## 概述
- **HTTP Base:** `/api`
- **WebSocket:** `/ws?token=<jwt>`
- **Web 前端配置:** `frontend/src/constants/config.ts`
- **后端路由:** `internal/app/router.go`
- **统一响应:** 多数本地接口返回 `{"code":0,"msg":"success","data":...}` 或模块自定义 JSON；代理类接口可能透传上游文本/JSON；下载类接口返回二进制。

## 认证方式
- `POST /api/auth/login` 使用访问码换取 JWT。
- HTTP 请求通过 `Authorization: Bearer <token>` 鉴权。
- `/ws` 握手通过 query 参数 `token` 校验。
- 当前中间件放行：`/api/auth/login`、`/api/auth/verify`、`/api/getMtPhotoThumb`、`/api/douyin/download`、`/api/douyin/cover`。

---

## 接口列表

### Auth
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 访问码登录，签发 JWT |
| GET | `/api/auth/verify` | 校验 Bearer Token 是否有效 |

### Identity
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/getIdentityList` | 查询本地身份列表 |
| POST | `/api/createIdentity` | 创建身份 |
| POST | `/api/quickCreateIdentity` | 随机快速创建身份 |
| POST | `/api/updateIdentity` | 更新身份名称/性别 |
| POST | `/api/updateIdentityId` | 修改身份 ID |
| POST | `/api/deleteIdentity` | 删除身份 |
| POST | `/api/selectIdentity` | 选择身份并更新最近使用时间 |

### Favorite
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/favorite/add` | 添加本地聊天收藏 |
| POST | `/api/favorite/remove` | 按身份和目标用户移除收藏 |
| POST | `/api/favorite/removeById` | 按收藏 ID 移除 |
| GET | `/api/favorite/listAll` | 查询全部本地收藏 |
| GET | `/api/favorite/check` | 检查目标用户是否已收藏 |

### Chat/User History Proxy
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/getHistoryUserList` | 代理上游历史用户列表并做本地增强 |
| POST | `/api/getFavoriteUserList` | 代理上游收藏用户列表并做本地增强 |
| POST | `/api/reportReferrer` | 上报 referrer 到上游 |
| POST | `/api/getMessageHistory` | 获取消息历史，Redis 模式下可合并本地聊天记录缓存 |
| POST | `/api/toggleFavorite` | 代理上游添加聊天收藏 |
| POST | `/api/cancelFavorite` | 代理上游取消聊天收藏 |
| POST | `/api/deleteUpstreamUser` | 删除上游会话用户并清理本地归档 |
| POST | `/api/batchDeleteUpstreamUsers` | 批量删除上游会话用户并清理本地归档 |

### Media
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/getImgServer` | 获取当前图片服务器 |
| POST | `/api/updateImgServer` | 更新本地图片服务器地址 |
| GET | `/api/downloadImgUpload` | 代理下载上游 `/img/Upload/{path}` |
| POST | `/api/uploadMedia` | 上传图片/视频到本地和上游 |
| POST | `/api/uploadImage` | 兼容图片上传入口 |
| POST | `/api/checkDuplicateMedia` | 按 MD5/pHash 查重 |
| GET | `/api/getCachedImages` | 查询用户缓存图片 |
| POST | `/api/recordImageSend` | 记录媒体发送关系 |
| GET | `/api/getUserUploadHistory` | 查询用户上传历史 |
| GET | `/api/getUserSentImages` | 查询用户已发送图片 |
| GET | `/api/getUserUploadStats` | 查询用户上传统计 |
| GET | `/api/getChatImages` | 查询会话相关媒体 |
| POST | `/api/reuploadHistoryImage` | 本地历史媒体重新上传上游 |
| GET | `/api/getAllUploadImages` | 分页查询全站媒体库 |
| POST | `/api/deleteMedia` | 删除单个媒体 |
| POST | `/api/batchDeleteMedia` | 批量删除媒体 |
| POST | `/api/repairMediaHistory` | 修复媒体历史 |
| POST | `/api/repairVideoPosters` | 修复视频海报 |

### Video Extract
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/uploadVideoExtractInput` | 上传抽帧源视频 |
| POST | `/api/cleanupVideoExtractInput` | 清理临时源视频 |
| GET | `/api/probeVideo` | ffprobe 探测视频元数据 |
| POST | `/api/createVideoExtractTask` | 创建抽帧任务 |
| GET | `/api/getVideoExtractTaskList` | 分页查询抽帧任务 |
| GET | `/api/getVideoExtractTaskDetail` | 查询任务详情和帧分页 |
| POST | `/api/cancelVideoExtractTask` | 暂停/取消任务 |
| POST | `/api/continueVideoExtractTask` | 续跑任务 |
| POST | `/api/deleteVideoExtractTask` | 删除任务和产物 |

### Douyin
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/douyin/account` | 解析抖音账号并拉取作品列表 |
| POST | `/api/douyin/detail` | 解析抖音分享/作品并生成下载缓存 key |
| GET/HEAD | `/api/douyin/cover` | 代理返回封面 |
| GET/HEAD | `/api/douyin/download` | 代理返回媒体流，支持预览/下载 |
| GET | `/api/douyin/livePhoto` | 导出实况照片 |
| POST | `/api/douyin/import` | 导入抖音媒体到本地 `./upload/douyin` |
| GET | `/api/douyin/favoriteUser/list` | 查询抖音用户收藏 |
| POST | `/api/douyin/favoriteUser/add` | 添加/更新抖音用户收藏 |
| POST | `/api/douyin/favoriteUser/remove` | 移除抖音用户收藏 |
| GET | `/api/douyin/favoriteUser/aweme/list` | 查询收藏用户作品 |
| POST | `/api/douyin/favoriteUser/aweme/upsert` | 合并收藏用户作品 |
| POST | `/api/douyin/favoriteUser/aweme/pullLatest` | 拉取收藏用户最新作品 |
| GET/POST | `/api/douyin/favoriteUser/tag/*` | 用户收藏标签列表、新增、更新、删除、应用、排序 |
| GET | `/api/douyin/favoriteAweme/list` | 查询抖音作品收藏 |
| POST | `/api/douyin/favoriteAweme/add` | 添加/更新抖音作品收藏 |
| POST | `/api/douyin/favoriteAweme/remove` | 移除抖音作品收藏 |
| GET/POST | `/api/douyin/favoriteAweme/tag/*` | 作品收藏标签列表、新增、更新、删除、应用、排序 |

### mtPhoto
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/getMtPhotoAlbums` | 查询 mtPhoto 相册 |
| GET | `/api/getMtPhotoAlbumFiles` | 查询相册文件 |
| GET | `/api/getMtPhotoFolderRoot` | 查询文件夹根节点 |
| GET | `/api/getMtPhotoFolderContent` | 查询文件夹内容 |
| GET | `/api/getMtPhotoFolderBreadcrumbs` | 查询文件夹面包屑 |
| GET | `/api/getMtPhotoFolderFavorites` | 查询文件夹收藏 |
| POST | `/api/upsertMtPhotoFolderFavorite` | 新增/更新文件夹收藏 |
| POST | `/api/removeMtPhotoFolderFavorite` | 移除文件夹收藏 |
| GET | `/api/getMtPhotoThumb` | 获取缩略图 |
| GET | `/api/downloadMtPhotoOriginal` | 下载原图/原文件 |
| GET | `/api/resolveMtPhotoFilePath` | 通过 MD5 解析 mtPhoto 文件路径 |
| GET | `/api/getMtPhotoSameMedia` | 查询相同 MD5 媒体 |
| POST | `/api/importMtPhotoMedia` | 导入 mtPhoto 媒体到本地 |

### System
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/getSystemConfig` | 查询全局系统配置 |
| POST | `/api/updateSystemConfig` | 更新全局系统配置 |
| POST | `/api/resolveImagePort` | 按策略解析图片端口 |
| GET | `/api/getConnectionStats` | 查询 WS 连接统计 |
| POST | `/api/disconnectAllConnections` | 断开全部 WS 连接 |
| GET | `/api/getForceoutUserCount` | 查询 forceout 禁止用户数 |
| POST | `/api/clearForceoutUsers` | 清空 forceout 禁止列表 |

---

## WebSocket 协议
- 连接地址：`ws(s)://{host}/ws?token=<jwt>`。
- 第一条业务消息应为 `{"act":"sign","id":"<userId>",...}`。
- 同一 `userId` 的多个下游连接共享一条上游连接。
- 上游 `code=-3` 且 `forceout=true` 会触发 5 分钟禁止重连；后端拒绝消息为 `code=-4`。
