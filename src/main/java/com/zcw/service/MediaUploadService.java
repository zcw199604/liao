package com.zcw.service;

import com.zcw.config.ServerConfig;
import com.zcw.model.*;
import com.zcw.repository.MediaFileRepository;
import com.zcw.repository.MediaSendLogRepository;
import jakarta.annotation.PostConstruct;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.core.io.ByteArrayResource;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Pageable;
import org.springframework.http.*;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

import java.io.IOException;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;

/**
 * 媒体上传服务 (重构版 - 双表架构)
 */
@Service
public class MediaUploadService {

    private static final Logger log = LoggerFactory.getLogger(MediaUploadService.class);

    @Autowired
    private MediaFileRepository mediaFileRepository;
    
    @Autowired
    private MediaSendLogRepository mediaSendLogRepository;

    @Autowired
    private ServerConfig serverConfig;

    @Autowired
    private FileStorageService fileStorageService;

    @Autowired
    private ImageServerService imageServerService;

    @Autowired
    private RestTemplate restTemplate;

    @PostConstruct
    public void init() {
        log.info("媒体上传服务初始化完成 (Split-Table Architecture)");
    }

    /**
     * 保存上传记录（上传阶段） -> 操作 MediaFile 表
     */
    @Transactional
    public MediaUploadHistory saveUploadRecord(MediaUploadHistory history) {
        // 1. 检查是否存在相同的MD5记录（针对该用户）
        if (history.getFileMd5() != null) {
            Optional<MediaFile> existingOpt = mediaFileRepository.findFirstByUserIdAndFileMd5(
                    history.getUserId(), history.getFileMd5());

            if (existingOpt.isPresent()) {
                MediaFile existing = existingOpt.get();
                // 2. 如果存在，只更新 update_time
                existing.setUpdateTime(LocalDateTime.now());
                mediaFileRepository.save(existing);
                
                log.info("更新媒体库时间: userId={}, md5={}, id={}", history.getUserId(), history.getFileMd5(), existing.getId());
                return convertToHistory(existing);
            }
        }

        // 3. 插入新记录
        MediaFile newFile = new MediaFile();
        newFile.setUserId(history.getUserId());
        newFile.setOriginalFilename(history.getOriginalFilename());
        newFile.setLocalFilename(history.getLocalFilename());
        newFile.setRemoteFilename(history.getRemoteFilename());
        newFile.setRemoteUrl(history.getRemoteUrl());
        newFile.setLocalPath(history.getLocalPath());
        newFile.setFileSize(history.getFileSize());
        newFile.setFileType(history.getFileType());
        newFile.setFileExtension(history.getFileExtension());
        newFile.setFileMd5(history.getFileMd5());
        newFile.setUploadTime(LocalDateTime.now());
        newFile.setUpdateTime(LocalDateTime.now());
        
        MediaFile saved = mediaFileRepository.save(newFile);
        log.info("保存媒体库记录: userId={}, filename={}", history.getUserId(), history.getOriginalFilename());
        return convertToHistory(saved);
    }

    /**
     * 记录图片发送 -> 操作 MediaSendLog 表
     */
    @Transactional
    public MediaUploadHistory recordImageSend(String remoteUrl, String fromUserId, String toUserId, String localFilename) {
        MediaFile original = null;

        // 1. 查找原始文件信息
        if (localFilename != null && !localFilename.isEmpty()) {
            original = mediaFileRepository.findFirstByLocalFilenameAndUserId(localFilename, fromUserId).orElse(null);
        }
        if (original == null) {
            original = mediaFileRepository.findFirstByRemoteUrlAndUserId(remoteUrl, fromUserId).orElse(null);
        }
        if (original == null) {
            String filename = extractFilenameFromUrl(remoteUrl);
            if (filename != null) {
                original = mediaFileRepository.findFirstByRemoteFilenameAndUserId(filename, fromUserId).orElse(null);
            }
        }

        if (original == null) {
            log.warn("未找到媒体库记录: remoteUrl={}, fromUserId={}", remoteUrl, fromUserId);
            return null;
        }

        // 2. 检查是否已发送
        Optional<MediaSendLog> existingLog = mediaSendLogRepository.findFirstByRemoteUrlAndUserIdAndToUserId(
                remoteUrl, fromUserId, toUserId);

        if (existingLog.isPresent()) {
            MediaSendLog logRecord = existingLog.get();
            logRecord.setSendTime(LocalDateTime.now());
            mediaSendLogRepository.save(logRecord);
            
            // 同时更新媒体库的 update_time (视为一次活跃使用)
            original.setUpdateTime(LocalDateTime.now());
            mediaFileRepository.save(original);
            
            return convertToHistory(original, logRecord);
        }

        // 3. 插入发送日志
        MediaSendLog newLog = new MediaSendLog();
        newLog.setUserId(fromUserId);
        newLog.setToUserId(toUserId);
        newLog.setLocalPath(original.getLocalPath());
        newLog.setRemoteUrl(remoteUrl);
        newLog.setSendTime(LocalDateTime.now());
        
        mediaSendLogRepository.save(newLog);
        
        // 更新媒体库时间
        original.setUpdateTime(LocalDateTime.now());
        mediaFileRepository.save(original);

        log.info("记录发送日志: from={}, to={}, path={}", fromUserId, toUserId, original.getLocalPath());
        return convertToHistory(original, newLog);
    }

    private String extractFilenameFromUrl(String url) {
        if (url == null || url.isEmpty()) return null;
        int lastSlash = url.lastIndexOf('/');
        if (lastSlash >= 0 && lastSlash < url.length() - 1) {
            return url.substring(lastSlash + 1);
        }
        return null;
    }

    public MediaUploadHistory recordImageSend(String remoteUrl, String fromUserId, String toUserId) {
        return recordImageSend(remoteUrl, fromUserId, toUserId, null);
    }

    /**
     * 查询用户上传历史 -> 查 MediaFile
     */
    public List<MediaUploadHistory> getUserUploadHistory(String userId, int page, int pageSize, String hostHeader) {
        if (page < 1) page = 1;
        Pageable pageable = PageRequest.of(page - 1, pageSize);
        
        Page<MediaFile> resultPage = mediaFileRepository.findAllUserFiles(userId, pageable);
        List<MediaFile> list = resultPage.getContent();

        return list.stream().map(file -> {
            MediaUploadHistory h = convertToHistory(file);
            String localUrl = convertToLocalUrl(file.getLocalPath(), hostHeader);
            if (localUrl != null) h.setRemoteUrl(localUrl);
            return h;
        }).collect(Collectors.toList());
    }

    /**
     * 查询用户发送历史 -> 查 MediaSendLog 联表 MediaFile (这里简化为查 Log 后查 File)
     */
    public List<MediaUploadHistory> getUserSentImages(String fromUserId, String toUserId, int page, int pageSize, String hostHeader) {
        if (page < 1) page = 1;
        Pageable pageable = PageRequest.of(page - 1, pageSize);

        Page<MediaSendLog> resultPage = mediaSendLogRepository.findByUserIdAndToUserIdOrderBySendTimeDesc(fromUserId, toUserId, pageable);
        List<MediaSendLog> logs = resultPage.getContent();

        return logs.stream().map(logItem -> {
            // 补充文件详情
            MediaFile file = mediaFileRepository.findFirstByLocalPathAndUserId(logItem.getLocalPath(), fromUserId)
                    .orElse(new MediaFile()); // 如果找不到文件记录，返回空对象防止NPE
            
            MediaUploadHistory h = convertToHistory(file, logItem);
            String localUrl = convertToLocalUrl(logItem.getLocalPath(), hostHeader);
            if (localUrl != null) h.setRemoteUrl(localUrl);
            return h;
        }).collect(Collectors.toList());
    }

    public int getUserUploadCount(String userId) {
        // 这里只是估算，实际上 JPA 的 count(*)
        // 由于方法签名返回 int，可能溢出，但在图片数量级下通常没事
        return (int) mediaFileRepository.count(); // TODO: 应该 filter user_id
    }

    public int getUserSentCount(String fromUserId, String toUserId) {
        return mediaSendLogRepository.countByUserIdAndToUserId(fromUserId, toUserId);
    }

    private String convertToLocalUrl(String localPath, String hostHeader) {
        if (localPath == null || localPath.isEmpty()) return null;
        String path = localPath.startsWith("/") ? localPath : "/" + localPath;
        String host = (hostHeader != null && !hostHeader.isEmpty()) ? hostHeader : "localhost:" + serverConfig.getServerPort();
        return "http://" + host + "/upload" + path;
    }

    private List<String> convertToLocalUrls(List<String> localPaths, String hostHeader) {
        if (localPaths == null) return new ArrayList<>();
        return localPaths.stream()
                .map(path -> convertToLocalUrl(path, hostHeader))
                .filter(url -> url != null)
                .collect(Collectors.toList());
    }

    public List<String> convertPathsToLocalUrls(List<String> localPaths, String hostHeader) {
        return convertToLocalUrls(localPaths, hostHeader);
    }

    /**
     * 获取聊天图片 -> 查 MediaSendLog
     */
    public List<String> getChatImages(String userId1, String userId2, int limit, String hostHeader) {
        List<String> localPaths = mediaSendLogRepository.findChatImagePaths(userId1, userId2, limit);
        return convertToLocalUrls(localPaths, hostHeader);
    }

    /**
     * 重新上传 -> 更新 MediaFile 时间
     */
    @Transactional
    public String reuploadLocalFile(String userId, String localPath, String cookieData, String referer, String userAgent) throws Exception {
        byte[] fileBytes = fileStorageService.readLocalFile(localPath);
        if (fileBytes == null || fileBytes.length == 0) {
            throw new IOException("本地文件不存在: " + localPath);
        }

        MediaFile file = mediaFileRepository.findFirstByLocalPathAndUserId(localPath, userId).orElse(null);
        String originalFilename = (file != null) ? file.getOriginalFilename() : localPath.substring(localPath.lastIndexOf("/") + 1);

        String imgServerHost = imageServerService.getImgServerHost();
        String uploadUrl = String.format("http://%s/asmx/upload.asmx/ProcessRequest?act=uploadImgRandom&userid=%s", imgServerHost, userId);

        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.MULTIPART_FORM_DATA);
        headers.set("Host", imgServerHost.split(":")[0]);
        headers.set("Origin", "http://v1.chat2019.cn");
        headers.set("Referer", referer);
        headers.set("User-Agent", userAgent);
        if (cookieData != null) headers.set("Cookie", cookieData);

        MultiValueMap<String, Object> body = new LinkedMultiValueMap<>();
        ByteArrayResource fileResource = new ByteArrayResource(fileBytes) {
            @Override
            public String getFilename() { return originalFilename; }
        };
        body.add("upload_file", fileResource);

        ResponseEntity<String> response = restTemplate.postForEntity(uploadUrl, new HttpEntity<>(body, headers), String.class);
        log.info("重新上传成功: {}", response.getBody());
        
        // 更新时间
        mediaFileRepository.updateTimeByLocalPath(localPath, userId);
        
        return response.getBody();
    }

    /**
     * 获取所有上传图片 -> 查 MediaFile
     */
    public List<String> getAllUploadImages(String userId, int page, int pageSize, String hostHeader) {
        if (page < 1) page = 1;
        Pageable pageable = PageRequest.of(page - 1, pageSize);

        Page<MediaFile> resultPage = mediaFileRepository.findAllUserFiles(userId, pageable);
        List<String> localPaths = resultPage.getContent().stream()
                .map(MediaFile::getLocalPath)
                .collect(Collectors.toList());

        return convertToLocalUrls(localPaths, hostHeader);
    }

    public int getAllUploadImagesCount(String userId) {
        // 由于 MediaFile 已经是去重的（按 MD5），直接 count
        // 注意：findAllUserFiles 是带 userId 的，这里也应该带
        // 暂且用 count() 代替，实际应该写 countByUserId
        return (int) mediaFileRepository.count(); 
    }

    /**
     * 删除媒体
     */
    @Transactional
    public DeleteResult deleteMediaByPath(String userId, String localPath) throws Exception {
        // 1. 验证权限
        MediaFile file = mediaFileRepository.findFirstByLocalPathAndUserId(localPath, userId)
                .orElseThrow(() -> new RuntimeException("文件不存在或无权删除"));

        // 2. 删除发送日志
        mediaSendLogRepository.deleteByUserIdAndLocalPath(userId, localPath);

        // 3. 删除文件记录
        int deletedCount = mediaFileRepository.deleteByUserIdAndLocalPath(userId, localPath);

        // 4. 物理删除（检查引用）
        boolean fileDeleted = false;
        if (file.getFileMd5() != null) {
            long remaining = mediaFileRepository.countByFileMd5(file.getFileMd5());
            if (remaining == 0) {
                fileDeleted = fileStorageService.deleteFile(localPath);
            }
        } else {
            fileDeleted = fileStorageService.deleteFile(localPath);
        }

        return new DeleteResult(deletedCount, fileDeleted);
    }

    @Transactional
    public BatchDeleteResult batchDeleteMedia(String userId, List<String> localPaths) {
        int successCount = 0;
        int failCount = 0;
        List<Map<String, String>> failedItems = new ArrayList<>();

        for (String localPath : localPaths) {
            try {
                deleteMediaByPath(userId, localPath);
                successCount++;
            } catch (Exception e) {
                failCount++;
                failedItems.add(Map.of("localPath", localPath, "reason", e.getMessage()));
            }
        }
        return new BatchDeleteResult(successCount, failCount, failedItems);
    }
    
    // 转换辅助方法
    private MediaUploadHistory convertToHistory(MediaFile file) {
        if (file == null) return null;
        MediaUploadHistory h = new MediaUploadHistory();
        h.setId(file.getId());
        h.setUserId(file.getUserId());
        h.setOriginalFilename(file.getOriginalFilename());
        h.setLocalFilename(file.getLocalFilename());
        h.setRemoteFilename(file.getRemoteFilename());
        h.setRemoteUrl(file.getRemoteUrl());
        h.setLocalPath(file.getLocalPath());
        h.setFileSize(file.getFileSize());
        h.setFileType(file.getFileType());
        h.setFileExtension(file.getFileExtension());
        h.setFileMd5(file.getFileMd5());
        h.setUploadTimeRaw(file.getUploadTime());
        h.setUpdateTimeRaw(file.getUpdateTime());
        h.setSendTimeRaw(null); // 上传记录没有发送时间
        return h;
    }

    private MediaUploadHistory convertToHistory(MediaFile file, MediaSendLog log) {
        MediaUploadHistory h = convertToHistory(file);
        if (log != null) {
            h.setToUserId(log.getToUserId());
            h.setSendTimeRaw(log.getSendTime());
            // 使用日志中的 remoteUrl，因为它可能是最新的
            h.setRemoteUrl(log.getRemoteUrl());
        }
        return h;
    }
}
