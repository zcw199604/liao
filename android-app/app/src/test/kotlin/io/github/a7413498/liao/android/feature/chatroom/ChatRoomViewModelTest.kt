package io.github.a7413498.liao.android.feature.chatroom

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.LiaoLogger
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.OutgoingMessageStatus
import io.github.a7413498.liao.android.core.websocket.LiaoWsEnvelope
import io.github.a7413498.liao.android.core.websocket.LiaoWsEvent
import io.github.a7413498.liao.android.core.websocket.MatchCandidate
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkObject
import io.mockk.unmockkObject
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.advanceTimeBy
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
class ChatRoomViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun `bind should load history refresh favorite observe connection and avoid duplicate peer reload`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        coEvery { fixture.repository.ensureConnected() } returnsMany listOf(
            AppResult.Success(WebSocketState.Idle),
            AppResult.Success(WebSocketState.Idle),
        )
        coEvery { fixture.repository.getFavoriteState("peer-1") } returns true
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(
                messages = listOf(
                    timelineMessage(id = "10", fromUserId = "peer-1", isSelf = false, content = "hi"),
                    timelineMessage(id = "7", fromUserId = "peer-1", isSelf = false, content = "again"),
                ),
                historyCursor = "999",
                hasMoreHistory = true,
            ),
        )
        coEvery { fixture.repository.requestUserInfo("peer-1") } returns AppResult.Success(Unit)

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()

        assertFalse(fixture.viewModel.uiState.loading)
        assertEquals(true, fixture.viewModel.uiState.peerFavorite)
        assertEquals(listOf("10", "7"), fixture.viewModel.uiState.messages.map { it.id })
        assertEquals("7", fixture.viewModel.uiState.historyCursor)
        assertEquals(true, fixture.viewModel.uiState.hasMoreHistory)
        assertEquals(1, fixture.viewModel.uiState.scrollToBottomVersion)
        assertEquals("Idle", fixture.viewModel.uiState.connectionStateLabel)

        fixture.connectionFlow.value = WebSocketState.Connected
        advanceUntilIdle()
        assertEquals("Connected", fixture.viewModel.uiState.connectionStateLabel)
        assertNull(fixture.viewModel.uiState.message)
        coVerify(exactly = 1) { fixture.repository.requestUserInfo("peer-1") }

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()
        assertEquals(listOf("10", "7"), fixture.viewModel.uiState.messages.map { it.id })
        coVerify(exactly = 2) { fixture.repository.ensureConnected() }
        coVerify(exactly = 1) { fixture.repository.getFavoriteState("peer-1") }
        coVerify(exactly = 1) { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") }
    }

    @Test
    fun `load more history should handle error and concurrent guard`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        val loadMoreDeferred = CompletableDeferred<AppResult<HistoryPageResult>>()
        coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { fixture.repository.getFavoriteState("peer-1") } returns false
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(
                messages = listOf(
                    timelineMessage(id = "10", fromUserId = "peer-1", isSelf = false),
                    timelineMessage(id = "7", fromUserId = "peer-1", isSelf = false),
                ),
                historyCursor = "999",
                hasMoreHistory = true,
            ),
        )
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = false, firstTid = "7") } coAnswers { loadMoreDeferred.await() }

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()

        fixture.viewModel.loadMoreHistory("peer-1")
        runCurrent()
        assertTrue(fixture.viewModel.uiState.loadingMore)

        fixture.viewModel.loadMoreHistory("peer-1")
        runCurrent()
        coVerify(exactly = 1) { fixture.repository.loadHistory(peerId = "peer-1", isFirst = false, firstTid = "7") }

        loadMoreDeferred.complete(AppResult.Error("加载更多失败"))
        advanceUntilIdle()
        assertFalse(fixture.viewModel.uiState.loadingMore)
        assertEquals("加载更多失败", fixture.viewModel.uiState.message)
    }

    fun `load more history should merge older pages and stop when cursor missing`() = runTest(mainDispatcherRule.dispatcher) {
        val successFixture = newFixture()
        coEvery { successFixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { successFixture.repository.getFavoriteState("peer-1") } returns false
        coEvery { successFixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(
                messages = listOf(
                    timelineMessage(id = "10", fromUserId = "peer-1", isSelf = false),
                    timelineMessage(id = "7", fromUserId = "peer-1", isSelf = false),
                ),
                historyCursor = "999",
                hasMoreHistory = true,
            ),
        )
        coEvery { successFixture.repository.loadHistory(peerId = "peer-1", isFirst = false, firstTid = "7") } returns AppResult.Success(
            HistoryPageResult(
                messages = listOf(
                    timelineMessage(id = "5", fromUserId = "peer-1", isSelf = false),
                    timelineMessage(id = "3", fromUserId = "peer-1", isSelf = false),
                ),
                historyCursor = "2",
                hasMoreHistory = true,
            ),
        )

        successFixture.viewModel.bind("peer-1")
        advanceUntilIdle()
        successFixture.viewModel.loadMoreHistory("peer-1")
        advanceUntilIdle()

        assertFalse(successFixture.viewModel.uiState.loadingMore)
        assertEquals(listOf("5", "3", "10", "7"), successFixture.viewModel.uiState.messages.map { it.id })
        assertEquals("3", successFixture.viewModel.uiState.historyCursor)
        assertEquals(true, successFixture.viewModel.uiState.hasMoreHistory)

        val guardFixture = newFixture()
        coEvery { guardFixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { guardFixture.repository.getFavoriteState("peer-2") } returns false
        coEvery { guardFixture.repository.loadHistory(peerId = "peer-2", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(
                messages = emptyList(),
                historyCursor = null,
                hasMoreHistory = false,
            ),
        )

        guardFixture.viewModel.bind("peer-2")
        advanceUntilIdle()
        guardFixture.viewModel.loadMoreHistory("peer-2")
        advanceUntilIdle()

        assertFalse(guardFixture.viewModel.uiState.hasMoreHistory)
        coVerify(exactly = 0) { guardFixture.repository.loadHistory(peerId = "peer-2", isFirst = false, firstTid = any()) }
    }

    @Test
    fun `media sheet and external import should load history close panel guard duplicate upload and surface error`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        val importDeferred = CompletableDeferred<AppResult<ChatMediaAsset>>()
        coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { fixture.repository.getFavoriteState("peer-1") } returns false
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(emptyList(), historyCursor = null, hasMoreHistory = false),
        )
        coEvery { fixture.repository.loadChatHistoryMedia("peer-1") } returns AppResult.Success(
            listOf(mediaAsset(id = "history-1", remotePath = "/upload/history.jpg", localFilename = "history.jpg")),
        )
        coEvery { fixture.repository.reuploadLocalMedia(localPath = "/mt/a.jpg", localFilename = "a.jpg") } coAnswers { importDeferred.await() }
        coEvery { fixture.repository.reuploadLocalMedia(localPath = "/dy/b.jpg", localFilename = "b.jpg") } returns AppResult.Error("导入失败")

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()
        fixture.viewModel.openMediaSheet("peer-1")
        advanceUntilIdle()

        assertTrue(fixture.viewModel.uiState.mediaSheetVisible)
        assertEquals(listOf("history-1"), fixture.viewModel.uiState.historyMedia.map { it.id })

        fixture.viewModel.openMediaSheet("peer-1")
        advanceUntilIdle()
        coVerify(exactly = 1) { fixture.repository.loadChatHistoryMedia("peer-1") }

        fixture.viewModel.closeMediaSheet()
        fixture.viewModel.closeMediaSheet()
        assertFalse(fixture.viewModel.uiState.mediaSheetVisible)

        fixture.viewModel.importMtPhotoMedia("/mt/a.jpg", "a.jpg")
        runCurrent()
        assertTrue(fixture.viewModel.uiState.mediaUploading)

        fixture.viewModel.importDouyinMedia("/dy/b.jpg", "b.jpg")
        runCurrent()
        coVerify(exactly = 0) { fixture.repository.reuploadLocalMedia(localPath = "/dy/b.jpg", localFilename = "b.jpg") }

        importDeferred.complete(
            AppResult.Success(mediaAsset(id = "upload-1", remotePath = "/upload/a.jpg", localFilename = "a.jpg")),
        )
        advanceUntilIdle()
        assertFalse(fixture.viewModel.uiState.mediaUploading)
        assertEquals(listOf("upload-1"), fixture.viewModel.uiState.uploadedMedia.map { it.id })
        assertTrue(fixture.viewModel.uiState.mediaSheetVisible)
        assertEquals("mtPhoto 已导入并加入待发送列表", fixture.viewModel.uiState.message)

        fixture.viewModel.importDouyinMedia("/dy/b.jpg", "b.jpg")
        advanceUntilIdle()
        assertFalse(fixture.viewModel.uiState.mediaUploading)
        assertEquals("导入失败", fixture.viewModel.uiState.message)
    }

    @Test
    fun `open media sheet should guard repeated history loads when loading or cached`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        val mediaDeferred = CompletableDeferred<AppResult<List<ChatMediaAsset>>>()
        coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { fixture.repository.getFavoriteState("peer-1") } returns false
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(emptyList(), historyCursor = null, hasMoreHistory = false),
        )
        coEvery { fixture.repository.loadChatHistoryMedia("peer-1") } coAnswers { mediaDeferred.await() }

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()

        fixture.viewModel.openMediaSheet("peer-1")
        runCurrent()
        assertTrue(fixture.viewModel.uiState.mediaSheetVisible)
        assertTrue(fixture.viewModel.uiState.historyMediaLoading)

        fixture.viewModel.openMediaSheet("peer-1")
        runCurrent()
        coVerify(exactly = 1) { fixture.repository.loadChatHistoryMedia("peer-1") }

        mediaDeferred.complete(AppResult.Success(listOf(mediaAsset(id = "history-1", remotePath = "/upload/history.jpg", localFilename = "history.jpg"))))
        advanceUntilIdle()
        assertFalse(fixture.viewModel.uiState.historyMediaLoading)
        assertEquals(listOf("history-1"), fixture.viewModel.uiState.historyMedia.map { it.id })

        fixture.viewModel.openMediaSheet("peer-1")
        advanceUntilIdle()
        coVerify(exactly = 1) { fixture.repository.loadChatHistoryMedia("peer-1") }
    }

    @Test
    fun `inbound events should cover notices current messages and ignored branches`() = runTest(mainDispatcherRule.dispatcher) {
        mockkObject(LiaoLogger)
        every { LiaoLogger.i(any(), any()) } returns 0
        try {
            val fixture = newFixture()
            coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
            coEvery { fixture.repository.getFavoriteState("peer-1") } returns false
            coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
                HistoryPageResult(emptyList(), historyCursor = null, hasMoreHistory = false),
            )
            coEvery { fixture.repository.hydrateTimelineMessage(match { it.id == "m-current" }) } returns timelineMessage(
                id = "m-current",
                fromUserId = "peer-1",
                toUserId = "self",
                isSelf = false,
                content = "hello",
            )

            fixture.viewModel.bind("peer-1")
            advanceUntilIdle()

            fixture.events.tryEmit(LiaoWsEvent.Typing(raw = "typing-other", peerId = "peer-2", peerName = "Other", typing = true))
            advanceUntilIdle()
            assertFalse(fixture.viewModel.uiState.isPeerTyping)

            fixture.events.tryEmit(LiaoWsEvent.Typing(raw = "typing-current", peerId = "peer-1", peerName = "Bob", typing = true))
            advanceUntilIdle()
            assertTrue(fixture.viewModel.uiState.isPeerTyping)

            fixture.events.tryEmit(LiaoWsEvent.Typing(raw = "typing-stop", peerId = "peer-1", peerName = "Bob", typing = false))
            advanceUntilIdle()
            assertFalse(fixture.viewModel.uiState.isPeerTyping)

            fixture.events.tryEmit(LiaoWsEvent.OnlineStatus(raw = "online", envelope = sampleEnvelope(), message = "在线中"))
            advanceUntilIdle()
            assertEquals("在线中", fixture.viewModel.uiState.peerStatusMessage)
            assertEquals("在线中", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(LiaoWsEvent.ConnectNotice(raw = "notice-blank", envelope = sampleEnvelope(), message = "   "))
            advanceUntilIdle()
            assertEquals("在线中", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(LiaoWsEvent.ConnectNotice(raw = "notice", envelope = sampleEnvelope(), message = "连接成功"))
            advanceUntilIdle()
            assertEquals("连接成功", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(LiaoWsEvent.Notice(raw = "system-blank", envelope = sampleEnvelope(), message = "   "))
            advanceUntilIdle()
            assertEquals("连接成功", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(LiaoWsEvent.Notice(raw = "system", envelope = sampleEnvelope(), message = "系统通知"))
            advanceUntilIdle()
            assertEquals("系统通知", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(LiaoWsEvent.MatchCancelled(raw = "cancel", envelope = sampleEnvelope(), message = "匹配已取消"))
            advanceUntilIdle()
            assertEquals("匹配已取消", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(LiaoWsEvent.MatchSuccess(raw = "match", candidate = sampleCandidate()))
            advanceUntilIdle()
            assertEquals("匹配已取消", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(
                LiaoWsEvent.ChatMessage(
                    raw = "other",
                    envelope = sampleEnvelope(),
                    timelineMessage = timelineMessage(id = "m-other", fromUserId = "peer-2", toUserId = "self", isSelf = false, content = "skip"),
                ),
            )
            advanceUntilIdle()
            assertTrue(fixture.viewModel.uiState.messages.isEmpty())

            fixture.events.tryEmit(
                LiaoWsEvent.ChatMessage(
                    raw = "current",
                    envelope = sampleEnvelope(),
                    timelineMessage = timelineMessage(id = "m-current", fromUserId = "peer-1", toUserId = "self", isSelf = false, content = "hello"),
                ),
            )
            advanceUntilIdle()
            assertEquals(listOf("m-current"), fixture.viewModel.uiState.messages.map { it.id })
            assertEquals(1, fixture.viewModel.uiState.scrollToBottomVersion)

            fixture.events.tryEmit(LiaoWsEvent.Unknown(raw = "unknown", envelope = null))
            advanceUntilIdle()
            assertEquals("匹配已取消", fixture.viewModel.uiState.message)

            fixture.events.tryEmit(LiaoWsEvent.Forceout(raw = "forceout", envelope = sampleEnvelope(), forbiddenUntilMillis = 0L, reason = "已禁连"))
            advanceUntilIdle()
            assertEquals("已禁连", fixture.viewModel.uiState.message)
        } finally {
            unmockkObject(LiaoLogger)
        }
    }

    @Test
    fun `send text should ignore blank draft and mark pending message failed after timeout`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        coEvery { fixture.repository.sendText("peer-1", "Bob", "hello", any()) } coAnswers {
            val clientId = arg<String>(3)
            AppResult.Success(
                timelineMessage(
                    id = clientId,
                    clientId = clientId,
                    fromUserId = "self",
                    toUserId = "peer-1",
                    isSelf = true,
                    content = "hello",
                    sendStatus = OutgoingMessageStatus.SENDING,
                ),
            )
        }

        fixture.viewModel.updateDraft("   ")
        fixture.viewModel.sendText("peer-1", "Bob")
        advanceUntilIdle()
        coVerify(exactly = 0) { fixture.repository.sendText(any(), any(), any(), any()) }

        fixture.viewModel.updateDraft("hello")
        fixture.viewModel.sendText("peer-1", "Bob")
        runCurrent()

        assertEquals("", fixture.viewModel.uiState.draft)
        assertEquals(1, fixture.viewModel.uiState.messages.size)
        assertEquals(OutgoingMessageStatus.SENDING, fixture.viewModel.uiState.messages.single().sendStatus)
        assertEquals(1, fixture.viewModel.uiState.scrollToBottomVersion)

        advanceTimeBy(10_000)
        advanceUntilIdle()
        assertEquals(OutgoingMessageStatus.FAILED, fixture.viewModel.uiState.messages.single().sendStatus)
        assertEquals("发送超时，请重试", fixture.viewModel.uiState.messages.single().sendError)
    }

    @Test
    fun `send text failure and retry should rollback failed message and consume banner`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        coEvery { fixture.repository.sendText("peer-1", "Bob", "hello", any()) } returnsMany listOf(
            AppResult.Error("首次失败"),
            AppResult.Error("WebSocket 未连接"),
        )

        fixture.viewModel.updateDraft("hello")
        fixture.viewModel.sendText("peer-1", "Bob")
        advanceUntilIdle()

        val failedClientId = fixture.viewModel.uiState.messages.single().clientId
        assertEquals(OutgoingMessageStatus.FAILED, fixture.viewModel.uiState.messages.single().sendStatus)
        assertEquals("首次失败", fixture.viewModel.uiState.message)

        fixture.viewModel.consumeMessage()
        assertNull(fixture.viewModel.uiState.message)

        fixture.viewModel.retryFailedMessage("peer-1", "Bob", "missing")
        advanceUntilIdle()
        coVerify(exactly = 1) { fixture.repository.sendText("peer-1", "Bob", "hello", any()) }

        fixture.viewModel.retryFailedMessage("peer-1", "Bob", failedClientId)
        advanceUntilIdle()
        assertEquals(OutgoingMessageStatus.FAILED, fixture.viewModel.uiState.messages.single().sendStatus)
        assertEquals("WebSocket 未连接", fixture.viewModel.uiState.messages.single().sendError)
        assertEquals("WebSocket 未连接", fixture.viewModel.uiState.message)
        coVerify(exactly = 2) { fixture.repository.sendText("peer-1", "Bob", "hello", any()) }
    }

    @Test
    fun `blank peer bind should ignore typing and online status events`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { fixture.repository.getFavoriteState("") } returns false
        coEvery { fixture.repository.loadHistory(peerId = "", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(emptyList(), historyCursor = null, hasMoreHistory = false),
        )

        fixture.viewModel.bind("")
        advanceUntilIdle()
        fixture.events.tryEmit(LiaoWsEvent.Typing(raw = "typing", peerId = "", peerName = "", typing = true))
        fixture.events.tryEmit(LiaoWsEvent.OnlineStatus(raw = "online", envelope = sampleEnvelope(), message = "不会显示"))
        advanceUntilIdle()

        assertFalse(fixture.viewModel.uiState.isPeerTyping)
        assertEquals("", fixture.viewModel.uiState.peerStatusMessage)
        assertNull(fixture.viewModel.uiState.message)
    }

    @Test
    fun `retry failed message should support success path with fallback client id`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { fixture.repository.getFavoriteState("peer-1") } returns false
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(
                messages = listOf(
                    timelineMessage(
                        id = "temp-1",
                        clientId = "",
                        fromUserId = "self",
                        toUserId = "peer-1",
                        isSelf = true,
                        content = "hello",
                        sendStatus = OutgoingMessageStatus.FAILED,
                        sendError = "old",
                    ),
                    timelineMessage(id = "stable", clientId = "stable", fromUserId = "peer-1", isSelf = false, content = "keep"),
                ),
                historyCursor = "temp-1",
                hasMoreHistory = true,
            ),
        )
        coEvery { fixture.repository.sendText("peer-1", "Bob", "hello", "temp-1") } returns AppResult.Success(
            timelineMessage(
                id = "temp-1",
                clientId = "",
                fromUserId = "self",
                toUserId = "peer-1",
                isSelf = true,
                content = "hello",
                sendStatus = OutgoingMessageStatus.SENT,
            ),
        )

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()
        fixture.viewModel.retryFailedMessage("peer-1", "Bob", "temp-1")
        advanceUntilIdle()

        assertEquals(listOf("temp-1", "stable"), fixture.viewModel.uiState.messages.map { it.id })
        assertEquals(OutgoingMessageStatus.SENT, fixture.viewModel.uiState.messages.first().sendStatus)
        assertNull(fixture.viewModel.uiState.messages.first().sendError)
    }

    @Test
    fun `consume message should ignore null state`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        fixture.viewModel.consumeMessage()
        assertNull(fixture.viewModel.uiState.message)
    }

    @Test
    fun `self echo should merge optimistic message without duplication`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { fixture.repository.getFavoriteState("peer-1") } returns false
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(emptyList(), historyCursor = null, hasMoreHistory = false),
        )
        coEvery { fixture.repository.sendText("peer-1", "Bob", "hello", any()) } coAnswers {
            val clientId = arg<String>(3)
            AppResult.Success(
                timelineMessage(
                    id = clientId,
                    clientId = clientId,
                    fromUserId = "self",
                    toUserId = "peer-1",
                    isSelf = true,
                    content = "hello",
                    sendStatus = OutgoingMessageStatus.SENDING,
                ),
            )
        }
        coEvery { fixture.repository.hydrateTimelineMessage(match { it.id == "remote-1" }) } returns timelineMessage(
            id = "remote-1",
            clientId = "",
            fromUserId = "self",
            toUserId = "peer-1",
            isSelf = true,
            content = "hello",
            sendStatus = OutgoingMessageStatus.SENT,
            sendError = "old",
        )

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()
        fixture.viewModel.updateDraft("hello")
        fixture.viewModel.sendText("peer-1", "Bob")
        runCurrent()

        val optimisticClientId = fixture.viewModel.uiState.messages.single().clientId
        fixture.events.tryEmit(
            LiaoWsEvent.ChatMessage(
                raw = "echo",
                envelope = sampleEnvelope(),
                timelineMessage = timelineMessage(
                    id = "remote-1",
                    clientId = "",
                    fromUserId = "self",
                    toUserId = "peer-1",
                    isSelf = true,
                    content = "hello",
                    sendStatus = OutgoingMessageStatus.SENT,
                ),
            ),
        )
        advanceUntilIdle()

        assertEquals(1, fixture.viewModel.uiState.messages.size)
        assertEquals(optimisticClientId, fixture.viewModel.uiState.messages.single().clientId)
        assertEquals(OutgoingMessageStatus.SENT, fixture.viewModel.uiState.messages.single().sendStatus)
        assertNull(fixture.viewModel.uiState.messages.single().sendError)
        assertEquals(2, fixture.viewModel.uiState.scrollToBottomVersion)
    }

    @Test
    fun `clear and reload should show loading banner and report empty history`() = runTest(mainDispatcherRule.dispatcher) {
        val fixture = newFixture()
        val reloadDeferred = CompletableDeferred<AppResult<HistoryPageResult>>()
        coEvery { fixture.repository.ensureConnected() } returns AppResult.Success(WebSocketState.Idle)
        coEvery { fixture.repository.getFavoriteState("peer-1") } returns false
        coEvery { fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0") } returns AppResult.Success(
            HistoryPageResult(
                messages = listOf(timelineMessage(id = "10", fromUserId = "peer-1", isSelf = false)),
                historyCursor = "10",
                hasMoreHistory = true,
            ),
        )
        coEvery { fixture.repository.clearAndReload("peer-1") } coAnswers { reloadDeferred.await() }

        fixture.viewModel.bind("peer-1")
        advanceUntilIdle()
        fixture.viewModel.clearAndReload("peer-1")
        runCurrent()

        assertTrue(fixture.viewModel.uiState.loading)
        assertTrue(fixture.viewModel.uiState.messages.isEmpty())
        assertEquals("正在重新加载聊天记录...", fixture.viewModel.uiState.message)

        reloadDeferred.complete(AppResult.Success(HistoryPageResult(emptyList(), historyCursor = null, hasMoreHistory = false)))
        advanceUntilIdle()

        assertFalse(fixture.viewModel.uiState.loading)
        assertFalse(fixture.viewModel.uiState.loadingMore)
        assertTrue(fixture.viewModel.uiState.messages.isEmpty())
        assertNull(fixture.viewModel.uiState.historyCursor)
        assertFalse(fixture.viewModel.uiState.hasMoreHistory)
        assertEquals("暂无聊天记录", fixture.viewModel.uiState.message)
    }

    private fun newFixture(): Fixture {
        val repository = mockk<ChatRoomRepository>()
        val connectionFlow = MutableStateFlow<WebSocketState>(WebSocketState.Idle)
        val events = MutableSharedFlow<LiaoWsEvent>(extraBufferCapacity = 16)
        every { repository.connectionState() } returns connectionFlow
        every { repository.inboundEvents() } returns events
        coEvery { repository.hydrateTimelineMessage(any()) } coAnswers { firstArg() }
        return Fixture(
            repository = repository,
            connectionFlow = connectionFlow,
            events = events,
            viewModel = ChatRoomViewModel(repository),
        )
    }

    private fun timelineMessage(
        id: String = "1",
        clientId: String = "",
        fromUserId: String = "self",
        toUserId: String = "peer-1",
        isSelf: Boolean = false,
        content: String = "content",
        type: ChatMessageType = ChatMessageType.TEXT,
        mediaUrl: String = "",
        fileName: String = "",
        sendStatus: OutgoingMessageStatus = OutgoingMessageStatus.SENT,
        sendError: String? = null,
    ): ChatTimelineMessage = ChatTimelineMessage(
        id = id,
        fromUserId = fromUserId,
        fromUserName = if (fromUserId == "self") "我" else "Bob",
        toUserId = toUserId,
        content = content,
        time = "10:00:00",
        isSelf = isSelf,
        type = type,
        mediaUrl = mediaUrl,
        fileName = fileName,
        clientId = clientId,
        sendStatus = sendStatus,
        sendError = sendError,
    )

    private fun mediaAsset(
        id: String,
        remotePath: String,
        localFilename: String,
        type: ChatMessageType = ChatMessageType.IMAGE,
        url: String = "https://cdn.test/$id",
    ) = ChatMediaAsset(
        id = id,
        url = url,
        type = type,
        localFilename = localFilename,
        remotePath = remotePath,
    )

    private fun sampleCandidate(): MatchCandidate = MatchCandidate(
        id = "peer-1",
        name = "Bob",
        sex = "男",
        age = "20",
        address = "Shenzhen",
    )

    private fun sampleEnvelope(): LiaoWsEnvelope = LiaoWsEnvelope(
        raw = "{}",
        code = null,
        knownCode = null,
        act = null,
        knownAct = null,
        forceout = false,
        content = "",
        fromUser = null,
        toUser = null,
        time = "",
        tid = "",
    )

    private data class Fixture(
        val repository: ChatRoomRepository,
        val connectionFlow: MutableStateFlow<WebSocketState>,
        val events: MutableSharedFlow<LiaoWsEvent>,
        val viewModel: ChatRoomViewModel,
    )
}
