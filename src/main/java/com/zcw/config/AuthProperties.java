package com.zcw.config;

import lombok.Data;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.stereotype.Component;

/**
 * 认证配置属性
 * 从 application.yml 中读取 auth 配置
 */
@Data
@Component
@ConfigurationProperties(prefix = "auth")
public class AuthProperties {
    /**
     * 访问码
     */
    private String accessCode;

    /**
     * JWT密钥
     */
    private String jwtSecret;

    /**
     * Token过期时间(小时)
     */
    private Integer tokenExpireHours = 24;
}
