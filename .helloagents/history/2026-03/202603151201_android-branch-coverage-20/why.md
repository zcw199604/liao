# 变更提案: Android branch 覆盖率提升到 20%

## 需求背景
- 2026-03-15 已通过 Android Debug Unit Test 将 JaCoCo `branch coverage` 提升到 `10.15%`，建立了首轮 JVM 单测回归基线。
- 当前覆盖率热点仍集中在 `videoextract`、`douyin`、`mtphoto` 等模块的纯 helper / Repository 分支，具备继续通过 JVM 单测快速补齐的条件。
- 继续提升到 20% 可以让 Android 端在复杂媒体、抽帧、抖音与 mtPhoto 逻辑上拥有更可靠的回归保护。

## 变更内容
1. 继续补充 `videoextract`、`douyin`、`mtphoto` 的高收益 helper 单测。
2. 为 `MtPhotoFolderFavoritesRepository` 与 `VideoExtractTaskCenterRepository` 补齐成功 / 失败 / fallback / 缓存更新分支测试。
3. 保持 JaCoCo 统计口径稳定，通过 fresh `clean testDebugUnitTest jacocoDebugUnitTestReport` 验证是否达到 `branch >= 20%`。

## 影响范围
- **模块:** Android client / testing / media / videoextract / douyin / mtphoto
- **文件:** `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/{videoextract,douyin,mtphoto}/**`、对应 `src/test/**`
- **API:** 无外部接口协议变更
- **数据:** 无持久化 schema 变更

## 核心场景
### 需求: Android branch 覆盖率继续提升
**模块:** android-client
围绕目前仍为 0% 或低覆盖的高分支模块，优先补充纯 Kotlin helper 与 Repository 单测。

#### 场景: helper 分支回归稳定
在不依赖真机 / Compose UI 的前提下，通过 JVM 单测覆盖枚举映射、JSON 解析、文案构造、路径拼接与业务参数归一化分支。
- 预期结果: `videoextract`、`douyin`、`mtphoto` 的 helper 分支被稳定回归。

#### 场景: Repository 远端/缓存/fallback 分支回归稳定
通过 MockK/fake 驱动仓储层，覆盖成功、失败、缓存回退、缓存更新与默认值兜底逻辑。
- 预期结果: `MtPhotoFolderFavoritesRepository` 与 `VideoExtractTaskCenterRepository` 的主要业务分支被稳定回归。

#### 场景: 覆盖率指标可复现
在 clean 环境下重新运行 Debug Unit Test 与 JaCoCo 报告，确认 branch 覆盖率达到 20% 或明确剩余缺口。
- 预期结果: 覆盖率结果不依赖历史 `.exec/.ec`，且报告可复现。

## 风险评估
- **风险:** 某些 helper 仍为 `private`，测试可达性不足。
- **缓解:** 仅对少量纯 helper 做最小 `internal` 暴露，不改变运行逻辑。
- **风险:** 覆盖率增量低于预期，20% 目标可能一次未达成。
- **缓解:** 优先选择分支密度最高的 helper / Repository；如仍不足，基于 fresh 报告继续补下一批热点模块。
