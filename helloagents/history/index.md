# 变更历史索引

本文件记录当前 `helloagents/history/` 下已归档方案包，便于追溯和查询。

---

## 索引

| 时间戳 | 功能名称 | 类型 | 状态 | 方案包路径 |
|--------|----------|------|------|------------|
| 202605290916 | mtphoto_api_key_migration | 重构 | ✅已完成 | [history/2026-05/202605290916_mtphoto_api_key_migration](2026-05/202605290916_mtphoto_api_key_migration/) |
| 202605270350 | fix_mobile_masonry_blank | 修复 | ✅已完成 | [history/2026-05/202605270350_fix_mobile_masonry_blank](2026-05/202605270350_fix_mobile_masonry_blank/) |
| 202605241501 | cross_identity_candidate_order | 修复 | ✅已完成 | [history/2026-05/202605241501_cross_identity_candidate_order](2026-05/202605241501_cross_identity_candidate_order/) |
| 202601062010 | fix_unread_badge_list | 修复 | 已完成 | [链接](2026-01/202601062010_fix_unread_badge_list/) |
| 202601062034 | refine_unread_route_cleanup | 修复 | 已完成 | [链接](2026-01/202601062034_refine_unread_route_cleanup/) |
| 202601071248 | go_backend_rewrite | 重构 | 已完成 | [链接](2026-01/202601071248_go_backend_rewrite/) |
| 202601071533 | release_v1_0_0 | 发布 | 已完成 | [链接](2026-01/202601071533_release_v1_0_0/) |
| 202605072331 | matched_user_archive | 功能 | 已完成 | [链接](2026-05/202605072331_matched_user_archive/) |
| 202605101413 | media_preview_video_ops | 修复 | 部分完成 | [链接](2026-05/202605101413_media_preview_video_ops/) |
| 202605160822 | media_preview_video_action_rework | 重构 | 已完成 | [链接](2026-05/202605160822_media_preview_video_action_rework/) |
| 202605241257 | cross_identity_contact_handoff | 功能 | 已完成 | [链接](2026-05/202605241257_cross_identity_contact_handoff/) |

---

## 按月归档

### 2026-05
- [202605290916_mtphoto_api_key_migration](2026-05/202605290916_mtphoto_api_key_migration/) - 将 mtPhoto 上游接入迁移为 API Key、`x-api-key` 与媒体 `auth_code` query。
- [202605270350_fix_mobile_masonry_blank](2026-05/202605270350_fix_mobile_masonry_blank/) - 为媒体库补齐持久化尺寸、历史回填接口和移动端瀑布流稳定布局。
- [202605241501_cross_identity_candidate_order](2026-05/202605241501_cross_identity_candidate_order/) - 修复跨身份联系人候选顺序，保留来源身份历史/收藏原始顺序，归档仅补充。
- [202605072331_matched_user_archive](2026-05/202605072331_matched_user_archive/) - 保存匹配未聊天用户并隔离不同身份列表。
- [202605101413_media_preview_video_ops](2026-05/202605101413_media_preview_video_ops/) - 收敛聊天视频预览、媒体预览抓帧状态和视频抽帧前端门禁。
- [202605160822_media_preview_video_action_rework](2026-05/202605160822_media_preview_video_action_rework/) - 将视频预览处理动作收敛为视频工具菜单并统一保存当前帧/创建抽帧任务语义。
- [202605241257_cross_identity_contact_handoff](2026-05/202605241257_cross_identity_contact_handoff/) - 支持从其他身份的历史、收藏和本地归档候选中临时接入联系人，并在首发后刷新当前身份历史。

### 2026-01
- [202601062010_fix_unread_badge_list](2026-01/202601062010_fix_unread_badge_list/) - 修复列表页未读气泡不显示。
- [202601062034_refine_unread_route_cleanup](2026-01/202601062034_refine_unread_route_cleanup/) - 简化未读路由判定与卸载清理。
- [202601071248_go_backend_rewrite](2026-01/202601071248_go_backend_rewrite/) - 以 Go 后端替换历史服务并保持 API/WS 兼容。
- [202601071533_release_v1_0_0](2026-01/202601071533_release_v1_0_0/) - 新增 Release 工作流并发布 v1.0.0。
