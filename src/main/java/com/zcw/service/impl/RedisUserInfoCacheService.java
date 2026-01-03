package com.zcw.service.impl;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import com.zcw.model.CachedUserInfo;
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

    @org.springframework.beans.factory.annotation.Value("${app.cache.redis.expire-days:7}")
    private long expireDays;

    @Autowired
    private StringRedisTemplate redisTemplate;
    
    private final ObjectMapper objectMapper = new ObjectMapper();

    // L1 本地缓存：5分钟过期，最大10000条
    private Cache<String, CachedUserInfo> localCache;

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
        CachedUserInfo info = localCache.getIfPresent(userId);
        if (info != null) {
            return info;
        }

        // 2. 查 Redis
        try {
            String key = keyPrefix + userId;
            String json = redisTemplate.opsForValue().get(key);
            if (json != null) {
                info = objectMapper.readValue(json, CachedUserInfo.class);
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
            CachedUserInfo info = localCache.getIfPresent(uid);
            if (info != null) {
                result.put(uid, info);
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
}
