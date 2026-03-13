/*
 * WebSocket 客户端负责对齐服务端 `/ws?token=` 握手与 `sign` 绑定协议。
 * 当前实现补齐最小 code/act 协议目录、结构化入站解析、forceout 处理与真实自动重连。
 */
package io.github.a7413498.liao.android.core.websocket

import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.LiaoLogger
import io.github.a7413498.liao.android.core.common.inferPrivateMessageIsSelf
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import java.util.concurrent.atomic.AtomicLong
import javax.inject.Inject
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okio.ByteString

sealed interface WebSocketState {
    data object Idle : WebSocketState
    data object Connecting : WebSocketState
    data object Connected : WebSocketState
    data object Reconnecting : WebSocketState
    data class Forceout(val forbiddenUntilMillis: Long) : WebSocketState
    data object Closed : WebSocketState
}

enum class LiaoWsKnownCode(val wireValue: Int) {
    Reject(-4),
    Forceout(-3),
    PrivateMessage(7),
    ConnectNotice(12),
    InputStart(13),
    InputStop(14),
    MatchSuccess(15),
    MatchCancel(16),
    RandomOut(18),
    WarningReport(19),
    OnlineStatus(30),
}

enum class LiaoWsKnownAct(val wireName: String) {
    Sign("sign"),
    ShowUserLoginInfo("ShowUserLoginInfo"),
    WarningReport("warningreport"),
    RandomOut("randomOut"),
    PrivateMessage("touser_*"),
}

object LiaoWsProtocolCatalog {
    object Codes {
        const val REJECT = -4
        const val FORCEOUT = -3
        const val PRIVATE_MESSAGE = 7
        const val CONNECT_NOTICE = 12
        const val INPUT_START = 13
        const val INPUT_STOP = 14
        const val MATCH_SUCCESS = 15
        const val MATCH_CANCEL = 16
        const val RANDOM_OUT = 18
        const val WARNING_REPORT = 19
        const val ONLINE_STATUS = 30
    }

    object Acts {
        const val SIGN = "sign"
        const val SHOW_USER_LOGIN_INFO = "ShowUserLoginInfo"
        const val WARNING_REPORT = "warningreport"
        const val RANDOM_OUT = "randomOut"

        fun privateMessage(targetUserId: String, targetUserName: String): String =
            "touser_${targetUserId}_${targetUserName}"
    }

    private val parserJson = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        isLenient = true
    }

    fun resolveKnownCode(rawCode: Int?): LiaoWsKnownCode? =
        LiaoWsKnownCode.entries.firstOrNull { it.wireValue == rawCode }

    fun resolveKnownAct(rawAct: String?): LiaoWsKnownAct? {
        val normalized = rawAct.orEmpty().trim()
        if (normalized.isBlank()) return null
        if (normalized.startsWith("touser_")) return LiaoWsKnownAct.PrivateMessage
        return LiaoWsKnownAct.entries.firstOrNull { it != LiaoWsKnownAct.PrivateMessage && it.wireName == normalized }
    }

    fun parseEnvelope(raw: String, json: Json = parserJson): LiaoWsEnvelope? = runCatching {
        val root = json.parseToJsonElement(raw).jsonObject
        val rawCode = root.intOrNull("code")
        val rawActValue = root.stringOrEmpty("act")
        val rawAct = rawActValue.takeIf { it.isNotEmpty() }
        val fromUser = root.objectOrNull("fromuser")?.toProtocolUser()
        val toUser = root.objectOrNull("touser")?.toProtocolUser()
        LiaoWsEnvelope(
            raw = raw,
            code = rawCode,
            knownCode = resolveKnownCode(rawCode),
            act = rawAct,
            knownAct = resolveKnownAct(rawAct),
            forceout = root.booleanOrFalse("forceout"),
            content = firstNonBlank(
                fromUser?.content,
                root.stringOrEmpty("content"),
                root.stringOrEmpty("msg"),
            ),
            fromUser = fromUser,
            toUser = toUser,
            time = firstNonBlank(
                fromUser?.time,
                root.stringOrEmpty("time"),
                root.stringOrEmpty("Time"),
            ),
            tid = firstNonBlank(
                root.stringOrEmpty("tid"),
                root.stringOrEmpty("Tid"),
                fromUser?.tid,
            ),
        )
    }.getOrNull()

    private fun JsonObject.intOrNull(key: String): Int? =
        this[key]?.jsonPrimitive?.intOrNull

    private fun JsonObject.stringOrEmpty(key: String): String =
        this[key]?.jsonPrimitive?.contentOrNull.orEmpty().trim()

    private fun JsonObject.booleanOrFalse(key: String): Boolean =
        this[key]?.jsonPrimitive?.booleanOrNull ?: false

    private fun JsonObject.objectOrNull(key: String): JsonObject? =
        this[key] as? JsonObject

    private fun JsonObject.toProtocolUser(): LiaoWsProtocolUser = LiaoWsProtocolUser(
        id = stringOrEmpty("id"),
        name = stringOrEmpty("name"),
        nickname = stringOrEmpty("nickname"),
        content = firstNonBlank(stringOrEmpty("content"), stringOrEmpty("msg")),
        time = firstNonBlank(stringOrEmpty("time"), stringOrEmpty("Time")),
        tid = firstNonBlank(stringOrEmpty("tid"), stringOrEmpty("Tid")),
    )

    private fun firstNonBlank(vararg values: String?): String =
        values.firstOrNull { !it.isNullOrBlank() }.orEmpty()
}

data class LiaoWsProtocolUser(
    val id: String,
    val name: String,
    val nickname: String,
    val content: String = "",
    val time: String = "",
    val tid: String = "",
) {
    val displayName: String
        get() = nickname.ifBlank { name.ifBlank { "匿名用户" } }
}

data class LiaoWsEnvelope(
    val raw: String,
    val code: Int?,
    val knownCode: LiaoWsKnownCode?,
    val act: String?,
    val knownAct: LiaoWsKnownAct?,
    val forceout: Boolean,
    val content: String,
    val fromUser: LiaoWsProtocolUser?,
    val toUser: LiaoWsProtocolUser?,
    val time: String,
    val tid: String,
) {
    fun isForceoutMessage(): Boolean = forceout && (
        knownCode == LiaoWsKnownCode.Forceout || knownCode == LiaoWsKnownCode.Reject
    )

    fun toTimelineMessage(currentUserId: String?): ChatTimelineMessage? {
        if (knownCode != LiaoWsKnownCode.PrivateMessage) return null
        val sender = fromUser ?: return null
        if (sender.id.isBlank()) return null
        val messageContent = content
        if (messageContent.isBlank()) return null
        val resolvedTime = time.ifBlank { "刚刚" }
        val resolvedTid = tid.ifBlank { "${resolvedTime}_${sender.id}_${messageContent.hashCode()}" }
        return ChatTimelineMessage(
            id = resolvedTid,
            fromUserId = sender.id,
            fromUserName = sender.displayName,
            toUserId = toUser?.id.orEmpty(),
            content = messageContent,
            time = resolvedTime,
            isSelf = !currentUserId.isNullOrBlank() && inferPrivateMessageIsSelf(
                currentUserId = currentUserId,
                fromUserId = sender.id,
            ),
        )
    }
}

sealed interface LiaoWsEvent {
    val raw: String

    data class ChatMessage(
        override val raw: String,
        val envelope: LiaoWsEnvelope,
        val timelineMessage: ChatTimelineMessage,
    ) : LiaoWsEvent

    data class Forceout(
        override val raw: String,
        val envelope: LiaoWsEnvelope,
        val forbiddenUntilMillis: Long,
        val reason: String,
    ) : LiaoWsEvent

    data class Notice(
        override val raw: String,
        val envelope: LiaoWsEnvelope,
        val message: String,
    ) : LiaoWsEvent

    data class Unknown(
        override val raw: String,
        val envelope: LiaoWsEnvelope?,
    ) : LiaoWsEvent
}

class LiaoWebSocketClient @Inject constructor(
    private val okHttpClient: OkHttpClient,
    private val baseUrlProvider: BaseUrlProvider,
    private val json: Json,
) {
    private val clientScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _state = MutableStateFlow<WebSocketState>(WebSocketState.Idle)
    val state: StateFlow<WebSocketState> = _state.asStateFlow()

    private val _messages = MutableSharedFlow<String>(extraBufferCapacity = 64)
    val messages: SharedFlow<String> = _messages.asSharedFlow()

    private val _events = MutableSharedFlow<LiaoWsEvent>(extraBufferCapacity = 64)
    val events: SharedFlow<LiaoWsEvent> = _events.asSharedFlow()

    private var webSocket: WebSocket? = null
    private var currentToken: String? = null
    private var currentSession: CurrentIdentitySession? = null
    private var forceoutUntilMillis: Long = 0L
    private var reconnectJob: Job? = null
    private var reconnectAttempts: Int = 0
    private var shouldReconnect: Boolean = false
    private val socketSerialGenerator = AtomicLong(0L)
    private var activeSocketSerial: Long = 0L

    fun connect(token: String, session: CurrentIdentitySession) {
        val sameBinding = currentToken == token && currentSession == session
        val hasActiveConnection = reconnectJob?.isActive == true || (
            webSocket != null && _state.value in setOf(
                WebSocketState.Connecting,
                WebSocketState.Connected,
                WebSocketState.Reconnecting,
            )
        )

        currentToken = token
        currentSession = session
        shouldReconnect = true

        if (isForceoutActive()) {
            _state.value = WebSocketState.Forceout(forceoutUntilMillis)
            return
        }
        forceoutUntilMillis = 0L

        if (sameBinding && hasActiveConnection) {
            LiaoLogger.i(TAG, "WebSocket 已处于可用连接流程，跳过重复 connect")
            return
        }

        reconnectAttempts = 0
        cancelReconnect()
        openSocket(isReconnect = false)
    }

    fun disconnect(manual: Boolean = true) {
        if (manual) {
            shouldReconnect = false
            cancelReconnect()
        }
        closeCurrentSocket(code = 1000, reason = "client_close")
        if (manual && _state.value !is WebSocketState.Forceout) {
            _state.value = WebSocketState.Closed
        }
    }

    fun sendPrivateMessage(targetUserId: String, targetUserName: String, senderId: String, content: String): Boolean =
        sendJson(
            buildJsonObject {
                put("act", JsonPrimitive(LiaoWsProtocolCatalog.Acts.privateMessage(targetUserId, targetUserName)))
                put("id", JsonPrimitive(senderId))
                put("msg", JsonPrimitive(content))
            }
        )

    fun sendShowUserLoginInfo(senderId: String, targetUserId: String): Boolean = sendJson(
        buildJsonObject {
            put("act", JsonPrimitive(LiaoWsProtocolCatalog.Acts.SHOW_USER_LOGIN_INFO))
            put("id", JsonPrimitive(senderId))
            put("msg", JsonPrimitive(targetUserId))
            put("randomvipcode", JsonPrimitive("vipali67fbff86676e361016812533"))
        }
    )

    fun sendWarningReport(senderId: String, targetUserId: String): Boolean = sendJson(
        buildJsonObject {
            put("act", JsonPrimitive(LiaoWsProtocolCatalog.Acts.WARNING_REPORT))
            put("id", JsonPrimitive(senderId))
            put("msg", JsonPrimitive(targetUserId))
        }
    )

    fun sendRandomOut(senderId: String): Boolean = sendJson(
        buildJsonObject {
            put("act", JsonPrimitive(LiaoWsProtocolCatalog.Acts.RANDOM_OUT))
            put("id", JsonPrimitive(senderId))
        }
    )

    private fun openSocket(isReconnect: Boolean) {
        val token = currentToken ?: return
        val session = currentSession ?: return
        if (isForceoutActive()) {
            _state.value = WebSocketState.Forceout(forceoutUntilMillis)
            shouldReconnect = false
            return
        }

        cancelReconnect()
        val request = Request.Builder().url(baseUrlProvider.currentWebSocketUrl(token)).build()
        val previousSocket = webSocket
        val socketSerial = socketSerialGenerator.incrementAndGet()
        activeSocketSerial = socketSerial
        webSocket = null
        previousSocket?.let { closeSocket(it, code = 1000, reason = "replace_connection") }
        _state.value = if (isReconnect) WebSocketState.Reconnecting else WebSocketState.Connecting
        LiaoLogger.i(TAG, "正在建立 WebSocket 连接: userId=${session.id}, reconnect=$isReconnect")
        webSocket = okHttpClient.newWebSocket(request, createSocketListener(socketSerial))
    }

    private fun cancelReconnect() {
        reconnectJob?.cancel()
        reconnectJob = null
    }

    private fun scheduleReconnect(trigger: String) {
        if (!shouldReconnect) {
            if (_state.value !is WebSocketState.Forceout) {
                _state.value = WebSocketState.Closed
            }
            return
        }
        if (currentToken.isNullOrBlank() || currentSession == null) {
            _state.value = WebSocketState.Closed
            return
        }
        if (isForceoutActive()) {
            _state.value = WebSocketState.Forceout(forceoutUntilMillis)
            shouldReconnect = false
            return
        }

        reconnectAttempts += 1
        val delayMillis = computeReconnectDelayMillis(reconnectAttempts)
        cancelReconnect()
        _state.value = WebSocketState.Reconnecting
        LiaoLogger.w(TAG, "WebSocket 已断开，准备第${reconnectAttempts}次自动重连，${delayMillis}ms 后执行，触发原因=$trigger")
        reconnectJob = clientScope.launch {
            delay(delayMillis)
            if (!shouldReconnect) return@launch
            if (isForceoutActive()) {
                _state.value = WebSocketState.Forceout(forceoutUntilMillis)
                shouldReconnect = false
                return@launch
            }
            openSocket(isReconnect = true)
        }
    }

    private fun sendJson(payload: JsonObject): Boolean {
        val ws = webSocket ?: return false
        return ws.send(json.encodeToString(payload))
    }

    private fun sendSign() {
        val session = currentSession ?: return
        sendJson(
            buildJsonObject {
                put("act", JsonPrimitive(LiaoWsProtocolCatalog.Acts.SIGN))
                put("id", JsonPrimitive(session.id))
                put("name", JsonPrimitive(session.name))
                put("userSex", JsonPrimitive(session.sex))
                put("address_show", JsonPrimitive("false"))
                put("randomhealthmode", JsonPrimitive("0"))
                put("randomvipsex", JsonPrimitive("0"))
                put("randomvipaddress", JsonPrimitive("0"))
                put("userip", JsonPrimitive(session.ip))
                put("useraddree", JsonPrimitive(session.area))
                put("randomvipcode", JsonPrimitive(""))
            }
        )
    }

    private fun createSocketListener(socketSerial: Long): WebSocketListener = object : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            if (!isActiveSocket(socketSerial, webSocket)) return
            reconnectAttempts = 0
            LiaoLogger.i(TAG, "WebSocket 已连接")
            _state.value = WebSocketState.Connected
            sendSign()
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
            if (!isActiveSocket(socketSerial, webSocket)) return
            handleInboundMessage(webSocket = webSocket, socketSerial = socketSerial, raw = text)
        }

        override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
            if (!isActiveSocket(socketSerial, webSocket)) return
            handleInboundMessage(webSocket = webSocket, socketSerial = socketSerial, raw = bytes.utf8())
        }

        override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
            if (!isActiveSocket(socketSerial, webSocket)) return
            closeSocket(webSocket, code, reason)
        }

        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
            if (!isActiveSocket(socketSerial, webSocket)) return
            releaseActiveSocket(webSocket)
            if (_state.value is WebSocketState.Forceout) {
                return
            }
            scheduleReconnect(trigger = "onClosed(code=$code, reason=$reason)")
        }

        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
            if (!isActiveSocket(socketSerial, webSocket)) return
            LiaoLogger.e(TAG, "WebSocket 连接失败", t)
            releaseActiveSocket(webSocket)
            if (_state.value is WebSocketState.Forceout) {
                return
            }
            scheduleReconnect(trigger = "onFailure")
        }
    }

    private fun handleInboundMessage(webSocket: WebSocket, socketSerial: Long, raw: String) {
        _messages.tryEmit(raw)
        val envelope = LiaoWsProtocolCatalog.parseEnvelope(raw = raw, json = json)
        if (envelope == null) {
            _events.tryEmit(LiaoWsEvent.Unknown(raw = raw, envelope = null))
            return
        }

        if (envelope.isForceoutMessage()) {
            forceoutUntilMillis = System.currentTimeMillis() + FORCEOUT_DURATION_MILLIS
            shouldReconnect = false
            cancelReconnect()
            _state.value = WebSocketState.Forceout(forceoutUntilMillis)
            _events.tryEmit(
                LiaoWsEvent.Forceout(
                    raw = raw,
                    envelope = envelope,
                    forbiddenUntilMillis = forceoutUntilMillis,
                    reason = envelope.content.ifBlank {
                        if (envelope.knownCode == LiaoWsKnownCode.Reject) {
                            "连接被拒绝，请稍后再试"
                        } else {
                            "请不要在同一个浏览器下重复登录"
                        }
                    },
                )
            )
            if (isActiveSocket(socketSerial, webSocket)) {
                closeCurrentSocket(
                    code = if (envelope.knownCode == LiaoWsKnownCode.Reject) 4004 else 4003,
                    reason = "forceout",
                )
            }
            return
        }

        val chatMessage = envelope.toTimelineMessage(currentUserId = currentSession?.id)
        if (chatMessage != null) {
            _events.tryEmit(LiaoWsEvent.ChatMessage(raw = raw, envelope = envelope, timelineMessage = chatMessage))
            return
        }

        if (envelope.content.isNotBlank()) {
            _events.tryEmit(
                LiaoWsEvent.Notice(
                    raw = raw,
                    envelope = envelope,
                    message = envelope.content,
                )
            )
            return
        }

        _events.tryEmit(LiaoWsEvent.Unknown(raw = raw, envelope = envelope))
    }

    private fun isForceoutActive(nowMillis: Long = System.currentTimeMillis()): Boolean =
        nowMillis < forceoutUntilMillis

    private fun isActiveSocket(socketSerial: Long, socket: WebSocket): Boolean =
        activeSocketSerial == socketSerial && webSocket === socket

    private fun releaseActiveSocket(socket: WebSocket) {
        if (webSocket === socket) {
            webSocket = null
        }
    }

    private fun closeCurrentSocket(code: Int, reason: String) {
        val activeSocket = webSocket ?: return
        webSocket = null
        closeSocket(activeSocket, code, reason)
    }

    private fun closeSocket(socket: WebSocket, code: Int, reason: String) {
        val closed = runCatching { socket.close(code, reason) }.getOrDefault(false)
        if (!closed) {
            socket.cancel()
        }
    }

    companion object {
        const val TAG = "LiaoWebSocket"
        const val FORCEOUT_DURATION_MILLIS = 5 * 60 * 1000L
        private const val RECONNECT_BASE_DELAY_MILLIS = 3_000L
        private const val RECONNECT_MAX_DELAY_MILLIS = 15_000L

        internal fun computeReconnectDelayMillis(attempt: Int): Long =
            (RECONNECT_BASE_DELAY_MILLIS * attempt.coerceAtLeast(1).toLong())
                .coerceAtMost(RECONNECT_MAX_DELAY_MILLIS)
    }
}
