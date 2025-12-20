package com.zcw.model;

/**
 * 媒体上传历史记录数据模型
 */
public class MediaUploadHistory {
    private Long id;
    private String userId;           // 上传用户ID（发送者）
    private String toUserId;         // 接收用户ID
    private String originalFilename; // 原始文件名
    private String localFilename;    // 本地存储文件名
    private String remoteFilename;   // 上游返回的文件名
    private String remoteUrl;        // 完整的远程访问URL
    private String localPath;        // 本地存储相对路径
    private Long fileSize;           // 文件大小（字节）
    private String fileType;         // 文件MIME类型
    private String fileExtension;    // 文件扩展名
    private String uploadTime;       // 上传时间 yyyy-MM-dd HH:mm:ss
    private String sendTime;         // 发送时间 yyyy-MM-dd HH:mm:ss
    private String fileMd5;          // 文件MD5哈希值（用于本地去重）
    private String createdAt;        // 记录创建时间 yyyy-MM-dd HH:mm:ss

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

    public String getUploadTime() {
        return uploadTime;
    }

    public void setUploadTime(String uploadTime) {
        this.uploadTime = uploadTime;
    }

    public String getSendTime() {
        return sendTime;
    }

    public void setSendTime(String sendTime) {
        this.sendTime = sendTime;
    }

    public String getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(String createdAt) {
        this.createdAt = createdAt;
    }

    public String getFileMd5() {
        return fileMd5;
    }

    public void setFileMd5(String fileMd5) {
        this.fileMd5 = fileMd5;
    }

    @Override
    public String toString() {
        return "MediaUploadHistory{" +
                "id=" + id +
                ", userId='" + userId + '\'' +
                ", toUserId='" + toUserId + '\'' +
                ", originalFilename='" + originalFilename + '\'' +
                ", localFilename='" + localFilename + '\'' +
                ", remoteUrl='" + remoteUrl + '\'' +
                ", fileSize=" + fileSize +
                ", uploadTime='" + uploadTime + '\'' +
                ", sendTime='" + sendTime + '\'' +
                '}';
    }
}
