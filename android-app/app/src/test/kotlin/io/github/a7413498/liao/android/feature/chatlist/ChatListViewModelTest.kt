package io.github.a7413498.liao.android.feature.chatlist

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatPeer
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

    private fun samplePeer(id: String, name: String): ChatPeer = ChatPeer(
        id = id,
        name = name,
        sex = "女",
        ip = "1.1.1.1",
        address = "深圳",
        isFavorite = id == "peer-2",
        lastMessage = "hello",
        lastTime = "20:00",
        unreadCount = 1,
    )
}
