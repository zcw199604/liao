# 任务清单: mtPhoto 相册列表新增收藏夹入口

目录: `helloagents/plan/202601190552_mtphoto_favorites_album/`

---

## 1. 前端：相册列表与收藏夹入口
- [√] 1.1 在 `frontend/src/stores/mtphoto.ts` 中为相册模型增加“本地唯一ID + mtPhotoAlbumId 映射”，并在 `loadAlbums()` 置顶注入“收藏夹”（封面为空），验证 why.md#需求-相册列表显示收藏夹入口-场景-打开-mtphoto-相册弹窗
- [√] 1.2 在 `frontend/src/stores/mtphoto.ts` 中调整 `openAlbum()/loadAlbumPage()` 使用 `mtPhotoAlbumId` 拉取媒体，并在加载后同步 `selectedAlbum.count`，验证 why.md#需求-浏览收藏夹媒体-场景-点击收藏夹并滚动加载

## 2. 安全检查
- [√] 2.1 执行安全检查（输入验证、敏感信息处理、权限控制、开放代理风险），确认本次变更未引入新的后端放行与不安全外链

## 3. 知识库更新
- [√] 3.1 更新 `helloagents/wiki/modules/mtphoto.md`，补充“收藏夹入口置顶、封面为空、复用 filesV2/1”说明
- [√] 3.2 更新 `helloagents/CHANGELOG.md` 记录本次变更

## 4. 测试
- [√] 4.1 执行 `npm run build`，确保前端类型检查与构建通过
- [?] 4.2 手动验证：打开 mtPhoto 相册弹窗 → 列表首项收藏夹 → 进入后加载/预览/导入上传流程正常
  > 备注: 需要在浏览器中手动验证交互与上游数据
