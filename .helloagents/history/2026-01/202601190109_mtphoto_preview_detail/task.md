# 任务清单: mtPhoto 预览支持查看图片详情

目录: `helloagents/plan/202601190109_mtphoto_preview_detail/`

---

## 1. 前端 - 预览详情入口
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 中调整 `hasMediaDetails` 判定逻辑，使 mtPhoto 预览在存在 `md5` 等详情字段时展示“查看详细信息”按钮，验证 why.md#需求-mtphoto-图片预览支持查看详情#场景-在-mtphoto-相册预览中查看详情

## 2. 安全检查
- [√] 2.1 执行安全检查（按G9：不新增后端透传能力、不展示敏感信息、仅基于既有元信息字段展示详情入口）。

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/mtphoto.md` 与 `helloagents/wiki/modules/media.md`，补充 mtPhoto 预览“查看详情”入口说明。
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录本次变更。

## 4. 测试
- [√] 4.1 执行 `cd frontend && npm run build`
- [?] 4.2 手工验证：mtPhoto 相册图片预览中可打开详情面板
  > 备注: 需要在浏览器中人工验证（相册预览点击信息按钮）

## 5. 方案包归档（强制）
- [√] 5.1 开发实施完成后：更新本 `task.md` 状态并迁移方案包至 `helloagents/history/2026-01/202601190109_mtphoto_preview_detail/`
- [√] 5.2 更新 `helloagents/history/index.md` 增加本次变更索引
