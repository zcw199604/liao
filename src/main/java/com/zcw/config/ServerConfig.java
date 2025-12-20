package com.zcw.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

/**
 * 服务器配置
 * 用于获取服务器端口等配置信息
 */
@Component
public class ServerConfig {

    @Value("${server.port:8080}")
    private int serverPort;

    /**
     * 获取服务器端口
     */
    public int getServerPort() {
        return serverPort;
    }
}
