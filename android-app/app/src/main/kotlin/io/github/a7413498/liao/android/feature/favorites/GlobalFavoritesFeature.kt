/*
 * 全局收藏页负责对齐 Web 端“按身份分组查看收藏 / 删除 / 切换身份并进入聊天”的主流程。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.favorites

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
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
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
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.GlobalFavoriteItem
import io.github.a7413498.liao.android.core.common.generateCookie
import io.github.a7413498.liao.android.core.common.generateRandomIp
import io.github.a7413498.liao.android.core.database.FavoriteDao
import io.github.a7413498.liao.android.core.database.FavoriteEntity
import io.github.a7413498.liao.android.core.database.IdentityDao
import io.github.a7413498.liao.android.core.database.IdentityEntity
import io.github.a7413498.liao.android.core.database.toFavoriteItem
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.FavoriteApiService
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.network.toFavoriteItemOrNull
import io.github.a7413498.liao.android.core.network.toSession
import javax.inject.Inject
import kotlinx.coroutines.launch

data class GlobalFavoritesPayload(
    val items: List<GlobalFavoriteItem>,
    val identityNames: Map<String, String>,
)

data class FavoriteGroupUi(
    val identityId: String,
    val identityName: String,
    val items: List<GlobalFavoriteItem>,
)

data class OpenChatPayload(
    val peerId: String,
    val peerName: String,
)

class GlobalFavoritesRepository @Inject constructor(
    private val favoriteApiService: FavoriteApiService,
    private val favoriteDao: FavoriteDao,
    private val identityApiService: IdentityApiService,
    private val identityDao: IdentityDao,
    private val preferencesStore: AppPreferencesStore,
) {
    suspend fun loadFavorites(): AppResult<GlobalFavoritesPayload> = runCatching {
        val response = favoriteApiService.listAllFavorites()
        if (response.code != 0 && response.data == null) {
            error(response.msg ?: response.message ?: "加载全局收藏失败")
        }
        val items = response.data.orEmpty().mapNotNull { it.toFavoriteItemOrNull() }
        favoriteDao.clearAll()
        if (items.isNotEmpty()) {
            favoriteDao.replaceAll(items.map { item ->
                FavoriteEntity(
                    id = item.id,
                    identityId = item.identityId,
                    targetUserId = item.targetUserId,
                    targetUserName = item.targetUserName,
                    createTime = item.createTime,
                )
            })
        }
        GlobalFavoritesPayload(items = items, identityNames = ensureIdentityNames(items.map { it.identityId }.toSet()))
    }.recoverCatching {
        val items = favoriteDao.getAll().map { it.toFavoriteItem() }
        GlobalFavoritesPayload(items = items, identityNames = ensureIdentityNames(items.map { it.identityId }.toSet()))
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载全局收藏失败", it) },
    )

    suspend fun removeFavoriteById(id: Int): AppResult<Unit> = runCatching {
        val response = favoriteApiService.removeFavoriteById(id)
        if (response.code != 0) error(response.msg ?: response.message ?: "取消收藏失败")
        favoriteDao.deleteById(id)
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "取消收藏失败", it) },
    )

    suspend fun switchIdentityAndPrepareChat(item: GlobalFavoriteItem): AppResult<OpenChatPayload> = runCatching {
        val identity = resolveIdentity(item.identityId) ?: error("身份不存在，无法切换")
        val response = identityApiService.selectIdentity(identity.id)
        val selected = response.data ?: identity
        val session = selected.toSession(
            cookie = generateCookie(selected.id, selected.name),
            ip = generateRandomIp(),
        )
        preferencesStore.saveCurrentSession(session)
        OpenChatPayload(
            peerId = item.targetUserId,
            peerName = item.targetUserName.ifBlank { "用户${item.targetUserId.take(4)}" },
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "切换身份失败", it) },
    )

    private suspend fun resolveIdentity(identityId: String): IdentityDto? {
        identityDao.getById(identityId)?.let {
            return IdentityDto(id = it.id, name = it.name, sex = it.sex, createdAt = it.createdAt, lastUsedAt = it.lastUsedAt)
        }
        val response = identityApiService.getIdentityList()
        val items = response.data.orEmpty()
        if (items.isNotEmpty()) {
            identityDao.replaceAll(items.map { dto ->
                IdentityEntity(
                    id = dto.id,
                    name = dto.name,
                    sex = dto.sex,
                    createdAt = dto.createdAt.orEmpty(),
                    lastUsedAt = dto.lastUsedAt.orEmpty(),
                )
            })
        }
        return items.firstOrNull { it.id == identityId }
    }

    private suspend fun ensureIdentityNames(identityIds: Set<String>): Map<String, String> {
        val cached = identityDao.getAll().associate { it.id to it.name }.toMutableMap()
        if (identityIds.any { !cached.containsKey(it) }) {
            val response = identityApiService.getIdentityList()
            val items = response.data.orEmpty()
            if (items.isNotEmpty()) {
                identityDao.replaceAll(items.map { dto ->
                    IdentityEntity(
                        id = dto.id,
                        name = dto.name,
                        sex = dto.sex,
                        createdAt = dto.createdAt.orEmpty(),
                        lastUsedAt = dto.lastUsedAt.orEmpty(),
                    )
                })
                items.forEach { cached[it.id] = it.name }
            }
        }
        return cached
    }
}

data class GlobalFavoritesUiState(
    val loading: Boolean = true,
    val groups: List<FavoriteGroupUi> = emptyList(),
    val message: String? = null,
)

@HiltViewModel
class GlobalFavoritesViewModel @Inject constructor(
    private val repository: GlobalFavoritesRepository,
) : ViewModel() {
    var uiState by mutableStateOf(GlobalFavoritesUiState())
        private set

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, message = null)
            when (val result = repository.loadFavorites()) {
                is AppResult.Success -> {
                    val payload = result.data
                    val groups = payload.items
                        .groupBy { it.identityId }
                        .map { (identityId, items) ->
                            FavoriteGroupUi(
                                identityId = identityId,
                                identityName = payload.identityNames[identityId] ?: "未知身份",
                                items = items.sortedByDescending { it.createTime },
                            )
                        }
                        .sortedBy { it.identityName }
                    uiState = uiState.copy(loading = false, groups = groups)
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, message = result.message)
            }
        }
    }

    fun removeFavoriteById(id: Int) {
        viewModelScope.launch {
            when (val result = repository.removeFavoriteById(id)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(message = "已取消收藏")
                    refresh()
                }
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun switchIdentityAndOpenChat(item: GlobalFavoriteItem, onOpenChat: (String, String) -> Unit) {
        viewModelScope.launch {
            when (val result = repository.switchIdentityAndPrepareChat(item)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(message = "已切换身份，准备进入会话")
                    onOpenChat(result.data.peerId, result.data.peerName)
                }
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }
}

@Composable
fun GlobalFavoritesScreen(
    viewModel: GlobalFavoritesViewModel,
    onBack: () -> Unit,
    onOpenChat: (String, String) -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("全局收藏") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                },
                actions = {
                    TextButton(onClick = viewModel::refresh) {
                        Text("刷新")
                    }
                }
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        when {
            state.loading && state.groups.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator()
                }
            }

            state.groups.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentAlignment = Alignment.Center,
                ) {
                    Text("暂无收藏")
                }
            }

            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding)
                        .padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    items(state.groups, key = { it.identityId }) { group ->
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(
                                text = "${group.identityName} · ${group.identityId}",
                                style = MaterialTheme.typography.titleMedium,
                            )
                            group.items.forEach { item ->
                                Card(modifier = Modifier.fillMaxWidth()) {
                                    Column(
                                        modifier = Modifier.padding(16.dp),
                                        verticalArrangement = Arrangement.spacedBy(8.dp),
                                    ) {
                                        Text(
                                            text = item.targetUserName.ifBlank { item.targetUserId },
                                            style = MaterialTheme.typography.titleSmall,
                                        )
                                        Text(
                                            text = item.targetUserId,
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.outline,
                                        )
                                        Text(
                                            text = item.createTime.ifBlank { "--" },
                                            style = MaterialTheme.typography.bodySmall,
                                        )
                                        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                            Button(
                                                onClick = { viewModel.switchIdentityAndOpenChat(item, onOpenChat) },
                                                modifier = Modifier.weight(1f),
                                            ) {
                                                Text("切换并聊天")
                                            }
                                            OutlinedButton(
                                                onClick = { viewModel.removeFavoriteById(item.id) },
                                                modifier = Modifier.weight(1f),
                                            ) {
                                                Text("取消收藏")
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
    }
}
