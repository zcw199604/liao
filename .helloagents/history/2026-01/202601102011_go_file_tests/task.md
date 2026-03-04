# 任务清单: Go 文件功能测试补齐

目录: `helloagents/plan/202601102011_go_file_tests/`

---

## 1. 可测试性最小改动
- [√] 1.1 为 `detectAvailablePort` 增加可注入入口，确保默认行为不变且测试可替换

## 2. FileStorage（本地文件存储）
- [√] 2.1 为 `FileStorageService` 增加单元测试：媒体类型校验、扩展名解析、MD5 计算
- [√] 2.2 增加落盘/读取/删除测试：`SaveFile`→`ReadLocalFile`→`DeleteFile`
- [√] 2.3 增加 `FindLocalPathByMD5` 测试（sqlmock + 文件存在性校验）

## 3. ImageCache（图片缓存）
- [√] 3.1 为 `ImageCacheService` 增加测试：写入/读取/过期清理/重建/清空

## 4. ImageHash（媒体查重）
- [√] 4.1 为 `ImageHashService.CalculatePHash` 增加测试：可解码图片/不可解码文件
- [√] 4.2 为阈值解析与距离换算增加测试：`resolvePHashThreshold/similarityThresholdToDistance`

## 5. MediaUpload（路径与删除/重传）
- [√] 5.1 为 `normalizeUploadLocalPathInput/convertToLocalURL/ConvertPathsToLocalURLs` 增加单元测试
- [√] 5.2 为 `DeleteMediaByPath` 增加测试：DB 记录删除 + 物理文件删除（sqlmock + temp dir）

## 6. Handler 级测试（文件功能接口）
- [√] 6.1 为 `/api/checkDuplicateMedia` 增加 handler 测试：MD5 命中 / pHash 不支持 / pHash 相似命中
- [√] 6.2 为 `/api/uploadMedia` 增加 handler 测试：本地落盘 + 上游模拟 + 返回增强 + 写缓存
- [√] 6.3 为 `/upload/**` 静态文件访问增加 file server 测试（含路径穿越防护）

## 7. 质量验证与知识库同步
- [√] 7.1 运行 `go test ./...` 并修复阻断性失败
- [√] 7.2 更新知识库 `helloagents/wiki/modules/media.md`（补充测试说明并刷新最后更新）
- [√] 7.3 更新 `helloagents/CHANGELOG.md` 记录本次新增测试
- [√] 7.4 迁移方案包至 `helloagents/history/2026-01/202601102011_go_file_tests/` 并更新 `helloagents/history/index.md`
