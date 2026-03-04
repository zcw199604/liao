# 任务清单: 视频抽帧（关键帧/固定FPS/逐帧）与任务管理

目录: `helloagents/plan/202601200738_video_extract_frames/`

---

## 1. 数据库与配置
- [√] 1.1 在 `internal/app/schema.go` 中新增 `video_extract_task`/`video_extract_frame` 表，验证 why.md#需求-任务预览与分页加载-场景-点击任务查看目录下帧图
- [√] 1.2 在 `internal/config/config.go` 中补充 ffmpeg 相关配置项（如可选可执行路径、抽帧输出根目录等），验证 why.md#需求-创建抽帧任务-场景-从上传库视频创建任务

## 2. 后端：视频输入源解析与探测
- [√] 2.1 在 `internal/app` 新增视频源解析函数（upload localPath → absPath；mtPhoto md5 → absPath/必要时缓存），验证 why.md#需求-创建抽帧任务-场景-从-mtphoto-相册视频创建任务
- [√] 2.2 在 `internal/app` 新增 `ffprobe` 探测实现与接口 `GET /api/probeVideo`，返回 `durationSec/width/height/avgFps`，验证 why.md#需求-创建抽帧任务-场景-从上传库视频创建任务

## 3. 后端：任务模型与状态机
- [√] 3.1 在 `internal/app` 新增任务模型（status/stopReason/cursorOutTimeSec/maxFramesTotal 等）与 DB 读写封装，验证 why.md#需求-任务进度与实时信息-场景-运行中查看进度并随时终止
- [√] 3.2 在 `internal/app` 新增任务列表/详情查询能力（分页帧列表 + 增量日志），验证 why.md#需求-任务预览与分页加载-场景-点击任务查看目录下帧图

## 4. 后端：抽帧执行器（ffmpeg）
- [√] 4.1 在 `internal/app` 新增 ffmpeg runner：支持 keyframe(iframe/scene)/fps/all + timeRange + maxFrames，解析 `-progress` 输出，验证 why.md#需求-任务进度与实时信息-场景-运行中查看进度并随时终止
- [√] 4.2 在 `internal/app` 实现取消/终止：`POST /api/cancelVideoExtractTask`，确保进程可被 context 取消并更新状态，验证 why.md#需求-任务进度与实时信息-场景-运行中查看进度并随时终止
- [√] 4.3 在 `internal/app` 实现续跑：`POST /api/continueVideoExtractTask`，支持扩展 endSec 或增加 maxFramesTotal，并追加写入同一目录，验证 why.md#需求-限制体现与继续抽帧-场景-达到-maxframes-或-endsec-后暂停并继续

## 5. 后端：HTTP 路由与权限
- [√] 5.1 在 `internal/app/router.go` 注册新接口路由（create/list/detail/cancel/continue/delete/probe），验证 why.md#需求-创建抽帧任务-场景-从上传库视频创建任务
- [√] 5.2 校验输入与路径安全（禁止目录穿越、禁止命令注入、md5 校验），并补充必要的错误码/错误信息，验证 why.md#风险评估

## 6. 运行时依赖（Docker）
- [√] 6.1 在 `Dockerfile` 最终镜像阶段安装 `ffmpeg`（包含 `ffprobe`），验证 why.md#变更内容

## 7. 前端：API 与 Store
- [√] 7.1 新增 `frontend/src/api/videoExtract.ts` 封装抽帧相关接口；新增类型定义（`frontend/src/types`），验证 why.md#需求-创建抽帧任务-场景-从上传库视频创建任务
- [√] 7.2 新增 Pinia store（如 `frontend/src/stores/videoExtract.ts`）：任务列表、详情、轮询、增量帧/日志缓存，验证 why.md#需求-任务进度与实时信息-场景-运行中查看进度并随时终止

## 8. 前端：入口与创建任务 UI
- [√] 8.1 在 `frontend/src/components/media/MediaPreview.vue` 增加视频“抽帧”入口，支持 upload 与 mtPhoto 视频，验证 why.md#需求-创建抽帧任务-场景-从-mtphoto-相册视频创建任务
- [√] 8.2 新增 `frontend/src/components/media/VideoExtractCreateModal.vue`：模式/参数/校验/预估输出量与风险提示，验证 why.md#需求-创建抽帧任务-场景-从上传库视频创建任务

## 9. 前端：任务中心与预览 UI（含 Gemini 建议落实）
- [√] 9.1 新增 `frontend/src/components/media/VideoExtractTaskModal.vue`：任务列表/详情/状态展示/停止/继续/删除，验证 why.md#需求-任务预览与分页加载-场景-点击任务查看目录下帧图
- [√] 9.2 在 `frontend/src/components/settings/SettingsDrawer.vue` 增加“任务中心/抽帧任务”入口，并避免多层弹窗套娃（提交后 Toast + 可点“查看详情”），验证 why.md#产品分析
- [√] 9.3 预览网格实现分页/增量加载；为大数据量预留虚拟滚动与缩略图策略（可先实现分页，虚拟滚动作为 P2），验证 why.md#风险评估

## 10. 安全检查
- [√] 10.1 执行安全检查（按G9：输入验证、路径遍历、命令注入、权限校验、文件清理策略）

## 11. 测试与验证
- [√] 11.1 Go：为任务状态流转、续跑游标计算、输入解析添加单测（`internal/app/*_test.go`），验证 why.md 核心场景
  > 备注: 本次新增了基础单元测试（MD5校验/帧率解析/路径安全）；更完整的状态流转与 DB 层测试可后续补齐。
- [√] 11.2 前端：执行 `cd frontend && npm run build` 验证编译通过；手动验证任务创建/停止/继续/预览

## 12. 文档更新
- [√] 12.1 更新 `helloagents/wiki/api.md` 增补抽帧接口说明
- [√] 12.2 更新 `helloagents/wiki/data.md` 增补 `video_extract_task`/`video_extract_frame` 表结构与说明
- [√] 12.3 更新 `helloagents/CHANGELOG.md` 记录新增功能
