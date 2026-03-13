/*
 * 聊天模块串联 HTTP 历史消息、WebSocket 连接状态与最小文本发送流程。
 * 当前版本改为消费结构化 WS 事件，并补齐 forceout / 自动重连后的状态同步。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.chatroom

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.LiaoLogger
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.database.MessageEntity
import io.github.a7413498.liao.android.core.database.toTimelineMessage
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.toTimeline
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.LiaoWsEvent
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import javax.inject.Inject
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch

class ChatRoomRepository @Inject constructor(
    private val chatApiService: ChatApiService,
    private val preferencesStore: AppPreferencesStore,
    private val messageDao: MessageDao,
    private val webSocketClient: LiaoWebSocketClient,
) {
    suspend fun ensureConnected(): AppResult<WebSocketState> = runCatching {
        val token = preferencesStore.readAuthToken().orEmpty()
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        if (token.isBlank()) error("请先登录")
        webSocketClient.connect(token = token, session = session)
        chatApiService.reportReferrer(
            referrerUrl = "",
            currUrl = "android://chatroom",
            userId = session.id,
            cookieData = session.cookie,
            referer = BuildConfig.DEFAULT_REFERER,
            userAgent = BuildConfig.DEFAULT_USER_AGENT,
        )
        webSocketClient.state.value
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "建立聊天连接失败", it) },
    )

    suspend fun loadHistory(peerId: String): AppResult<List<ChatTimelineMessage>> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val items = chatApiService.getMessageHistory(
            myUserId = session.id,
            userToId = peerId,
            isFirst = "1",
            firstTid = "0",
            cookieData = session.cookie,
            referer = BuildConfig.DEFAULT_REFERER,
            userAgent = BuildConfig.DEFAULT_USER_AGENT,
        ).map { it.toTimeline(currentUserId = session.id) }
        messageDao.upsert(items.map {
            MessageEntity(
                id = it.id,
                peerId = peerId,
                fromUserId = it.fromUserId,
                fromUserName = it.fromUserName,
                toUserId = it.toUserId,
                content = it.content,
                time = it.time,
                isSelf = it.isSelf,
            )
        })
        items
    }.recoverCatching {
        messageDao.listByPeer(peerId).map { entity -> entity.toTimelineMessage() }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载聊天记录失败", it) },
    )

    suspend fun sendText(peerId: String, peerName: String, content: String): AppResult<Unit> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val sent = webSocketClient.sendPrivateMessage(
            targetUserId = peerId,
            targetUserName = peerName,
            senderId = session.id,
            content = content,
        )
        if (!sent) error("WebSocket 未连接")
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "发送失败", it) },
    )

    fun connectionState() = webSocketClient.state
    fun inboundEvents() = webSocketClient.events

    suspend fun requestUserInfo(peerId: String): AppResult<Unit> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val sent = webSocketClient.sendShowUserLoginInfo(senderId = session.id, targetUserId = peerId)
        if (!sent) error("WebSocket 未连接，暂时无法查询对方信息")
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "请求用户信息失败", it) },
    )
}

data class ChatRoomUiState(
    val loading: Boolean = true,
    val connectionStateLabel: String = "Idle",
    val messages: List<ChatTimelineMessage> = emptyList(),
    val draft: String = "",
    val message: String? = null,
)

@HiltViewModel
class ChatRoomViewModel @Inject constructor(
    private val repository: ChatRoomRepository,
) : ViewModel() {
    var uiState by mutableStateOf(ChatRoomUiState())
        private set

    private var boundPeerId: String? = null
    private var observersStarted = false
    private var pendingUserInfoPeerId: String? = null

    fun bind(peerId: String) {
        val isPeerChanged = boundPeerId != peerId
        boundPeerId = peerId
        if (isPeerChanged) {
            pendingUserInfoPeerId = peerId
            uiState = uiState.copy(loading = true, messages = emptyList(), message = null)
        }

        viewModelScope.launch {
            when (val result = repository.ensureConnected()) {
                is AppResult.Success -> uiState = uiState.copy(connectionStateLabel = result.data.toDisplayText())
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }

        if (isPeerChanged) {
            viewModelScope.launch {
                when (val result = repository.loadHistory(peerId)) {
                    is AppResult.Success -> uiState = uiState.copy(
                        loading = false,
                        messages = mergeTimelineMessages(
                            current = uiState.messages,
                            incoming = result.data,
                        ),
                    )
                    is AppResult.Error -> uiState = uiState.copy(loading = false, message = result.message)
                }
            }

            if (observersStarted && repository.connectionState().value == WebSocketState.Connected) {
                requestPeerInfo(peerId = peerId, silent = true)
            }
        }

        if (!observersStarted) {
            observersStarted = true
            observeConnectionState()
            observeInboundEvents()
        }
    }

    private fun observeConnectionState() {
        viewModelScope.launch {
            repository.connectionState().collectLatest { state ->
                uiState = uiState.copy(connectionStateLabel = state.toDisplayText())
                if (state == WebSocketState.Connected) {
                    val peerId = pendingUserInfoPeerId
                    if (!peerId.isNullOrBlank()) {
                        requestPeerInfo(peerId = peerId, silent = true)
                    }
                }
            }
        }
    }

    private fun observeInboundEvents() {
        viewModelScope.launch {
            repository.inboundEvents().collectLatest { event ->
                when (event) {
                    is LiaoWsEvent.ChatMessage -> handleIncomingChat(event.timelineMessage)
                    is LiaoWsEvent.Forceout -> uiState = uiState.copy(message = event.reason)
                    is LiaoWsEvent.Notice -> {
                        if (event.message.isNotBlank()) {
                            uiState = uiState.copy(message = event.message)
                        }
                    }
                    is LiaoWsEvent.Unknown -> LiaoLogger.i("ChatRoomViewModel", "收到未识别 WS 消息: ${event.raw}")
                }
            }
        }
    }

    private fun handleIncomingChat(message: ChatTimelineMessage) {
        val currentPeerId = boundPeerId.orEmpty()
        if (currentPeerId.isBlank()) return
        val isCurrentPeerMessage = message.fromUserId == currentPeerId || message.toUserId == currentPeerId
        if (!isCurrentPeerMessage) {
            LiaoLogger.i("ChatRoomViewModel", "忽略非当前会话消息: peerId=$currentPeerId, rawMessageId=${message.id}")
            return
        }
        uiState = uiState.copy(messages = appendTimelineMessage(uiState.messages, message))
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }

    fun requestPeerInfo(peerId: String, silent: Boolean = false) {
        viewModelScope.launch {
            when (val result = repository.requestUserInfo(peerId)) {
                is AppResult.Success -> {
                    pendingUserInfoPeerId = null
                    if (!silent) {
                        uiState = uiState.copy(message = "已请求对方资料刷新")
                    }
                }
                is AppResult.Error -> {
                    if (silent) {
                        pendingUserInfoPeerId = peerId
                    } else {
                        uiState = uiState.copy(message = result.message)
                    }
                }
            }
        }
    }

    fun updateDraft(value: String) {
        uiState = uiState.copy(draft = value)
    }

    fun sendText(peerId: String, peerName: String) {
        if (uiState.draft.isBlank()) return
        val draft = uiState.draft
        viewModelScope.launch {
            when (val result = repository.sendText(peerId, peerName, draft)) {
                is AppResult.Success -> uiState = uiState.copy(
                    draft = "",
                    messages = appendTimelineMessage(
                        current = uiState.messages,
                        incoming = ChatTimelineMessage(
                            id = "local_${uiState.messages.size}",
                            fromUserId = "self",
                            fromUserName = "我",
                            toUserId = peerId,
                            content = draft,
                            time = "刚刚",
                            isSelf = true,
                        ),
                    ),
                )
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }
}

private fun mergeTimelineMessages(
    current: List<ChatTimelineMessage>,
    incoming: List<ChatTimelineMessage>,
): List<ChatTimelineMessage> {
    val merged = linkedMapOf<String, ChatTimelineMessage>()
    (incoming + current).forEach { message ->
        merged[message.id] = message
    }
    return merged.values.toList()
}

private fun appendTimelineMessage(
    current: List<ChatTimelineMessage>,
    incoming: ChatTimelineMessage,
): List<ChatTimelineMessage> {
    if (current.any { it.id == incoming.id }) {
        return current
    }
    return current + incoming
}

private fun WebSocketState.toDisplayText(): String = when (this) {
    WebSocketState.Idle -> "Idle"
    WebSocketState.Connecting -> "Connecting"
    WebSocketState.Connected -> "Connected"
    WebSocketState.Reconnecting -> "Reconnecting"
    is WebSocketState.Forceout -> {
        val remainingMillis = (forbiddenUntilMillis - System.currentTimeMillis()).coerceAtLeast(0L)
        val remainingSeconds = (remainingMillis + 999L) / 1000L
        "Forceout(${remainingSeconds}s)"
    }
    WebSocketState.Closed -> "Closed"
}

@Composable
fun ChatRoomScreen(
    peerId: String,
    peerName: String,
    viewModel: ChatRoomViewModel,
    onBack: () -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(peerId) {
        viewModel.bind(peerId)
    }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(peerName) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.requestPeerInfo(peerId) }) {
                        Icon(Icons.Outlined.Info, contentDescription = "刷新对方资料")
                    }
                }
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("连接状态：${state.connectionStateLabel}")
            if (state.loading) {
                CircularProgressIndicator()
            } else {
                LazyColumn(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(state.messages, key = { it.id }) { message ->
                        Column(
                            modifier = Modifier.fillMaxWidth(),
                            verticalArrangement = Arrangement.spacedBy(2.dp),
                        ) {
                            Text(
                                text = if (message.isSelf) "我" else message.fromUserName,
                                style = MaterialTheme.typography.labelMedium,
                                color = if (message.isSelf) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.secondary,
                            )
                            Text(text = message.content)
                            Text(text = message.time, style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedTextField(
                    modifier = Modifier.weight(1f),
                    value = state.draft,
                    onValueChange = viewModel::updateDraft,
                    label = { Text("输入消息") },
                )
                Button(onClick = { viewModel.sendText(peerId, peerName) }) {
                    Text("发送")
                }
            }
        }
    }
}
