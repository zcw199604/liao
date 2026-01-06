package com.zcw.controller;

import com.zcw.model.MediaUploadHistory;
import com.zcw.service.FileStorageService;
import com.zcw.service.ImageCacheService;
import com.zcw.service.ImageServerService;
import com.zcw.service.MediaUploadService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.io.ByteArrayResource;
import org.springframework.http.*;
import org.springframework.http.client.SimpleClientHttpRequestFactory;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.multipart.MultipartFile;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 用户历史记录接口
 */
@RestController
@RequestMapping("/api")
public class UserHistoryController {

    private static final Logger log = LoggerFactory.getLogger(UserHistoryController.class);

    private static long elapsedMs(long startNs) {
        return (System.nanoTime() - startNs) / 1_000_000;
    }

    private final RestTemplate restTemplate;

    @Autowired
    private ImageServerService imageServerService;

    @Autowired
    private ImageCacheService imageCacheService;

    @Autowired(required = false)
    private com.zcw.service.UserInfoCacheService userInfoCacheService;

    // 上游接口地址
    private static final String UPSTREAM_API_URL =
        "http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserList_Random";

    private static final String UPSTREAM_FAVORITE_API_URL =
        "http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserList_My";

    private static final String UPSTREAM_REPORT_URL =
        "http://v1.chat2019.cn/asmx/method.asmx/referrer_record";

    private static final String UPSTREAM_MSG_HISTORY_URL =
        "http://v1.chat2019.cn/asmx/method.asmx/randomVIPGetHistoryUserMsgsPage";

    private static final String UPSTREAM_IMG_SERVER_URL =
        "http://v1.chat2019.cn/asmx/method.asmx/getImgServer";

    @Autowired
    private FileStorageService fileStorageService;

    @Autowired
    private MediaUploadService mediaUploadService;

    public UserHistoryController(RestTemplate restTemplate) {
        this.restTemplate = restTemplate;
    }

    /**
     * 获取历史用户列表
     * @param myUserID 用户ID
     * @param vipcode VIP码
     * @param serverPort 服务器端口
     * @param cookieData Cookie数据
     * @param referer Referer header
     * @param userAgent User-Agent header
     * @return 历史用户列表
     */
    @PostMapping("/getHistoryUserList")
    public ResponseEntity<String> getHistoryUserList(
            @RequestParam(required = false, defaultValue = "5be810d731d340f090b098392f9f0a31") String myUserID,
            @RequestParam(required = false, defaultValue = "") String vipcode,
            @RequestParam(required = false, defaultValue = "1001") String serverPort,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        long totalStartNs = System.nanoTime();
        long upstreamMs = -1L;
        long enrichUserInfoMs = -1L;
        long lastMsgMs = -1L;
        int resultSize = -1;
        HttpStatusCode upstreamStatus = null;
        boolean cacheEnabled = userInfoCacheService != null;

        log.info("获取历史用户列表请求: myUserID={}, vipcode={}, serverPort={}", myUserID, vipcode, serverPort);

        try {
            // 设置请求头
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

            // 设置必要的 headers
            headers.set("Host", "v1.chat2019.cn");
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);

            // 如果有 Cookie，添加 Cookie
            if (cookieData != null && !cookieData.isEmpty()) {
                headers.set("Cookie", cookieData);
            }

            // 设置请求参数
            MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
            params.add("myUserID", myUserID);
            params.add("vipcode", vipcode);
            params.add("serverPort", serverPort);

            log.info("请求参数: myUserID={}, vipcode={}, serverPort={}", myUserID, vipcode, serverPort);

            // 创建请求实体
            HttpEntity<MultiValueMap<String, String>> request = new HttpEntity<>(params, headers);

            // 调用上游接口
            long upstreamStartNs = System.nanoTime();
            ResponseEntity<String> response = restTemplate.postForEntity(UPSTREAM_API_URL, request, String.class);
            upstreamMs = elapsedMs(upstreamStartNs);
            upstreamStatus = response.getStatusCode();
            String responseBody = response.getBody();

            log.info("上游接口返回: status={}, bodyLength={}", response.getStatusCode(), responseBody == null ? 0 : responseBody.length());
            log.debug("上游接口 body: {}", responseBody);

            // 增强数据：补充用户信息
            if (response.getStatusCode() == HttpStatus.OK && cacheEnabled && responseBody != null) {
                try {
                    com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
                    com.fasterxml.jackson.databind.JsonNode root = mapper.readTree(responseBody);
                    
                    if (root.isArray()) {
                        java.util.List<Map<String, Object>> list = new ArrayList<>();
                        for (com.fasterxml.jackson.databind.JsonNode node : root) {
                            @SuppressWarnings("unchecked")
                            Map<String, Object> map = mapper.convertValue(node, Map.class);
                            list.add(map);
                        }
                        
                        // 批量增强数据
                        // 明确指定使用 "id" 字段作为用户ID
                        // 为了兼容性，如果 id 不存在，尝试 userid (视实际情况而定，但既然确认是 id，我们可以优先)
                        String idKey = "id";
                        if (!list.isEmpty() && !list.get(0).containsKey("id")) {
                             if (list.get(0).containsKey("UserID")) idKey = "UserID";
                             else if (list.get(0).containsKey("userid")) idKey = "userid";
                        }

                        // 1. 批量增强用户信息（昵称、性别、年龄、地址）
                        long enrichUserInfoStartNs = System.nanoTime();
                        list = userInfoCacheService.batchEnrichUserInfo(list, idKey);
                        enrichUserInfoMs = elapsedMs(enrichUserInfoStartNs);

                        // 2. 批量增强最后消息（lastMsg、lastTime）
                        long enrichLastMsgStartNs = System.nanoTime();
                        list = userInfoCacheService.batchEnrichWithLastMessage(list, myUserID);
                        lastMsgMs = elapsedMs(enrichLastMsgStartNs);
                        resultSize = list.size();

                        return ResponseEntity.ok(mapper.writeValueAsString(list));
                    }
                } catch (Exception e) {
                    log.error("增强历史用户列表失败", e);
                }
            }

            return ResponseEntity.ok(responseBody);

        } catch (Exception e) {
            log.error("调用上游接口失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"调用上游接口失败: " + e.getMessage() + "\"}");
        } finally {
            long totalMs = elapsedMs(totalStartNs);
            log.info(
                    "[timing] /api/getHistoryUserList myUserID={} status={} size={} upstreamMs={} enrichUserInfoMs={} lastMsgMs={} totalMs={} cacheEnabled={}",
                    myUserID,
                    upstreamStatus,
                    resultSize,
                    upstreamMs,
                    enrichUserInfoMs,
                    lastMsgMs,
                    totalMs,
                    cacheEnabled
            );
        }
    }

    /**
     * 获取收藏用户列表
     * @param myUserID 用户ID
     * @param vipcode VIP码
     * @param serverPort 服务器端口
     * @param cookieData Cookie数据
     * @param referer Referer header
     * @param userAgent User-Agent header
     * @return 收藏用户列表
     */
    @PostMapping("/getFavoriteUserList")
    public ResponseEntity<String> getFavoriteUserList(
            @RequestParam(required = false, defaultValue = "5be810d731d340f090b098392f9f0a31") String myUserID,
            @RequestParam(required = false, defaultValue = "") String vipcode,
            @RequestParam(required = false, defaultValue = "1001") String serverPort,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        long totalStartNs = System.nanoTime();
        long upstreamMs = -1L;
        long enrichUserInfoMs = -1L;
        long lastMsgMs = -1L;
        int resultSize = -1;
        HttpStatusCode upstreamStatus = null;
        boolean cacheEnabled = userInfoCacheService != null;

        log.info("获取收藏用户列表请求: myUserID={}, vipcode={}, serverPort={}", myUserID, vipcode, serverPort);

        try {
            // 设置请求头
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

            // 设置必要的 headers
            headers.set("Host", "v1.chat2019.cn");
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);

            // 如果有 Cookie，添加 Cookie
            if (cookieData != null && !cookieData.isEmpty()) {
                headers.set("Cookie", cookieData);
            }

            // 设置请求参数
            MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
            params.add("myUserID", myUserID);
            params.add("vipcode", vipcode);
            params.add("serverPort", serverPort);

            log.info("请求参数: myUserID={}, vipcode={}, serverPort={}", myUserID, vipcode, serverPort);

            // 创建请求实体
            HttpEntity<MultiValueMap<String, String>> request = new HttpEntity<>(params, headers);

            // 调用上游接口
            long upstreamStartNs = System.nanoTime();
            ResponseEntity<String> response = restTemplate.postForEntity(UPSTREAM_FAVORITE_API_URL, request, String.class);
            upstreamMs = elapsedMs(upstreamStartNs);
            upstreamStatus = response.getStatusCode();
            String responseBody = response.getBody();

            log.info("上游收藏接口返回: status={}, bodyLength={}", response.getStatusCode(), responseBody == null ? 0 : responseBody.length());
            log.debug("上游收藏接口 body: {}", responseBody);

            // 增强数据：补充用户信息
            if (response.getStatusCode() == HttpStatus.OK && cacheEnabled && responseBody != null) {
                try {
                    com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
                    com.fasterxml.jackson.databind.JsonNode root = mapper.readTree(responseBody);

                    if (root.isArray()) {
                        java.util.List<Map<String, Object>> list = new ArrayList<>();
                        for (com.fasterxml.jackson.databind.JsonNode node : root) {
                            @SuppressWarnings("unchecked")
                            Map<String, Object> map = mapper.convertValue(node, Map.class);
                            list.add(map);
                        }

                        // 批量增强数据
                        // 明确指定使用 "id" 字段作为用户ID
                        // 为了兼容性，如果 id 不存在，尝试 userid (视实际情况而定，但既然确认是 id，我们可以优先)
                        String idKey = "id";
                        if (!list.isEmpty() && !list.get(0).containsKey("id")) {
                             if (list.get(0).containsKey("UserID")) idKey = "UserID";
                             else if (list.get(0).containsKey("userid")) idKey = "userid";
                        }

                        // 1. 批量增强用户信息（昵称、性别、年龄、地址）
                        long enrichUserInfoStartNs = System.nanoTime();
                        list = userInfoCacheService.batchEnrichUserInfo(list, idKey);
                        enrichUserInfoMs = elapsedMs(enrichUserInfoStartNs);

                        // 2. 批量增强最后消息（lastMsg、lastTime）
                        long enrichLastMsgStartNs = System.nanoTime();
                        list = userInfoCacheService.batchEnrichWithLastMessage(list, myUserID);
                        lastMsgMs = elapsedMs(enrichLastMsgStartNs);
                        resultSize = list.size();

                        return ResponseEntity.ok(mapper.writeValueAsString(list));
                    }
                } catch (Exception e) {
                    log.error("增强收藏用户列表失败", e);
                }
            }

            return ResponseEntity.ok(responseBody);

        } catch (Exception e) {
            log.error("调用上游收藏接口失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"调用上游接口失败: " + e.getMessage() + "\"}");
        } finally {
            long totalMs = elapsedMs(totalStartNs);
            log.info(
                    "[timing] /api/getFavoriteUserList myUserID={} status={} size={} upstreamMs={} enrichUserInfoMs={} lastMsgMs={} totalMs={} cacheEnabled={}",
                    myUserID,
                    upstreamStatus,
                    resultSize,
                    upstreamMs,
                    enrichUserInfoMs,
                    lastMsgMs,
                    totalMs,
                    cacheEnabled
            );
        }
    }

    /**
     * 上报用户访问记录
     * @param referrerUrl 来源URL
     * @param currUrl 当前URL
     * @param userid 用户ID
     * @param cookieData Cookie数据（前端传递完整的Cookie字符串）
     * @param referer Referer header
     * @param userAgent User-Agent header
     * @return 上报结果
     */
    @PostMapping("/reportReferrer")
    public ResponseEntity<String> reportReferrer(
            @RequestParam String referrerUrl,
            @RequestParam String currUrl,
            @RequestParam String userid,
            @RequestParam String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        log.info("上报访问记录: referrerUrl={}, currUrl={}, userid={}", referrerUrl, currUrl, userid);

        try {
            // 设置请求头
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

            // 设置必要的 headers
            headers.set("Host", "v1.chat2019.cn");
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);
            headers.set("Cookie", cookieData);

            log.info("请求 Headers - Host: v1.chat2019.cn, Origin: http://v1.chat2019.cn");
            log.info("Referer: {}", referer);
            log.info("User-Agent: {}", userAgent);
            log.info("Cookie: {}", cookieData);

            // 设置请求参数
            MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
            params.add("referrer_url", referrerUrl);
            params.add("curr_url", currUrl);
            params.add("userid", userid);

            log.info("上报参数: referrer_url={}, curr_url={}, userid={}", referrerUrl, currUrl, userid);

            // 创建请求实体
            HttpEntity<MultiValueMap<String, String>> request = new HttpEntity<>(params, headers);

            // 调用上游接口
            ResponseEntity<String> response = restTemplate.postForEntity(UPSTREAM_REPORT_URL, request, String.class);

            log.info("上报接口返回: status={}, body={}", response.getStatusCode(), response.getBody());

            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            log.error("上报访问记录失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"上报失败: " + e.getMessage() + "\"}");
        }
    }

    /**
     * 获取用户消息历史记录
     * @param myUserID 当前用户ID
     * @param UserToID 对方用户ID
     * @param isFirst 是否首次加载（1=首次，0=加载更多）
     * @param firstTid 第一条消息ID（加载更多时使用）
     * @param vipcode VIP码
     * @param serverPort 服务器端口
     * @param cookieData Cookie数据
     * @param referer Referer header
     * @param userAgent User-Agent header
     * @return 消息历史记录
     */
    @PostMapping("/getMessageHistory")
    public ResponseEntity<String> getMessageHistory(
            @RequestParam String myUserID,
            @RequestParam String UserToID,
            @RequestParam(required = false, defaultValue = "1") String isFirst,
            @RequestParam(required = false, defaultValue = "0") String firstTid,
            @RequestParam(required = false, defaultValue = "") String vipcode,
            @RequestParam(required = false, defaultValue = "1001") String serverPort,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        log.info("获取消息历史请求: myUserID={}, UserToID={}, isFirst={}, firstTid={}",
                myUserID, UserToID, isFirst, firstTid);

        try {
            // 设置请求头
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);

            // 设置必要的 headers
            headers.set("Host", "v1.chat2019.cn");
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);

            // 如果有 Cookie，添加 Cookie
            if (cookieData != null && !cookieData.isEmpty()) {
                headers.set("Cookie", cookieData);
            }

            // 设置请求参数
            MultiValueMap<String, String> params = new LinkedMultiValueMap<>();
            params.add("myUserID", myUserID);
            params.add("UserToID", UserToID);
            params.add("isFirst", isFirst);
            params.add("firstTid", firstTid);
            params.add("vipcode", vipcode);
            params.add("serverPort", serverPort);

            log.info("请求参数: myUserID={}, UserToID={}, isFirst={}, firstTid={}, vipcode={}, serverPort={}",
                    myUserID, UserToID, isFirst, firstTid, vipcode, serverPort);

            // 创建请求实体
            HttpEntity<MultiValueMap<String, String>> request = new HttpEntity<>(params, headers);

            // 调用上游接口
            ResponseEntity<String> response = restTemplate.postForEntity(UPSTREAM_MSG_HISTORY_URL, request, String.class);

//            log.info("消息历史接口返回: status={}, body={}", response.getStatusCode(), response.getBody());

            // 增强数据：补充用户信息，并更新最后消息缓存
            // 注意：消息列表可能包含对方的信息，虽然主要是消息内容，但如果有用户信息字段也可以补充
            if (response.getStatusCode() == HttpStatus.OK && userInfoCacheService != null && response.getBody() != null) {
                try {
                    com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
                    com.fasterxml.jackson.databind.JsonNode root = mapper.readTree(response.getBody());

                    // 检查是否有 contents_list 字段（新格式）
                    if (root.has("contents_list")) {
                        com.fasterxml.jackson.databind.JsonNode contentsList = root.get("contents_list");

                        // 缓存最后一条消息（列表第一条，按时间倒序）
                        if (contentsList.isArray() && contentsList.size() > 0) {
                            com.fasterxml.jackson.databind.JsonNode firstMsg = contentsList.get(0);

                            String fromUserId = firstMsg.path("id").asText();
                            String toUserId = firstMsg.path("toid").asText();
                            String content = firstMsg.path("content").asText();
                            String time = firstMsg.path("time").asText();
                            String type = inferMessageType(content);

                            if (!fromUserId.isEmpty() && !toUserId.isEmpty() && !content.isEmpty() && !time.isEmpty()) {
                                // 兼容：上游返回的id/toid有时不包含myUserID（例如使用了别名ID），
                                // 这里以请求参数的(myUserID, UserToID)为准补写，确保历史列表按myUserID能命中最后消息缓存
                                String cacheFromUserId = fromUserId;
                                String cacheToUserId = toUserId;
                                if (!myUserID.equals(fromUserId) && !myUserID.equals(toUserId)) {
                                    if (UserToID.equals(fromUserId)) {
                                        cacheToUserId = myUserID;
                                    } else if (UserToID.equals(toUserId)) {
                                        cacheFromUserId = myUserID;
                                    }
                                }

                                com.zcw.model.CachedLastMessage lastMsg = new com.zcw.model.CachedLastMessage(
                                    cacheFromUserId, cacheToUserId, content, type, time
                                );
                                userInfoCacheService.saveLastMessage(lastMsg);
                                log.debug("缓存历史消息中的最后一条: {} -> {}, content={}, time={}",
                                          cacheFromUserId, cacheToUserId, content, time);
                            } else {
                                log.warn("历史消息字段不完整: fromUserId={}, toUserId={}, content={}, time={}",
                                         fromUserId, toUserId, content, time);
                            }
                        }

                        // 返回原始响应，保持接口兼容性
                        return ResponseEntity.ok(response.getBody());
                    }

                    // 兼容旧格式：直接是数组
                    if (root.isArray()) {
                        java.util.List<Map<String, Object>> list = new ArrayList<>();
                        for (com.fasterxml.jackson.databind.JsonNode node : root) {
                            @SuppressWarnings("unchecked")
                            Map<String, Object> map = mapper.convertValue(node, Map.class);
                            list.add(map);
                        }

                        // 批量增强数据，消息列表中的用户ID字段通常是 userid
                        // 这样会把发送方的信息（无论自己还是对方）都尝试补充，如果缓存有的话
                        list = userInfoCacheService.batchEnrichUserInfo(list, "userid");

                        return ResponseEntity.ok(mapper.writeValueAsString(list));
                    }
                } catch (Exception e) {
                    log.error("增强消息历史失败", e);
                }
            }

            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            log.error("获取消息历史失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"获取消息历史失败: " + e.getMessage() + "\"}");
        }
    }

    /**
     * 获取图片服务器地址
     * @return 图片服务器信息
     */
    @GetMapping("/getImgServer")
    public ResponseEntity<String> getImgServer() {
        log.info("获取图片服务器地址请求");

        try {
            // 添加时间戳参数
            String url = UPSTREAM_IMG_SERVER_URL + "?_=" + System.currentTimeMillis();

            log.info("请求 URL: {}", url);

            // 调用上游接口
            String response = restTemplate.getForObject(url, String.class);

            log.info("图片服务器接口返回: {}", response);

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            log.error("获取图片服务器地址失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"获取图片服务器失败: " + e.getMessage() + "\"}");
        }
    }

    /**
     * 更新图片服务器地址
     * @param server 图片服务器地址
     * @return 更新结果
     */
    @PostMapping("/updateImgServer")
    public ResponseEntity<String> updateImgServer(@RequestParam String server) {
        log.info("更新图片服务器地址: {}", server);
        imageServerService.setImgServerHost(server);
        return ResponseEntity.ok("{\"success\":true}");
    }

    /**
     * 上传图片到上游服务器
     * @param file 图片文件
     * @param userid 用户ID
     * @param cookieData Cookie数据
     * @param referer Referer header
     * @param userAgent User-Agent header
     * @return 上传结果
     */
    /**
     * 上传媒体文件（图片/视频）
     */
    @PostMapping("/uploadMedia")
    public ResponseEntity<String> uploadMedia(
            @RequestParam("file") MultipartFile file,
            @RequestParam String userid,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        log.info("上传媒体请求: userid={}, fileName={}, fileSize={}, contentType={}",
                userid, file.getOriginalFilename(), file.getSize(), file.getContentType());

        String localPath = null;
        String md5 = null;

        try {
            // 1. 验证文件类型
            if (!fileStorageService.isValidMediaType(file.getContentType())) {
                log.warn("不支持的文件类型: {}", file.getContentType());
                return ResponseEntity.status(400).body("{\"error\":\"不支持的文件类型\"}");
            }

            // 2. 计算文件MD5
            try {
                md5 = fileStorageService.calculateMD5(file);
                log.info("文件MD5: {}", md5);
            } catch (Exception e) {
                log.error("计算MD5失败", e);
                return ResponseEntity.status(500).body("{\"error\":\"MD5计算失败\"}");
            }

            // 3. 检查本地是否已存在相同MD5的文件
            String existingLocalPath = fileStorageService.findLocalPathByMD5(md5);
            if (existingLocalPath != null) {
                // 本地文件已存在，复用
                log.info("文件已存在（MD5={}），复用本地路径: {}", md5, existingLocalPath);
                localPath = existingLocalPath;
            } else {
                // 4. 本地文件不存在，保存新文件
                try {
                    // 根据MIME类型自动推断存储分类
                    String category = fileStorageService.getCategoryFromContentType(file.getContentType());
                    localPath = fileStorageService.saveFile(file, category);
                    log.info("文件已保存到本地: {}，分类: {} -> {}", localPath, file.getContentType(), category);
                } catch (Exception e) {
                    log.error("本地文件保存失败", e);
                    return ResponseEntity.status(500).body("{\"error\":\"本地存储失败: " + e.getMessage() + "\"}");
                }
            }

            // 5. 上传到上游服务器（每次都上传）
            String uploadUrl = String.format("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s",
                    imageServerService.getImgServerHost(), userid);

            log.info("上传到媒体服务器: {}", uploadUrl);

            // 设置请求头
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.MULTIPART_FORM_DATA);

            // 添加必要的 headers
            headers.set("Host", imageServerService.getImgServerHost().split(":")[0]);
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);

            // 如果有 Cookie，添加 Cookie
            if (cookieData != null && !cookieData.isEmpty()) {
                headers.set("Cookie", cookieData);
            }

            log.info("上传请求 Headers - Host: {}, Origin: http://v1.chat2019.cn", imageServerService.getImgServerHost().split(":")[0]);
            log.info("Referer: {}", referer);

            // 构造 multipart 请求
            MultiValueMap<String, Object> body = new LinkedMultiValueMap<>();

            // 将文件转换为资源
            ByteArrayResource fileResource = new ByteArrayResource(file.getBytes()) {
                @Override
                public String getFilename() {
                    return file.getOriginalFilename();
                }
            };

            body.add("upload_file", fileResource);

            // 创建请求实体
            HttpEntity<MultiValueMap<String, Object>> requestEntity = new HttpEntity<>(body, headers);

            // 调用上游接口
            ResponseEntity<String> response = restTemplate.postForEntity(uploadUrl, requestEntity, String.class);

            log.info("媒体上传成功: contentType={}, response={}", file.getContentType(), response.getBody());

            // 4. 解析返回结果并保存到数据库
            try {
                com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
                com.fasterxml.jackson.databind.JsonNode jsonNode = mapper.readTree(response.getBody());

                if ("OK".equals(jsonNode.get("state").asText()) && jsonNode.has("msg")) {
                    String imagePath = jsonNode.get("msg").asText();
                    String imgServerHostClean = imageServerService.getImgServerHost().split(":")[0];

                    // 检测可用端口
                    String availablePort = detectAvailablePort(imgServerHostClean);
                    String imageUrl = String.format("http://%s:%s/img/Upload/%s", imgServerHostClean, availablePort, imagePath);

                    // 5. 保存上传历史到数据库
                    MediaUploadHistory history = new MediaUploadHistory();
                    history.setUserId(userid);
                    history.setToUserId(null);  // 上传时不知道接收方
                    history.setOriginalFilename(file.getOriginalFilename());
                    history.setLocalFilename(localPath.substring(localPath.lastIndexOf("/") + 1));
                    history.setRemoteFilename(imagePath);
                    history.setRemoteUrl(imageUrl);
                    history.setLocalPath(localPath);
                    history.setFileSize(file.getSize());
                    history.setFileType(file.getContentType());
                    history.setFileExtension(fileStorageService.getFileExtension(file.getOriginalFilename()));
                    history.setFileMd5(md5);  // 设置MD5值

                    try {
                        mediaUploadService.saveUploadRecord(history);
                        log.info("上传历史已保存到数据库");
                    } catch (Exception e) {
                        // 数据库保存失败不影响返回，仅记录日志
                        log.error("保存上传历史到数据库失败", e);
                    }

                    // 6. 添加到内存缓存（存储 local_path 而非 remote_url）
                    imageCacheService.addImageToCache(userid, localPath);
                    log.info("媒体已添加到缓存: userid={}, localPath={}, contentType={}", userid, localPath, file.getContentType());

                    // 7. 构造包含端口信息的响应返回给前端
                    com.fasterxml.jackson.databind.node.ObjectNode enhancedResponse = mapper.createObjectNode();
                    enhancedResponse.put("state", "OK");
                    enhancedResponse.put("msg", imagePath);
                    enhancedResponse.put("port", availablePort);  // 添加端口信息
                    enhancedResponse.put("localFilename", history.getLocalFilename()); // 添加本地文件名，用于精确关联

                    return ResponseEntity.ok(enhancedResponse.toString());
                }
            } catch (Exception e) {
                log.warn("解析上传结果失败，跳过缓存和数据库保存", e);
            }

            // 如果解析失败或state不是OK，返回原始响应
            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            // 上游上传失败，保留本地文件供用户在"全站图片库"中重试
            log.error("上传媒体到上游失败，本地文件已保存: localPath={}, contentType={}", localPath, file.getContentType(), e);

            // 返回错误信息，包含本地路径供前端使用
            String errorMessage = e.getMessage() != null ? e.getMessage() : "上传到上游服务器失败";
            return ResponseEntity.status(500).body(String.format(
                "{\"error\":\"上传媒体失败: %s\",\"localPath\":\"%s\"}",
                errorMessage,
                localPath != null ? localPath : ""
            ));
        }
    }

    /**
     * 上传图片（已废弃，请使用 /uploadMedia）
     * @deprecated 为保持向后兼容保留，内部转发到 uploadMedia
     */
    @PostMapping("/uploadImage")
    @Deprecated
    public ResponseEntity<String> uploadImage(
            @RequestParam("file") MultipartFile file,
            @RequestParam String userid,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        // 转发到新接口
        return uploadMedia(file, userid, cookieData, referer, userAgent);
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
     * 获取用户的缓存图片列表
     *
     * @param userid     用户ID
     * @param hostHeader HTTP请求的Host头
     * @return 图片本地访问URL列表和可用端口
     */
    @GetMapping("/getCachedImages")
    public ResponseEntity<Map<String, Object>> getCachedImages(
            @RequestParam String userid,
            @RequestHeader(value = "Host", required = false) String hostHeader) {

        log.info("获取缓存图片列表: userid={}, host={}", userid, hostHeader);

        ImageCacheService.CachedImages cached = imageCacheService.getCachedImages(userid);

        if (cached == null) {
            log.info("用户 {} 没有缓存图片", userid);
            Map<String, Object> emptyResponse = new HashMap<>();
            emptyResponse.put("port", "9006");
            emptyResponse.put("data", new ArrayList<>());
            return ResponseEntity.ok(emptyResponse);
        }

        // 检测可用端口
        String imgServerHost = imageServerService.getImgServerHost().split(":")[0];
        String availablePort = detectAvailablePort(imgServerHost);

        // 将缓存的 local_path 转换为本地访问URL
        List<String> localUrls = mediaUploadService.convertPathsToLocalUrls(cached.getImageUrls(), hostHeader);

        log.info("返回 {} 张缓存图片，端口: {}", localUrls.size(), availablePort);

        // 返回包含端口信息的对象
        Map<String, Object> response = new HashMap<>();
        response.put("port", availablePort);
        response.put("data", localUrls);

        return ResponseEntity.ok(response);
    }

    /**
     * 收藏/取消收藏用户
     */
    @PostMapping("/toggleFavorite")
    public ResponseEntity<String> toggleFavorite(
            @RequestParam String myUserID,
            @RequestParam String UserToID,
            @RequestParam(required = false, defaultValue = "") String vipcode,
            @RequestParam(required = false, defaultValue = "1001") String serverPort,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0") String userAgent) {

        log.info("收藏操作: myUserID={}, UserToID={}", myUserID, UserToID);

        try {
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);
            headers.set("Host", "v1.chat2019.cn");
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);
            headers.set("Cookie", cookieData);

            MultiValueMap<String, String> formData = new LinkedMultiValueMap<>();
            formData.add("myUserID", myUserID);
            formData.add("UserToID", UserToID);
            formData.add("vipcode", vipcode);
            formData.add("serverPort", serverPort);

            HttpEntity<MultiValueMap<String, String>> requestEntity = new HttpEntity<>(formData, headers);

            String upstreamUrl = "http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Do";
            ResponseEntity<String> response = restTemplate.exchange(
                upstreamUrl,
                HttpMethod.POST,
                requestEntity,
                String.class
            );

            log.info("收藏操作响应: {}", response.getBody());
            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            log.error("收藏操作失败", e);
            return ResponseEntity.ok("{\"state\":\"ERROR\",\"msg\":\"" + e.getMessage() + "\"}");
        }
    }

    /**
     * 取消收藏用户
     */
    @PostMapping("/cancelFavorite")
    public ResponseEntity<String> cancelFavorite(
            @RequestParam String myUserID,
            @RequestParam String UserToID,
            @RequestParam(required = false, defaultValue = "1001") String serverPort,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0") String userAgent) {

        log.info("取消收藏操作: myUserID={}, UserToID={}", myUserID, UserToID);

        try {
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_FORM_URLENCODED);
            headers.set("Host", "v1.chat2019.cn");
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);
            headers.set("Cookie", cookieData);

            MultiValueMap<String, String> formData = new LinkedMultiValueMap<>();
            formData.add("myUserID", myUserID);
            formData.add("UserToID", UserToID);
            formData.add("serverPort", serverPort);

            HttpEntity<MultiValueMap<String, String>> requestEntity = new HttpEntity<>(formData, headers);

            String upstreamUrl = "http://v1.chat2019.cn/asmx/method.asmx/random_MyHeart_Cancle";
            ResponseEntity<String> response = restTemplate.exchange(
                upstreamUrl,
                HttpMethod.POST,
                requestEntity,
                String.class
            );

            log.info("取消收藏操作响应: {}", response.getBody());
            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            log.error("取消收藏操作失败", e);
            return ResponseEntity.ok("{\"state\":\"ERROR\",\"msg\":\"" + e.getMessage() + "\"}");
        }
    }

    /**
     * 根据消息内容推断消息类型
     * @param content 消息内容
     * @return 消息类型：text/image/video/audio/file
     */
    private String inferMessageType(String content) {
        if (content == null || content.isEmpty()) {
            return "text";
        }

        // 检查是否是媒体路径格式 [path/to/file.ext]
        if (content.startsWith("[") && content.endsWith("]")) {
            String path = content.substring(1, content.length() - 1).toLowerCase();

            if (path.matches(".*\\.(jpg|jpeg|png|gif|bmp)$")) {
                return "image";
            }
            if (path.matches(".*\\.(mp4|avi|mov|wmv|flv)$")) {
                return "video";
            }
            if (path.matches(".*\\.(mp3|wav|aac|flac)$")) {
                return "audio";
            }
            return "file";
        }

        return "text";
    }
}
