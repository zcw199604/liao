package io.github.a7413498.liao.android.app

import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.database.MessageEntity
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.LiaoWsEnvelope
import io.github.a7413498.liao.android.core.websocket.LiaoWsEvent
import io.github.a7413498.liao.android.core.websocket.MatchCandidate
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import io.mockk.verify
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class AppCoordinatorViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val webSocketClient = mockk<LiaoWebSocketClient>(relaxUnitFun = true)
    private val conversationDao = mockk<ConversationDao>(relaxUnitFun = true)
    private val messageDao = mockk<MessageDao>(relaxUnitFun = true)
    private val authTokenFlow = MutableStateFlow<String?>(null)
    private val currentSessionFlow = MutableStateFlow<CurrentIdentitySession?>(null)
    private val events = MutableSharedFlow<LiaoWsEvent>(extraBufferCapacity = 16)

    init {
        every { preferencesStore.authTokenFlow } returns authTokenFlow
        every { preferencesStore.currentSessionFlow } returns currentSessionFlow
        every { webSocketClient.events } returns events
        every { webSocketClient.connect(any(), any()) } just runs
        every { webSocketClient.disconnect(manual = true) } just runs
    }

    @Test
    fun `init should switch launch route from login to identity and chat list`() = runTest(mainDispatcherRule.dispatcher) {
        val session = sampleSession()

        val viewModel = AppCoordinatorViewModel(
            preferencesStore = preferencesStore,
            webSocketClient = webSocketClient,
            conversationDao = conversationDao,
            messageDao = messageDao,
        )
        advanceUntilIdle()

        assertEquals(true, viewModel.uiState.value.launchResolved)
        assertEquals(LiaoRoute.LOGIN, viewModel.uiState.value.launchRoute)

        authTokenFlow.value = "jwt-token"
        advanceUntilIdle()
        assertEquals(LiaoRoute.IDENTITY, viewModel.uiState.value.launchRoute)

        currentSessionFlow.value = session
        advanceUntilIdle()
        assertEquals(LiaoRoute.CHAT_LIST, viewModel.uiState.value.launchRoute)

        verify { webSocketClient.connect(token = "jwt-token", session = session) }
        verify(atLeast = 2) { webSocketClient.disconnect(manual = true) }
        coVerify(atLeast = 2) { conversationDao.clearAll() }
        coVerify(atLeast = 2) { messageDao.clearAll() }
    }

    @Test
    fun `match success should reuse cached conversation fields and mark active peer as read`() = runTest(mainDispatcherRule.dispatcher) {
        authTokenFlow.value = "jwt-token"
        currentSessionFlow.value = sampleSession()
        val cachedConversation = ConversationEntity(
            id = "peer-1",
            name = "",
            sex = "保密",
            ip = "1.1.1.1",
            address = "深圳",
            isFavorite = true,
            lastMessage = "旧消息",
            lastTime = "昨天",
            unreadCount = 5,
        )
        coEvery { conversationDao.getById("peer-1") } returns cachedConversation
        val upserted = slot<ConversationEntity>()
        coEvery { conversationDao.upsert(capture(upserted)) } just runs

        val viewModel = AppCoordinatorViewModel(
            preferencesStore = preferencesStore,
            webSocketClient = webSocketClient,
            conversationDao = conversationDao,
            messageDao = messageDao,
        )
        advanceUntilIdle()

        viewModel.setActiveChatPeer("peer-1")
        advanceUntilIdle()
        events.tryEmit(
            LiaoWsEvent.MatchSuccess(
                raw = "match-success",
                candidate = MatchCandidate(
                    id = "peer-1",
                    name = "",
                    sex = "",
                    age = "18",
                    address = "",
                ),
            ),
        )
        advanceUntilIdle()

        assertEquals("peer-1", upserted.captured.name)
        assertEquals("保密", upserted.captured.sex)
        assertEquals("1.1.1.1", upserted.captured.ip)
        assertEquals("深圳", upserted.captured.address)
        assertEquals(true, upserted.captured.isFavorite)
        assertEquals("匹配成功", upserted.captured.lastMessage)
        assertEquals("刚刚", upserted.captured.lastTime)
        assertEquals(0, upserted.captured.unreadCount)
        coVerify(atLeast = 2) { conversationDao.markAsRead("peer-1") }
    }

    @Test
    fun `chat message should ignore blank peer and persist inbound self plus fallback branches`() = runTest(mainDispatcherRule.dispatcher) {
        authTokenFlow.value = "jwt-token"
        currentSessionFlow.value = sampleSession()
        val inboundCurrent = ConversationEntity(
            id = "peer-inbound",
            name = "旧名字",
            sex = "女",
            ip = "2.2.2.2",
            address = "杭州",
            isFavorite = false,
            lastMessage = "old",
            lastTime = "old",
            unreadCount = 2,
        )
        val selfCurrent = ConversationEntity(
            id = "peer-self",
            name = "已保存好友",
            sex = "男",
            ip = "3.3.3.3",
            address = "上海",
            isFavorite = true,
            lastMessage = "old",
            lastTime = "old",
            unreadCount = 7,
        )
        coEvery { conversationDao.getById(any()) } answers {
            when (firstArg<String>()) {
                "peer-inbound" -> inboundCurrent
                "peer-self" -> selfCurrent
                else -> null
            }
        }
        val upsertedConversations = mutableListOf<ConversationEntity>()
        val upsertedMessages = mutableListOf<MessageEntity>()
        coEvery { conversationDao.upsert(capture(upsertedConversations)) } just runs
        coEvery { messageDao.upsert(capture(upsertedMessages)) } just runs

        val viewModel = AppCoordinatorViewModel(
            preferencesStore = preferencesStore,
            webSocketClient = webSocketClient,
            conversationDao = conversationDao,
            messageDao = messageDao,
        )
        advanceUntilIdle()

        viewModel.setActiveChatPeer("peer-inbound")
        advanceUntilIdle()

        events.tryEmit(LiaoWsEvent.ChatMessage(raw = "blank", envelope = sampleEnvelope(), timelineMessage = sampleIncomingMessage(fromUserId = "", fromUserName = "", content = "空", toUserId = "session-1")))
        events.tryEmit(LiaoWsEvent.ChatMessage(raw = "inbound", envelope = sampleEnvelope(), timelineMessage = sampleIncomingMessage(fromUserId = "peer-inbound", fromUserName = "Alice", content = "你好", toUserId = "session-1")))
        events.tryEmit(LiaoWsEvent.ChatMessage(raw = "self", envelope = sampleEnvelope(), timelineMessage = sampleSelfMessage(toUserId = "peer-self", content = "[photo.jpg]", type = ChatMessageType.IMAGE, fileName = "photo.jpg")))
        events.tryEmit(LiaoWsEvent.ChatMessage(raw = "fallback", envelope = sampleEnvelope(), timelineMessage = sampleIncomingMessage(fromUserId = "fallback-peer", fromUserName = "", content = "来自陌生人", toUserId = "session-1")))
        advanceUntilIdle()

        assertEquals(3, upsertedMessages.size)
        assertEquals(3, upsertedConversations.size)
        assertEquals("peer-inbound", upsertedMessages[0].peerId)
        assertEquals("Alice", upsertedConversations[0].name)
        assertEquals(0, upsertedConversations[0].unreadCount)
        assertEquals("已保存好友", upsertedConversations[1].name)
        assertEquals(true, upsertedConversations[1].isFavorite)
        assertEquals("我: [图片] photo.jpg", upsertedConversations[1].lastMessage)
        assertEquals(0, upsertedConversations[1].unreadCount)
        assertEquals("fallback", upsertedConversations[2].name)
        assertEquals(1, upsertedConversations[2].unreadCount)
        coVerify(atLeast = 2) { conversationDao.markAsRead("peer-inbound") }
    }

    @Test
    fun `forceout should clear auth cache and expose consumable session expired message`() = runTest(mainDispatcherRule.dispatcher) {
        authTokenFlow.value = "jwt-token"
        currentSessionFlow.value = sampleSession()

        val viewModel = AppCoordinatorViewModel(
            preferencesStore = preferencesStore,
            webSocketClient = webSocketClient,
            conversationDao = conversationDao,
            messageDao = messageDao,
        )
        advanceUntilIdle()

        events.tryEmit(
            LiaoWsEvent.Forceout(
                raw = "forceout",
                envelope = sampleEnvelope(forceout = true),
                forbiddenUntilMillis = 123456L,
                reason = "登录已过期",
            ),
        )
        advanceUntilIdle()

        assertEquals("登录已过期", viewModel.uiState.value.sessionExpiredMessage)
        coVerify { preferencesStore.clearAuthToken() }
        coVerify { preferencesStore.clearCurrentSession() }
        coVerify(atLeast = 1) { conversationDao.clearAll() }
        coVerify(atLeast = 1) { messageDao.clearAll() }

        viewModel.consumeSessionExpiredMessage()
        assertNull(viewModel.uiState.value.sessionExpiredMessage)
    }

    private fun sampleSession(): CurrentIdentitySession = CurrentIdentitySession(
        id = "session-1",
        name = "Alice",
        sex = "女",
        cookie = "cookie",
        ip = "1.1.1.1",
        area = "深圳",
    )

    private fun sampleIncomingMessage(
        fromUserId: String,
        fromUserName: String,
        content: String,
        toUserId: String,
    ): ChatTimelineMessage = ChatTimelineMessage(
        id = "msg-$fromUserId-$content",
        fromUserId = fromUserId,
        fromUserName = fromUserName,
        toUserId = toUserId,
        content = content,
        time = "20:00",
        isSelf = false,
    )

    private fun sampleSelfMessage(
        toUserId: String,
        content: String,
        type: ChatMessageType,
        fileName: String,
    ): ChatTimelineMessage = ChatTimelineMessage(
        id = "self-$toUserId-$content",
        fromUserId = "session-1",
        fromUserName = "Alice",
        toUserId = toUserId,
        content = content,
        time = "20:01",
        isSelf = true,
        type = type,
        fileName = fileName,
    )

    private fun sampleEnvelope(forceout: Boolean = false): LiaoWsEnvelope = LiaoWsEnvelope(
        raw = "raw",
        code = null,
        knownCode = null,
        act = null,
        knownAct = null,
        forceout = forceout,
        content = "",
        fromUser = null,
        toUser = null,
        time = "",
        tid = "",
    )
}
