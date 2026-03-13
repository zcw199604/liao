/*
 * 会话列表模块负责加载历史 / 收藏会话，并同步更新本地会话缓存。
 * 当前实现补齐了显式空态、错误态与全局收藏入口占位，方便后续继续扩展。
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
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.toPeer
import javax.inject.Inject
import kotlinx.coroutines.launch

class ChatListRepository @Inject constructor(
    private val chatApiService: ChatApiService,
    private val conversationDao: ConversationDao,
    private val preferencesStore: AppPreferencesStore,
) {
    suspend fun loadHistory(): AppResult<List<ChatPeer>> = loadConversations(isFavorite = false)

    suspend fun loadFavorite(): AppResult<List<ChatPeer>> = loadConversations(isFavorite = true)

    private suspend fun loadConversations(isFavorite: Boolean): AppResult<List<ChatPeer>> = runCatching {
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
            ConversationEntity(
                id = it.id,
                name = it.name,
                sex = it.sex,
                ip = it.ip,
                address = it.address,
                isFavorite = it.isFavorite,
                lastMessage = it.lastMessage,
                lastTime = it.lastTime,
                unreadCount = it.unreadCount,
            )
        })
        items
    }.fold(
        onSuccess = { AppResult.Success(it) },
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
) : ViewModel() {
    var uiState by mutableStateOf(ChatListUiState())
        private set

    init {
        refresh()
    }

    fun switchTab(tab: ConversationTab) {
        if (uiState.tab == tab) return
        uiState = uiState.copy(tab = tab)
        refresh()
    }

    fun showGlobalFavoriteEntryTip() {
        uiState = uiState.copy(infoMessage = "已预留全局收藏入口，后续将接入独立页面。")
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
                is AppResult.Success -> uiState = uiState.copy(
                    loading = false,
                    items = result.data,
                    errorMessage = null,
                )

                is AppResult.Error -> uiState = uiState.copy(
                    loading = false,
                    items = emptyList(),
                    errorMessage = result.message,
                )
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
) {
    Card(modifier = Modifier.fillMaxWidth()) {
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

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("会话列表") },
                actions = {
                    TextButton(onClick = viewModel::showGlobalFavoriteEntryTip) {
                        Text("全局收藏")
                    }
                    IconButton(onClick = onOpenSettings) {
                        Icon(imageVector = Icons.Outlined.Settings, contentDescription = "设置")
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
        ) {
            TabRow(selectedTabIndex = state.tab.ordinal) {
                Tab(
                    selected = state.tab == ConversationTab.HISTORY,
                    onClick = { viewModel.switchTab(ConversationTab.HISTORY) },
                    text = { Text("历史") },
                )
                Tab(
                    selected = state.tab == ConversationTab.FAVORITE,
                    onClick = { viewModel.switchTab(ConversationTab.FAVORITE) },
                    text = { Text("收藏") },
                )
            }
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                OutlinedButton(
                    onClick = viewModel::showGlobalFavoriteEntryTip,
                    modifier = Modifier.weight(1f),
                ) {
                    Text("全局收藏入口（占位）")
                }
                Button(
                    onClick = viewModel::refresh,
                    enabled = !state.loading,
                    modifier = Modifier.weight(1f),
                ) {
                    Text("刷新列表")
                }
            }
            when {
                state.loading -> {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f),
                        contentAlignment = Alignment.Center,
                    ) {
                        CircularProgressIndicator()
                    }
                }

                !state.errorMessage.isNullOrBlank() -> {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f)
                            .padding(horizontal = 16.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        ChatListStateCard(
                            title = if (state.tab == ConversationTab.HISTORY) "历史会话加载失败" else "收藏会话加载失败",
                            description = state.errorMessage,
                            primaryActionText = "重试",
                            onPrimaryAction = viewModel::refresh,
                            secondaryActionText = "打开全局收藏入口",
                            onSecondaryAction = viewModel::showGlobalFavoriteEntryTip,
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
                            title = if (state.tab == ConversationTab.HISTORY) "暂无历史会话" else "暂无收藏会话",
                            description = if (state.tab == ConversationTab.HISTORY) {
                                "开始聊天后，这里会显示最近联系的人。"
                            } else {
                                "你还没有收藏任何会话，可以先使用上方的全局收藏入口占位按钮。"
                            },
                            primaryActionText = "刷新",
                            onPrimaryAction = viewModel::refresh,
                            secondaryActionText = "查看全局收藏入口",
                            onSecondaryAction = viewModel::showGlobalFavoriteEntryTip,
                        )
                    }
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f)
                            .padding(horizontal = 16.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        items(state.items, key = { it.id }) { peer ->
                            Card(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable { onOpenChat(peer.id, peer.name) }
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
}
