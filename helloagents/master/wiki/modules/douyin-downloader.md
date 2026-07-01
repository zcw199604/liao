# Douyin Downloader

## 目的
对接 TikTokDownloader Web API，实现抖音分享解析、账号作品列表、媒体下载代理、本地导入和全局收藏。

## 模块概述
- **职责:** 作品详情、账号解析、封面/下载代理、实况照片导出、导入本地媒体库、用户/作品收藏、标签管理。
- **状态:** 稳定
- **最后更新:** 2026-07-01

## 规范

### 需求: 外部上游可选启用
**模块:** Douyin Downloader  
依赖 `TIKTOKDOWNLOADER_BASE_URL`；未配置时相关接口应返回未启用错误，而不是影响主应用启动。

#### 场景: 解析作品
- 服务端调用外部 Web API，并生成短期缓存 key。
- 前端使用 `/api/douyin/download?key=...&index=...` 预览或下载，不直接信任任意 URL。

### 需求: 下载代理
**模块:** Douyin Downloader  
`download` 和 `cover` 可被 `<img>/<video>` 直接请求，因此中间件放行，安全边界依赖随机 key 和已鉴权入口生成。

#### 场景: 视频预览
- 支持 `Range` 请求透传，保证 `<video>` 可播放和拖动。

### 需求: 收藏全局共享
**模块:** Douyin Downloader  
抖音用户收藏、作品收藏和标签是全局数据，不按本地身份隔离。

#### 场景: 标签应用
- `mode=set/add/remove` 分别表示覆盖、追加和移除。

#### 场景: 已收藏用户再次解析作品
- 前端收藏用户列表和详情中的“再次解析”会切到用户作品模式并重新调用账号解析。
- 解析完成后会调用 `/api/douyin/favoriteUser/aweme/upsert`，把当前解析到的作品、封面、下载地址、置顶状态、发布时间和作者快照合并到 `douyin_favorite_user_aweme`。
- 媒体下载路径可能随上游变化，因此再次解析已收藏用户时必须刷新后端作品快照，而不是只更新前端展示。

## API接口
- `POST /api/douyin/detail`
- `POST /api/douyin/account`
- `GET/HEAD /api/douyin/cover`
- `GET/HEAD /api/douyin/download`
- `GET /api/douyin/livePhoto`
- `POST /api/douyin/import`
- `/api/douyin/favoriteUser/*`
- `/api/douyin/favoriteAweme/*`

## 数据模型
- `douyin_media_file`
- `douyin_favorite_user`
- `douyin_favorite_aweme`
- `douyin_favorite_user_aweme`
- `douyin_favorite_user_tag`
- `douyin_favorite_user_tag_map`
- `douyin_favorite_aweme_tag`
- `douyin_favorite_aweme_tag_map`

## 依赖
- `internal/app/douyin_downloader.go`
- `internal/app/douyin_handlers.go`
- `internal/app/douyin_favorite*.go`
- `internal/app/douyin_livephoto_handlers.go`
- `internal/app/douyin_cookiecloud_provider.go`
- `frontend/src/api/douyin.ts`
- `frontend/src/stores/douyin.ts`
