package com.zcw.service.impl;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import com.zcw.model.CachedUserInfo;
import com.zcw.model.CachedLastMessage;
import com.zcw.service.UserInfoCacheService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import jakarta.annotation.PostConstruct;
import java.util.*;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

/**
 * Redis 实现的缓存服务 (L1: Caffeine, L2: Redis)
 */
@Service
@ConditionalOnProperty(name = "app.cache.type", havingValue = "redis")
public class RedisUserInfoCacheService implements UserInfoCacheService {

    private static final Logger log = LoggerFactory.getLogger(RedisUserInfoCacheService.class);

    @org.springframework.beans.factory.annotation.Value("${app.cache.redis.key-prefix:user:info:}")
    private String keyPrefix;

    @org.springframework.beans.factory.annotation.Value("${app.cache.redis.last-message-prefix:user:lastmsg:}")
    private String lastMessagePrefix;

    @org.springframework.beans.factory.annotation.Value("${app.cache.redis.expire-days:7}")
    private long expireDays;

    @Autowired
    private StringRedisTemplate redisTemplate;

    private final ObjectMapper objectMapper = new ObjectMapper();

    // L1 本地缓存：5分钟过期，最大10000条
    private Cache<String, Object> localCache;

    public RedisUserInfoCacheService() {
        log.info("启用 Redis 用户缓存服务 (带 L1 本地缓存)");
    }

    @PostConstruct
    public void init() {
        localCache = Caffeine.newBuilder()
                .expireAfterWrite(5, TimeUnit.MINUTES)
                .maximumSize(10000)
                .build();
    }

    @Override
    public void saveUserInfo(CachedUserInfo info) {
        if (info == null || info.getUserId() == null) return;
        
        // 1. 更新本地缓存
        localCache.put(info.getUserId(), info);

        // 2. 更新 Redis
        try {
            String key = keyPrefix + info.getUserId();
            String json = objectMapper.writeValueAsString(info);
            redisTemplate.opsForValue().set(key, json, expireDays, TimeUnit.DAYS);
            log.debug("Redis 缓存保存用户: {}", info.getUserId());
        } catch (Exception e) {
            log.error("Redis 保存失败", e);
        }
    }

    @Override
    public CachedUserInfo getUserInfo(String userId) {
        if (userId == null) return null;

        // 1. 查本地缓存
        Object cached = localCache.getIfPresent(userId);
        if (cached instanceof CachedUserInfo) {
            return (CachedUserInfo) cached;
        }

        // 2. 查 Redis
        try {
            String key = keyPrefix + userId;
            String json = redisTemplate.opsForValue().get(key);
            if (json != null) {
                CachedUserInfo info = objectMapper.readValue(json, CachedUserInfo.class);
                // 回填本地缓存
                localCache.put(userId, info);
                return info;
            }
        } catch (Exception e) {
            log.error("Redis 读取失败", e);
        }
        return null;
    }

    @Override
    public Map<String, Object> enrichUserInfo(String userId, Map<String, Object> originalData) {
        CachedUserInfo info = getUserInfo(userId);
        fillData(originalData, info);
        return originalData;
    }

    @Override
    public List<Map<String, Object>> batchEnrichUserInfo(List<Map<String, Object>> userList, String userIdKey) {
        if (userList == null || userList.isEmpty()) return userList;

        // 1. 收集所有 UserId
        Set<String> userIds = new HashSet<>();
        for (Map<String, Object> map : userList) {
            Object idVal = map.get(userIdKey);
            if (idVal != null) {
                userIds.add(String.valueOf(idVal));
            }
        }

        if (userIds.isEmpty()) return userList;

        // 2. 批量查找用户信息 (L1 -> L2)
        Map<String, CachedUserInfo> infoMap = multiGetUserInfo(userIds);

        // 3. 填充数据
        for (Map<String, Object> map : userList) {
            Object idVal = map.get(userIdKey);
            if (idVal != null) {
                String uid = String.valueOf(idVal);
                CachedUserInfo info = infoMap.get(uid);
                if (info != null) {
                    fillData(map, info);
                }
            }
        }

        return userList;
    }

    /**
     * 批量获取用户信息 (核心优化逻辑)
     */
    private Map<String, CachedUserInfo> multiGetUserInfo(Set<String> userIds) {
        Map<String, CachedUserInfo> result = new HashMap<>();
        List<String> missingInLocal = new ArrayList<>();

        // 1. 先查本地缓存
        for (String uid : userIds) {
            Object cached = localCache.getIfPresent(uid);
            if (cached instanceof CachedUserInfo) {
                result.put(uid, (CachedUserInfo) cached);
            } else {
                missingInLocal.add(uid);
            }
        }

        if (missingInLocal.isEmpty()) {
            return result;
        }

        // 2. 查 Redis (MultiGet)
        try {
            List<String> redisKeys = missingInLocal.stream()
                    .map(uid -> keyPrefix + uid)
                    .collect(Collectors.toList());

            List<String> jsonList = redisTemplate.opsForValue().multiGet(redisKeys);

            if (jsonList != null) {
                for (int i = 0; i < missingInLocal.size(); i++) {
                    String json = jsonList.get(i);
                    String uid = missingInLocal.get(i);

                    if (json != null && !json.isEmpty()) {
                        try {
                            CachedUserInfo info = objectMapper.readValue(json, CachedUserInfo.class);
                            if (info != null) {
                                result.put(uid, info);
                                // 回填本地缓存
                                localCache.put(uid, info);
                            }
                        } catch (Exception e) {
                            log.warn("解析 Redis 用户信息失败: uid={}", uid, e);
                        }
                    }
                }
            }
        } catch (Exception e) {
            log.error("Redis 批量读取失败", e);
        }

        return result;
    }

    private void fillData(Map<String, Object> data, CachedUserInfo info) {
        if (info != null) {
            data.putIfAbsent("nickname", info.getNickname());
            data.putIfAbsent("sex", info.getGender());
            data.putIfAbsent("age", info.getAge());
            data.putIfAbsent("address", info.getAddress());
        }
    }

    @Override
    public void saveLastMessage(CachedLastMessage message) {
        if (message == null) return;

        // 1. 更新本地缓存
        String cacheKey = "lastmsg_" + message.getConversationKey();
        localCache.put(cacheKey, message);

        // 2. 更新 Redis
        try {
            String key = lastMessagePrefix + message.getConversationKey();
            String json = objectMapper.writeValueAsString(message);
            redisTemplate.opsForValue().set(key, json, expireDays, TimeUnit.DAYS);
            log.debug("Redis 缓存保存最后消息: {}", message.getConversationKey());
        } catch (Exception e) {
            log.error("Redis 保存最后消息失败", e);
        }
    }

    @Override
    public CachedLastMessage getLastMessage(String myUserId, String otherUserId) {
        String conversationKey = CachedLastMessage.generateConversationKey(myUserId, otherUserId);
        String cacheKey = "lastmsg_" + conversationKey;

        // 1. 查本地缓存
        Object cached = localCache.getIfPresent(cacheKey);
        if (cached instanceof CachedLastMessage) {
            return (CachedLastMessage) cached;
        }

        // 2. 查 Redis
        try {
            String key = lastMessagePrefix + conversationKey;
            String json = redisTemplate.opsForValue().get(key);
            if (json != null) {
                CachedLastMessage msg = objectMapper.readValue(json, CachedLastMessage.class);
                localCache.put(cacheKey, msg);
                return msg;
            }
        } catch (Exception e) {
            log.error("Redis 读取最后消息失败", e);
        }
        return null;
    }

    @Override
    public List<Map<String, Object>> batchEnrichWithLastMessage(List<Map<String, Object>> userList, String myUserId) {
        if (userList == null || userList.isEmpty()) return userList;
        if (myUserId == null || myUserId.isBlank()) return userList;

        // 1. 收集所有会话 Key
        List<String> conversationKeys = new ArrayList<>();
        for (Map<String, Object> user : userList) {
            String otherUserId = extractUserId(user);
            if (otherUserId == null || otherUserId.isBlank()) {
                continue;
            }
            String key = CachedLastMessage.generateConversationKey(myUserId, otherUserId);
            if (key != null && !key.isBlank()) {
                conversationKeys.add(key);
            }
        }

        if (conversationKeys.isEmpty()) {
            return userList;
        }

        // 2. 批量查询（优化：一次 MultiGet）
        Map<String, CachedLastMessage> messageMap = multiGetLastMessages(conversationKeys);

        // 3. 填充数据
        for (Map<String, Object> user : userList) {
            String otherUserId = extractUserId(user);
            if (otherUserId == null || otherUserId.isBlank()) {
                continue;
            }
            String key = CachedLastMessage.generateConversationKey(myUserId, otherUserId);
            if (key == null || key.isBlank()) {
                continue;
            }
            CachedLastMessage msg = messageMap.get(key);

            if (msg != null) {
                String displayContent = formatLastMessage(msg, myUserId);
                user.put("lastMsg", displayContent);           // 强制覆盖，确保缓存数据优先
                user.put("lastTime", formatTime(msg.getTime())); // 强制覆盖，确保缓存数据优先
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
     * 批量获取最后消息（Redis MultiGet 优化）
     */
    private Map<String, CachedLastMessage> multiGetLastMessages(List<String> conversationKeys) {
        Map<String, CachedLastMessage> result = new HashMap<>();
        List<String> missingInLocal = new ArrayList<>();

        // 1. 先查本地缓存
        for (String key : conversationKeys) {
            String cacheKey = "lastmsg_" + key;
            Object cached = localCache.getIfPresent(cacheKey);
            if (cached instanceof CachedLastMessage) {
                result.put(key, (CachedLastMessage) cached);
            } else {
                missingInLocal.add(key);
            }
        }

        if (missingInLocal.isEmpty()) {
            return result;
        }

        // 2. 批量查询 Redis
        try {
            List<String> redisKeys = missingInLocal.stream()
                    .map(k -> lastMessagePrefix + k)
                    .collect(Collectors.toList());

            List<String> jsonList = redisTemplate.opsForValue().multiGet(redisKeys);

            if (jsonList != null) {
                for (int i = 0; i < missingInLocal.size(); i++) {
                    String json = jsonList.get(i);
                    String conversationKey = missingInLocal.get(i);

                    if (json != null && !json.isEmpty()) {
                        try {
                            CachedLastMessage msg = objectMapper.readValue(json, CachedLastMessage.class);
                            result.put(conversationKey, msg);
                            localCache.put("lastmsg_" + conversationKey, msg);
                        } catch (Exception e) {
                            log.warn("解析最后消息失败: key={}", conversationKey, e);
                        }
                    }
                }
            }
        } catch (Exception e) {
            log.error("Redis 批量读取最后消息失败", e);
        }

        return result;
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
