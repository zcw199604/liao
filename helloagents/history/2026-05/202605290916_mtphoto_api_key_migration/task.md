# 任务清单: mtPhoto API Key 接入方式迁移

目录: `helloagents/history/2026-05/202605290916_mtphoto_api_key_migration/`

---

## 并行子代理标注

- 并行组 A: 任务 [1.1, 2.1]；允许写入: `internal/config/config.go`, `src/main/resources/application.yml`, `internal/app/mtphoto_client.go`；冲突域: `NewMtPhotoService` 构造签名需由主线协调；验证: `go test ./internal/config ./internal/app -run 'MtPhoto|Config'`
- 并行组 B: 任务 [4.1]；允许写入: `helloagents/wiki/modules/mtphoto.md`, `helloagents/wiki/api.md`, `helloagents/project.md`；冲突域: 需等待最终实现细节确认；验证: `git diff --check`
- 不可并行任务: [2.2, 3.1, 5.1, 6.1]；原因: 媒体网关、视频抽帧依赖和最终验证需要在核心客户端签名稳定后执行

---

## 0. 方案边界确认
- [√] 0.1 确认本次任务仅覆盖 why.md 的范围内切片，不代理全部 MT Photos OpenAPI，验证 why.md#范围边界
- [√] 0.2 确认 how.md 的设计边界完整，尤其是 `MTPHOTO_API_KEY`、本地 API 兼容和无数据库变更，验证 why.md#核心场景
- [√] 0.3 大型项目确认最小改动策略: 不改本地路由命名、不重写前端 mtPhoto UI、不做无关重构，验证 how.md#设计边界

## 1. 配置迁移
- [√] 1.1 RED: 在 `internal/config/config_test.go` 增加 `MTPHOTO_API_KEY` 加载与空值兼容测试，确认当前实现缺少该字段，验证 why.md#需求-api-key-认证替换账号密码登录
- [√] 1.2 GREEN: 在 `internal/config/config.go` 新增 `MtPhotoAPIKey`，读取 `MTPHOTO_API_KEY`；保留旧 `MTPHOTO_LOGIN_*` 字段但标注废弃，依赖任务1.1
- [√] 1.3 在 `src/main/resources/application.yml` 的 mtPhoto 配置段新增 `api-key: ${MTPHOTO_API_KEY:}`，并注明旧登录配置仅兼容历史说明，依赖任务1.2
- [√] 1.4 在 `internal/app/app.go` 调整 `NewMtPhotoService` 调用参数为 `baseURL/apiKey/lspRoot/httpClient`，依赖任务1.2

## 2. 上游客户端认证替换
- [√] 2.1 RED: 在 `internal/app/mtphoto_client_test.go` 或相邻测试中新增断言：普通 API 请求发送 `x-api-key`，不调用 `/auth/login`，验证 why.md#需求-api-key-认证替换账号密码登录-场景-查询相册列表
- [√] 2.2 GREEN: 在 `internal/app/mtphoto_client.go` 改造 `MtPhotoService` 字段和构造函数，移除主链路 username/password/otp 登录依赖，`configured()` 改为 `baseURL + apiKey`，依赖任务2.1
- [√] 2.3 在 `internal/app/mtphoto_client.go` 改造普通 JSON 请求通道，统一添加 `x-api-key`，保留 401/403 错误映射，依赖任务2.2
- [√] 2.4 更新 `GetAlbums`、`GetAlbumFilesPage`、`getFolderData`、`getFolderTimelineFiles`、`ListSameMediaByMD5`、`GetFileInfo` 等调用点，移除 `useJWT/useCookie` 参数语义，依赖任务2.3

## 3. 媒体网关授权码替换
- [√] 3.1 RED: 为 `GatewayGet` 添加失败测试，断言它调用 `/auth/auth_code` 并请求 `/gateway/{size}/{md5}?auth_code=...`，验证 why.md#需求-媒体资源使用-auth_code-query-参数-场景-获取缩略图
- [√] 3.2 GREEN: 在 `internal/app/mtphoto_client.go` 新增 `ensureAuthCode(ctx, force)`，用 `POST /auth/auth_code` body `{api_key}` 获取并缓存授权码，依赖任务3.1
- [√] 3.3 改造 `GatewayGet` 和 `GatewayFileDownload`，在 URL query 添加 `auth_code`，不再设置 `Cookie: auth_code=...`，依赖任务3.2
- [√] 3.4 为媒体请求添加 401/403 后强制刷新 auth_code 并重试一次的逻辑，避免过期授权码导致图片批量失效，依赖任务3.3
- [√] 3.5 确认 `handleGetMtPhotoThumb` 的 size 白名单、`handleDownloadMtPhotoOriginal` 的响应头裁剪和错误处理保持不变，依赖任务3.3

## 4. 关联模块适配
- [√] 4.1 更新 `internal/app/video_extract.go`、`internal/app/video_extract_handlers.go` 中对 mtPhoto resolver 的测试替身或构造调用，确保 mtPhoto 视频作为抽帧来源仍可解析 `/lsp/*`，验证 why.md#需求-现有本地接口兼容-场景-浏览文件夹与导入媒体
- [√] 4.2 更新所有 `NewMtPhotoService(...)` 测试调用点，使用 API Key 新签名并替换对 `jwt`/`Cookie` 的旧断言，依赖任务2.2、3.3
- [√] 4.3 检查 `frontend/src/api/mtphoto.ts` 和 `frontend/src/stores/mtphoto.ts`，确认无需变更本地 API 形态；若发现因响应字段变化产生类型风险，只做最小兼容修正，依赖任务2.4

## 5. 安全检查
- [√] 5.1 执行安全检查: 确认 `MTPHOTO_API_KEY` 不进入日志、错误响应、测试快照或知识库示例真实值，验证 how.md#安全与性能
- [√] 5.2 检查 `/api/getMtPhotoThumb` 未成为开放代理: size 仍只允许 `s260`、`h220`，md5 仍不能为空，验证 why.md#需求-媒体资源使用-auth_code-query-参数-场景-获取缩略图
- [√] 5.3 检查 `importMtPhotoMedia`、`getMtPhotoSameMedia` 的 `/lsp` 与 `/upload` 路径越界防护没有因客户端迁移被削弱，验证 why.md#需求-现有本地接口兼容-场景-浏览文件夹与导入媒体

## 6. 文档更新
- [√] 6.1 更新 `helloagents/wiki/modules/mtphoto.md`，将配置规范改为 `MTPHOTO_BASE_URL + MTPHOTO_API_KEY`，并标记旧登录配置废弃，依赖任务1.2、2.4、3.3
- [√] 6.2 更新 `helloagents/wiki/api.md` 中 mtPhoto 上游说明，说明本地 API 兼容但上游认证已改为 API Key，依赖任务6.1
- [√] 6.3 如配置约定变化影响项目技术约定，更新 `helloagents/project.md` 的配置来源说明，依赖任务6.1
- [√] 6.4 更新 `helloagents/CHANGELOG.md`，记录 mtPhoto 上游接入方式迁移，依赖任务6.1

## 7. 测试
- [√] 7.1 REFACTOR: 在测试通过前提下整理 mtPhoto 客户端认证辅助函数命名，避免保留误导性的 `ensureLogin`、`token` 等旧概念，依赖任务2.4、3.4
- [√] 7.2 VERIFY: 运行 `go test ./internal/config ./internal/app -run 'MtPhoto|VideoExtract|Config'`，记录结果，依赖任务7.1
- [√] 7.3 VERIFY: 运行 `go test ./...`，记录结果，依赖任务7.2
- [√] 7.4 VERIFY: 运行 `cd frontend && npm run build`，确认本地 API 兼容未破坏前端构建，依赖任务4.3
- [√] 7.5 VERIFY: 运行 `git diff --check`，确认无格式问题，依赖任务6.4
