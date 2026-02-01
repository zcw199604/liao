# Identity（本地身份管理）

## 目的
提供本地身份池 CRUD 能力，并维护 `last_used_at` 以支持前端身份选择/排序。

## 模块概述
- **职责:** 身份列表查询、创建/快速创建、更新、删除、选择、ID 变更（迁移）
- **状态:** ✅稳定
- **最后更新:** 2026-01-09

## 规范

### 需求: 身份列表按最近使用排序
**模块:** Identity
身份列表按 `last_used_at DESC` 排序，用于默认展示最近使用身份。

### 需求: 身份创建与参数校验
**模块:** Identity
- `name` 不能为空
- `sex` 仅允许 `男/女`
- ID 生成：32 位随机字符串（UUID 去除连字符）

### 需求: 选择身份会刷新 last_used_at
**模块:** Identity
`POST /api/selectIdentity` 在成功返回身份信息后，会更新该身份的 `last_used_at`。

### 需求: 身份 ID 变更需保证唯一性
**模块:** Identity
`POST /api/updateIdentityId`：
- 旧身份必须存在
- 新 ID 不能已存在
- 数据变更以事务完成（删除旧记录并插入新记录），`created_at` 保持不变，`last_used_at` 更新为当前时间

## API接口
### [GET] /api/getIdentityList
**描述:** 获取本地身份列表（按最近使用排序）

### [POST] /api/createIdentity
**描述:** 创建身份

### [POST] /api/quickCreateIdentity
**描述:** 快速创建身份（随机昵称 + 随机性别）

### [POST] /api/updateIdentity
**描述:** 更新身份信息

### [POST] /api/updateIdentityId
**描述:** 变更身份 ID（迁移）

### [POST] /api/deleteIdentity
**描述:** 删除身份

### [POST] /api/selectIdentity
**描述:** 选择身份并刷新 `last_used_at`

## 数据模型
### identity
详见 `helloagents/modules/data.md` 的 `identity` 表定义。

## 依赖
- `internal/app/identity.go`
- `internal/app/identity_handlers.go`
- `internal/app/schema.go`

## 变更历史
- [202601071248_go_backend_rewrite](../../history/2026-01/202601071248_go_backend_rewrite/) - Go 后端重构并对齐身份管理行为
