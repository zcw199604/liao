# 任务清单: 标记 Java 版本为弃用（仅参考）

目录: `helloagents/plan/202601102213_java_backend_deprecated/`

---

## 1. 目录标记
- [√] 1.1 新增 `src/README.md`，声明 Java 版本已弃用仅供参考，并指向 Go 版本入口，验证 why.md#需求-java-版本弃用标记-场景-开发者查看仓库结构

## 2. 文档同步
- [√] 2.1 更新 `CLAUDE.md`：将“后端=Spring Boot/Java”为主的描述调整为“后端=Go”为主，Java 仅参考，验证 why.md#需求-java-版本弃用标记-场景-开发者查看仓库结构
- [√] 2.2 更新 `helloagents/wiki/overview.md` 与 `helloagents/wiki/arch.md`：明确 Go 为当前后端实现，Java 为弃用参考，验证 why.md#需求-java-版本弃用标记-场景-开发者查看仓库结构

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 确认未引入敏感信息、未修改权限/鉴权逻辑、无EHRB风险）

## 4. 测试
- [√] 4.1 执行 `go test ./...`
