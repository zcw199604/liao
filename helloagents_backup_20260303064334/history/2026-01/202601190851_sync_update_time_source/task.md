# 轻量迭代任务清单：sync_update_time_source

目标：统一 `update_time` 的写入时间源，避免应用侧 `now` 与数据库 `CURRENT_TIMESTAMP` 混用导致“全站图片库”排序偶发异常。

## 任务

- [√] Go 后端：将 `media_file.update_time` 的更新统一改为应用侧 `now` 写入（reupload/MD5 去重命中等路径）。
- [√] Java 后端：将 `MediaFileRepository.updateTimeByLocalPathIgnoreUser` 改为显式传入 `LocalDateTime.now()`，避免依赖数据库时间函数。
- [√] 测试：更新 Go 单测断言并通过 `go test ./...`。
- [√] 知识库：同步更新数据模型说明与变更日志。

## 验证

- [√] `go test ./...`
- [?] 手动验证：点击“全站图片库”里某张图片的“上传/重新上传”，确认该图片在列表中会稳定上浮到最新位置（不受 DB 时区/时钟影响）。
  > 备注: 需在实际环境验证。

