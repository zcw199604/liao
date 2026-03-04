# 任务清单: Go 文件相关接口测试补齐（第二批）

目录: `helloagents/plan/202601102152_go_file_tests_more/`

---

## 1. Handler 级测试补齐（Media）
- [√] 1.1 为 `GET /api/getAllUploadImages` 增加测试：分页/port 字段/URL 生成
- [√] 1.2 为 `GET /api/getUserUploadHistory` 增加测试：list/total/totalPages 与 URL 生成
- [√] 1.3 为 `GET /api/getUserSentImages` 增加测试：sendTime/toUserId 与 URL 生成
- [√] 1.4 为 `GET /api/getUserUploadStats` 增加测试：totalCount
- [√] 1.5 为 `GET /api/getChatImages` 增加测试：字符串数组返回
- [√] 1.6 为 `POST /api/reuploadHistoryImage` 增加测试：成功写缓存/错误返回
- [√] 1.7 为 `POST /api/deleteMedia` 增加测试：成功/403/500 分支
- [√] 1.8 为 `POST /api/batchDeleteMedia` 增加测试：参数校验 + 部分失败统计
- [√] 1.9 为 `POST /api/recordImageSend` 增加测试：成功记录/未命中返回

## 2. Handler 级测试补齐（User History / Cache）
- [√] 2.1 为 `GET /api/getCachedImages` 增加测试：空缓存/有缓存（含 X-Forwarded-Host）

## 3. Handler 级测试补齐（Repair）
- [√] 3.1 为 `POST /api/repairMediaHistory` 增加测试：非法 JSON/未知字段/负数限制错误

## 4. 质量验证与知识库同步
- [√] 4.1 运行 `go test ./...`
- [√] 4.2 更新知识库 `helloagents/wiki/modules/media.md`（补充测试清单并刷新最后更新）
- [√] 4.3 更新 `helloagents/CHANGELOG.md`（补充本次测试覆盖范围）
- [√] 4.4 迁移方案包至 `helloagents/history/2026-01/202601102152_go_file_tests_more/` 并更新 `helloagents/history/index.md`
