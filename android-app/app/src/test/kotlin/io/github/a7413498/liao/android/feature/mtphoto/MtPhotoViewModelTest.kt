package io.github.a7413498.liao.android.feature.mtphoto

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
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
class MtPhotoViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<MtPhotoRepository>()
    private val folderFavoritesRepository = mockk<MtPhotoFolderFavoritesRepository>()

    @Test
    fun `init open album and load more should update album state and guard end page`() = runTest(mainDispatcherRule.dispatcher) {
        val album = sampleAlbum(id = 1L, name = "旅行")
        val page1 = sampleAlbumPage(page = 1, totalPages = 2, items = listOf(sampleMedia(md5 = "m1")))
        val page2 = sampleAlbumPage(page = 2, totalPages = 2, items = listOf(sampleMedia(md5 = "m2")))
        coEvery { repository.loadAlbums() } returns AppResult.Success(listOf(album))
        coEvery { repository.loadAlbumFiles(album.id, page = 1) } returns AppResult.Success(page1)
        coEvery { repository.loadAlbumFiles(album.id, page = 2) } returns AppResult.Success(page2)

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        assertTrue(viewModel.uiState.albumsLoaded)
        assertEquals(listOf(album), viewModel.uiState.albums)

        viewModel.loadMoreAlbum()
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.loadAlbumFiles(album.id, page = 2) }

        viewModel.openAlbum(album)
        advanceUntilIdle()
        assertEquals(album, viewModel.uiState.selectedAlbum)
        assertEquals(page1.items, viewModel.uiState.albumItems)

        viewModel.loadMoreAlbum()
        advanceUntilIdle()
        assertEquals(page1.items + page2.items, viewModel.uiState.albumItems)
        assertEquals(2, viewModel.uiState.albumPage)
        assertEquals(2, viewModel.uiState.albumTotalPages)

        viewModel.loadMoreAlbum()
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.loadAlbumFiles(album.id, page = 2) }

        viewModel.backToAlbums()
        assertNull(viewModel.uiState.selectedAlbum)
        assertEquals(emptyList<MtPhotoMediaSummary>(), viewModel.uiState.albumItems)
        assertEquals(0, viewModel.uiState.albumPage)
        assertNull(viewModel.uiState.previewItem)
    }

    @Test
    fun `switch folder mode should load root favorites and resolve deferred timeline on demand`() = runTest(mainDispatcherRule.dispatcher) {
        val rootFolder = sampleFolder(id = 10L, name = "根目录子目录", path = "/root/child")
        val favorite = sampleFavorite(folderId = 10L, folderName = "根目录子目录", folderPath = "/root/child")
        val deferredFolder = sampleFolder(id = 20L, name = "旅行-2026", path = "/trip")
        val rootPage = sampleFolderPage(
            folderName = "根目录",
            folderPath = "/root",
            folders = listOf(rootFolder),
            files = listOf(sampleMedia(md5 = "root-file")),
            page = 1,
            totalPages = 1,
        )
        val detailPage = sampleFolderPage(
            folderName = "旅行-2026",
            folderPath = "/trip",
            folders = listOf(sampleFolder(id = 21L, name = "一月", path = "/trip/01"), sampleFolder(id = 22L, name = "二月", path = "/trip/02")),
            files = listOf(sampleMedia(md5 = "detail-file")),
            page = 1,
            totalPages = 3,
        )
        val timelinePage = sampleFolderPage(
            folderName = "旅行-2026",
            folderPath = "/trip",
            folders = detailPage.folders,
            files = listOf(sampleMedia(md5 = "timeline-file")),
            page = 1,
            totalPages = 3,
        )
        coEvery { repository.loadAlbums() } returns AppResult.Success(emptyList())
        coEvery { repository.loadFolderRoot() } returns AppResult.Success(rootPage)
        coEvery { folderFavoritesRepository.loadFavorites() } returns AppResult.Success(listOf(favorite))
        coEvery { repository.loadTimelineThreshold() } returns 1
        coEvery { repository.loadFolderContent(folderId = deferredFolder.id, page = 1, includeTimeline = false) } returns AppResult.Success(detailPage)
        coEvery { repository.loadFolderContent(folderId = deferredFolder.id, page = 1, includeTimeline = true) } returns AppResult.Success(timelinePage)

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.switchMode(MtPhotoMode.FOLDERS)
        advanceUntilIdle()
        assertEquals(MtPhotoMode.FOLDERS, viewModel.uiState.mode)
        assertTrue(viewModel.uiState.foldersLoaded)
        assertEquals(listOf(favorite), viewModel.uiState.folderFavorites)
        assertEquals(listOf(rootFolder), viewModel.uiState.currentFolders)

        viewModel.openFolder(deferredFolder)
        advanceUntilIdle()
        assertTrue(viewModel.uiState.folderTimelineDeferred)
        assertEquals(emptyList<MtPhotoMediaSummary>(), viewModel.uiState.folderItems)
        assertEquals(1, viewModel.uiState.timelineThreshold)

        viewModel.loadFolderTimeline()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.folderTimelineDeferred)
        assertEquals(timelinePage.files, viewModel.uiState.folderItems)
        assertEquals("旅行-2026", viewModel.uiState.folderHistory.last().folderName)
    }

    @Test
    fun `open folder by id should support save and remove current favorite`() = runTest(mainDispatcherRule.dispatcher) {
        val targetFolderId = 30L
        val savedFavorite = sampleFavorite(folderId = targetFolderId, folderName = "猫猫", folderPath = "/cats", coverMd5 = "md5-cats")
        val detailPage = sampleFolderPage(
            folderName = "猫猫",
            folderPath = "/cats",
            folders = listOf(sampleFolder(id = 31L, name = "一号", path = "/cats/1")),
            files = listOf(sampleMedia(md5 = "cats-file")),
            page = 1,
            totalPages = 1,
        )
        coEvery { repository.loadAlbums() } returns AppResult.Success(emptyList())
        coEvery { folderFavoritesRepository.loadFavorites() } returns AppResult.Success(emptyList())
        coEvery { repository.loadTimelineThreshold() } returns 99
        coEvery { repository.loadFolderContent(folderId = targetFolderId, page = 1, includeTimeline = false) } returns AppResult.Success(detailPage)
        coEvery { repository.loadFolderContent(folderId = targetFolderId, page = 1, includeTimeline = true) } returns AppResult.Success(detailPage)
        coEvery {
            folderFavoritesRepository.upsertFavorite(
                folderId = targetFolderId,
                folderName = "猫猫",
                folderPath = "/cats",
                coverMd5 = "md5-cats",
            )
        } returns AppResult.Success(savedFavorite)
        coEvery { folderFavoritesRepository.removeFavorite(targetFolderId) } returns AppResult.Success(Unit)

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.removeFavoriteFolder(0)
        advanceUntilIdle()
        coVerify(exactly = 0) { folderFavoritesRepository.removeFavorite(0) }

        viewModel.openFolderById(folderId = targetFolderId, folderName = "猫猫", folderPath = "/cats", coverMd5 = "md5-cats")
        advanceUntilIdle()
        assertEquals(MtPhotoMode.FOLDERS, viewModel.uiState.mode)
        assertEquals(targetFolderId, viewModel.uiState.folderHistory.last().folderId)

        viewModel.saveCurrentFolderFavorite()
        advanceUntilIdle()
        assertEquals(listOf(savedFavorite), viewModel.uiState.folderFavorites)
        assertEquals("目录收藏已保存", viewModel.uiState.message)

        viewModel.removeCurrentFolderFavorite()
        advanceUntilIdle()
        assertEquals(emptyList<MtPhotoFolderFavorite>(), viewModel.uiState.folderFavorites)
        assertEquals("已取消目录收藏", viewModel.uiState.message)
    }

    @Test
    fun `timeline fallback and preview import should surface success and error messages`() = runTest(mainDispatcherRule.dispatcher) {
        val folderId = 40L
        val detailPage = sampleFolderPage(
            folderName = "狗狗",
            folderPath = "/dogs",
            folders = emptyList(),
            files = listOf(sampleMedia(md5 = "dogs-file")),
            page = 1,
            totalPages = 2,
        )
        val imported = ImportedMtPhotoMedia(localPath = "/tmp/dogs.jpg", localFilename = "dogs.jpg", dedup = true)
        coEvery { repository.loadAlbums() } returns AppResult.Success(emptyList())
        coEvery { folderFavoritesRepository.loadFavorites() } returns AppResult.Success(emptyList())
        coEvery { repository.loadTimelineThreshold() } returns 10
        coEvery { repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = false) } returns AppResult.Success(detailPage)
        coEvery { repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = true) } returns AppResult.Error("时间线加载失败")
        coEvery { repository.importMedia("dogs-file") } returnsMany listOf(
            AppResult.Success(imported),
            AppResult.Error("导入失败"),
        )

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.openFolderById(folderId = folderId, folderName = "狗狗", folderPath = "/dogs")
        advanceUntilIdle()
        assertEquals(detailPage.files, viewModel.uiState.folderItems)
        assertEquals("时间线加载失败", viewModel.uiState.message)
        assertFalse(viewModel.uiState.folderTimelineDeferred)

        val importedCallbacks = mutableListOf<ImportedMtPhotoMedia>()
        val media = detailPage.files.first()
        viewModel.showPreview(media)
        assertEquals(media, viewModel.uiState.previewItem)

        viewModel.importPreview { importedCallbacks += it }
        advanceUntilIdle()
        assertEquals(listOf(imported), importedCallbacks)
        assertNull(viewModel.uiState.previewItem)
        assertEquals("已存在（去重复用）", viewModel.uiState.message)

        viewModel.showPreview(media)
        viewModel.importPreview()
        advanceUntilIdle()
        assertEquals(media, viewModel.uiState.previewItem)
        assertEquals("导入失败", viewModel.uiState.message)

        viewModel.dismissPreview()
        assertNull(viewModel.uiState.previewItem)
        assertFalse(viewModel.uiState.importingPreview)

        viewModel.consumeMessage()
        assertNull(viewModel.uiState.message)
    }

    @Test
    fun `load more folder should append next page and stop at last page`() = runTest(mainDispatcherRule.dispatcher) {
        val folderId = 50L
        val page1 = sampleFolderPage(
            folderName = "更多页",
            folderPath = "/more",
            folders = emptyList(),
            files = listOf(sampleMedia(md5 = "page-1")),
            page = 1,
            totalPages = 2,
        )
        val page2 = sampleFolderPage(
            folderName = "更多页",
            folderPath = "/more",
            folders = emptyList(),
            files = listOf(sampleMedia(md5 = "page-2")),
            page = 2,
            totalPages = 2,
        )
        coEvery { repository.loadAlbums() } returns AppResult.Success(emptyList())
        coEvery { folderFavoritesRepository.loadFavorites() } returns AppResult.Success(emptyList())
        coEvery { repository.loadTimelineThreshold() } returns 10
        coEvery { repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = false) } returns AppResult.Success(page1)
        coEvery { repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = true) } returns AppResult.Success(page1)
        coEvery { repository.loadFolderContent(folderId = folderId, page = 2, includeTimeline = true) } returns AppResult.Success(page2)

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.openFolderById(folderId = folderId, folderName = "更多页", folderPath = "/more")
        advanceUntilIdle()
        assertEquals(page1.files, viewModel.uiState.folderItems)

        viewModel.loadMoreFolder()
        advanceUntilIdle()
        assertEquals(page1.files + page2.files, viewModel.uiState.folderItems)
        assertEquals(2, viewModel.uiState.folderPage)

        viewModel.loadMoreFolder()
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.loadFolderContent(folderId = folderId, page = 2, includeTimeline = true) }
    }

    @Test
    fun `refresh current should reload visible album and folder views while ignoring same mode switch`() = runTest(mainDispatcherRule.dispatcher) {
        val album = sampleAlbum(id = 61L, name = "精选")
        val albumPage = sampleAlbumPage(page = 1, totalPages = 1, items = listOf(sampleMedia(md5 = "album-refresh")))
        val rootPage = sampleFolderPage(
            folderName = "根目录",
            folderPath = "/root",
            folders = listOf(sampleFolder(id = 62L, name = "刷新目录", path = "/refresh")),
            files = listOf(sampleMedia(md5 = "root-refresh")),
            page = 1,
            totalPages = 1,
        )
        val favorite = sampleFavorite(folderId = 62L, folderName = "刷新目录", folderPath = "/refresh")
        val folderPage = sampleFolderPage(
            folderName = "刷新目录",
            folderPath = "/refresh",
            folders = emptyList(),
            files = listOf(sampleMedia(md5 = "folder-refresh")),
            page = 1,
            totalPages = 1,
        )
        coEvery { repository.loadAlbums() } returnsMany listOf(
            AppResult.Success(listOf(album)),
            AppResult.Success(listOf(album)),
        )
        coEvery { repository.loadAlbumFiles(album.id, page = 1) } returnsMany listOf(
            AppResult.Success(albumPage),
            AppResult.Success(albumPage),
        )
        coEvery { repository.loadFolderRoot() } returnsMany listOf(
            AppResult.Success(rootPage),
            AppResult.Success(rootPage),
        )
        coEvery { folderFavoritesRepository.loadFavorites() } returnsMany listOf(
            AppResult.Success(listOf(favorite)),
            AppResult.Success(listOf(favorite)),
            AppResult.Success(listOf(favorite)),
        )
        coEvery { repository.loadTimelineThreshold() } returns 99
        coEvery { repository.loadFolderContent(folderId = 62L, page = 1, includeTimeline = false) } returnsMany listOf(
            AppResult.Success(folderPage),
            AppResult.Success(folderPage),
        )
        coEvery { repository.loadFolderContent(folderId = 62L, page = 1, includeTimeline = true) } returnsMany listOf(
            AppResult.Success(folderPage),
            AppResult.Success(folderPage),
        )

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.switchMode(MtPhotoMode.ALBUMS)
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.loadAlbums() }

        viewModel.refreshCurrent()
        advanceUntilIdle()
        coVerify(exactly = 2) { repository.loadAlbums() }

        viewModel.openAlbum(album)
        advanceUntilIdle()
        viewModel.refreshCurrent()
        advanceUntilIdle()
        coVerify(exactly = 2) { repository.loadAlbumFiles(album.id, page = 1) }

        viewModel.switchMode(MtPhotoMode.FOLDERS)
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.loadFolderRoot() }
        coVerify(exactly = 1) { folderFavoritesRepository.loadFavorites() }

        viewModel.refreshCurrent()
        advanceUntilIdle()
        coVerify(exactly = 2) { repository.loadFolderRoot() }
        coVerify(exactly = 2) { folderFavoritesRepository.loadFavorites() }

        viewModel.openFolderById(folderId = 62L, folderName = "刷新目录", folderPath = "/refresh")
        advanceUntilIdle()
        viewModel.refreshCurrent()
        advanceUntilIdle()
        coVerify(exactly = 2) { repository.loadFolderContent(folderId = 62L, page = 1, includeTimeline = false) }
        coVerify(exactly = 2) { repository.loadFolderContent(folderId = 62L, page = 1, includeTimeline = true) }
        coVerify(exactly = 3) { folderFavoritesRepository.loadFavorites() }
    }

    @Test
    fun `root guards back navigation and silent favorite refresh should preserve current message`() = runTest(mainDispatcherRule.dispatcher) {
        val childFolder = sampleFolder(id = 71L, name = "子目录", path = "/root/child")
        val rootPage = sampleFolderPage(
            folderName = "根目录",
            folderPath = "/root",
            folders = listOf(childFolder),
            files = listOf(sampleMedia(md5 = "root-guard")),
            page = 1,
            totalPages = 1,
        )
        val childPage = sampleFolderPage(
            folderName = "子目录",
            folderPath = "/root/child",
            folders = emptyList(),
            files = listOf(sampleMedia(md5 = "child-file")),
            page = 1,
            totalPages = 1,
        )
        coEvery { repository.loadAlbums() } returns AppResult.Success(emptyList())
        coEvery { repository.loadFolderRoot() } returnsMany listOf(
            AppResult.Success(rootPage),
            AppResult.Success(rootPage),
            AppResult.Success(rootPage),
        )
        coEvery { folderFavoritesRepository.loadFavorites() } returnsMany listOf(
            AppResult.Success(emptyList()),
            AppResult.Error("收藏刷新失败"),
        )
        coEvery { repository.loadTimelineThreshold() } returns 99
        coEvery { repository.loadFolderContent(folderId = childFolder.id, page = 1, includeTimeline = false) } returns AppResult.Success(childPage)
        coEvery { repository.loadFolderContent(folderId = childFolder.id, page = 1, includeTimeline = true) } returns AppResult.Success(childPage)

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.switchMode(MtPhotoMode.FOLDERS)
        advanceUntilIdle()

        viewModel.saveCurrentFolderFavorite()
        assertEquals("根目录暂不支持收藏", viewModel.uiState.message)

        viewModel.loadFolderFavorites(silent = true)
        advanceUntilIdle()
        assertEquals("根目录暂不支持收藏", viewModel.uiState.message)

        viewModel.openFolder(childFolder)
        advanceUntilIdle()
        assertEquals(childFolder.id, viewModel.uiState.folderHistory.last().folderId)

        viewModel.backFolder()
        advanceUntilIdle()
        assertNull(viewModel.uiState.folderHistory.last().folderId)

        viewModel.backFolder()
        advanceUntilIdle()
        coVerify(exactly = 3) { repository.loadFolderRoot() }

        viewModel.openFolderById(folderId = 0L, folderName = "无效目录")
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.loadFolderContent(folderId = 0L, page = any(), includeTimeline = any()) }
    }

    @Test
    fun `favorite and pagination errors should surface messages`() = runTest(mainDispatcherRule.dispatcher) {
        val folderId = 81L
        val folderPage = sampleFolderPage(
            folderName = "错误目录",
            folderPath = "/error",
            folders = emptyList(),
            files = listOf(sampleMedia(md5 = "error-page-1")),
            page = 1,
            totalPages = 2,
        )
        coEvery { repository.loadAlbums() } returns AppResult.Success(emptyList())
        coEvery { folderFavoritesRepository.loadFavorites() } returns AppResult.Success(emptyList())
        coEvery { repository.loadTimelineThreshold() } returns 99
        coEvery { repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = false) } returns AppResult.Success(folderPage)
        coEvery { repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = true) } returns AppResult.Success(folderPage)
        coEvery { repository.loadFolderContent(folderId = folderId, page = 2, includeTimeline = true) } returns AppResult.Error("更多加载失败")
        coEvery {
            folderFavoritesRepository.upsertFavorite(
                folderId = folderId,
                folderName = "错误目录",
                folderPath = "/error",
                coverMd5 = "",
            )
        } returns AppResult.Error("收藏保存失败")
        coEvery { folderFavoritesRepository.removeFavorite(folderId) } returns AppResult.Error("收藏取消失败")

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.openFolderById(folderId = folderId, folderName = "错误目录", folderPath = "/error")
        advanceUntilIdle()

        viewModel.saveCurrentFolderFavorite()
        advanceUntilIdle()
        assertEquals("收藏保存失败", viewModel.uiState.message)

        viewModel.removeCurrentFolderFavorite()
        advanceUntilIdle()
        assertEquals("收藏取消失败", viewModel.uiState.message)

        viewModel.loadMoreFolder()
        advanceUntilIdle()
        assertEquals("更多加载失败", viewModel.uiState.message)
    }

    @Test
    fun `root guards and refresh should preserve message while reloading folder root`() = runTest(mainDispatcherRule.dispatcher) {
        val rootPage = sampleFolderPage(
            folderName = "根目录",
            folderPath = "/root",
            folders = listOf(sampleFolder(id = 60L, name = "子目录", path = "/root/sub")),
            files = listOf(sampleMedia(md5 = "root-file")),
            page = 1,
            totalPages = 1,
        )
        coEvery { repository.loadAlbums() } returns AppResult.Success(emptyList())
        coEvery { repository.loadFolderRoot() } returns AppResult.Success(rootPage)
        coEvery { folderFavoritesRepository.loadFavorites() } returnsMany listOf(
            AppResult.Success(emptyList()),
            AppResult.Error("刷新收藏失败"),
            AppResult.Success(emptyList()),
        )

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.switchMode(MtPhotoMode.FOLDERS)
        advanceUntilIdle()
        assertEquals(rootPage.files, viewModel.uiState.folderItems)

        viewModel.saveCurrentFolderFavorite()
        assertEquals("根目录暂不支持收藏", viewModel.uiState.message)

        viewModel.loadFolderTimeline()
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.loadTimelineThreshold() }

        viewModel.loadFolderFavorites(silent = true)
        advanceUntilIdle()
        assertEquals("根目录暂不支持收藏", viewModel.uiState.message)

        viewModel.refreshCurrent()
        advanceUntilIdle()
        assertEquals(rootPage.files, viewModel.uiState.folderItems)
        coVerify(exactly = 2) { repository.loadFolderRoot() }

        viewModel.backFolder()
        advanceUntilIdle()
        coVerify(exactly = 3) { repository.loadFolderRoot() }
    }

    @Test
    fun `refresh current should reload albums and nested favorite folder`() = runTest(mainDispatcherRule.dispatcher) {
        val album = sampleAlbum(id = 70L, name = "精选")
        val albumPage = sampleAlbumPage(page = 1, totalPages = 1, items = listOf(sampleMedia(md5 = "album-1")))
        val rootPage = sampleFolderPage(
            folderName = "根目录",
            folderPath = "/root",
            folders = emptyList(),
            files = listOf(sampleMedia(md5 = "root-refresh")),
            page = 1,
            totalPages = 1,
        )
        val favorite = sampleFavorite(folderId = 71L, folderName = "收藏目录", folderPath = "/favorites/71", coverMd5 = "cover-71")
        val detailPage = sampleFolderPage(
            folderName = "收藏目录",
            folderPath = "/favorites/71",
            folders = listOf(sampleFolder(id = 72L, name = "子项", path = "/favorites/71/sub")),
            files = listOf(sampleMedia(md5 = "fav-file")),
            page = 1,
            totalPages = 1,
        )
        coEvery { repository.loadAlbums() } returnsMany listOf(
            AppResult.Success(listOf(album)),
            AppResult.Success(listOf(album)),
        )
        coEvery { repository.loadAlbumFiles(album.id, page = 1) } returnsMany listOf(
            AppResult.Success(albumPage),
            AppResult.Success(albumPage),
        )
        coEvery { repository.loadFolderRoot() } returns AppResult.Success(rootPage)
        coEvery { folderFavoritesRepository.loadFavorites() } returns AppResult.Success(listOf(favorite))
        coEvery { repository.loadTimelineThreshold() } returns 10
        coEvery { repository.loadFolderContent(folderId = favorite.folderId, page = 1, includeTimeline = false) } returnsMany listOf(
            AppResult.Success(detailPage),
            AppResult.Success(detailPage),
        )
        coEvery { repository.loadFolderContent(folderId = favorite.folderId, page = 1, includeTimeline = true) } returnsMany listOf(
            AppResult.Success(detailPage),
            AppResult.Success(detailPage),
        )

        val viewModel = MtPhotoViewModel(repository, folderFavoritesRepository)
        advanceUntilIdle()

        viewModel.refreshCurrent()
        advanceUntilIdle()
        coVerify(exactly = 2) { repository.loadAlbums() }

        viewModel.openAlbum(album)
        advanceUntilIdle()
        viewModel.showPreview(sampleMedia(md5 = "preview-refresh"))
        viewModel.refreshCurrent()
        advanceUntilIdle()
        assertNull(viewModel.uiState.previewItem)
        coVerify(exactly = 2) { repository.loadAlbumFiles(album.id, page = 1) }

        viewModel.switchMode(MtPhotoMode.FOLDERS)
        advanceUntilIdle()
        viewModel.openFavoriteFolder(favorite)
        advanceUntilIdle()
        assertEquals(favorite.folderId, viewModel.uiState.folderHistory.last().folderId)

        viewModel.refreshCurrent()
        advanceUntilIdle()
        coVerify(exactly = 2) { repository.loadFolderContent(folderId = favorite.folderId, page = 1, includeTimeline = false) }
        coVerify(exactly = 2) { repository.loadFolderContent(folderId = favorite.folderId, page = 1, includeTimeline = true) }

        viewModel.backFolder()
        advanceUntilIdle()
        assertNull(viewModel.uiState.folderHistory.last().folderId)
        assertEquals(rootPage.files, viewModel.uiState.folderItems)
        coVerify(atLeast = 2) { repository.loadFolderRoot() }
    }

    private fun sampleAlbum(id: Long, name: String): MtPhotoAlbumSummary = MtPhotoAlbumSummary(
        id = id,
        name = name,
        coverMd5 = "cover-$id",
        coverUrl = "https://example.test/$id.jpg",
        count = 1,
    )

    private fun sampleFolder(id: Long, name: String, path: String): MtPhotoFolderSummary = MtPhotoFolderSummary(
        id = id,
        name = name,
        path = path,
        coverMd5 = "cover-$id",
        coverUrl = "https://example.test/folder/$id.jpg",
        subFolderNum = 0,
        subFileNum = 1,
    )

    private fun sampleMedia(md5: String): MtPhotoMediaSummary = MtPhotoMediaSummary(
        id = md5.hashCode().toLong(),
        md5 = md5,
        type = ChatMessageType.IMAGE,
        title = "媒体-$md5",
        subtitle = "2026-03-16",
        thumbUrl = "https://example.test/thumb/$md5.jpg",
    )

    private fun sampleAlbumPage(
        page: Int,
        totalPages: Int,
        items: List<MtPhotoMediaSummary>,
    ): MtPhotoAlbumPage = MtPhotoAlbumPage(
        items = items,
        total = items.size * totalPages,
        page = page,
        totalPages = totalPages,
    )

    private fun sampleFolderPage(
        folderName: String,
        folderPath: String,
        folders: List<MtPhotoFolderSummary>,
        files: List<MtPhotoMediaSummary>,
        page: Int,
        totalPages: Int,
    ): MtPhotoFolderPage = MtPhotoFolderPage(
        folderName = folderName,
        folderPath = folderPath,
        folders = folders,
        files = files,
        total = files.size * totalPages,
        page = page,
        totalPages = totalPages,
    )

    private fun sampleFavorite(
        folderId: Long,
        folderName: String,
        folderPath: String,
        coverMd5: String = "",
    ): MtPhotoFolderFavorite = MtPhotoFolderFavorite(
        id = folderId,
        folderId = folderId,
        folderName = folderName,
        folderPath = folderPath,
        coverMd5 = coverMd5,
        coverUrl = if (coverMd5.isBlank()) "" else "https://example.test/favorite/$coverMd5.jpg",
        tags = emptyList(),
        note = "",
        updateTime = "2026-03-16 20:00:00",
    )
}
