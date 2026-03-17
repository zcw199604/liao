package io.github.a7413498.liao.android.feature.videoextract

import android.content.ContentResolver
import android.content.Context
import android.database.Cursor
import android.net.Uri
import android.provider.OpenableColumns
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.VideoExtractApiService
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import java.io.ByteArrayInputStream
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MultipartBody
import okio.Buffer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class VideoExtractCreateRepositoryTest {
    private val apiService = mockk<VideoExtractApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val appContext = mockk<Context>(relaxed = true)
    private val contentResolver = mockk<ContentResolver>(relaxed = true)

    private val repository = VideoExtractCreateRepository(apiService, preferencesStore, appContext)

    @Before
    fun setUp() {
        every { appContext.contentResolver } returns contentResolver
    }

    @Test
    fun `upload video should build multipart from resolver and map response fallbacks`() = runTest {
        val uri = mockk<Uri>(relaxed = true)
        stubFileMeta(uri = uri, displayName = "demo.mp4", size = 4L, mimeType = null, bytes = "demo".toByteArray())
        val partSlot = slot<MultipartBody.Part>()
        coEvery { apiService.uploadVideoExtractInput(capture(partSlot)) } returns ApiEnvelope(
            code = 0,
            data = buildJsonObject {
                put("contentType", JsonPrimitive(""))
                put("fileSize", JsonPrimitive("bad"))
                put("localPath", JsonPrimitive("/upload/demo.mp4"))
                put("localFilename", JsonPrimitive(""))
                put("originalFilename", JsonPrimitive(""))
            },
        )

        val result = repository.uploadVideo(uri)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("demo.mp4", payload.displayName)
        assertEquals("video/mp4", payload.mimeType)
        assertEquals(4L, payload.size)
        assertEquals("/upload/demo.mp4", payload.localPath)
        assertEquals("demo.mp4", payload.localFilename)
        assertEquals("demo.mp4", payload.originalFilename)

        val part = partSlot.captured
        assertTrue(part.headers?.get("Content-Disposition")?.contains("demo.mp4") == true)
        assertEquals("video/mp4", part.body.contentType()?.toString())
        assertEquals(4L, part.body.contentLength())
        val buffer = Buffer()
        part.body.writeTo(buffer)
        assertEquals("demo", buffer.readUtf8())
    }

    @Test
    fun `upload video should surface api error and missing local path`() = runTest {
        val uri = mockk<Uri>(relaxed = true)
        stubFileMeta(uri = uri, displayName = "error.mp4", size = 2L, mimeType = "video/mp4", bytes = byteArrayOf(1, 2))
        coEvery { apiService.uploadVideoExtractInput(any()) } returnsMany listOf(
            ApiEnvelope(code = 1, msg = "上传失败"),
            ApiEnvelope(code = 0, data = buildJsonObject { put("fileSize", JsonPrimitive(2)) }),
        )

        val apiError = repository.uploadVideo(uri)
        val missingPath = repository.uploadVideo(uri)

        assertTrue(apiError is AppResult.Error)
        assertEquals("上传失败", (apiError as AppResult.Error).message)
        assertTrue(missingPath is AppResult.Error)
        assertEquals("上传结果缺少 localPath", (missingPath as AppResult.Error).message)
    }

    @Test
    fun `probe upload should map payload and surface envelope error`() = runTest {
        coEvery { apiService.probeVideo(sourceType = "upload", localPath = "/upload/demo.mp4") } returnsMany listOf(
            ApiEnvelope(
                code = 0,
                data = buildJsonObject {
                    put("durationSec", JsonPrimitive("12.5"))
                    put("width", JsonPrimitive("1280"))
                    put("height", JsonPrimitive("720"))
                    put("avgFps", JsonPrimitive("29.97"))
                },
            ),
            ApiEnvelope(code = 1, msg = "探测失败"),
        )

        val success = repository.probeUpload("/upload/demo.mp4")
        val failure = repository.probeUpload("/upload/demo.mp4")

        assertTrue(success is AppResult.Success)
        val summary = (success as AppResult.Success).data
        assertEquals(12.5, summary.durationSec, 0.0)
        assertEquals(1280, summary.width)
        assertEquals(720, summary.height)
        assertEquals(29.97, summary.avgFps ?: 0.0, 0.0)
        assertTrue(failure is AppResult.Error)
        assertEquals("探测失败", (failure as AppResult.Error).message)
    }

    @Test
    fun `create task should include scene payload and map response probe`() = runTest {
        val payloadSlot = slot<JsonElement>()
        coEvery { preferencesStore.readCurrentSession() } returns sampleSession(id = "self-1")
        coEvery { apiService.createTask(capture(payloadSlot)) } returns ApiEnvelope(
            code = 0,
            data = buildJsonObject {
                put("taskId", JsonPrimitive("task-1"))
                put(
                    "probe",
                    buildJsonObject {
                        put("durationSec", JsonPrimitive("60"))
                        put("width", JsonPrimitive("1920"))
                        put("height", JsonPrimitive("1080"))
                        put("avgFps", JsonPrimitive("24"))
                    },
                )
            },
        )

        val result = repository.createTask(
            localPath = "/upload/demo.mp4",
            payload = VideoExtractCreatePayload(
                mode = VideoExtractModeOption.KEYFRAME,
                keyframeMode = VideoExtractKeyframeModeOption.SCENE,
                sceneThreshold = 0.45,
                fps = null,
                startSec = 1.5,
                endSec = 5.5,
                maxFrames = 120,
                outputFormat = VideoExtractOutputFormatOption.JPG,
                jpgQuality = 12,
            ),
        )

        assertTrue(result is AppResult.Success)
        val created = (result as AppResult.Success).data
        assertEquals("task-1", created.taskId)
        assertEquals(60.0, created.probe?.durationSec ?: 0.0, 0.0)
        val json = payloadSlot.captured.jsonObject
        assertEquals("self-1", json["userId"]?.jsonPrimitive?.content)
        assertEquals("upload", json["sourceType"]?.jsonPrimitive?.content)
        assertEquals("/upload/demo.mp4", json["localPath"]?.jsonPrimitive?.content)
        assertEquals("keyframe", json["mode"]?.jsonPrimitive?.content)
        assertEquals("scene", json["keyframeMode"]?.jsonPrimitive?.content)
        assertEquals("0.45", json["sceneThreshold"]?.jsonPrimitive?.content)
        assertNull(json["fps"])
        assertEquals("1.5", json["startSec"]?.jsonPrimitive?.content)
        assertEquals("5.5", json["endSec"]?.jsonPrimitive?.content)
        assertEquals("120", json["maxFrames"]?.jsonPrimitive?.content)
        assertEquals("jpg", json["outputFormat"]?.jsonPrimitive?.content)
        assertEquals("12", json["jpgQuality"]?.jsonPrimitive?.content)
    }

    @Test
    fun `create task should omit optional fields for blank session fps mode and png output`() = runTest {
        val payloadSlot = slot<JsonElement>()
        coEvery { preferencesStore.readCurrentSession() } returns null
        coEvery { apiService.createTask(capture(payloadSlot)) } returns ApiEnvelope(
            code = 0,
            data = buildJsonObject {
                put("taskId", JsonPrimitive("task-2"))
            },
        )

        val result = repository.createTask(
            localPath = "/upload/fps.mp4",
            payload = VideoExtractCreatePayload(
                mode = VideoExtractModeOption.FPS,
                keyframeMode = VideoExtractKeyframeModeOption.IFRAME,
                sceneThreshold = null,
                fps = 2.5,
                startSec = null,
                endSec = null,
                maxFrames = 80,
                outputFormat = VideoExtractOutputFormatOption.PNG,
                jpgQuality = 8,
            ),
        )

        assertTrue(result is AppResult.Success)
        val json = payloadSlot.captured.jsonObject
        assertFalse(json.containsKey("userId"))
        assertEquals("fps", json["mode"]?.jsonPrimitive?.content)
        assertEquals("2.5", json["fps"]?.jsonPrimitive?.content)
        assertFalse(json.containsKey("keyframeMode"))
        assertFalse(json.containsKey("sceneThreshold"))
        assertEquals("png", json["outputFormat"]?.jsonPrimitive?.content)
        assertFalse(json.containsKey("jpgQuality"))
    }

    @Test
    fun `create task should surface remote error and missing task id`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns sampleSession()
        coEvery { apiService.createTask(any()) } returnsMany listOf(
            ApiEnvelope(code = 1, msg = "创建失败"),
            ApiEnvelope(code = 0, data = buildJsonObject { put("probe", buildJsonObject {}) }),
        )

        val remoteError = repository.createTask(localPath = "/upload/demo.mp4", payload = samplePayload())
        val missingTaskId = repository.createTask(localPath = "/upload/demo.mp4", payload = samplePayload())

        assertTrue(remoteError is AppResult.Error)
        assertEquals("创建失败", (remoteError as AppResult.Error).message)
        assertTrue(missingTaskId is AppResult.Error)
        assertEquals("创建结果缺少 taskId", (missingTaskId as AppResult.Error).message)
        coVerify(exactly = 2) { apiService.createTask(any()) }
    }

    private fun stubFileMeta(
        uri: Uri,
        displayName: String,
        size: Long,
        mimeType: String?,
        bytes: ByteArray,
    ) {
        val cursor = mockk<Cursor>()
        every { contentResolver.query(eq(uri), any(), any(), any(), any()) } returns cursor
        every { cursor.moveToFirst() } returns true
        every { cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME) } returns 0
        every { cursor.getColumnIndex(OpenableColumns.SIZE) } returns 1
        every { cursor.getString(0) } returns displayName
        every { cursor.getLong(1) } returns size
        every { cursor.close() } just runs
        every { contentResolver.getType(uri) } returns mimeType
        every { contentResolver.openInputStream(uri) } returns ByteArrayInputStream(bytes)
    }

    private fun sampleSession(id: String = "self-1") = CurrentIdentitySession(
        id = id,
        name = "Alice",
        sex = "女",
        cookie = "cookie",
        ip = "127.0.0.1",
        area = "Shenzhen",
    )

    private fun samplePayload() = VideoExtractCreatePayload(
        mode = VideoExtractModeOption.KEYFRAME,
        keyframeMode = VideoExtractKeyframeModeOption.IFRAME,
        sceneThreshold = null,
        fps = null,
        startSec = null,
        endSec = null,
        maxFrames = 20,
        outputFormat = VideoExtractOutputFormatOption.JPG,
        jpgQuality = null,
    )
}
