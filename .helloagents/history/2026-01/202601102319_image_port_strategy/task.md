# 任务清单: 全局图片端口策略配置（DB + Settings 面板）

目录: `helloagents/plan/202601102319_image_port_strategy/`

---

## 1. 后端（Go）
- [√] 1.1 在 `internal/app/schema.go` 新增 `system_config` 建表语句，确保启动自动建表
- [√] 1.2 新增 `SystemConfigService`（DB 读写 + 默认值 + 校验），并在 `internal/app/app.go` 注入到 `App`
- [√] 1.3 新增图片端口解析器（支持 `fixed/probe/real` + 缓存 + 最小字节阈值），并提供 `resolveImagePort` 方法
- [√] 1.4 新增 API：
  - [√] 1.4.1 `GET /api/getSystemConfig`
  - [√] 1.4.2 `POST /api/updateSystemConfig`
  - [√] 1.4.3 `POST /api/resolveImagePort`
- [√] 1.5 将上传增强返回的 `port` 由“探测端口”改为“按系统配置策略解析的端口”

## 2. 前端（Vue）
- [√] 2.1 在 `frontend/src/api/system.ts` 增加系统配置与端口解析 API 调用
- [√] 2.2 新增 Pinia store：`frontend/src/stores/systemConfig.ts`（加载/保存配置 + resolveImagePort 缓存）
- [√] 2.3 Settings 面板新增可视化切换与保存：
  - [√] 2.3.1 更新 `frontend/src/components/settings/SystemSettings.vue` 增加“图片端口策略”区块
- [√] 2.4 替换硬编码端口入口（图片按策略，视频保持固定）：
  - [√] 2.4.1 `frontend/src/composables/useWebSocket.ts`（收到 WS 媒体消息）
  - [√] 2.4.2 `frontend/src/stores/message.ts`（loadHistory 历史消息解析）
  - [√] 2.4.3 `frontend/src/stores/media.ts`（loadCachedImages 本地缓存映射）
  - [√] 2.4.4 `frontend/src/composables/useUpload.ts`（上传成功 URL 拼接优先使用后端返回 port）
  - [√] 2.4.5 `frontend/src/views/ChatRoomView.vue`（重传历史图片 URL 拼接）

## 3. 安全检查
- [√] 3.1 检查新增 API 的输入校验、鉴权范围、外部请求超时与读取上限（按G9）

## 4. 文档更新（知识库）
- [√] 4.1 更新 `helloagents/wiki/api.md`（补充系统配置与端口解析 API）
- [√] 4.2 更新 `helloagents/wiki/data.md`（补充 `system_config` 表）
- [√] 4.3 更新相关模块文档：
  - [√] 4.3.1 `helloagents/wiki/modules/chat-ui.md`
  - [√] 4.3.2 `helloagents/wiki/modules/websocket-proxy.md`
  - [√] 4.3.3 `helloagents/wiki/modules/media.md`
- [√] 4.4 更新 `helloagents/CHANGELOG.md`

## 5. 测试
- [√] 5.1 运行后端测试：`go test ./...`
- [√] 5.2 运行前端测试：`cd frontend && npm test`
- [√] 5.3 运行前端构建校验：`cd frontend && npm run build`

## 6. 收尾
- [√] 6.1 迁移方案包至 `helloagents/history/2026-01/202601102319_image_port_strategy/` 并更新 `helloagents/history/index.md`
- [√] 6.2 执行 `git status` 自检并生成 commit（中文 Conventional Commit）
