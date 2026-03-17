# 实施思路

1. 为 `VideoExtractCreateRepository` 新增 JVM 测试，覆盖 `queryFileMeta`/`buildMultipartFilePart` 间接路径、上传返回值兜底、probe 与 createTask 的 payload 组装及错误分支。
2. 为 `VideoExtractCreateViewModel` 新增 JVM 测试，覆盖无 source 守卫、上传后自动探测、探测失败回写、创建成功/失败与消息消费分支。
3. 复用 `MainDispatcherRule`、mockk、`ContentResolver`/`Cursor` mock 与 `ApiEnvelope<JsonElement>` 假数据，不改动生产代码。
4. 复跑 Android 单元测试与 JaCoCo，更新 README、模块知识库、CHANGELOG 与 history 索引。
5. 完成后迁移方案包到 `history/`。
