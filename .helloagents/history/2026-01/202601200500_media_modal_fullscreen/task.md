# 任务清单: 图片弹窗全屏模式（全站图片库/mtPhoto 相册）

目录: `helloagents/plan/202601200500_media_modal_fullscreen/`

---

## 1. 前端功能实现（Media / mtPhoto）
- [√] 1.1 在 `frontend/src/components/media/AllUploadImageModal.vue` 中增加“全屏/退出全屏”模式（按钮 + 容器尺寸切换 + 快捷键 + localStorage 持久化），验证 why.md#需求-弹窗全屏模式-场景-开启全屏 与 why.md#需求-弹窗全屏模式-场景-快捷键控制
- [√] 1.2 在 `frontend/src/components/media/MtPhotoAlbumModal.vue` 中增加“全屏/退出全屏”模式（按钮 + 容器尺寸切换 + 快捷键 + localStorage 持久化），验证 why.md#需求-弹窗全屏模式-场景-开启全屏 与 why.md#需求-弹窗全屏模式-场景-快捷键控制
- [√] 1.3 抽取 `frontend/src/composables/useModalFullscreen.ts` 以复用全屏状态与键盘逻辑，并确保两处弹窗行为一致，验证 why.md#需求-弹窗全屏模式-场景-状态持久化

## 2. 质量验证
- [√] 2.1 执行 `npm -C frontend run build`，确保类型检查与构建通过
- [?] 2.2 手工验证：两处弹窗在普通/全屏模式下滚动加载正常、瀑布流/网格切换正常、关闭行为（按钮/遮罩/快捷键）符合预期
  > 备注: 需要在浏览器中手工验收（本次仅完成构建验证）
- [√] 2.3 使用本地 `gemini` 对变更做审查（基于 diff），记录需修正点并处理

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9：无敏感信息引入、无危险命令/路径处理变更、无权限逻辑变更）

## 4. 文档更新
- [√] 4.1 更新 `helloagents/wiki/modules/media.md`（补充“已上传图片浏览”新增全屏模式与持久化说明）
- [√] 4.2 更新 `helloagents/wiki/modules/mtphoto.md`（补充“相册弹窗”新增全屏模式与持久化说明）
- [√] 4.3 更新 `helloagents/CHANGELOG.md`
