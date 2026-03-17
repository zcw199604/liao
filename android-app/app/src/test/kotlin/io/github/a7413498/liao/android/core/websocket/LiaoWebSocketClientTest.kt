package io.github.a7413498.liao.android.core.websocket

import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.LiaoLogger
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkObject
import io.mockk.unmockkObject
import io.mockk.verify
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okio.ByteString.Companion.encodeUtf8
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LiaoWebSocketClientTest {
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        isLenient = true
    }

    @Before
    fun setUp() {
        mockkObject(LiaoLogger)
        every { LiaoLogger.i(any(), any()) } returns 0
        every { LiaoLogger.w(any(), any(), any()) } returns 0
        every { LiaoLogger.e(any(), any(), any()) } returns 0
    }

    @After
    fun tearDown() {
        unmockkObject(LiaoLogger)
    }

    @Test
    fun `connect should send sign and outbound actions while duplicate binding is ignored`() = runTest {
        val okHttpClient = mockk<OkHttpClient>()
        val baseUrlProvider = mockk<io.github.a7413498.liao.android.core.network.BaseUrlProvider>()
        val webSocket = mockk<WebSocket>(relaxed = true)
        val requests = mutableListOf<Request>()
        val listeners = mutableListOf<WebSocketListener>()
        val sentPayloads = mutableListOf<String>()
        every { baseUrlProvider.currentWebSocketUrl(any()) } answers { "ws://demo.test/ws?token=${firstArg<String>()}" }
        every { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) } answers {
            requests += firstArg<Request>()
            listeners += secondArg<WebSocketListener>()
            webSocket
        }
        every { webSocket.send(any<String>()) } answers {
            sentPayloads += firstArg<String>()
            true
        }
        every { webSocket.close(any<Int>(), any<String>()) } returns true

        val client = LiaoWebSocketClient(okHttpClient, baseUrlProvider, json)
        val session = sampleSession(id = "self-1", name = "Alice", sex = "女")

        client.connect("token-1", session)
        assertEquals(WebSocketState.Connecting, client.state.value)
        assertEquals("http://demo.test/ws?token=token-1", requests.single().url.toString())

        listeners.single().onOpen(webSocket, mockk<Response>())
        assertEquals(WebSocketState.Connected, client.state.value)

        val signPayload = parse(sentPayloads[0])
        assertEquals("sign", signPayload["act"]?.jsonPrimitive?.content)
        assertEquals("self-1", signPayload["id"]?.jsonPrimitive?.content)
        assertEquals("Alice", signPayload["name"]?.jsonPrimitive?.content)
        assertEquals("女", signPayload["userSex"]?.jsonPrimitive?.content)
        assertEquals("127.0.0.1", signPayload["userip"]?.jsonPrimitive?.content)
        assertEquals("Shenzhen", signPayload["useraddree"]?.jsonPrimitive?.content)

        assertTrue(client.sendPrivateMessage("peer-1", "Bob", "self-1", "hello"))
        assertTrue(client.sendShowUserLoginInfo("self-1", "peer-1"))
        assertTrue(client.sendWarningReport("peer-1", "warn-1"))
        assertTrue(client.sendRandomOut("self-1"))
        assertTrue(client.sendChangeName("self-1", "Alice2"))
        assertTrue(client.sendModifyInfo("self-1", "保密"))

        val privatePayload = parse(sentPayloads[1])
        assertEquals("touser_peer-1_Bob", privatePayload["act"]?.jsonPrimitive?.content)
        assertEquals("hello", privatePayload["msg"]?.jsonPrimitive?.content)
        val loginInfoPayload = parse(sentPayloads[2])
        assertEquals("ShowUserLoginInfo", loginInfoPayload["act"]?.jsonPrimitive?.content)
        assertEquals("vipali67fbff86676e361016812533", loginInfoPayload["randomvipcode"]?.jsonPrimitive?.content)
        val warningPayload = parse(sentPayloads[3])
        assertEquals("warningreport", warningPayload["act"]?.jsonPrimitive?.content)
        assertEquals("warn-1", warningPayload["msg"]?.jsonPrimitive?.content)
        val randomOutPayload = parse(sentPayloads[4])
        assertEquals("randomOut", randomOutPayload["act"]?.jsonPrimitive?.content)
        val changeNamePayload = parse(sentPayloads[5])
        assertEquals("chgname", changeNamePayload["act"]?.jsonPrimitive?.content)
        assertEquals("Alice2", changeNamePayload["msg"]?.jsonPrimitive?.content)
        val modifyPayload = parse(sentPayloads[6])
        assertEquals("modinfo", modifyPayload["act"]?.jsonPrimitive?.content)
        assertEquals("保密", modifyPayload["userSex"]?.jsonPrimitive?.content)
        assertEquals("false", modifyPayload["address_show"]?.jsonPrimitive?.content)

        client.connect("token-1", session)
        verify(exactly = 1) { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) }
    }

    @Test
    fun `connect with new binding should replace previous socket and manual disconnect should fallback to cancel`() = runTest {
        val okHttpClient = mockk<OkHttpClient>()
        val baseUrlProvider = mockk<io.github.a7413498.liao.android.core.network.BaseUrlProvider>()
        val webSocket1 = mockk<WebSocket>(relaxed = true)
        val webSocket2 = mockk<WebSocket>(relaxed = true)
        val listeners = mutableListOf<WebSocketListener>()
        var createCount = 0
        every { baseUrlProvider.currentWebSocketUrl(any()) } answers { "ws://demo.test/ws?token=${firstArg<String>()}" }
        every { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) } answers {
            listeners += secondArg<WebSocketListener>()
            createCount += 1
            if (createCount == 1) webSocket1 else webSocket2
        }
        every { webSocket1.send(any<String>()) } returns true
        every { webSocket2.send(any<String>()) } returns true
        every { webSocket1.close(1000, "replace_connection") } returns false
        every { webSocket2.close(1000, "client_close") } returns false

        val client = LiaoWebSocketClient(okHttpClient, baseUrlProvider, json)
        client.connect("token-1", sampleSession(id = "self-1"))
        listeners[0].onOpen(webSocket1, mockk<Response>())

        client.connect("token-2", sampleSession(id = "self-2"))
        assertEquals(WebSocketState.Connecting, client.state.value)
        verify(exactly = 1) { webSocket1.close(1000, "replace_connection") }
        verify(exactly = 1) { webSocket1.cancel() }
        verify(exactly = 2) { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) }

        listeners[1].onOpen(webSocket2, mockk<Response>())
        client.disconnect()
        assertEquals(WebSocketState.Closed, client.state.value)
        verify(exactly = 1) { webSocket2.close(1000, "client_close") }
        verify(exactly = 1) { webSocket2.cancel() }
    }

    @Test
    fun `send helpers should return false when socket unavailable`() {
        val client = LiaoWebSocketClient(
            okHttpClient = mockk(relaxed = true),
            baseUrlProvider = mockk(relaxed = true),
            json = json,
        )

        assertFalse(client.sendPrivateMessage("peer-1", "Bob", "self-1", "hello"))
        assertFalse(client.sendShowUserLoginInfo("self-1", "peer-1"))
        assertFalse(client.sendWarningReport("peer-1", "warn-1"))
        assertFalse(client.sendRandomOut("self-1"))
        assertFalse(client.sendChangeName("self-1", "Alice2"))
        assertFalse(client.sendModifyInfo("self-1", "保密"))
    }

    @Test
    fun `inbound messages should emit raw unknown typing notices match chat and fallback events`() = runTest {
        val okHttpClient = mockk<OkHttpClient>()
        val baseUrlProvider = mockk<io.github.a7413498.liao.android.core.network.BaseUrlProvider>()
        val webSocket = mockk<WebSocket>(relaxed = true)
        val listeners = mutableListOf<WebSocketListener>()
        every { baseUrlProvider.currentWebSocketUrl(any()) } returns "ws://demo.test/ws?token=t"
        every { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) } answers {
            listeners += secondArg<WebSocketListener>()
            webSocket
        }
        every { webSocket.send(any<String>()) } returns true
        every { webSocket.close(any<Int>(), any<String>()) } returns true

        val client = LiaoWebSocketClient(okHttpClient, baseUrlProvider, json)
        val events = mutableListOf<LiaoWsEvent>()
        val raws = mutableListOf<String>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) { client.events.collect { events += it } }
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) { client.messages.collect { raws += it } }

        client.connect("token-1", sampleSession(id = "self-1"))
        listeners.single().onOpen(webSocket, mockk<Response>())
        listeners.single().onMessage(webSocket, "not-json")
        listeners.single().onMessage(webSocket, """{"act":"inputStatusOn_peer-1_","fromuser":{"nickname":"Bob"}}""")
        listeners.single().onMessage(webSocket, """{"code":14,"act":"inputStatusOff_peer-2_","fromuser":{"nickname":"Carol"}}""")
        listeners.single().onMessage(webSocket, """{"code":12,"content":""}""")
        listeners.single().onMessage(webSocket, """{"code":30}""")
        listeners.single().onMessage(webSocket, """{"code":15,"sel_userid":"peer-9"}""")
        listeners.single().onMessage(webSocket, """{"code":16,"content":""}""")
        listeners.single().onMessage(webSocket, """{"code":19,"content":"系统提示"}""".encodeUtf8())
        listeners.single().onMessage(webSocket, sampleChatMessageRaw())
        listeners.single().onMessage(webSocket, """{"code":15}""")
        listeners.single().onMessage(webSocket, """{"code":18}""")
        advanceUntilIdle()

        assertEquals(11, raws.size)
        assertEquals(10, events.size)
        assertTrue(events[0] is LiaoWsEvent.Unknown)
        assertEquals(null, (events[0] as LiaoWsEvent.Unknown).envelope)
        assertEquals(true, (events[1] as LiaoWsEvent.Typing).typing)
        assertEquals("peer-1", (events[1] as LiaoWsEvent.Typing).peerId)
        assertEquals("Bob", (events[1] as LiaoWsEvent.Typing).peerName)
        assertEquals(false, (events[2] as LiaoWsEvent.Typing).typing)
        assertEquals("Carol", (events[2] as LiaoWsEvent.Typing).peerName)
        assertEquals("连接成功", (events[3] as LiaoWsEvent.ConnectNotice).message)
        assertEquals("已返回在线状态", (events[4] as LiaoWsEvent.OnlineStatus).message)
        val candidate = (events[5] as LiaoWsEvent.MatchSuccess).candidate
        assertEquals("peer-9", candidate.id)
        assertEquals("匿名用户", candidate.name)
        assertEquals("未知", candidate.sex)
        assertEquals("0", candidate.age)
        assertEquals("未知", candidate.address)
        assertEquals("匹配已取消", (events[6] as LiaoWsEvent.MatchCancelled).message)
        assertEquals("系统提示", (events[7] as LiaoWsEvent.Notice).message)
        val chatMessage = (events[8] as LiaoWsEvent.ChatMessage).timelineMessage
        assertEquals("m-1", chatMessage.id)
        assertEquals("sender-1", chatMessage.fromUserId)
        assertEquals("hello", chatMessage.content)
        assertFalse(chatMessage.isSelf)
        assertTrue(events[9] is LiaoWsEvent.Unknown)
        assertNotNull((events[9] as LiaoWsEvent.Unknown).envelope)
    }

    @Test
    fun `forceout inbound should update state close socket and block reconnect connect`() = runTest {
        val okHttpClient = mockk<OkHttpClient>()
        val baseUrlProvider = mockk<io.github.a7413498.liao.android.core.network.BaseUrlProvider>()
        val webSocket = mockk<WebSocket>(relaxed = true)
        val listeners = mutableListOf<WebSocketListener>()
        val events = mutableListOf<LiaoWsEvent>()
        every { baseUrlProvider.currentWebSocketUrl(any()) } answers { "ws://demo.test/ws?token=${firstArg<String>()}" }
        every { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) } answers {
            listeners += secondArg<WebSocketListener>()
            webSocket
        }
        every { webSocket.send(any<String>()) } returns true
        every { webSocket.close(any<Int>(), any<String>()) } returns true
        val client = LiaoWebSocketClient(okHttpClient, baseUrlProvider, json)
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) { client.events.collect { events += it } }
        client.connect("token-1", sampleSession(id = "self-1"))
        listeners.single().onOpen(webSocket, mockk<Response>())

        listeners.single().onMessage(webSocket, """{"code":-3,"forceout":true,"content":""}""")
        advanceUntilIdle()

        val state = client.state.value
        assertTrue(state is WebSocketState.Forceout)
        assertTrue((state as WebSocketState.Forceout).forbiddenUntilMillis > System.currentTimeMillis())
        val forceoutEvent = events.single() as LiaoWsEvent.Forceout
        assertEquals("请不要在同一个浏览器下重复登录", forceoutEvent.reason)
        verify(exactly = 1) { webSocket.close(4003, "forceout") }

        client.connect("token-2", sampleSession(id = "self-2"))
        verify(exactly = 1) { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) }
        assertTrue(client.state.value is WebSocketState.Forceout)
    }

    @Test
    fun `stale listener should be ignored and close failure should schedule reconnect`() = runTest {
        val okHttpClient = mockk<OkHttpClient>()
        val baseUrlProvider = mockk<io.github.a7413498.liao.android.core.network.BaseUrlProvider>()
        val webSocket1 = mockk<WebSocket>(relaxed = true)
        val webSocket2 = mockk<WebSocket>(relaxed = true)
        val webSocket3 = mockk<WebSocket>(relaxed = true)
        val listeners = mutableListOf<WebSocketListener>()
        val events = mutableListOf<LiaoWsEvent>()
        val raws = mutableListOf<String>()
        var createCount = 0
        every { baseUrlProvider.currentWebSocketUrl(any()) } answers { "ws://demo.test/ws?token=${firstArg<String>()}" }
        every { okHttpClient.newWebSocket(any<Request>(), any<WebSocketListener>()) } answers {
            listeners += secondArg<WebSocketListener>()
            createCount += 1
            when (createCount) {
                1 -> webSocket1
                2 -> webSocket2
                else -> webSocket3
            }
        }
        every { webSocket1.send(any<String>()) } returns true
        every { webSocket2.send(any<String>()) } returns true
        every { webSocket3.send(any<String>()) } returns true
        every { webSocket1.close(any<Int>(), any<String>()) } returns true
        every { webSocket2.close(1001, "bye") } returns false
        every { webSocket3.close(any<Int>(), any<String>()) } returns true

        val client = LiaoWebSocketClient(okHttpClient, baseUrlProvider, json)
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) { client.events.collect { events += it } }
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) { client.messages.collect { raws += it } }

        client.connect("token-1", sampleSession(id = "self-1"))
        listeners[0].onOpen(webSocket1, mockk<Response>())
        client.connect("token-2", sampleSession(id = "self-2"))
        listeners[1].onOpen(webSocket2, mockk<Response>())

        listeners[0].onMessage(webSocket1, "not-json")
        advanceUntilIdle()
        assertTrue(events.isEmpty())
        assertTrue(raws.isEmpty())

        listeners[1].onClosing(webSocket2, 1001, "bye")
        verify(exactly = 1) { webSocket2.close(1001, "bye") }
        verify(exactly = 1) { webSocket2.cancel() }
        listeners[1].onClosed(webSocket2, 1001, "bye")
        assertEquals(WebSocketState.Reconnecting, client.state.value)
        client.disconnect()
        assertEquals(WebSocketState.Closed, client.state.value)

        client.connect("token-3", sampleSession(id = "self-3"))
        listeners[2].onOpen(webSocket3, mockk<Response>())
        listeners[2].onFailure(webSocket3, IllegalStateException("boom"), null)
        assertEquals(WebSocketState.Reconnecting, client.state.value)
        client.disconnect()
        assertEquals(WebSocketState.Closed, client.state.value)
    }

    private fun parse(raw: String) = json.parseToJsonElement(raw).jsonObject

    private fun sampleSession(
        id: String,
        name: String = "Alice",
        sex: String = "女",
    ) = CurrentIdentitySession(
        id = id,
        name = name,
        sex = sex,
        cookie = "cookie-$id",
        ip = "127.0.0.1",
        area = "Shenzhen",
    )

    private fun sampleChatMessageRaw(): String =
        """{"code":7,"Tid":"m-1","time":"10:00","fromuser":{"id":"sender-1","name":"Bob","content":"hello"},"touser":{"id":"peer-1"}}"""
}
