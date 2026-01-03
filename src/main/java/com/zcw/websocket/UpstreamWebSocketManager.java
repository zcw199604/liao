package com.zcw.websocket;

import com.zcw.service.WebSocketAddressService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.WebSocketSession;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.*;

/**
 * 上游 WebSocket 管理器
 * 按用户ID管理上游连接，避免重复登录
 */
@Component
public class UpstreamWebSocketManager {

    private static final Logger log = LoggerFactory.getLogger(UpstreamWebSocketManager.class);

    private final WebSocketAddressService addressService;
    private final ForceoutManager forceoutManager;

    // 用户ID -> 上游WebSocket客户端
    private final Map<String, UpstreamWebSocketClient> upstreamClients = new ConcurrentHashMap<>();

    // 用户ID -> 下游客户端列表
    private final Map<String, List<WebSocketSession>> downstreamSessions = new ConcurrentHashMap<>();

    // 延迟关闭任务映射
    private final Map<String, ScheduledFuture<?>> pendingCloseTasks = new ConcurrentHashMap<>();

    // 延迟关闭定时器
    private final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(1);

    // 延迟时间：30秒
    private static final long CLOSE_DELAY_SECONDS = 80;

    // 最大同时连接的不同身份数量
    private static final int MAX_CONCURRENT_IDENTITIES = 2;

    // 记录每个userId的创建时间戳（用于FIFO淘汰）
    private final Map<String, Long> connectionCreationTime = new ConcurrentHashMap<>();

    public UpstreamWebSocketManager(WebSocketAddressService addressService,
                                    ForceoutManager forceoutManager) {
        this.addressService = addressService;
        this.forceoutManager = forceoutManager;
    }

    /**
     * 注册下游客户端连接
     * @param userId 用户ID
     * @param session 客户端会话
     * @param signMessage 登录消息（仅在首次创建上游连接时需要）
     */
    public void registerDownstream(String userId, WebSocketSession session, String signMessage) {
        log.info("注册客户端: userId={}, sessionId={}", userId, session.getId());

        // 检查是否被forceout禁止
        if (forceoutManager.isForbidden(userId)) {
            long remainingSeconds = forceoutManager.getRemainingSeconds(userId);
            log.warn("用户 {} 被forceout禁止连接，剩余时间: {}秒", userId, remainingSeconds);

            try {
                // 发送拒绝消息（code=-4）
                String rejectMessage = String.format(
                    "{\"code\":-4,\"content\":\"由于重复登录，您的连接被暂时禁止，请%d秒后再试\",\"forceout\":true}",
                    remainingSeconds
                );
                session.sendMessage(new org.springframework.web.socket.TextMessage(rejectMessage));

                // 立即关闭连接
                session.close();
            } catch (Exception e) {
                log.error("发送拒绝消息失败", e);
            }
            return;
        }

        // 如果有待执行的关闭任务，取消它
        ScheduledFuture<?> pendingTask = pendingCloseTasks.remove(userId);
        if (pendingTask != null && !pendingTask.isDone()) {
            pendingTask.cancel(false);
            log.info("用户 {} 重新连接，取消关闭上游连接任务", userId);
        }

        // 添加到下游会话列表
        downstreamSessions.computeIfAbsent(userId, k -> new ArrayList<>()).add(session);

        // 同步块：确保上游连接创建的线程安全
        synchronized (this) {
            // 如果该用户还没有上游连接，检查是否超出限制
            if (!upstreamClients.containsKey(userId)) {

                // 检查当前活跃连接数
                int currentActiveCount = upstreamClients.size();

                if (currentActiveCount >= MAX_CONCURRENT_IDENTITIES) {
                    // 达到限制，执行FIFO淘汰
                    log.warn("已达到最大连接数限制: 当前={}, 最大={}, 将淘汰最早创建的连接",
                             currentActiveCount, MAX_CONCURRENT_IDENTITIES);

                    // 找到最早创建的userId
                    String oldestUserId = connectionCreationTime.entrySet().stream()
                        .min(Map.Entry.comparingByValue())
                        .map(Map.Entry::getKey)
                        .orElse(null);

                    if (oldestUserId != null && !oldestUserId.equals(userId)) {
                        log.warn("淘汰最早创建的连接: userId={}, 创建时间={}",
                                 oldestUserId, connectionCreationTime.get(oldestUserId));

                        // 向被淘汰的用户广播通知消息（code=-6）
                        String evictMessage = "{\"code\":-6,\"content\":\"由于新身份连接，您已被自动断开\",\"evicted\":true}";
                        broadcastToDownstream(oldestUserId, evictMessage);

                        // 延迟1秒后关闭连接（让消息先送达）
                        scheduler.schedule(() -> {
                            closeUpstreamConnection(oldestUserId);

                            // 关闭所有下游连接
                            List<WebSocketSession> sessionsToClose = downstreamSessions.get(oldestUserId);
                            if (sessionsToClose != null) {
                                List<WebSocketSession> sessionsCopy = new ArrayList<>(sessionsToClose);
                                sessionsCopy.forEach(s -> {
                                    try {
                                        if (s.isOpen()) {
                                            s.close();
                                        }
                                    } catch (Exception e) {
                                        log.error("关闭被淘汰连接的下游会话失败", e);
                                    }
                                });
                                downstreamSessions.remove(oldestUserId);
                            }

                            // 取消待执行的延迟关闭任务
                            ScheduledFuture<?> oldPendingTask = pendingCloseTasks.remove(oldestUserId);
                            if (oldPendingTask != null && !oldPendingTask.isDone()) {
                                oldPendingTask.cancel(false);
                            }

                        }, 1, TimeUnit.SECONDS);
                    }
                }

                // 创建新的上游连接
                log.info("用户 {} 首次连接，创建上游连接（当前活跃数: {}/{}）",
                         userId, upstreamClients.size() + 1, MAX_CONCURRENT_IDENTITIES);
                createUpstreamConnection(userId, signMessage);

                // 记录创建时间
                connectionCreationTime.put(userId, System.currentTimeMillis());

            } else {
                log.info("用户 {} 复用已有上游连接", userId);
            }
        }
    }

    /**
     * 注销下游客户端连接
     * @param userId 用户ID
     * @param session 客户端会话
     */
    public void unregisterDownstream(String userId, WebSocketSession session) {
        log.info("注销客户端: userId={}, sessionId={}", userId, session.getId());

        // 从下游会话列表移除
        List<WebSocketSession> sessions = downstreamSessions.get(userId);
        if (sessions != null) {
            sessions.remove(session);

            // 如果该用户已经没有下游连接了，延迟关闭上游连接
            if (sessions.isEmpty()) {
                log.info("用户 {} 的所有客户端已断开，延迟{}秒关闭上游连接", userId, CLOSE_DELAY_SECONDS);
                downstreamSessions.remove(userId);
                scheduleCloseUpstream(userId);
            }
        }
    }

    /**
     * 发送消息到上游
     * @param userId 用户ID
     * @param message 消息内容
     */
    public void sendToUpstream(String userId, String message) {
        UpstreamWebSocketClient client = upstreamClients.get(userId);

        // 检查连接是否存在且有效
        if (client == null || !client.isOpen()) {
            log.warn("用户 {} 的上游连接不存在或已关闭，尝试重新建立连接", userId);

            // 如果该用户还有下游连接，则重新创建上游连接
            if (downstreamSessions.containsKey(userId) && !downstreamSessions.get(userId).isEmpty()) {
                // 重新创建连接（不需要 sign 消息，因为不是首次登录）
                createUpstreamConnection(userId, null);

                // 获取新创建的客户端
                client = upstreamClients.get(userId);
                if (client != null) {
                    client.sendMessage(message);
                }
            } else {
                log.error("用户 {} 没有下游连接，无法重新建立上游连接", userId);
            }
        } else {
            // 连接有效，直接发送
            client.sendMessage(message);
        }
    }

    /**
     * 广播消息到该用户的所有下游客户端
     * @param userId 用户ID
     * @param message 消息内容
     */
    public void broadcastToDownstream(String userId, String message) {
        log.info("准备广播消息到用户: userId={}", userId);

        List<WebSocketSession> sessions = downstreamSessions.get(userId);
        if (sessions != null && !sessions.isEmpty()) {
            log.info("找到 {} 个下游会话", sessions.size());
            sessions.forEach(session -> {
                if (session.isOpen()) {
                    try {
                        session.sendMessage(new org.springframework.web.socket.TextMessage(message));
                        log.info("✓ 消息已转发到客户端: userId={}, sessionId={}", userId, session.getId());
                    } catch (Exception e) {
                        log.error("✗ 转发消息到客户端失败: sessionId={}", session.getId(), e);
                    }
                } else {
                    log.warn("✗ 会话已关闭，无法转发: sessionId={}", session.getId());
                }
            });
        } else {
            log.warn("✗ 用户 {} 没有下游连接，消息丢弃", userId);
        }
    }

    /**
     * 创建上游连接
     * @param userId 用户ID
     * @param signMessage 登录消息（可以为null）
     */
    protected void createUpstreamConnection(String userId, String signMessage) {
        try {
            // 获取上游 WebSocket 地址
            String upstreamUrl = addressService.getUpstreamWebSocketUrl();
            log.info("为用户 {} 创建上游连接: {}", userId, upstreamUrl);

            // 创建上游客户端
            UpstreamWebSocketClient client = createWebSocketClient(upstreamUrl, userId);

            // 如果有 sign 消息，缓存等连接建立后发送
            if (signMessage != null) {
                log.info("准备发送 sign 消息");
                client.sendMessage(signMessage);
            } else {
                log.info("重新连接，不发送 sign 消息");
            }

            // 连接到上游
            client.connect();

            upstreamClients.put(userId, client);
            log.info("上游连接已创建: userId={}", userId);

        } catch (Exception e) {
            log.error("创建上游连接失败: userId={}", userId, e);
        }
    }

    /**
     * 创建 WebSocket 客户端实例（工厂方法，用于测试 Mock）
     */
    protected UpstreamWebSocketClient createWebSocketClient(String url, String userId) throws java.net.URISyntaxException {
        return new UpstreamWebSocketClient(url, userId, this);
    }

    /**
     * 调度延迟关闭上游连接
     * @param userId 用户ID
     */
    private void scheduleCloseUpstream(String userId) {
        // 如果已有待执行的任务，先取消
        ScheduledFuture<?> existingTask = pendingCloseTasks.get(userId);
        if (existingTask != null && !existingTask.isDone()) {
            existingTask.cancel(false);
        }

        // 调度新的延迟关闭任务
        ScheduledFuture<?> task = scheduler.schedule(() -> {
            // 再次检查是否有下游连接
            List<WebSocketSession> sessions = downstreamSessions.get(userId);
            if (sessions == null || sessions.isEmpty()) {
                log.info("延迟时间到，关闭用户 {} 的上游连接", userId);
                closeUpstreamConnection(userId);
            } else {
                log.info("用户 {} 已重新连接，取消关闭上游连接", userId);
            }
            pendingCloseTasks.remove(userId);
        }, CLOSE_DELAY_SECONDS, TimeUnit.SECONDS);

        pendingCloseTasks.put(userId, task);
        log.info("已调度延迟关闭任务: userId={}, 延迟{}秒", userId, CLOSE_DELAY_SECONDS);
    }

    /**
     * 关闭上游连接
     * @param userId 用户ID
     */
    private void closeUpstreamConnection(String userId) {
        UpstreamWebSocketClient client = upstreamClients.remove(userId);
        connectionCreationTime.remove(userId); // 清理创建时间记录

        if (client != null) {
            client.close();
            log.info("上游连接已关闭: userId={}, 当前活跃数: {}/{}",
                     userId, upstreamClients.size(), MAX_CONCURRENT_IDENTITIES);
        }
    }

    /**
     * 处理forceout消息
     * 当收到上游的forceout消息时，将用户添加到禁止列表，并关闭所有连接
     * @param userId 用户ID
     * @param message forceout消息内容
     */
    public void handleForceout(String userId, String message) {
        log.warn("处理forceout: userId={}, message={}", userId, message);

        // 1. 添加到禁止列表（5分钟）
        forceoutManager.addForceoutUser(userId);

        // 2. 广播forceout消息到所有下游连接（让前端停止重连）
        broadcastToDownstream(userId, message);

        // 3. 关闭上游连接
        closeUpstreamConnection(userId);

        // 4. 延迟1秒后关闭���有下游连接（让前端先收到消息）
        scheduler.schedule(() -> {
            List<WebSocketSession> sessions = downstreamSessions.get(userId);
            if (sessions != null) {
                log.info("延迟关闭用户 {} 的 {} 个下游连接", userId, sessions.size());
                List<WebSocketSession> sessionsCopy = new ArrayList<>(sessions);
                sessionsCopy.forEach(session -> {
                    try {
                        if (session.isOpen()) {
                            session.close();
                            log.info("✓ 已关闭forceout用户的下游连接: sessionId={}", session.getId());
                        }
                    } catch (Exception e) {
                        log.error("✗ 关闭下游连接失败", e);
                    }
                });
                downstreamSessions.remove(userId);
            }
        }, 1, TimeUnit.SECONDS);

        log.warn("Forceout处理完成: userId={}, 已添加到禁止列表5分钟", userId);
    }

    /**
     * 处理上游连接断开
     * @param userId 用户ID
     */
    public void handleUpstreamDisconnect(String userId) {
        log.info("处理上游连接断开: userId={}", userId);

        // 移除已断开的上游连接
        upstreamClients.remove(userId);

        // 获取该用户的所有下游连接
        List<WebSocketSession> sessions = downstreamSessions.get(userId);
        if (sessions != null && !sessions.isEmpty()) {
            log.info("用户 {} 有 {} 个下游连接，关闭所有下游连接让前端重连", userId, sessions.size());

            // 复制列表，避免并发修改异常
            List<WebSocketSession> sessionsCopy = new ArrayList<>(sessions);

            // 关闭所有下游连接
            sessionsCopy.forEach(session -> {
                if (session.isOpen()) {
                    try {
                        session.close();
                        log.info("✓ 已关闭下游连接: sessionId={}", session.getId());
                    } catch (Exception e) {
                        log.error("✗ 关闭下游连接失败: sessionId={}", session.getId(), e);
                    }
                }
            });

            // 清理会话列表
            downstreamSessions.remove(userId);
            log.info("用户 {} 的下游连接已全部关闭，等待前端重连", userId);

        } else {
            log.info("用户 {} 已无下游连接", userId);
        }
    }

    /**
     * 断开所有WebSocket连接（包括上游和下游）
     * 用于系统管理员强制关闭所有连接
     */
    public void closeAllConnections() {
        log.info("开始断开所有WebSocket连接...");

        // 1. 断开所有上游连接
        log.info("断开 {} 个上游连接", upstreamClients.size());
        upstreamClients.forEach((userId, client) -> {
            try {
                if (client.isOpen()) {
                    client.close();
                    log.info("已断开用户 {} 的上游连接", userId);
                }
            } catch (Exception e) {
                log.error("断开上游连接失败: userId={}", userId, e);
            }
        });
        upstreamClients.clear();
        connectionCreationTime.clear(); // 清理创建时间记录

        // 2. 断开所有下游连接
        int totalDownstream = downstreamSessions.values().stream()
                .mapToInt(List::size)
                .sum();
        log.info("断开 {} 个下游连接", totalDownstream);

        downstreamSessions.forEach((userId, sessions) -> {
            // 复制列表避免并发修改
            List<WebSocketSession> sessionsCopy = new ArrayList<>(sessions);
            sessionsCopy.forEach(session -> {
                try {
                    if (session.isOpen()) {
                        session.close();
                    }
                } catch (Exception e) {
                    log.error("断开下游连接失败: userId={}", userId, e);
                }
            });
        });
        downstreamSessions.clear();

        // 3. 取消所有延迟关闭任务
        pendingCloseTasks.forEach((userId, task) -> {
            task.cancel(false);
        });
        pendingCloseTasks.clear();

        log.info("所有WebSocket连接已断开");
    }

    /**
     * 获取连接统计信息
     */
    public Map<String, Object> getConnectionStats() {
        Map<String, Object> stats = new java.util.HashMap<>();

        // 上游连接数
        int upstreamCount = upstreamClients.size();

        // 下游连接数
        int downstreamCount = downstreamSessions.values().stream()
                .mapToInt(List::size)
                .sum();

        // 活跃连接数（上游+下游）
        stats.put("active", upstreamCount + downstreamCount);
        stats.put("upstream", upstreamCount);
        stats.put("downstream", downstreamCount);
        stats.put("maxIdentities", MAX_CONCURRENT_IDENTITIES); // 最大身份数
        stats.put("availableSlots", MAX_CONCURRENT_IDENTITIES - upstreamCount); // 剩余槽位

        return stats;
    }
}
