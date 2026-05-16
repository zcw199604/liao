# 媒体预览视频操作重构技术方案

## 推荐方案

采用“最小结构重构 + 场景化动作配置”方案。

核心做法：

1. 保留 `MediaPreview` 作为统一预览层。
2. 在 `MediaPreview` 内新增视频工具菜单，收纳低频处理动作。
3. 将“抓帧”统一改为“保存当前帧”，将“抽帧”统一改为“创建抽帧任务”。
4. 为 `MediaPreview` 增加轻量 props，允许调用方控制视频工具项是否显示。
5. 各入口只暴露自己的主业务按钮，把视频处理动作降为次级工具。

## 备选方案评估

### 方案 1: 最小结构重构 + 场景化动作配置（推荐）

优点：

- 保留现有组件和 store 逻辑，改动集中在前端交互层。
- 不动后端接口和抽帧任务模型，风险较低。
- 可以逐步迁移调用方，不需要一次性重写媒体系统。

缺点：

- `MediaPreview` 仍然较重，只是通过配置降低不同场景的外显复杂度。
- 需要补测试覆盖不同 props 组合。

### 方案 2: 拆分独立 `VideoPreview` 与 `ImagePreview`

优点：

- 职责最清晰，长期维护性更好。
- 视频工具可以独立演进。

缺点：

- 改动范围大，涉及所有预览入口和测试。
- 容易引入行为回退，当前不适合作为第一阶段。

### 方案 3: 仅改文案，不调整按钮结构

优点：

- 成本最低。

缺点：

- 无法解决底部按钮并列导致的主次混乱。
- 入口场景差异仍然存在。

## 交互设计

### 1. 预览层动作分区

`MediaPreview` 保留以下固定区域：

- 顶部工具栏：详情、倍速、下载、关闭。
- 播放区域：播放/暂停、快退/快进、手势、全屏。
- 底部主动作：由调用方通过 `canUpload`、`uploadText` 等现有能力控制。
- 视频工具菜单：仅视频显示，收纳次级处理动作。

### 2. 视频工具菜单

新增一个统一入口：

- 图标：`fa-ellipsis-h` 或 `fa-tools`
- 文案：`视频工具`
- 菜单项：
  - `保存当前帧`
  - `创建抽帧任务`

显示规则：

- `保存当前帧` 默认对视频可见，可通过 props 关闭。
- `创建抽帧任务` 仅当 `canExtractFrames` 为 true 且 props 允许时显示。
- 全屏时菜单可放在右侧浮层中，非全屏时放在底部或顶部工具区，但不与主按钮并列。

### 3. 文案统一

替换规则：

- `抓帧` → `保存当前帧`
- `抽帧（进入抽帧任务）` → `创建抽帧任务`
- `预览/抓帧` → `预览源视频`
- `抽帧任务中心` 保留或调整为 `视频抽帧任务`

### 4. 来源入口策略

#### mtPhoto 相册

- 图片预览主按钮：`导入此图片`
- 视频预览主按钮：`导入此视频`
- 视频工具菜单保留：
  - `保存当前帧`
  - `创建抽帧任务`

#### 全站媒体库

- 普通模式主按钮：保留重新上传/加载发送相关动作。
- 管理模式主动作：删除、选择，不展示上传主按钮。
- 视频工具菜单保留，但不抢占底部主按钮。
- 标题可从“所有上传图片”逐步改为“媒体库”，因为列表已包含视频。

#### 抖音下载

- 解析结果卡片主动作：预览、导入、下载。
- 预览层主按钮：导入或下载。
- 视频工具菜单可显示 `保存当前帧`。
- `创建抽帧任务` 仅在来源可被 `openCreateFromMedia` 解析时显示；不可解析时不显示，避免点击后失败。

#### 抽帧任务中心

- 创建弹窗头部按钮改为 `预览源视频`。
- 如果从创建弹窗内打开源视频预览，默认隐藏 `创建抽帧任务`，避免在抽帧创建流程里再次打开新的创建流程。
- 可保留 `保存当前帧`，作为查看源视频时的辅助动作。

## 技术设计

### 1. `MediaPreview` props 扩展

建议新增：

```ts
interface Props {
  showVideoTools?: boolean
  showCaptureFrame?: boolean
  showExtractTask?: boolean
}
```

默认值：

```ts
showVideoTools: true
showCaptureFrame: true
showExtractTask: true
```

计算逻辑：

- `canShowVideoTools = currentMedia.type === 'video' && showVideoTools && (canShowCaptureFrame || canShowExtractTask)`
- `canShowCaptureFrame = showCaptureFrame && currentMedia.type === 'video'`
- `canShowExtractTask = showExtractTask && canExtractFrames`

### 2. 菜单状态

新增本地状态：

```ts
const showVideoToolMenu = ref(false)
```

关闭时机：

- 切换媒体
- 关闭预览
- 点击菜单外部
- 执行任一菜单项后

### 3. 处理函数调整

- `handleCaptureFrame` 保持现有逻辑，文案改为“保存当前帧”。
- `handleExtractFrames` 保持现有逻辑，但点击入口从独立按钮迁移到菜单项。
- 执行 `handleExtractFrames` 前关闭菜单，再调用 `videoExtractStore.openCreateFromMedia`。

### 4. 调用方调整

`VideoExtractCreateModal` 中打开源视频预览时传入：

```vue
<MediaPreview
  :show-extract-task="false"
/>
```

其他入口先使用默认配置，后续可按场景收紧。

### 5. 样式原则

- 不新增大面积卡片或装饰背景。
- 工具菜单沿用现有 overlay 菜单样式。
- 移动端按钮尺寸保持不小于 44px 可点击区域。
- 非全屏底部主按钮只保留主业务动作，视频工具入口使用小型图标按钮或菜单按钮。

## 安全与性能

- 不新增后端接口，不改变鉴权边界。
- 不扩大跨域 canvas 抓帧能力；继续捕获 `SecurityError` 并提示使用本地库或抽帧任务。
- 抽帧任务入口继续由 `openCreateFromMedia` 做来源解析和门禁，普通远程 URL 不直接创建任务。
- 菜单状态和按钮计算为前端轻量逻辑，对渲染性能影响很小。

## 测试策略

### 单元测试

更新或新增前端测试：

- `MediaPreview` 视频场景显示 `视频工具`。
- 点击视频工具后显示 `保存当前帧`。
- 当 `showExtractTask=false` 时不显示 `创建抽帧任务`。
- `VideoExtractCreateModal` 源视频预览不显示再次创建抽帧任务入口。

### 构建验证

执行：

```bash
cd frontend && npm test
cd frontend && npm run build
```

### 人工验证

- mtPhoto 相册图片/视频预览。
- 全站媒体库普通模式和管理模式。
- 抖音下载解析后的视频预览。
- 抽帧任务中心上传视频后创建任务。
- 视频全屏时工具菜单不遮挡播放控制。

## ADR

### ADR-202605160822: 视频处理动作从预览主按钮降级为视频工具菜单

- **状态:** 提议
- **日期:** 2026-05-16
- **影响模块:** Media, Video Extract, mtPhoto, Douyin Downloader
- **决策:** 不再在视频预览底部并列展示“抓帧 / 抽帧 / 上传”。即时帧保存和任务型抽帧入口统一收进视频工具菜单，底部保留来源入口的主业务动作。
- **理由:** 保持预览层主任务清晰，降低“保存当前帧”和“创建抽帧任务”的语义混淆，同时避免一次性拆分预览组件带来的高风险。
- **后果:** `MediaPreview` 增加少量配置和菜单状态；后续如继续膨胀，可再拆分独立 `VideoPreview`。
