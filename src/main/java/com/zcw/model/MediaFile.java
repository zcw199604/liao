package com.zcw.model;

import jakarta.persistence.*;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;

/**
 * 媒体文件实体（对应媒体库）
 * 存储文件的物理信息和元数据
 */
@Entity
@Table(name = "media_file", indexes = {
    @Index(name = "idx_mf_user_id", columnList = "user_id"),
    @Index(name = "idx_mf_file_md5", columnList = "file_md5"),
    @Index(name = "idx_mf_update_time", columnList = "update_time DESC"),
    @Index(name = "idx_mf_local_path", columnList = "local_path")
})
public class MediaFile {
    
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "user_id", length = 32, nullable = false)
    private String userId;           // 上传用户ID

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

    @Column(name = "file_md5", length = 32)
    private String fileMd5;          // 文件MD5

    @Column(name = "upload_time", nullable = false)
    private LocalDateTime uploadTime; // 首次上传时间

    @Column(name = "update_time")
    private LocalDateTime updateTime; // 最后更新时间（用于排序）

    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @PrePersist
    protected void onCreate() {
        if (createdAt == null) createdAt = LocalDateTime.now();
        if (uploadTime == null) uploadTime = LocalDateTime.now();
        if (updateTime == null) updateTime = LocalDateTime.now();
    }
    
    @PreUpdate
    protected void onUpdate() {
        // 注意：我们可能只在某些操作下更新 updateTime，而不是每次 update 都更新
        // 但为了简单起见，这里可以保持逻辑，或者在 Service 层显式设置 updateTime
    }

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public String getUserId() { return userId; }
    public void setUserId(String userId) { this.userId = userId; }

    public String getOriginalFilename() { return originalFilename; }
    public void setOriginalFilename(String originalFilename) { this.originalFilename = originalFilename; }

    public String getLocalFilename() { return localFilename; }
    public void setLocalFilename(String localFilename) { this.localFilename = localFilename; }

    public String getRemoteFilename() { return remoteFilename; }
    public void setRemoteFilename(String remoteFilename) { this.remoteFilename = remoteFilename; }

    public String getRemoteUrl() { return remoteUrl; }
    public void setRemoteUrl(String remoteUrl) { this.remoteUrl = remoteUrl; }

    public String getLocalPath() { return localPath; }
    public void setLocalPath(String localPath) { this.localPath = localPath; }

    public Long getFileSize() { return fileSize; }
    public void setFileSize(Long fileSize) { this.fileSize = fileSize; }

    public String getFileType() { return fileType; }
    public void setFileType(String fileType) { this.fileType = fileType; }

    public String getFileExtension() { return fileExtension; }
    public void setFileExtension(String fileExtension) { this.fileExtension = fileExtension; }

    public String getFileMd5() { return fileMd5; }
    public void setFileMd5(String fileMd5) { this.fileMd5 = fileMd5; }

    public LocalDateTime getUploadTime() { return uploadTime; }
    public void setUploadTime(LocalDateTime uploadTime) { this.uploadTime = uploadTime; }

    public LocalDateTime getUpdateTime() { return updateTime; }
    public void setUpdateTime(LocalDateTime updateTime) { this.updateTime = updateTime; }

    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime createdAt) { this.createdAt = createdAt; }
    
    // 兼容旧代码的 String Getter
    @Transient
    public String getUploadTimeStr() {
        return uploadTime != null ? uploadTime.format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")) : null;
    }
}
