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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.app.testing.ChatListTestTags
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.common.generateCookie
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.database.IdentityDao
import io.github.a7413498.liao.android.core.database.IdentityEntity
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.database.toPeer as toCachedPeer
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ChatArchiveSearchItemDto
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.ContactCandidateDto
import io.github.a7413498.liao.android.core.network.FavoriteApiService
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.network.toPeer
import io.github.a7413498.liao.android.core.network.toFavoriteItemOrNull
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
    private val messageDao: MessageDao,
    private val identityApiService: IdentityApiService,
    private val identityDao: IdentityDao,
    private val favoriteApiService: FavoriteApiService,
    private val preferencesStore: AppPreferencesStore,
    private val webSocketClient: LiaoWebSocketClient,
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

    suspend fun deletePeer(peerId: String): AppResult<Unit> = runCatching {
        val targetId = peerId.trim().ifBlank { error("请选择要删除的会话") }
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val response = chatApiService.deleteUpstreamUser(
            myUserId = session.id,
            userToId = targetId,
        )
        if (response.code != 0) error(response.msg ?: response.message ?: "删除失败")
        conversationDao.deleteById(targetId)
        messageDao.clearByPeer(targetId)
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "删除失败", it) },
    )

    suspend fun requestOnlineStatus(peerId: String): AppResult<Unit> = runCatching {
        val targetId = peerId.trim().ifBlank { error("请选择要查询的会话") }
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val sent = webSocketClient.sendShowUserLoginInfo(senderId = session.id, targetUserId = targetId)
        if (!sent) error("WebSocket 未连接，暂时无法查询在线状态")
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "查询在线状态失败", it) },
    )

    suspend fun loadGlobalFavoriteTargetIds(): AppResult<Set<String>> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val response = favoriteApiService.listAllFavorites()
        if (response.code != 0) error(response.msg ?: response.message ?: "加载全局收藏失败")
        response.data.orEmpty()
            .mapNotNull { it.toFavoriteItemOrNull() }
            .filter { it.identityId == session.id }
            .map { it.targetUserId }
            .filter { it.isNotBlank() }
            .toSet()
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载全局收藏失败", it) },
    )

    suspend fun toggleGlobalFavorite(peer: ChatPeer, isGlobalFavorite: Boolean): AppResult<Boolean> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val targetId = peer.id.trim().ifBlank { error("请选择要收藏的会话") }
        val response = if (isGlobalFavorite) {
            favoriteApiService.removeFavorite(
                identityId = session.id,
                targetUserId = targetId,
            )
        } else {
            favoriteApiService.addFavorite(
                identityId = session.id,
                targetUserId = targetId,
                targetUserName = peer.name.ifBlank { targetId },
            )
        }
        if (response.code != 0) error(response.msg ?: response.message ?: "全局收藏操作失败")
        !isGlobalFavorite
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "全局收藏操作失败", it) },
    )

    suspend fun loadSourceIdentities(): AppResult<List<IdentityDto>> = runCatching {
        val currentSession = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val cached = identityDao.getAll().map { it.toDto() }
        val items = if (cached.isNotEmpty()) {
            cached
        } else {
            val response = identityApiService.getIdentityList()
            val remote = response.data.orEmpty()
            if (remote.isNotEmpty()) {
                identityDao.replaceAll(remote.map { it.toEntity() })
            }
            remote
        }
        items.filterNot { it.id == currentSession.id }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载来源身份失败", it) },
    )

    suspend fun searchArchive(keyword: String, limit: Int = 100): AppResult<List<ChatArchiveSearchItemDto>> = runCatching {
        val normalized = keyword.trim()
        if (normalized.isBlank()) return@runCatching emptyList()
        val response = chatApiService.searchChatArchive(query = normalized, limit = limit)
        if (response.code != 0) error(response.msg ?: response.message ?: "搜索归档失败")
        response.data?.items.orEmpty()
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "搜索归档失败", it) },
    )

    suspend fun prepareArchivedConversation(item: ChatArchiveSearchItemDto): AppResult<ChatPeer> = runCatching {
        upsertTemporaryConversation(item.toContactCandidate(), missingIdMessage = "归档用户缺少 ID")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "接入归档会话失败", it) },
    )

    suspend fun loadContactCandidates(
        sourceIdentity: IdentityDto,
        keyword: String = "",
        limit: Int = 300,
    ): AppResult<List<ContactCandidateDto>> = runCatching {
        val sourceId = sourceIdentity.id.trim().ifBlank { error("请选择来源身份") }
        val response = chatApiService.getContactCandidates(
            sourceIdentityId = sourceId,
            includeUpstream = "1",
            query = keyword.trim(),
            limit = limit,
            cookieData = generateCookie(sourceId, sourceIdentity.name),
            referer = BuildConfig.DEFAULT_REFERER,
            userAgent = BuildConfig.DEFAULT_USER_AGENT,
        )
        if (response.code != 0) error(response.msg ?: response.message ?: "加载跨身份联系人失败")
        response.data?.items.orEmpty()
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载跨身份联系人失败", it) },
    )

    suspend fun prepareContactCandidate(candidate: ContactCandidateDto): AppResult<ChatPeer> = runCatching {
        upsertTemporaryConversation(candidate, missingIdMessage = "候选用户缺少 ID")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "接入跨身份会话失败", it) },
    )

    private suspend fun upsertTemporaryConversation(candidate: ContactCandidateDto, missingIdMessage: String): ChatPeer {
        val targetUserId = candidate.targetUserId.trim().ifBlank { error(missingIdMessage) }
        val entity = ConversationEntity(
            id = targetUserId,
            name = candidate.displayName(),
            sex = candidate.sex.orEmpty(),
            ip = "",
            address = candidate.address.orEmpty().ifBlank { candidate.area.orEmpty() },
            isFavorite = candidate.sources.any { it.equals("favorite", ignoreCase = true) },
            lastMessage = candidate.lastMsg.orEmpty().ifBlank { "临时接入" },
            lastTime = candidate.lastTime.orEmpty().ifBlank { "刚刚" },
            unreadCount = 0,
        )
        conversationDao.upsert(entity)
        return entity.toCachedPeer()
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

private fun ContactCandidateDto.displayName(): String =
    nickname.orEmpty()
        .ifBlank { name.orEmpty() }
        .ifBlank { targetUserName.orEmpty() }
        .ifBlank { targetUserId.take(8) }

private fun ChatArchiveSearchItemDto.displayName(): String = toContactCandidate().displayName()

private fun ChatArchiveSearchItemDto.archiveSummary(): String {
    val sourceLabel = sources.joinToString(separator = " / ").ifBlank { "归档" }
    val message = lastMsg.orEmpty().ifBlank { "暂无消息" }
    return "$sourceLabel · $message"
}

private fun ContactCandidateDto.candidateSummary(): String {
    val sourceLabel = sources.joinToString(separator = " / ").ifBlank { "候选" }
    val message = lastMsg.orEmpty().ifBlank { "暂无消息" }
    return "$sourceLabel · $message"
}

private fun filterContactCandidates(candidates: List<ContactCandidateDto>, keyword: String): List<ContactCandidateDto> {
    val normalized = keyword.trim().lowercase()
    if (normalized.isBlank()) return candidates
    return candidates.filter { candidate ->
        listOf(
            candidate.targetUserId,
            candidate.targetUserName,
            candidate.name,
            candidate.nickname,
            candidate.area,
            candidate.address,
            candidate.lastMsg,
        ).any { value -> value.orEmpty().lowercase().contains(normalized) }
    }
}

private fun IdentityEntity.toDto(): IdentityDto = IdentityDto(
    id = id,
    name = name,
    sex = sex,
    createdAt = createdAt,
    lastUsedAt = lastUsedAt,
)

private fun IdentityDto.toEntity(): IdentityEntity = IdentityEntity(
    id = id,
    name = name,
    sex = sex,
    createdAt = createdAt.orEmpty(),
    lastUsedAt = lastUsedAt.orEmpty(),
)

enum class ConversationTab {
    HISTORY,
    FAVORITE,
}

data class ChatListUiState(
    val tab: ConversationTab = ConversationTab.HISTORY,
    val loading: Boolean = true,
    val items: List<ChatPeer> = emptyList(),
    val crossIdentityVisible: Boolean = false,
    val crossIdentitySourceIdentities: List<IdentityDto> = emptyList(),
    val crossIdentitySelectedSourceId: String = "",
    val crossIdentityKeyword: String = "",
    val crossIdentityLoading: Boolean = false,
    val crossIdentityCandidates: List<ContactCandidateDto> = emptyList(),
    val crossIdentityError: String? = null,
    val archiveSearchVisible: Boolean = false,
    val archiveSearchKeyword: String = "",
    val archiveSearchLoading: Boolean = false,
    val archiveSearchSearched: Boolean = false,
    val archiveSearchItems: List<ChatArchiveSearchItemDto> = emptyList(),
    val archiveSearchError: String? = null,
    val deleteConfirmPeer: ChatPeer? = null,
    val deletingPeerId: String? = null,
    val globalFavoriteTargetIds: Set<String> = emptySet(),
    val togglingGlobalFavoritePeerId: String? = null,
    val checkingOnlinePeerId: String? = null,
    val onlineStatusVisible: Boolean = false,
    val onlineStatusPeerName: String = "",
    val onlineStatusOnline: Boolean? = null,
    val onlineStatusLastTime: String = "",
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
    private var pendingOnlineStatusPeerName: String? = null

    init {
        observeCurrentTab()
        observeWebSocketEvents()
        refresh()
        loadGlobalFavorites()
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

    fun checkPeerOnlineStatus(peer: ChatPeer) {
        val targetId = peer.id.trim()
        if (targetId.isBlank()) {
            uiState = uiState.copy(errorMessage = "请选择要查询的会话")
            return
        }
        val peerName = peer.name.ifBlank { targetId }
        pendingOnlineStatusPeerName = peerName
        viewModelScope.launch {
            uiState = uiState.copy(
                checkingOnlinePeerId = targetId,
                onlineStatusPeerName = peerName,
                errorMessage = null,
                infoMessage = "正在查询在线状态...",
            )
            when (val result = repository.requestOnlineStatus(targetId)) {
                is AppResult.Success -> Unit
                is AppResult.Error -> {
                    pendingOnlineStatusPeerName = null
                    uiState = uiState.copy(
                        checkingOnlinePeerId = null,
                        errorMessage = result.message,
                    )
                }
            }
        }
    }

    fun dismissOnlineStatus() {
        uiState = uiState.copy(onlineStatusVisible = false)
    }

    fun clearPeerUnread(peer: ChatPeer) {
        if (peer.unreadCount <= 0) return
        markPeerRead(peer.id)
    }

    fun requestDeletePeer(peer: ChatPeer) {
        uiState = uiState.copy(deleteConfirmPeer = peer, errorMessage = null)
    }

    fun cancelDeletePeer() {
        if (uiState.deletingPeerId != null) return
        uiState = uiState.copy(deleteConfirmPeer = null)
    }

    fun deletePendingPeer() {
        val peer = uiState.deleteConfirmPeer ?: return
        viewModelScope.launch {
            uiState = uiState.copy(deletingPeerId = peer.id, errorMessage = null)
            when (val result = repository.deletePeer(peer.id)) {
                is AppResult.Success -> uiState = uiState.copy(
                    deleteConfirmPeer = null,
                    deletingPeerId = null,
                    infoMessage = "删除成功",
                )
                is AppResult.Error -> uiState = uiState.copy(
                    deleteConfirmPeer = null,
                    deletingPeerId = null,
                    errorMessage = result.message,
                )
            }
        }
    }

    fun toggleGlobalFavorite(peer: ChatPeer, isGlobalFavorite: Boolean = uiState.globalFavoriteTargetIds.contains(peer.id)) {
        viewModelScope.launch {
            uiState = uiState.copy(togglingGlobalFavoritePeerId = peer.id, errorMessage = null)
            when (val result = repository.toggleGlobalFavorite(peer, isGlobalFavorite)) {
                is AppResult.Success -> {
                    val updatedIds = if (result.data) {
                        uiState.globalFavoriteTargetIds + peer.id
                    } else {
                        uiState.globalFavoriteTargetIds - peer.id
                    }
                    uiState = uiState.copy(
                        globalFavoriteTargetIds = updatedIds,
                        togglingGlobalFavoritePeerId = null,
                        infoMessage = if (result.data) "已加入全局收藏" else "已取消全局收藏",
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(
                    togglingGlobalFavoritePeerId = null,
                    errorMessage = result.message,
                )
            }
        }
    }

    fun openArchiveSearch() {
        uiState = uiState.copy(archiveSearchVisible = true, archiveSearchError = null)
    }

    fun openCrossIdentityPicker() {
        uiState = uiState.copy(crossIdentityVisible = true, crossIdentityError = null)
        viewModelScope.launch {
            uiState = uiState.copy(crossIdentityLoading = true)
            when (val result = repository.loadSourceIdentities()) {
                is AppResult.Success -> {
                    val selectedId = uiState.crossIdentitySelectedSourceId.ifBlank { result.data.firstOrNull()?.id.orEmpty() }
                    uiState = uiState.copy(
                        crossIdentityLoading = false,
                        crossIdentitySourceIdentities = result.data,
                        crossIdentitySelectedSourceId = selectedId,
                        crossIdentityError = null,
                    )
                    if (selectedId.isNotBlank()) {
                        loadContactCandidates()
                    }
                }
                is AppResult.Error -> uiState = uiState.copy(
                    crossIdentityLoading = false,
                    crossIdentitySourceIdentities = emptyList(),
                    crossIdentityCandidates = emptyList(),
                    crossIdentityError = result.message,
                )
            }
        }
    }

    fun closeCrossIdentityPicker() {
        uiState = uiState.copy(
            crossIdentityVisible = false,
            crossIdentityKeyword = "",
            crossIdentityLoading = false,
            crossIdentityCandidates = emptyList(),
            crossIdentityError = null,
        )
    }

    fun selectCrossIdentitySource(sourceIdentityId: String) {
        if (uiState.crossIdentitySelectedSourceId == sourceIdentityId) return
        uiState = uiState.copy(
            crossIdentitySelectedSourceId = sourceIdentityId,
            crossIdentityCandidates = emptyList(),
            crossIdentityError = null,
        )
        loadContactCandidates()
    }

    fun updateCrossIdentityKeyword(value: String) {
        uiState = uiState.copy(crossIdentityKeyword = value)
    }

    fun loadContactCandidates() {
        val sourceIdentity = uiState.crossIdentitySourceIdentities.firstOrNull { it.id == uiState.crossIdentitySelectedSourceId }
        if (sourceIdentity == null) {
            uiState = uiState.copy(crossIdentityCandidates = emptyList(), crossIdentityError = "请选择来源身份")
            return
        }
        viewModelScope.launch {
            uiState = uiState.copy(crossIdentityLoading = true, crossIdentityError = null)
            when (val result = repository.loadContactCandidates(sourceIdentity = sourceIdentity)) {
                is AppResult.Success -> uiState = uiState.copy(
                    crossIdentityLoading = false,
                    crossIdentityCandidates = result.data,
                    crossIdentityError = null,
                )
                is AppResult.Error -> uiState = uiState.copy(
                    crossIdentityLoading = false,
                    crossIdentityCandidates = emptyList(),
                    crossIdentityError = result.message,
                )
            }
        }
    }

    fun closeArchiveSearch() {
        uiState = uiState.copy(
            archiveSearchVisible = false,
            archiveSearchKeyword = "",
            archiveSearchLoading = false,
            archiveSearchSearched = false,
            archiveSearchItems = emptyList(),
            archiveSearchError = null,
        )
    }

    fun updateArchiveSearchKeyword(value: String) {
        uiState = uiState.copy(archiveSearchKeyword = value)
    }

    fun searchArchive() {
        val keyword = uiState.archiveSearchKeyword.trim()
        if (keyword.isBlank()) {
            uiState = uiState.copy(
                archiveSearchItems = emptyList(),
                archiveSearchSearched = false,
                archiveSearchError = null,
            )
            return
        }
        viewModelScope.launch {
            uiState = uiState.copy(archiveSearchLoading = true, archiveSearchSearched = true, archiveSearchError = null)
            when (val result = repository.searchArchive(keyword = keyword)) {
                is AppResult.Success -> uiState = uiState.copy(
                    archiveSearchLoading = false,
                    archiveSearchItems = result.data,
                    archiveSearchError = null,
                )
                is AppResult.Error -> uiState = uiState.copy(
                    archiveSearchLoading = false,
                    archiveSearchItems = emptyList(),
                    archiveSearchError = result.message,
                )
            }
        }
    }

    fun openArchivedChat(item: ChatArchiveSearchItemDto, onOpenChat: (String, String) -> Unit) {
        viewModelScope.launch {
            when (val result = repository.prepareArchivedConversation(item)) {
                is AppResult.Success -> {
                    closeArchiveSearch()
                    onOpenChat(result.data.id, result.data.name)
                }
                is AppResult.Error -> uiState = uiState.copy(archiveSearchError = result.message)
            }
        }
    }

    fun openCrossIdentityChat(candidate: ContactCandidateDto, onOpenChat: (String, String) -> Unit) {
        viewModelScope.launch {
            when (val result = repository.prepareContactCandidate(candidate)) {
                is AppResult.Success -> {
                    closeCrossIdentityPicker()
                    onOpenChat(result.data.id, result.data.name)
                }
                is AppResult.Error -> uiState = uiState.copy(crossIdentityError = result.message)
            }
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

    private fun loadGlobalFavorites() {
        viewModelScope.launch {
            when (val result = repository.loadGlobalFavoriteTargetIds()) {
                is AppResult.Success -> uiState = uiState.copy(globalFavoriteTargetIds = result.data)
                is AppResult.Error -> uiState = uiState.copy(infoMessage = result.message)
            }
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
                    is LiaoWsEvent.OnlineStatus -> handleOnlineStatus(event)
                    else -> Unit
                }
            }
        }
    }

    private fun handleOnlineStatus(event: LiaoWsEvent.OnlineStatus) {
        val peerName = pendingOnlineStatusPeerName ?: return
        pendingOnlineStatusPeerName = null
        uiState = uiState.copy(
            checkingOnlinePeerId = null,
            onlineStatusVisible = true,
            onlineStatusPeerName = peerName,
            onlineStatusOnline = event.isOnline,
            onlineStatusLastTime = event.lastTime,
        )
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
            onOpenCrossIdentity = viewModel::openCrossIdentityPicker,
            onOpenArchiveSearch = viewModel::openArchiveSearch,
            onOpenGlobalFavorites = onOpenGlobalFavorites,
            onOpenSettings = onOpenSettings,
            onOpenChat = onOpenChat,
            onMarkPeerRead = viewModel::markPeerRead,
            onClearPeerUnread = viewModel::clearPeerUnread,
            onToggleGlobalFavorite = { peer, isGlobalFavorite -> viewModel.toggleGlobalFavorite(peer, isGlobalFavorite) },
            onRequestDeletePeer = viewModel::requestDeletePeer,
            onCheckOnlineStatus = viewModel::checkPeerOnlineStatus,
            onDismissOnlineStatus = viewModel::dismissOnlineStatus,
            modifier = Modifier.padding(padding),
        )
        if (state.archiveSearchVisible) {
            ChatArchiveSearchDialog(
                state = state,
                onKeywordChange = viewModel::updateArchiveSearchKeyword,
                onSearch = viewModel::searchArchive,
                onDismiss = viewModel::closeArchiveSearch,
                onSelect = { item -> viewModel.openArchivedChat(item, onOpenChat) },
            )
        }
        if (state.crossIdentityVisible) {
            CrossIdentityContactDialog(
                state = state,
                onSelectSource = viewModel::selectCrossIdentitySource,
                onKeywordChange = viewModel::updateCrossIdentityKeyword,
                onRefreshCandidates = viewModel::loadContactCandidates,
                onDismiss = viewModel::closeCrossIdentityPicker,
                onSelect = { candidate -> viewModel.openCrossIdentityChat(candidate, onOpenChat) },
            )
        }
        state.deleteConfirmPeer?.let { peer ->
            AlertDialog(
                onDismissRequest = viewModel::cancelDeletePeer,
                title = { Text("删除会话") },
                text = { Text("确定删除与 ${peer.name} 的会话吗？本地消息缓存也会清理。") },
                confirmButton = {
                    TextButton(
                        onClick = viewModel::deletePendingPeer,
                        enabled = state.deletingPeerId == null,
                        modifier = Modifier.testTag(ChatListTestTags.DELETE_DIALOG_CONFIRM),
                    ) {
                        Text(if (state.deletingPeerId == peer.id) "删除中..." else "删除")
                    }
                },
                dismissButton = {
                    TextButton(
                        onClick = viewModel::cancelDeletePeer,
                        enabled = state.deletingPeerId == null,
                        modifier = Modifier.testTag(ChatListTestTags.DELETE_DIALOG_CANCEL),
                    ) {
                        Text("取消")
                    }
                },
            )
        }
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
private fun ChatArchiveSearchDialog(
    state: ChatListUiState,
    onKeywordChange: (String) -> Unit,
    onSearch: () -> Unit,
    onDismiss: () -> Unit,
    onSelect: (ChatArchiveSearchItemDto) -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("全局归档搜索") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = state.archiveSearchKeyword,
                        onValueChange = onKeywordChange,
                        modifier = Modifier.weight(1f),
                        singleLine = true,
                        placeholder = { Text("用户 ID 或名称") },
                    )
                    Button(
                        onClick = onSearch,
                        enabled = state.archiveSearchKeyword.trim().isNotBlank() && !state.archiveSearchLoading,
                    ) {
                        Text("搜索")
                    }
                }

                when {
                    state.archiveSearchLoading -> {
                        Box(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = 24.dp),
                            contentAlignment = Alignment.Center,
                        ) {
                            CircularProgressIndicator()
                        }
                    }
                    !state.archiveSearchError.isNullOrBlank() -> Text(
                        text = state.archiveSearchError,
                        color = MaterialTheme.colorScheme.error,
                    )
                    !state.archiveSearchSearched -> Text("暂无搜索")
                    state.archiveSearchItems.isEmpty() -> Text("未找到归档用户。")
                    else -> {
                        LazyColumn(
                            modifier = Modifier
                                .fillMaxWidth()
                                .heightIn(max = 360.dp),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            items(state.archiveSearchItems, key = { "${it.ownerUserId}:${it.targetUserId}" }) { item ->
                                OutlinedButton(
                                    onClick = { onSelect(item) },
                                    modifier = Modifier.fillMaxWidth(),
                                ) {
                                    Column(
                                        modifier = Modifier.fillMaxWidth(),
                                        verticalArrangement = Arrangement.spacedBy(2.dp),
                                    ) {
                                        Text(item.displayName(), style = MaterialTheme.typography.titleSmall)
                                        Text(
                                            text = "${item.targetUserId} · 所属身份 ${item.ownerUserId}",
                                            style = MaterialTheme.typography.bodySmall,
                                        )
                                        Text(
                                            text = item.archiveSummary(),
                                            style = MaterialTheme.typography.bodySmall,
                                        )
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text("关闭")
            }
        },
    )
}

@Composable
private fun CrossIdentityContactDialog(
    state: ChatListUiState,
    onSelectSource: (String) -> Unit,
    onKeywordChange: (String) -> Unit,
    onRefreshCandidates: () -> Unit,
    onDismiss: () -> Unit,
    onSelect: (ContactCandidateDto) -> Unit,
) {
    val filteredCandidates = filterContactCandidates(state.crossIdentityCandidates, state.crossIdentityKeyword)
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("从其他身份接入") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                when {
                    state.crossIdentityLoading && state.crossIdentitySourceIdentities.isEmpty() -> {
                        Box(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = 24.dp),
                            contentAlignment = Alignment.Center,
                        ) {
                            CircularProgressIndicator()
                        }
                    }
                    state.crossIdentitySourceIdentities.isEmpty() -> Text(state.crossIdentityError ?: "暂无其它身份")
                    else -> {
                        LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            items(state.crossIdentitySourceIdentities, key = { it.id }) { identity ->
                                val selected = identity.id == state.crossIdentitySelectedSourceId
                                if (selected) {
                                    Button(onClick = { onSelectSource(identity.id) }) {
                                        Text(identity.name.ifBlank { identity.id })
                                    }
                                } else {
                                    OutlinedButton(onClick = { onSelectSource(identity.id) }) {
                                        Text(identity.name.ifBlank { identity.id })
                                    }
                                }
                            }
                        }

                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            OutlinedTextField(
                                value = state.crossIdentityKeyword,
                                onValueChange = onKeywordChange,
                                modifier = Modifier.weight(1f),
                                singleLine = true,
                                placeholder = { Text("搜索用户") },
                            )
                            Button(
                                onClick = onRefreshCandidates,
                                enabled = state.crossIdentitySelectedSourceId.isNotBlank() && !state.crossIdentityLoading,
                            ) {
                                Text("刷新")
                            }
                        }

                        when {
                            state.crossIdentityLoading -> {
                                Box(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(vertical = 24.dp),
                                    contentAlignment = Alignment.Center,
                                ) {
                                    CircularProgressIndicator()
                                }
                            }
                            !state.crossIdentityError.isNullOrBlank() -> Text(
                                text = state.crossIdentityError,
                                color = MaterialTheme.colorScheme.error,
                            )
                            filteredCandidates.isEmpty() -> Text("暂无可接入用户")
                            else -> {
                                LazyColumn(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .heightIn(max = 360.dp),
                                    verticalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    items(filteredCandidates, key = { it.targetUserId }) { candidate ->
                                        OutlinedButton(
                                            onClick = { onSelect(candidate) },
                                            modifier = Modifier.fillMaxWidth(),
                                        ) {
                                            Column(
                                                modifier = Modifier.fillMaxWidth(),
                                                verticalArrangement = Arrangement.spacedBy(2.dp),
                                            ) {
                                                Text(candidate.displayName(), style = MaterialTheme.typography.titleSmall)
                                                Text(
                                                    text = candidate.targetUserId,
                                                    style = MaterialTheme.typography.bodySmall,
                                                )
                                                Text(
                                                    text = candidate.candidateSummary(),
                                                    style = MaterialTheme.typography.bodySmall,
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
        },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text("关闭")
            }
        },
    )
}

@Composable
fun ChatListScreenContent(
    state: ChatListUiState,
    onSwitchTab: (ConversationTab) -> Unit,
    onRefresh: () -> Unit,
    onOpenCrossIdentity: () -> Unit,
    onOpenArchiveSearch: () -> Unit,
    onOpenGlobalFavorites: () -> Unit,
    onOpenSettings: () -> Unit,
    onOpenChat: (String, String) -> Unit,
    onMarkPeerRead: (String) -> Unit,
    onClearPeerUnread: (ChatPeer) -> Unit,
    onToggleGlobalFavorite: (ChatPeer, Boolean) -> Unit,
    onRequestDeletePeer: (ChatPeer) -> Unit,
    onCheckOnlineStatus: (ChatPeer) -> Unit = {},
    onDismissOnlineStatus: () -> Unit = {},
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
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                OutlinedButton(
                    onClick = onOpenCrossIdentity,
                    modifier = Modifier
                        .weight(1f)
                        .testTag(ChatListTestTags.CROSS_IDENTITY_BUTTON),
                ) {
                    Text("跨身份")
                }
                OutlinedButton(
                    onClick = onOpenArchiveSearch,
                    modifier = Modifier
                        .weight(1f)
                        .testTag(ChatListTestTags.ARCHIVE_SEARCH_BUTTON),
                ) {
                    Text("归档搜索")
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
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
                    Text("刷新")
                }
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
                        val isGlobalFavorite = state.globalFavoriteTargetIds.contains(peer.id)
                        val isTogglingGlobalFavorite = state.togglingGlobalFavoritePeerId == peer.id
                        val isCheckingOnline = state.checkingOnlinePeerId == peer.id
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
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    OutlinedButton(
                                        onClick = { onClearPeerUnread(peer) },
                                        enabled = peer.unreadCount > 0,
                                        modifier = Modifier
                                            .weight(1f)
                                            .testTag(ChatListTestTags.clearUnreadButton(peer.id)),
                                    ) {
                                        Text("清未读")
                                    }
                                    OutlinedButton(
                                        onClick = { onToggleGlobalFavorite(peer, isGlobalFavorite) },
                                        enabled = !isTogglingGlobalFavorite,
                                        modifier = Modifier
                                            .weight(1f)
                                            .testTag(ChatListTestTags.globalFavoriteButton(peer.id)),
                                    ) {
                                        Text(
                                            when {
                                                isTogglingGlobalFavorite -> "处理中..."
                                                isGlobalFavorite -> "取消全局收藏"
                                                else -> "全局收藏"
                                            }
                                        )
                                    }
                                }
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    OutlinedButton(
                                        onClick = { onCheckOnlineStatus(peer) },
                                        enabled = !isCheckingOnline,
                                        modifier = Modifier
                                            .weight(1f)
                                            .testTag(ChatListTestTags.checkOnlineButton(peer.id)),
                                    ) {
                                        Text(if (isCheckingOnline) "查询中..." else "查在线")
                                    }
                                    OutlinedButton(
                                        onClick = { onRequestDeletePeer(peer) },
                                        modifier = Modifier
                                            .weight(1f)
                                            .testTag(ChatListTestTags.deleteButton(peer.id)),
                                    ) {
                                        Text("删除")
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    if (state.onlineStatusVisible) {
        AlertDialog(
            onDismissRequest = onDismissOnlineStatus,
            title = { Text("在线状态") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(state.onlineStatusPeerName.ifBlank { "对方" })
                    Text(onlineStatusLabel(state.onlineStatusOnline))
                    Text(state.onlineStatusLastTime.ifBlank { "暂无时间" })
                }
            },
            confirmButton = {
                TextButton(onClick = onDismissOnlineStatus) {
                    Text("知道了")
                }
            },
        )
    }
}

private fun onlineStatusLabel(isOnline: Boolean?): String = when (isOnline) {
    true -> "在线"
    false -> "离线"
    null -> "状态未知"
}
