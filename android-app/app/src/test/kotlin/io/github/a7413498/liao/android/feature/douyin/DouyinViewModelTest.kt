package io.github.a7413498.liao.android.feature.douyin

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
class DouyinViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<DouyinRepository>()

    @Test
    fun `switch mode clear current mode and cookie editor should refresh favorites as needed`() = runTest(mainDispatcherRule.dispatcher) {
        stubFavoritesSequence(
            AppResult.Success(emptySnapshot()),
            AppResult.Error("收藏刷新失败"),
            AppResult.Success(emptySnapshot()),
        )
        coEvery { repository.resolveDetail("detail-1", "") } returns AppResult.Success(sampleDetail())
        coEvery { repository.resolveAccount("account-1", "") } returns AppResult.Success(sampleAccount())

        val viewModel = DouyinViewModel(repository)
        advanceUntilIdle()

        viewModel.toggleCookieEditor()
        assertTrue(viewModel.uiState.showCookieEditor)
        viewModel.toggleCookieEditor()
        assertFalse(viewModel.uiState.showCookieEditor)

        viewModel.switchMode(DouyinScreenMode.DETAIL)
        advanceUntilIdle()
        coVerify(exactly = 1) { repository.refreshFavoritesSnapshot() }

        viewModel.switchMode(DouyinScreenMode.FAVORITES)
        advanceUntilIdle()
        assertEquals(DouyinScreenMode.FAVORITES, viewModel.uiState.mode)
        assertEquals("收藏刷新失败", viewModel.uiState.message)

        viewModel.clearCurrentMode()
        advanceUntilIdle()
        coVerify(exactly = 3) { repository.refreshFavoritesSnapshot() }

        viewModel.switchMode(DouyinScreenMode.DETAIL)
        viewModel.updateInput("detail-1")
        viewModel.resolveDetail()
        advanceUntilIdle()
        assertEquals("解析成功，共 1 项", viewModel.uiState.message)
        viewModel.showPreview(sampleMediaItem(index = 0))
        viewModel.clearCurrentMode()
        assertEquals("", viewModel.uiState.input)
        assertNull(viewModel.uiState.result)
        assertNull(viewModel.uiState.previewItem)
        assertTrue(viewModel.uiState.importedItems.isEmpty())

        viewModel.switchMode(DouyinScreenMode.ACCOUNT)
        viewModel.updateAccountInput("account-1")
        viewModel.resolveAccount()
        advanceUntilIdle()
        assertEquals("已获取 1 个作品", viewModel.uiState.message)
        viewModel.clearCurrentMode()
        assertEquals("", viewModel.uiState.accountInput)
        assertNull(viewModel.uiState.accountResult)
    }

    @Test
    fun `resolve detail and account should handle success empty payload and error`() = runTest(mainDispatcherRule.dispatcher) {
        stubFavoritesSequence(AppResult.Success(emptySnapshot()))
        coEvery { repository.resolveDetail("detail-empty", "cookie-1") } returns AppResult.Success(sampleDetail(items = emptyList()))
        coEvery { repository.resolveDetail("detail-bad", "cookie-1") } returns AppResult.Error("解析失败")
        coEvery { repository.resolveAccount("account-empty", "cookie-1") } returns AppResult.Success(sampleAccount(items = emptyList()))
        coEvery { repository.resolveAccount("account-bad", "cookie-1") } returns AppResult.Error("获取失败")

        val viewModel = DouyinViewModel(repository)
        advanceUntilIdle()
        viewModel.updateCookie("cookie-1")

        viewModel.updateInput("detail-empty")
        viewModel.resolveDetail()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.loading)
        assertEquals("解析成功，但未返回可预览媒体", viewModel.uiState.message)

        viewModel.updateInput("detail-bad")
        viewModel.resolveDetail()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.loading)
        assertNull(viewModel.uiState.result)
        assertEquals("解析失败", viewModel.uiState.message)

        viewModel.updateAccountInput("account-empty")
        viewModel.resolveAccount()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.accountLoading)
        assertEquals("获取成功，但当前页暂无作品", viewModel.uiState.message)

        viewModel.updateAccountInput("account-bad")
        viewModel.resolveAccount()
        advanceUntilIdle()
        assertFalse(viewModel.uiState.accountLoading)
        assertNull(viewModel.uiState.accountResult)
        assertEquals("获取失败", viewModel.uiState.message)
    }

    @Test
    fun `import item should guard missing result handle success error and already imported`() = runTest(mainDispatcherRule.dispatcher) {
        val detail = sampleDetail(items = listOf(sampleMediaItem(index = 0), sampleMediaItem(index = 1)))
        val imported = ImportedDouyinMedia(localPath = "/tmp/a.jpg", localFilename = "a.jpg", dedup = false)
        stubFavoritesSequence(AppResult.Success(emptySnapshot()))
        coEvery { repository.resolveDetail("detail-1", "") } returns AppResult.Success(detail)
        coEvery { repository.importMedia(detail.key, 0) } returns AppResult.Success(imported)
        coEvery { repository.importMedia(detail.key, 1) } returns AppResult.Error("导入失败")

        val viewModel = DouyinViewModel(repository)
        advanceUntilIdle()

        viewModel.importItem(sampleMediaItem(index = 0))
        assertEquals("请先解析抖音作品", viewModel.uiState.message)

        viewModel.updateInput("detail-1")
        viewModel.resolveDetail()
        advanceUntilIdle()

        val callbacks = mutableListOf<ImportedDouyinMedia>()
        val firstItem = detail.items[0]
        viewModel.showPreview(firstItem)
        viewModel.importItem(firstItem) { callbacks += it }
        advanceUntilIdle()
        assertEquals(listOf(imported), callbacks)
        assertNull(viewModel.uiState.previewItem)
        assertEquals(DouyinImportStatus.IMPORTED, viewModel.uiState.importedItems[firstItem.index])
        assertNull(viewModel.uiState.message)

        viewModel.importItem(firstItem)
        assertEquals("该媒体已导入到本地库", viewModel.uiState.message)

        val secondItem = detail.items[1]
        viewModel.showPreview(secondItem)
        viewModel.importItem(secondItem)
        advanceUntilIdle()
        assertEquals(secondItem, viewModel.uiState.previewItem)
        assertEquals("导入失败", viewModel.uiState.message)
    }

    @Test
    fun `favorite operations should handle guards add and remove branches`() = runTest(mainDispatcherRule.dispatcher) {
        val existingSnapshot = snapshot(
            users = listOf(sampleFavoriteUser(secUserId = "sec-1")),
            awemes = listOf(sampleFavoriteAweme(awemeId = "aweme-1")),
        )
        stubFavoritesSequence(
            AppResult.Success(existingSnapshot),
            AppResult.Success(snapshot(users = listOf(sampleFavoriteUser(secUserId = "sec-1")))),
            AppResult.Success(emptySnapshot()),
            AppResult.Success(snapshot(awemes = listOf(sampleFavoriteAweme(awemeId = "aweme-2")))),
        )
        coEvery { repository.resolveDetail("detail-blank", "") } returns AppResult.Success(sampleDetail(detailId = " "))
        coEvery { repository.resolveDetail("detail-ok", "") } returns AppResult.Success(sampleDetail(detailId = "aweme-1"))
        coEvery { repository.resolveAccount("account-blank", "") } returns AppResult.Success(sampleAccount(secUserId = " "))
        coEvery { repository.resolveAccount("account-ok", "") } returns AppResult.Success(sampleAccount(secUserId = "sec-1", items = listOf(sampleAccountItem(detailId = "aweme-2"))))
        coEvery { repository.removeFavoriteAweme("aweme-1") } returns AppResult.Success(Unit)
        coEvery { repository.removeFavoriteUser("sec-1") } returns AppResult.Success(Unit)
        coEvery {
            repository.upsertFavoriteAwemeFromAccount(
                "sec-1",
                match { it.detailId == "aweme-2" },
            )
        } returns AppResult.Success(sampleFavoriteAweme(awemeId = "aweme-2"))

        val viewModel = DouyinViewModel(repository)
        advanceUntilIdle()

        viewModel.toggleFavoriteCurrentDetail()
        assertEquals("请先解析抖音作品", viewModel.uiState.message)

        viewModel.updateInput("detail-blank")
        viewModel.resolveDetail()
        advanceUntilIdle()
        viewModel.toggleFavoriteCurrentDetail()
        assertEquals("当前解析结果缺少 awemeId", viewModel.uiState.message)

        viewModel.updateInput("detail-ok")
        viewModel.resolveDetail()
        advanceUntilIdle()
        viewModel.toggleFavoriteCurrentDetail()
        advanceUntilIdle()
        assertEquals("已取消收藏作品", viewModel.uiState.message)

        viewModel.toggleFavoriteCurrentUser()
        assertEquals("请先获取用户作品", viewModel.uiState.message)

        viewModel.updateAccountInput("account-blank")
        viewModel.resolveAccount()
        advanceUntilIdle()
        viewModel.toggleFavoriteCurrentUser()
        assertEquals("当前用户缺少 secUserId", viewModel.uiState.message)

        viewModel.updateAccountInput("account-ok")
        viewModel.resolveAccount()
        advanceUntilIdle()
        viewModel.toggleFavoriteCurrentUser()
        advanceUntilIdle()
        assertEquals("已取消收藏作者", viewModel.uiState.message)

        viewModel.toggleFavoriteAccountAweme(sampleAccountItem(detailId = " "))
        assertEquals("当前作品缺少 awemeId", viewModel.uiState.message)

        val accountItem = sampleAccountItem(detailId = "aweme-2")
        viewModel.toggleFavoriteAccountAweme(accountItem)
        advanceUntilIdle()
        assertEquals("已收藏作品", viewModel.uiState.message)

        viewModel.removeFavoriteUser(" ")
        viewModel.removeFavoriteAweme("  ")
        coVerify(exactly = 1) { repository.removeFavoriteUser("sec-1") }
        coVerify(exactly = 1) { repository.removeFavoriteAweme("aweme-1") }
    }

    @Test
    fun `tag editor selection and save should handle guard success and error`() = runTest(mainDispatcherRule.dispatcher) {
        val initSnapshot = snapshot(userTags = listOf(sampleTag(1, "人物"), sampleTag(2, "作者")))
        stubFavoritesSequence(
            AppResult.Success(initSnapshot),
            AppResult.Success(snapshot(userTags = listOf(sampleTag(1, "人物")))),
        )
        coEvery { repository.applyTags(DouyinTagKind.USERS, "sec-1", listOf(1L, 3L)) } returns AppResult.Error("标签保存失败")
        coEvery { repository.applyTags(DouyinTagKind.USERS, "sec-2", listOf(1L)) } returns AppResult.Success(Unit)

        val viewModel = DouyinViewModel(repository)
        advanceUntilIdle()

        viewModel.openTagEditor(DouyinTagKind.USERS, "   ", "", listOf(-1L, 1L, 2L))
        assertNull(viewModel.uiState.tagDialog)

        viewModel.openTagEditor(DouyinTagKind.USERS, " sec-1 ", "标题", listOf(-1L, 1L, 2L))
        assertEquals(setOf(1L, 2L), viewModel.uiState.tagDialog?.selectedTagIds)

        viewModel.toggleTagSelection(0L)
        assertEquals(setOf(1L, 2L), viewModel.uiState.tagDialog?.selectedTagIds)
        viewModel.toggleTagSelection(2L)
        viewModel.toggleTagSelection(3L)
        assertEquals(setOf(1L, 3L), viewModel.uiState.tagDialog?.selectedTagIds)

        viewModel.saveTagSelection()
        advanceUntilIdle()
        assertEquals("标签保存失败", viewModel.uiState.message)
        assertEquals("标签保存失败", viewModel.uiState.tagDialog?.error)
        assertFalse(viewModel.uiState.tagDialog?.saving ?: true)

        viewModel.closeTagEditor()
        assertNull(viewModel.uiState.tagDialog)

        viewModel.openTagEditor(DouyinTagKind.USERS, "sec-2", "", listOf(1L))
        viewModel.saveTagSelection()
        advanceUntilIdle()
        assertNull(viewModel.uiState.tagDialog)
        assertEquals("已更新标签", viewModel.uiState.message)
    }

    @Test
    fun `tag manager preview and consume message should handle guard success and error`() = runTest(mainDispatcherRule.dispatcher) {
        stubFavoritesSequence(
            AppResult.Success(emptySnapshot()),
            AppResult.Success(emptySnapshot()),
            AppResult.Success(emptySnapshot()),
        )
        coEvery { repository.createTag(DouyinTagKind.USERS, "人物") } returns AppResult.Success(sampleTag(5L, "人物"))
        coEvery { repository.removeTag(DouyinTagKind.USERS, 5L) } returns AppResult.Error("删除标签失败")

        val viewModel = DouyinViewModel(repository)
        advanceUntilIdle()

        viewModel.closeTagManager()
        assertNull(viewModel.uiState.tagManager)
        viewModel.updateTagManagerName("ignored")
        assertNull(viewModel.uiState.tagManager)

        viewModel.openTagManager(DouyinTagKind.USERS)
        viewModel.createTag()
        coVerify(exactly = 0) { repository.createTag(any(), any()) }

        viewModel.updateTagManagerName("人物")
        viewModel.createTag()
        advanceUntilIdle()
        assertEquals("已创建标签", viewModel.uiState.message)
        assertEquals("", viewModel.uiState.tagManager?.nameInput)
        assertFalse(viewModel.uiState.tagManager?.creating ?: true)

        viewModel.removeTag(0L)
        coVerify(exactly = 0) { repository.removeTag(DouyinTagKind.USERS, 0L) }

        viewModel.removeTag(5L)
        advanceUntilIdle()
        assertEquals("删除标签失败", viewModel.uiState.message)
        assertEquals("删除标签失败", viewModel.uiState.tagManager?.error)
        assertNull(viewModel.uiState.tagManager?.removingTagId)

        val preview = sampleMediaItem(index = 9)
        viewModel.showPreview(preview)
        assertEquals(preview, viewModel.uiState.previewItem)
        viewModel.dismissPreview()
        assertNull(viewModel.uiState.previewItem)
        viewModel.dismissPreview()
        assertNull(viewModel.uiState.previewItem)

        viewModel.consumeMessage()
        assertNull(viewModel.uiState.message)
    }

    private fun stubFavoritesSequence(vararg results: AppResult<DouyinFavoritesSnapshot>) {
        val queue = ArrayDeque(results.toList())
        coEvery { repository.refreshFavoritesSnapshot() } answers {
            if (queue.isEmpty()) AppResult.Success(emptySnapshot()) else queue.removeFirst()
        }
    }

    private fun emptySnapshot(): DouyinFavoritesSnapshot = snapshot()

    private fun snapshot(
        users: List<DouyinFavoriteUser> = emptyList(),
        awemes: List<DouyinFavoriteAweme> = emptyList(),
        userTags: List<DouyinFavoriteTag> = emptyList(),
        awemeTags: List<DouyinFavoriteTag> = emptyList(),
    ): DouyinFavoritesSnapshot = DouyinFavoritesSnapshot(
        users = users,
        awemes = awemes,
        userTags = userTags,
        awemeTags = awemeTags,
    )

    private fun sampleMediaItem(index: Int): DouyinMediaItem = DouyinMediaItem(
        index = index,
        type = "image",
        url = "https://example.test/$index.jpg",
        downloadUrl = "https://example.test/$index.jpg",
        originalFilename = "item-$index.jpg",
        thumbUrl = "https://example.test/$index-thumb.jpg",
    )

    private fun sampleDetail(
        key: String = "key-1",
        detailId: String = "aweme-1",
        items: List<DouyinMediaItem> = listOf(sampleMediaItem(index = 0)),
    ): DouyinDetailResult = DouyinDetailResult(
        key = key,
        detailId = detailId,
        type = "image",
        mediaType = "image",
        title = "作品",
        coverUrl = "https://example.test/cover.jpg",
        imageCount = items.size,
        videoDuration = 0.0,
        isLivePhoto = false,
        livePhotoPairs = 0,
        items = items,
    )

    private fun sampleAccount(
        secUserId: String = "sec-1",
        items: List<DouyinAccountItem> = listOf(sampleAccountItem(detailId = "aweme-2")),
    ): DouyinAccountResult = DouyinAccountResult(
        secUserId = secUserId,
        displayName = "作者",
        signature = "简介",
        avatarUrl = "https://example.test/avatar.jpg",
        profileUrl = "https://example.test/profile",
        followerCount = 1L,
        followingCount = 2L,
        awemeCount = 3L,
        totalFavorited = 4L,
        items = items,
    )

    private fun sampleAccountItem(detailId: String): DouyinAccountItem = DouyinAccountItem(
        detailId = detailId,
        type = "image",
        mediaType = "image",
        desc = "描述",
        coverUrl = "https://example.test/account-cover.jpg",
        imageCount = 1,
        videoDuration = 0.0,
        isLivePhoto = false,
        livePhotoPairs = 0,
        isPinned = false,
        pinnedRank = null,
        publishAt = "2026-03-17",
        status = "ok",
        authorUniqueId = "author-1",
        authorName = "作者",
    )

    private fun sampleFavoriteUser(secUserId: String): DouyinFavoriteUser = DouyinFavoriteUser(
        secUserId = secUserId,
        sourceInput = secUserId,
        displayName = "作者",
        signature = "简介",
        avatarUrl = "",
        profileUrl = "",
        followerCount = 1L,
        followingCount = 2L,
        awemeCount = 3L,
        totalFavorited = 4L,
        lastParsedAt = "2026-03-17",
        lastParsedCount = 1,
        createTime = "2026-03-17",
        updateTime = "2026-03-17",
        tagIds = listOf(1L),
    )

    private fun sampleFavoriteAweme(awemeId: String): DouyinFavoriteAweme = DouyinFavoriteAweme(
        awemeId = awemeId,
        secUserId = "sec-1",
        type = "image",
        desc = "作品",
        coverUrl = "https://example.test/favorite-cover.jpg",
        createTime = "2026-03-17",
        updateTime = "2026-03-17",
        tagIds = listOf(1L),
    )

    private fun sampleTag(id: Long, name: String): DouyinFavoriteTag = DouyinFavoriteTag(
        id = id,
        name = name,
        sortOrder = id,
        count = 1L,
        createTime = "2026-03-17",
        updateTime = "2026-03-17",
    )
}
