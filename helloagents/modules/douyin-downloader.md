# Douyin Downloader（抖音抓取与下载）

## 目的
在应用内对接外部 TikTokDownloader Web API（FastAPI），实现抖音作品的：
- 分享链接解析（短链/分享文本/URL/作品ID）
- 作品详情抓取（视频/图集）
- 下载到本地（以作品标题命名）
- 导入到本地媒体库（仅落盘 + 全局去重，不自动上传上游；需要发送时在“全站图片库/所有图片”中手动上传）

## 模块概述
该模块由后端统一调用 TikTokDownloader Web API：
- `/douyin/share`：将短链/分享文本解析为重定向后的完整链接
- `/douyin/detail`：获取结构化作品详情（包含 `desc/type/downloads` 等字段）

前端通过弹窗交互完成输入与配置，并复用现有 `MediaPreview` 完成预览、下载与导入。

## 入口与交互
- 入口：侧边栏顶部菜单 → “图片管理” → “抖音下载”（打开弹窗）
- 交互：
  - 模式A：作品解析
    1) 粘贴分享文本/短链/URL/作品ID
    2)（可选）填写 `cookie`（仅本地保存）
    3) 点击“解析”获取作品资源列表（支持“多选模式”批量下载/导入，并展示逐项状态）
    4) 点击资源进入预览（预览中可一键导入到本地；导入成功后不强制关闭弹窗）：
       - “下载”：走 `/api/douyin/download` 获取下载流并以作品标题命名
       - “上传”：走 `/api/douyin/import` 仅导入到本地（MD5 全局去重）；导入后需在“全站图片库/所有图片”中手动点击“上传此图片”上传到上游后发送
    5) 当作品资源同时包含图片+视频（如“实况照片”），预览画廊会合并全部资源，便于左右切换查看
       - 在“实况照片”中：静态图支持**长按播放**对应的实况短视频（松开停止），并提供“下载实况 (Live Photo)”按钮生成 iOS 可识别的实况文件对（ZIP：`.jpg` + `.mov`）

> 备注：抖音 CDN 对跨站媒体子资源有防盗链校验，`items[].url` 直链可能无法在站内 `<img>/<video>` 预览；
> 前端应优先使用 `items[].downloadUrl`（`/api/douyin/download`）进行缩略图与预览加载。
  - 模式B：用户作品
    1) 粘贴用户主页链接/分享文本/sec_uid
    2) 点击“获取作品”拉取该用户发布作品列表（默认只拉取一页；可用“加载更多”手动翻页，或点击“全量拉取”一次性补齐剩余分页）
    3) 点击某个作品 → 进入预览：支持左右滑动在“当前已加载作品”之间切换，并在预览顶部显示当前作品名称（best-effort 使用 `/api/douyin/account` 返回的 `key/items`，避免再请求 `/api/douyin/detail`；若缺失则回退到“作品解析”抓取资源列表）
  - 模式C：收藏
    1) 查看“用户收藏 / 作品收藏”列表（全局一份，不按本地身份隔离）
    1.1) 分类标签：用户收藏标签与作品收藏标签两套体系互不影响；每条收藏可绑定多个标签；无标签即“未分类”
    1.2) 支持按“全部 / 未分类 / 指定标签”筛选收藏条目
    1.3) 支持批量模式：勾选多个条目后批量添加标签（不移除已有标签）
    1.4) 提供独立“标签管理页”：新建/重命名/删除；删除标签会从所有绑定条目移除（条目保留 → 未分类）
    2) 用户收藏：支持“再次解析”一键重新拉取该用户作品列表，并同步更新收藏记录的 `last_parsed_at/last_parsed_count`
    2.1) 用户收藏：点击用户卡片可打开详情抽屉，展示头像/简介/统计，并支持“刷新信息”更新昵称与头像（头像失效常见于用户更换头像）
    2.2) 用户收藏：当用户在“用户作品”模式中收藏后，会将**当前已抓取到的作品列表元信息**入库；在详情抽屉中展示作品网格，支持滚动分页加载；点击作品可预览并生成“当前已加载作品合并画廊”；提供“获取最新作品”按钮调用上游拉取最新数据并合并入库，完成后提示新增作品数并刷新列表
    3) 作品收藏：支持“一键解析”重新抓取该作品详情，获取最新可下载资源列表
    4) 在“用户作品/作品解析”模式中支持点星标收藏/取消收藏（用户在用户作品页头部收藏；作品在卡片右上角/详情页头部收藏）

**本地配置（localStorage）**
- `douyin_cookie`：Cookie（可选；支持一键清除）
- `douyin_auto_clipboard`：打开弹窗自动读取剪贴板（默认开启；写入 `1/0`）
- `douyin_auto_resolve_clipboard`：读取剪贴板后自动解析（默认关闭；写入 `1/0`）

**剪贴板兼容性说明**
- 移动端/部分 WebView（如微信/抖音内置浏览器）可能不支持 `navigator.clipboard.readText/writeText`，或要求 HTTPS + 用户手势授权。
- 弹窗“粘贴”在不支持读取剪贴板时会提示并聚焦输入框，需要用户长按手动粘贴。
- 手动粘贴会触发输入识别（作品/用户），并在开启“读取后自动解析/获取”时自动执行解析流程。
- 弹窗内的“复制”会优先使用 `navigator.clipboard.writeText`，失败时回退到 `document.execCommand('copy')`，以提升移动端兼容性。

## 核心流程
### 1) 解析与详情抓取
`POST /api/douyin/detail`：
- 输入优先本地解析 `detail_id`（支持 `/video/<id>`、`/note/<id>`、`modal_id=<id>`、纯数字）
- 不能解析时调用 `/douyin/share` 获取重定向 URL 后再提取 `detail_id`
- 调用 `/douyin/detail` 获取结构化 `data` 并抽取：
  - `desc` → 标题（用于命名）
  - `type` → `视频/图集/实况`
  - `downloads` → 可下载资源（可能是字符串或 URL 列表；视频通常为单条 URL；图集为 URL 列表；实况可能为“静态图 + 播放直链视频”的混合列表）
- 服务端生成短期缓存 `key`（TTL），供下载/导入使用
  - 服务端会按每个 URL best-effort 推断 `items[].type`（`image/video`），避免上游 `type` 字段不一致导致图片被误判为视频

### 1.2) iOS 实况下载（Live Photo）
`GET /api/douyin/livePhoto`：
- 将同一作品缓存 `key` 下的一张静态图（`imageIndex`）与一段短视频（`videoIndex`）打包导出为“实况照片”
- 支持 `format` 参数：
  - `format=zip`（默认）：输出 ZIP 包：`{title_01_live}.zip`（文件名与普通图片下载一致并追加 `_live`）；ZIP 内包含 `{title_01}.jpg` + `{title_01}.mov`，并写入相同 `ContentIdentifier`（Apple Live Photo 配对）
  - `format=jpg`：输出单文件 Motion Photo：`{title_01_live}.jpg`（JPEG 末尾附加 MP4，并写入 XMP `GCamera:MicroVideoOffset`）；为提升 MIUI 相册识别概率，JPEG 头部段顺序为 `APP1(Exif) → APP1(XMP) → APP0(JFIF)`，并在 `EOI` 与 MP4 之间插入 24 字节 gap
- 依赖系统命令：`ffmpeg`（两种格式都需要）；`format=zip` 额外依赖 `exiftool`

### 1.5) 用户作品列表
`POST /api/douyin/account`：
- 输入优先本地解析 `sec_user_id`（支持 `/user/<sec_uid>`、`sec_uid=<sec_uid>`、直接粘贴 `sec_uid`）
- 不能解析时调用 `/douyin/share` 获取重定向 URL 后再提取 `sec_user_id`
- 调用包装镜像的单页分页接口 `/douyin/account/page` 拉取发布作品列表（`items`），并在应用侧转换为 `cursor/hasMore/items`
- best-effort：服务端会尝试从返回数据中提取用户信息（`displayName/avatarUrl/profileUrl`），供前端收藏/展示昵称使用
- best-effort：服务端会尝试从 `aweme_list` 直接抽取可预览资源（视频 `play_addr` / 图集 `images[].url_list`），并为每个作品生成缓存 `key` 与 `items[].downloadUrl`；前端可直接预览/下载/导入，无需再请求 `/api/douyin/detail`
  - 当 `images[].url_list` 同时包含 `.webp`/`.jpeg` 等多种格式时：普通图集默认优先选择 WebP；若判定为“实况”（图片+视频混合）则优先选择 JPEG
- 兼容：当上游返回 `data[]` 扁平结构时（如包含 `type/downloads/static_cover/dynamic_cover`），服务端也会为每个作品生成 `key/items/coverDownloadUrl`，避免前端点击后再回退请求详情

### 2) 下载到本地
`GET /api/douyin/download?key=...&index=...`：
- 仅允许使用服务端缓存的 `key + index` 取出下载直链（不接受任意 URL），降低 SSRF 风险
- 透传下载流并设置 `Content-Disposition`：
  - 视频：`标题.mp4`
  - 图集：`标题_01.jpg`（按序号追加）
  - 实况：按 `items[].type` 分别命名（视频默认 `.mp4`，图片默认 `.jpg`；若 URL 带扩展名则优先使用）

`HEAD /api/douyin/download?key=...&index=...`：
- 用于前端“最佳努力”探测 `Content-Length` 展示文件大小徽标（CDN 不支持 `HEAD` 时后端会回退 `Range: bytes=0-0`）。

### 3) 导入到本地（加入媒体库）
`POST /api/douyin/import`：
- 后端下载媒体 → 保存到 `./upload/douyin`（`douyin/images/YYYY/MM/DD` 或 `douyin/videos/YYYY/MM/DD`）→ 计算 MD5
- 若全局已存在同 MD5 的媒体文件（`media_file` 或 `douyin_media_file`），则删除临时落盘文件并复用已有文件（响应 `dedup=true`，同时刷新 `update_time` 便于在“全站图片库”置顶）
- 写入/更新 `douyin_media_file`（包含 `sec_user_id/detail_id` 元信息；`remote_url/remote_filename` 为空），响应返回 `localPath/localFilename` 与 `uploaded=false`

> 说明：当未选择本地身份时，导入会使用 `userid=pre_identity` 作为兜底，不影响按抖音 `sec_user_id` 的筛选与归档。

## 安全与约束
- **敏感信息处理（cookie）**：
  - 默认行为：抖音 `cookie` 不落库；前端仅保存在 localStorage 并在请求中透传（页面填写优先）。
  - 启用 CookieCloud 自动 cookie 时：服务端会将“抖音 cookie header value”（例如 `a=1; b=2`）缓存到 Redis 或内存中并设置 TTL（默认 72 小时），以减少频繁拉取；仍不写入数据库。
- **SSRF 风险控制**：download/import 不接受任意 URL，只接受 `key+index` 并从服务端缓存读取下载直链。
- **大文件处理**：下载与导入均采用流式转发与落盘，避免一次性读入内存。

## API接口
详见 `helloagents/modules/api.md` 的 “4.9 抖音下载（TikTokDownloader Web API）”（含抖音收藏接口）。

## 配置（环境变量）
- `TIKTOKDOWNLOADER_BASE_URL`：TikTokDownloader Web API 地址（必配才能启用）
- `TIKTOKDOWNLOADER_TOKEN`：可选，上游 token Header（默认上游不校验）
- `TIKTOKDOWNLOADER_TIMEOUT_SECONDS`：可选，调用 TikTokDownloader Web API 超时（秒，默认跟随 `UPSTREAM_HTTP_TIMEOUT_SECONDS`）
- `DOUYIN_COOKIE`：可选，默认抖音 Cookie（页面填写优先；当启用 CookieCloud 时优先级低于 CookieCloud）
- `DOUYIN_PROXY`：可选，服务端默认代理（前端不提供输入）
- `UPSTREAM_HTTP_TIMEOUT_SECONDS`：可选，调用上游 HTTP 接口超时（秒，默认 60）

### CookieCloud 自动抖音 Cookie（可选）

当抖音相关请求未显式传入 `cookie` 时，服务端可自动从 CookieCloud 拉取并解密抖音 cookie，并进行缓存复用。

优先级（从高到低）：
1) 请求体显式传入的 `cookie`（保持现有行为）
2) CookieCloud 拉取到的 cookie（本项目新增）
3) `DOUYIN_COOKIE` 默认值

相关环境变量：
- `COOKIECLOUD_BASE_URL`：CookieCloud 服务地址（可包含 API_ROOT 前缀）
- `COOKIECLOUD_UUID`：CookieCloud UUID
- `COOKIECLOUD_PASSWORD`：CookieCloud 解密密码
- `COOKIECLOUD_DOMAIN`：用于提取抖音 cookie 的 domain（默认 `douyin.com`，会自动兼容 `.douyin.com`）
- `COOKIECLOUD_CRYPTO_TYPE`：可选，`legacy` / `aes-128-cbc-fixed`；留空表示使用服务端返回的 `crypto_type`
- `COOKIECLOUD_COOKIE_EXPIRE_HOURS`：可选，缓存 TTL（小时），默认 `72`（3 天）

缓存介质：
- `CACHE_TYPE=redis` 时：缓存写入 Redis（带 TTL），服务重启后 TTL 内仍可复用
- 否则：仅进程内缓存（重启即失效）

## 依赖
- 外部服务：TikTokDownloader Web API（FastAPI）
- 前端：`MediaPreview`（预览/下载/导入复用现有交互）

## 测试
- Go：`go test ./...`（包含 douyin handler 单测）
- 前端：`npm run build`（作为编译验证）

## 变更历史
- [202601211132_douyin_downloader](../../history/2026-01/202601211132_douyin_downloader/) - 抖音抓取/下载/导入上传对接
- [202601211234_douyin_downloader_ux](../../history/2026-01/202601211234_douyin_downloader_ux/) - 抖音下载弹窗交互增强（批量/剪贴板预填/文件大小探测/导入状态与去重提示）
