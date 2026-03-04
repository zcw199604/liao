# 任务清单: 抖音作品抓取与下载对接（TikTokDownloader Web API）

目录: `helloagents/history/2026-01/202601211132_douyin_downloader/`

---

## 1. 后端：TikTokDownloader 对接与 API
- [√] 1.1 在 `internal/config/config.go` 增加 TikTokDownloader/Douyin 相关配置项与 env 读取，验证 why.md#需求-解析与详情抓取-场景-解析短链分享文本
- [√] 1.2 在 `internal/app/` 增加 TikTokDownloader Web API 客户端与 `detail_id` 提取逻辑，验证 why.md#需求-解析与详情抓取-场景-解析完整-url作品id
- [√] 1.3 在 `internal/app/router.go` 增加 `/api/douyin/*` 路由并实现 handlers（detail/download/import），验证 why.md#需求-下载到本地（流式）-场景-视频下载
- [√] 1.4 在 `internal/app/` 增加短期缓存（key+TTL）用于 download/import，验证 why.md#风险评估
- [√] 1.5 在 `internal/app/` 实现导入上传（下载→SaveFileFromReader→MD5 去重→上传上游→写入 media_file），验证 why.md#需求-可选导入上传到系统媒体库-场景-导入上传并去重

## 2. 前端：抖音下载入口与弹窗
- [√] 2.1 在 `frontend/src/components/chat/UploadMenu.vue` 增加“抖音下载”入口并在聊天页触发打开，验证 why.md#需求-解析与详情抓取
- [√] 2.2 新增 `frontend/src/components/media/DouyinDownloadModal.vue`（解析输入/配置 proxy+cookie/展示解析结果/预览），验证 why.md#需求-解析与详情抓取
- [√] 2.3 新增 `frontend/src/api/douyin.ts` 与 `frontend/src/stores/douyin.ts`，实现解析与导入上传调用，验证 why.md#需求-可选导入上传到系统媒体库

## 3. 安全检查
- [√] 3.1 执行安全检查（输入校验、避免任意 URL 下载、cookie 不落库、鉴权继承现有 JWT）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/api.md` 增补抖音下载相关接口说明
- [√] 4.2 新增 `helloagents/wiki/modules/douyin-downloader.md` 模块文档，并在 `helloagents/wiki/overview.md`（如有模块索引）补充入口
- [√] 4.3 更新 `helloagents/CHANGELOG.md`（Unreleased）与 `README.md` 环境变量说明

## 5. 测试
- [√] 5.1 新增/更新 Go 单测（`internal/app/*_test.go`）覆盖 detail_id 提取与 handler 关键路径
- [√] 5.2 运行 `go test ./...` 与 `cd frontend && npm run build` 验证编译与测试通过
