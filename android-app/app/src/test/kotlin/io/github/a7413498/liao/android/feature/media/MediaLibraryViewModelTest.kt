package io.github.a7413498.liao.android.feature.media

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
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
class MediaLibraryViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<MediaLibraryRepository>()
    private val mtPhotoSameMediaRepository = mockk<MtPhotoSameMediaRepository>()

    @Test
    fun `refresh and load more should merge pages keep cache banner and guard duplicate loads`() = runTest(mainDispatcherRule.dispatcher) {
        val page1 = samplePage(
            page = 1,
            totalPages = 2,
            fromCache = true,
            items = listOf(sampleItem(localPath = "/a.jpg", title = "A")),
        )
        val page2 = samplePage(
            page = 2,
            totalPages = 2,
            fromCache = false,
            items = listOf(
                sampleItem(localPath = "/a.jpg", title = "A-dup"),
                sampleItem(localPath = "/b.jpg", title = "B"),
            ),
        )
        val loadMoreDeferred = CompletableDeferred<AppResult<MediaLibraryPage>>()
        coEvery { repository.loadMedia(page = 1) } returns AppResult.Success(page1)
        coEvery { repository.loadMedia(page = 2) } coAnswers { loadMoreDeferred.await() }

        val viewModel = MediaLibraryViewModel(repository, mtPhotoSameMediaRepository)
        advanceUntilIdle()

        assertEquals(page1.items, viewModel.uiState.items)
        assertEquals("网络不可用，已展示最近缓存的媒体库", viewModel.uiState.message)

        viewModel.loadMore()
        runCurrent()
        assertTrue(viewModel.uiState.loadingMore)

        viewModel.loadMore()
        runCurrent()
        coVerify(exactly = 1) { repository.loadMedia(page = 2) }

        loadMoreDeferred.complete(AppResult.Success(page2))
        advanceUntilIdle()
        assertFalse(viewModel.uiState.loadingMore)
        assertEquals(2, viewModel.uiState.items.size)
        assertEquals(listOf("/a.jpg", "/b.jpg"), viewModel.uiState.items.map { it.localPath })
        assertEquals("网络不可用，已展示最近缓存的媒体库", viewModel.uiState.message)

        viewModel.loadMore()
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.loadMedia(page = 2) }
    }

    @Test
    fun `selection and delete should handle blank toggle success refresh and error`() = runTest(mainDispatcherRule.dispatcher) {
        val page1 = samplePage(
            page = 1,
            totalPages = 1,
            items = listOf(
                sampleItem(localPath = "/a.jpg", title = "A"),
                sampleItem(localPath = "/b.mp4", title = "B", type = ChatMessageType.VIDEO),
            ),
        )
        val emptyPage = samplePage(page = 1, totalPages = 0, items = emptyList())
        coEvery { repository.loadMedia(page = 1) } returnsMany listOf(
            AppResult.Success(page1),
            AppResult.Success(emptyPage),
        )
        coEvery { repository.deleteMedia(listOf("/a.jpg")) } returns AppResult.Success("已删除 1 项")
        coEvery { repository.deleteMedia(listOf("/b.mp4")) } returns AppResult.Error("删除失败")

        val viewModel = MediaLibraryViewModel(repository, mtPhotoSameMediaRepository)
        advanceUntilIdle()

        viewModel.toggleSelectionMode()
        assertTrue(viewModel.uiState.selectionMode)

        viewModel.toggleSelection("   ")
        assertTrue(viewModel.uiState.selectedLocalPaths.isEmpty())

        viewModel.toggleSelection("/a.jpg")
        assertEquals(setOf("/a.jpg"), viewModel.uiState.selectedLocalPaths)
        viewModel.toggleSelection("/a.jpg")
        assertTrue(viewModel.uiState.selectedLocalPaths.isEmpty())

        viewModel.toggleSelectionMode()
        assertFalse(viewModel.uiState.selectionMode)
        assertTrue(viewModel.uiState.selectedLocalPaths.isEmpty())

        viewModel.deleteSingle("/a.jpg")
        advanceUntilIdle()
        assertFalse(viewModel.uiState.deleting)
        assertFalse(viewModel.uiState.selectionMode)
        assertNull(viewModel.uiState.message)
        assertTrue(viewModel.uiState.items.isEmpty())
        coVerify(exactly = 2) { repository.loadMedia(page = 1) }

        viewModel.toggleSelectionMode()
        viewModel.toggleSelection("/b.mp4")
        viewModel.deleteSelected()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.deleting)
        assertEquals("删除失败", viewModel.uiState.message)
        assertTrue(viewModel.uiState.selectionMode)
        assertEquals(setOf("/b.mp4"), viewModel.uiState.selectedLocalPaths)
    }

    @Test
    fun `same media lookup should handle blank path success error dismiss and consume message`() = runTest(mainDispatcherRule.dispatcher) {
        val page1 = samplePage(
            page = 1,
            totalPages = 1,
            items = listOf(sampleItem(localPath = "/same/a.jpg", title = "A")),
        )
        val sameMediaDeferred = CompletableDeferred<AppResult<List<MtPhotoSameMediaItem>>>()
        val sameItem = MtPhotoSameMediaItem(
            id = 1L,
            md5 = "md5-a",
            filePath = "/upload/a.jpg",
            fileName = "a.jpg",
            folderId = 9L,
            folderPath = "/folder/a",
            folderName = "目录A",
            day = "2026-03-17",
            canOpenFolder = true,
        )
        coEvery { repository.loadMedia(page = 1) } returns AppResult.Success(page1)
        coEvery { mtPhotoSameMediaRepository.queryByLocalPath("/same/a.jpg") } coAnswers { sameMediaDeferred.await() } andThen AppResult.Error("查询失败")

        val viewModel = MediaLibraryViewModel(repository, mtPhotoSameMediaRepository)
        advanceUntilIdle()

        viewModel.lookupMtPhotoSameMedia(sampleItem(localPath = "   ", title = "Blank"))
        assertEquals("当前媒体缺少本地路径，无法查询同媒体", viewModel.uiState.message)
        viewModel.consumeMessage()
        assertNull(viewModel.uiState.message)

        val sourceItem = page1.items.first()
        viewModel.lookupMtPhotoSameMedia(sourceItem)
        runCurrent()
        assertTrue(viewModel.uiState.sameMediaVisible)
        assertTrue(viewModel.uiState.sameMediaLoading)

        viewModel.lookupMtPhotoSameMedia(sourceItem)
        runCurrent()
        coVerify(exactly = 1) { mtPhotoSameMediaRepository.queryByLocalPath("/same/a.jpg") }

        sameMediaDeferred.complete(AppResult.Success(listOf(sameItem)))
        advanceUntilIdle()
        assertFalse(viewModel.uiState.sameMediaLoading)
        assertEquals(listOf(sameItem), viewModel.uiState.sameMediaItems)
        assertNull(viewModel.uiState.sameMediaError)

        viewModel.dismissMtPhotoSameMedia()
        assertFalse(viewModel.uiState.sameMediaVisible)
        assertTrue(viewModel.uiState.sameMediaItems.isEmpty())
        assertEquals("", viewModel.uiState.sameMediaSourceLocalPath)

        viewModel.dismissMtPhotoSameMedia()
        assertFalse(viewModel.uiState.sameMediaVisible)

        viewModel.lookupMtPhotoSameMedia(sourceItem)
        advanceUntilIdle()
        assertFalse(viewModel.uiState.sameMediaLoading)
        assertEquals("查询失败", viewModel.uiState.sameMediaError)
        assertTrue(viewModel.uiState.sameMediaItems.isEmpty())
    }

    private fun samplePage(
        page: Int,
        totalPages: Int,
        items: List<MediaLibraryItem>,
        fromCache: Boolean = false,
    ): MediaLibraryPage = MediaLibraryPage(
        items = items,
        page = page,
        total = items.size * (if (totalPages <= 0) 1 else totalPages),
        totalPages = totalPages,
        fromCache = fromCache,
    )

    private fun sampleItem(
        localPath: String,
        title: String,
        type: ChatMessageType = ChatMessageType.IMAGE,
    ): MediaLibraryItem = MediaLibraryItem(
        url = "https://demo.test/upload${localPath.ifBlank { "/blank" }}",
        localPath = localPath,
        type = type,
        title = title,
        subtitle = "2026-03-17",
        posterUrl = "",
        updateTime = "2026-03-17 12:00:00",
        source = "mtphoto",
    )
}
