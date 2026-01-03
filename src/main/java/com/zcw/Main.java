package com.zcw;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

/**
 * Spring Boot 应用启动类
 */
@SpringBootApplication
@EnableScheduling  // 启用定时任务支持（用于ForceoutManager的定期清理）
public class Main {
    public static void main(String[] args) {
        SpringApplication.run(Main.class, args);
    }
}
