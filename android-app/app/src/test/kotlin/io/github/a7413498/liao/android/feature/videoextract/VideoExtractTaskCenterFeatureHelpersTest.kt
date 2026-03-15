package io.github.a7413498.liao.android.feature.videoextract

import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoExtractTaskCenterFeatureHelpersTest {
    private fun baseTask(
        taskId: String = "task-1",
        sourceRef: String = "/upload/source/demo.mp4",
        status: String = "RUNNING",
        mode: String = "fps",
    ) = VideoExtractTaskItem(
        taskId = taskId,
        sourceType = "upload",
        sourceRef = sourceRef,
        sourcePreviewUrl = "https://demo.test$sourceRef",
        outputDirLocalPath = "/tmp/out",
        outputDirUrl = "/upload/out",
        outputFormat = "jpg",
        jpgQuality = 12,
        mode = mode,
        keyframeMode = "scene",
        fps = 2.5,
        sceneThreshold = 0.4,
        startSec = 1.0,
        endSec = 9.0,
        maxFrames = 30,
        framesExtracted = 12,
        videoWidth = 1920,
        videoHeight = 1080,
        durationSec = 100.0,
        cursorOutTimeSec = 25.0,
        status = status,
        stopReason = "",
        lastError = "",
        createdAt = "2026-03-15T10:20:00",
        updatedAt = "2026-03-15T10:21:00",
        runtimeLogs = listOf("a"),
    )

    @Test
    fun `task item display helpers should cover title subtitle and status branches`() {
        assertEquals("demo.mp4", baseTask().displayTitle())
        assertEquals("task-2", baseTask(taskId = "task-2", sourceRef = "").displayTitle())
        assertEquals("2026-03-15 · 1920×1080 · 1:40", baseTask().displaySubtitle())
        assertEquals("", baseTask(status = "", mode = "all").copy(createdAt = "", videoWidth = 0, videoHeight = 0, durationSec = null).displaySubtitle())

        assertEquals("排队中", baseTask(status = "PENDING").statusText())
        assertEquals("准备中", baseTask(status = "PREPARING").statusText())
        assertEquals("运行中", baseTask(status = "RUNNING").statusText())
        assertEquals("已终止", baseTask(status = "PAUSED_USER").statusText())
        assertEquals("因限制暂停", baseTask(status = "PAUSED_LIMIT").statusText())
        assertEquals("已完成", baseTask(status = "FINISHED").statusText())
        assertEquals("失败", baseTask(status = "FAILED").statusText())
        assertEquals("custom", baseTask(status = "custom").statusText())
        assertEquals("未知", baseTask(status = "").statusText())
    }

    @Test
    fun `task item mode progress limit and action helpers should cover branches`() {
        assertEquals("固定 FPS 2.5", baseTask(mode = "fps").modeText())
        assertEquals("固定 FPS", baseTask(mode = "fps").copy(fps = null).modeText())
        assertEquals("逐帧输出", baseTask(mode = "all").modeText())
        assertEquals("关键帧(场景 0.4)", baseTask(mode = "keyframe").modeText())
        assertEquals("关键帧(I 帧)", baseTask(mode = "keyframe").copy(keyframeMode = "iframe").modeText())
        assertEquals("other", baseTask(mode = "other").modeText())

        assertEquals("进度 25%", baseTask().progressText())
        assertEquals("已输出 12/30 张", baseTask().copy(durationSec = null).progressText())
        assertEquals("已输出 12/30 张", baseTask().copy(cursorOutTimeSec = 0.0).progressText())

        assertEquals("时间区间：1.0 ~ 9.0s；maxFrames=30；格式=JPG / Q=12", baseTask().limitText())
        assertEquals("时间区间：全程；maxFrames=30；格式=PNG", baseTask().copy(startSec = null, endSec = null, outputFormat = "png", jpgQuality = null).limitText())

        assertTrue(baseTask(status = "RUNNING").canCancel())
        assertTrue(baseTask(status = "PREPARING").canCancel())
        assertFalse(baseTask(status = "FINISHED").canCancel())
        assertTrue(baseTask(status = "PAUSED_LIMIT").canContinue())
        assertTrue(baseTask(status = "PAUSED_USER").canContinue())
        assertFalse(baseTask(status = "FAILED").canContinue())
    }

    @Test
    fun `task item mapping should cover preview url runtime logs and field defaults`() {
        val item = buildJsonObject {
            put("taskId", JsonPrimitive("task-1"))
            put("sourceType", JsonPrimitive("upload"))
            put("sourceRef", JsonPrimitive("videos/demo.mp4"))
            put("outputDirLocalPath", JsonPrimitive("/tmp/out"))
            put("outputDirUrl", JsonPrimitive("/upload/out"))
            put("outputFormat", JsonPrimitive("jpg"))
            put("jpgQuality", JsonPrimitive("18"))
            put("mode", JsonPrimitive("keyframe"))
            put("keyframeMode", JsonPrimitive("scene"))
            put("fps", JsonPrimitive("3.5"))
            put("sceneThreshold", JsonPrimitive("0.4"))
            put("startSec", JsonPrimitive("1.5"))
            put("endSec", JsonPrimitive("9.5"))
            put("maxFrames", JsonPrimitive("40"))
            put("framesExtracted", JsonPrimitive("6"))
            put("videoWidth", JsonPrimitive("720"))
            put("videoHeight", JsonPrimitive("1280"))
            put("durationSec", JsonPrimitive("90.5"))
            put("cursorOutTimeSec", JsonPrimitive("30.0"))
            put("status", JsonPrimitive("RUNNING"))
            put("stopReason", JsonPrimitive(""))
            put("lastError", JsonPrimitive("boom"))
            put("createdAt", JsonPrimitive("2026-03-15T12:00:00"))
            put("updatedAt", JsonPrimitive("2026-03-15T12:01:00"))
            put("runtime", buildJsonObject {
                put("logs", buildJsonArray {
                    add(JsonPrimitive("line-1"))
                    add(buildJsonObject { put("bad", JsonPrimitive("x")) })
                    add(JsonPrimitive("line-2"))
                })
            })
        }.toTaskItem("https://demo.test")

        requireNotNull(item)
        assertEquals("https://demo.test/upload/videos/demo.mp4", item.sourcePreviewUrl)
        assertEquals(18, item.jpgQuality)
        assertEquals(40, item.maxFrames)
        assertEquals(6, item.framesExtracted)
        assertEquals(720, item.videoWidth)
        assertEquals(1280, item.videoHeight)
        assertEquals(90.5, item.durationSec ?: 0.0, 0.0)
        assertEquals(listOf("line-1", "line-2"), item.runtimeLogs)
        assertEquals("boom", item.lastError)

        val fallback = buildJsonObject {
            put("taskId", JsonPrimitive("task-2"))
            put("sourceType", JsonPrimitive("remote"))
            put("sourceRef", JsonPrimitive(" "))
            put("maxFrames", JsonPrimitive("bad"))
        }.toTaskItem("https://demo.test")
        requireNotNull(fallback)
        assertEquals("", fallback.sourcePreviewUrl)
        assertEquals(0, fallback.maxFrames)
        assertEquals(0, fallback.videoWidth)
        assertEquals(emptyList<String>(), fallback.runtimeLogs)

        assertNull(buildJsonObject { put("sourceType", JsonPrimitive("upload")) }.toTaskItem("https://demo.test"))
    }

    @Test
    fun `frame and json helpers should cover fallback branches`() {
        val frame = buildJsonObject {
            put("seq", JsonPrimitive("3"))
            put("url", JsonPrimitive("https://demo.test/f3.jpg"))
        }.toFrameItem()
        requireNotNull(frame)
        assertEquals(3, frame.seq)
        assertEquals("https://demo.test/f3.jpg", frame.url)
        assertNull(buildJsonObject { put("url", JsonPrimitive("x")) }.toFrameItem())
        assertNull(buildJsonObject { put("seq", JsonPrimitive("1")) }.toFrameItem())

        assertEquals(9, buildJsonObject { put("value", JsonPrimitive("9")) }.intOrNull("value"))
        assertNull(buildJsonObject { put("value", JsonPrimitive("bad")) }.intOrNull("value"))
        assertTrue(buildJsonObject { put("enabled", JsonPrimitive(true)) }.booleanOrDefault("enabled", false))
        assertFalse(buildJsonObject {}.booleanOrDefault("enabled", false))
    }

    @Test
    fun `upload preview url should cover source type path normalization branches`() {
        assertEquals("", buildUploadPreviewUrl("https://demo.test", "remote", "/upload/a.mp4"))
        assertEquals("", buildUploadPreviewUrl("https://demo.test", "upload", "  "))
        assertEquals("https://files.test/demo.mp4", buildUploadPreviewUrl("https://demo.test", "upload", "https://files.test/demo.mp4"))
        assertEquals("https://demo.test/upload/a.mp4", buildUploadPreviewUrl("https://demo.test", "upload", "/upload/a.mp4"))
        assertEquals("https://demo.test/upload/videos/demo.mp4", buildUploadPreviewUrl("https://demo.test", "upload", "videos/demo.mp4"))
    }
}
