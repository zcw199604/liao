# 轻量迭代任务清单：mtPhoto 刷新 Token 接口补齐

> **类型:** 轻量迭代  
> **模块:** mtPhoto  
> **状态:** 已完成

## 任务

- [√] 后端：补齐 mtPhoto `/auth/refresh` 调用与响应解析（保存 `refresh_token`）
- [√] 后端：更新 `MtPhotoService` 续期策略（优先 refresh，失败回退 login；401/403 强制续期后重试一次）
- [√] 测试：补齐 refresh 成功续期与 refresh 失败回退 login 的单测覆盖
- [√] 知识库：更新 `helloagents/wiki/modules/mtphoto.md` 与 `helloagents/wiki/api.md` 的鉴权/续期说明
- [√] 知识库：更新 `helloagents/CHANGELOG.md`，并将方案包迁移到 `helloagents/history/`（更新 `helloagents/history/index.md`）
