package io.github.a7413498.liao.android.feature.videoextract

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class VideoExtractTaskCenterViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<VideoExtractTaskCenterRepository>()

    @Test
    fun `init refresh and load more should guard while list loading merge pages and keep cache banner`() = runTest(mainDispatcherRule.dispatcher) {
        val task1 = sampleTask(taskId = "task-1")
        val task1Updated = sampleTask(taskId = "task-1", status = "PAUSED_USER")
        val task2 = sampleTask(taskId = "task-2", status = "FINISHED")
        val initialDeferred = CompletableDeferred<AppResult<VideoExtractTaskListPage>>()
        coEvery { repository.loadTasks(page = 1, pageSize = 20) } coAnswers { initialDeferred.await() }
        coEvery { repository.loadTasks(page = 2, pageSize = 20) } returns AppResult.Success(
            samplePage(page = 2, total = 2, items = listOf(task1Updated, task2)),
        )

        val viewModel = VideoExtractTaskCenterViewModel(repository)
        runCurrent()

        viewModel.loadMoreTasks()
        runCurrent()
        coVerify(exactly = 0) { repository.loadTasks(page = 2, pageSize = 20) }

        initialDeferred.complete(AppResult.Success(samplePage(page = 1, total = 2, items = listOf(task1), fromCache = true)))
        advanceUntilIdle()
        assertFalse(viewModel.uiState.listLoading)
        assertEquals(listOf("task-1"), viewModel.uiState.tasks.map { it.taskId })
        assertEquals("网络不可用，已展示最近缓存的抽帧任务", viewModel.uiState.message)

        viewModel.loadMoreTasks()
        advanceUntilIdle()
        assertEquals(2, viewModel.uiState.page)
        assertEquals(listOf("task-1", "task-2"), viewModel.uiState.tasks.map { it.taskId })
        assertEquals("RUNNING", viewModel.uiState.tasks.first { it.taskId == "task-1" }.status)
        assertEquals("网络不可用，已展示最近缓存的抽帧任务", viewModel.uiState.message)

        viewModel.loadMoreTasks()
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.loadTasks(page = 2, pageSize = 20) }
    }

    @Test
    fun `toggle task should select merge detail clear selection and surface detail error`() = runTest(mainDispatcherRule.dispatcher) {
        val task1 = sampleTask(taskId = "task-1", status = "RUNNING")
        val mergedTask1 = sampleTask(taskId = "task-1", status = "PAUSED_USER")
        val task2 = sampleTask(taskId = "task-2", status = "PAUSED_USER")
        coEvery { repository.loadTasks(page = 1, pageSize = 20) } returns AppResult.Success(
            samplePage(items = listOf(task1, task2)),
        )
        coEvery { repository.loadTaskDetail(taskId = "task-1", cursor = 0) } returns AppResult.Success(
            sampleDetail(task = mergedTask1, frames = listOf(sampleFrame(1)), nextCursor = 1, hasMore = true, fromCache = true),
        )
        coEvery { repository.loadTaskDetail(taskId = "task-2", cursor = 0) } returns AppResult.Error("详情失败")

        val viewModel = VideoExtractTaskCenterViewModel(repository)
        advanceUntilIdle()

        viewModel.toggleTask("task-1")
        advanceUntilIdle()
        assertFalse(viewModel.uiState.detailLoading)
        assertEquals("task-1", viewModel.uiState.selectedTaskId)
        assertEquals("PAUSED_USER", viewModel.uiState.selectedTask?.status)
        assertEquals(listOf(1), viewModel.uiState.frames.items.map { it.seq })
        assertEquals("网络不可用，已展示最近缓存的任务详情", viewModel.uiState.message)
        assertEquals("PAUSED_USER", viewModel.uiState.tasks.first { it.taskId == "task-1" }.status)

        viewModel.updateContinueEndSec("2.0")
        viewModel.updateContinueMaxFrames("10")
        viewModel.toggleTask("task-1")
        assertNull(viewModel.uiState.selectedTaskId)
        assertNull(viewModel.uiState.selectedTask)
        assertTrue(viewModel.uiState.frames.items.isEmpty())
        assertEquals("", viewModel.uiState.continueEndSec)
        assertEquals("", viewModel.uiState.continueMaxFrames)

        viewModel.toggleTask("task-2")
        advanceUntilIdle()
        assertFalse(viewModel.uiState.detailLoading)
        assertEquals("task-2", viewModel.uiState.selectedTaskId)
        assertNull(viewModel.uiState.selectedTask)
        assertEquals("详情失败", viewModel.uiState.message)
    }

    @Test
    fun `load more frames and refresh selected task should guard merge frames and surface error`() = runTest(mainDispatcherRule.dispatcher) {
        val task1 = sampleTask(taskId = "task-1", status = "RUNNING")
        coEvery { repository.loadTasks(page = 1, pageSize = 20) } returns AppResult.Success(samplePage(items = listOf(task1)))
        coEvery { repository.loadTaskDetail(taskId = "task-1", cursor = 0) } returnsMany listOf(
            AppResult.Success(sampleDetail(task = task1, frames = listOf(sampleFrame(1), sampleFrame(2)), nextCursor = 2, hasMore = true)),
            AppResult.Error("刷新失败"),
        )
        coEvery { repository.loadTaskDetail(taskId = "task-1", cursor = 2) } returns AppResult.Success(
            sampleDetail(
                task = sampleTask(taskId = "task-1", status = "FINISHED"),
                frames = listOf(sampleFrame(2, "https://demo.test/frame-2-new.jpg"), sampleFrame(3)),
                nextCursor = 3,
                hasMore = false,
                fromCache = true,
            ),
        )

        val viewModel = VideoExtractTaskCenterViewModel(repository)
        advanceUntilIdle()

        viewModel.loadMoreFrames()
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.loadTaskDetail(taskId = "task-1", cursor = 2) }

        viewModel.toggleTask("task-1")
        advanceUntilIdle()

        viewModel.loadMoreFrames()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.framesLoadingMore)
        assertEquals(listOf(1, 2, 3), viewModel.uiState.frames.items.map { it.seq })
        assertEquals(false, viewModel.uiState.frames.hasMore)
        assertEquals("网络不可用，已展示最近缓存的任务详情", viewModel.uiState.message)

        viewModel.loadMoreFrames()
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.loadTaskDetail(taskId = "task-1", cursor = 2) }

        viewModel.refreshSelectedTask()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.detailLoading)
        assertEquals("刷新失败", viewModel.uiState.message)
    }

    @Test
    fun `cancel selected task should guard unsupported status and handle success with follow-up detail failure`() = runTest(mainDispatcherRule.dispatcher) {
        val finishedTask = sampleTask(taskId = "task-finished", status = "FINISHED")
        val runningTask = sampleTask(taskId = "task-running", status = "RUNNING")
        coEvery { repository.loadTasks(page = 1, pageSize = 20) } returnsMany listOf(
            AppResult.Success(samplePage(items = listOf(finishedTask, runningTask))),
            AppResult.Success(samplePage(items = listOf(finishedTask, sampleTask(taskId = "task-running", status = "PAUSED_USER")))),
        )
        coEvery { repository.loadTaskDetail(taskId = "task-finished", cursor = 0) } returns AppResult.Success(sampleDetail(task = finishedTask))
        coEvery { repository.loadTaskDetail(taskId = "task-running", cursor = 0) } returnsMany listOf(
            AppResult.Success(sampleDetail(task = runningTask)),
            AppResult.Error("详情刷新失败"),
        )
        coEvery { repository.cancelTask("task-running") } returns AppResult.Success("已终止任务")

        val viewModel = VideoExtractTaskCenterViewModel(repository)
        advanceUntilIdle()

        viewModel.cancelSelectedTask()
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.cancelTask(any()) }

        viewModel.toggleTask("task-finished")
        advanceUntilIdle()
        viewModel.cancelSelectedTask()
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.cancelTask("task-finished") }

        viewModel.toggleTask("task-running")
        advanceUntilIdle()
        viewModel.cancelSelectedTask()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.actionLoading)
        assertEquals("详情刷新失败", viewModel.uiState.message)
        coVerify(exactly = 1) { repository.cancelTask("task-running") }
    }

    @Test
    fun `continue selected task should validate guard and handle success`() = runTest(mainDispatcherRule.dispatcher) {
        val finishedTask = sampleTask(taskId = "task-finished", status = "FINISHED")
        val pausedTask = sampleTask(taskId = "task-paused", status = "PAUSED_USER")
        coEvery { repository.loadTasks(page = 1, pageSize = 20) } returnsMany listOf(
            AppResult.Success(samplePage(items = listOf(finishedTask, pausedTask))),
            AppResult.Success(samplePage(items = listOf(finishedTask, sampleTask(taskId = "task-paused", status = "RUNNING")))),
        )
        coEvery { repository.loadTaskDetail(taskId = "task-finished", cursor = 0) } returns AppResult.Success(sampleDetail(task = finishedTask))
        coEvery { repository.loadTaskDetail(taskId = "task-paused", cursor = 0) } returnsMany listOf(
            AppResult.Success(sampleDetail(task = pausedTask)),
            AppResult.Success(sampleDetail(task = sampleTask(taskId = "task-paused", status = "RUNNING"), fromCache = true)),
        )
        coEvery { repository.continueTask("task-paused", 1.5, 20) } returns AppResult.Success("继续中")

        val viewModel = VideoExtractTaskCenterViewModel(repository)
        advanceUntilIdle()

        viewModel.continueSelectedTask()
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.continueTask(any(), any(), any()) }

        viewModel.toggleTask("task-finished")
        advanceUntilIdle()
        viewModel.continueSelectedTask()
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.continueTask("task-finished", any(), any()) }

        viewModel.toggleTask("task-paused")
        advanceUntilIdle()
        viewModel.updateContinueEndSec("bad")
        viewModel.continueSelectedTask()
        assertEquals("继续 endSec 格式非法", viewModel.uiState.message)

        viewModel.updateContinueEndSec("1.5")
        viewModel.updateContinueMaxFrames("bad")
        viewModel.continueSelectedTask()
        assertEquals("继续 maxFrames 格式非法", viewModel.uiState.message)

        viewModel.updateContinueMaxFrames("20")
        viewModel.continueSelectedTask()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.actionLoading)
        assertEquals("", viewModel.uiState.continueEndSec)
        assertEquals("", viewModel.uiState.continueMaxFrames)
        assertEquals("继续中", viewModel.uiState.message)
        coVerify(exactly = 1) { repository.continueTask("task-paused", 1.5, 20) }
    }

    @Test
    fun `continue selected task should surface repository error`() = runTest(mainDispatcherRule.dispatcher) {
        val pausedTask = sampleTask(taskId = "task-1", status = "PAUSED_LIMIT")
        coEvery { repository.loadTasks(page = 1, pageSize = 20) } returns AppResult.Success(samplePage(items = listOf(pausedTask)))
        coEvery { repository.loadTaskDetail(taskId = "task-1", cursor = 0) } returns AppResult.Success(sampleDetail(task = pausedTask))
        coEvery { repository.continueTask("task-1", null, null) } returns AppResult.Error("不能继续")

        val viewModel = VideoExtractTaskCenterViewModel(repository)
        advanceUntilIdle()
        viewModel.toggleTask("task-1")
        advanceUntilIdle()

        viewModel.continueSelectedTask()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.actionLoading)
        assertEquals("不能继续", viewModel.uiState.message)
    }

    @Test
    fun `delete selected task should handle success error update flag and consume message`() = runTest(mainDispatcherRule.dispatcher) {
        val task1 = sampleTask(taskId = "task-1", status = "FINISHED")
        val task2 = sampleTask(taskId = "task-2", status = "FINISHED")
        coEvery { repository.loadTasks(page = 1, pageSize = 20) } returnsMany listOf(
            AppResult.Success(samplePage(items = listOf(task1, task2))),
            AppResult.Success(samplePage(items = listOf(task2))),
        )
        coEvery { repository.loadTaskDetail(taskId = "task-1", cursor = 0) } returns AppResult.Success(sampleDetail(task = task1))
        coEvery { repository.loadTaskDetail(taskId = "task-2", cursor = 0) } returns AppResult.Success(sampleDetail(task = task2))
        coEvery { repository.deleteTask("task-1", false) } returns AppResult.Success("已删除任务记录")
        coEvery { repository.deleteTask("task-2", false) } returns AppResult.Error("删除失败")

        val viewModel = VideoExtractTaskCenterViewModel(repository)
        advanceUntilIdle()

        viewModel.updateDeleteFiles(false)
        assertFalse(viewModel.uiState.deleteFiles)

        viewModel.toggleTask("task-1")
        advanceUntilIdle()
        viewModel.deleteSelectedTask()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.actionLoading)
        assertNull(viewModel.uiState.selectedTaskId)
        assertNull(viewModel.uiState.selectedTask)
        assertTrue(viewModel.uiState.frames.items.isEmpty())
        assertEquals("已删除任务记录", viewModel.uiState.message)

        viewModel.consumeMessage()
        assertNull(viewModel.uiState.message)

        viewModel.toggleTask("task-2")
        advanceUntilIdle()
        viewModel.deleteSelectedTask()
        advanceUntilIdle()
        assertEquals("task-2", viewModel.uiState.selectedTaskId)
        assertEquals("删除失败", viewModel.uiState.message)
    }

    private fun sampleTask(
        taskId: String = "task-1",
        status: String = "RUNNING",
        sourceRef: String = "/upload/source/demo.mp4",
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
        runtimeLogs = listOf("runtime"),
    )

    private fun samplePage(
        items: List<VideoExtractTaskItem>,
        page: Int = 1,
        pageSize: Int = 20,
        total: Int = items.size,
        fromCache: Boolean = false,
    ) = VideoExtractTaskListPage(
        items = items,
        page = page,
        pageSize = pageSize,
        total = total,
        fromCache = fromCache,
    )

    private fun sampleDetail(
        task: VideoExtractTaskItem = sampleTask(),
        frames: List<VideoExtractFrameItem> = emptyList(),
        nextCursor: Int = 0,
        hasMore: Boolean = false,
        fromCache: Boolean = false,
    ) = VideoExtractTaskDetailResult(
        task = task,
        frames = VideoExtractFramesUiPage(items = frames, nextCursor = nextCursor, hasMore = hasMore),
        fromCache = fromCache,
    )

    private fun sampleFrame(seq: Int, url: String = "https://demo.test/frame-$seq.jpg") = VideoExtractFrameItem(
        seq = seq,
        url = url,
    )
}
