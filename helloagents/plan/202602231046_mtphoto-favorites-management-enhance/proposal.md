# 变更提案: mtphoto-favorites-management-enhance

## 元信息
```yaml
类型: 功能增强
方案类型: implementation
优先级: P1
状态: 实施中（代码完成，待手工回归）
创建: 2026-02-23
关联会话: ec377fd6-8ed0-4969-b9f2-71f04ca2311c
```

---

## 1. 需求

### 背景
当前 mtPhoto 文件夹收藏能力已支持：
- 收藏当前目录
- 编辑当前目录的标签/备注
- 在收藏列表中展示标签并跳转目录

但仍存在三个核心缺口：
1. 无法在收藏列表中直接编辑标签/备注（必须先进入目录）。
2. 收藏列表无法按标签筛选，目录多时查找效率低。
3. 排序/分组缺少明确扩展位，后续增强（如按标签分组）需要重构风险。

### 目标
- 在收藏列表中直接编辑标签/备注（不要求先切换目录）。
- 支持按标签筛选收藏目录，满足快速定位。
- 提供排序与分组能力的架构预留，保证后续扩展可平滑接入。
- 保证移动端（<1024px）可用性，不挤压图片主内容区。

### 非目标（本期不做）
- 不引入复杂的后端聚合统计接口（如分组计数接口）。
- 不实现批量编辑/批量打标。
- 不改动数据库表结构（沿用 `tags_json`）。

### 约束条件
```yaml
兼容性: 维持现有 API 路径不变（/getMtPhotoFolderFavorites /upsertMtPhotoFolderFavorite /removeMtPhotoFolderFavorite）
性能: 前端本地筛选、排序需可承载 200+ 收藏项，不出现明显卡顿
移动端: 收藏筛选与编辑需在抽屉/弹层内可完整触达
安全: 服务端继续校验 tags 与 note，禁止绕过长度与空值规范
回滚: 变更需保持可按前端/后端粒度独立回滚
```

### 验收标准
- [ ] 收藏列表支持“编辑标签/备注”，保存后列表即时更新。
- [ ] 支持按标签筛选（关键字 + 标签 chips + any/all 模式）。
- [ ] 桌面端与移动端均可完成筛选、编辑、跳转、移除全链路操作。
- [ ] 排序能力可配置（至少 `updatedAt/name/tagCount`）。
- [ ] 代码门禁通过：`go test ./...`、`cd frontend && npm run build`。

---

## 2. 方案

### 2.1 备选方案对比

| 方案 | 描述 | 优点 | 风险 | 结论 |
|------|------|------|------|------|
| A 前端本地增强 | 仅前端实现筛选/排序/分组，后端不扩展参数 | 交付快、改动小 | 收藏量继续上升时本地计算压力增大 | 可做短期兜底 |
| B 混合增强（推荐） | V1 前端本地能力落地；后端 API 预留可选 query 参数（不破坏兼容） | 快速交付 + 后续可演进 | 设计时需明确参数语义 | **推荐** |
| C 后端优先 | 立即改造后端为服务端筛选/排序/分组主导 | 大数据量稳定 | 周期长、测试面广、收益滞后 | 本期不建议 |

**最终采用：方案 B（混合增强）**

---

### 2.2 前端交互设计（`MtPhotoAlbumModal.vue`）

#### 收藏区（桌面端）
- 在收藏列表头部增加筛选工具行：
  - 标签关键字输入框（支持模糊匹配）
  - 匹配模式切换：`任一标签(ANY)` / `全部标签(ALL)`
  - 排序下拉：`最近更新`、`名称`、`标签数`
- 收藏卡片增加“编辑”入口：
  - 打开行内编辑面板或右侧浮层（建议行内，改动更小）
  - 编辑字段：`tags`、`note`
  - 保存后关闭并刷新对应卡片内容

#### 收藏区（移动端）
- 复用已有收藏抽屉（`isMobileFavoritesOpen`）。
- 筛选区放在抽屉顶部；编辑区使用 `fixed` 弹层（或 Teleport 到 body），避免挤压主图片区。
- 抽屉与编辑弹层分层明确：`遮罩 < 抽屉 < 编辑弹层`。

#### 交互细节
- 输入筛选加 200ms 防抖。
- 切换模式/切换目录时，保持筛选条件（便于连续操作）；仅在关闭弹窗时重置。
- 保存成功后 toast 提示并高亮更新项（可选）。

---

### 2.3 状态与数据设计（`frontend/src/stores/mtphoto.ts`）

新增状态：
- `favoriteFilterKeyword: string`
- `favoriteFilterMode: 'any' | 'all'`
- `favoriteSortBy: 'updatedAt' | 'name' | 'tagCount'`
- `favoriteSortOrder: 'asc' | 'desc'`
- `favoriteEditingFolderId: number | null`
- `favoriteDraftTags: string`
- `favoriteDraftNote: string`

新增计算：
- `allUniqueTags`：从 `folderFavorites` 聚合去重标签（供 chips/自动补全）。
- `filteredFolderFavorites`：按关键字 + 模式筛选。
- `sortedFolderFavorites`：在筛选结果上排序。
- `groupedFolderFavorites`（预留）：当前默认 `groupBy=none`，仅保留计算入口。

新增动作：
- `upsertFolderFavorite(payload)`：通用 upsert（可编辑任意收藏项）。
- `startEditFavorite(item)` / `cancelEditFavorite()`。
- `applyFavoriteFilter(...)` / `resetFavoriteFilter()`。

兼容策略：
- 保留 `upsertCurrentFolderFavorite`，内部可复用 `upsertFolderFavorite`。
- `folderFavorites` 更新采用“替换对应项 + 保持当前排序策略”，避免无意跳序。

---

### 2.4 API 与后端设计

#### 前端 API（`frontend/src/api/mtphoto.ts`）
- 保持现有接口路径。
- `getMtPhotoFolderFavorites` 预留可选参数：
  - `tagKeyword`
  - `tagMode`（any/all）
  - `sortBy`、`sortOrder`
  - `groupBy`（none/tag，当前默认 none）

> V1 可先不下发 query，全部本地计算；参数定义先完成，便于 V2 切换服务端筛选。

#### 后端（`internal/app/mtphoto_folder_favorite_handlers.go` / `mtphoto_folder_favorite.go`）
- V1 行为保持兼容：`GET /getMtPhotoFolderFavorites` 返回全量。
- V2 扩展预留：
  - 允许读取 query 参数但默认忽略（或仅校验后透传给 service）。
  - service 层新增 `ListOptions` 结构以承接后续服务端筛选/排序。

---

### 2.5 影响范围

```yaml
前端:
  - frontend/src/components/media/MtPhotoAlbumModal.vue
  - frontend/src/stores/mtphoto.ts
  - frontend/src/api/mtphoto.ts
后端（预留层）:
  - internal/app/mtphoto_folder_favorite_handlers.go
  - internal/app/mtphoto_folder_favorite.go
测试:
  - internal/app/mtphoto_folder_favorite*_test.go
  - frontend（如有现有测试体系则补充 store/component 用例）
预计改动文件: 5~8
```

---

### 2.6 风险与应对

| 风险 | 等级 | 应对 |
|------|------|------|
| 编辑收藏后 UI 不一致（当前目录区与收藏卡片不同步） | 中 | store 统一以 upsert 返回值回写；若编辑的是当前目录，同步刷新顶部草稿 |
| 直接编辑 payload 缺字段导致 upsert 失败 | 中 | 编辑入口基于 `folderFavorites` 原始项补齐 `folderName/folderPath/folderId` |
| 移动端抽屉 + 编辑弹层遮挡冲突 | 高 | 弹层使用 fixed/Teleport 并统一 z-index 规范 |
| 本地筛选在收藏过多时卡顿 | 中 | 输入防抖 + 计算结果缓存（computed） |
| 排序策略导致条目位置突变 | 低 | 明确“按更新时间排序”语义；若非该模式则保持稳定替换 |

---

## 3. Gemini 审核结论

### 审核摘要
- 审核模型：Gemini（通过 `collaborating-with-gemini`）
- 会话：`ec377fd6-8ed0-4969-b9f2-71f04ca2311c`
- 结论：**建议按此方案实施**

### 必改项（已纳入本方案）
1. 明确“编辑后数据一致性”处理：当前目录态与收藏列表态同步。
2. 明确“直接编辑 payload 补完”策略：保证 `folderName/folderPath/folderId` 完整。
3. 移动端编辑弹层采用 fixed/Teleport，避免被抽屉遮挡。
4. 增加 `allUniqueTags` 聚合计算，支撑筛选 chips 与编辑补全。
5. 关键字筛选加防抖，避免大列表计算抖动。

### 可选优化（进入后续迭代）
- 标签自动补全候选。
- 键盘快捷聚焦筛选框（桌面端 `/`）。
- 批量操作视觉预留（多选框位）。

---

## 4. 交付计划（里程碑）

- M1：前端交互落地（筛选 + 直接编辑 + 移动端弹层）
- M2：store 重构（通用 upsert、筛选排序计算）
- M3：API/后端预留参数结构（兼容不破坏）
- M4：回归与验收（桌面/移动/异常分支）
