# 方案归档索引

> 通过此文件快速查找历史方案
> 历史年份: [2024](_index-2024.md) | [2023](_index-2023.md) | ...

## 快速索引（当前年份）

| 时间戳 | 名称 | 类型 | 涉及模块 | 决策 | 结果 |
|--------|------|------|---------|------|------|
| 202602082347 | douyin-import-author-link-works | 新增 | Douyin/Media | douyin-import-author-link-works#D001, douyin-import-author-link-works#D002 | ✅完成 |
| 202602071149 | chat-uploadmenu-douyin-favorites | - | - | - | ✅完成 |
| 202602011533 | db-dualstack-mysql-postgres | 重构 | DB/internal/database | db-dualstack-mysql-postgres#D001, db-dualstack-mysql-postgres#D002, db-dualstack-mysql-postgres#D003, db-dualstack-mysql-postgres#D004 | ✅完成 |
| 202602011343 | douyin-cookiecloud-cookie | - | - | - | ✅完成 |
| 202601311525 | ui-theme-media-menu-fix | 修复 | Frontend/Settings/Media | ui-theme-media-menu-fix#D001, ui-theme-media-menu-fix#D002 | ✅完成 |
| 202601240917 | douyin-favorite-tags | - | - | - | ✅完成 |
| 202601230536 | feat-douyin-account-preview-gallery | 新增 | Douyin/MediaPreview | feat-douyin-account-preview-gallery#D001 | ✅完成 |
| 202601220110 | fix-favorite-removebyid-invalid-id | 修复 | Favorite | fix-favorite-removebyid-invalid-id#D001 | ✅完成 |
| 202601220044 | tests-boundary-cases | 测试 | Identity/Favorite/Douyin/MediaUpload/Chat UI | tests-boundary-cases#D001 | ✅完成 |

## 按月归档

### 2026-02
- [202602082347_douyin-import-author-link-works](./2026-02/202602082347_douyin-import-author-link-works/) - 抖音导入记录补齐作者快照，并支持媒体详情点击作者直达其全部作品
- [202602071149_chat-uploadmenu-douyin-favorites](./2026-02/202602071149_chat-uploadmenu-douyin-favorites/) - 聊天页“+”上传菜单新增“抖音收藏作者”入口，直达收藏作者列表并浏览作品
- [202602011533_db-dualstack-mysql-postgres](./2026-02/202602011533_db-dualstack-mysql-postgres/) - 数据库支持 MySQL + PostgreSQL 双栈（仅通过 `DB_URL` scheme 切换）
- [202602011343_douyin-cookiecloud-cookie](./2026-02/202602011343_douyin-cookiecloud-cookie/) - 抖音上游请求支持从 CookieCloud 懒加载获取并缓存 Cookie

### 2026-01
- [202601311525_ui-theme-media-menu-fix](./2026-01/202601311525_ui-theme-media-menu-fix/) - 修复浅色主题图片管理对比度不足与移动端弹窗头部竖排
- [202601230536_feat-douyin-account-preview-gallery](./2026-01/202601230536_feat-douyin-account-preview-gallery/) - 用户作品预览支持跨作品画廊左右滑动 + 展示作品名
- [202601220110_fix-favorite-removebyid-invalid-id](./2026-01/202601220110_fix-favorite-removebyid-invalid-id/) - `removeById` 非法 id 返回 400
- [202601220044_tests-boundary-cases](./2026-01/202601220044_tests-boundary-cases/) - 补齐前后端关键边界测试用例

## 结果状态说明
- ✅ 完成
- ⚠️ 部分完成
- ❌ 失败/中止
- ⏸ 未执行
- 🔄 已回滚
