package io.github.a7413498.liao.android

import io.github.a7413498.liao.android.core.common.generateCookie
import io.github.a7413498.liao.android.core.common.generateRandomIp
import io.github.a7413498.liao.android.core.common.normalizeTextForMatch
import io.github.a7413498.liao.android.core.datastore.CachedMediaLibraryItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedMediaLibrarySnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractFrameItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskDetailSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskDetailsSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskItemSnapshot
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class CommonAndCacheSnapshotTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `generate cookie should keep expected structure`() {
        val cookie = generateCookie(userId = "user1", nickname = "Alice")
        val segments = cookie.split('_')

        assertEquals(4, segments.size)
        assertEquals("user1", segments[0])
        assertEquals("Alice", segments[1])
        assertTrue(segments[2].all(Char::isDigit))
        assertEquals(6, segments[3].length)
        assertTrue(segments[3].all { it in 'a'..'z' })
    }

    @Test
    fun `generate random ip should stay within ipv4 octet range`() {
        repeat(20) {
            val parts = generateRandomIp().split('.')
            assertEquals(4, parts.size)
            parts.forEach { part ->
                val octet = part.toInt()
                assertTrue(octet in 0..255)
            }
        }
    }

    @Test
    fun `normalize text should trim and convert crlf to lf`() {
        val normalized = normalizeTextForMatch("  hello\r\nworld\r\n  ")
        assertEquals("hello\nworld", normalized)
    }

    @Test
    fun `cached media library snapshot should roundtrip through json`() {
        val snapshot = CachedMediaLibrarySnapshot(
            items = listOf(
                CachedMediaLibraryItemSnapshot(
                    url = "https://example.com/a.jpg",
                    localPath = "/tmp/a.jpg",
                    type = "image",
                    title = "A",
                    subtitle = "B",
                    posterUrl = "",
                    updateTime = "2026-03-15 10:00:00",
                    source = "upload",
                ),
            ),
            page = 2,
            total = 21,
            totalPages = 3,
        )

        val encoded = json.encodeToString(CachedMediaLibrarySnapshot.serializer(), snapshot)
        val decoded = json.decodeFromString(CachedMediaLibrarySnapshot.serializer(), encoded)

        assertEquals(snapshot, decoded)
    }

    @Test
    fun `cached video extract detail snapshot should roundtrip through json`() {
        val detail = CachedVideoExtractTaskDetailSnapshot(
            task = CachedVideoExtractTaskItemSnapshot(
                taskId = "task-1",
                sourceType = "upload",
                sourceRef = "source-1",
                sourcePreviewUrl = "https://example.com/video.mp4",
                outputDirLocalPath = "/tmp/task-1",
                outputDirUrl = "https://example.com/out/task-1",
                outputFormat = "jpg",
                jpgQuality = 90,
                mode = "fps",
                keyframeMode = null,
                fps = 1.0,
                sceneThreshold = null,
                startSec = 0.0,
                endSec = 8.0,
                maxFrames = 8,
                framesExtracted = 3,
                videoWidth = 1920,
                videoHeight = 1080,
                durationSec = 8.2,
                cursorOutTimeSec = 3.0,
                status = "running",
                stopReason = "",
                lastError = "",
                createdAt = "2026-03-15 10:01:00",
                updatedAt = "2026-03-15 10:02:00",
                runtimeLogs = listOf("start", "frame-1"),
            ),
            frames = listOf(
                CachedVideoExtractFrameItemSnapshot(seq = 1, url = "https://example.com/frame-1.jpg"),
            ),
            nextCursor = 2,
            hasMore = true,
        )

        val encoded = json.encodeToString(CachedVideoExtractTaskDetailSnapshot.serializer(), detail)
        val decoded = json.decodeFromString(CachedVideoExtractTaskDetailSnapshot.serializer(), encoded)

        assertEquals(detail, decoded)
    }

    @Test
    fun `cached video extract details snapshot should preserve default empty list`() {
        val snapshot = json.decodeFromString(
            CachedVideoExtractTaskDetailsSnapshot.serializer(),
            "{}",
        )

        assertTrue(snapshot.items.isEmpty())
    }
}
