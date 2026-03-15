# 轻量迭代任务 - Android 覆盖率任务接入

- [√] 1. 评估 Android Gradle 当前覆盖率接入点与任务命名
- [√] 2. 正式接入 Android JaCoCo 覆盖率任务，支持输出 XML/HTML 报告
- [√] 3. 更新 Android README / 知识库 / 变更记录，补充覆盖率命令与报告位置
- [√] 4. 执行覆盖率任务验证并记录当前覆盖率结果
  - 验证: `cd android-app && ./gradlew jacocoDebugUnitTestReport --no-daemon` 通过；line 5.02% / branch 2.69% / method 6.82% / class 5.12%
