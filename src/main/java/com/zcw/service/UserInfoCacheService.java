package com.zcw.service;

import com.zcw.model.CachedUserInfo;
import java.util.Map;

public interface UserInfoCacheService {
    /**
     * 保存用户信息
     * @param info 用户信息
     */
    void saveUserInfo(CachedUserInfo info);

    /**
     * 获取用户信息
     * @param userId 用户ID
     * @return 缓存的信息，如果不存在返回null
     */
    CachedUserInfo getUserInfo(String userId);

    /**
     * 增强用户数据（用于列表补充信息）
     * @param userId 用户ID
     * @param originalData 原始数据Map
     * @return 增强后的数据Map
     */
    Map<String, Object> enrichUserInfo(String userId, Map<String, Object> originalData);

    /**
     * 批量增强用户数据
     * @param userList 用户数据列表，每个元素是一个Map
     * @param userIdKey Map中存储用户ID的key
     * @return 增强后的列表
     */
    java.util.List<java.util.Map<String, Object>> batchEnrichUserInfo(java.util.List<java.util.Map<String, Object>> userList, String userIdKey);
}
