# 技术设计: 媒体图库弹窗扩大显示区域

## 技术方案

### 核心技术
- Vue 3 + TailwindCSS（基于现有 class 体系做布局微调）

### 实现要点
- 弹窗容器：
  - 统一调整 `AllUploadImageModal` 与 `MtPhotoAlbumModal` 的宽高约束（提升 `max-width`，提高弹窗高度；并在支持的浏览器使用 `dvh` 优化移动端动态地址栏场景）。
  - 保持 `flex flex-col` 结构与 `InfiniteMediaGrid` 的 `flex-1` 行为不变，避免影响加载逻辑。
- 列表滚动容器：
  - 下调 `InfiniteMediaGrid` 顶层滚动容器内边距（`p-*`），在不贴边的前提下扩大有效内容区。
  - 保持 `no-scrollbar`、`overflow-y-auto`、滚动触底阈值逻辑不变。

## 安全与性能
- **安全:** 纯前端样式调整，无鉴权/数据面风险；不引入新依赖。
- **性能:** 仅 class 变更，不增加渲染开销；瀑布流列计算逻辑不变。

## 测试与部署
- **测试:** `npm -C frontend run build`（类型检查 + 构建）；手工验证两处弹窗在瀑布流/网格下的展示与滚动加载。
- **部署:** 无额外步骤，随前端资源构建发布。
