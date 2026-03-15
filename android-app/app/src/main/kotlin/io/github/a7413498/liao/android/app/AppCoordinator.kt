package io.github.a7413498.liao.android.app

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.database.MessageEntity
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.LiaoWsEvent
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch

data class AppCoordinatorUiState(
    val launchResolved: Boolean = false,
    val launchRoute: String = LiaoRoute.LOGIN,
    val sessionExpiredMessage: String? = null,
)

@HiltViewModel
class AppCoordinatorViewModel @Inject constructor(
    private val preferencesStore: AppPreferencesStore,
    private val webSocketClient: LiaoWebSocketClient,
    private val conversationDao: ConversationDao,
    private val messageDao: MessageDao,
) : ViewModel() {
    private val _uiState = MutableStateFlow(AppCoordinatorUiState())
    val uiState: StateFlow<AppCoordinatorUiState> = _uiState.asStateFlow()

    private val activeChatPeerId = MutableStateFlow<String?>(null)

    init {
        observeSessionBinding()
        observeWebSocketEvents()
    }

    fun setActiveChatPeer(peerId: String?) {
        activeChatPeerId.value = peerId
        if (!peerId.isNullOrBlank()) {
            viewModelScope.launch {
                conversationDao.markAsRead(peerId)
            }
        }
    }

    fun consumeSessionExpiredMessage() {
        if (_uiState.value.sessionExpiredMessage != null) {
            _uiState.value = _uiState.value.copy(sessionExpiredMessage = null)
        }
    }

    private fun observeSessionBinding() {
        viewModelScope.launch {
            combine(
                preferencesStore.authTokenFlow,
                preferencesStore.currentSessionFlow,
            ) { token, session -> token.orEmpty() to session }
                .collectLatest { (token, session) ->
                    val launchRoute = when {
                        token.isBlank() -> LiaoRoute.LOGIN
                        session == null -> LiaoRoute.IDENTITY
                        else -> LiaoRoute.CHAT_LIST
                    }
                    _uiState.value = _uiState.value.copy(
                        launchResolved = true,
                        launchRoute = launchRoute,
                    )

                    when {
                        token.isBlank() || session == null -> {
                            webSocketClient.disconnect(manual = true)
                            conversationDao.clearAll()
                            messageDao.clearAll()
                        }
                        else -> webSocketClient.connect(token = token, session = session)
                    }
                }
        }
    }

    private fun observeWebSocketEvents() {
        viewModelScope.launch {
            webSocketClient.events.collectLatest { event ->
                when (event) {
                    is LiaoWsEvent.ChatMessage -> persistMessage(event.timelineMessage)
                    is LiaoWsEvent.MatchSuccess -> persistMatchedConversation(event)
                    is LiaoWsEvent.Forceout -> {
                        preferencesStore.clearAuthToken()
                        preferencesStore.clearCurrentSession()
                        conversationDao.clearAll()
                        messageDao.clearAll()
                        _uiState.value = _uiState.value.copy(sessionExpiredMessage = event.reason)
                    }
                    else -> Unit
                }
            }
        }
    }

    private suspend fun persistMatchedConversation(event: LiaoWsEvent.MatchSuccess) {
        val candidate = event.candidate
        val current = conversationDao.getById(candidate.id)
        conversationDao.upsert(
            ConversationEntity(
                id = candidate.id,
                name = candidate.name.ifBlank { current?.name.orEmpty().ifBlank { candidate.id.take(8) } },
                sex = candidate.sex.ifBlank { current?.sex.orEmpty() },
                ip = current?.ip.orEmpty(),
                address = candidate.address.ifBlank { current?.address.orEmpty() },
                isFavorite = current?.isFavorite ?: false,
                lastMessage = "匹配成功",
                lastTime = "刚刚",
                unreadCount = if (activeChatPeerId.value == candidate.id) 0 else (current?.unreadCount ?: 0),
            )
        )
        if (activeChatPeerId.value == candidate.id) {
            conversationDao.markAsRead(candidate.id)
        }
    }

    private suspend fun persistMessage(message: ChatTimelineMessage) {
        val peerId = message.peerId
        if (peerId.isBlank()) return
        messageDao.upsert(
            MessageEntity(
                id = message.id,
                peerId = peerId,
                fromUserId = message.fromUserId,
                fromUserName = message.fromUserName,
                toUserId = message.toUserId,
                content = message.content,
                time = message.time,
                isSelf = message.isSelf,
                type = message.type.name,
                mediaUrl = message.mediaUrl,
                fileName = message.fileName,
            )
        )

        val current = conversationDao.getById(peerId)
        val isActiveChat = activeChatPeerId.value == peerId
        val displayName = when {
            !message.isSelf && message.fromUserName.isNotBlank() -> message.fromUserName
            !current?.name.isNullOrBlank() -> current?.name.orEmpty()
            else -> peerId.take(8)
        }
        val unreadCount = when {
            message.isSelf -> 0
            isActiveChat -> 0
            else -> (current?.unreadCount ?: 0) + 1
        }
        conversationDao.upsert(
            ConversationEntity(
                id = peerId,
                name = displayName,
                sex = current?.sex.orEmpty(),
                ip = current?.ip.orEmpty(),
                address = current?.address.orEmpty(),
                isFavorite = current?.isFavorite ?: false,
                lastMessage = message.lastMessagePreview(),
                lastTime = message.time,
                unreadCount = unreadCount,
            )
        )
        if (isActiveChat) {
            conversationDao.markAsRead(peerId)
        }
    }
}
