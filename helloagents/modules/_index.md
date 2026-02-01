# 模块索引

> 模块文档统一位于 `helloagents/modules/`。
> `helloagents/wiki/` 为兼容旧链接保留（stub）。

## 模块清单

| 模块 | 职责 | 状态 | 文档 |
|------|------|------|------|
| auth | 登录/鉴权（access code + JWT） | ✅ | [auth.md](./auth.md) |
| chat-ui | 聊天 UI/交互规范 | ✅ | [chat-ui.md](./chat-ui.md) |
| douyin-downloader | 抖音下载/导入 | ✅ | [douyin-downloader.md](./douyin-downloader.md) |
| douyin-livephoto | 抖音实况（Live Photo）支持 | ✅ | [douyin-livephoto.md](./douyin-livephoto.md) |
| identity | 身份管理 | ✅ | [identity.md](./identity.md) |
| media | 媒体上传/预览/画廊 | ✅ | [media.md](./media.md) |
| mtphoto | mtPhoto 相册 | ✅ | [mtphoto.md](./mtphoto.md) |
| user-history | 历史用户列表/最后消息缓存 | ✅ | [user-history.md](./user-history.md) |
| websocket-proxy | WebSocket 代理（/ws）与连接池 | ✅ | [websocket-proxy.md](./websocket-proxy.md) |
| frontend-theme | 前端主题与排版约定 | ✅ | [frontend-theme.md](./frontend-theme.md) |

## 通用文档

- 架构概览: [arch.md](./arch.md)
- 功能概览: [overview.md](./overview.md)
- API 说明: [api.md](./api.md)
- 数据模型: [data.md](./data.md)
- 外部参考:
  - [external/cookiecloud.md](./external/cookiecloud.md)
  - [external/tiktokdownloader-web-api.md](./external/tiktokdownloader-web-api.md)
  - [external/tiktokdownloader-web-api-sdk.md](./external/tiktokdownloader-web-api-sdk.md)

## 模块依赖关系（高层）

```
chat-ui → websocket-proxy → 上游 WS
chat-ui → auth/identity
chat-ui → media → upload/
douyin-downloader → media/data
```
