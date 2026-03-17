package io.github.a7413498.liao.android.feature.videoextract

import android.net.Uri
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class VideoExtractCreateViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<VideoExtractCreateRepository>()
    private val sourceUri = mockk<Uri>(relaxed = true)

    @Test
    fun `update methods should rewrite ui state`() = runTest(mainDispatcherRule.dispatcher) {
        val viewModel = VideoExtractCreateViewModel(repository)

        viewModel.updateMode(VideoExtractModeOption.FPS)
        viewModel.updateKeyframeMode(VideoExtractKeyframeModeOption.SCENE)
        viewModel.updateSceneThreshold("0.45")
        viewModel.updateFps("2.5")
        viewModel.updateStartSec("1.5")
        viewModel.updateEndSec("5.5")
        viewModel.updateMaxFrames("88")
        viewModel.updateOutputFormat(VideoExtractOutputFormatOption.PNG)
        viewModel.updateJpgQuality("12")

        assertEquals(VideoExtractModeOption.FPS, viewModel.uiState.value.mode)
        assertEquals(VideoExtractKeyframeModeOption.SCENE, viewModel.uiState.value.keyframeMode)
        assertEquals("0.45", viewModel.uiState.value.sceneThreshold)
        assertEquals("2.5", viewModel.uiState.value.fps)
        assertEquals("1.5", viewModel.uiState.value.startSec)
        assertEquals("5.5", viewModel.uiState.value.endSec)
        assertEquals("88", viewModel.uiState.value.maxFrames)
        assertEquals(VideoExtractOutputFormatOption.PNG, viewModel.uiState.value.outputFormat)
        assertEquals("12", viewModel.uiState.value.jpgQuality)
    }

    @Test
    fun `refresh probe should guard missing source and support silent consume flow`() = runTest(mainDispatcherRule.dispatcher) {
        val viewModel = VideoExtractCreateViewModel(repository)

        viewModel.refreshProbe()
        assertEquals("请先选择本地视频", viewModel.uiState.value.message)

        viewModel.consumeMessage()
        assertNull(viewModel.uiState.value.message)

        viewModel.refreshProbe(autoMessage = false)
        assertNull(viewModel.uiState.value.message)

        viewModel.consumeMessage()
        assertNull(viewModel.uiState.value.message)
        coVerify(exactly = 0) { repository.probeUpload(any()) }
    }

    @Test
    fun `upload source should handle success and auto probe success`() = runTest(mainDispatcherRule.dispatcher) {
        val source = sampleSource()
        val probe = sampleProbe()
        coEvery { repository.uploadVideo(sourceUri) } returns AppResult.Success(source)
        coEvery { repository.probeUpload(source.localPath) } returns AppResult.Success(probe)

        val viewModel = VideoExtractCreateViewModel(repository)
        viewModel.uploadSource(sourceUri)
        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.uploading)
        assertFalse(viewModel.uiState.value.probing)
        assertEquals(source, viewModel.uiState.value.source)
        assertEquals(probe, viewModel.uiState.value.probe)
        assertNull(viewModel.uiState.value.probeError)
        assertNull(viewModel.uiState.value.message)
    }

    @Test
    fun `upload source should surface upload failure and skip probe`() = runTest(mainDispatcherRule.dispatcher) {
        coEvery { repository.uploadVideo(sourceUri) } returns AppResult.Error("上传失败")

        val viewModel = VideoExtractCreateViewModel(repository)
        viewModel.uploadSource(sourceUri)
        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.uploading)
        assertNull(viewModel.uiState.value.source)
        assertEquals("上传失败", viewModel.uiState.value.message)
        coVerify(exactly = 0) { repository.probeUpload(any()) }
    }

    @Test
    fun `refresh probe should surface repository error after source uploaded`() = runTest(mainDispatcherRule.dispatcher) {
        val source = sampleSource()
        coEvery { repository.uploadVideo(sourceUri) } returns AppResult.Success(source)
        coEvery { repository.probeUpload(source.localPath) } returnsMany listOf(
            AppResult.Success(sampleProbe()),
            AppResult.Error("探测失败"),
        )

        val viewModel = VideoExtractCreateViewModel(repository)
        viewModel.uploadSource(sourceUri)
        advanceUntilIdle()
        viewModel.refreshProbe()
        advanceUntilIdle()

        assertFalse(viewModel.uiState.value.probing)
        assertNull(viewModel.uiState.value.probe)
        assertEquals("探测失败", viewModel.uiState.value.probeError)
        assertEquals("探测失败", viewModel.uiState.value.message)
    }

    @Test
    fun `create task should guard missing source and invalid payload`() = runTest(mainDispatcherRule.dispatcher) {
        val source = sampleSource()
        val probe = sampleProbe()
        coEvery { repository.uploadVideo(sourceUri) } returns AppResult.Success(source)
        coEvery { repository.probeUpload(source.localPath) } returns AppResult.Success(probe)

        val viewModel = VideoExtractCreateViewModel(repository)
        viewModel.createTask()
        assertEquals("请先选择本地视频", viewModel.uiState.value.message)

        viewModel.uploadSource(sourceUri)
        advanceUntilIdle()
        viewModel.updateMode(VideoExtractModeOption.FPS)
        viewModel.updateFps("0")
        viewModel.createTask()

        assertEquals("fps 必须大于 0", viewModel.uiState.value.message)
        coVerify(exactly = 0) { repository.createTask(any(), any()) }
    }

    @Test
    fun `create task should handle success failure and probe replacement branches`() = runTest(mainDispatcherRule.dispatcher) {
        val source = sampleSource()
        val probe = sampleProbe()
        val replacementProbe = probe.copy(width = 1280, height = 720)
        coEvery { repository.uploadVideo(sourceUri) } returns AppResult.Success(source)
        coEvery { repository.probeUpload(source.localPath) } returns AppResult.Success(probe)
        coEvery { repository.createTask(source.localPath, any()) } returnsMany listOf(
            AppResult.Success(VideoExtractCreatedTask(taskId = "task-1", probe = null)),
            AppResult.Success(VideoExtractCreatedTask(taskId = "task-2", probe = replacementProbe)),
            AppResult.Error("创建失败"),
        )

        val viewModel = VideoExtractCreateViewModel(repository)
        viewModel.uploadSource(sourceUri)
        advanceUntilIdle()

        viewModel.createTask()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.value.creating)
        assertEquals("task-1", viewModel.uiState.value.createdTask?.taskId)
        assertEquals(probe, viewModel.uiState.value.probe)
        assertEquals("抽帧任务已创建：task-1", viewModel.uiState.value.message)

        viewModel.consumeMessage()
        viewModel.createTask()
        advanceUntilIdle()
        assertEquals("task-2", viewModel.uiState.value.createdTask?.taskId)
        assertEquals(replacementProbe, viewModel.uiState.value.probe)
        assertEquals("抽帧任务已创建：task-2", viewModel.uiState.value.message)

        viewModel.consumeMessage()
        viewModel.createTask()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.value.creating)
        assertEquals("创建失败", viewModel.uiState.value.message)
    }

    private fun sampleSource() = VideoExtractUploadedSource(
        displayName = "demo.mp4",
        mimeType = "video/mp4",
        size = 1024,
        localPath = "/upload/demo.mp4",
        localFilename = "demo.mp4",
        originalFilename = "demo.mp4",
    )

    private fun sampleProbe() = VideoExtractProbeSummary(
        durationSec = 30.0,
        width = 1920,
        height = 1080,
        avgFps = 24.0,
    )
}
