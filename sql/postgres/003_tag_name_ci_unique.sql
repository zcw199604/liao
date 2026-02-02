-- PostgreSQL schema migration: 003_tag_name_ci_unique
-- Ensure tag names behave like MySQL's case-insensitive collation:
-- duplicates that differ only by case should be rejected.

ALTER TABLE douyin_favorite_user_tag DROP CONSTRAINT IF EXISTS uk_dfut_name;
ALTER TABLE douyin_favorite_aweme_tag DROP CONSTRAINT IF EXISTS uk_dfat_name;

CREATE UNIQUE INDEX IF NOT EXISTS uk_dfut_name_ci ON douyin_favorite_user_tag (LOWER(name));
CREATE UNIQUE INDEX IF NOT EXISTS uk_dfat_name_ci ON douyin_favorite_aweme_tag (LOWER(name));

