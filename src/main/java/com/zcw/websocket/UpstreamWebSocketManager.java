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

    public UpstreamWebSocketManager(WebSocketAddressService addressService) {
        this.addressService = addressService;
    }

    /**
     * 注册下游客户端连接
     * @param userId 用户ID
     * @param session 客户端会话
     * @param signMessage 登录消息（仅在首次创建上游连接时需要）
     */
    public void registerDownstream(String userId, WebSocketSession session, String signMessage) {
        log.info("注册客户端: userId={}, sessionId={}", userId, session.getId());

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
            // 如果该用户还没有上游连接，创建一个
            if (!upstreamClients.containsKey(userId)) {
                log.info("用户 {} 首次连接，创建上游连接", userId);
                createUpstreamConnection(userId, signMessage);
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
    private void createUpstreamConnection(String userId, String signMessage) {
        try {
            // 获取上游 WebSocket 地址
            String upstreamUrl = addressService.getUpstreamWebSocketUrl();
            log.info("为用户 {} 创建上游连接: {}", userId, upstreamUrl);

            // 创建上游客户端
            UpstreamWebSocketClient client = new UpstreamWebSocketClient(upstreamUrl, userId, this);

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
        if (client != null) {
            client.close();
            log.info("上游连接已关闭: userId={}", userId);
        }
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
}
