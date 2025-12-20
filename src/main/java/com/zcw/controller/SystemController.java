package com.zcw.controller;

import com.zcw.websocket.UpstreamWebSocketManager;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.Map;

/**
 * 系统管理接口
 */
@Slf4j
@RestController
@RequestMapping("/api")
public class SystemController {

    @Autowired
    private UpstreamWebSocketManager upstreamWebSocketManager;

    /**
     * 获取连接统计信息
     */
    @GetMapping("/getConnectionStats")
    public Map<String, Object> getConnectionStats() {
        log.info("获取连接统计信息");

        Map<String, Object> response = new HashMap<>();
        try {
            Map<String, Object> stats = upstreamWebSocketManager.getConnectionStats();

            response.put("code", 0);
            response.put("msg", "success");
            response.put("data", stats);
        } catch (Exception e) {
            log.error("获取连接统计失败", e);
            response.put("code", -1);
            response.put("msg", "获取统计信息失败: " + e.getMessage());
        }

        return response;
    }

    /**
     * 断开所有WebSocket连接
     */
    @PostMapping("/disconnectAllConnections")
    public Map<String, Object> disconnectAllConnections() {
        log.info("执行断开所有连接操作");

        Map<String, Object> response = new HashMap<>();
        try {
            upstreamWebSocketManager.closeAllConnections();

            response.put("code", 0);
            response.put("msg", "所有连接已断开");
        } catch (Exception e) {
            log.error("断开所有连接失败", e);
            response.put("code", -1);
            response.put("msg", "操作失败: " + e.getMessage());
        }

        return response;
    }
}
