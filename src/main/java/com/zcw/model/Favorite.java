package com.zcw.model;

import jakarta.persistence.*;
import lombok.Data;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;

import java.time.LocalDateTime;

/**
 * 聊天收藏数据模型
 * 用于存储用户的全局收藏（身份+聊天对象）
 */
@Entity
@Table(name = "chat_favorites", indexes = {
    @Index(name = "idx_identity_id", columnList = "identityId"),
    @Index(name = "idx_target_user_id", columnList = "targetUserId")
})
@Data
@NoArgsConstructor
@AllArgsConstructor
public class Favorite {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    /**
     * 本地身份ID（即谁收藏的）
     */
    @Column(nullable = false, length = 32)
    private String identityId;

    /**
     * 目标用户ID（被收藏的人）
     */
    @Column(nullable = false, length = 64)
    private String targetUserId;

    /**
     * 目标用户名（用于列表显示）
     */
    @Column(length = 64)
    private String targetUserName;

    /**
     * 创建时间
     */
    @Column(nullable = false)
    private LocalDateTime createTime;

    @PrePersist
    public void prePersist() {
        if (createTime == null) {
            createTime = LocalDateTime.now();
        }
    }
}
