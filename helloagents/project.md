# 项目技术约定

---

## 技术栈
- **后端:** Go 1.25.6，`chi` HTTP 路由，`gorilla/websocket` WebSocket，`database/sql` + MySQL/PostgreSQL 方言层，`golang-jwt/jwt` JWT。
- **前端:** Vue 3.5，Vite 7，TypeScript 5.9，Pinia，Vue Router，Axios，Tailwind CSS，Vitest。
- **Android:** Kotlin，Jetpack Compose，Hilt，Retrofit/OkHttp，Room，DataStore，WorkManager。
- **数据:** MySQL 或 PostgreSQL 由 `DB_URL` scheme 选择；Redis 可选，默认内存缓存；上传媒体落在 `./upload`。
- **构建:** 前端生产产物由 Vite 输出，Go 服务托管静态资源；Docker 使用 Node 20 + Go 1.25.6 多阶段构建。

---

## 开发约定
- **后端入口:** `cmd/liao`，业务实现位于 `internal/app`，配置位于 `internal/config`，数据库工具位于 `internal/database`。
- **接口命名:** REST 路径保留项目历史 camelCase 约定，例如 `/api/getHistoryUserList`、`/api/uploadMedia`。
- **前端组织:** Vue 组件使用 `PascalCase.vue`；组合式函数使用 `useXxx.ts`；Pinia stores 位于 `frontend/src/stores/`。
- **Android 组织:** 客户端位于 `android-app/`，按 `core/*`、`feature/*`、`app/*` 分层。
- **配置来源:** Go 服务运行时读取环境变量，不读取 `src/main/resources/application.yml`；该 YAML 仅保留变量名和默认值对照。
- **生成产物:** 不提交 `node_modules/`、`target/`、`src/main/resources/static/`、`upload/`、覆盖率产物和本地日志。

---

## 安全与日志
- **凭据:** 生产环境必须通过环境变量提供 `DB_URL`、`DB_USERNAME`、`DB_PASSWORD`、`AUTH_ACCESS_CODE`、`JWT_SECRET`；禁止提交真实凭据。
- **鉴权:** HTTP API 以 Bearer JWT 为主；`/ws` 握手通过 query token 校验。
- **媒体代理:** 公开预览类接口只能暴露必要资源；涉及随机 key 的下载接口依赖短期缓存和鉴权入口生成。
- **日志:** 后端使用 `slog`，`LOG_LEVEL` 支持 `debug/info/warn/error`，`LOG_FORMAT` 支持 `json/text`。

---

## 测试与流程
- **后端:** `go test ./...`
- **前端:** `cd frontend && npm test`；构建门禁为 `cd frontend && npm run build`
- **Android:** `cd android-app && .\gradlew testDebugUnitTest`
- **提交:** 遵循 Conventional Commits，例如 `feat:`、`fix:`、`refactor:`、`chore:`。
