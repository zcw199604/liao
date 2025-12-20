package com.zcw.controller;

import com.zcw.model.MediaUploadHistory;
import com.zcw.service.ImageServerService;
import com.zcw.service.MediaUploadService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.http.client.SimpleClientHttpRequestFactory;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.RestTemplate;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * 媒体历史记录接口
 * 提供图片上传历史查询和发送记录功能
 */
@RestController
@RequestMapping("/api")
public class MediaHistoryController {

    private static final Logger log = LoggerFactory.getLogger(MediaHistoryController.class);

    @Autowired
    private MediaUploadService mediaUploadService;

    @Autowired
    private ImageServerService imageServerService;

    /**
     * 记录图片发送
     * 当图片实际发送给某人时调用此接口
     *
     * @param remoteUrl  远程图片URL
     * @param fromUserId 发送者ID
     * @param toUserId   接收者ID
     * @return 记录结果
     */
    @PostMapping("/recordImageSend")
    public ResponseEntity<Map<String, Object>> recordImageSend(
            @RequestParam String remoteUrl,
            @RequestParam String fromUserId,
            @RequestParam String toUserId) {

        log.info("记录图片发送: fromUserId={}, toUserId={}, remoteUrl={}", fromUserId, toUserId, remoteUrl);

        Map<String, Object> result = new HashMap<>();

        try {
            MediaUploadHistory record = mediaUploadService.recordImageSend(remoteUrl, fromUserId, toUserId);

            if (record != null) {
                result.put("success", true);
                result.put("message", "记录成功");
                result.put("data", record);
                log.info("图片发送记录成功: id={}", record.getId());
            } else {
                result.put("success", false);
                result.put("message", "未找到原始上传记录");
                log.warn("未找到原始上传记录: remoteUrl={}, fromUserId={}", remoteUrl, fromUserId);
            }

            return ResponseEntity.ok(result);

        } catch (Exception e) {
            log.error("记录图片发送失败", e);
            result.put("success", false);
            result.put("message", "记录失败: " + e.getMessage());
            return ResponseEntity.status(500).body(result);
        }
    }

    /**
     * 查询用户上传的所有图片
     *
     * @param userId     用户ID
     * @param page       页码（从1开始）
     * @param pageSize   每页数量
     * @param hostHeader HTTP请求的Host头
     * @return 图片列表和分页信息
     */
    @GetMapping("/getUserUploadHistory")
    public ResponseEntity<Map<String, Object>> getUserUploadHistory(
            @RequestParam String userId,
            @RequestParam(defaultValue = "1") int page,
            @RequestParam(defaultValue = "20") int pageSize,
            @RequestHeader(value = "Host", required = false) String hostHeader) {

        log.info("查询用户上传历史: userId={}, page={}, pageSize={}", userId, page, pageSize);

        Map<String, Object> result = new HashMap<>();

        try {
            // 查询列表（传递hostHeader）
            List<MediaUploadHistory> list = mediaUploadService.getUserUploadHistory(userId, page, pageSize, hostHeader);

            // 查询总数
            int total = mediaUploadService.getUserUploadCount(userId);

            // 计算总页数
            int totalPages = (int) Math.ceil((double) total / pageSize);

            Map<String, Object> data = new HashMap<>();
            data.put("list", list);
            data.put("total", total);
            data.put("page", page);
            data.put("pageSize", pageSize);
            data.put("totalPages", totalPages);

            result.put("success", true);
            result.put("message", "查询成功");
            result.put("data", data);

            log.info("查询用户上传历史成功: userId={}, 返回{}条记录", userId, list.size());
            return ResponseEntity.ok(result);

        } catch (Exception e) {
            log.error("查询用户上传历史失败", e);
            result.put("success", false);
            result.put("message", "查询失败: " + e.getMessage());
            return ResponseEntity.status(500).body(result);
        }
    }

    /**
     * 查询用户发给特定对方的图片
     *
     * @param fromUserId 发送者ID
     * @param toUserId   接收者ID
     * @param page       页码（从1开始）
     * @param pageSize   每页数量
     * @param hostHeader HTTP请求的Host头
     * @return 图片列表和分页信息
     */
    @GetMapping("/getUserSentImages")
    public ResponseEntity<Map<String, Object>> getUserSentImages(
            @RequestParam String fromUserId,
            @RequestParam String toUserId,
            @RequestParam(defaultValue = "1") int page,
            @RequestParam(defaultValue = "20") int pageSize,
            @RequestHeader(value = "Host", required = false) String hostHeader) {

        log.info("查询用户发送图片: fromUserId={}, toUserId={}, page={}, pageSize={}",
                fromUserId, toUserId, page, pageSize);

        Map<String, Object> result = new HashMap<>();

        try {
            // 查询列表（传递hostHeader）
            List<MediaUploadHistory> list = mediaUploadService.getUserSentImages(fromUserId, toUserId, page, pageSize, hostHeader);

            // 查询总数
            int total = mediaUploadService.getUserSentCount(fromUserId, toUserId);

            // 计算总页数
            int totalPages = (int) Math.ceil((double) total / pageSize);

            Map<String, Object> data = new HashMap<>();
            data.put("list", list);
            data.put("total", total);
            data.put("page", page);
            data.put("pageSize", pageSize);
            data.put("totalPages", totalPages);

            result.put("success", true);
            result.put("message", "查询成功");
            result.put("data", data);

            log.info("查询用户发送图片成功: fromUserId={}, toUserId={}, 返回{}条记录",
                    fromUserId, toUserId, list.size());
            return ResponseEntity.ok(result);

        } catch (Exception e) {
            log.error("查询用户发送图片失败", e);
            result.put("success", false);
            result.put("message", "查询失败: " + e.getMessage());
            return ResponseEntity.status(500).body(result);
        }
    }

    /**
     * 查询用户上传统计信息
     *
     * @param userId 用户ID
     * @return 统计信息
     */
    @GetMapping("/getUserUploadStats")
    public ResponseEntity<Map<String, Object>> getUserUploadStats(@RequestParam String userId) {

        log.info("查询用户上传统计: userId={}", userId);

        Map<String, Object> result = new HashMap<>();

        try {
            int totalCount = mediaUploadService.getUserUploadCount(userId);

            Map<String, Object> data = new HashMap<>();
            data.put("totalCount", totalCount);

            result.put("success", true);
            result.put("message", "查询成功");
            result.put("data", data);

            log.info("查询用户上传统计成功: userId={}, totalCount={}", userId, totalCount);
            return ResponseEntity.ok(result);

        } catch (Exception e) {
            log.error("查询用户上传统计失败", e);
            result.put("success", false);
            result.put("message", "查询失败: " + e.getMessage());
            return ResponseEntity.status(500).body(result);
        }
    }

    /**
     * 获取两个用户之间的聊天图片（双向）
     * 用于上传弹出框展示历史图片
     *
     * @param userId1    用户1 ID
     * @param userId2    用户2 ID
     * @param limit      返回数量限制（默认20）
     * @param hostHeader HTTP请求的Host头（自动注入）
     * @return 图片本地访问URL列表
     */
    @GetMapping("/getChatImages")
    public ResponseEntity<List<String>> getChatImages(
            @RequestParam String userId1,
            @RequestParam String userId2,
            @RequestParam(defaultValue = "20") int limit,
            @RequestHeader(value = "Host", required = false) String hostHeader) {

        log.info("获取聊天图片: userId1={}, userId2={}, limit={}, host={}", userId1, userId2, limit, hostHeader);

        try {
            // 查询双向发送的图片并转换为本地访问URL
            List<String> imageUrls = mediaUploadService.getChatImages(userId1, userId2, limit, hostHeader);

            log.info("返回{}张聊天图片", imageUrls.size());
            return ResponseEntity.ok(imageUrls);

        } catch (Exception e) {
            log.error("获取聊天图片失败", e);
            return ResponseEntity.status(500).body(new ArrayList<>());
        }
    }

    /**
     * 重新上传历史图片到上游服务器
     * 用于用户点击聊天历史图片时，从本地文件重新上传
     *
     * @param userId     用户ID
     * @param localPath  本地文件相对路径
     * @param cookieData Cookie数据
     * @param referer    Referer头
     * @param userAgent  User-Agent头
     * @return 上游服务器返回结果
     */
    @PostMapping("/reuploadHistoryImage")
    public ResponseEntity<String> reuploadHistoryImage(
            @RequestParam String userId,
            @RequestParam String localPath,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        log.info("重新上传历史图片: userId={}, localPath={}", userId, localPath);

        try {
            // 调用 Service 层处理
            String response = mediaUploadService.reuploadLocalFile(userId, localPath, cookieData, referer, userAgent);

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            log.error("重新上传历史图片失败", e);
            String errorResponse = "{\"state\":\"ERROR\",\"msg\":\"" + e.getMessage() + "\"}";
            return ResponseEntity.status(500).body(errorResponse);
        }
    }

    /**
     * 检测图片服务器可用端口
     * 模仿上游 NetPing 逻辑，遍历端口找到可用的
     *
     * @param imgServerHost 图片服务器地址
     * @return 可用的端口号
     */
    private String detectAvailablePort(String imgServerHost) {
        // 端口优先级顺序（9系列优先，8系列备选）
        String[] ports = {"9006", "9005", "9003", "9002", "9001", "8006", "8005", "8003", "8002", "8001"};

        for (String port : ports) {
            try {
                String testUrl = "http://" + imgServerHost + ":" + port + "/useripaddressv23.js";

                // 创建专用的测试RestTemplate，设置超时
                RestTemplate testTemplate = new RestTemplate();
                SimpleClientHttpRequestFactory factory = new SimpleClientHttpRequestFactory();
                factory.setConnectTimeout(800);  // 连接超时800ms
                factory.setReadTimeout(800);     // 读取超时800ms
                testTemplate.setRequestFactory(factory);

                // 尝试访问测试URL
                testTemplate.getForEntity(testUrl, String.class);
                log.info("端口 {} 可用", port);
                return port;
            } catch (Exception e) {
                log.debug("端口 {} 不可用: {}", port, e.getMessage());
            }
        }

        log.warn("未找到可用端口，使用默认9006");
        return "9006";  // 默认端口
    }

    /**
     * 获取用户所有上传的图片（分页，MD5去重）
     *
     * @param userId     用户ID
     * @param page       页码（默认1）
     * @param pageSize   每页数量（默认20）
     * @param hostHeader Host头
     * @return 分页数据
     */
    @GetMapping("/getAllUploadImages")
    public ResponseEntity<Map<String, Object>> getAllUploadImages(
            @RequestParam String userId,
            @RequestParam(defaultValue = "1") int page,
            @RequestParam(defaultValue = "20") int pageSize,
            @RequestHeader(value = "Host", required = false) String hostHeader) {

        log.info("获取所有上传图片: userId={}, page={}, pageSize={}", userId, page, pageSize);

        try {
            // 查询图片列表
            List<String> imageUrls = mediaUploadService.getAllUploadImages(userId, page, pageSize, hostHeader);

            // 查询总数
            int total = mediaUploadService.getAllUploadImagesCount(userId);

            // 检测可用端口
            String imgServerHost = imageServerService.getImgServerHost().split(":")[0];
            String availablePort = detectAvailablePort(imgServerHost);

            // 构造分页响应
            Map<String, Object> response = new HashMap<>();
            response.put("port", availablePort);
            response.put("data", imageUrls);
            response.put("total", total);
            response.put("page", page);
            response.put("pageSize", pageSize);
            response.put("totalPages", (int) Math.ceil((double) total / pageSize));

            log.info("返回 {} 张图片，总共 {} 张，第{}/{}页", imageUrls.size(), total, page, response.get("totalPages"));
            return ResponseEntity.ok(response);

        } catch (Exception e) {
            log.error("获取所有上传图片失败", e);
            Map<String, Object> errorResponse = new HashMap<>();
            errorResponse.put("error", e.getMessage());
            return ResponseEntity.status(500).body(errorResponse);
        }
    }
}
