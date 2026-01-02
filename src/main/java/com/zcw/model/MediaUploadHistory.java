package com.zcw.model;

import jakarta.persistence.*;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;

/**
 * 媒体上传历史记录数据模型
 */
@Entity
@Table(name = "media_upload_history", indexes = {
    @Index(name = "idx_user_id", columnList = "user_id"),
    @Index(name = "idx_to_user_id", columnList = "to_user_id"),
    @Index(name = "idx_remote_url", columnList = "remote_url"),
    @Index(name = "idx_upload_time", columnList = "upload_time DESC"),
    @Index(name = "idx_update_time", columnList = "update_time DESC")
    // idx_user_to_user 复合索引可以通过数据库直接创建，JPA注解定义复合索引比较繁琐，这里主要定义单列索引
})
public class MediaUploadHistory {
    
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "user_id", length = 32, nullable = false)
    private String userId;           // 上传用户ID（发送者）

    @Column(name = "to_user_id", length = 32)
    private String toUserId;         // 接收用户ID

    @Column(name = "original_filename", nullable = false)
    private String originalFilename; // 原始文件名

    @Column(name = "local_filename", nullable = false)
    private String localFilename;    // 本地存储文件名

    @Column(name = "remote_filename", nullable = false)
    private String remoteFilename;   // 上游返回的文件名

    @Column(name = "remote_url", length = 500, nullable = false)
    private String remoteUrl;        // 完整的远程访问URL

    @Column(name = "local_path", length = 500, nullable = false)
    private String localPath;        // 本地存储相对路径

    @Column(name = "file_size", nullable = false)
    private Long fileSize;           // 文件大小（字节）

    @Column(name = "file_type", length = 50, nullable = false)
    private String fileType;         // 文件MIME类型

    @Column(name = "file_extension", length = 10, nullable = false)
    private String fileExtension;    // 文件扩展名

    // 为了兼容旧代码的 String 类型时间，这里保留 String 字段，但在数据库层面可能是 datetime
    // 更好的做法是改为 LocalDateTime，但为了最小化改动，我们可以使用 String，
    // 或者我们直接把字段改为 String 类型存储（数据库也是 varchar/datetime 自动转换）
    // 鉴于旧代码中是 String, 且数据库中是 DATETIME。
    // JPA中 String 映射到 DATETIME 会有问题。
    // 策略：修改字段类型为 LocalDateTime，并在 Getter/Setter 中做转换，或者保留 String 字段但标记为 @Transient，另起一个 LocalDateTime 字段映射数据库。
    // 但最简单的方案是：直接将字段类型改为 String，数据库中如果是 DATETIME，Hibernate 可能会报错或需要转换器。
    // 观察旧代码 RowMapper：
    // java.sql.Timestamp uploadTime = rs.getTimestamp("upload_time");
    // history.setUploadTime(uploadTime.toLocalDateTime().format(DATE_FORMATTER));
    // 这说明实体类中是 String。
    
    // 方案调整：为了让 JPA 正常工作，最好将实体字段改为 LocalDateTime，
    // 然后为了兼容旧代码调用（getUploadTime 返回 String），我们保留 getUploadTime 方法名为 String 返回值，
    // 但内部字段改为 LocalDateTime。
    
    // 等等，如果我改了字段类型，Getter/Setter 签名变了，会破坏大量代码。
    // 妥协方案：字段名为 uploadTime, 类型 String。JPA 默认会将 String 映射为 VARCHAR。
    // 但数据库里是 DATETIME。
    // 我们需要使用 @Column(columnDefinition = "DATETIME") 并在存取时转换？
    // 或者，更标准的做法：使用 JPA 的 AttributeConverter。
    
    // 让我们采用 JPA 标准做法：字段类型改为 LocalDateTime，但提供兼容的 Getter/Setter（String类型）给业务逻辑用。
    // 真正映射数据库的字段改名或重构。
    // 为了不改动太多 Getter/Setter 签名，我可以使用 @PrePersist / @PreUpdate 等，或者直接修改字段类型。
    // 检查调用处：主要在 Service 和 Controller。
    // Controller 返回 JSON，Jackson 会序列化。
    // Service 中有 setUploadTime(String)。
    
    // 决定：为了 JPA 的顺滑使用，将字段类型改为 String (映射为 VARCHAR) 可能会导致数据库 schema 变更（DATETIME -> VARCHAR）。
    // 如果数据库已经是 DATETIME，Hibernate 启动时 validation 可能会失败。
    // 
    // 正确的做法是：使用 LocalDateTime 类型映射数据库的 DATETIME。
    // 同时修改 Getter/Setter 签名，并在代码中修复编译错误。
    // 鉴于这是一个 "refactor" 任务，修改类型是合理的。
    // 但是用户要求 "中文回复我" 且 "不要使用这种方式（JdbcTemplate）"，意味着他想要现代化的写法。
    // 我将把字段改为 String，但为了让 Hibernate 映射到 DATETIME，这比较麻烦。
    // 
    // 既然要引入 JPA，那就彻底一点。
    // 1. 字段改为 String，但映射时告诉 Hibernate 这是日期？不，String 映射日期不安全。
    // 2. 字段改为 LocalDateTime。
    //    public LocalDateTime getUploadTime() { ... }
    //    public void setUploadTime(LocalDateTime t) { ... }
    //    为了兼容旧代码 setUploadTime(String)，可以加一个重载方法。
    //    JSON 序列化时，LocalDateTime 默认格式可能不是 yyyy-MM-dd HH:mm:ss。需要 @JsonFormat。
    
    // 让我们试着把字段改为 String, 看看 Hibernate 是否能容忍 String <-> Datetime 的转换（通常不能）。
    // 所以我必须把字段改成 LocalDateTime。
    
    // 实际上，为了最小化影响，我可以这样做：
    // 数据库字段名：upload_time (DATETIME)
    // 实体字段名：uploadTimeRaw (LocalDateTime)
    // 业务字段名：uploadTime (String, @Transient)
    
    // 这样不需要改动 Service/Controller 的逻辑，只需要在 Entity 内部做转换。
    
    @Column(name = "upload_time", nullable = false)
    private LocalDateTime uploadTimeRaw;

    @Column(name = "update_time")
    private LocalDateTime updateTimeRaw;

    @Column(name = "send_time")
    private LocalDateTime sendTimeRaw;
    
    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAtRaw;

    @Column(name = "file_md5", length = 32)
    private String fileMd5;

    // 辅助工具
    private static final DateTimeFormatter DATE_FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss");

    public MediaUploadHistory() {
    }

    public MediaUploadHistory(String userId, String originalFilename, String localFilename,
                              String remoteFilename, String remoteUrl, String localPath,
                              Long fileSize, String fileType, String fileExtension) {
        this.userId = userId;
        this.originalFilename = originalFilename;
        this.localFilename = localFilename;
        this.remoteFilename = remoteFilename;
        this.remoteUrl = remoteUrl;
        this.localPath = localPath;
        this.fileSize = fileSize;
        this.fileType = fileType;
        this.fileExtension = fileExtension;
    }
    
    @PrePersist
    protected void onCreate() {
        if (createdAtRaw == null) {
            createdAtRaw = LocalDateTime.now();
        }
        if (uploadTimeRaw == null) {
            uploadTimeRaw = LocalDateTime.now();
        }
        if (updateTimeRaw == null) {
            updateTimeRaw = LocalDateTime.now();
        }
    }
    
    @PreUpdate
    protected void onUpdate() {
        updateTimeRaw = LocalDateTime.now();
    }

    // Getter and Setter methods

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getUserId() {
        return userId;
    }

    public void setUserId(String userId) {
        this.userId = userId;
    }

    public String getToUserId() {
        return toUserId;
    }

    public void setToUserId(String toUserId) {
        this.toUserId = toUserId;
    }

    public String getOriginalFilename() {
        return originalFilename;
    }

    public void setOriginalFilename(String originalFilename) {
        this.originalFilename = originalFilename;
    }

    public String getLocalFilename() {
        return localFilename;
    }

    public void setLocalFilename(String localFilename) {
        this.localFilename = localFilename;
    }

    public String getRemoteFilename() {
        return remoteFilename;
    }

    public void setRemoteFilename(String remoteFilename) {
        this.remoteFilename = remoteFilename;
    }

    public String getRemoteUrl() {
        return remoteUrl;
    }

    public void setRemoteUrl(String remoteUrl) {
        this.remoteUrl = remoteUrl;
    }

    public String getLocalPath() {
        return localPath;
    }

    public void setLocalPath(String localPath) {
        this.localPath = localPath;
    }

    public Long getFileSize() {
        return fileSize;
    }

    public void setFileSize(Long fileSize) {
        this.fileSize = fileSize;
    }

    public String getFileType() {
        return fileType;
    }

    public void setFileType(String fileType) {
        this.fileType = fileType;
    }

    public String getFileExtension() {
        return fileExtension;
    }

    public void setFileExtension(String fileExtension) {
        this.fileExtension = fileExtension;
    }

    public String getFileMd5() {
        return fileMd5;
    }

    public void setFileMd5(String fileMd5) {
        this.fileMd5 = fileMd5;
    }

    // Special getters/setters for time fields to maintain compatibility
    
    @Transient // 不映射到数据库
    public String getUploadTime() {
        return uploadTimeRaw != null ? uploadTimeRaw.format(DATE_FORMATTER) : null;
    }

    public void setUploadTime(String uploadTime) {
        if (uploadTime != null) {
            this.uploadTimeRaw = LocalDateTime.parse(uploadTime, DATE_FORMATTER);
        }
    }
    
    public LocalDateTime getUploadTimeRaw() {
        return uploadTimeRaw;
    }
    
    // 添加直接操作 LocalDateTime 的方法，方便内部使用
    public void setUploadTimeRaw(LocalDateTime uploadTimeRaw) {
        this.uploadTimeRaw = uploadTimeRaw;
    }

    @Transient
    public String getUpdateTime() {
        return updateTimeRaw != null ? updateTimeRaw.format(DATE_FORMATTER) : null;
    }

    public void setUpdateTime(String updateTime) {
        if (updateTime != null) {
            this.updateTimeRaw = LocalDateTime.parse(updateTime, DATE_FORMATTER);
        }
    }
    
    public LocalDateTime getUpdateTimeRaw() {
        return updateTimeRaw;
    }
    
    public void setUpdateTimeRaw(LocalDateTime updateTimeRaw) {
        this.updateTimeRaw = updateTimeRaw;
    }

    @Transient
    public String getSendTime() {
        return sendTimeRaw != null ? sendTimeRaw.format(DATE_FORMATTER) : null;
    }

    public void setSendTime(String sendTime) {
        if (sendTime != null) {
            this.sendTimeRaw = LocalDateTime.parse(sendTime, DATE_FORMATTER);
        }
    }
    
    public void setSendTimeRaw(LocalDateTime sendTimeRaw) {
        this.sendTimeRaw = sendTimeRaw;
    }

    @Transient
    public String getCreatedAt() {
        return createdAtRaw != null ? createdAtRaw.format(DATE_FORMATTER) : null;
    }

    public void setCreatedAt(String createdAt) {
        if (createdAt != null) {
            this.createdAtRaw = LocalDateTime.parse(createdAt, DATE_FORMATTER);
        }
    }

    @Override
    public String toString() {
        return "MediaUploadHistory{"
                + "id=" + id +
                ", userId='" + userId + "'"
                + ", toUserId='" + toUserId + "'"
                + ", originalFilename='" + originalFilename + "'"
                + ", localFilename='" + localFilename + "'"
                + ", remoteUrl='" + remoteUrl + "'"
                + ", fileSize=" + fileSize +
                ", uploadTime='" + getUploadTime() + "'"
                + ", sendTime='" + getSendTime() + "'"
                + '}';
    }
}