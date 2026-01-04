package com.zcw.model;

import lombok.Data;
import java.io.Serializable;

/**
 * 缓存的最后一条消息
 * 用于在用户列表中显示与每个用户的最后聊天记录
 */
@Data
public class CachedLastMessage implements Serializable {
    /**
     * 会话唯一标识：userId1_userId2（按字典序排序）
     * 双向会话共享同一个key，例如：A和B的会话key统一为 "A_B"
     */
    private String conversationKey;

    /**
     * 发送方用户ID
     */
    private String fromUserId;

    /**
     * 接收方用户ID
     */
    private String toUserId;

    /**
     * 消息内容（原始格式）
     */
    private String content;

    /**
     * 消息类型：text/image/video/file
     */
    private String type;

    /**
     * 消息时间
     */
    private String time;

    /**
     * 缓存更新时间戳
     */
    private Long updateTime;

    public CachedLastMessage() {}

    /**
     * 构造函数
     * @param fromUserId 发送方ID
     * @param toUserId 接收方ID
     * @param content 消息内容
     * @param type 消息类型
     * @param time 消息时间
     */
    public CachedLastMessage(String fromUserId, String toUserId, String content, String type, String time) {
        this.fromUserId = fromUserId;
        this.toUserId = toUserId;
        this.content = content;
        this.type = type;
        this.time = time;
        this.conversationKey = generateConversationKey(fromUserId, toUserId);
        this.updateTime = System.currentTimeMillis();
    }

    /**
     * 生成会话唯一标识（双向会话共享）
     * 规则：userId1 和 userId2 按字典序排序后用下划线连接
     *
     * @param userId1 用户1的ID
     * @param userId2 用户2的ID
     * @return 会话唯一标识，如 "user1_user2"
     */
    public static String generateConversationKey(String userId1, String userId2) {
        if (userId1 == null || userId2 == null) {
            return "";
        }
        if (userId1.compareTo(userId2) < 0) {
            return userId1 + "_" + userId2;
        } else {
            return userId2 + "_" + userId1;
        }
    }
}
