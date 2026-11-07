-- MySQL schema migration: 012_wanfeng_media_map
-- 记录「晚风媒体键（文件ID）→ 我方 chat2019 媒体文件名」的永久映射。
--
-- 背景：晚风媒体只有约 30 分钟有效期，过期后下载接口返回 404「文件不存在或已失效」。
-- 我们收到/发送媒体时会把字节搬到我方媒体服务器，但历史上「映射」只放在内存缓存里（30 分钟），
-- 于是缓存一过，拉历史时又要回头找晚风要文件 → 已过期 → 历史里的图片/视频整条消失。
-- 有了这张表，只要搬过一次就永久可解析。

CREATE TABLE IF NOT EXISTS wanfeng_media_map (
	wf_key VARCHAR(191) NOT NULL COMMENT '晚风媒体键（媒体地址最后一段，即文件ID）',
	remote_filename VARCHAR(512) NOT NULL COMMENT '我方媒体服务器文件名（前端 "[文件名]" 里的部分）',
	identity_id VARCHAR(64) NULL COMMENT '搬运时使用的我方身份ID（排查用）',
	created_at DATETIME NOT NULL COMMENT '创建时间',
	updated_at DATETIME NOT NULL COMMENT '更新时间',
	PRIMARY KEY (wf_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='晚风媒体与我方媒体文件名映射';
