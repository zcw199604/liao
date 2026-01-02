package com.zcw.model;

import jakarta.persistence.*;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;

/**
 * 媒体发送日志实体
 * 记录发送行为
 */
@Entity
@Table(name = "media_send_log", indexes = {
    @Index(name = "idx_msl_user_id", columnList = "user_id"),
    @Index(name = "idx_msl_to_user_id", columnList = "to_user_id"),
    @Index(name = "idx_msl_send_time", columnList = "send_time DESC"),
    // 复合索引用于查询特定两人的聊天图片
    // @Index(name = "idx_msl_chat", columnList = "user_id, to_user_id, send_time") 
})
public class MediaSendLog {
    
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "user_id", length = 32, nullable = false)
    private String userId;           // 发送者ID

    @Column(name = "to_user_id", length = 32, nullable = false)
    private String toUserId;         // 接收者ID

    // 关联 MediaFile 的 local_path 或 id。
    // 为了解耦和灵活性，这里存储 local_path，因为它在文件系统中是唯一的标识。
    // 也可以存储 file_md5，但 md5 碰撞虽然概率低但也存在，且不同用户可能上传相同文件但视为不同记录？
    // 在 MediaFile 中，file_md5 + user_id 基本上是唯一的（根据我们的逻辑）。
    // 这里我们存 local_path，方便关联查询。
    @Column(name = "local_path", length = 500, nullable = false)
    private String localPath; 
    
    // 冗余存储一些显示所需的信息，避免频繁联表查询（视需求而定，这里仅冗余 remote_url）
    @Column(name = "remote_url", length = 500, nullable = false)
    private String remoteUrl;

    @Column(name = "send_time", nullable = false)
    private LocalDateTime sendTime;

    @Column(name = "created_at", nullable = false, updatable = false)
    private LocalDateTime createdAt;

    @PrePersist
    protected void onCreate() {
        if (createdAt == null) createdAt = LocalDateTime.now();
        if (sendTime == null) sendTime = LocalDateTime.now();
    }

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public String getUserId() { return userId; }
    public void setUserId(String userId) { this.userId = userId; }

    public String getToUserId() { return toUserId; }
    public void setToUserId(String toUserId) { this.toUserId = toUserId; }

    public String getLocalPath() { return localPath; }
    public void setLocalPath(String localPath) { this.localPath = localPath; }

    public String getRemoteUrl() { return remoteUrl; }
    public void setRemoteUrl(String remoteUrl) { this.remoteUrl = remoteUrl; }

    public LocalDateTime getSendTime() { return sendTime; }
    public void setSendTime(LocalDateTime sendTime) { this.sendTime = sendTime; }

    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime createdAt) { this.createdAt = createdAt; }
    
    @Transient
    public String getSendTimeStr() {
        return sendTime != null ? sendTime.format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")) : null;
    }
}
