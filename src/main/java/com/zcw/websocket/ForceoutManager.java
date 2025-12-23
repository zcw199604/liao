package com.zcw.websocket;

import lombok.extern.slf4j.Slf4j;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.util.Date;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Forceout管理器
 * 用于管理被强制登出的用户禁止列表
 */
@Slf4j
@Component
public class ForceoutManager {

    /**
     * 存储被forceout的用户
     * key: userId
     * value: 禁止到期时间戳（毫秒）
     */
    private final Map<String, Long> forceoutUsers = new ConcurrentHashMap<>();

    /**
     * 禁止时长：5分钟
     */
    private static final long FORCEOUT_DURATION_MS = 5 * 60 * 1000;

    /**
     * 检查用户是否被禁止连接
     * @param userId 用户ID
     * @return true=被禁止，false=允许连接
     */
    public boolean isForbidden(String userId) {
        Long expireTime = forceoutUsers.get(userId);
        if (expireTime == null) {
            return false;
        }

        // 检查是否过期
        if (System.currentTimeMillis() > expireTime) {
            forceoutUsers.remove(userId);
            log.info("用户 {} 的forceout禁止已过期，移除禁止", userId);
            log.info("📊 移除后禁止列表总数: {}", forceoutUsers.size());
            return false;
        }

        return true;
    }

    /**
     * 添加用户到禁止列表
     * @param userId 用户ID
     */
    public void addForceoutUser(String userId) {
        long expireTime = System.currentTimeMillis() + FORCEOUT_DURATION_MS;
        forceoutUsers.put(userId, expireTime);
        log.warn("用户 {} 被添加到forceout禁止列表，过期时间: {}", userId, new Date(expireTime));
        log.info("📊 当前禁止列表总数: {}", forceoutUsers.size());
    }

    /**
     * 获取剩余禁止时间（秒）
     * @param userId 用户ID
     * @return 剩余秒数，如果未被禁止则返回0
     */
    public long getRemainingSeconds(String userId) {
        Long expireTime = forceoutUsers.get(userId);
        if (expireTime == null) {
            return 0;
        }

        long remaining = (expireTime - System.currentTimeMillis()) / 1000;
        return Math.max(0, remaining);
    }

    /**
     * 定期清理过期记录
     * 每分钟执行一次
     */
    @Scheduled(fixedRate = 60000)
    public void cleanExpired() {
        long now = System.currentTimeMillis();
        int removedCount = 0;

        for (Map.Entry<String, Long> entry : forceoutUsers.entrySet()) {
            if (entry.getValue() < now) {
                forceoutUsers.remove(entry.getKey());
                removedCount++;
            }
        }

        if (removedCount > 0) {
            log.info("清理过期的forceout记录，清理数量: {}", removedCount);
        }
    }

    /**
     * 手动移除用户的禁止状态（管理功能）
     * @param userId 用户ID
     * @return true=成功移除，false=用户未被禁止
     */
    public boolean removeForceout(String userId) {
        Long removed = forceoutUsers.remove(userId);
        if (removed != null) {
            log.info("手动移除用户 {} 的forceout禁止状态", userId);
            return true;
        }
        return false;
    }

    /**
     * 清除所有被禁止的用户（管理功能）
     * @return 清除的用户数量
     */
    public int clearAllForceout() {
        int count = forceoutUsers.size();
        forceoutUsers.clear();
        log.warn("管理员清除了所有forceout禁止状态，共清除 {} 个用户", count);
        return count;
    }

    /**
     * 获取当前被禁止的用户数量
     * @return 被禁止的用户数
     */
    public int getForbiddenUserCount() {
        return forceoutUsers.size();
    }
}
