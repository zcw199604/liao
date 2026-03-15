package io.github.a7413498.liao.android.feature.videoextract

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractFrameItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskDetailSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskListSnapshot
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.VideoExtractApiService
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class VideoExtractTaskCenterRepositoryTest {
    private val apiService = mockk<VideoExtractApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val repository = VideoExtractTaskCenterRepository(apiService, preferencesStore, baseUrlProvider)

    @Before
    fun setUp() {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test/api/".toHttpUrl()
    }

    @Test
    fun `load tasks should normalize api payload and cache first page`() = runTest {
        val savedSnapshot = slot<CachedVideoExtractTaskListSnapshot>()
        coEvery { apiService.getTaskList(page = 1, pageSize = 20) } returns ApiEnvelope(
            code = 0,
            data = buildJsonObject {
                put("items", buildJsonArray {
                    add(taskJson(taskId = "task-1", sourceRef = "clips/a.mp4"))
                    add(buildJsonObject { put("status", JsonPrimitive("RUNNING")) })
                })
                put("page", JsonPrimitive(1))
                put("pageSize", JsonPrimitive(20))
                put("total", JsonPrimitive(1))
            },
        )
        coEvery { preferencesStore.saveCachedVideoExtractTaskList(capture(savedSnapshot)) } just runs

        val result = repository.loadTasks(page = 1)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertFalse(payload.fromCache)
        assertEquals(1, payload.items.size)
        assertEquals("https://demo.test/upload/clips/a.mp4", payload.items.single().sourcePreviewUrl)
        assertEquals(listOf("task-1"), savedSnapshot.captured.items.map { it.taskId })
    }

    @Test
    fun `load tasks page over one should merge existing cache and keep cached duplicate`() = runTest {
        val savedSnapshot = slot<CachedVideoExtractTaskListSnapshot>()
        coEvery { preferencesStore.readCachedVideoExtractTaskList() } returns CachedVideoExtractTaskListSnapshot(
            items = listOf(snapshotTask(taskId = "task-1", sourceRef = "cached.mp4")),
            page = 1,
            pageSize = 20,
            total = 1,
        )
        coEvery { apiService.getTaskList(page = 2, pageSize = 20) } returns ApiEnvelope(
            code = 0,
            data = buildJsonObject {
                put("items", buildJsonArray {
                    add(taskJson(taskId = "task-1", sourceRef = "remote.mp4"))
                    add(taskJson(taskId = "task-2", sourceRef = "remote-2.mp4"))
                })
                put("page", JsonPrimitive(2))
                put("pageSize", JsonPrimitive(20))
                put("total", JsonPrimitive(2))
            },
        )
        coEvery { preferencesStore.saveCachedVideoExtractTaskList(capture(savedSnapshot)) } just runs

        val result = repository.loadTasks(page = 2)

        assertTrue(result is AppResult.Success)
        assertEquals(listOf("task-1", "task-2"), savedSnapshot.captured.items.map { it.taskId })
        assertEquals("cached.mp4", savedSnapshot.captured.items.first().sourceRef)
    }

    @Test
    fun `load tasks should fallback to cached snapshot when api fails`() = runTest {
        coEvery { apiService.getTaskList(page = 1, pageSize = 20) } throws IllegalStateException("network down")
        coEvery { preferencesStore.readCachedVideoExtractTaskList() } returns CachedVideoExtractTaskListSnapshot(
            items = listOf(snapshotTask(taskId = "task-cache")),
            page = 3,
            pageSize = 20,
            total = 7,
        )

        val result = repository.loadTasks(page = 1)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertTrue(payload.fromCache)
        assertEquals(3, payload.page)
        assertEquals("task-cache", payload.items.single().taskId)
    }

    @Test
    fun `load tasks should return error when api and cache both fail`() = runTest {
        coEvery { apiService.getTaskList(page = 1, pageSize = 20) } throws IllegalStateException("network down")
        coEvery { preferencesStore.readCachedVideoExtractTaskList() } returns null

        val result = repository.loadTasks(page = 1)

        assertTrue(result is AppResult.Error)
        assertEquals("network down", (result as AppResult.Error).message)
    }

    @Test
    fun `load task detail should cache first page and merge task into cached list`() = runTest {
        val savedDetail = slot<CachedVideoExtractTaskDetailSnapshot>()
        val savedList = slot<CachedVideoExtractTaskListSnapshot>()
        coEvery { apiService.getTaskDetail(taskId = "task-1", cursor = 0, pageSize = 80) } returns ApiEnvelope(
            code = 0,
            data = buildJsonObject {
                put("task", taskJson(taskId = "task-1", framesExtracted = 5, sourceRef = "clips/task-1.mp4"))
                put(
                    "frames",
                    buildJsonObject {
                        put("items", buildJsonArray { add(frameJson(seq = 1, url = "https://demo.test/frame-1.jpg")) })
                        put("nextCursor", JsonPrimitive(1))
                        put("hasMore", JsonPrimitive(true))
                    },
                )
            },
        )
        coEvery { preferencesStore.readCachedVideoExtractTaskList() } returns CachedVideoExtractTaskListSnapshot(
            items = listOf(snapshotTask(taskId = "task-old")),
            page = 1,
            pageSize = 20,
            total = 0,
        )
        coEvery { preferencesStore.saveCachedVideoExtractTaskDetail(capture(savedDetail)) } just runs
        coEvery { preferencesStore.saveCachedVideoExtractTaskList(capture(savedList)) } just runs

        val result = repository.loadTaskDetail(taskId = "task-1")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertFalse(payload.fromCache)
        assertEquals(listOf(1), payload.frames.items.map { it.seq })
        assertEquals(listOf(1), savedDetail.captured.frames.map { it.seq })
        assertEquals(listOf("task-1", "task-old"), savedList.captured.items.map { it.taskId })
        assertEquals(2, savedList.captured.total)
    }

    @Test
    fun `load task detail with cursor should merge cached frames and preserve first duplicate`() = runTest {
        val savedDetail = slot<CachedVideoExtractTaskDetailSnapshot>()
        coEvery { apiService.getTaskDetail(taskId = "task-1", cursor = 2, pageSize = 80) } returns ApiEnvelope(
            code = 0,
            data = buildJsonObject {
                put("task", taskJson(taskId = "task-1"))
                put(
                    "frames",
                    buildJsonObject {
                        put(
                            "items",
                            buildJsonArray {
                                add(frameJson(seq = 2, url = "https://demo.test/frame-2-new.jpg"))
                                add(frameJson(seq = 3, url = "https://demo.test/frame-3.jpg"))
                            },
                        )
                        put("nextCursor", JsonPrimitive(3))
                        put("hasMore", JsonPrimitive(false))
                    },
                )
            },
        )
        coEvery { preferencesStore.readCachedVideoExtractTaskDetail("task-1") } returns CachedVideoExtractTaskDetailSnapshot(
            task = snapshotTask(taskId = "task-1"),
            frames = listOf(
                CachedVideoExtractFrameItemSnapshot(seq = 1, url = "https://demo.test/frame-1.jpg"),
                CachedVideoExtractFrameItemSnapshot(seq = 2, url = "https://demo.test/frame-2-old.jpg"),
            ),
            nextCursor = 2,
            hasMore = true,
        )
        coEvery { preferencesStore.readCachedVideoExtractTaskList() } returns null
        coEvery { preferencesStore.saveCachedVideoExtractTaskDetail(capture(savedDetail)) } just runs

        val result = repository.loadTaskDetail(taskId = "task-1", cursor = 2)

        assertTrue(result is AppResult.Success)
        assertEquals(listOf(1, 2, 3), savedDetail.captured.frames.map { it.seq })
        assertEquals("https://demo.test/frame-2-old.jpg", savedDetail.captured.frames[1].url)
    }

    @Test
    fun `load task detail should fallback to cached detail when api fails`() = runTest {
        coEvery { apiService.getTaskDetail(taskId = "task-1", cursor = 0, pageSize = 80) } throws IllegalStateException("detail down")
        coEvery { preferencesStore.readCachedVideoExtractTaskDetail("task-1") } returns CachedVideoExtractTaskDetailSnapshot(
            task = snapshotTask(taskId = "task-1"),
            frames = listOf(CachedVideoExtractFrameItemSnapshot(seq = 9, url = "https://demo.test/frame-9.jpg")),
            nextCursor = 9,
            hasMore = false,
        )

        val result = repository.loadTaskDetail(taskId = " task-1 ")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertTrue(payload.fromCache)
        assertEquals(listOf(9), payload.frames.items.map { it.seq })
    }

    @Test
    fun `cancel and continue should trim payloads and surface failures`() = runTest {
        val cancelPayload = slot<JsonElement>()
        val continuePayload = slot<JsonElement>()
        coEvery { apiService.cancelTask(capture(cancelPayload)) } returns ApiEnvelope(code = 0)
        coEvery { apiService.continueTask(capture(continuePayload)) } returns ApiEnvelope(code = 0, message = "继续中")

        val cancelResult = repository.cancelTask(" task-1 ")
        val continueResult = repository.continueTask(" task-2 ", endSec = 12.5, maxFrames = 80)

        assertTrue(cancelResult is AppResult.Success)
        assertEquals("已终止任务", (cancelResult as AppResult.Success).data)
        assertEquals("task-1", cancelPayload.captured.jsonObject["taskId"]?.jsonPrimitive?.content)

        assertTrue(continueResult is AppResult.Success)
        assertEquals("继续中", (continueResult as AppResult.Success).data)
        val continueJson = continuePayload.captured.jsonObject
        assertEquals("task-2", continueJson["taskId"]?.jsonPrimitive?.content)
        assertEquals("12.5", continueJson["endSec"]?.jsonPrimitive?.content)
        assertEquals("80", continueJson["maxFrames"]?.jsonPrimitive?.content)

        coEvery { apiService.cancelTask(any()) } returns ApiEnvelope(code = 1, msg = "不能终止")
        coEvery { apiService.continueTask(any()) } returns ApiEnvelope(code = 1, message = "不能继续")

        val cancelFailed = repository.cancelTask("task-3")
        val continueFailed = repository.continueTask("task-4", endSec = null, maxFrames = null)

        assertTrue(cancelFailed is AppResult.Error)
        assertEquals("不能终止", (cancelFailed as AppResult.Error).message)
        assertTrue(continueFailed is AppResult.Error)
        assertEquals("不能继续", (continueFailed as AppResult.Error).message)
    }

    @Test
    fun `delete task should update cache on success and keep cache untouched on failure`() = runTest {
        val deletePayload = slot<JsonElement>()
        val savedSnapshot = slot<CachedVideoExtractTaskListSnapshot>()
        coEvery { apiService.deleteTask(capture(deletePayload)) } returns ApiEnvelope(code = 0)
        coEvery { preferencesStore.removeCachedVideoExtractTaskDetail("task-1") } just runs
        coEvery { preferencesStore.readCachedVideoExtractTaskList() } returns CachedVideoExtractTaskListSnapshot(
            items = listOf(snapshotTask(taskId = "task-1")),
            page = 3,
            pageSize = 20,
            total = 1,
        )
        coEvery { preferencesStore.saveCachedVideoExtractTaskList(capture(savedSnapshot)) } just runs

        val success = repository.deleteTask(" task-1 ", deleteFiles = false)

        assertTrue(success is AppResult.Success)
        assertEquals("已删除任务记录", (success as AppResult.Success).data)
        assertEquals("task-1", deletePayload.captured.jsonObject["taskId"]?.jsonPrimitive?.content)
        assertEquals("false", deletePayload.captured.jsonObject["deleteFiles"]?.jsonPrimitive?.content)
        assertTrue(savedSnapshot.captured.items.isEmpty())
        assertEquals(0, savedSnapshot.captured.total)
        assertEquals(1, savedSnapshot.captured.page)

        coEvery { apiService.deleteTask(any()) } returns ApiEnvelope(code = 1, msg = "删除失败")

        val failed = repository.deleteTask("task-9", deleteFiles = true)

        assertTrue(failed is AppResult.Error)
        assertEquals("删除失败", (failed as AppResult.Error).message)
        coVerify(exactly = 1) { preferencesStore.removeCachedVideoExtractTaskDetail(any()) }
    }

    private fun snapshotTask(
        taskId: String,
        sourceRef: String = "$taskId.mp4",
        framesExtracted: Int = 3,
    ) = CachedVideoExtractTaskItemSnapshot(
        taskId = taskId,
        sourceType = "upload",
        sourceRef = sourceRef,
        sourcePreviewUrl = "https://demo.test/upload/$sourceRef",
        outputDirLocalPath = "/tmp/$taskId",
        outputDirUrl = "https://demo.test/out/$taskId",
        outputFormat = "jpg",
        jpgQuality = 6,
        mode = "fps",
        keyframeMode = null,
        fps = 2.0,
        sceneThreshold = null,
        startSec = 0.0,
        endSec = 8.0,
        maxFrames = 8,
        framesExtracted = framesExtracted,
        videoWidth = 1920,
        videoHeight = 1080,
        durationSec = 8.0,
        cursorOutTimeSec = 2.0,
        status = "RUNNING",
        stopReason = "",
        lastError = "",
        createdAt = "2026-03-15T10:00:00",
        updatedAt = "2026-03-15T10:01:00",
        runtimeLogs = emptyList(),
    )

    private fun taskJson(
        taskId: String,
        sourceRef: String = "$taskId.mp4",
        framesExtracted: Int = 3,
    ) = buildJsonObject {
        put("taskId", JsonPrimitive(taskId))
        put("sourceType", JsonPrimitive("upload"))
        put("sourceRef", JsonPrimitive(sourceRef))
        put("outputDirLocalPath", JsonPrimitive("/tmp/$taskId"))
        put("outputDirUrl", JsonPrimitive("https://demo.test/out/$taskId"))
        put("outputFormat", JsonPrimitive("jpg"))
        put("jpgQuality", JsonPrimitive(6))
        put("mode", JsonPrimitive("fps"))
        put("fps", JsonPrimitive(2.0))
        put("maxFrames", JsonPrimitive(8))
        put("framesExtracted", JsonPrimitive(framesExtracted))
        put("videoWidth", JsonPrimitive(1920))
        put("videoHeight", JsonPrimitive(1080))
        put("durationSec", JsonPrimitive(8.0))
        put("cursorOutTimeSec", JsonPrimitive(2.0))
        put("status", JsonPrimitive("RUNNING"))
        put("stopReason", JsonPrimitive(""))
        put("lastError", JsonPrimitive(""))
        put("createdAt", JsonPrimitive("2026-03-15T10:00:00"))
        put("updatedAt", JsonPrimitive("2026-03-15T10:01:00"))
        put(
            "runtime",
            buildJsonObject {
                put("logs", buildJsonArray { add(JsonPrimitive("start")) })
            },
        )
    }

    private fun frameJson(seq: Int, url: String) = buildJsonObject {
        put("seq", JsonPrimitive(seq))
        put("url", JsonPrimitive(url))
    }
}
