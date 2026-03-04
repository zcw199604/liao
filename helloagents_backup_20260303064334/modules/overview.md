# Liao（匿名匹配聊天应用）

> 本文件包含项目级别的核心信息。详细的模块规范见 `modules/` 目录；接口与数据以 `api.md` / `data.md` 为准（SSOT）。

---

## 1. 项目概述

### 目标与背景
Liao 是一个基于 WebSocket 的匿名匹配聊天应用/代理服务：前端通过 `/ws` 与后端建立下游连接，后端再连接上游聊天服务并转发消息，同时提供本地身份管理、缓存增强、媒体上传与静态资源托管能力。

> 注：历史 Java(Spring Boot) 源码已从仓库移除；本文与 `api.md` / `data.md` 均以 Go 实现为准（`cmd/liao` + `internal/`），并在必要处说明与“原 Spring Boot 行为”的兼容点。

### 范围
- **范围内:** 匿名聊天核心链路（登录/JWT、会话列表、聊天收发、WebSocket 代理）、本地身份池、媒体上传与媒体库、上游 HTTP/WS 代理与缓存增强（用户信息/最后消息）。
- **范围外:** 上游业务逻辑的修改（仅代理与兼容）、第三方账号体系/支付体系、运营/后台管理系统。

### 干系人
- **负责人:** （待补充）

---

## 2. 模块索引

| 模块名称 | 职责 | 状态 | 文档 |
|---------|------|------|------|
| Auth | 认证与鉴权（访问码登录/JWT/WS 握手校验） | ✅稳定 | [链接](modules/auth.md) |
| Chat UI | 聊天界面交互规范（手势/弹层/WS 身份切换） | ✅稳定 | [链接](modules/chat-ui.md) |
| Identity | 本地身份池管理（CRUD/最近使用） | ✅稳定 | [链接](modules/identity.md) |
| Media | 媒体上传/媒体库/删除/重传/修复 | ✅稳定 | [链接](modules/media.md) |
| mtPhoto | mtPhoto 相册接入与导入上传 | ✅稳定 | [链接](modules/mtphoto.md) |
| Douyin Downloader | 抖音作品抓取/下载/导入上传（对接 TikTokDownloader Web API） | ✅稳定 | [链接](modules/douyin-downloader.md) |
| User History | 历史/收藏用户列表与缓存增强规范 | ✅稳定 | [链接](modules/user-history.md) |
| WebSocket Proxy | `/ws` 下游/上游连接池与转发、forceout 防重连 | ✅稳定 | [链接](modules/websocket-proxy.md) |

---

## 3. 快速链接
- [技术约定](../project.md)
- [架构设计](arch.md)
- [API 手册](api.md)
- [数据模型](data.md)
- [变更历史](../history/index.md)

---

## 4. 测试与验证

- 后端（Go）：`go test ./...`
- 前端（Vue/Vitest）：`cd frontend && npm test`
- 说明：Vitest 在 `jsdom` 下的浏览器能力由 `frontend/vitest.setup.ts` 提供最小 polyfills；当引入新组件/依赖需要额外 Web API 时，优先在该文件补齐以保证用例可重复执行。
