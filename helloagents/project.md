# 项目技术约定

---

## 技术栈
- **后端:** Go 1.22（HTTP: chi；WebSocket: gorilla/websocket；MySQL: go-sql-driver/mysql；Redis 可选: go-redis）
- **前端:** Vue 3 / Vite / TypeScript
- **缓存:** Redis（可选）/ 内存（默认）

---

## 开发约定
- **Go:** 以 `cmd/liao` 为入口，业务代码在 `internal/app`；保持接口/返回体与原 Spring Boot 行为一致
- **接口路径:** 使用 camelCase（项目约定）
- **配置:** 以 `src/main/resources/application.yml` 为默认，优先通过环境变量覆盖

---

## 错误与日志
- **策略:** 接口保持与上游兼容，增强失败时降级返回上游原始响应
- **日志:** 关键路径使用 INFO；异常使用 ERROR 并带堆栈

---

## 测试与流程
- **测试:** `go test ./...`
- **前端单元测试:** `cd frontend && npm test`（如有）
- **提交:** Conventional Commits：`feat:` / `fix:` / `refactor:` / `chore:`
