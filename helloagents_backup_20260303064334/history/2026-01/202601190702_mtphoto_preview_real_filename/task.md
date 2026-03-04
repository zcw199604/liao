# 任务清单: mtPhoto 预览详情展示真实文件名

目录: `helloagents/plan/202601190702_mtphoto_preview_real_filename/`

---

## 1. 前端 - 预览详情文件名补齐
- [√] 1.1 在 `frontend/src/components/media/MediaPreview.vue` 增加可选 resolver 并在打开详情面板前按需补齐 `originalFilename`，验证 why.md#需求-mtphoto-预览详情展示真实文件名#场景-在-mtphoto-相册预览中查看真实文件名
- [√] 1.2 在 `frontend/src/components/media/MtPhotoAlbumModal.vue` 传入 mtPhoto 文件名 resolver（`md5` 缓存 + basename 提取），验证 why.md#需求-mtphoto-预览详情展示真实文件名#场景-在-mtphoto-相册预览中查看真实文件名

## 2. 安全检查
- [√] 2.1 执行安全检查（按G9：不在 UI 展示 `filePath` 全路径，仅展示 basename；不新增未授权接口）

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/mtphoto.md` 与 `helloagents/wiki/modules/media.md`，补充“查看详情展示真实文件名（按需解析）”说明
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录本次修复

## 4. 测试
- [√] 4.1 执行 `cd frontend && npm run build`
- [?] 4.2 手工验证：mtPhoto 相册图片预览中打开详情面板可展示真实文件名
  > 备注: 需要在浏览器中人工验证（相册预览点击信息按钮）
