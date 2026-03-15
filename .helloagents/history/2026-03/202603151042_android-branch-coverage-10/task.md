# 任务清单

- [-] 1. 调整 Android JaCoCo 任务排除规则，仅排除 Kotlin/Compose 合成 `*Kt$*` 类
- [-] 2. 为 videoextract helper 暴露最小可测接口并补充单元测试
- [-] 3. 为 douyin / mtphoto / settings helper 暴露最小可测接口并补充单元测试
- [-] 4. 执行 `cd android-app && ./gradlew :app:jacocoDebugUnitTestReport --no-daemon` 验证 branch 覆盖率是否达到目标
- [-] 5. 更新 Android README、知识库与变更记录，回写覆盖率口径与最新结果


## 归档说明
- 2026-03-15：该方案包为后续 `202603151043_android-branch-coverage-10` 的重复草稿，未单独执行，现按遗留方案迁移归档。
