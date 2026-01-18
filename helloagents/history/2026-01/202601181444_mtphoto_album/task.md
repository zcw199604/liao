# 任务清单: mtPhoto 相册接入与导入上传

目录: `helloagents/plan/202601181444_mtphoto_album/`

---

## 1. 后端：mtPhoto 对接与导入上传
- [√] 1.1 在 `internal/config/config.go` 中新增 mtPhoto 配置项（BaseURL/登录凭证），验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入
- [√] 1.2 新增 `internal/app/mtphoto_client.go` 实现 mtPhoto Client（login/get albums/get album files/filesInMD5/gateway 代理），验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入
- [√] 1.3 新增 `internal/app/mtphoto_auth.go`（或同文件）实现认证管理器：自动登录、过期/401 重新登录并单次重试，验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入
- [√] 1.4 新增 `internal/app/mtphoto_handlers.go` 并在 `internal/app/router.go` 注册接口：`getMtPhotoAlbums/getMtPhotoAlbumFiles/getMtPhotoThumb/resolveMtPhotoFilePath/importMtPhotoMedia`，验证 why.md#需求-展示相册内媒体图片视频-场景-无限滚动加载更多
- [√] 1.5 在 `internal/app/file_storage.go` 中补充“从本地绝对路径导入保存到 upload/”的能力（或新增专用函数/文件），验证 why.md#需求-导入并上传到上游-场景-在预览中点击上传
- [√] 1.6 在 `internal/app/static.go` 中加固 `/lsp/*` 静态文件服务的路径校验（拒绝 path traversal），验证 why.md#需求-预览媒体-场景-点击缩略图打开预览

## 2. 前端：相册 Modal 与入口集成
- [√] 2.1 新增 `frontend/src/api/mtphoto.ts` 封装 mtPhoto 相关 API 调用，验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入
- [√] 2.2 新增 `frontend/src/stores/mtphoto.ts` 管理弹窗状态、相册/媒体列表、分页与加载状态，验证 why.md#需求-展示相册内媒体图片视频-场景-无限滚动加载更多
- [√] 2.3 新增 `frontend/src/components/media/MtPhotoAlbumModal.vue`：相册列表→相册媒体网格（无限滚动 + 懒加载）→ 预览→导入上传，验证 why.md#需求-导入并上传到上游-场景-在预览中点击上传
- [√] 2.4 在 `frontend/src/components/chat/UploadMenu.vue` 增加 “mtPhoto 相册” 按钮并补齐事件，验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入
- [√] 2.5 在 `frontend/src/views/ChatRoomView.vue` 接入打开 mtPhoto 相册逻辑（与现有全站图片库类似），验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入
- [√] 2.6 在 `frontend/src/components/settings/SettingsDrawer.vue` 增加 “mtPhoto 相册” 入口（系统设置），验证 why.md#需求-展示-mtphoto-相册列表-场景-从系统设置进入
- [√] 2.7 在 `frontend/src/App.vue` 挂载 `MtPhotoAlbumModal`（与 `AllUploadImageModal` 同级），验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入

## 3. 安全检查
- [√] 3.1 执行安全检查（按G9: 输入验证、敏感信息处理、权限控制、路径遍历风险规避），验证 why.md#风险评估

## 4. 文档更新（知识库）
- [√] 4.1 更新 `helloagents/wiki/api.md`：补充 mtPhoto 相关 API 与环境变量说明，验证 why.md#变更内容
- [√] 4.2 按需要更新 `helloagents/wiki/arch.md` 或新增模块文档（记录 mtPhoto 接入点与数据流），验证 why.md#影响范围
- [√] 4.3 更新 `helloagents/CHANGELOG.md` 记录新增功能，验证 why.md#变更内容

## 5. 测试
- [√] 5.1 在 `internal/app/mtphoto_client_test.go`（或同目录新文件）补充 mtPhoto client/认证管理器单测（使用 `httptest` 模拟 mtPhoto），验证 why.md#需求-展示-mtphoto-相册列表-场景-从上传菜单进入
- [√] 5.2 在 `internal/app/mtphoto_import_test.go` 增加导入上传路径校验与本地文件读取的单测（使用临时文件），验证 why.md#需求-导入并上传到上游-场景-在预览中点击上传
- [√] 5.3 前端执行 `npm run build` 验证编译通过，验证 why.md#变更内容
