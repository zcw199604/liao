# 任务清单: mtphoto-favorites-management-enhance

目录: `helloagents/plan/202602231046_mtphoto-favorites-management-enhance/`

---

## 任务状态符号说明

| 符号 | 状态 | 说明 |
|------|------|------|
| `[ ]` | pending | 待执行 |
| `[√]` | completed | 已完成 |
| `[X]` | failed | 执行失败 |
| `[-]` | skipped | 已跳过 |
| `[?]` | uncertain | 待确认 |

---

## 执行状态
```yaml
总任务: 31
已完成: 28
完成率: 90.3%
```

---

## 任务列表

### 0. 方案阶段

- [√] 0.1 完成 Gemini 审核并回填结论（会话：`ec377fd6-8ed0-4969-b9f2-71f04ca2311c`）
  - 验证: 方案包 `proposal.md` 已包含“必改项/可选优化/结论”

### 1. 前端：收藏筛选与直接编辑 UI

- [√] 1.1 在 `frontend/src/components/media/MtPhotoAlbumModal.vue` 收藏头部新增标签筛选输入框
  - 验证: 输入关键字后收藏列表可实时缩小

- [√] 1.2 在收藏头部新增筛选模式切换（ANY/ALL）
  - 验证: 同一关键字下两种模式结果可区分

- [√] 1.3 在收藏头部新增排序选择（updatedAt/name/tagCount）
  - 验证: 切换排序后结果顺序可预测

- [√] 1.4 收藏卡片新增“编辑”按钮与编辑态样式
  - 验证: 点击后可编辑标签与备注

- [√] 1.5 完成收藏卡片编辑保存/取消交互
  - 验证: 保存成功后卡片即时刷新，取消不落库

- [√] 1.6 移动端抽屉内接入筛选区
  - 验证: <1024px 下收藏筛选入口可见可用

- [√] 1.7 移动端编辑采用 fixed 弹层或 Teleport，避免挤压主图区
  - 验证: 开启编辑时图片网格高度不被压缩

### 2. 前端：store 状态与动作重构

- [√] 2.1 在 `frontend/src/stores/mtphoto.ts` 新增筛选状态（keyword/mode）
  - 验证: 状态变化可驱动列表变化

- [√] 2.2 新增排序状态（sortBy/sortOrder）
  - 验证: 排序切换后列表顺序符合预期

- [√] 2.3 新增编辑状态（editingFolderId/draftTags/draftNote）
  - 验证: 不同卡片编辑态互斥

- [√] 2.4 新增 `allUniqueTags` 聚合计算
  - 验证: 标签 chips 数据来自实时收藏列表

- [√] 2.5 新增 `filteredFolderFavorites` 计算（支持 any/all）
  - 验证: 关键字+模式组合覆盖正确

- [√] 2.6 新增 `sortedFolderFavorites` 计算
  - 验证: 三种排序维度均生效

- [√] 2.7 实现 `upsertFolderFavorite(payload)` 通用动作
  - 验证: 可对任意收藏项直接保存，不依赖当前目录

- [√] 2.8 `upsertCurrentFolderFavorite` 改为复用通用动作（兼容保留）
  - 验证: 当前目录收藏功能无回归

- [√] 2.9 编辑保存后同步当前目录草稿（若命中同 folderId）
  - 验证: 顶部收藏区与列表内容一致

- [√] 2.10 筛选输入增加 200ms 防抖
  - 验证: 快速输入时无明显卡顿

### 3. 前端：API 封装与参数预留

- [√] 3.1 在 `frontend/src/api/mtphoto.ts` 为 `getMtPhotoFolderFavorites` 增加可选查询参数类型
  - 验证: 不传参数时请求行为与当前一致

- [√] 3.2 在调用处保持 V1 默认“本地筛选”路径（不依赖后端过滤）
  - 验证: 后端未实现过滤时功能仍完整

### 4. 后端：可演进接口预留（兼容）

- [√] 4.1 在 `internal/app/mtphoto_folder_favorite_handlers.go` 预留 query 参数读取结构（可先不启用过滤）
  - 验证: 老请求无感知，新参数不报错

- [√] 4.2 在 `internal/app/mtphoto_folder_favorite.go` 增加 `ListOptions` 结构体（预留）
  - 验证: 默认 options 下与旧 `List` 结果一致

- [√] 4.3 补充注释与文档，明确 V1/V2 语义边界
  - 验证: 维护者可直接理解演进策略

### 5. 测试与回归

- [?] 5.1 前端手工回归：桌面端筛选 + 编辑 + 跳转 + 删除链路
  - 验证: 无报错、状态一致

- [?] 5.2 前端手工回归：移动端抽屉与编辑弹层链路
  - 验证: 不遮挡、可关闭、可保存

- [-] 5.3 后端单测补充（如启用 options 预留结构）
  - 验证: 兼容路径稳定

- [√] 5.4 执行 `go test ./...`
  - 验证: 后端门禁通过

- [√] 5.5 执行 `cd frontend && npm run build`
  - 验证: 前端门禁通过

### 6. 文档与收尾

- [√] 6.1 更新 `helloagents/modules/mtphoto.md`（收藏筛选/编辑能力说明）
  - 验证: 模块文档与实现一致

- [√] 6.2 更新 `helloagents/modules/api.md`（参数预留说明）
  - 验证: API 文档兼容语义明确

- [√] 6.3 更新 `helloagents/CHANGELOG.md`
  - 验证: 本次功能变更可追溯

