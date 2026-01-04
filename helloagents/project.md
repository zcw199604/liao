# 项目技术约定

---

## 技术栈
- **后端:** Java 17 / Spring Boot 3
- **前端:** Vue 3 / Vite / TypeScript
- **缓存:** Redis（可选）/ 内存（默认）

---

## 开发约定
- **Java:** 4 空格缩进；包名 `com.zcw.*`
- **接口路径:** 使用 camelCase（项目约定）
- **配置:** 以 `src/main/resources/application.yml` 为默认，优先通过环境变量覆盖

---

## 错误与日志
- **策略:** 接口保持与上游兼容，增强失败时降级返回上游原始响应
- **日志:** 关键路径使用 INFO；异常使用 ERROR 并带堆栈

---

## 测试与流程
- **测试:** `mvn test`（需要 JDK 17）
- **前端单元测试:** `cd frontend && npm test`（Vitest / jsdom）
- **提交:** Conventional Commits：`feat:` / `fix:` / `refactor:` / `chore:`
