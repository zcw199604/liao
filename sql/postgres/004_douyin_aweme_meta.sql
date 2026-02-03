-- PostgreSQL schema migration: 004_douyin_aweme_meta
-- Safe to re-run: statements use IF NOT EXISTS where possible.

-- Per-user works table (used by 收藏用户作品列表)
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS is_pinned boolean NOT NULL DEFAULT false;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS pinned_rank int NULL;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS pinned_at timestamp NULL;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS publish_at timestamp NULL;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS crawled_at timestamp NULL;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS last_seen_at timestamp NULL;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS status varchar(32) NULL;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS author_unique_id varchar(64) NULL;
ALTER TABLE douyin_favorite_user_aweme ADD COLUMN IF NOT EXISTS author_name varchar(128) NULL;

CREATE INDEX IF NOT EXISTS idx_dfua_user_pinned_publish ON douyin_favorite_user_aweme (sec_user_id, is_pinned, pinned_rank, publish_at);

-- Global aweme table (used by 作品收藏/标签系统)
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS is_pinned boolean NOT NULL DEFAULT false;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS pinned_rank int NULL;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS pinned_at timestamp NULL;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS publish_at timestamp NULL;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS crawled_at timestamp NULL;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS last_seen_at timestamp NULL;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS status varchar(32) NULL;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS author_unique_id varchar(64) NULL;
ALTER TABLE douyin_favorite_aweme ADD COLUMN IF NOT EXISTS author_name varchar(128) NULL;

CREATE INDEX IF NOT EXISTS idx_dfa_publish_at ON douyin_favorite_aweme (publish_at DESC);
