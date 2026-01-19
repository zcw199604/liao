# 变更提案: mtPhoto 图片下载改为原图

## 需求背景
当前 mtPhoto 相册列表与预览展示使用的是缩略图（通过后端代理 `GET /api/getMtPhotoThumb` 获取）。这导致用户在预览面板点击“下载”时，实际下载到的仍是缩略图而非原图，无法满足保存高清原图的需求。

mtPhoto 上游提供了原图下载接口：`/gateway/fileDownload/{id}/{md5}`（示例：`http://pc.zcw.work:38064/gateway/fileDownload/756956/315c8ea48a132db5e1e3cfe33b6a864a`），因此需要在“下载”动作中改为调用该接口获取原图。

## 变更内容
1. 后端新增 mtPhoto 原图下载代理接口（要求 JWT 登录态），内部调用 mtPhoto `gateway/fileDownload` 并以流式方式透传二进制到前端。
2. 前端在构造 mtPhoto 预览媒体对象时补充原图下载地址（与展示用缩略图 URL 解耦），`MediaPreview` 的下载按钮优先使用原图下载地址进行下载。
3. 保持非 mtPhoto 场景下载行为不变（仍按原 URL 直接下载）。

## 影响范围
- **模块:** mtPhoto、Media（前端预览组件）、API（后端新增下载代理）
- **文件（预估）:**
  - `internal/app/mtphoto_client.go`
  - `internal/app/mtphoto_handlers.go`
  - `internal/app/router.go`
  - `frontend/src/components/media/MtPhotoAlbumModal.vue`
  - `frontend/src/components/media/MediaPreview.vue`
  - `frontend/src/types/media.ts`
  - `helloagents/wiki/api.md`
  - `helloagents/wiki/modules/mtphoto.md`
- **API:** 新增 `GET /api/downloadMtPhotoOriginal`
- **数据:** 无

## 核心场景

### 需求: mtPhoto 图片下载应下载原图
**模块:** mtPhoto / Media / API
在 mtPhoto 相册预览中点击“下载”时，下载内容应为原图而非缩略图。

#### 场景: 在 mtPhoto 相册预览中点击下载下载原图
在 mtPhoto 相册弹窗中进入某个相册并点击任意图片进入预览：
- 预览仍展示缩略图/预览图（保持加载性能）
- 点击“下载”后保存到本地的是原图文件（来自 mtPhoto fileDownload 接口）

#### 场景: 画廊切换后下载应对齐当前图片
在预览画廊中左右切换到另一张图片后：
- 点击“下载”下载的是当前所见图片的原图（不下载上一张/缩略图）

### 需求: 非 mtPhoto 下载行为保持不变
**模块:** Media
其他来源（全站图片库、查重预览、聊天消息内文件等）下载逻辑不应被本次改动影响。

#### 场景: 非 mtPhoto 图片/文件仍可正常下载
在任意非 mtPhoto 的预览面板中点击“下载”：
- 下载行为与当前一致（URL 不变、可正常保存）

## 风险评估
- **风险:** 原图文件体积更大，若处理不当可能导致内存占用上升或下载失败。
  - **缓解:** 后端使用流式透传（`io.Copy`）避免将全量文件读入内存；仅透传必要响应头。
- **风险:** 下载接口可能被滥用为代理下载通道。
  - **缓解:** 该接口保持 JWT 鉴权；对 `id/md5` 做参数校验（格式/范围）；不加入 JWT 放行名单。
