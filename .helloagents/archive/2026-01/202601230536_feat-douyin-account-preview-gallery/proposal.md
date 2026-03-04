# 变更提案: feat_douyin_account_preview_gallery

## 元信息
```yaml
类型: 新增
方案类型: implementation
优先级: P2
状态: 已完成
创建: 2026-01-23
```

---

## 1. 需求

### 背景
抖音下载弹窗的“用户作品”模式会返回多个作品，但预览目前只针对“单个作品”的资源列表（仅当该作品为图集时才会出现画廊）。
期望：多个作品也能构成一个画廊，在预览时左右滑动切换不同作品，并能看到当前作品名称。

### 目标
- 用户作品模式：点击任一作品进入预览后，可左右滑动切换到同一批次“已加载作品”的其他作品。
- 预览顶部展示当前作品名称（优先 `desc`，兜底 `detailId`）。
- 切换作品后，“下载/导入上传”仍使用正确的 `key + index` 上下文，不因不同作品的 `index` 重复而串线。

### 约束条件
```yaml
依赖约束: 不新增第三方依赖，复用现有 MediaPreview 画廊能力
兼容性约束: 仅增强预览体验，不改变下载/导入接口语义
```

### 验收标准
- [x] `npm -C frontend run build` 通过
- [x] `go test ./...` 通过
- [x] `helloagents/wiki/modules/douyin-downloader.md` 已同步交互说明

---

## 2. 方案

### 技术方案
- **前端（DouyinDownloadModal）**
  - 在打开“用户作品预览”时，将当前 `accountItems` 中可用的 `items` 聚合为一个 `MediaPreview.mediaList`，实现跨作品画廊滑动。
  - 为每个预览条目写入 `UploadedMedia.title`（作品名）与 `UploadedMedia.context={provider:'douyin',key,index}`，用于在切换时恢复正确导入上下文。
  - 导入状态/文件大小缓存由“仅按 index”改为“按 key:index”复合键，避免不同作品 `index` 冲突。
- **前端（MediaPreview）**
  - 顶部工具栏新增可选标题展示：当 `currentMedia.title` 存在时展示。

### 影响范围
```yaml
涉及模块:
  - Douyin Downloader: 用户作品预览交互
  - MediaPreview: 顶部标题展示
预计变更文件: 3
```

### 风险评估
| 风险 | 等级 | 应对 |
|------|------|------|
| 不同作品 index 重复导致导入状态错乱 | 中 | 状态 key 改为 `key:index`，并在预览切换时更新上下文 |

---

## 3. 技术决策

### feat-douyin-account-preview-gallery#D001: 用 `mediaList + context(key,index)` 实现跨作品画廊且保持导入上下文正确
**日期**: 2026-01-23  
**状态**: ✅采纳  
**理由**: 复用既有 `MediaPreview` 画廊与滑动能力，最小化 UI 改动，并避免为“作品级别”再引入第二套滑动组件。

