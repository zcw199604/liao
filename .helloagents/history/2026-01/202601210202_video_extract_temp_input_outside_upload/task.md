# 任务清单: 临时输入视频迁出 upload 挂载目录

目录: `helloagents/plan/202601210202_video_extract_temp_input_outside_upload/`

---

## 1. 后端（落盘/解析/静态服务）
- [√] 1.1 在 `internal/app/file_storage.go` 将抽帧临时输入视频改为保存到系统临时目录（默认 `os.TempDir()/video_extract_inputs`），不写入 `./upload`
- [√] 1.2 在 `internal/app/video_extract.go` 扩展路径解析：`localPath=/tmp/video_extract_inputs/...` 映射到临时目录绝对路径
- [√] 1.3 在 `internal/app/static.go` 扩展 `/upload` 静态文件服务：`/upload/tmp/video_extract_inputs/*` 映射到系统临时目录
- [√] 1.4 在 `internal/app/video_extract_handlers.go` 调整清理接口：仅允许删除 `/tmp/video_extract_inputs/...` 对应的临时文件

## 2. 安全检查
- [√] 2.1 执行安全检查：清理/静态服务均限制在临时目录范围内，防路径越界与误删

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/api.md`、`helloagents/wiki/modules/media.md` 与 `helloagents/CHANGELOG.md`，同步“临时目录不在 upload 下”的行为说明

## 4. 测试
- [√] 4.1 更新/新增 Go 测试覆盖：临时落盘、路径解析、清理接口、静态文件服务映射

