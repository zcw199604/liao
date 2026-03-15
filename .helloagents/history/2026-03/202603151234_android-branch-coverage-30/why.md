# 变更提案: Android branch 覆盖率提升到 30%

## 需求背景
- 2026-03-15 已将 Android Debug Unit Test `branch coverage` 提升到 `21.63%`，完成第二轮稳定回归基线建设。
- 当前剩余大量未覆盖分支主要集中在 `chatroom`、`douyin`、`mtphoto` 等模块的 Repository 与纯 helper；这些逻辑仍适合继续通过 JVM 单测推进。
- 将 branch 覆盖率进一步推到 30% 可以显著增强聊天、媒体、抖音与图片浏览等核心 Android 能力的回归稳定性。

## 变更内容
1. 迁移遗留方案包后，继续补 `chatroom`、`douyin`、`mtphoto` 的高收益 helper 与 Repository 单测。
2. 重点覆盖 `ChatRoomRepository`、`ChatRoomFeature` 纯 helper，以及 `DouyinRepository` / `MtPhotoRepository` 中剩余高分支映射与仓储分支。
3. 通过 fresh `clean testDebugUnitTest jacocoDebugUnitTestReport` 验证是否达到 `branch >= 30%`。

## 影响范围
- **模块:** Android client / testing / chatroom / douyin / mtphoto
- **文件:** `android-app/app/src/main/kotlin/io/github/a7413498/liao/android/feature/{chatroom,douyin,mtphoto}/**`、对应 `src/test/**`
- **API:** 无外部接口协议变更
- **数据:** 无持久化 schema 变更

## 核心场景
### 需求: Android branch 覆盖率继续提升到 30%
**模块:** android-client
围绕聊天、抖音、mtPhoto 的未覆盖高分支逻辑继续补齐 JVM 单测。

#### 场景: 聊天核心分支具备 JVM 回归
在不依赖真机与 Compose UI 的前提下，覆盖聊天页 Repository 的连接、收藏、历史、媒体 URL、重传与消息合并逻辑。
- 预期结果: `chatroom` 的关键业务分支具备稳定回归。

#### 场景: 抖音/mtPhoto 仓储映射分支具备 JVM 回归
通过 helper/Repository 单测覆盖 URL 归一化、媒体映射、收藏/标签/分页等剩余高分支逻辑。
- 预期结果: `douyin` 与 `mtphoto` 的关键映射与仓储分支被稳定回归。

#### 场景: 覆盖率结果可复现且达到 30%
在 clean 环境下重新运行 Debug Unit Test 与 JaCoCo 报告，验证 branch 覆盖率达到 30% 或明确剩余缺口。
- 预期结果: 覆盖率结果不依赖历史产物，且报告可复现。

## 风险评估
- **风险:** `chatroom` 依赖较多（WebSocket/DAO/媒体接口），测试搭建复杂。
- **缓解:** 优先覆盖可 mock 的 public method 与纯 helper，必要时仅对少量 helper 做最小 `internal` 暴露。
- **风险:** `FeatureKt` 中 Compose UI 分支占比较高，纯 JVM 单测无法覆盖全部。
- **缓解:** 聚焦同文件中的非 Compose helper 与 Repository，避免引入高成本 UI 测试基建。
- **风险:** 30% 目标仍可能需要多轮推进。
- **缓解:** 每轮都基于 fresh JaCoCo 热点排序，优先实现单位工作量最高的分支增益。
