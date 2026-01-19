# 任务清单: mtPhoto 下载中文文件名解码

目录: `helloagents/plan/202601190728_mtphoto_download_filename_cn/`

---

## 1. 前端：下载文件名解码
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 增强 `Content-Disposition` 解析：支持对 `filename=` 的 URL 编码结果执行 decode，避免中文文件名保存为 `%E4%B8%AD%E6%96%87.jpg`
- [√] 1.2 下载文件名在触发保存前执行 basename 级别净化（避免路径片段/查询参数污染下载名）

## 2. 测试
- [√] 2.1 在 `frontend/src/__tests__/settings-media-components.test.ts` 增加“filename 为 URL 编码中文”的用例，验证下载名会被解码为中文
- [√] 2.2 执行 `cd frontend && npm test`
- [?] 2.3 手动验证：在 mtPhoto 相册预览中点击“下载”，当原始文件名为中文时保存为中文文件名（非 `%XX` 编码）
  > 备注: 需要在浏览器中验证真实下载行为与不同浏览器兼容性

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/mtphoto.md` 补充下载文件名解码说明
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录本次修复

## 4. 方案包归档（强制）
- [√] 4.1 开发实施完成后：更新本 `task.md` 状态并迁移方案包至 `helloagents/history/2026-01/202601190728_mtphoto_download_filename_cn/`
- [√] 4.2 更新 `helloagents/history/index.md` 增加本次变更索引
