package com.zcw.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

/**
 * 图片服务器地址管理服务
 * 统一管理上游图片服务器地址，供多个Controller和Service使用
 */
@Service
public class ImageServerService {

    private static final Logger log = LoggerFactory.getLogger(ImageServerService.class);

    /**
     * 图片服务器地址（动态获取）
     */
    private volatile String imgServerHost = "149.88.79.98:9003";

    private final RestTemplate restTemplate = new RestTemplate();

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
        this.imgServerHost = server + ":9003";
        log.info("更新图片服务器地址: {}", this.imgServerHost);
    }

    /**
     * 从上游获取图片服务器地址
     */
    public void updateFromUpstream() {
        try {
            String url = "http://v1.chat2019.cn/asmx/method.asmx/getImgServer?_=" + System.currentTimeMillis();
            String response = restTemplate.getForObject(url, String.class);
            log.info("从上游获取图片服务器地址: {}", response);
            // TODO: 解析响应并更新 imgServerHost
        } catch (Exception e) {
            log.warn("从上游获取图片服务器地址失败，使用默认值", e);
        }
    }
}
