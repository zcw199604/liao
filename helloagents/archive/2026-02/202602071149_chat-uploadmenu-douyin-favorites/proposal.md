# 变更提案: chat-uploadmenu-douyin-favorites

## 元信息
```yaml
类型: 新功能
方案类型: implementation
优先级: P1
状态: 已完成
创建: 2026-02-07
```

---

## 1. 需求

### 背景
当前聊天页“+”上传菜单（`UploadMenu`）仅提供：
- 选择文件上传
- 历史聊天图片
- 所有上传图片
- mtPhoto 相册

你补充的核心诉求是：**收藏对象是“抖音作者”，不是单条作品**。
即：希望在聊天发送场景里，从“+”菜单直接进入“已收藏作者列表”，点击作者即可查看该作者的全部已入库作品，并继续导入到本地服务器。

### 目标
- 在聊天页“+”菜单中新增“抖音收藏作者”入口。
- 点击后直接进入“抖音下载”弹窗的“收藏 -> 用户收藏（作者列表）”视图。
- 用户点击作者后可查看该作者全部作品（分页/下拉加载），并可在预览中执行导入到本地服务器。

### 约束条件
```yaml
时间约束: 本次仅做前端入口与交互打通，不做大规模重构
性能约束: 不新增重型轮询；继续复用现有分页/懒加载接口
兼容性约束: 保持设置页等原有抖音下载入口行为不变
业务约束: 导入语义保持与现有 /api/douyin/import 一致（导入本地，不改变后端协议）
```

### 验收标准
- [ ] 聊天页点击“+”后，可看到“抖音收藏作者”按钮。
- [ ] 点击该按钮后，抖音弹窗直接进入 `activeMode=favorites` 且默认 `favoritesTab=users`。
- [ ] 点击任一收藏作者后，能进入作者详情并查看其全部作品（含分页加载）。
- [ ] 在作者作品预览中点击导入，调用既有导入流程成功。
- [ ] 原入口（设置页 -> 抖音下载）仍按旧行为默认进入“作品解析”模式，不受影响。

---

## 2. 方案

### 技术方案
采用“复用现有抖音弹窗 + 增加入口上下文”的轻量改造：

1. **UploadMenu 增加新动作按钮与事件**
   - 文件：`frontend/src/components/chat/UploadMenu.vue`
   - 新增按钮“抖音收藏作者”，触发 `openDouyinFavoriteAuthors` 事件。

2. **聊天页接入新事件**
   - 文件：`frontend/src/views/ChatRoomView.vue`
   - 监听 `@open-douyin-favorite-authors`，调用 `douyinStore.open({...})`。
   - 关闭上传菜单并打开抖音弹窗。

3. **扩展 douyin store 打开参数**
   - 文件：`frontend/src/stores/douyin.ts`
   - 在 `open` 中支持上下文参数（如 `entryMode: 'favorites'`、`favoritesTab: 'users'`），并在 `close` 清理。

4. **DouyinDownloadModal 消费入口上下文**
   - 文件：`frontend/src/components/media/DouyinDownloadModal.vue`
   - `watch(douyinStore.showModal)` 初始化时读取 store 上下文：
     - 来自聊天上传菜单时，默认进入“收藏/用户收藏（作者）”模式。
     - 其他入口保持原默认行为。
   - 点击作者后复用现有 `openFavoriteUserDetail` + `listDouyinFavoriteUserAwemes` 流程展示该作者全部作品。

5. **测试补齐**
   - 文件：
     - `frontend/src/__tests__/chat-components.test.ts`
     - `frontend/src/__tests__/upload-menu-more.test.ts`
     - `frontend/src/__tests__/chatroom-more.test.ts`（或现有 ChatRoomView 相关测试）
     - `frontend/src/__tests__/douyin-download-modal-modes.test.ts`
   - 校验新增按钮事件、聊天页触发链路、弹窗默认模式与作者详情展开分支。

### 影响范围
```yaml
涉及模块:
  - chat-ui: UploadMenu 新入口与 ChatRoomView 触发链路
  - douyin-downloader: 弹窗入口模式初始化逻辑（默认落用户收藏）
  - state-store: douyin store 打开上下文能力
  - tests: 前端单测分支更新
预计变更文件: 5~8
```

### 风险评估
| 风险 | 等级 | 应对 |
|------|------|------|
| 新增打开上下文导致旧入口行为变化 | 中 | 上下文参数可选，默认值保持当前行为；补回归测试 |
| UploadMenu 按钮顺序变化导致既有测试脆弱（按 index 点击） | 中 | 调整测试为按文本/语义选择器触发，减少 index 依赖 |
| 用户将“导入本地”误解为“已直接发送” | 低 | 提示文案明确“导入后请在上传菜单发送” |

---

## 3. 技术设计（可选）

### 架构设计
```mermaid
flowchart TD
    A[ChatRoomView: 点击 + 菜单] --> B[UploadMenu: 抖音收藏作者]
    B --> C[douyinStore.open(entry=favorites, tab=users)]
    C --> D[DouyinDownloadModal 收藏-作者列表]
    D --> E[点击作者 -> 作者作品列表]
    E --> F[MediaPreview 导入]
    F --> G[/api/douyin/import]
```

### API设计
本次不新增后端 API，继续使用：
- `GET /api/douyin/favoriteUser/list`
- `GET /api/douyin/favoriteUser/aweme/list`
- `POST /api/douyin/import`

### 数据模型
本次不新增后端表结构，仅新增前端打开上下文（store 临时状态）。

---

## 4. 核心场景

### 场景: 聊天页直达收藏作者
**模块**: chat-ui / douyin-downloader
**条件**: 用户在聊天页已打开“+”上传菜单
**行为**: 点击“抖音收藏作者” -> 打开抖音弹窗并进入“收藏/用户收藏”
**结果**: 用户无需先进入设置页，即可浏览收藏作者

### 场景: 点击作者查看全部作品
**模块**: douyin-downloader
**条件**: 已进入收藏作者列表
**行为**: 点击作者卡片 -> 打开作者详情 -> 下拉加载该作者作品
**结果**: 可以浏览作者已入库的全部作品并进入预览

### 场景: 作者作品导入本地
**模块**: douyin-downloader
**条件**: 已打开作者作品预览
**行为**: 点击导入
**结果**: 走现有导入链路写入本地服务器（成功/去重复用状态可见）

---

## 5. 技术决策

### chat-uploadmenu-douyin-favorites#D001: 复用现有 DouyinDownloadModal，而非新建独立作者弹窗
**日期**: 2026-02-07
**状态**: ✅采纳
**背景**: 聊天上传菜单需要“低成本接入收藏作者与作者作品浏览”，现有弹窗已具备收藏用户详情与作品加载能力。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: 复用 DouyinDownloadModal + 增加打开上下文 | 改动面小、复用现有测试与能力、上线风险低 | 需要在 store/modal 增加少量上下文逻辑 |
| B: 新建 Chat 专用“收藏作者”弹窗 | 可高度定制聊天场景 | 重复实现作者列表/作品预览/导入链路，维护成本高 |
**决策**: 选择方案 A
**理由**: 满足“收藏作者 -> 查看作者全部作品”需求且工程成本最低。
**影响**: `stores/douyin.ts`、`DouyinDownloadModal.vue`、`ChatRoomView.vue`、`UploadMenu.vue`

### chat-uploadmenu-douyin-favorites#D002: 入口模式采用“可选上下文参数”并强制聊天入口落在作者收藏
**日期**: 2026-02-07
**状态**: ✅采纳
**背景**: 抖音弹窗有多个入口（设置页、聊天页），聊天入口需要直接看到作者收藏，设置入口仍需保持默认解析模式。
**选项分析**:
| 选项 | 优点 | 缺点 |
|------|------|------|
| A: `open(options?)` 支持上下文（entry/mode/tab） | 兼容旧调用，按入口精细化控制 | 需要在 open/close 增加状态清理 |
| B: 统一默认改为 favorites/users | 实现最简单 | 会改变设置页等旧入口习惯，回归风险高 |
**决策**: 选择方案 A
**理由**: 兼顾新需求与旧行为稳定性。
**影响**: `frontend/src/stores/douyin.ts`, `frontend/src/components/media/DouyinDownloadModal.vue`
