/*
 * 全局媒体库页面对齐 Web 端“图片管理 / 全局媒体库”的最小主流程：浏览、打开、单删、批量删除。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.media

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
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
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import coil.compose.AsyncImage
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.inferFileName
import io.github.a7413498.liao.android.core.common.inferMessageType
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.datastore.CachedMediaLibraryItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedMediaLibrarySnapshot
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MediaApiService
import java.net.URI
import javax.inject.Inject
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive

data class MediaLibraryItem(
    val url: String,
    val localPath: String,
    val type: ChatMessageType,
    val title: String,
    val subtitle: String,
    val posterUrl: String = "",
    val updateTime: String = "",
    val source: String = "",
)

private const val MAX_CACHED_MEDIA_ITEMS = 120

data class MediaLibraryPage(
    val items: List<MediaLibraryItem>,
    val page: Int,
    val total: Int,
    val totalPages: Int,
    val fromCache: Boolean = false,
)

class MediaLibraryRepository @Inject constructor(
    private val mediaApiService: MediaApiService,
    private val preferencesStore: AppPreferencesStore,
    private val baseUrlProvider: BaseUrlProvider,
) {
    suspend fun loadMedia(page: Int, pageSize: Int = 20): AppResult<MediaLibraryPage> = runCatching {
        val payload = mediaApiService.getAllUploadImages(page = page, pageSize = pageSize)
        val root = payload as? JsonObject ?: error("媒体库响应格式异常")
        val items = root["data"]?.jsonArray.orEmpty().mapNotNull { element ->
            element.toMediaLibraryItem()
        }
        MediaLibraryPage(
            items = items,
            page = root.intOrDefault("page", page),
            total = root.intOrDefault("total", items.size),
            totalPages = root.intOrDefault("totalPages", if (items.isEmpty()) 1 else page),
        ).also { pageData ->
            updateCachedMediaLibrary(requestedPage = page, pageData = pageData)
        }
    }.recoverCatching { throwable ->
        preferencesStore.readCachedMediaLibrary()?.toUiPage(fromCache = true) ?: throw throwable
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载媒体库失败", it) },
    )

    suspend fun deleteMedia(localPaths: List<String>): AppResult<String> = runCatching {
        val normalizedPaths = localPaths
            .map { it.trim() }
            .filter { it.isNotBlank() }
            .distinct()
            .take(50)
        if (normalizedPaths.isEmpty()) error("请选择要删除的媒体")
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val response = mediaApiService.batchDeleteMedia(
            buildJsonObject {
                put("userId", JsonPrimitive(session.id))
                put(
                    "localPaths",
                    JsonArray(normalizedPaths.map { path -> JsonPrimitive(path) }),
                )
            }
        )
        if (response.code != 0) {
            error(response.msg ?: response.message ?: "删除媒体失败")
        }
        removeCachedMedia(normalizedPaths.toSet())
        response.msg ?: response.message ?: "已删除 ${normalizedPaths.size} 项媒体"
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "删除媒体失败", it) },
    )

    private suspend fun updateCachedMediaLibrary(requestedPage: Int, pageData: MediaLibraryPage) {
        val existingItems = if (requestedPage > 1) {
            preferencesStore.readCachedMediaLibrary()?.items.orEmpty().map { it.toUiModel() }
        } else {
            emptyList()
        }
        val mergedItems = (existingItems + pageData.items)
            .distinctBy { it.cacheKey() }
            .take(MAX_CACHED_MEDIA_ITEMS)
        preferencesStore.saveCachedMediaLibrary(
            CachedMediaLibrarySnapshot(
                items = mergedItems.map { it.toSnapshot() },
                page = pageData.page,
                total = pageData.total,
                totalPages = pageData.totalPages,
            )
        )
    }

    private suspend fun removeCachedMedia(localPaths: Set<String>) {
        val snapshot = preferencesStore.readCachedMediaLibrary() ?: return
        val filteredItems = snapshot.items.filterNot { it.localPath in localPaths }
        val removedCount = snapshot.items.size - filteredItems.size
        preferencesStore.saveCachedMediaLibrary(
            snapshot.copy(
                items = filteredItems,
                total = if (removedCount > 0) {
                    (snapshot.total - removedCount).coerceAtLeast(filteredItems.size)
                } else {
                    snapshot.total.coerceAtLeast(filteredItems.size)
                },
                page = if (filteredItems.isEmpty()) 1 else snapshot.page,
                totalPages = if (filteredItems.isEmpty()) 0 else snapshot.totalPages.coerceAtLeast(1),
            )
        )
    }

    private fun CachedMediaLibrarySnapshot.toUiPage(fromCache: Boolean): MediaLibraryPage = MediaLibraryPage(
        items = items.map { it.toUiModel() },
        page = page,
        total = total,
        totalPages = totalPages,
        fromCache = fromCache,
    )

    private fun CachedMediaLibraryItemSnapshot.toUiModel(): MediaLibraryItem = MediaLibraryItem(
        url = url,
        localPath = localPath,
        type = type.toChatMessageType(url),
        title = title,
        subtitle = subtitle,
        posterUrl = posterUrl,
        updateTime = updateTime,
        source = source,
    )

    private fun MediaLibraryItem.toSnapshot(): CachedMediaLibraryItemSnapshot = CachedMediaLibraryItemSnapshot(
        url = url,
        localPath = localPath,
        type = type.name,
        title = title,
        subtitle = subtitle,
        posterUrl = posterUrl,
        updateTime = updateTime,
        source = source,
    )

    private fun MediaLibraryItem.cacheKey(): String = localPath.ifBlank { url }

    private fun JsonElement.toMediaLibraryItem(): MediaLibraryItem? {
        val root = this as? JsonObject ?: return null
        val url = root.stringOrNull("url")?.let(::normalizeUrl).orEmpty()
        if (url.isBlank()) return null
        val title = root.stringOrNull("originalFilename")
            ?.takeIf { it.isNotBlank() }
            ?: root.stringOrNull("localFilename")
            ?: inferFileName(url)
        val type = root.stringOrNull("type").toChatMessageType(url)
        val updateTime = root.stringOrNull("updateTime").orEmpty()
        val source = root.stringOrNull("source").orEmpty()
        val sizeLabel = root.longOrNull("fileSize")?.takeIf { it > 0 }?.let(::formatFileSize).orEmpty()
        val subtitle = listOf(source.takeIf { it.isNotBlank() }, updateTime.takeIf { it.isNotBlank() }, sizeLabel.takeIf { it.isNotBlank() })
            .joinToString(separator = " · ")
        return MediaLibraryItem(
            url = url,
            localPath = extractUploadLocalPath(url),
            type = type,
            title = title,
            subtitle = subtitle,
            posterUrl = root.stringOrNull("posterUrl")?.let(::normalizeUrl).orEmpty(),
            updateTime = updateTime,
            source = source,
        )
    }

    private fun normalizeUrl(raw: String): String {
        val value = raw.trim()
        if (value.isBlank()) return ""
        if (value.startsWith("http://") || value.startsWith("https://")) return value
        val origin = currentApiOrigin()
        return when {
            value.startsWith("/upload/") -> origin + value
            value.startsWith("/") -> origin + value
            else -> "$origin/upload/$value"
        }
    }

    private fun currentApiOrigin(): String {
        val apiBaseUrl = baseUrlProvider.currentApiBaseUrl()
        val isDefaultPort = (apiBaseUrl.isHttps && apiBaseUrl.port == 443) || (!apiBaseUrl.isHttps && apiBaseUrl.port == 80)
        val portSuffix = if (isDefaultPort) "" else ":${apiBaseUrl.port}"
        return "${apiBaseUrl.scheme}://${apiBaseUrl.host}$portSuffix"
    }
}

data class MediaLibraryUiState(
    val loading: Boolean = true,
    val loadingMore: Boolean = false,
    val deleting: Boolean = false,
    val selectionMode: Boolean = false,
    val items: List<MediaLibraryItem> = emptyList(),
    val selectedLocalPaths: Set<String> = emptySet(),
    val page: Int = 0,
    val total: Int = 0,
    val totalPages: Int = 0,
    val sameMediaVisible: Boolean = false,
    val sameMediaLoading: Boolean = false,
    val sameMediaSourceTitle: String = "",
    val sameMediaSourceLocalPath: String = "",
    val sameMediaItems: List<MtPhotoSameMediaItem> = emptyList(),
    val sameMediaError: String? = null,
    val message: String? = null,
)

@HiltViewModel
class MediaLibraryViewModel @Inject constructor(
    private val repository: MediaLibraryRepository,
    private val mtPhotoSameMediaRepository: MtPhotoSameMediaRepository,
) : ViewModel() {
    var uiState by mutableStateOf(MediaLibraryUiState())
        private set

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, message = null, selectedLocalPaths = emptySet())
            when (val result = repository.loadMedia(page = 1)) {
                is AppResult.Success -> {
                    val page = result.data
                    uiState = uiState.copy(
                        loading = false,
                        items = page.items,
                        page = page.page,
                        total = page.total,
                        totalPages = page.totalPages,
                        message = if (page.fromCache) "网络不可用，已展示最近缓存的媒体库" else null,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, message = result.message)
            }
        }
    }

    fun loadMore() {
        val state = uiState
        if (state.loading || state.loadingMore || state.page >= state.totalPages) return
        viewModelScope.launch {
            uiState = uiState.copy(loadingMore = true)
            when (val result = repository.loadMedia(page = state.page + 1)) {
                is AppResult.Success -> {
                    val page = result.data
                    uiState = uiState.copy(
                        loadingMore = false,
                        items = (state.items + page.items).distinctBy { item -> item.localPath.ifBlank { item.url } },
                        page = page.page,
                        total = page.total,
                        totalPages = page.totalPages,
                        message = if (page.fromCache) "网络不可用，已展示最近缓存的媒体库" else state.message,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loadingMore = false, message = result.message)
            }
        }
    }

    fun toggleSelectionMode() {
        val next = !uiState.selectionMode
        uiState = uiState.copy(selectionMode = next, selectedLocalPaths = if (next) uiState.selectedLocalPaths else emptySet())
    }

    fun toggleSelection(localPath: String) {
        if (localPath.isBlank()) return
        val next = uiState.selectedLocalPaths.toMutableSet()
        if (!next.add(localPath)) {
            next.remove(localPath)
        }
        uiState = uiState.copy(selectedLocalPaths = next)
    }

    fun deleteSingle(localPath: String) {
        deleteInternal(listOf(localPath))
    }

    fun deleteSelected() {
        deleteInternal(uiState.selectedLocalPaths.toList())
    }

    private fun deleteInternal(localPaths: List<String>) {
        if (uiState.deleting) return
        viewModelScope.launch {
            uiState = uiState.copy(deleting = true, message = null)
            when (val result = repository.deleteMedia(localPaths)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        deleting = false,
                        message = result.data,
                        selectedLocalPaths = emptySet(),
                        selectionMode = false,
                    )
                    refresh()
                }
                is AppResult.Error -> uiState = uiState.copy(deleting = false, message = result.message)
            }
        }
    }

    fun lookupMtPhotoSameMedia(item: MediaLibraryItem) {
        val localPath = item.localPath.trim()
        if (localPath.isBlank()) {
            uiState = uiState.copy(message = "当前媒体缺少本地路径，无法查询同媒体")
            return
        }
        if (uiState.sameMediaLoading) return
        viewModelScope.launch {
            uiState = uiState.copy(
                sameMediaVisible = true,
                sameMediaLoading = true,
                sameMediaSourceTitle = item.title,
                sameMediaSourceLocalPath = localPath,
                sameMediaItems = emptyList(),
                sameMediaError = null,
            )
            when (val result = mtPhotoSameMediaRepository.queryByLocalPath(localPath)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        sameMediaLoading = false,
                        sameMediaItems = result.data,
                        sameMediaError = null,
                    )
                }
                is AppResult.Error -> {
                    uiState = uiState.copy(
                        sameMediaLoading = false,
                        sameMediaItems = emptyList(),
                        sameMediaError = result.message,
                    )
                }
            }
        }
    }

    fun dismissMtPhotoSameMedia() {
        if (!uiState.sameMediaVisible && uiState.sameMediaSourceLocalPath.isBlank() && uiState.sameMediaItems.isEmpty()) return
        uiState = uiState.copy(
            sameMediaVisible = false,
            sameMediaLoading = false,
            sameMediaSourceTitle = "",
            sameMediaSourceLocalPath = "",
            sameMediaItems = emptyList(),
            sameMediaError = null,
        )
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }
}

@Composable
fun MediaLibraryScreen(
    viewModel: MediaLibraryViewModel,
    onBack: () -> Unit,
    onOpenMtPhotoFolder: (Long, String) -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }
    val context = LocalContext.current

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    MtPhotoSameMediaDialog(
        visible = state.sameMediaVisible,
        loading = state.sameMediaLoading,
        sourceTitle = state.sameMediaSourceTitle,
        error = state.sameMediaError,
        items = state.sameMediaItems,
        onDismiss = viewModel::dismissMtPhotoSameMedia,
        onRetry = {
            val retryItem = state.items.firstOrNull { it.localPath == state.sameMediaSourceLocalPath }
            if (retryItem != null) {
                viewModel.lookupMtPhotoSameMedia(retryItem)
            }
        },
        onOpenFolder = { item ->
            viewModel.dismissMtPhotoSameMedia()
            onOpenMtPhotoFolder(item.folderId, item.folderName.ifBlank { "目录 ${item.folderId}" })
        },
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("图片管理") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        if (state.loading && state.items.isEmpty()) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                CircularProgressIndicator()
            }
            return@Scaffold
        }

        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 4.dp),
                ) {
                    Column(
                        modifier = Modifier.padding(16.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Text(
                            text = "已加载 ${state.items.size} / ${state.total.coerceAtLeast(state.items.size)} 项上传媒体",
                            style = MaterialTheme.typography.titleMedium,
                        )
                        Text(
                            text = "支持浏览、打开、单删与批量删除；当前先对齐第一页与“加载更多”主流程。",
                            style = MaterialTheme.typography.bodySmall,
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            OutlinedButton(
                                onClick = viewModel::refresh,
                                modifier = Modifier.weight(1f),
                            ) {
                                Text("刷新")
                            }
                            OutlinedButton(
                                onClick = viewModel::toggleSelectionMode,
                                modifier = Modifier.weight(1f),
                            ) {
                                Text(if (state.selectionMode) "退出批量" else "批量管理")
                            }
                        }
                        if (state.selectionMode) {
                            Button(
                                onClick = viewModel::deleteSelected,
                                enabled = state.selectedLocalPaths.isNotEmpty() && !state.deleting,
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Text(
                                    if (state.deleting) {
                                        "删除中..."
                                    } else {
                                        "删除选中 (${state.selectedLocalPaths.size})"
                                    }
                                )
                            }
                        }
                    }
                }
            }

            items(
                items = state.items,
                key = { item -> item.localPath.ifBlank { item.url } },
            ) { item ->
                MediaLibraryCard(
                    item = item,
                    selectionMode = state.selectionMode,
                    selected = state.selectedLocalPaths.contains(item.localPath),
                    deleting = state.deleting,
                    onOpen = { openMediaExternally(context, item.url, item.type) },
                    onDelete = { viewModel.deleteSingle(item.localPath) },
                    onToggleSelection = { viewModel.toggleSelection(item.localPath) },
                    onLookupSameMedia = if (item.localPath.startsWith("/images/") || item.localPath.startsWith("/videos/")) {
                        { viewModel.lookupMtPhotoSameMedia(item) }
                    } else {
                        null
                    },
                )
            }

            if (state.page < state.totalPages) {
                item {
                    OutlinedButton(
                        onClick = viewModel::loadMore,
                        enabled = !state.loadingMore,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp, vertical = 8.dp),
                    ) {
                        Text(if (state.loadingMore) "加载中..." else "加载更多")
                    }
                }
            }
        }
    }
}

@Composable
private fun MediaLibraryCard(
    item: MediaLibraryItem,
    selectionMode: Boolean,
    selected: Boolean,
    deleting: Boolean,
    onOpen: () -> Unit,
    onDelete: () -> Unit,
    onToggleSelection: () -> Unit,
    onLookupSameMedia: (() -> Unit)? = null,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp)
            .clickable(enabled = !selectionMode) { onOpen() },
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            MediaThumbnail(
                item = item,
                modifier = Modifier.size(88.dp),
                onOpen = onOpen,
            )
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    text = item.title,
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = item.type.displayLabel(item.title),
                    style = MaterialTheme.typography.labelMedium,
                )
                if (item.subtitle.isNotBlank()) {
                    Text(
                        text = item.subtitle,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(
                        onClick = onOpen,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text("打开")
                    }
                    if (selectionMode) {
                        Button(
                            onClick = onToggleSelection,
                            enabled = item.localPath.isNotBlank() && !deleting,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(if (selected) "取消" else "选择")
                        }
                    } else {
                        OutlinedButton(
                            onClick = onDelete,
                            enabled = item.localPath.isNotBlank() && !deleting,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text(if (deleting) "删除中..." else "删除")
                        }
                    }
                }
                if (!selectionMode && onLookupSameMedia != null) {
                    OutlinedButton(
                        onClick = onLookupSameMedia,
                        enabled = !deleting,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text("查看 mtPhoto 同媒体")
                    }
                }
            }
        }
    }
}

@Composable
private fun MediaThumbnail(
    item: MediaLibraryItem,
    modifier: Modifier = Modifier,
    onOpen: () -> Unit,
) {
    Box(
        modifier = modifier,
        contentAlignment = Alignment.Center,
    ) {
        when {
            item.type == ChatMessageType.IMAGE && item.url.isNotBlank() -> {
                AsyncImage(
                    model = item.url,
                    contentDescription = item.title,
                    modifier = Modifier
                        .fillMaxSize()
                        .clickable(onClick = onOpen),
                    contentScale = ContentScale.Crop,
                )
            }
            item.type == ChatMessageType.VIDEO && item.posterUrl.isNotBlank() -> {
                AsyncImage(
                    model = item.posterUrl,
                    contentDescription = item.title,
                    modifier = Modifier
                        .fillMaxSize()
                        .clickable(onClick = onOpen),
                    contentScale = ContentScale.Crop,
                )
            }
            else -> {
                Text(
                    text = when (item.type) {
                        ChatMessageType.IMAGE -> "图片"
                        ChatMessageType.VIDEO -> "视频"
                        ChatMessageType.FILE -> "文件"
                        ChatMessageType.TEXT -> "文本"
                    },
                    style = MaterialTheme.typography.labelLarge,
                )
            }
        }
    }
}

internal fun ChatMessageType.openMimeType(): String = when (this) {
    ChatMessageType.IMAGE -> "image/*"
    ChatMessageType.VIDEO -> "video/*"
    ChatMessageType.FILE -> "*/*"
    ChatMessageType.TEXT -> "text/plain"
}

internal fun ChatMessageType.displayLabel(fileName: String): String = when (this) {
    ChatMessageType.IMAGE -> if (fileName.isNotBlank()) "图片 · $fileName" else "图片"
    ChatMessageType.VIDEO -> if (fileName.isNotBlank()) "视频 · $fileName" else "视频"
    ChatMessageType.FILE -> if (fileName.isNotBlank()) "文件 · $fileName" else "文件"
    ChatMessageType.TEXT -> fileName
}

private fun openMediaExternally(context: Context, url: String, type: ChatMessageType) {
    if (url.isBlank()) return
    val uri = Uri.parse(url)
    val typedIntent = Intent(Intent.ACTION_VIEW).apply {
        data = uri
        setDataAndType(uri, type.openMimeType())
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }
    val fallbackIntent = Intent(Intent.ACTION_VIEW, uri).apply {
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }
    runCatching { context.startActivity(typedIntent) }
        .onFailure { runCatching { context.startActivity(fallbackIntent) } }
}

internal fun String?.toChatMessageType(url: String): ChatMessageType = when (this?.trim()?.lowercase()) {
    "image" -> ChatMessageType.IMAGE
    "video" -> ChatMessageType.VIDEO
    "file" -> ChatMessageType.FILE
    else -> inferMessageType("[$url]")
}

private fun JsonObject.stringOrNull(key: String): String? =
    this[key]?.let { runCatching { it.jsonPrimitive.contentOrNull ?: it.jsonPrimitive.content }.getOrNull() }?.takeIf { it.isNotBlank() }

private fun JsonObject.longOrNull(key: String): Long? =
    stringOrNull(key)?.toLongOrNull()

private fun JsonObject.intOrDefault(key: String, defaultValue: Int): Int =
    stringOrNull(key)?.toIntOrNull() ?: defaultValue

internal fun extractUploadLocalPath(rawUrl: String): String {
    val trimmed = rawUrl.trim()
    if (trimmed.isBlank()) return ""
    val path = if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
        runCatching { URI(trimmed).path.orEmpty() }.getOrElse {
            trimmed.substringAfter("://", trimmed).substringAfter('/', "")
        }
    } else {
        trimmed
    }.substringBefore('?').substringBefore('#')

    return when {
        path.startsWith("/upload/") -> "/" + path.removePrefix("/upload/").trimStart('/')
        path.startsWith("upload/") -> "/" + path.removePrefix("upload/").trimStart('/')
        path.startsWith("/") -> path
        else -> "/$path"
    }
}

internal fun formatFileSize(size: Long): String = when {
    size >= 1024 * 1024 * 1024 -> String.format("%.1f GB", size / 1024f / 1024f / 1024f)
    size >= 1024 * 1024 -> String.format("%.1f MB", size / 1024f / 1024f)
    size >= 1024 -> String.format("%.1f KB", size / 1024f)
    else -> "$size B"
}
