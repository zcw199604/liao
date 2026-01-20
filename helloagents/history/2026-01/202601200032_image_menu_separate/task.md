# 任务清单: 图片管理菜单拆分

目录: `helloagents/history/2026-01/202601200032_image_menu_separate/`

---

## 1. 菜单结构调整
- [√] 1.1 前端新增顶部菜单“图片管理”，从 `frontend/src/components/chat/ChatSidebar.vue` 打开 `frontend/src/components/settings/SettingsDrawer.vue` 的 `media` 模式
- [√] 1.2 将“所有上传图片 / mtPhoto 相册 / 图片查重”入口迁移到 `frontend/src/components/settings/SettingsDrawer.vue` 的 `media` 模式，并从“系统设置”移除

## 2. 验证
- [√] 2.1 运行前端测试：`npm -C frontend test`

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/mtphoto.md` 的入口描述
- [√] 3.2 更新 `helloagents/wiki/modules/media.md` 的入口描述（含图片查重入口）
- [√] 3.3 更新 `helloagents/CHANGELOG.md`
- [√] 3.4 更新 `helloagents/history/index.md`
