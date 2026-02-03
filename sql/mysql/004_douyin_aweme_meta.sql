-- MySQL schema migration: 004_douyin_aweme_meta
-- Safe to re-run: duplicate column/index errors are ignored by the migrator.

-- Per-user works table (used by 收藏用户作品列表)
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN is_pinned TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否置顶（1=置顶）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN pinned_rank INT NULL COMMENT '置顶顺序（越小越靠前）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN pinned_at DATETIME NULL COMMENT '置顶时间（best-effort）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN publish_at DATETIME NULL COMMENT '发布时间（best-effort）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN crawled_at DATETIME NULL COMMENT '采集时间（最近一次抓取）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN last_seen_at DATETIME NULL COMMENT '最近一次仍可见时间（best-effort）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN status VARCHAR(32) NULL COMMENT '作品状态（normal/private/deleted/unavailable 等）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN author_unique_id VARCHAR(64) NULL COMMENT '作者抖音号（unique_id）';
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN author_name VARCHAR(128) NULL COMMENT '作者名称/昵称（快照）';

ALTER TABLE douyin_favorite_user_aweme ADD INDEX idx_dfua_user_pinned_publish (sec_user_id, is_pinned, pinned_rank, publish_at);

-- Global aweme table (used by 作品收藏/标签系统)
ALTER TABLE douyin_favorite_aweme ADD COLUMN is_pinned TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否置顶（best-effort）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN pinned_rank INT NULL COMMENT '置顶顺序（best-effort）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN pinned_at DATETIME NULL COMMENT '置顶时间（best-effort）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN publish_at DATETIME NULL COMMENT '发布时间（best-effort）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN crawled_at DATETIME NULL COMMENT '采集时间（最近一次抓取）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN last_seen_at DATETIME NULL COMMENT '最近一次仍可见时间（best-effort）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN status VARCHAR(32) NULL COMMENT '作品状态（normal/private/deleted/unavailable 等）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN author_unique_id VARCHAR(64) NULL COMMENT '作者抖音号（unique_id）';
ALTER TABLE douyin_favorite_aweme ADD COLUMN author_name VARCHAR(128) NULL COMMENT '作者名称/昵称（快照）';

ALTER TABLE douyin_favorite_aweme ADD INDEX idx_dfa_publish_at (publish_at DESC);
