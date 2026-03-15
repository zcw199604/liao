package io.github.a7413498.liao.android.feature.media

import io.github.a7413498.liao.android.core.common.ChatMessageType
import org.junit.Assert.assertEquals
import org.junit.Test

class MediaLibraryHelpersTest {
    @Test
    fun `open mime type should align with media type`() {
        assertEquals("image/*", ChatMessageType.IMAGE.openMimeType())
        assertEquals("video/*", ChatMessageType.VIDEO.openMimeType())
        assertEquals("*/*", ChatMessageType.FILE.openMimeType())
        assertEquals("text/plain", ChatMessageType.TEXT.openMimeType())
    }

    @Test
    fun `display label should include file name when available`() {
        assertEquals("图片 · a.png", ChatMessageType.IMAGE.displayLabel("a.png"))
        assertEquals("视频", ChatMessageType.VIDEO.displayLabel(""))
        assertEquals("文件 · report.pdf", ChatMessageType.FILE.displayLabel("report.pdf"))
        assertEquals("文本", ChatMessageType.TEXT.displayLabel("文本"))
    }

    @Test
    fun `chat message type helper should respect explicit type and fallback to url`() {
        assertEquals(ChatMessageType.IMAGE, " image ".toChatMessageType("https://x/y.mp4"))
        assertEquals(ChatMessageType.VIDEO, "video".toChatMessageType("https://x/y.jpg"))
        assertEquals(ChatMessageType.FILE, "file".toChatMessageType("https://x/y.jpg"))
        assertEquals(ChatMessageType.IMAGE, null.toChatMessageType("https://x/y.jpeg"))
        assertEquals(ChatMessageType.FILE, "unknown".toChatMessageType("https://x/y.bin"))
    }

    @Test
    fun `extract upload local path should normalize upload prefixes and query`() {
        assertEquals("/foo/bar.jpg", extractUploadLocalPath("https://example.com/upload/foo/bar.jpg?x=1#hash"))
        assertEquals("/foo/bar.jpg", extractUploadLocalPath("upload/foo/bar.jpg"))
        assertEquals("/foo/bar.jpg", extractUploadLocalPath("foo/bar.jpg"))
        assertEquals("/already/abs.jpg", extractUploadLocalPath("/already/abs.jpg"))
        assertEquals("", extractUploadLocalPath("   "))
    }

    @Test
    fun `format file size should choose appropriate unit`() {
        assertEquals("512 B", formatFileSize(512))
        assertEquals("1.0 KB", formatFileSize(1024))
        assertEquals("1.0 MB", formatFileSize(1024 * 1024L))
        assertEquals("1.0 GB", formatFileSize(1024 * 1024 * 1024L))
    }
}
