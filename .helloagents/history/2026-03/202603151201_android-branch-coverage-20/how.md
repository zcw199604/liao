# 技术设计: Android branch 覆盖率提升到 20%

## 技术方案
### 核心技术
- Kotlin / JUnit4 / kotlinx-coroutines-test / MockK / JaCoCo

### 实现要点
- 为 `VideoExtractCreateFeatureKt`、`VideoExtractTaskCenterFeatureKt`、`DouyinFeatureKt`、`MtPhotoFeatureKt` 的高分支 helper 建立纯 JVM 单测。
- 为 `MtPhotoFolderFavoritesRepository`、`VideoExtractTaskCenterRepository` 建立基于 MockK 的 Repository 单测，覆盖远端成功、失败、缓存 fallback、缓存更新与默认值分支。
- 必要时将少量纯 helper 从 `private` 调整为 `internal` 以支持测试访问，避免通过反射增加脆弱性。
- 继续通过 `clean testDebugUnitTest jacocoDebugUnitTestReport` 生成 fresh JaCoCo 报告，验证 branch 覆盖率是否达到 20%。

## 安全与性能
- **安全:** 仅增加本地 JVM 单测与少量 helper 可见性调整，不触碰生产密钥、网络凭据与生产环境资源。
- **性能:** 测试优先覆盖纯函数与 mock 仓储，不引入额外运行时依赖或真机测试开销。

## 测试与部署
- **测试:** `cd android-app && ./gradlew clean testDebugUnitTest jacocoDebugUnitTestReport --no-daemon`
- **部署:** 无部署动作，仅提交代码与知识库更新。
