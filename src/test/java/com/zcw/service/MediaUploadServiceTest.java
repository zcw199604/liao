package com.zcw.service;

import com.zcw.config.ServerConfig;
import com.zcw.model.MediaFile;
import com.zcw.model.MediaSendLog;
import com.zcw.model.MediaUploadHistory;
import com.zcw.repository.MediaFileRepository;
import com.zcw.repository.MediaSendLogRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import java.time.LocalDateTime;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("媒体上传服务测试")
class MediaUploadServiceTest {

    @Mock
    private MediaFileRepository mediaFileRepository;

    @Mock
    private MediaSendLogRepository mediaSendLogRepository;

    @Mock
    private ServerConfig serverConfig;

    @InjectMocks
    private MediaUploadService service;

    @BeforeEach
    void setUp() {
        // 注入配置，避免 NPE (虽然本测试可能用不到)
        ReflectionTestUtils.setField(service, "serverConfig", serverConfig);
    }

    @Test
    @DisplayName("保存上传记录 - MD5已存在则更新时间")
    void saveUploadRecord_ShouldUpdate_WhenMd5Exists() {
        // Arrange
        String userId = "u1";
        String md5 = "hash123";
        MediaUploadHistory history = new MediaUploadHistory();
        history.setUserId(userId);
        history.setFileMd5(md5);

        MediaFile existing = new MediaFile();
        existing.setId(1L);
        existing.setFileMd5(md5);
        
        when(mediaFileRepository.findFirstByUserIdAndFileMd5(userId, md5))
                .thenReturn(Optional.of(existing));

        // Act
        MediaUploadHistory result = service.saveUploadRecord(history);

        // Assert
        assertNotNull(result);
        assertEquals(1L, result.getId());
        verify(mediaFileRepository, times(1)).save(existing); // Should update existing
        // verify(mediaFileRepository, never()).save(any(MediaFile.class)); // Removed conflicting assertion
    }

    @Test
    @DisplayName("保存上传记录 - MD5不存在则插入新记录")
    void saveUploadRecord_ShouldInsert_WhenMd5NotExists() {
        // Arrange
        String userId = "u1";
        String md5 = "hashNew";
        MediaUploadHistory history = new MediaUploadHistory();
        history.setUserId(userId);
        history.setFileMd5(md5);
        history.setOriginalFilename("test.jpg");

        when(mediaFileRepository.findFirstByUserIdAndFileMd5(userId, md5))
                .thenReturn(Optional.empty());
        
        when(mediaFileRepository.save(any(MediaFile.class))).thenAnswer(invocation -> {
            MediaFile f = invocation.getArgument(0);
            f.setId(2L);
            return f;
        });

        // Act
        MediaUploadHistory result = service.saveUploadRecord(history);

        // Assert
        assertNotNull(result);
        assertEquals(2L, result.getId());
        assertEquals("test.jpg", result.getOriginalFilename());
    }

    @Test
    @DisplayName("记录图片发送 - 成功匹配本地文件名")
    void recordImageSend_ShouldMatchByLocalFilename() {
        // Arrange
        String fromUser = "u1";
        String toUser = "u2";
        String localName = "img_123.jpg";
        String remoteUrl = "http://remote/img_123.jpg";

        MediaFile file = new MediaFile();
        file.setId(10L);
        file.setLocalPath("path/to/img_123.jpg");

        // Mock 匹配逻辑
        when(mediaFileRepository.findFirstByLocalFilenameAndUserId(localName, fromUser))
                .thenReturn(Optional.of(file));
        
        // Mock 日志不存在
        when(mediaSendLogRepository.findFirstByRemoteUrlAndUserIdAndToUserId(remoteUrl, fromUser, toUser))
                .thenReturn(Optional.empty());

        // Act
        MediaUploadHistory result = service.recordImageSend(remoteUrl, fromUser, toUser, localName);

        // Assert
        assertNotNull(result);
        assertEquals(10L, result.getId());
        verify(mediaSendLogRepository, times(1)).save(any(MediaSendLog.class));
        verify(mediaFileRepository, times(1)).save(file); // Update updateTime
    }

    @Test
    @DisplayName("记录图片发送 - 匹配失败返回Null")
    void recordImageSend_ShouldReturnNull_WhenNoMatch() {
        // Arrange
        String fromUser = "u1";
        
        // Mock 所有查询都为空
        when(mediaFileRepository.findFirstByLocalFilenameAndUserId(anyString(), anyString())).thenReturn(Optional.empty());
        when(mediaFileRepository.findFirstByLocalFilename(anyString())).thenReturn(Optional.empty());
        // ... (其他查询默认返回 empty)

        // Act
        MediaUploadHistory result = service.recordImageSend("http://url", fromUser, "u2", "name.jpg");

        // Assert
        assertNull(result);
    }
}
