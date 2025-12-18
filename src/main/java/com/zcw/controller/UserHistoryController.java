package com.zcw.controller;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.core.io.ByteArrayResource;
import org.springframework.http.*;
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

    private final RestTemplate restTemplate;

    // 图片服务器地址（动态获取）
    private volatile String imgServerHost = "149.88.79.98:9003";

    // 图片缓存：用户ID -> 图片URL列表及过期时间
    private final Map<String, CachedImages> imageCache = new ConcurrentHashMap<>();

    // 缓存过期时间：3小时（毫秒）
    private static final long CACHE_EXPIRE_TIME = 3 * 60 * 60 * 1000;

    /**
     * 缓存图片数据结构
     */
    private static class CachedImages {
        List<String> imageUrls;
        long expireTime;

        CachedImages(List<String> imageUrls, long expireTime) {
            this.imageUrls = imageUrls;
            this.expireTime = expireTime;
        }

        boolean isExpired() {
            return System.currentTimeMillis() > expireTime;
        }
    }

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

    public UserHistoryController() {
        this.restTemplate = new RestTemplate();
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
            ResponseEntity<String> response = restTemplate.postForEntity(UPSTREAM_API_URL, request, String.class);

            log.info("上游接口返回: status={}, body={}", response.getStatusCode(), response.getBody());

            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            log.error("调用上游接口失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"调用上游接口失败: " + e.getMessage() + "\"}");
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
            ResponseEntity<String> response = restTemplate.postForEntity(UPSTREAM_FAVORITE_API_URL, request, String.class);

            log.info("上游收藏接口返回: status={}, body={}", response.getStatusCode(), response.getBody());

            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            log.error("调用上游收藏接口失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"调用上游接口失败: " + e.getMessage() + "\"}");
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
        this.imgServerHost = server + ":9003";
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
    @PostMapping("/uploadImage")
    public ResponseEntity<String> uploadImage(
            @RequestParam("file") MultipartFile file,
            @RequestParam String userid,
            @RequestParam(required = false, defaultValue = "") String cookieData,
            @RequestParam(required = false, defaultValue = "http://v1.chat2019.cn/randomdeskrynew4m1phj.html?v=4m1phj") String referer,
            @RequestParam(required = false, defaultValue = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36") String userAgent) {

        log.info("上传图片请求: userid={}, fileName={}, fileSize={}",
                userid, file.getOriginalFilename(), file.getSize());

        try {
            // 构造上传URL
            String uploadUrl = String.format("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s",
                    imgServerHost, userid);

            log.info("上传到图片服务器: {}", uploadUrl);

            // 设置请求头
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.MULTIPART_FORM_DATA);

            // 添加必要的 headers
            headers.set("Host", imgServerHost.split(":")[0]);
            headers.set("Origin", "http://v1.chat2019.cn");
            headers.set("Referer", referer);
            headers.set("User-Agent", userAgent);

            // 如果有 Cookie，添加 Cookie
            if (cookieData != null && !cookieData.isEmpty()) {
                headers.set("Cookie", cookieData);
            }

            log.info("上传请求 Headers - Host: {}, Origin: http://v1.chat2019.cn", imgServerHost.split(":")[0]);
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

            log.info("图片上传成功: {}", response.getBody());

            // 解析返回结果并缓存图片URL
            try {
                com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
                com.fasterxml.jackson.databind.JsonNode jsonNode = mapper.readTree(response.getBody());

                if ("OK".equals(jsonNode.get("state").asText()) && jsonNode.has("msg")) {
                    String imagePath = jsonNode.get("msg").asText();
                    String imageUrl = String.format("http://%s:9006/img/Upload/%s", imgServerHost.split(":")[0], imagePath);

                    // 添加到缓存
                    addImageToCache(userid, imageUrl);
                    log.info("图片已添加到缓存: userid={}, imageUrl={}", userid, imageUrl);
                }
            } catch (Exception e) {
                log.warn("解析上传结果失败，跳过缓存", e);
            }

            return ResponseEntity.ok(response.getBody());

        } catch (Exception e) {
            log.error("上传图片失败", e);
            return ResponseEntity.status(500).body("{\"error\":\"上传图片失败: " + e.getMessage() + "\"}");
        }
    }

    /**
     * 添加图片到缓存
     * @param userid 用户ID
     * @param imageUrl 图片URL
     */
    private void addImageToCache(String userid, String imageUrl) {
        long expireTime = System.currentTimeMillis() + CACHE_EXPIRE_TIME;

        CachedImages cached = imageCache.get(userid);
        if (cached == null || cached.isExpired()) {
            // 创建新缓存
            List<String> urls = new ArrayList<>();
            urls.add(imageUrl);
            imageCache.put(userid, new CachedImages(urls, expireTime));
        } else {
            // 添加到现有缓存
            cached.imageUrls.add(imageUrl);
            // 更新过期时间
            cached.expireTime = expireTime;
        }
    }

    /**
     * 获取用户的缓存图片列表
     * @param userid 用户ID
     * @return 图片URL列表
     */
    @GetMapping("/getCachedImages")
    public ResponseEntity<List<String>> getCachedImages(@RequestParam String userid) {
        log.info("获取缓存图片列表: userid={}", userid);

        CachedImages cached = imageCache.get(userid);

        if (cached == null) {
            log.info("用户 {} 没有缓存图片", userid);
            return ResponseEntity.ok(new ArrayList<>());
        }

        if (cached.isExpired()) {
            log.info("用户 {} 的缓存已过期，清除缓存", userid);
            imageCache.remove(userid);
            return ResponseEntity.ok(new ArrayList<>());
        }

        log.info("返回 {} 张缓存图片", cached.imageUrls.size());
        return ResponseEntity.ok(cached.imageUrls);
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
}
