# Douyin Downloader（抖音抓取与下载）

## 目的
在应用内对接外部 TikTokDownloader Web API（FastAPI），实现抖音作品的：
- 分享链接解析（短链/分享文本/URL/作品ID）
- 作品详情抓取（视频/图集）
- 下载到本地（以作品标题命名）
- 可选导入上传到系统媒体库（并加入“已上传的文件”，用于发送）

## 模块概述
该模块由后端统一调用 TikTokDownloader Web API：
- `/douyin/share`：将短链/分享文本解析为重定向后的完整链接
- `/douyin/detail`：获取结构化作品详情（包含 `desc/type/downloads` 等字段）

前端通过弹窗交互完成输入与配置，并复用现有 `MediaPreview` 完成预览、下载与导入上传。

## 入口与交互
- 入口：侧边栏顶部菜单 → “图片管理” → “抖音下载”（打开弹窗）
- 交互：
  - 模式A：作品解析
    1) 粘贴分享文本/短链/URL/作品ID
    2)（可选）填写 `cookie`（仅本地保存）
    3) 点击“解析”获取作品资源列表（支持“多选模式”批量下载/导入，并展示逐项状态）
    4) 点击资源进入预览（预览中可一键导入上传，导入成功后不强制关闭弹窗）：
       - “下载”：走 `/api/douyin/download` 获取下载流并以作品标题命名
       - “上传”：走 `/api/douyin/import` 由后端下载并导入上传（MD5 去重）

> 备注：抖音 CDN 对跨站媒体子资源有防盗链校验，`items[].url` 直链可能无法在站内 `<img>/<video>` 预览；
> 前端应优先使用 `items[].downloadUrl`（`/api/douyin/download`）进行缩略图与预览加载。
  - 模式B：用户作品
    1) 粘贴用户主页链接/分享文本/sec_uid
    2) 点击“获取作品”拉取该用户发布作品列表（当前实现默认全量拉取；列表展示仍可复用分页接口）
    3) 点击某个作品 → 进入预览：支持左右滑动在“当前已加载作品”之间切换，并在预览顶部显示当前作品名称（best-effort 使用 `/api/douyin/account` 返回的 `key/items`，避免再请求 `/api/douyin/detail`；若缺失则回退到“作品解析”抓取资源列表）
  - 模式C：收藏
    1) 查看“用户收藏 / 作品收藏”列表（全局一份，不按本地身份隔离）
    2) 用户收藏：支持“再次解析”一键重新拉取该用户作品列表，并同步更新收藏记录的 `last_parsed_at/last_parsed_count`
    3) 作品收藏：支持“一键解析”重新抓取该作品详情，获取最新可下载资源列表
    4) 在“用户作品/作品解析”模式中支持点星标收藏/取消收藏（用户在用户作品页头部收藏；作品在卡片右上角/详情页头部收藏）

**本地配置（localStorage）**
- `douyin_cookie`：Cookie（可选；支持一键清除）
- `douyin_auto_clipboard`：打开弹窗自动读取剪贴板（默认开启；写入 `1/0`）
- `douyin_auto_resolve_clipboard`：读取剪贴板后自动解析（默认关闭；写入 `1/0`）

## 核心流程
### 1) 解析与详情抓取
`POST /api/douyin/detail`：
- 输入优先本地解析 `detail_id`（支持 `/video/<id>`、`/note/<id>`、`modal_id=<id>`、纯数字）
- 不能解析时调用 `/douyin/share` 获取重定向 URL 后再提取 `detail_id`
- 调用 `/douyin/detail` 获取结构化 `data` 并抽取：
  - `desc` → 标题（用于命名）
  - `type` → `视频/图集/实况`
  - `downloads` → 可下载资源（视频为单条 URL；图集为 URL 列表）
- 服务端生成短期缓存 `key`（TTL），供下载/导入使用

### 1.5) 用户作品列表
`POST /api/douyin/account`：
- 输入优先本地解析 `sec_user_id`（支持 `/user/<sec_uid>`、`sec_uid=<sec_uid>`、直接粘贴 `sec_uid`）
- 不能解析时调用 `/douyin/share` 获取重定向 URL 后再提取 `sec_user_id`
- 调用 `/douyin/account` 拉取发布作品列表（`aweme_list`），并返回 `cursor/hasMore/items`
- best-effort：服务端会尝试从 `aweme_list` 直接抽取可预览资源（视频 `play_addr` / 图集 `images[].url_list`），并为每个作品生成缓存 `key` 与 `items[].downloadUrl`；前端可直接预览/下载/导入，无需再请求 `/api/douyin/detail`
- 兼容：当上游返回 `data[]` 扁平结构时（如包含 `type/downloads/static_cover/dynamic_cover`），服务端也会为每个作品生成 `key/items/coverDownloadUrl`，避免前端点击后再回退请求详情

### 2) 下载到本地
`GET /api/douyin/download?key=...&index=...`：
- 仅允许使用服务端缓存的 `key + index` 取出下载直链（不接受任意 URL），降低 SSRF 风险
- 透传下载流并设置 `Content-Disposition`：
  - 视频：`标题.mp4`
  - 图集：`标题_01.jpg`（按序号追加）

`HEAD /api/douyin/download?key=...&index=...`：
- 用于前端“最佳努力”探测 `Content-Length` 展示文件大小徽标（CDN 不支持 `HEAD` 时后端会回退 `Range: bytes=0-0`）。

### 3) 导入上传（加入媒体库）
`POST /api/douyin/import`：
- 后端下载媒体 → 保存到 `./upload` → 计算 MD5
- 若当前用户已存在同 MD5 的媒体记录，则直接复用（避免重复写文件/重复上传；响应 `dedup=true`）
- 否则按既有上传链路上传到上游图片服务器并写入 `media_file`，最后加入“已上传的文件”缓存

## 安全与约束
- **不落库敏感信息**：抖音 `cookie` 不写入服务端存储；前端仅保存在 localStorage 并在请求中透传（页面填写优先）。
- **SSRF 风险控制**：download/import 不接受任意 URL，只接受 `key+index` 并从服务端缓存读取下载直链。
- **大文件处理**：下载与导入均采用流式转发与落盘，避免一次性读入内存。

## API接口
详见 `helloagents/wiki/api.md` 的 “4.9 抖音下载（TikTokDownloader Web API）”（含抖音收藏接口）。

## 配置（环境变量）
- `TIKTOKDOWNLOADER_BASE_URL`：TikTokDownloader Web API 地址（必配才能启用）
- `TIKTOKDOWNLOADER_TOKEN`：可选，上游 token Header（默认上游不校验）
- `TIKTOKDOWNLOADER_TIMEOUT_SECONDS`：可选，调用 TikTokDownloader Web API 超时（秒，默认跟随 `UPSTREAM_HTTP_TIMEOUT_SECONDS`）
- `DOUYIN_COOKIE`：可选，默认抖音 Cookie（页面填写优先）
- `DOUYIN_PROXY`：可选，服务端默认代理（前端不提供输入）
- `UPSTREAM_HTTP_TIMEOUT_SECONDS`：可选，调用上游 HTTP 接口超时（秒，默认 60）

## 依赖
- 外部服务：TikTokDownloader Web API（FastAPI）
- 前端：`MediaPreview`（预览/下载/导入上传复用现有交互）

## 测试
- Go：`go test ./...`（包含 douyin handler 单测）
- 前端：`npm run build`（作为编译验证）

## 变更历史
- [202601211132_douyin_downloader](../../history/2026-01/202601211132_douyin_downloader/) - 抖音抓取/下载/导入上传对接
- [202601211234_douyin_downloader_ux](../../history/2026-01/202601211234_douyin_downloader_ux/) - 抖音下载弹窗交互增强（批量/剪贴板预填/文件大小探测/导入状态与去重提示）
