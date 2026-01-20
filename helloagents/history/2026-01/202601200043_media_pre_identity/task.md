# 任务清单: 身份选择前使用图片功能

目录: `helloagents/history/2026-01/202601200043_media_pre_identity/`

---

## 1. 前置入口
- [√] 1.1 在 `frontend/src/views/IdentityPicker.vue` 增加“图片管理”入口（登录后、选择身份前可打开）

## 2. 交互与权限
- [√] 2.1 调整 `frontend/src/components/settings/SettingsDrawer.vue` 的 `media` 模式逻辑，允许未选择身份时打开并加载“所有上传图片/mtPhoto/图片查重”
- [√] 2.2 调整 `frontend/src/components/media/AllUploadImageModal.vue`：未选择身份时仍可浏览/加载更多/删除；重新上传按钮在未选择身份时不可用
- [√] 2.3 调整 `frontend/src/components/media/MtPhotoAlbumModal.vue`：未选择身份时可浏览/下载；导入上传需选择身份（toast 提示）

## 3. 验证
- [√] 3.1 运行前端测试：`npm -C frontend test`

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/media.md`（身份选择前入口与限制说明）
- [√] 4.2 更新 `helloagents/wiki/modules/mtphoto.md`（身份选择前入口与限制说明）
- [√] 4.3 更新 `helloagents/CHANGELOG.md`
- [√] 4.4 更新 `helloagents/history/index.md`
