package io.github.a7413498.liao.android.feature.chatlist

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.network.ChatArchiveSearchItemDto
import io.github.a7413498.liao.android.core.network.ContactCandidateDto
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.LiaoWsEnvelope
import io.github.a7413498.liao.android.core.websocket.LiaoWsEvent
import io.github.a7413498.liao.android.core.websocket.LiaoWsKnownCode
import io.github.a7413498.liao.android.core.websocket.MatchCandidate
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import io.mockk.verify
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class ChatListViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<ChatListRepository>()
    private val webSocketClient = mockk<LiaoWebSocketClient>()
    private val events = MutableSharedFlow<LiaoWsEvent>(extraBufferCapacity = 8)
    private val historyFlow = MutableStateFlow(listOf(samplePeer("peer-1", "Alice")))
    private val favoriteFlow = MutableStateFlow(listOf(samplePeer("peer-2", "Bob")))

    @Before
    fun setUp() {
        coEvery { repository.loadGlobalFavoriteTargetIds() } returns AppResult.Success(emptySet())
    }

    @Test
    fun `init should collect history tab items and refresh successfully`() = runTest(mainDispatcherRule.dispatcher) {
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        assertEquals(ConversationTab.HISTORY, viewModel.uiState.tab)
        assertEquals(historyFlow.value, viewModel.uiState.items)
        assertEquals(false, viewModel.uiState.loading)
        assertNull(viewModel.uiState.errorMessage)
    }

    @Test
    fun `init should load global favorite target ids`() = runTest(mainDispatcherRule.dispatcher) {
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.loadGlobalFavoriteTargetIds() } returns AppResult.Success(setOf("peer-1"))

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        assertEquals(setOf("peer-1"), viewModel.uiState.globalFavoriteTargetIds)
    }

    @Test
    fun `switch tab should skip same tab and load favorite on change`() = runTest(mainDispatcherRule.dispatcher) {
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.switchTab(ConversationTab.HISTORY)
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.loadFavorite() }

        viewModel.switchTab(ConversationTab.FAVORITE)
        advanceUntilIdle()
        assertEquals(ConversationTab.FAVORITE, viewModel.uiState.tab)
        assertEquals(favoriteFlow.value, viewModel.uiState.items)
        coVerify(atLeast = 1) { repository.loadFavorite() }
    }

    @Test
    fun `refresh should surface repository error and mark peer read should delegate`() = runTest(mainDispatcherRule.dispatcher) {
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Error("收藏刷新失败")
        coEvery { repository.markPeerRead("peer-2") } returns Unit

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()
        viewModel.switchTab(ConversationTab.FAVORITE)
        advanceUntilIdle()

        assertEquals("收藏刷新失败", viewModel.uiState.errorMessage)

        viewModel.markPeerRead("peer-2")
        advanceUntilIdle()
        coVerify { repository.markPeerRead("peer-2") }
    }

    @Test
    fun `clear peer unread should delegate only for unread conversations`() = runTest(mainDispatcherRule.dispatcher) {
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.markPeerRead("peer-unread") } returns Unit

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.clearPeerUnread(samplePeer("peer-unread", "未读用户", unreadCount = 2))
        advanceUntilIdle()
        viewModel.clearPeerUnread(samplePeer("peer-read", "已读用户", unreadCount = 0))
        advanceUntilIdle()

        coVerify(exactly = 1) { repository.markPeerRead("peer-unread") }
        coVerify(exactly = 0) { repository.markPeerRead("peer-read") }
    }

    @Test
    fun `delete pending peer should close dialog and emit success message`() = runTest(mainDispatcherRule.dispatcher) {
        val peer = samplePeer("peer-delete", "待删除")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.deletePeer("peer-delete") } returns AppResult.Success(Unit)

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.requestDeletePeer(peer)
        assertEquals(peer, viewModel.uiState.deleteConfirmPeer)

        viewModel.deletePendingPeer()
        advanceUntilIdle()

        assertNull(viewModel.uiState.deleteConfirmPeer)
        assertNull(viewModel.uiState.deletingPeerId)
        assertEquals("删除成功", viewModel.uiState.infoMessage)
        coVerify { repository.deletePeer("peer-delete") }
    }

    @Test
    fun `delete pending peer should close dialog and surface failure`() = runTest(mainDispatcherRule.dispatcher) {
        val peer = samplePeer("peer-delete", "待删除")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.deletePeer("peer-delete") } returns AppResult.Error("删除失败")

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.requestDeletePeer(peer)
        viewModel.deletePendingPeer()
        advanceUntilIdle()

        assertNull(viewModel.uiState.deleteConfirmPeer)
        assertNull(viewModel.uiState.deletingPeerId)
        assertEquals("删除失败", viewModel.uiState.errorMessage)
    }

    @Test
    fun `toggle global favorite should update ids and message on add success`() = runTest(mainDispatcherRule.dispatcher) {
        val peer = samplePeer("peer-global", "全局用户")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.toggleGlobalFavorite(peer, isGlobalFavorite = false) } returns AppResult.Success(true)

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.toggleGlobalFavorite(peer)
        advanceUntilIdle()

        assertEquals(setOf("peer-global"), viewModel.uiState.globalFavoriteTargetIds)
        assertNull(viewModel.uiState.togglingGlobalFavoritePeerId)
        assertEquals("已加入全局收藏", viewModel.uiState.infoMessage)
    }

    @Test
    fun `toggle global favorite should update ids and message on remove success`() = runTest(mainDispatcherRule.dispatcher) {
        val peer = samplePeer("peer-global", "全局用户")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.loadGlobalFavoriteTargetIds() } returns AppResult.Success(setOf("peer-global"))
        coEvery { repository.toggleGlobalFavorite(peer, isGlobalFavorite = true) } returns AppResult.Success(false)

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.toggleGlobalFavorite(peer)
        advanceUntilIdle()

        assertEquals(emptySet<String>(), viewModel.uiState.globalFavoriteTargetIds)
        assertNull(viewModel.uiState.togglingGlobalFavoritePeerId)
        assertEquals("已取消全局收藏", viewModel.uiState.infoMessage)
    }

    @Test
    fun `toggle global favorite should keep ids on failure`() = runTest(mainDispatcherRule.dispatcher) {
        val peer = samplePeer("peer-global", "全局用户")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.toggleGlobalFavorite(peer, isGlobalFavorite = false) } returns AppResult.Error("添加失败")

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.toggleGlobalFavorite(peer)
        advanceUntilIdle()

        assertEquals(emptySet<String>(), viewModel.uiState.globalFavoriteTargetIds)
        assertNull(viewModel.uiState.togglingGlobalFavoritePeerId)
        assertEquals("添加失败", viewModel.uiState.errorMessage)
    }

    @Test
    fun `websocket events and consume info message should update snackbar text`() = runTest(mainDispatcherRule.dispatcher) {
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        events.tryEmit(connectNotice(""))
        advanceUntilIdle()
        assertNull(viewModel.uiState.infoMessage)

        events.tryEmit(connectNotice("连接成功"))
        advanceUntilIdle()
        assertEquals("连接成功", viewModel.uiState.infoMessage)
        viewModel.consumeInfoMessage()
        assertNull(viewModel.uiState.infoMessage)

        events.tryEmit(matchCancelled("对方取消了匹配"))
        advanceUntilIdle()
        assertEquals("对方取消了匹配", viewModel.uiState.infoMessage)

        events.tryEmit(LiaoWsEvent.MatchSuccess(raw = "match", candidate = MatchCandidate("peer-3", "Carol", "女", "18", "上海")))
        advanceUntilIdle()
        assertEquals("匹配成功：Carol", viewModel.uiState.infoMessage)
    }

    @Test
    fun `check peer online status should request websocket and show status dialog on result`() = runTest(mainDispatcherRule.dispatcher) {
        val peer = samplePeer("peer-online", "在线用户")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.requestOnlineStatus("peer-online") } returns AppResult.Success(Unit)

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.checkPeerOnlineStatus(peer)
        advanceUntilIdle()

        assertEquals("peer-online", viewModel.uiState.checkingOnlinePeerId)
        assertEquals("正在查询在线状态...", viewModel.uiState.infoMessage)
        coVerify { repository.requestOnlineStatus("peer-online") }

        events.tryEmit(
            LiaoWsEvent.OnlineStatus(
                raw = "online",
                envelope = sampleEnvelope(LiaoWsKnownCode.OnlineStatus),
                message = "已返回在线状态",
                isOnline = true,
                lastTime = "2026-07-07 13:00",
            )
        )
        advanceUntilIdle()

        assertEquals(null, viewModel.uiState.checkingOnlinePeerId)
        assertEquals(true, viewModel.uiState.onlineStatusVisible)
        assertEquals("在线用户", viewModel.uiState.onlineStatusPeerName)
        assertEquals(true, viewModel.uiState.onlineStatusOnline)
        assertEquals("2026-07-07 13:00", viewModel.uiState.onlineStatusLastTime)

        viewModel.dismissOnlineStatus()
        assertEquals(false, viewModel.uiState.onlineStatusVisible)
    }

    @Test
    fun `check peer online status should clear loading and surface failure`() = runTest(mainDispatcherRule.dispatcher) {
        val peer = samplePeer("peer-offline", "离线用户")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.requestOnlineStatus("peer-offline") } returns AppResult.Error("WebSocket 未连接")

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.checkPeerOnlineStatus(peer)
        advanceUntilIdle()

        assertEquals(null, viewModel.uiState.checkingOnlinePeerId)
        assertEquals("WebSocket 未连接", viewModel.uiState.errorMessage)
        assertEquals(false, viewModel.uiState.onlineStatusVisible)
    }

    @Test
    fun `batch selection should keep failed peers selected after partial delete`() = runTest(mainDispatcherRule.dispatcher) {
        val peer1 = samplePeer("peer-1", "Alice")
        val peer2 = samplePeer("peer-2", "Bob")
        historyFlow.value = listOf(peer1, peer2)
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.batchDeletePeers(listOf("peer-1", "peer-2")) } returns AppResult.Success(
            BatchDeletePeersResult(
                requestedIds = listOf("peer-1", "peer-2"),
                successIds = listOf("peer-1"),
                failedIds = setOf("peer-2"),
                failedReasons = mapOf("peer-2" to "上游失败"),
            )
        )

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.enterSelectionMode(peer1)
        viewModel.togglePeerSelection(peer2.id)
        viewModel.requestBatchDelete()
        assertEquals(true, viewModel.uiState.selectionMode)
        assertEquals(setOf("peer-1", "peer-2"), viewModel.uiState.selectedPeerIds)
        assertEquals(true, viewModel.uiState.batchDeleteConfirmVisible)

        viewModel.deleteSelectedPeers()
        advanceUntilIdle()

        assertEquals(true, viewModel.uiState.selectionMode)
        assertEquals(setOf("peer-2"), viewModel.uiState.selectedPeerIds)
        assertEquals(false, viewModel.uiState.batchDeleteConfirmVisible)
        assertEquals(false, viewModel.uiState.batchDeleting)
        assertEquals("已删除 1 个，失败 1 个", viewModel.uiState.infoMessage)
    }

    @Test
    fun `batch delete should exit selection mode when all selected peers are deleted`() = runTest(mainDispatcherRule.dispatcher) {
        val peer1 = samplePeer("peer-1", "Alice")
        val peer2 = samplePeer("peer-2", "Bob")
        historyFlow.value = listOf(peer1, peer2)
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.batchDeletePeers(listOf("peer-1", "peer-2")) } returns AppResult.Success(
            BatchDeletePeersResult(
                requestedIds = listOf("peer-1", "peer-2"),
                successIds = listOf("peer-1", "peer-2"),
                failedIds = emptySet(),
                failedReasons = emptyMap(),
            )
        )

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.enterSelectionMode()
        viewModel.selectAllVisiblePeers()
        viewModel.requestBatchDelete()
        viewModel.deleteSelectedPeers()
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.selectionMode)
        assertEquals(emptySet<String>(), viewModel.uiState.selectedPeerIds)
        assertEquals("已删除 2 个会话", viewModel.uiState.infoMessage)
    }

    @Test
    fun `archive search should load results and open selected archived chat`() = runTest(mainDispatcherRule.dispatcher) {
        val archived = ChatArchiveSearchItemDto(
            ownerUserId = "owner-1",
            targetUserId = "peer-archived",
            nickname = "归档用户",
            sources = listOf("archive"),
            localArchived = true,
        )
        val prepared = samplePeer("peer-archived", "归档用户")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.searchArchive("归档", 100) } returns AppResult.Success(listOf(archived))
        coEvery { repository.prepareArchivedConversation(archived) } returns AppResult.Success(prepared)
        val opened = mutableListOf<Pair<String, String>>()

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.openArchiveSearch()
        viewModel.updateArchiveSearchKeyword("归档")
        viewModel.searchArchive()
        advanceUntilIdle()

        assertEquals(true, viewModel.uiState.archiveSearchVisible)
        assertEquals(false, viewModel.uiState.archiveSearchLoading)
        assertEquals(true, viewModel.uiState.archiveSearchSearched)
        assertEquals(listOf(archived), viewModel.uiState.archiveSearchItems)

        viewModel.openArchivedChat(archived) { peerId, peerName -> opened += peerId to peerName }
        advanceUntilIdle()

        assertEquals(listOf("peer-archived" to "归档用户"), opened)
        assertEquals(false, viewModel.uiState.archiveSearchVisible)
    }

    @Test
    fun `cross identity picker should load sources candidates and open selected chat`() = runTest(mainDispatcherRule.dispatcher) {
        val source = IdentityDto("source-1", "来源身份", "女")
        val candidate = ContactCandidateDto(
            targetUserId = "peer-cross",
            nickname = "跨身份用户",
            sources = listOf("history"),
        )
        val prepared = samplePeer("peer-cross", "跨身份用户")
        every { repository.observeConversations(ConversationTab.HISTORY) } returns historyFlow
        every { repository.observeConversations(ConversationTab.FAVORITE) } returns favoriteFlow
        every { webSocketClient.events } returns events
        coEvery { repository.loadHistory() } returns AppResult.Success(Unit)
        coEvery { repository.loadFavorite() } returns AppResult.Success(Unit)
        coEvery { repository.loadSourceIdentities() } returns AppResult.Success(listOf(source))
        coEvery { repository.loadContactCandidates(sourceIdentity = source, keyword = "", limit = 300) } returns AppResult.Success(listOf(candidate))
        coEvery { repository.prepareContactCandidate(candidate) } returns AppResult.Success(prepared)
        val opened = mutableListOf<Pair<String, String>>()

        val viewModel = ChatListViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.openCrossIdentityPicker()
        advanceUntilIdle()

        assertEquals(true, viewModel.uiState.crossIdentityVisible)
        assertEquals("source-1", viewModel.uiState.crossIdentitySelectedSourceId)
        assertEquals(listOf(candidate), viewModel.uiState.crossIdentityCandidates)

        viewModel.openCrossIdentityChat(candidate) { peerId, peerName -> opened += peerId to peerName }
        advanceUntilIdle()

        assertEquals(listOf("peer-cross" to "跨身份用户"), opened)
        assertEquals(false, viewModel.uiState.crossIdentityVisible)
    }

    private fun connectNotice(message: String): LiaoWsEvent.ConnectNotice = LiaoWsEvent.ConnectNotice(
        raw = "notice",
        envelope = sampleEnvelope(LiaoWsKnownCode.ConnectNotice),
        message = message,
    )

    private fun matchCancelled(message: String): LiaoWsEvent.MatchCancelled = LiaoWsEvent.MatchCancelled(
        raw = "cancel",
        envelope = sampleEnvelope(LiaoWsKnownCode.MatchCancel),
        message = message,
    )

    private fun sampleEnvelope(code: LiaoWsKnownCode): LiaoWsEnvelope = LiaoWsEnvelope(
        raw = code.name,
        code = code.wireValue,
        knownCode = code,
        act = null,
        knownAct = null,
        forceout = false,
        content = "",
        fromUser = null,
        toUser = null,
        time = "",
        tid = "",
    )

    private fun samplePeer(id: String, name: String, unreadCount: Int = 1): ChatPeer = ChatPeer(
        id = id,
        name = name,
        sex = "女",
        ip = "1.1.1.1",
        address = "深圳",
        isFavorite = id == "peer-2",
        lastMessage = "hello",
        lastTime = "20:00",
        unreadCount = unreadCount,
    )
}
