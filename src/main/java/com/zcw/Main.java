package com.zcw;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.web.client.RestTemplate;

/**
 * Spring Boot 应用启动类
 */
@SpringBootApplication
@EnableScheduling  // 启用定时任务支持（用于ForceoutManager的定期清理）
public class Main {
    public static void main(String[] args) {
        SpringApplication.run(Main.class, args);
    }

    /**
     * 配置 RestTemplate Bean
     * 用于调用外部 HTTP 接口
     */
    @Bean
    public RestTemplate restTemplate() {
        return new RestTemplate();
    }
}
