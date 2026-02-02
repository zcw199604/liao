-- MySQL schema migration: 002_legacy_sort_order
-- Safe to re-run: duplicate column/index errors are ignored by the migrator.

ALTER TABLE douyin_favorite_user_aweme ADD COLUMN sort_order INT NOT NULL DEFAULT 2147483647 COMMENT '展示顺序（越小越靠前）';
ALTER TABLE douyin_favorite_user_aweme ADD INDEX idx_dfua_user_sort_order (sec_user_id, sort_order);
ALTER TABLE douyin_favorite_user_tag ADD COLUMN sort_order INT NOT NULL DEFAULT 0 COMMENT '展示顺序（越小越靠前）';
ALTER TABLE douyin_favorite_aweme_tag ADD COLUMN sort_order INT NOT NULL DEFAULT 0 COMMENT '展示顺序（越小越靠前）';
