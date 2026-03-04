# 任务清单: 修复 lastMsg 表情误识别为[文件]（轻量迭代）

目录: `helloagents/plan/202601120630_fix_lastmsg_emoji_preview/`

---

## 1. 后端逻辑
- [√] 1.1 Java：`formatLastMessage` 对无路径分隔符且无扩展名的 `[...]` 按文本返回（避免 `[doge]` → `[文件]`）
- [√] 1.2 Go：`formatLastMessage` 同步对齐上述规则

## 2. 测试
- [√] 2.1 新增 `MemoryUserInfoCacheServiceLastMessageTest` 用例覆盖 `[doge]` 场景
- [√] 2.2 新增 `RedisUserInfoCacheServiceLastMessageTest` 用例覆盖 `[doge]` 场景
- [-] 2.3 运行 `mvn test` / `go test ./...`（当前环境缺少 Maven/Go 工具链）

## 3. 文档
- [√] 3.1 更新 `helloagents/wiki/modules/user-history.md` 记录 lastMsg 格式化规则
- [√] 3.2 更新 `helloagents/CHANGELOG.md`
