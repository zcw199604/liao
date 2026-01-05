# WebSocket 消息 Code 码参考文档

> 本文档整理了匿名匹配聊天应用中所有WebSocket消息的code码及其处理逻辑
> 最后更新：2026-01-05

---

## 📚 目录

- [快速查询表](#快速查询表)
- [按功能分类详解](#按功能分类详解)
  - [系统管理类 (-3 ~ -1)](#系统管理类--3---1)
  - [用户管理类 (3, 4, 17, 30)](#用户管理类-3-4-17-30)
  - [消息通信类 (5 ~ 9)](#消息通信类-5--9)
  - [系统通知类 (11, 12, 20 ~ 22)](#系统通知类-11-12-20--22)
  - [随机匹配类 (13 ~ 16)](#随机匹配类-13--16)
  - [输入状态类 (18 ~ 19)](#输入状态类-18--19)
  - [VIP管理类 (130 ~ 139)](#vip管理类-130--139)
- [本地消息过滤规则](#本地消息过滤规则)
- [已知问题](#已知问题)
- [相关文件](#相关文件)

---

## 快速查询表

| Code | 消息类型 | 是否显示在聊天记录 | 备注 |
|------|---------|------------------|------|
| -3 | 强制断开旧连接 | ❌ | 防止重复登录 |
| -2 | 服务器反馈信息 | ✅ | 根据具体内容 |
| -1 | 警告信息 | ✅ | 包含forceout标记 |
| 3 | 用户离线 | ✅ | 显示离线通知 |
| 4 | 用户改名 | ✅ | 格式：`旧名 --> 新名` |
| 5 | 广播消息 | ✅ | 公共聊天室消息 |
| 6 | 历史广播消息 | ✅ | 加载历史记录 |
| 7 | 私信消息 | ✅ | 一对一聊天 |
| 9 | 撤回消息 | ✅ | 消息显示为"已撤回" |
| 11 | 全体公告 | ✅ | 系统公告 |
| 12 | 连接成功 | ❌ Toast | 仅提示，不入聊天记录 |
| 13 | 进入匹配队列 | ❌ Toast | 仅提示 |
| 14 | 退出匹配队列 | ⚠️ **会显示** | **设计缺陷，应该过滤** |
| 15 | 匹配成功 | ✅ | 获取到匹配对象ID |
| 16 | 对方离开匹配 | ❌ Toast | 仅提示 |
| 17 | 修改性别/地址 | ✅ | 个人信息变更 |
| 18 | 对方正在输入 | ❌ 状态显示 | 输入提示动画 |
| 19 | 停止输入 | ❌ Toast | 取消输入提示 |
| 20 | 弹幕 | ✅ 弹幕动画 | 特殊展示形式 |
| 21 | 违规封禁 | ✅ | 封禁提示 |
| 22 | 举报成功 | ❌ Toast | 加入黑名单提示 |
| 30 | 用户登录信息 | ✅ | 查看用户详情 |
| 130-139 | VIP相关 | ✅ | VIP激活码管理 |

---

## 按功能分类详解

### 系统管理类 (-3 ~ -1)

#### Code: -3 - 强制断开旧连接

**消息示例**：
```json
{
  "code": -3,
  "content": "有新连接，旧的断开"
}
```

**处理逻辑**：
- 上游处理函数：`fun_closeOld(ws)`
- 本地行为：强制关闭当前WebSocket连接
- 使用场景：防止同一用户重复登录，保证账号安全

**相关代码**：无（直接断开连接）

---

#### Code: -2 - 服务器反馈信息

**消息示例**：
```json
{
  "code": -2,
  "content": "操作成功/失败信息"
}
```

**处理逻辑**：
- 上游处理函数：`fun_fkxx(json.content)`
- 本地行为：根据content内容显示反馈信息
- 使用场景：服务器对客户端操作的响应

---

#### Code: -1 - 警告信息（强制下线）

**消息示例**：
```json
{
  "code": -1,
  "content": "您的账号在其他地方登录",
  "forceout": true
}
```

**处理逻辑**：
- 上游处理函数：`Double_Con_Limit(json)`
- 本地行为：
  - 如果 `forceout=true`，触发ForceoutManager机制
  - 80秒内禁止重连，防止IP被封
  - 显示警告提示
- 特殊字段：`forceout` - 是否为强制下线

**相关代码**：
- ForceoutManager: `src/main/java/com/zcw/websocket/ForceoutManager.java`

---

### 用户管理类 (3, 4, 17, 30)

#### Code: 3 - 用户离线

**消息示例**：
```json
{
  "code": 3,
  "userid": "abc123",
  "nickname": "用户A",
  "time": "14:30:25"
}
```

**处理逻辑**：
- 上游处理函数：`fun_userlogout(json)`
- 本地行为：
  - 从在线用户列表移除该用户
  - 在聊天记录显示离线通知
  - 更新用户列表UI

**相关代码**：
- `frontend/src/composables/useWebSocket.ts` - 用户列表管理逻辑

---

#### Code: 4 - 用户更改名称

**消息示例**：
```json
{
  "code": 4,
  "userid": "abc123",
  "beforename": "旧昵称",
  "nickname": "新昵称",
  "time": "14:30:25"
}
```

**处理逻辑**：
- 上游处理函数：`fun_userchgname(json)`
- 本地行为：
  - 更新用户列表中的昵称
  - 在聊天记录显示：`旧昵称 --> 新昵称`
  - 更新所有相关UI组件

**显示格式**：
```
14:30:25
旧昵称 --> 新昵称
```

**相关代码**：
- `frontend/src/stores/identity.ts` - 身份信息管理

---

#### Code: 17 - 用户更改性别/地址开关

**消息示例**：
```json
{
  "code": 17,
  "userid": "abc123"
}
```

**处理逻辑**：
- 上游处理函数：`fun_userchgsexaddress()`
- 本地行为：通知用户个人信息已更新
- 使用场景：用户在设置中修改性别或地址显示开关

**相关代码**：
- `frontend/src/components/settings/` - 设置面板组件

---

#### Code: 30 - 查看用户登录信息

**消息示例**：
```json
{
  "code": 30,
  "userid": "abc123",
  "ip": "192.168.1.1",
  "address": "广东省深圳市",
  "loginTime": "2026-01-05 14:00:00"
}
```

**处理逻辑**：
- 上游处理函数：`ShowUserLoginInfo(json)`
- 本地行为：展示用户的登录详情（IP、地址、登录时间等）
- 使用场景：管理员或VIP用户查看其他用户信息

---

### 消息通信类 (5 ~ 9)

#### Code: 5 - 接收广播消息（公共聊天）

**消息示例**：
```json
{
  "code": 5,
  "fromuser": {
    "id": "abc123",
    "nickname": "用户A"
  },
  "type": "text",
  "content": "大家好！",
  "time": "14:30:25",
  "tid": "1704441025123"
}
```

**处理逻辑**：
- 上游处理函数：`fun_recbrodata(json)`
- 本地行为：
  - 添加到公共聊天记录
  - 实时显示在聊天界面
  - 如果包含@提及，高亮显示
- 消息类型：text（文本）、image（图片）、video（视频）

**相关代码**：
- `frontend/src/stores/message.ts:addMessage()` - 消息存储
- `frontend/src/components/chat/MessageList.vue` - 消息渲染

---

#### Code: 6 - 绑定所有广播消息（历史记录）

**消息示例**：
```json
{
  "code": 6,
  "messages": [
    {
      "fromuser": {"id": "abc123", "nickname": "用户A"},
      "content": "历史消息1",
      "time": "14:20:00",
      "tid": "1704440400000"
    },
    {
      "fromuser": {"id": "def456", "nickname": "用户B"},
      "content": "历史消息2",
      "time": "14:25:00",
      "tid": "1704440700000"
    }
  ]
}
```

**处理逻辑**：
- 上游处理函数：`fun_bindtoallmsg(json)`
- 本地行为：
  - 批量加载公共聊天历史记录
  - 按时间排序插入聊天记录
  - 用于用户刚进入聊天室时加载历史消息
- 使用场景：用户首次进入房间或刷新页面

---

#### Code: 7 - 用户发送的新私信

**消息示例**：
```json
{
  "code": 7,
  "fromuser": {
    "id": "abc123",
    "nickname": "用户A"
  },
  "touser": {
    "id": "def456",
    "nickname": "用户B"
  },
  "type": "text",
  "content": "你好，私聊消息",
  "time": "14:30:25",
  "tid": "1704441025456"
}
```

**处理逻辑**：
- 上游处理函数：`fun_fromusermsg(json)`
- 本地行为：
  - 添加到对应用户的私聊记录
  - 如果不是当前聊天对象，显示未读消息提示
  - 播放消息提示音（可配置）
- 消息类型：text、image、video

**相关代码**：
- `frontend/src/stores/message.ts` - 私聊消息管理
- `frontend/src/components/chat/PrivateChat.vue` - 私聊界面

---

#### Code: 9 - 撤回消息

**消息示例**：
```json
{
  "code": 9,
  "tid": "1704441025456",
  "userid": "abc123"
}
```

**处理逻辑**：
- 上游处理函数：`fun_msgrevoke(json)`
- 本地行为：
  - 根据tid找到对应消息
  - 将消息内容替换为 `"已撤回"`
  - 移除图片/视频预览（如果有）
- 限制：只能撤回自己发送的最新消息

**显示效果**：
```
[已撤回]
```

**相关代码**：
- 后端API：`/api/revokeMessage`
- 前端处理：`frontend/src/composables/useWebSocket.ts` - 撤回逻辑

---

### 系统通知类 (11, 12, 20 ~ 22)

#### Code: 11 - 全体公告

**消息示例**：
```json
{
  "code": 11,
  "content": "系统维护通知：今晚22:00-24:00进行服务器升级",
  "time": "14:30:25"
}
```

**处理逻辑**：
- 上游处理函数：`fun_NoticeShow(json)`
- 本地行为：
  - 弹出模态框显示公告内容
  - 支持HTML格式（如图片、链接）
  - 显示关注公众号二维码（如果配置）
- 使用场景：系统重要通知、活动公告

**相关代码**：
- `frontend/src/components/common/NoticeModal.vue` - 公告弹窗

---

#### Code: 12 - 连接服务器成功

**消息示例**：
```json
{
  "code": 12,
  "content": "连接服务器成功"
}
```

**处理逻辑**：
- 上游处理函数：`Con_Succ(json.content)`
- 本地行为：
  - **仅通过Toast提示显示**
  - **不添加到聊天记录**
  - 触发后续的用户签到、加载历史记录等操作

**本地过滤**：在 `useWebSocket.ts:161` 中被过滤

```typescript
if (code === 12 || code === 13 || code === 16 || code === 19) {
  show(data.content)  // 仅toast显示
  return  // 不添加到聊天记录
}
```

---

#### Code: 20 - 弹幕

**消息示例**：
```json
{
  "code": 20,
  "content": "这是一条弹幕消息",
  "userid": "abc123",
  "nickname": "用户A"
}
```

**处理逻辑**：
- 上游处理函数：`fun_barrager(json)`
- 本地行为：
  - 以弹幕动画形式飘过屏幕
  - 不添加到聊天记录（特殊展示）
  - 支持自定义弹幕颜色和速度

---

#### Code: 21 - 违规禁止访问

**消息示例**：
```json
{
  "code": 21,
  "content": "您的账号因违规被封禁，封禁时间至：2026-01-10 14:30:25",
  "reason": "发送违规内容"
}
```

**处理逻辑**：
- 上游处理函数：`limitTips(json)`
- 本地行为：
  - 显示封禁提示弹窗
  - 禁用消息发送功能
  - 可能强制断开连接

---

#### Code: 22 - 举报成功反馈

**消息示例**：
```json
{
  "code": 22,
  "content": "举报成功"
}
```

**处理逻辑**：
- 本地行为：直接显示Toast提示
- 提示内容：
  ```
  【举报成功】
  感谢您的反馈！
  已将对方列入黑名单
  您已不会再收到其所发信息
  ```

---

### 随机匹配类 (13 ~ 16)

#### Code: 13 - 进入随机好友队列

**消息示例**：
```json
{
  "code": 13,
  "content": "已进入随机好友队列，正在为您匹配..."
}
```

**处理逻辑**：
- 上游处理函数：`random_queue_get_random(json.content)`
- 本地行为：
  - **仅通过Toast提示显示**
  - **不添加到聊天记录**
  - 显示匹配进度动画

**本地过滤**：在 `useWebSocket.ts:161` 中被过滤

---

#### Code: 14 - 退出随机好友队列 ⚠️

**消息示例**：
```json
{
  "code": 14,
  "content": "已退出随机好友队列"
}
```

**处理逻辑**：
- 上游处理函数：`random_queue_get_random_cancel(json.content)`
- 本地行为：
  - ⚠️ **会显示在聊天记录中**（设计缺陷）
  - 应该像code=13一样只显示Toast

**已知问题**：
> **设计不一致**：code=13（进入队列）仅Toast提示，但code=14（退出队列）会显示在聊天记录中。
> **原因**：code=14未被加入系统消息过滤列表。
> **建议**：在 `useWebSocket.ts:161` 中添加 `code === 14` 到过滤条件。

---

#### Code: 15 - 匹配成功，获取随机好友ID

**消息示例**：
```json
{
  "code": 15,
  "userid": "xyz789",
  "nickname": "随机好友",
  "sex": "男",
  "address": "广东省广州市"
}
```

**处理逻辑**：
- 上游处理函数：`random_queue_get_random_getID(json)`
- 本地行为：
  - 自动打开与匹配用户的私聊窗口
  - 添加该用户到好友列表
  - 显示匹配成功提示

**相关代码**：
- `frontend/src/composables/useRandomMatch.ts` - 随机匹配逻辑

---

#### Code: 16 - 获取随机好友 - 对方离开

**消息示例**：
```json
{
  "code": 16,
  "content": "对方已离开，匹配已结束"
}
```

**处理逻辑**：
- 上游处理函数：`random_queue_get_random_Out()`
- 本地行为：
  - **仅通过Toast提示显示**
  - **不添加到聊天记录**
  - 关闭当前匹配会话

**本地过滤**：在 `useWebSocket.ts:161` 中被过滤

---

### 输入状态类 (18 ~ 19)

#### Code: 18 - 对方正在输入

**消息示例**：
```json
{
  "code": 18,
  "userid": "abc123"
}
```

**处理逻辑**：
- 上游处理函数：`fun_inputStatusOn()`
- 本地行为：
  - 在聊天界面显示 "对方正在输入..." 提示
  - 显示输入动画（三个点跳动）
  - **不添加到聊天记录**

**相关代码**：
- `frontend/src/components/chat/TypingIndicator.vue` - 输入提示组件

---

#### Code: 19 - 停止输入提示

**消息示例**：
```json
{
  "code": 19,
  "userid": "abc123"
}
```

**处理逻辑**：
- 上游处理函数：`fun_inputStatusOff()`
- 本地行为：
  - 隐藏 "对方正在输入..." 提示
  - **仅通过Toast提示（可选）**
  - **不添加到聊天记录**

**本地过滤**：在 `useWebSocket.ts:161` 中被过滤

---

### VIP管理类 (130 ~ 139)

所有VIP相关code统一使用处理函数：`random_queue_get_viprandom(json)`

#### Code: 130 - VIP激活码已过期

**消息示例**：
```json
{
  "code": 130,
  "content": "您的VIP激活码已过期"
}
```

**本地行为**：
- 清空本地VIP激活码缓存
- 隐藏VIP标识
- 提示用户重新激活

---

#### Code: 131 - VIP激活码不存在

**消息示例**：
```json
{
  "code": 131,
  "content": "VIP激活码不存在或无效"
}
```

**本地行为**：
- 显示错误提示
- 要求用户检查激活码

---

#### Code: 132 - VIP激活码已激活

**消息示例**：
```json
{
  "code": 132,
  "content": "该激活码已被使用"
}
```

**本地行为**：
- 显示已激活提示
- 防止重复激活

---

#### Code: 133 - VIP验证通过，进入VIP匹配队列

**消息示例**：
```json
{
  "code": 133,
  "content": "VIP激活码验证通过！已进入VIP匹配队列"
}
```

**本地行为**：
- 显示VIP标识
- 进入VIP专属匹配队列
- 保存VIP激活码到Cookie

---

#### Code: 134 - 网络异常 [E001]

**消息示例**：
```json
{
  "code": 134,
  "content": "网络异常，请稍后重试 [E001]"
}
```

**本地行为**：
- 显示网络错误提示
- 建议用户检查网络连接

---

#### Code: 135 - VIP激活码防止多人同时使用

**消息示例**：
```json
{
  "code": 135,
  "content": "该激活码正在被其他用户使用"
}
```

**本地行为**：
- 显示并发使用警告
- 防止激活码共享

---

#### Code: 136 - 保留

**说明**：保留code，暂无业务逻辑

---

#### Code: 137 - VIP激活码不能为空

**消息示例**：
```json
{
  "code": 137,
  "content": "请输入VIP激活码"
}
```

**本地行为**：
- 输入验证失败提示
- 聚焦到激活码输入框

---

#### Code: 138 - 首次激活并已进入VIP匹配队列

**消息示例**：
```json
{
  "code": 138,
  "content": "激活成功！已进入VIP匹配队列",
  "vipExpireTime": "2026-02-05 14:30:25"
}
```

**本地行为**：
- 首次激活成功提示
- 显示VIP到期时间
- 自动进入VIP匹配

---

#### Code: 139 - 已进入VIP匹配队列

**消息示例**：
```json
{
  "code": 139,
  "content": "已进入VIP匹配队列"
}
```

**本地行为**：
- 再次进入VIP匹配提示
- 显示VIP匹配进度

---

## 本地消息过滤规则

### 过滤代码位置

文件：`frontend/src/composables/useWebSocket.ts:160-173`

**最新修改（2026-01-05）**：

```typescript
// Code=12 单独处理（保留Toast提示）
if (code === 12) {
  console.log('连接成功提示:', data)
  if (data.content) {
    show(data.content)
  }
  return
}

// Code=13, 14, 16, 19 静默处理（不Toast，不加入聊天记录）
if (code === 13 || code === 14 || code === 16 || code === 19) {
  console.log('系统消息（静默处理）[code=' + code + ']:', data)
  return
}
```

### 当前过滤的Code

| Code | 消息类型 | 过滤方式 | 说明 |
|------|---------|---------|------|
| 12 | 连接服务器成功 | Toast显示 ✅ | 重要的连接状态反馈，保留Toast提示 |
| 13 | 进入随机队列 | **完全静默** ❌ | 系统状态通知，不Toast，不加入聊天记录 |
| 14 | 退出随机队列 | **完全静默** ❌ | **已修复**：与code=13保持一致，完全静默 |
| 16 | 对方离开匹配 | **完全静默** ❌ | 系统状态通知，不Toast，不加入聊天记录 |
| 19 | 停止输入提示 | **完全静默** ❌ | 输入状态变化，不Toast，不加入聊天记录 |

### ~~未过滤但应该过滤的Code~~

~~| Code | 消息类型 | 问题 |~~
~~|------|---------|------|~~
~~| **14** | **退出随机队列** | **与code=13行为不一致，应该加入过滤列表** |~~

**✅ 已修复（2026-01-05）**：code=14已加入静默处理列表

---

## ~~已知问题~~

### ~~⚠️ 问题1：Code=14的设计不一致~~

**✅ 已修复（2026-01-05）**

~~**问题描述**：~~
~~- code=13（进入队列）仅显示Toast，不加入聊天记录~~
~~- **code=14（退出队列）会显示在聊天记录中** ❌~~
~~- code=16（对方离开）仅显示Toast，不加入聊天记录~~

~~**影响**：~~
~~- 用户会在聊天窗口看到"已退出随机好友队列"消息，但看不到"已进入随机好友队列"~~
~~- 行为不一致，体验不好~~

**修复方案（已实施）**：

修改 `frontend/src/composables/useWebSocket.ts:160-173`，将code=12单独处理（保留Toast），code=13/14/16/19统一静默处理：

```typescript
// Code=12 单独处理（保留Toast提示）
if (code === 12) {
  console.log('连接成功提示:', data)
  if (data.content) {
    show(data.content)
  }
  return
}

// Code=13, 14, 16, 19 静默处理（不Toast，不加入聊天记录）
if (code === 13 || code === 14 || code === 16 || code === 19) {
  console.log('系统消息（静默处理）[code=' + code + ']:', data)
  return
}
```

**修复效果**：
- ✅ Code=12：保留Toast提示（连接成功反馈）
- ✅ Code=13：完全静默（进入队列）
- ✅ Code=14：完全静默（退出队列）- **与code=13行为一致**
- ✅ Code=16：完全静默（对方离开）
- ✅ Code=19：完全静默（停止输入）

```typescript
// 修改前
if (code === 12 || code === 13 || code === 16 || code === 19) {

// 修改后（添加 code === 14）
if (code === 12 || code === 13 || code === 14 || code === 16 || code === 19) {
```

**优先级**：中等（影响用户体验，但不影响核心功能）

---

### 💡 改进建议

#### 建议1：统一系统通知消息样式

对于需要显示在聊天记录中的系统消息（如code=11公告、code=21封禁等），可以添加特殊样式：

```vue
<!-- MessageList.vue -->
<div
  class="message-item"
  :class="{
    'system-message': isSystemMessage(msg.code),
    'user-message': !isSystemMessage(msg.code)
  }"
>
  <!-- 消息内容 -->
</div>
```

```typescript
// 判断是否为系统消息
function isSystemMessage(code: number): boolean {
  return [11, 21, -2, -1].includes(code)
}
```

```css
.system-message {
  background-color: #f0f0f0;
  text-align: center;
  font-style: italic;
  color: #666;
}
```

#### 建议2：建立Code管理枚举

在类型定义中明确管理所有code：

```typescript
// frontend/src/types/websocket.ts

export enum WebSocketMessageCode {
  // 系统管理
  FORCE_DISCONNECT = -3,
  SERVER_FEEDBACK = -2,
  WARNING = -1,

  // 用户管理
  USER_LOGOUT = 3,
  USER_RENAME = 4,
  USER_UPDATE_INFO = 17,
  USER_LOGIN_INFO = 30,

  // 消息通信
  BROADCAST_MESSAGE = 5,
  BROADCAST_HISTORY = 6,
  PRIVATE_MESSAGE = 7,
  MESSAGE_REVOKE = 9,

  // 系统通知
  SYSTEM_NOTICE = 11,
  CONNECT_SUCCESS = 12,
  BARRAGE = 20,
  BANNED = 21,
  REPORT_SUCCESS = 22,

  // 随机匹配
  ENTER_QUEUE = 13,
  EXIT_QUEUE = 14,
  MATCH_SUCCESS = 15,
  MATCH_END = 16,

  // 输入状态
  TYPING_START = 18,
  TYPING_END = 19,

  // VIP管理
  VIP_EXPIRED = 130,
  VIP_NOT_EXIST = 131,
  VIP_ALREADY_USED = 132,
  VIP_SUCCESS = 133,
  // ... 其他VIP code
}

// 定义需要过滤的系统消息
export const FILTERED_SYSTEM_CODES = [
  WebSocketMessageCode.CONNECT_SUCCESS,
  WebSocketMessageCode.ENTER_QUEUE,
  WebSocketMessageCode.EXIT_QUEUE,  // 修复后加入
  WebSocketMessageCode.MATCH_END,
  WebSocketMessageCode.TYPING_END,
]
```

---

## 相关文件

### 前端核心文件

| 文件路径 | 功能说明 |
|---------|---------|
| `frontend/src/composables/useWebSocket.ts` | WebSocket连接和消息处理核心逻辑 |
| `frontend/src/stores/message.ts` | 消息存储和管理（Pinia Store） |
| `frontend/src/stores/identity.ts` | 用户身份和在线列表管理 |
| `frontend/src/types/websocket.ts` | WebSocket消息类型定义 |
| `frontend/src/types/message.ts` | 聊天消息类型定义 |
| `frontend/src/components/chat/MessageList.vue` | 聊天记录列表渲染 |
| `frontend/src/components/chat/PrivateChat.vue` | 私聊界面 |
| `frontend/src/components/common/NoticeModal.vue` | 系统公告弹窗 |
| `frontend/src/composables/useRandomMatch.ts` | 随机匹配业务逻辑 |

### 后端核心文件

| 文件路径 | 功能说明 |
|---------|---------|
| `src/main/java/com/zcw/websocket/ProxyWebSocketHandler.java` | WebSocket消息代理处理器 |
| `src/main/java/com/zcw/websocket/UpstreamWebSocketClient.java` | 上游WebSocket客户端 |
| `src/main/java/com/zcw/websocket/UpstreamWebSocketManager.java` | 上游连接池管理 |
| `src/main/java/com/zcw/websocket/ForceoutManager.java` | 强制下线管理（80秒防重连） |

### 上游参考

- **原始JS源码**：`http://v1.chat2019.cn/randomdeskry/js/randomdeskry.js?v76`
- **上游WebSocket服务器**：`ws://nmpipei.com:7007`

---

## 消息流转流程

```
┌─────────────────┐
│   上游服务器     │
│  (nmpipei.com)  │
└────────┬────────┘
         │ WebSocket
         │ 发送消息 {code, content, ...}
         ▼
┌─────────────────────────────────────┐
│  后端代理服务器 (Spring Boot)        │
│  UpstreamWebSocketClient             │
│  - 接收上游消息                       │
│  - ForceoutManager检查（code=-1）    │
│  - 转发给所有下游客户端                │
└────────┬────────────────────────────┘
         │ WebSocket
         │ 转发消息
         ▼
┌─────────────────────────────────────┐
│  前端客户端 (Vue 3)                  │
│  useWebSocket.ts                     │
│  1. ws.onmessage接收消息             │
│  2. 解析JSON：{code, content, ...}   │
│  3. 系统消息过滤检查                  │
│     if (code in [12,13,16,19])       │
│       -> 仅Toast显示                 │
│     else                             │
│       -> 继续处理                    │
│  4. 根据code分发到不同处理逻辑        │
│  5. messageStore.addMessage()        │
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  UI界面渲染                          │
│  MessageList.vue                     │
│  - 从messageStore读取消息            │
│  - 渲染聊天气泡                       │
│  - 区分自己/对方消息                  │
│  - 显示图片/视频预览                  │
└─────────────────────────────────────┘
```

---

## 版本历史

| 版本 | 日期 | 变更说明 |
|-----|------|---------|
| 1.0 | 2026-01-05 | 初始版本，整理所有WebSocket消息code |

---

## 附录

### A. 快速测试WebSocket消息

如果需要测试特定code的处理逻辑，可以使用浏览器控制台：

```javascript
// 获取当前WebSocket实例（需在useWebSocket中暴露）
const ws = window.__debug_ws

// 手动构造消息并发送
const testMessage = {
  code: 14,
  content: "测试退出队列消息"
}
ws.send(JSON.stringify(testMessage))
```

### B. 调试技巧

在 `useWebSocket.ts` 中启用详细日志：

```typescript
ws.onmessage = (event) => {
  const data = JSON.parse(event.data)
  const code = data.code

  // 添加详细日志
  console.group(`📨 收到消息 [code=${code}]`)
  console.log('原始数据:', data)
  console.log('是否会被过滤:', [12,13,16,19].includes(code))
  console.log('处理方式:', /* 根据code输出处理方式 */)
  console.groupEnd()

  // ... 原有处理逻辑
}
```

---

**文档维护者**：项目开发团队
**最后更新**：2026-01-05
**反馈渠道**：请提交Issue到项目仓库
