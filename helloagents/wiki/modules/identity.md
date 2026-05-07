# Identity

## 目的
维护本地匿名聊天身份池，并支持快速切换身份。

## 模块概述
- **职责:** 身份列表、创建、快速创建、更新、改 ID、删除、选择并更新最近使用时间。
- **状态:** 稳定
- **最后更新:** 2026-05-07

## 规范

### 需求: 本地身份 CRUD
**模块:** Identity  
身份数据保存在 `identity` 表，列表按 `last_used_at` 倒序展示。

#### 场景: 创建身份
- `name` 必填。
- `sex` 只能为 `男` 或 `女`。
- ID 由服务端生成 32 位随机字符串。

#### 场景: 选择身份
- 更新 `last_used_at`。
- 前端进入需要身份的页面前必须存在 `currentUser`。

## API接口
- `GET /api/getIdentityList`
- `POST /api/createIdentity`
- `POST /api/quickCreateIdentity`
- `POST /api/updateIdentity`
- `POST /api/updateIdentityId`
- `POST /api/deleteIdentity`
- `POST /api/selectIdentity`

## 数据模型
### `identity`
| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 身份 ID |
| name | VARCHAR(50) | 名字 |
| sex | VARCHAR(10) | 性别 |
| created_at | DATETIME/TIMESTAMP | 创建时间 |
| last_used_at | DATETIME/TIMESTAMP | 最近使用时间 |

## 依赖
- `internal/app/identity.go`
- `internal/app/identity_handlers.go`
- `frontend/src/api/identity.ts`
- `frontend/src/stores/identity.ts`
- `frontend/src/views/IdentityPicker.vue`
