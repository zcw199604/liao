/*
 * 会话列表模块负责加载历史 / 收藏会话，并同步更新本地会话缓存。
 * 当前版本改为以 Room 缓存为真实展示数据源，并允许通过 WS 驱动实时更新。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.chatlist

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.app.testing.ChatListTestTags
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.database.toPeer as toCachedPeer
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.toPeer
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.LiaoWsEvent
import javax.inject.Inject
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch

class ChatListRepository @Inject constructor(
    private val chatApiService: ChatApiService,
    private val conversationDao: ConversationDao,
    private val preferencesStore: AppPreferencesStore,
) {
    fun observeConversations(tab: ConversationTab): Flow<List<ChatPeer>> =
        conversationDao.observeAll().map { items ->
            items.filter { if (tab == ConversationTab.HISTORY) true else it.isFavorite }.map { it.toCachedPeer() }
        }

    suspend fun loadHistory(): AppResult<Unit> = loadConversations(isFavorite = false)

    suspend fun loadFavorite(): AppResult<Unit> = loadConversations(isFavorite = true)

    suspend fun markPeerRead(peerId: String) {
        conversationDao.markAsRead(peerId)
    }

    private suspend fun loadConversations(isFavorite: Boolean): AppResult<Unit> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val items = if (isFavorite) {
            chatApiService.getFavoriteUserList(
                myUserId = session.id,
                cookieData = session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            ).map { it.toPeer(isFavoriteOverride = true) }
        } else {
            chatApiService.getHistoryUserList(
                myUserId = session.id,
                cookieData = session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            ).filterNot { it.id == session.id }.map { it.toPeer() }
        }
        conversationDao.upsert(items.map {
            val current = conversationDao.getById(it.id)
            ConversationEntity(
                id = it.id,
                name = it.name,
                sex = it.sex.ifBlank { current?.sex.orEmpty() },
                ip = it.ip.ifBlank { current?.ip.orEmpty() },
                address = it.address.ifBlank { current?.address.orEmpty() },
                isFavorite = if (isFavorite) true else (current?.isFavorite ?: it.isFavorite),
                lastMessage = it.lastMessage.ifBlank { current?.lastMessage.orEmpty() },
                lastTime = it.lastTime.ifBlank { current?.lastTime.orEmpty() },
                unreadCount = current?.unreadCount ?: it.unreadCount,
            )
        })
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "加载会话失败", it) },
    )
}

enum class ConversationTab {
    HISTORY,
    FAVORITE,
}

data class ChatListUiState(
    val tab: ConversationTab = ConversationTab.HISTORY,
    val loading: Boolean = true,
    val items: List<ChatPeer> = emptyList(),
    val infoMessage: String? = null,
    val errorMessage: String? = null,
)

@HiltViewModel
class ChatListViewModel @Inject constructor(
    private val repository: ChatListRepository,
    private val webSocketClient: LiaoWebSocketClient,
) : ViewModel() {
    var uiState by mutableStateOf(ChatListUiState())
        private set

    private var observeJob: Job? = null

    init {
        observeCurrentTab()
        observeWebSocketEvents()
        refresh()
    }

    fun switchTab(tab: ConversationTab) {
        if (uiState.tab == tab) return
        uiState = uiState.copy(tab = tab, loading = true, errorMessage = null)
        observeCurrentTab()
        refresh()
    }

    fun consumeInfoMessage() {
        if (uiState.infoMessage != null) {
            uiState = uiState.copy(infoMessage = null)
        }
    }

    fun refresh() {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, errorMessage = null)
            when (val result = if (uiState.tab == ConversationTab.HISTORY) repository.loadHistory() else repository.loadFavorite()) {
                is AppResult.Success -> uiState = uiState.copy(loading = false, errorMessage = null)
                is AppResult.Error -> uiState = uiState.copy(loading = false, errorMessage = result.message)
            }
        }
    }

    fun markPeerRead(peerId: String) {
        viewModelScope.launch {
            repository.markPeerRead(peerId)
        }
    }

    private fun observeCurrentTab() {
        observeJob?.cancel()
        observeJob = viewModelScope.launch {
            repository.observeConversations(uiState.tab).collect { items ->
                uiState = uiState.copy(items = items, loading = false)
            }
        }
    }

    private fun observeWebSocketEvents() {
        viewModelScope.launch {
            webSocketClient.events.collectLatest { event ->
                when (event) {
                    is LiaoWsEvent.ConnectNotice -> {
                        if (event.message.isNotBlank()) {
                            uiState = uiState.copy(infoMessage = event.message)
                        }
                    }
                    is LiaoWsEvent.MatchCancelled -> {
                        if (event.message.isNotBlank()) {
                            uiState = uiState.copy(infoMessage = event.message)
                        }
                    }
                    is LiaoWsEvent.MatchSuccess -> {
                        uiState = uiState.copy(infoMessage = "匹配成功：${event.candidate.name}")
                    }
                    else -> Unit
                }
            }
        }
    }
}

@Composable
private fun ChatListStateCard(
    title: String,
    description: String,
    primaryActionText: String,
    onPrimaryAction: () -> Unit,
    secondaryActionText: String,
    onSecondaryAction: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Card(modifier = modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(title, style = MaterialTheme.typography.headlineSmall)
            Text(description, style = MaterialTheme.typography.bodyMedium)
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Button(onClick = onPrimaryAction, modifier = Modifier.weight(1f)) {
                    Text(primaryActionText)
                }
                OutlinedButton(onClick = onSecondaryAction, modifier = Modifier.weight(1f)) {
                    Text(secondaryActionText)
                }
            }
        }
    }
}

@Composable
fun ChatListScreen(
    viewModel: ChatListViewModel,
    onOpenSettings: () -> Unit,
    onOpenGlobalFavorites: () -> Unit,
    onOpenChat: (String, String) -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(state.infoMessage) {
        state.infoMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeInfoMessage()
        }
    }

    LaunchedEffect(state.errorMessage) {
        if (!state.errorMessage.isNullOrBlank() && state.items.isNotEmpty()) {
            snackbarHostState.showSnackbar(state.errorMessage)
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        ChatListScreenContent(
            state = state,
            onSwitchTab = viewModel::switchTab,
            onRefresh = viewModel::refresh,
            onOpenGlobalFavorites = onOpenGlobalFavorites,
            onOpenSettings = onOpenSettings,
            onOpenChat = onOpenChat,
            onMarkPeerRead = viewModel::markPeerRead,
            modifier = Modifier.padding(padding),
        )
    }
}

internal fun chatListErrorTitle(tab: ConversationTab): String =
    if (tab == ConversationTab.HISTORY) "历史会话加载失败" else "收藏会话加载失败"

internal fun chatListEmptyTitle(tab: ConversationTab): String =
    if (tab == ConversationTab.HISTORY) "暂无历史会话" else "暂无收藏会话"

internal fun chatListEmptyDescription(tab: ConversationTab): String =
    if (tab == ConversationTab.HISTORY) {
        "开始聊天后，这里会显示最近联系的人。"
    } else {
        "你还没有收藏任何会话，可以通过全局收藏查看不同身份下的收藏对象。"
    }

@Composable
fun ChatListScreenContent(
    state: ChatListUiState,
    onSwitchTab: (ConversationTab) -> Unit,
    onRefresh: () -> Unit,
    onOpenGlobalFavorites: () -> Unit,
    onOpenSettings: () -> Unit,
    onOpenChat: (String, String) -> Unit,
    onMarkPeerRead: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.fillMaxSize(),
    ) {
        TopAppBar(
            title = { Text("会话列表") },
            actions = {
                TextButton(
                    onClick = onOpenGlobalFavorites,
                    modifier = Modifier.testTag(ChatListTestTags.TOP_GLOBAL_FAVORITES_BUTTON),
                ) {
                    Text("全局收藏")
                }
                IconButton(
                    onClick = onOpenSettings,
                    modifier = Modifier.testTag(ChatListTestTags.SETTINGS_BUTTON),
                ) {
                    Icon(imageVector = Icons.Outlined.Settings, contentDescription = "设置")
                }
            }
        )
        TabRow(selectedTabIndex = state.tab.ordinal) {
            Tab(
                selected = state.tab == ConversationTab.HISTORY,
                onClick = { onSwitchTab(ConversationTab.HISTORY) },
                text = { Text("历史") },
                modifier = Modifier.testTag(ChatListTestTags.HISTORY_TAB),
            )
            Tab(
                selected = state.tab == ConversationTab.FAVORITE,
                onClick = { onSwitchTab(ConversationTab.FAVORITE) },
                text = { Text("收藏") },
                modifier = Modifier.testTag(ChatListTestTags.FAVORITE_TAB),
            )
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            OutlinedButton(
                onClick = onOpenGlobalFavorites,
                modifier = Modifier
                    .weight(1f)
                    .testTag(ChatListTestTags.QUICK_GLOBAL_FAVORITES_BUTTON),
            ) {
                Text("全局收藏")
            }
            Button(
                onClick = onRefresh,
                enabled = !state.loading,
                modifier = Modifier
                    .weight(1f)
                    .testTag(ChatListTestTags.REFRESH_BUTTON),
            ) {
                Text("刷新列表")
            }
        }
        when {
            state.loading && state.items.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(modifier = Modifier.testTag(ChatListTestTags.LOADING_INDICATOR))
                }
            }

            !state.errorMessage.isNullOrBlank() && state.items.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f)
                        .padding(horizontal = 16.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    ChatListStateCard(
                        title = chatListErrorTitle(state.tab),
                        description = state.errorMessage,
                        primaryActionText = "重试",
                        onPrimaryAction = onRefresh,
                        secondaryActionText = "打开全局收藏",
                        onSecondaryAction = onOpenGlobalFavorites,
                        modifier = Modifier.testTag(ChatListTestTags.STATE_CARD),
                    )
                }
            }

            state.items.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f)
                        .padding(horizontal = 16.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    ChatListStateCard(
                        title = chatListEmptyTitle(state.tab),
                        description = chatListEmptyDescription(state.tab),
                        primaryActionText = "刷新",
                        onPrimaryAction = onRefresh,
                        secondaryActionText = "查看全局收藏",
                        onSecondaryAction = onOpenGlobalFavorites,
                        modifier = Modifier.testTag(ChatListTestTags.STATE_CARD),
                    )
                }
            }

            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f)
                        .padding(horizontal = 16.dp)
                        .testTag(ChatListTestTags.LIST),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    items(state.items, key = { it.id }) { peer ->
                        Card(
                            modifier = Modifier
                                .fillMaxWidth()
                                .testTag(ChatListTestTags.item(peer.id))
                                .clickable {
                                    onMarkPeerRead(peer.id)
                                    onOpenChat(peer.id, peer.name)
                                }
                        ) {
                            Column(
                                modifier = Modifier.padding(16.dp),
                                verticalArrangement = Arrangement.spacedBy(4.dp),
                            ) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                ) {
                                    Text(text = peer.name, style = MaterialTheme.typography.titleMedium)
                                    Text(text = peer.lastTime.ifBlank { "--" })
                                }
                                Text(text = peer.lastMessage.ifBlank { "暂无消息" }, maxLines = 2)
                                if (peer.unreadCount > 0) {
                                    Text(
                                        text = "未读 ${peer.unreadCount}",
                                        color = MaterialTheme.colorScheme.primary,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
