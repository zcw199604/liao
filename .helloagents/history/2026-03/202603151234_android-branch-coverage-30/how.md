# 技术设计: Android branch 覆盖率提升到 30%

## 技术方案
### 核心技术
- Kotlin / JUnit4 / kotlinx-coroutines-test / MockK / JaCoCo

### 实现要点
- 对 `ChatRoomFeature.kt` 中的高分支纯 helper（消息合并、文案映射、状态映射、媒体标签）建立 JVM 单测。
- 对 `ChatRoomRepository` 建立基于 MockK 的单测，覆盖连接建立、历史回退、收藏切换、媒体 URL 解析、历史媒体加载、重传与发送分支。
- 对 `DouyinRepository` / `MtPhotoRepository` 中仍为高分支热点的映射 helper 与 public method 建立扩展单测；必要时将少量纯 helper 从 `private` 调整为 `internal`。
- 最后执行 `clean testDebugUnitTest jacocoDebugUnitTestReport` 生成 fresh 覆盖率报告，并回写 README / 知识库 / CHANGELOG / 历史索引。

## 安全与性能
- **安全:** 仅增加 JVM 单测与少量 helper 可见性调整，不接触生产数据、密钥或真实外部服务。
- **性能:** 继续以 mock/fake 为主，不引入真机或 Robolectric 基建，确保反馈速度可控。

## 测试与部署
- **测试:** `cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`
- **部署:** 无部署动作，仅提交代码与知识库更新。
