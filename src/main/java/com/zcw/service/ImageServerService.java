package com.zcw.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import jakarta.annotation.PostConstruct;

/**
 * 图片服务器地址管理服务
 * 统一管理上游图片服务器地址，供多个Controller和Service使用
 */
@Service
public class ImageServerService {

    private static final Logger log = LoggerFactory.getLogger(ImageServerService.class);

    @Value("${app.image-server.host}")
    private String defaultHost;

    @Value("${app.image-server.port}")
    private String port;

    @Value("${app.image-server.upstream-url}")
    private String upstreamUrl;

    /**
     * 图片服务器地址（动态获取）
     */
    private volatile String imgServerHost;

    private final RestTemplate restTemplate = new RestTemplate();

    @PostConstruct
    public void init() {
        // 初始化默认值
        this.imgServerHost = defaultHost + ":" + port;
        log.info("初始化图片服务器地址: {}", this.imgServerHost);
    }

    /**
     * 获取图片服务器地址
     */
    public String getImgServerHost() {
        return imgServerHost;
    }

    /**
     * 设置图片服务器地址
     *
     * @param server 服务器地址（不含端口）
     */
    public void setImgServerHost(String server) {
        this.imgServerHost = server + ":" + port;
        log.info("更新图片服务器地址: {}", this.imgServerHost);
    }

    /**
     * 从上游获取图片服务器地址
     */
    public void updateFromUpstream() {
        try {
            String url = upstreamUrl + "?_=" + System.currentTimeMillis();
            String response = restTemplate.getForObject(url, String.class);
            log.info("从上游获取图片服务器地址响应: {}", response);
            // TODO: 解析响应并更新 imgServerHost
        } catch (Exception e) {
            log.warn("从上游获取图片服务器地址失败，保持当前值: {}", imgServerHost, e);
        }
    }
}
