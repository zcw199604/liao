# 任务清单: mtPhoto 收藏夹数量展示（相册列表）

目录: `helloagents/plan/202601190613_mtphoto_favorites_count/`

---

## 1. 前端：收藏夹数量
- [√] 1.1 在 `frontend/src/stores/mtphoto.ts` 的 `loadAlbums()` 中补充拉取收藏夹数量（复用 `getMtPhotoAlbumFiles(albumId=1)` 读取 `total`），并更新相册列表中“收藏夹”的 `count` 展示

## 2. 安全检查
- [√] 2.1 执行安全检查：确认未新增后端放行/敏感信息输出/开放代理风险

## 3. 文档更新
- [√] 3.1 更新 `helloagents/wiki/modules/mtphoto.md`：补充收藏夹数量展示逻辑说明
- [√] 3.2 更新 `helloagents/CHANGELOG.md`：补充本次变更记录

## 4. 测试
- [√] 4.1 执行 `cd frontend && npm run build`
- [?] 4.2 手动验证：打开 mtPhoto 相册弹窗后，“收藏夹”卡片展示正确数量
  > 备注: 需要在浏览器中验证上游数据与UI展示
