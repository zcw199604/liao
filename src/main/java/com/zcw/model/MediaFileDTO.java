package com.zcw.model;

import java.time.LocalDateTime;

/**
 * 媒体文件数据传输对象
 * 用于向前端返回完整的文件元数据
 */
public class MediaFileDTO {
    private String url;              // 本地访问URL
    private String type;             // image | video | file
    private String localFilename;    // 本地存储文件名
    private String originalFilename; // 原始文件名
    private Long fileSize;           // 文件大小(字节)
    private String fileType;         // MIME类型
    private String fileExtension;    // 扩展名
    private LocalDateTime uploadTime;     // 首次上传时间
    private LocalDateTime updateTime;     // 最后更新时间

    // Getters and Setters
    public String getUrl() {
        return url;
    }

    public void setUrl(String url) {
        this.url = url;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public String getLocalFilename() {
        return localFilename;
    }

    public void setLocalFilename(String localFilename) {
        this.localFilename = localFilename;
    }

    public String getOriginalFilename() {
        return originalFilename;
    }

    public void setOriginalFilename(String originalFilename) {
        this.originalFilename = originalFilename;
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

    public LocalDateTime getUploadTime() {
        return uploadTime;
    }

    public void setUploadTime(LocalDateTime uploadTime) {
        this.uploadTime = uploadTime;
    }

    public LocalDateTime getUpdateTime() {
        return updateTime;
    }

    public void setUpdateTime(LocalDateTime updateTime) {
        this.updateTime = updateTime;
    }
}
