package com.zcw.service;

import com.zcw.config.ServerConfig;
import com.zcw.model.MediaUploadHistory;
import jakarta.annotation.PostConstruct;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.io.ByteArrayResource;
import org.springframework.http.*;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Service;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

import java.io.IOException;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

/**
 * 媒体上传服务
 * 负责媒体上传历史记录的数据库操作
 */
@Service
public class MediaUploadService {

    private static final Logger log = LoggerFactory.getLogger(MediaUploadService.class);

    /**
     * 日期时间格式化器
     */
    private static final DateTimeFormatter DATE_FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss");

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @Autowired
    private ServerConfig serverConfig;

    @Autowired
    private FileStorageService fileStorageService;

    @Autowired
    private ImageServerService imageServerService;

    @Autowired
    private RestTemplate restTemplate;

    /**
     * MediaUploadHistory行映射器
     */
    private final RowMapper<MediaUploadHistory> mediaUploadHistoryRowMapper = new RowMapper<MediaUploadHistory>() {
        @Override
        public MediaUploadHistory mapRow(ResultSet rs, int rowNum) throws SQLException {
            MediaUploadHistory history = new MediaUploadHistory();
            history.setId(rs.getLong("id"));
            history.setUserId(rs.getString("user_id"));
            history.setToUserId(rs.getString("to_user_id"));
            history.setOriginalFilename(rs.getString("original_filename"));
            history.setLocalFilename(rs.getString("local_filename"));
            history.setRemoteFilename(rs.getString("remote_filename"));
            history.setRemoteUrl(rs.getString("remote_url"));
            history.setLocalPath(rs.getString("local_path"));
            history.setFileSize(rs.getLong("file_size"));
            history.setFileType(rs.getString("file_type"));
            history.setFileExtension(rs.getString("file_extension"));

            java.sql.Timestamp uploadTime = rs.getTimestamp("upload_time");
            if (uploadTime != null) {
                history.setUploadTime(uploadTime.toLocalDateTime().format(DATE_FORMATTER));
            }

            java.sql.Timestamp sendTime = rs.getTimestamp("send_time");
            if (sendTime != null) {
                history.setSendTime(sendTime.toLocalDateTime().format(DATE_FORMATTER));
            }

            java.sql.Timestamp createdAt = rs.getTimestamp("created_at");
            if (createdAt != null) {
                history.setCreatedAt(createdAt.toLocalDateTime().format(DATE_FORMATTER));
            }

            return history;
        }
    };

    /**
     * 初始化服务，确保表存在
     */
    @PostConstruct
    public void init() {
        createTableIfNotExists();
        log.info("媒体上传服务初始化完成");
    }

    /**
     * 如果表不存在则创建
     */
    private void createTableIfNotExists() {
        String sql = """
            CREATE TABLE IF NOT EXISTS media_upload_history (
                id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
                user_id VARCHAR(32) NOT NULL COMMENT '上传用户ID（发送者）',
                to_user_id VARCHAR(32) COMMENT '接收用户ID（发送时填充，上传时为NULL）',
                original_filename VARCHAR(255) NOT NULL COMMENT '原始文件名',
                local_filename VARCHAR(255) NOT NULL COMMENT '本地存储文件名（UUID命名）',
                remote_filename VARCHAR(255) NOT NULL COMMENT '上游返回的文件名',
                remote_url VARCHAR(500) NOT NULL COMMENT '完整的远程访问URL',
                local_path VARCHAR(500) NOT NULL COMMENT '本地存储相对路径',
                file_size BIGINT NOT NULL COMMENT '文件大小（字节）',
                file_type VARCHAR(50) NOT NULL COMMENT '文件MIME类型',
                file_extension VARCHAR(10) NOT NULL COMMENT '文件扩展名',
                upload_time DATETIME NOT NULL COMMENT '上传时间',
                send_time DATETIME COMMENT '发送时间（实际发送给某人的时间）',
                created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
                INDEX idx_user_id (user_id),
                INDEX idx_to_user_id (to_user_id),
                INDEX idx_user_to_user (user_id, to_user_id, send_time DESC),
                INDEX idx_remote_url (remote_url),
                INDEX idx_upload_time (upload_time DESC)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='媒体上传历史记录表'
            """;
        try {
            jdbcTemplate.execute(sql);
            log.info("media_upload_history表检查/创建完成");
        } catch (Exception e) {
            log.error("创建media_upload_history表失败", e);
        }
    }

    /**
     * 保存上传记录（上传阶段）
     *
     * @param history 上传历史记录
     * @return 保存后的记录（包含生成的ID）
     */
    public MediaUploadHistory saveUploadRecord(MediaUploadHistory history) {
        String now = LocalDateTime.now().format(DATE_FORMATTER);
        history.setUploadTime(now);

        String sql = """
            INSERT INTO media_upload_history
            (user_id, to_user_id, original_filename, local_filename, remote_filename,
             remote_url, local_path, file_size, file_type, file_extension, upload_time, file_md5)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """;

        jdbcTemplate.update(sql,
                history.getUserId(),
                history.getToUserId(),
                history.getOriginalFilename(),
                history.getLocalFilename(),
                history.getRemoteFilename(),
                history.getRemoteUrl(),
                history.getLocalPath(),
                history.getFileSize(),
                history.getFileType(),
                history.getFileExtension(),
                history.getUploadTime(),
                history.getFileMd5()
        );

        log.info("保存上传记录: userId={}, filename={}", history.getUserId(), history.getOriginalFilename());
        return history;
    }

    /**
     * 记录图片发送（复制记录并填充接收方）
     *
     * @param remoteUrl  远程图片URL
     * @param fromUserId 发送者ID
     * @param toUserId   接收者ID
     * @return 新的发送记录
     */
    public MediaUploadHistory recordImageSend(String remoteUrl, String fromUserId, String toUserId) {
        // 查找原始上传记录
        MediaUploadHistory original = getByRemoteUrl(remoteUrl, fromUserId);
        if (original == null) {
            log.warn("未找到原始上传记录: remoteUrl={}, fromUserId={}", remoteUrl, fromUserId);
            return null;
        }

        // 检查是否已经记录过发送给该用户
        MediaUploadHistory existing = getExistingSendRecord(remoteUrl, fromUserId, toUserId);
        if (existing != null) {
            log.info("该媒体已经发送给该用户，更新发送时间: fromUserId={}, toUserId={}, remoteUrl={}",
                    fromUserId, toUserId, remoteUrl);
            // 更新发送时间
            updateSendTime(existing.getId());
            return existing;
        }

        // 插入新的发送记录
        String now = LocalDateTime.now().format(DATE_FORMATTER);

        String sql = """
            INSERT INTO media_upload_history
            (user_id, to_user_id, original_filename, local_filename, remote_filename,
             remote_url, local_path, file_size, file_type, file_extension, upload_time, send_time)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """;

        jdbcTemplate.update(sql,
                fromUserId,
                toUserId,
                original.getOriginalFilename(),
                original.getLocalFilename(),
                original.getRemoteFilename(),
                remoteUrl,
                original.getLocalPath(),
                original.getFileSize(),
                original.getFileType(),
                original.getFileExtension(),
                original.getUploadTime(),
                now
        );

        log.info("记录媒体发送: fromUserId={}, toUserId={}, remoteUrl={}", fromUserId, toUserId, remoteUrl);

        // 返回新记录
        return getExistingSendRecord(remoteUrl, fromUserId, toUserId);
    }

    /**
     * 更新发送时间
     */
    private void updateSendTime(Long id) {
        String now = LocalDateTime.now().format(DATE_FORMATTER);
        String sql = "UPDATE media_upload_history SET send_time = ? WHERE id = ?";
        jdbcTemplate.update(sql, now, id);
    }

    /**
     * 检查是否已存在发送记录
     */
    private MediaUploadHistory getExistingSendRecord(String remoteUrl, String fromUserId, String toUserId) {
        String sql = "SELECT * FROM media_upload_history WHERE remote_url = ? AND user_id = ? AND to_user_id = ? LIMIT 1";
        List<MediaUploadHistory> results = jdbcTemplate.query(sql, mediaUploadHistoryRowMapper, remoteUrl, fromUserId, toUserId);
        return results.isEmpty() ? null : results.get(0);
    }

    /**
     * 根据远程URL查询原始上传记录
     *
     * @param remoteUrl 远程图片URL
     * @param userId    用户ID
     * @return 原始上传记录（to_user_id为NULL的记录）
     */
    public MediaUploadHistory getByRemoteUrl(String remoteUrl, String userId) {
        String sql = "SELECT * FROM media_upload_history WHERE remote_url = ? AND user_id = ? AND to_user_id IS NULL LIMIT 1";
        List<MediaUploadHistory> results = jdbcTemplate.query(sql, mediaUploadHistoryRowMapper, remoteUrl, userId);
        return results.isEmpty() ? null : results.get(0);
    }

    /**
     * 查询用户上传的所有图片（分页）
     *
     * @param userId     用户ID
     * @param page       页码（从1开始）
     * @param pageSize   每页数量
     * @param hostHeader HTTP请求的Host头
     * @return 上传历史列表（remoteUrl字段已转换为本地访问URL）
     */
    public List<MediaUploadHistory> getUserUploadHistory(String userId, int page, int pageSize, String hostHeader) {
        if (page < 1) page = 1;
        int offset = (page - 1) * pageSize;

        String sql = "SELECT * FROM media_upload_history WHERE user_id = ? AND to_user_id IS NULL ORDER BY upload_time DESC LIMIT ? OFFSET ?";
        List<MediaUploadHistory> list = jdbcTemplate.query(sql, mediaUploadHistoryRowMapper, userId, pageSize, offset);

        // 转换每个对象的 remoteUrl 为本地URL
        list.forEach(history -> {
            String localUrl = convertToLocalUrl(history.getLocalPath(), hostHeader);
            if (localUrl != null && !localUrl.isEmpty()) {
                history.setRemoteUrl(localUrl);
            }
        });

        return list;
    }

    /**
     * 查询用户发给特定对方的图片（分页）
     *
     * @param fromUserId 发送者ID
     * @param toUserId   接收者ID
     * @param page       页码（从1开始）
     * @param pageSize   每页数量
     * @param hostHeader HTTP请求的Host头
     * @return 发送历史列表（remoteUrl字段已转换为本地访问URL）
     */
    public List<MediaUploadHistory> getUserSentImages(String fromUserId, String toUserId, int page, int pageSize, String hostHeader) {
        if (page < 1) page = 1;
        int offset = (page - 1) * pageSize;

        String sql = "SELECT * FROM media_upload_history WHERE user_id = ? AND to_user_id = ? ORDER BY send_time DESC LIMIT ? OFFSET ?";
        List<MediaUploadHistory> list = jdbcTemplate.query(sql, mediaUploadHistoryRowMapper, fromUserId, toUserId, pageSize, offset);

        // 转换每个对象的 remoteUrl 为本地URL
        list.forEach(history -> {
            String localUrl = convertToLocalUrl(history.getLocalPath(), hostHeader);
            if (localUrl != null && !localUrl.isEmpty()) {
                history.setRemoteUrl(localUrl);
            }
        });

        return list;
    }

    /**
     * 统计用户上传总数
     *
     * @param userId 用户ID
     * @return 上传总数
     */
    public int getUserUploadCount(String userId) {
        String sql = "SELECT COUNT(*) FROM media_upload_history WHERE user_id = ? AND to_user_id IS NULL";
        Integer count = jdbcTemplate.queryForObject(sql, Integer.class, userId);
        return count != null ? count : 0;
    }

    /**
     * 统计发送给特定用户的图片数
     *
     * @param fromUserId 发送者ID
     * @param toUserId   接收者ID
     * @return 发送总数
     */
    public int getUserSentCount(String fromUserId, String toUserId) {
        String sql = "SELECT COUNT(*) FROM media_upload_history WHERE user_id = ? AND to_user_id = ?";
        Integer count = jdbcTemplate.queryForObject(sql, Integer.class, fromUserId, toUserId);
        return count != null ? count : 0;
    }

    /**
     * 将本地路径转换为本地访问URL
     *
     * @param localPath  本地相对路径
     * @param hostHeader 请求的Host头
     * @return 本地访问URL
     */
    private String convertToLocalUrl(String localPath, String hostHeader) {
        if (localPath == null || localPath.isEmpty()) {
            return null;
        }

        String path = localPath.startsWith("/") ? localPath : "/" + localPath;

        String host = (hostHeader != null && !hostHeader.isEmpty())
                ? hostHeader
                : "localhost:" + serverConfig.getServerPort();

        return "http://" + host + "/upload" + path;
    }

    /**
     * 批量转换URL列表
     *
     * @param localPaths 本地路径列表
     * @param hostHeader 请求的Host头
     * @return 本地访问URL列表
     */
    private List<String> convertToLocalUrls(List<String> localPaths, String hostHeader) {
        if (localPaths == null || localPaths.isEmpty()) {
            return new ArrayList<>();
        }
        return localPaths.stream()
                .map(path -> convertToLocalUrl(path, hostHeader))
                .filter(url -> url != null)
                .collect(Collectors.toList());
    }

    /**
     * 公共方法：将路径列表转换为本地URL列表
     * 供Controller层调用
     */
    public List<String> convertPathsToLocalUrls(List<String> localPaths, String hostHeader) {
        return convertToLocalUrls(localPaths, hostHeader);
    }

    /**
     * 获取两个用户之间的聊天图片（双向）
     * 用于上传弹出框展示历史图片
     *
     * @param userId1    用户1
     * @param userId2    用户2
     * @param limit      返回数量
     * @param hostHeader HTTP请求的Host头
     * @return 图片本地访问URL列表（按发送时间倒序，去重）
     */
    public List<String> getChatImages(String userId1, String userId2, int limit, String hostHeader) {
        String sql = """
            SELECT local_path
            FROM media_upload_history
            WHERE ((user_id = ? AND to_user_id = ?) OR (user_id = ? AND to_user_id = ?))
              AND send_time IS NOT NULL
            GROUP BY local_path
            ORDER BY MAX(send_time) DESC
            LIMIT ?
            """;

        List<String> localPaths = jdbcTemplate.queryForList(sql, String.class,
                userId1, userId2, userId2, userId1, limit);

        // 转换为本地访问URL
        List<String> urls = convertToLocalUrls(localPaths, hostHeader);

        log.debug("查询聊天媒体: userId1={}, userId2={}, 返回{}个", userId1, userId2, urls.size());
        return urls;
    }

    /**
     * 根据 local_path 查询原始文件名
     *
     * @param localPath 本地文件路径
     * @return 原始文件名
     */
    private String getOriginalFilenameByLocalPath(String localPath) {
        String sql = "SELECT original_filename FROM media_upload_history WHERE local_path = ? LIMIT 1";
        List<String> results = jdbcTemplate.queryForList(sql, String.class, localPath);
        return results.isEmpty() ? null : results.get(0);
    }

    /**
     * 从本地文件重新上传到上游服务器
     * 用于用户点击聊天历史图片时，从本地重新上传
     *
     * @param userId     用户ID
     * @param localPath  本地文件相对路径
     * @param cookieData Cookie数据
     * @param referer    Referer头
     * @param userAgent  User-Agent头
     * @return 上游服务器响应JSON
     */
    public String reuploadLocalFile(String userId, String localPath, String cookieData, String referer, String userAgent) throws Exception {
        // 1. 读取本地文件
        byte[] fileBytes = fileStorageService.readLocalFile(localPath);
        if (fileBytes == null || fileBytes.length == 0) {
            throw new IOException("本地文件不存在或为空: " + localPath);
        }

        // 2. 从数据库获取原始文件名
        String originalFilename = getOriginalFilenameByLocalPath(localPath);
        if (originalFilename == null) {
            // 如果查不到，使用本地文件名
            originalFilename = localPath.substring(localPath.lastIndexOf("/") + 1);
        }
        final String finalOriginalFilename = originalFilename;

        // 3. 获取图片服务器地址
        String imgServerHost = imageServerService.getImgServerHost();

        // 4. 构造上传URL
        String uploadUrl = String.format(
                "http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s",
                imgServerHost, userId);

        log.info("重新上传到图片服务器: {}", uploadUrl);

        // 5. 设置请求头
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.MULTIPART_FORM_DATA);
        headers.set("Host", imgServerHost.split(":")[0]);
        headers.set("Origin", "http://v1.chat2019.cn");
        headers.set("Referer", referer);
        headers.set("User-Agent", userAgent);
        if (cookieData != null && !cookieData.isEmpty()) {
            headers.set("Cookie", cookieData);
        }

        // 6. 构造multipart请求
        MultiValueMap<String, Object> body = new LinkedMultiValueMap<>();
        ByteArrayResource fileResource = new ByteArrayResource(fileBytes) {
            @Override
            public String getFilename() {
                return finalOriginalFilename;
            }
        };
        body.add("upload_file", fileResource);

        // 7. 创建请求实体
        HttpEntity<MultiValueMap<String, Object>> requestEntity = new HttpEntity<>(body, headers);

        // 8. 调用上游接口
        ResponseEntity<String> response = restTemplate.postForEntity(uploadUrl, requestEntity, String.class);

        log.info("重新上传成功: {}", response.getBody());
        return response.getBody();
    }

    /**
     * 查询用户所有上传的图片（基于MD5去重，支持分页）
     *
     * @param userId     用户ID（当前实现不再按用户过滤）
     * @param page       页码（从1开始）
     * @param pageSize   每页数量
     * @param hostHeader Host头（用于URL转换）
     * @return 图片本地访问URL列表
     */
    public List<String> getAllUploadImages(String userId, int page, int pageSize, String hostHeader) {
        int offset = (page - 1) * pageSize;

        String sql = """
            SELECT local_path
            FROM media_upload_history
            WHERE file_md5 IS NOT NULL
              AND id IN (
                  SELECT MAX(id)
                  FROM media_upload_history
                  WHERE file_md5 IS NOT NULL
                  GROUP BY file_md5
              )
            ORDER BY upload_time DESC
            LIMIT ? OFFSET ?
            """;

        List<String> localPaths = jdbcTemplate.queryForList(sql, String.class, pageSize, offset);

        return convertToLocalUrls(localPaths, hostHeader);
    }

    /**
     * 统计用户上传的图片总数（去重后）
     *
     * @param userId 用户ID（当前实现不再按用户过滤）
     * @return 去重后的图片总数
     */
    public int getAllUploadImagesCount(String userId) {
        String sql = """
            SELECT COUNT(DISTINCT file_md5)
            FROM media_upload_history
            WHERE file_md5 IS NOT NULL
            """;

        Integer count = jdbcTemplate.queryForObject(sql, Integer.class);
        return count != null ? count : 0;
    }
}
