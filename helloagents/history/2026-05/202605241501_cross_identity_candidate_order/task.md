# 任务清单: 修复跨身份联系人候选顺序

目录: `helloagents/history/2026-05/202605241501_cross_identity_candidate_order/`

---

## 1. 后端候选排序修复
- [√] 1.1 在 `internal/app/user_history_handlers.go` 中梳理并固定 `handleGetContactCandidates` 的合并顺序，验证 why.md#需求-保留上游历史顺序-场景-历史列表原序输出
- [√] 1.2 在 `internal/app/user_history_handlers.go` 中确保收藏来源只追加未出现用户、已出现用户仅合并标签和字段，验证 why.md#需求-保留上游收藏顺序-场景-收藏列表追加输出，依赖任务1.1
- [√] 1.3 在 `internal/app/user_archive.go` 中确认 `ListContactCandidates` 仅作为归档补充排序来源；必要时补充注释说明排序语义，验证 why.md#需求-归档只做补充且排序稳定-场景-上游与归档混合，依赖任务1.2

## 2. 测试覆盖
- [√] 2.1 在 `internal/app/user_history_archive_handlers_test.go` 中新增历史列表原序测试，验证点: 返回顺序与上游历史顺序一致
- [√] 2.2 在 `internal/app/user_history_archive_handlers_test.go` 中新增收藏追加与重复用户合并测试，验证点: 收藏顺序保留、重复用户不移动位置、`sources` 包含 `favorite`
- [√] 2.3 在 `internal/app/user_history_archive_handlers_test.go` 或 `internal/app/user_archive_test.go` 中新增归档补充测试，验证点: 归档用户追加在上游候选之后且 `localArchived=true`

## 3. 前端确认
- [√] 3.1 检查 `frontend/src/components/chat/CrossIdentityContactPicker.vue` 是否存在本地二次排序；如无排序则保持不变，验证 why.md#需求-保留上游历史顺序-场景-历史列表原序输出
- [-] 3.2 如需要传递后端搜索关键词或调整展示提示，在 `frontend/src/components/chat/CrossIdentityContactPicker.vue` 中做兼容改动，依赖任务3.1
  > 备注: 本次未修改前端交互；组件未做二次排序，后端顺序修复后可直接生效。

## 4. 安全检查
- [√] 4.1 执行安全检查（按G9: JWT权限、来源身份参数校验、cookie不落日志、不新增敏感字段输出）

## 5. 文档更新
- [√] 5.1 更新 `helloagents/wiki/modules/user-history.md`，记录跨身份候选排序契约
- [√] 5.2 更新 `helloagents/wiki/modules/chat-ui.md`，记录弹窗展示顺序规则
- [√] 5.3 更新 `helloagents/CHANGELOG.md` 的 Unreleased 修复项

## 6. 验证
- [√] 6.1 执行 `go test ./...`，验证后端测试通过
- [-] 6.2 如修改前端，执行 `cd frontend && npm run build`，验证类型检查和构建通过
  > 备注: 未修改前端文件，跳过前端构建。
