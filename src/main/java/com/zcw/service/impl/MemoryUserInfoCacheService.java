package com.zcw.service.impl;

import com.zcw.model.CachedUserInfo;
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
}
