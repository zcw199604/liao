package com.zcw.service.impl;

import com.zcw.model.CachedUserInfo;
import com.zcw.model.CachedLastMessage;
import com.zcw.service.UserInfoCacheService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.stereotype.Service;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 内存实现的缓存服务
 */
@Service
@ConditionalOnProperty(name = "app.cache.type", havingValue = "memory", matchIfMissing = true)
public class MemoryUserInfoCacheService implements UserInfoCacheService {

    private static final Logger log = LoggerFactory.getLogger(MemoryUserInfoCacheService.class);
    private final Map<String, CachedUserInfo> cache = new ConcurrentHashMap<>();
    private final Map<String, CachedLastMessage> lastMessageCache = new ConcurrentHashMap<>();

    public MemoryUserInfoCacheService() {
        log.info("启用内存用户缓存服务");
    }

    @Override
    public void saveUserInfo(CachedUserInfo info) {
        if (info == null || info.getUserId() == null) return;
        cache.put(info.getUserId(), info);
        log.debug("内存缓存保存用户: {}", info.getUserId());
    }

    @Override
    public CachedUserInfo getUserInfo(String userId) {
        return cache.get(userId);
    }

    @Override
    public Map<String, Object> enrichUserInfo(String userId, Map<String, Object> originalData) {
        CachedUserInfo info = cache.get(userId);
        if (info != null) {
            // 补充缺失的字段
            originalData.putIfAbsent("nickname", info.getNickname());
            originalData.putIfAbsent("sex", info.getGender());
            originalData.putIfAbsent("age", info.getAge());
            originalData.putIfAbsent("address", info.getAddress());
        }
        return originalData;
    }

    @Override
    public java.util.List<Map<String, Object>> batchEnrichUserInfo(java.util.List<Map<String, Object>> userList, String userIdKey) {
        if (userList == null) return java.util.Collections.emptyList();

        for (Map<String, Object> map : userList) {
            String userId = null;
            if (map.get(userIdKey) != null) {
                userId = String.valueOf(map.get(userIdKey));
            }

            if (userId != null) {
                enrichUserInfo(userId, map);
            }
        }
        return userList;
    }

    @Override
    public void saveLastMessage(CachedLastMessage message) {
        if (message == null) return;
        lastMessageCache.put(message.getConversationKey(), message);
        log.debug("内存缓存保存最后消息: {}", message.getConversationKey());
    }

    @Override
    public CachedLastMessage getLastMessage(String myUserId, String otherUserId) {
        String key = CachedLastMessage.generateConversationKey(myUserId, otherUserId);
        return lastMessageCache.get(key);
    }

    @Override
    public java.util.List<Map<String, Object>> batchEnrichWithLastMessage(java.util.List<Map<String, Object>> userList, String myUserId) {
        if (userList == null) return java.util.Collections.emptyList();

        for (Map<String, Object> user : userList) {
            String otherUserId = extractUserId(user);
            if (otherUserId == null || otherUserId.isBlank()) {
                continue;
            }
            CachedLastMessage lastMsg = getLastMessage(myUserId, otherUserId);

            if (lastMsg != null) {
                // 根据发送方判断是"我"还是"对方"
                String displayContent = formatLastMessage(lastMsg, myUserId);
                user.putIfAbsent("lastMsg", displayContent);
                user.putIfAbsent("lastTime", formatTime(lastMsg.getTime()));
            }
        }
        return userList;
    }

    private String extractUserId(Map<String, Object> user) {
        if (user == null) {
            return null;
        }

        Object idVal = user.get("id");
        if (idVal == null) idVal = user.get("UserID");
        if (idVal == null) idVal = user.get("userid");
        if (idVal == null) idVal = user.get("userId");

        return idVal == null ? null : String.valueOf(idVal);
    }

    /**
     * 格式化最后一条消息的显示内容
     * @param msg 缓存的消息
     * @param myUserId 当前用户ID
     * @return 格式化后的显示文本
     */
    private String formatLastMessage(CachedLastMessage msg, String myUserId) {
        String prefix = msg.getFromUserId().equals(myUserId) ? "我: " : "";
        String content = msg.getContent();

        if (content == null || content.isEmpty()) {
            return prefix + "[消息]";
        }

        // 判断是否是媒体路径格式 [20250104/xxx.ext]
        if (content.startsWith("[") && content.endsWith("]")) {
            String path = content.substring(1, content.length() - 1);
            // 兼容表情文本（如 [doge]）：无路径分隔符且无扩展名时，按普通文本显示
            if (!path.contains("/") && !path.contains("\\") && !path.contains(".")) {
                return prefix + content;
            }
            // 简单判断文件扩展名
            if (path.matches(".*\\.(jpg|jpeg|png|gif|bmp)$")) {
                return prefix + "[图片]";
            } else if (path.matches(".*\\.(mp4|avi|mov|wmv|flv)$")) {
                return prefix + "[视频]";
            } else if (path.matches(".*\\.(mp3|wav|aac|flac)$")) {
                return prefix + "[音频]";
            } else {
                return prefix + "[文件]";
            }
        }

        // 文本消息截断显示
        if (content.length() > 30) {
            return prefix + content.substring(0, 30) + "...";
        }
        return prefix + content;
    }

    /**
     * 格式化时间显示
     * @param timeStr 时间字符串
     * @return 格式化后的时间
     */
    private String formatTime(String timeStr) {
        // 简单实现，直接返回原始时间
        // 可以根据需要优化为相对时间（如"刚刚"、"5分钟前"等）
        return timeStr != null ? timeStr : "刚刚";
    }
}
