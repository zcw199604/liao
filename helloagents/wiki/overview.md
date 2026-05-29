# Liao

> 本文件包含项目级别核心信息。详细模块规范见 `wiki/modules/`，接口与数据结构以 `api.md` 和 `data.md` 为准。

---

## 1. 项目概述

### 目标与背景
Liao 是一个匿名匹配聊天代理应用：浏览器或 Android 客户端连接本地服务，服务端代理上游聊天 HTTP/WebSocket 能力，并补充本地身份、媒体库、缓存、抖音导入、mtPhoto 相册、视频抽帧和系统配置能力。

### 范围
- **范围内:** 访问码登录/JWT、本地身份池、聊天列表与消息、WebSocket 双向代理、媒体上传与媒体库、抖音下载/收藏、mtPhoto 相册接入、视频抽帧、系统配置、Android 客户端。
- **范围外:** 上游匿名聊天业务逻辑修改、第三方支付、生产运维后台、真实凭据托管。

### 项目规模
- **规模判定:** 大型项目。
- **依据:** 2026-05-10 有效扫描约 748 个非生成/非备份文件，其中 Go、TypeScript/Vue、Kotlin、SQL/KTS 代码与脚本文件约 526 个，满足大型项目判定。

---

## 2. 模块索引

| 模块名称 | 职责 | 状态 | 文档 |
|---------|------|------|------|
| Auth | 访问码登录、JWT、HTTP/WS 鉴权 | 稳定 | [auth.md](modules/auth.md) |
| Identity | 本地身份 CRUD 与最近使用排序 | 稳定 | [identity.md](modules/identity.md) |
| WebSocket Proxy | 下游 `/ws` 到上游 WebSocket 的连接池和转发 | 稳定 | [websocket-proxy.md](modules/websocket-proxy.md) |
| User History | 上游历史/收藏/消息代理与本地缓存增强 | 稳定 | [user-history.md](modules/user-history.md) |
| Chat UI | Web 聊天界面、列表、输入、消息展示与未读规则 | 稳定 | [chat-ui.md](modules/chat-ui.md) |
| Media | 上传、媒体库、发送日志、查重、修复、海报 | 稳定 | [media.md](modules/media.md) |
| Douyin Downloader | TikTokDownloader 对接、抖音媒体导入与收藏标签 | 稳定 | [douyin-downloader.md](modules/douyin-downloader.md) |
| mtPhoto | mtPhoto 相册浏览、缩略图、原图下载、导入与文件夹收藏 | 稳定 | [mtphoto.md](modules/mtphoto.md) |
| MT Photos 上游 API | MT Photos 官方 OpenAPI 快照、认证、全量端点和 schema 索引 | 文档快照 | [mtphotos-upstream-api.md](modules/mtphotos-upstream-api.md) |
| Video Extract | ffmpeg/ffprobe 视频抽帧任务与帧索引 | 稳定 | [video-extract.md](modules/video-extract.md) |
| System Config | 全局配置、图片端口解析、连接与 forceout 管理 | 稳定 | [system-config.md](modules/system-config.md) |
| Android Client | Kotlin/Compose 移动端客户端 | 开发中 | [android-client.md](modules/android-client.md) |

---

## 3. 快速链接
- [技术约定](../project.md)
- [架构设计](arch.md)
- [API 手册](api.md)
- [数据模型](data.md)
- [变更历史](../history/index.md)
