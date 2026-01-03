package com.zcw.service;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.test.util.ReflectionTestUtils;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("文件存储服务测试")
class FileStorageServiceTest {

    @TempDir
    Path tempDir;

    @Test
    @DisplayName("保存文件 - 成功")
    void saveFile_ShouldSaveToDisk() throws IOException {
        // Arrange
        FileStorageService service = new FileStorageService();
        // 修改 BASE_UPLOAD_PATH 为临时目录，避免污染环境
        ReflectionTestUtils.setField(service, "BASE_UPLOAD_PATH", tempDir.toString());

        MockMultipartFile file = new MockMultipartFile(
                "file", "test.jpg", "image/jpeg", "content".getBytes());

        // Act
        String path = service.saveFile(file, "image");

        // Assert
        assertNotNull(path);
        assertTrue(path.startsWith("/image")); // 相对路径
        
        // 验证文件存在
        File savedFile = new File(tempDir.toString(), path);
        assertTrue(savedFile.exists());
        assertArrayEquals("content".getBytes(), Files.readAllBytes(savedFile.toPath()));
    }

    @Test
    @DisplayName("保存文件 - 空文件抛出异常")
    void saveFile_ShouldThrow_WhenFileEmpty() {
        FileStorageService service = new FileStorageService();
        MockMultipartFile file = new MockMultipartFile("file", new byte[0]);

        assertThrows(IOException.class, () -> service.saveFile(file, "image"));
    }

    @Test
    @DisplayName("验证媒体类型")
    void isValidMediaType_ShouldReturnCorrectly() {
        FileStorageService service = new FileStorageService();
        
        assertTrue(service.isValidMediaType("image/jpeg"));
        assertTrue(service.isValidMediaType("video/mp4"));
        assertFalse(service.isValidMediaType("application/pdf"));
        assertFalse(service.isValidMediaType(null));
    }

    @Test
    @DisplayName("计算MD5")
    void calculateMD5_ShouldReturnCorrectHash() throws Exception {
        FileStorageService service = new FileStorageService();
        MockMultipartFile file = new MockMultipartFile(
                "file", "test.txt", "text/plain", "hello".getBytes());
        
        // "hello" MD5 = 5d41402abc4b2a76b9719d911017c592
        String md5 = service.calculateMD5(file);
        assertEquals("5d41402abc4b2a76b9719d911017c592", md5);
    }
}
