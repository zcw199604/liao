package com.zcw.service;

import com.zcw.model.CachedUserInfo;
import com.zcw.model.CachedLastMessage;
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

    /**
     * 保存最后一条消息
     * @param message 消息信息
     */
    void saveLastMessage(CachedLastMessage message);

    /**
     * 获取最后一条消息（从某个用户的视角）
     * @param myUserId 我的用户ID
     * @param otherUserId 对方用户ID
     * @return 缓存的消息，如果不存在返回null
     */
    CachedLastMessage getLastMessage(String myUserId, String otherUserId);

    /**
     * 批量增强用户数据（补充lastMsg字段）
     * @param userList 用户数据列表
     * @param myUserId 当前登录用户ID（用于确定是"我发送的"还是"对方发送的"）
     * @return 增强后的列表
     */
    java.util.List<java.util.Map<String, Object>> batchEnrichWithLastMessage(java.util.List<java.util.Map<String, Object>> userList, String myUserId);
}
