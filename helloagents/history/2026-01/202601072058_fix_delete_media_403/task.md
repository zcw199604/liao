# 任务清单: 修复 /api/deleteMedia 返回 403（localPath 兼容 + 全站删除）

目录: `helloagents/plan/202601072058_fix_delete_media_403/`

---

## 1. 删除逻辑兼容性
- [√] 1.1 兼容 `localPath` 可能仍包含 `%2F` 等编码的场景（兜底解码一次）
- [√] 1.2 查询/删除时兼容 `local_path` 可能含 `/upload` 前缀或缺少前导 `/` 的历史存储形式
- [√] 1.3 物理文件删除时统一使用归一化后的 `/images/...` 形式，避免出现 `upload/upload/...` 路径

## 2. 可观测性
- [√] 2.1 `/api/deleteMedia` 增加关键日志（请求参数/失败原因/删除结果）
- [√] 2.2 删除行为对齐“全站图片库”展示：不校验上传者归属（不按 userId 过滤）

## 3. 知识库同步
- [√] 3.1 更新 `helloagents/wiki/api.md` 的 deleteMedia 说明（不按 userId 校验归属）
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录本次修复

## 4. 验证
- [X] 4.1 运行 `go test ./...` 复验
  > 备注: 当前环境未安装/未配置 Go CLI（`go` 命令不可用），需在具备 Go 1.22+ 的环境中复验。
