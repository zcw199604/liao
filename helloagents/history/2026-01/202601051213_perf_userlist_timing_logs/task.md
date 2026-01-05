# 任务清单: 用户列表/收藏列表分段耗时日志

目录: `helloagents/plan/202601051213_perf_userlist_timing_logs/`

---

## 1. 后端接口耗时日志
- [√] 1.1 在 `src/main/java/com/zcw/controller/UserHistoryController.java` 为 `/api/getHistoryUserList` 增加分段耗时日志（upstream/enrichUserInfo/lastMsg/total）
- [√] 1.2 在 `src/main/java/com/zcw/controller/UserHistoryController.java` 为 `/api/getFavoriteUserList` 增加分段耗时日志（upstream/enrichUserInfo/lastMsg/total）

## 2. 文档更新
- [√] 2.1 更新 `helloagents/wiki/modules/user-history.md` 记录新增耗时日志说明
- [√] 2.2 更新 `helloagents/CHANGELOG.md` 记录本次变更

## 3. 测试
- [X] 3.1 运行 `mvn test` 验证编译与测试通过
  > 备注: 当前环境 JAVA=1.8.0_101，无法编译（`--release 17` 无效）；请切换到 JDK 17 后重新执行。
