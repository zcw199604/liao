package com.zcw.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.MessageDigest;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.*;

/**
 * 文件存储服务
 * 负责文件的本地保存、删除、验证等操作
 */
@Service
public class FileStorageService {

    private static final Logger log = LoggerFactory.getLogger(FileStorageService.class);

    @Autowired
    private JdbcTemplate jdbcTemplate;

    /**
     * 基础上传目录（使用项目根目录）
     */
    private static String BASE_UPLOAD_PATH = System.getProperty("user.dir") + "/upload";

    /**
     * 支持的媒体类型（图片/视频）
     */
    private static final Set<String> SUPPORTED_MEDIA_TYPES = Set.of(
            // 图片格式
            "image/jpeg",
            "image/png",
            "image/gif",
            "image/webp",
            // 视频格式
            "video/mp4"
    );

    /**
     * 媒体类型到存储分类的映射
     * 用于自动判断存储路径：image → ./upload/images/，video → ./upload/videos/
     */
    private static final Map<String, String> MEDIA_TYPE_CATEGORY = Map.of(
            "image/jpeg", "image",
            "image/png", "image",
            "image/gif", "image",
            "image/webp", "image",
            "video/mp4", "video"
    );

    /**
     * 保存文件到本地
     *
     * @param file     上传的文件
     * @param fileType 文件类型（如 "image"）
     * @return 本地存储的相对路径
     * @throws IOException 文件保存失败
     */
    public String saveFile(MultipartFile file, String fileType) throws IOException {
        if (file == null || file.isEmpty()) {
            throw new IOException("文件为空");
        }

        // 生成唯一文件名
        String originalFilename = file.getOriginalFilename();
        String uniqueFilename = generateUniqueFilename(originalFilename);

        // 获取存储目录
        String storageDir = getStorageDirectory(fileType);

        // 创建目录（如果不存在）
        File directory = new File(storageDir);
        if (!directory.exists()) {
            boolean created = directory.mkdirs();
            if (!created) {
                throw new IOException("无法创建存储目录: " + storageDir);
            }
            log.info("创建存储目录: {}", storageDir);
        }

        // 保存文件（使用Files.write避免临时文件问题）
        Path filePath = Paths.get(storageDir, uniqueFilename);
        Files.write(filePath, file.getBytes());

        // 返回相对路径（使用Path对象计算，避免分隔符问题）
        Path basePath = Paths.get(BASE_UPLOAD_PATH);
        Path relativePath = basePath.relativize(filePath);
        String relativePathStr = "/" + relativePath.toString().replace("\\", "/");

        log.info("媒体文件保存成功: {} -> {}", originalFilename, relativePathStr);
        return relativePathStr;
    }

    /**
     * 删除本地文件
     *
     * @param localPath 本地文件相对路径
     * @return 是否删除成功
     */
    public boolean deleteFile(String localPath) {
        if (localPath == null || localPath.isEmpty()) {
            return false;
        }

        try {
            // 移除开头的斜杠，然后构建完整路径
            String cleanPath = localPath.startsWith("/") ? localPath.substring(1) : localPath;
            Path fullPath = Paths.get(BASE_UPLOAD_PATH, cleanPath);
            File file = fullPath.toFile();

            if (file.exists() && file.isFile()) {
                boolean deleted = file.delete();
                if (deleted) {
                    log.info("媒体文件删除成功: {}", localPath);
                    return true;
                } else {
                    log.warn("文件删除失败: {}", localPath);
                    return false;
                }
            } else {
                log.warn("文件不存在: {}", localPath);
                return false;
            }
        } catch (Exception e) {
            log.error("删除文件时发生异常: {}", localPath, e);
            return false;
        }
    }

    /**
     * 验证媒体文件类型
     *
     * @param contentType 文件的MIME类型
     * @return 是否为支持的类型
     */
    public boolean isValidMediaType(String contentType) {
        if (contentType == null || contentType.isEmpty()) {
            return false;
        }
        return SUPPORTED_MEDIA_TYPES.contains(contentType.toLowerCase());
    }

    /**
     * 验证文件类型（已废弃，请使用 isValidMediaType）
     *
     * @param contentType 文件的MIME类型
     * @return 是否为支持的类型
     * @deprecated 为保持向后兼容保留，内部转发到 isValidMediaType
     */
    @Deprecated
    public boolean isValidFileType(String contentType) {
        return isValidMediaType(contentType);
    }

    /**
     * 根据MIME类型推断存储分类
     *
     * @param contentType 文件的MIME类型
     * @return 存储分类（image/video/file）
     */
    public String getCategoryFromContentType(String contentType) {
        if (contentType == null || contentType.isEmpty()) {
            return "file"; // 默认分类
        }
        return MEDIA_TYPE_CATEGORY.getOrDefault(contentType.toLowerCase(), "file");
    }

    /**
     * 获取文件扩展名
     *
     * @param filename 文件名
     * @return 文件扩展名（不含点号），如果没有扩展名返回空字符串
     */
    public String getFileExtension(String filename) {
        if (filename == null || filename.isEmpty()) {
            return "";
        }

        int lastDotIndex = filename.lastIndexOf('.');
        if (lastDotIndex > 0 && lastDotIndex < filename.length() - 1) {
            return filename.substring(lastDotIndex + 1).toLowerCase();
        }

        return "";
    }

    /**
     * 读取本地文件内容
     *
     * @param localPath 本地文件相对路径
     * @return 文件字节数组
     * @throws IOException 文件读取失败
     */
    public byte[] readLocalFile(String localPath) throws IOException {
        if (localPath == null || localPath.isEmpty()) {
            throw new IOException("文件路径为空");
        }

        // 移除开头的斜杠，然后构建完整路径
        String cleanPath = localPath.startsWith("/") ? localPath.substring(1) : localPath;
        Path fullPath = Paths.get(BASE_UPLOAD_PATH, cleanPath);
        File file = fullPath.toFile();

        if (!file.exists() || !file.isFile()) {
            throw new IOException("文件不存在: " + localPath);
        }

        log.info("读取本地文件: {} -> {}", localPath, fullPath);
        return Files.readAllBytes(fullPath);
    }

    /**
     * 生成唯一文件名
     *
     * @param originalFilename 原始文件名
     * @return 唯一文件名，格式: {UUID}_{timestamp}.{ext}
     */
    private String generateUniqueFilename(String originalFilename) {
        String extension = getFileExtension(originalFilename);
        String uuid = UUID.randomUUID().toString().replace("-", "");
        long timestamp = System.currentTimeMillis();

        if (extension.isEmpty()) {
            return uuid + "_" + timestamp;
        } else {
            return uuid + "_" + timestamp + "." + extension;
        }
    }

    /**
     * 获取存储目录（按日期分组）
     *
     * @param fileType 文件类型（如 "image"）
     * @return 存储目录的完整路径
     */
    private String getStorageDirectory(String fileType) {
        LocalDateTime now = LocalDateTime.now();
        String year = now.format(DateTimeFormatter.ofPattern("yyyy"));
        String month = now.format(DateTimeFormatter.ofPattern("MM"));
        String day = now.format(DateTimeFormatter.ofPattern("dd"));

        // 构建目录路径: ./upload/images/2025/12/19/
        return BASE_UPLOAD_PATH + "/" + fileType + "s/" + year + "/" + month + "/" + day;
    }

    /**
     * 计算文件MD5哈希值
     * 使用流式读取，支持大文件
     *
     * @param file 上传的文件
     * @return MD5哈希值（32位十六进制字符串）
     * @throws Exception MD5计算失败
     */
    public String calculateMD5(MultipartFile file) throws Exception {
        MessageDigest md = MessageDigest.getInstance("MD5");
        byte[] buffer = new byte[8192];
        int bytesRead;

        try (InputStream is = file.getInputStream()) {
            while ((bytesRead = is.read(buffer)) != -1) {
                md.update(buffer, 0, bytesRead);
            }
        }

        byte[] digest = md.digest();
        StringBuilder sb = new StringBuilder();
        for (byte b : digest) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }

    /**
     * 根据MD5查找本地文件路径
     * 用于文件去重：如果本地已存在相同MD5的文件，直接复用
     *
     * @param md5 文件MD5值
     * @return 本地文件路径，如果不存在返回null
     */
    public String findLocalPathByMD5(String md5) {
        if (md5 == null || md5.isEmpty()) {
            return null;
        }

        try {
            // 查询数据库，找到第一个匹配的本地路径
            String sql = "SELECT local_path FROM media_upload_history WHERE file_md5 = ? LIMIT 1";
            List<String> results = jdbcTemplate.queryForList(sql, String.class, md5);

            if (!results.isEmpty()) {
                String localPath = results.get(0);
                log.debug("找到MD5匹配的文件: {}", localPath);

                // 验证文件是否仍然存在
                String cleanPath = localPath.startsWith("/") ? localPath.substring(1) : localPath;
                Path fullPath = Paths.get(BASE_UPLOAD_PATH, cleanPath);
                File file = fullPath.toFile();

                if (file.exists() && file.isFile()) {
                    log.info("本地文件存在，复用: {}", localPath);
                    return localPath;
                } else {
                    log.warn("数据库中有记录但文件不存在: {}", localPath);
                    return null;
                }
            }

            log.debug("未找到MD5匹配的文件: {}", md5);
            return null;
        } catch (Exception e) {
            log.error("查询MD5文件失败: {}", md5, e);
            return null;
        }
    }
}
