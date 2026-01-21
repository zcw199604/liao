# 任务清单: 同步文档（移除 Java 代码后）

> 类型: 轻量迭代  
> 状态: ✅已完成

目录: `helloagents/history/2026-01/202601211358_sync_docs_remove_java/`

---

## 1. 文档同步
- [√] 1.1 清理仓库文档中对 Java/Spring Boot 与 `src/main/java` 的引用（README、AGENTS、docs、frontend 文档）
- [√] 1.2 同步知识库：更新 `helloagents/wiki/*.md` 中的后端实现引用为 Go，并移除不存在的 Java 路径
- [√] 1.3 修正文档中 WebSocket/Forceout 行为描述以对齐当前 Go 实现（forceout=5分钟、上游延迟关闭=80秒）

## 2. 知识库更新
- [√] 2.1 更新 `helloagents/CHANGELOG.md` 记录本次文档同步
- [√] 2.2 迁移方案包到 `helloagents/history/` 并更新 `helloagents/history/index.md`

## 3. 验证
- [√] 3.1 `rg` 检查：非 `helloagents/history/**` 与 `helloagents/plan/**` 目录不再出现 `src/main/java` / `mvn` / `pom.xml` 等已删除代码相关引用
