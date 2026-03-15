package io.github.a7413498.liao.android.feature.videoextract

import io.github.a7413498.liao.android.core.common.AppResult
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoExtractCreateFeatureHelpersTest {
    private val source = VideoExtractUploadedSource(
        displayName = "demo.mp4",
        mimeType = "video/mp4",
        size = 1024,
        localPath = "/tmp/demo.mp4",
        localFilename = "demo.mp4",
        originalFilename = "demo.mp4",
    )

    private val probe = VideoExtractProbeSummary(
        durationSec = 120.0,
        width = 1920,
        height = 1080,
        avgFps = 24.0,
    )

    private fun baseState() = VideoExtractCreateUiState(
        source = source,
        probe = probe,
        maxFrames = "120",
        outputFormat = VideoExtractOutputFormatOption.JPG,
    )

    @Test
    fun `validation error should cover required field and range branches`() {
        assertEquals("请先选择本地视频", VideoExtractCreateUiState().validationError())
        assertEquals("探测失败", baseState().copy(probe = null, probeError = "探测失败").validationError())
        assertEquals("请先完成视频探测", baseState().copy(probe = null, probeError = null).validationError())
        assertEquals("maxFrames 必须为正整数", baseState().copy(maxFrames = "abc").validationError())
        assertEquals("maxFrames 必须大于 0", baseState().copy(maxFrames = "0").validationError())
        assertEquals("startSec 格式非法", baseState().copy(startSec = "x").validationError())
        assertEquals("endSec 格式非法", baseState().copy(endSec = "x").validationError())
        assertEquals("startSec 不能小于 0", baseState().copy(startSec = "-1").validationError())
        assertEquals("endSec 不能小于 0", baseState().copy(endSec = "-1").validationError())
        assertEquals("endSec 必须大于 startSec", baseState().copy(startSec = "8", endSec = "8").validationError())
    }

    @Test
    fun `validation error should cover mode specific branches`() {
        assertEquals(
            "fps 必须大于 0",
            baseState().copy(mode = VideoExtractModeOption.FPS, fps = " ").validationError(),
        )
        assertEquals(
            "fps 必须大于 0",
            baseState().copy(mode = VideoExtractModeOption.FPS, fps = "0").validationError(),
        )
        assertEquals(
            "sceneThreshold 范围为 0-1",
            baseState().copy(
                keyframeMode = VideoExtractKeyframeModeOption.SCENE,
                sceneThreshold = "bad",
            ).validationError(),
        )
        assertEquals(
            "sceneThreshold 范围为 0-1",
            baseState().copy(
                keyframeMode = VideoExtractKeyframeModeOption.SCENE,
                sceneThreshold = "1.5",
            ).validationError(),
        )
        assertEquals(
            "jpgQuality 范围为 1-31",
            baseState().copy(jpgQuality = "bad").validationError(),
        )
        assertEquals(
            "jpgQuality 范围为 1-31",
            baseState().copy(jpgQuality = "32").validationError(),
        )
        assertNull(
            baseState().copy(
                mode = VideoExtractModeOption.FPS,
                fps = "2.5",
                outputFormat = VideoExtractOutputFormatOption.PNG,
                jpgQuality = "",
            ).validationError(),
        )
    }

    @Test
    fun `to create payload should surface validation errors`() {
        val result = baseState().copy(mode = VideoExtractModeOption.FPS, fps = "0").toCreatePayloadOrError()

        assertTrue(result is AppResult.Error)
        assertEquals("fps 必须大于 0", (result as AppResult.Error).message)
    }

    @Test
    fun `to create payload should map optional fields by mode and format`() {
        val fpsResult = baseState().copy(
            mode = VideoExtractModeOption.FPS,
            fps = "2.5",
            startSec = "1.5",
            endSec = "5",
            jpgQuality = "12",
        ).toCreatePayloadOrError()
        assertTrue(fpsResult is AppResult.Success)
        val fpsPayload = (fpsResult as AppResult.Success).data
        assertEquals(VideoExtractModeOption.FPS, fpsPayload.mode)
        assertEquals(2.5, fpsPayload.fps)
        assertEquals(1.5, fpsPayload.startSec)
        assertEquals(5.0, fpsPayload.endSec)
        assertEquals(12, fpsPayload.jpgQuality)
        assertNull(fpsPayload.sceneThreshold)

        val sceneResult = baseState().copy(
            keyframeMode = VideoExtractKeyframeModeOption.SCENE,
            sceneThreshold = "0.45",
            outputFormat = VideoExtractOutputFormatOption.PNG,
            jpgQuality = "8",
        ).toCreatePayloadOrError()
        assertTrue(sceneResult is AppResult.Success)
        val scenePayload = (sceneResult as AppResult.Success).data
        assertEquals(VideoExtractKeyframeModeOption.SCENE, scenePayload.keyframeMode)
        assertEquals(0.45, scenePayload.sceneThreshold)
        assertNull(scenePayload.fps)
        assertNull(scenePayload.jpgQuality)
    }

    @Test
    fun `probe summary helpers should map values and defaults`() {
        val summary = buildJsonObject {
            put("durationSec", JsonPrimitive("15.5"))
            put("width", JsonPrimitive("1280"))
            put("height", JsonPrimitive("720"))
            put("avgFps", JsonPrimitive("29.97"))
        }.toProbeSummary()
        assertEquals(15.5, summary.durationSec, 0.0)
        assertEquals(1280, summary.width)
        assertEquals(720, summary.height)
        assertEquals(29.97, summary.avgFps ?: 0.0, 0.0)

        val defaults = buildJsonObject {
            put("durationSec", JsonPrimitive("bad"))
            put("width", JsonPrimitive("bad"))
            put("height", JsonPrimitive("bad"))
        }.toProbeSummary()
        assertEquals(0.0, defaults.durationSec, 0.0)
        assertEquals(0, defaults.width)
        assertEquals(0, defaults.height)
        assertNull(defaults.avgFps)
        assertEquals(42L, buildJsonObject { put("value", JsonPrimitive("42")) }.longOrDefault("value", 1L))
        assertEquals(7L, buildJsonObject {}.longOrDefault("value", 7L))
    }

    @Test
    fun `estimate frames should cover mode specific branches`() {
        assertNull(estimateFrames(baseState().copy(probe = null)))
        assertNull(estimateFrames(baseState().copy(startSec = "10", endSec = "5")))
        assertNull(estimateFrames(baseState().copy(mode = VideoExtractModeOption.FPS, fps = "bad")))
        assertNull(estimateFrames(baseState().copy(mode = VideoExtractModeOption.FPS, fps = "0")))
        assertEquals(20, estimateFrames(baseState().copy(mode = VideoExtractModeOption.FPS, fps = "2", endSec = "10")))
        assertNull(estimateFrames(baseState().copy(mode = VideoExtractModeOption.ALL, probe = probe.copy(avgFps = null))))
        assertNull(estimateFrames(baseState().copy(mode = VideoExtractModeOption.ALL, probe = probe.copy(avgFps = 0.0))))
        assertEquals(240, estimateFrames(baseState().copy(mode = VideoExtractModeOption.ALL, endSec = "10")))
        assertNull(estimateFrames(baseState().copy(mode = VideoExtractModeOption.KEYFRAME)))
    }

    @Test
    fun `format helpers should cover duration and file size branches`() {
        assertEquals("-", formatDuration(null))
        assertEquals("-", formatDuration(0.0))
        assertEquals("1:05", formatDuration(65.0))
        assertEquals("1:01:01", formatDuration(3661.0))

        assertEquals("未知", 0L.toReadableSize())
        assertEquals("512 B", 512L.toReadableSize())
        assertEquals("1.00 KB", 1024L.toReadableSize())
        assertEquals("1.00 MB", (1024L * 1024L).toReadableSize())
        assertEquals("1.00 GB", (1024L * 1024L * 1024L).toReadableSize())
    }
}
