-- PostgreSQL schema migration: 012_wanfeng_media_map
-- 记录「晚风媒体键（文件ID）→ 我方 chat2019 媒体文件名」的永久映射。
--
-- 背景：晚风媒体只有约 30 分钟有效期，过期后下载接口返回 404「文件不存在或已失效」。
-- 我们收到/发送媒体时会把字节搬到我方媒体服务器，但历史上「映射」只放在内存缓存里（30 分钟），
-- 于是缓存一过，拉历史时又要回头找晚风要文件 → 已过期 → 历史里的图片/视频整条消失。
-- 有了这张表，只要搬过一次就永久可解析。

CREATE TABLE IF NOT EXISTS wanfeng_media_map (
	wf_key VARCHAR(191) NOT NULL,
	remote_filename VARCHAR(512) NOT NULL,
	identity_id VARCHAR(64) NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	PRIMARY KEY (wf_key)
);

COMMENT ON TABLE wanfeng_media_map IS '晚风媒体与我方媒体文件名映射';
