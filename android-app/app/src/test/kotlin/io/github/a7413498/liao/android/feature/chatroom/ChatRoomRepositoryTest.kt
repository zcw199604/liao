package io.github.a7413498.liao.android.feature.chatroom

import android.content.ContentResolver
import android.content.Context
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.OutgoingMessageStatus
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.database.MessageEntity
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.MediaApiService
import io.github.a7413498.liao.android.core.network.SystemApiService
import io.github.a7413498.liao.android.core.network.SystemConfigDto
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import io.mockk.verify
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ChatRoomRepositoryTest {
    @Test
    fun `ensure connected should connect websocket report referrer and expose current state`() = runTest {
        val fixture = newFixture()
        val state = MutableStateFlow<WebSocketState>(WebSocketState.Connected)
        every { fixture.webSocketClient.state } returns state
        coEvery { fixture.preferencesStore.readAuthToken() } returns "jwt-token"
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        every { fixture.webSocketClient.connect(token = "jwt-token", session = fixture.session) } just runs
        coEvery {
            fixture.chatApiService.reportReferrer(
                referrerUrl = "",
                currUrl = "android://chatroom",
                userId = fixture.session.id,
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns ApiEnvelope(code = 0, data = Unit)

        val result = fixture.repository.ensureConnected()

        assertTrue(result is AppResult.Success)
        assertEquals(WebSocketState.Connected, (result as AppResult.Success).data)
        verify { fixture.webSocketClient.connect(token = "jwt-token", session = fixture.session) }
        coVerify { fixture.chatApiService.reportReferrer(any(), any(), any(), any(), any(), any()) }
    }

    @Test
    fun `ensure connected should fail when token blank`() = runTest {
        val fixture = newFixture()
        every { fixture.webSocketClient.state } returns MutableStateFlow(WebSocketState.Idle)
        coEvery { fixture.preferencesStore.readAuthToken() } returns "   "
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session

        val result = fixture.repository.ensureConnected()

        assertTrue(result is AppResult.Error)
        assertEquals("请先登录", (result as AppResult.Error).message)
        verify(exactly = 0) { fixture.webSocketClient.connect(any(), any()) }
    }

    @Test
    fun `current api origin and parse helpers should normalize host and poster path`() = runTest {
        val fixture = newFixture()
        every { fixture.baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()

        assertEquals("https://demo.test:8443", fixture.repository.currentApiOrigin())
        assertEquals("img.demo.test", fixture.repository.parseImageServerHost("{\"msg\":{\"server\":\"https://img.demo.test/path\"}}"))
        assertEquals(
            "https://demo.test:8443/upload/videos/demo.poster.jpg",
            fixture.repository.buildPosterUrlFromLocalPath("/videos/demo.mp4?download=1#hash"),
        )
        assertEquals("", fixture.repository.buildPosterUrlFromLocalPath("/images/demo.jpg"))
    }

    @Test
    fun `resolve media url should cover absolute upload relative and remote image server branches`() = runTest {
        val fixture = newFixture()
        every { fixture.baseUrlProvider.currentApiBaseUrl() } returns "http://demo.test/api/".toHttpUrl()
        coEvery { fixture.mediaApiService.getImgServer() } returns "{\"server\":\"img.demo.test\"}".toResponseBody("application/json".toMediaType())
        coEvery { fixture.systemApiService.getSystemConfig() } returns ApiEnvelope(data = SystemConfigDto(imagePortMode = "dynamic", imagePortFixed = "9100"))
        coEvery { fixture.preferencesStore.saveCachedSystemConfig(any()) } just runs
        coEvery { fixture.systemApiService.resolveImagePort(any()) } returns ApiEnvelope(
            data = buildJsonObject { put("port", JsonPrimitive("9200")) }
        )

        assertEquals("", fixture.repository.resolveMediaUrl("   "))
        assertEquals("https://cdn.test/a.jpg", fixture.repository.resolveMediaUrl("https://cdn.test/a.jpg"))
        assertEquals("http://demo.test/upload/a.jpg", fixture.repository.resolveMediaUrl("/upload/a.jpg"))
        assertEquals("http://demo.test/upload/images/a.jpg", fixture.repository.resolveMediaUrl("/images/a.jpg"))
        assertEquals("http://img.demo.test:9200/img/Upload/folder/a.jpg", fixture.repository.resolveMediaUrl("folder/a.jpg"))
        assertEquals("http://img.demo.test:9200/img/Upload/folder/b.jpg", fixture.repository.resolveMediaUrl("folder/b.jpg"))
        coVerify(exactly = 1) { fixture.systemApiService.resolveImagePort(any()) }
    }

    @Test
    fun `infer media type should prefer mime type and fallback to path`() {
        val fixture = newFixture()

        assertEquals(ChatMessageType.IMAGE, fixture.repository.inferMediaTypeFromPath("file.bin", "image/png"))
        assertEquals(ChatMessageType.VIDEO, fixture.repository.inferMediaTypeFromPath("file.bin", "video/mp4"))
        assertEquals(ChatMessageType.FILE, fixture.repository.inferMediaTypeFromPath("file.bin", "application/pdf"))
        assertEquals(ChatMessageType.IMAGE, fixture.repository.inferMediaTypeFromPath("demo.jpg"))
        assertEquals(ChatMessageType.FILE, fixture.repository.inferMediaTypeFromPath("demo.bin"))
    }

    @Test
    fun `hydrate timeline message should return text as is and fallback to raw path on resolve failure`() = runTest {
        val fixture = newFixture()
        val textMessage = timelineMessage(type = ChatMessageType.TEXT, content = "hello")
        val mediaMessage = timelineMessage(type = ChatMessageType.IMAGE, content = "[folder/demo.jpg]", fileName = "")
        coEvery { fixture.mediaApiService.getImgServer() } returns "   ".toResponseBody("text/plain".toMediaType())

        val textResult = fixture.repository.hydrateTimelineMessage(textMessage)
        val mediaResult = fixture.repository.hydrateTimelineMessage(mediaMessage)

        assertEquals(textMessage, textResult)
        assertEquals("folder/demo.jpg", mediaResult.mediaUrl)
        assertEquals("demo.jpg", mediaResult.fileName)
    }

    @Test
    fun `reupload local media should resolve video poster and fallback filename`() = runTest {
        val fixture = newFixture()
        every { fixture.baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test/api/".toHttpUrl()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        coEvery {
            fixture.mediaApiService.reuploadHistoryImage(
                userId = fixture.session.id,
                localPath = "/videos/demo.mp4",
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns buildJsonObject {
            put("state", JsonPrimitive("OK"))
            put("msg", JsonPrimitive("folder/demo.mp4"))
        }
        coEvery { fixture.mediaApiService.getImgServer() } returns "{\"server\":\"img.demo.test\"}".toResponseBody("application/json".toMediaType())
        coEvery { fixture.systemApiService.getSystemConfig() } returns ApiEnvelope(data = SystemConfigDto(imagePortMode = "fixed", imagePortFixed = "9300"))
        coEvery { fixture.preferencesStore.saveCachedSystemConfig(any()) } just runs

        val result = fixture.repository.reuploadLocalMedia(localPath = "/videos/demo.mp4", localFilename = "")

        assertTrue(result is AppResult.Success)
        val asset = (result as AppResult.Success).data
        assertEquals("folder/demo.mp4", asset.remotePath)
        assertEquals(ChatMessageType.VIDEO, asset.type)
        assertEquals("demo.mp4", asset.localFilename)
        assertEquals("https://demo.test/upload/videos/demo.poster.jpg", asset.posterUrl)
        assertEquals("http://img.demo.test:9300/img/Upload/folder/demo.mp4", asset.url)
    }

    @Test
    fun `reupload local media should report api error message`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        coEvery {
            fixture.mediaApiService.reuploadHistoryImage(
                userId = fixture.session.id,
                localPath = "/images/demo.jpg",
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns buildJsonObject {
            put("state", JsonPrimitive("FAIL"))
            put("error", JsonPrimitive("上传失败"))
        }

        val result = fixture.repository.reuploadLocalMedia(localPath = "/images/demo.jpg")

        assertTrue(result is AppResult.Error)
        assertEquals("上传失败", (result as AppResult.Error).message)
    }

    @Test
    fun `load chat history media should parse json array and object payloads`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        coEvery { fixture.mediaApiService.getChatImages(fixture.session.id, "peer-1", 2) } returns JsonArray(
            listOf(JsonPrimitive("https://cdn.test/a.jpg"), JsonPrimitive("/upload/b.mp4"), JsonPrimitive(""))
        )

        val arrayResult = fixture.repository.loadChatHistoryMedia(peerId = "peer-1", limit = 2)

        assertTrue(arrayResult is AppResult.Success)
        assertEquals(listOf(ChatMessageType.IMAGE, ChatMessageType.VIDEO), (arrayResult as AppResult.Success).data.map { it.type })

        coEvery { fixture.mediaApiService.getChatImages(fixture.session.id, "peer-1", 1) } returns buildJsonObject {
            put("data", buildJsonArray { add(JsonPrimitive("https://cdn.test/c.bin")) })
        }

        val objectResult = fixture.repository.loadChatHistoryMedia(peerId = "peer-1", limit = 1)

        assertTrue(objectResult is AppResult.Success)
        assertEquals(ChatMessageType.FILE, (objectResult as AppResult.Success).data.single().type)
    }

    @Test
    fun `send uploaded media should ignore record error and fallback file name`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        every {
            fixture.webSocketClient.sendPrivateMessage(
                targetUserId = "peer-1",
                targetUserName = "Bob",
                senderId = fixture.session.id,
                content = "[folder/demo.jpg]",
            )
        } returns true
        coEvery {
            fixture.mediaApiService.recordImageSend(
                remoteUrl = "https://cdn.test/demo.jpg",
                fromUserId = fixture.session.id,
                toUserId = "peer-1",
                localFilename = "",
            )
        } throws IllegalStateException("record failed")

        val result = fixture.repository.sendUploadedMedia(
            peerId = "peer-1",
            peerName = "Bob",
            media = ChatMediaAsset(
                id = "folder/demo.jpg",
                url = "https://cdn.test/demo.jpg",
                type = ChatMessageType.IMAGE,
                localFilename = "",
                remotePath = "folder/demo.jpg",
            ),
            clientId = "client-1",
        )

        assertTrue(result is AppResult.Success)
        val message = (result as AppResult.Success).data
        assertEquals(ChatMessageType.IMAGE, message.type)
        assertEquals("demo.jpg", message.fileName)
        assertEquals(OutgoingMessageStatus.SENDING, message.sendStatus)
    }

    @Test
    fun `send uploaded media should fail when remote path missing`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session

        val result = fixture.repository.sendUploadedMedia(
            peerId = "peer-1",
            peerName = "Bob",
            media = ChatMediaAsset(
                id = "",
                url = "https://cdn.test/demo.jpg",
                type = ChatMessageType.IMAGE,
            ),
            clientId = "client-1",
        )

        assertTrue(result is AppResult.Error)
        assertEquals("媒体缺少远端路径，无法发送", (result as AppResult.Error).message)
    }

    @Test
    fun `load history should clear cache and persist hydrated first page`() = runTest {
        val fixture = newFixture()
        val upserted = slot<List<MessageEntity>>()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        coEvery {
            fixture.chatApiService.getMessageHistory(
                myUserId = fixture.session.id,
                userToId = "peer-1",
                isFirst = "1",
                firstTid = "0",
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns buildJsonArray {
            add(historyMessageJson(tid = "1", id = "peer-1", content = "你好", time = "10:00:00"))
            add(historyMessageJson(tid = "2", id = fixture.session.id, content = "[folder/demo.jpg]", time = "10:01:00", type = "image"))
        }
        coEvery { fixture.mediaApiService.getImgServer() } returns "{\"server\":\"img.demo.test\"}".toResponseBody("application/json".toMediaType())
        coEvery { fixture.systemApiService.getSystemConfig() } returns ApiEnvelope(data = SystemConfigDto(imagePortMode = "fixed", imagePortFixed = "9400"))
        coEvery { fixture.preferencesStore.saveCachedSystemConfig(any()) } just runs
        coEvery { fixture.messageDao.clearByPeer("peer-1") } just runs
        coEvery { fixture.messageDao.upsert(capture(upserted)) } just runs

        val result = fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(listOf("2", "1"), payload.messages.map { it.id })
        assertEquals("1", payload.historyCursor)
        assertFalse(payload.hasMoreHistory)
        coVerify { fixture.messageDao.clearByPeer("peer-1") }
        assertEquals(listOf("2", "1"), upserted.captured.map { it.id })
        assertEquals("http://img.demo.test:9400/img/Upload/folder/demo.jpg", payload.messages.first().mediaUrl)
    }

    @Test
    fun `load history should fall back to cached messages on first page failure`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        coEvery {
            fixture.chatApiService.getMessageHistory(
                myUserId = fixture.session.id,
                userToId = "peer-1",
                isFirst = "1",
                firstTid = "0",
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } throws IllegalStateException("network down")
        coEvery { fixture.messageDao.listByPeer("peer-1") } returns listOf(
            MessageEntity(
                id = "9",
                peerId = "peer-1",
                fromUserId = fixture.session.id,
                fromUserName = fixture.session.name,
                toUserId = "peer-1",
                content = "cached",
                time = "10:00:00",
                isSelf = true,
                type = ChatMessageType.TEXT.name,
                mediaUrl = "",
                fileName = "",
            )
        )

        val result = fixture.repository.loadHistory(peerId = "peer-1", isFirst = true, firstTid = "0")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(listOf("9"), payload.messages.map { it.id })
        assertEquals("9", payload.historyCursor)
        assertFalse(payload.hasMoreHistory)
    }

    @Test
    fun `load history should return error for subsequent page failure`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        coEvery {
            fixture.chatApiService.getMessageHistory(
                myUserId = fixture.session.id,
                userToId = "peer-1",
                isFirst = "0",
                firstTid = "99",
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } throws IllegalStateException("page failed")

        val result = fixture.repository.loadHistory(peerId = "peer-1", isFirst = false, firstTid = "99")

        assertTrue(result is AppResult.Error)
        assertEquals("page failed", (result as AppResult.Error).message)
    }

    @Test
    fun `send text should build optimistic message when websocket accepts payload`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        every {
            fixture.webSocketClient.sendPrivateMessage(
                targetUserId = "peer-1",
                targetUserName = "Bob",
                senderId = fixture.session.id,
                content = "hello",
            )
        } returns true

        val result = fixture.repository.sendText(peerId = "peer-1", peerName = "Bob", content = "hello", clientId = "client-1")

        assertTrue(result is AppResult.Success)
        val message = (result as AppResult.Success).data
        assertEquals("client-1", message.id)
        assertEquals("client-1", message.clientId)
        assertEquals(OutgoingMessageStatus.SENDING, message.sendStatus)
    }

    @Test
    fun `send text should return error when websocket disconnected`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        every { fixture.webSocketClient.sendPrivateMessage(any(), any(), any(), any()) } returns false

        val result = fixture.repository.sendText(peerId = "peer-1", peerName = "Bob", content = "hello", clientId = "client-1")

        assertTrue(result is AppResult.Error)
        assertEquals("WebSocket 未连接", (result as AppResult.Error).message)
    }

    @Test
    fun `request user info should map websocket result to app result`() = runTest {
        val fixture = newFixture()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        every { fixture.webSocketClient.sendShowUserLoginInfo(senderId = fixture.session.id, targetUserId = "peer-1") } returns true

        val success = fixture.repository.requestUserInfo("peer-1")

        assertTrue(success is AppResult.Success)

        every { fixture.webSocketClient.sendShowUserLoginInfo(senderId = fixture.session.id, targetUserId = "peer-1") } returns false

        val failure = fixture.repository.requestUserInfo("peer-1")

        assertTrue(failure is AppResult.Error)
        assertEquals("WebSocket 未连接，暂时无法查询对方信息", (failure as AppResult.Error).message)
    }

    @Test
    fun `toggle favorite should call proper api update cache and surface failure`() = runTest {
        val fixture = newFixture()
        val updatedConversation = slot<ConversationEntity>()
        coEvery { fixture.preferencesStore.readCurrentSession() } returns fixture.session
        coEvery {
            fixture.chatApiService.toggleFavorite(
                myUserId = fixture.session.id,
                userToId = "peer-1",
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns ApiEnvelope(code = 0, data = Unit)
        coEvery { fixture.conversationDao.getById("peer-1") } returns ConversationEntity(
            id = "peer-1",
            name = "Bob",
            sex = "男",
            ip = "1.1.1.1",
            address = "上海",
            isFavorite = false,
            lastMessage = "hi",
            lastTime = "10:00:00",
            unreadCount = 0,
        )
        coEvery { fixture.conversationDao.upsert(capture(updatedConversation)) } just runs

        val success = fixture.repository.toggleFavorite(peerId = "peer-1", favorite = false)

        assertTrue(success is AppResult.Success)
        assertEquals(true, (success as AppResult.Success).data)
        assertTrue(updatedConversation.captured.isFavorite)

        coEvery {
            fixture.chatApiService.cancelFavorite(
                myUserId = fixture.session.id,
                userToId = "peer-1",
                cookieData = fixture.session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns ApiEnvelope(code = 1, msg = "取消失败")

        val failure = fixture.repository.toggleFavorite(peerId = "peer-1", favorite = true)

        assertTrue(failure is AppResult.Error)
        assertEquals("取消失败", (failure as AppResult.Error).message)
    }

    private fun newFixture(): Fixture {
        val chatApiService = mockk<ChatApiService>()
        val mediaApiService = mockk<MediaApiService>()
        val systemApiService = mockk<SystemApiService>()
        val baseUrlProvider = mockk<BaseUrlProvider>()
        val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
        val messageDao = mockk<MessageDao>(relaxUnitFun = true)
        val conversationDao = mockk<ConversationDao>(relaxUnitFun = true)
        val webSocketClient = mockk<LiaoWebSocketClient>()
        val appContext = mockk<Context>(relaxed = true)
        val contentResolver = mockk<ContentResolver>(relaxed = true)
        every { appContext.contentResolver } returns contentResolver
        every { webSocketClient.events } returns MutableSharedFlow()

        return Fixture(
            chatApiService = chatApiService,
            mediaApiService = mediaApiService,
            systemApiService = systemApiService,
            baseUrlProvider = baseUrlProvider,
            preferencesStore = preferencesStore,
            messageDao = messageDao,
            conversationDao = conversationDao,
            webSocketClient = webSocketClient,
            appContext = appContext,
            repository = ChatRoomRepository(
                chatApiService = chatApiService,
                mediaApiService = mediaApiService,
                systemApiService = systemApiService,
                baseUrlProvider = baseUrlProvider,
                preferencesStore = preferencesStore,
                messageDao = messageDao,
                conversationDao = conversationDao,
                webSocketClient = webSocketClient,
                appContext = appContext,
            ),
        )
    }

    private fun historyMessageJson(
        tid: String,
        id: String,
        content: String,
        time: String,
        type: String? = null,
    ) = buildJsonObject {
        put("Tid", JsonPrimitive(tid))
        put("id", JsonPrimitive(id))
        put("content", JsonPrimitive(content))
        put("time", JsonPrimitive(time))
        if (type != null) {
            put("type", JsonPrimitive(type))
        }
    }

    private fun timelineMessage(
        id: String = "1",
        content: String = "content",
        type: ChatMessageType = ChatMessageType.TEXT,
        mediaUrl: String = "",
        fileName: String = "",
    ): ChatTimelineMessage = ChatTimelineMessage(
        id = id,
        fromUserId = "self-1",
        fromUserName = "Alice",
        toUserId = "peer-1",
        content = content,
        time = "10:00:00",
        isSelf = true,
        type = type,
        mediaUrl = mediaUrl,
        fileName = fileName,
    )

    private data class Fixture(
        val chatApiService: ChatApiService,
        val mediaApiService: MediaApiService,
        val systemApiService: SystemApiService,
        val baseUrlProvider: BaseUrlProvider,
        val preferencesStore: AppPreferencesStore,
        val messageDao: MessageDao,
        val conversationDao: ConversationDao,
        val webSocketClient: LiaoWebSocketClient,
        val appContext: Context,
        val repository: ChatRoomRepository,
        val session: CurrentIdentitySession = CurrentIdentitySession(
            id = "self-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
        ),
    )
}
