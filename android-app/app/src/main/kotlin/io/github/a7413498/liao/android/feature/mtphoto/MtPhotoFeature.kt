/*
 * mtPhoto 页面先对齐 Web 端最小浏览主流程：相册列表、目录列表、基础缩略图浏览与时间线延迟加载。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.mtphoto

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
import androidx.compose.material3.AlertDialog
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import coil.compose.AsyncImage
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import io.github.a7413498.liao.android.core.network.SystemApiService
import javax.inject.Inject
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive

private const val DEFAULT_TIMELINE_THRESHOLD = 10

enum class MtPhotoMode {
    ALBUMS,
    FOLDERS,
}

data class MtPhotoAlbumSummary(
    val id: Long,
    val name: String,
    val coverMd5: String,
    val coverUrl: String,
    val count: Int,
)

data class MtPhotoFolderSummary(
    val id: Long,
    val name: String,
    val path: String,
    val coverMd5: String,
    val coverUrl: String,
    val subFolderNum: Int,
    val subFileNum: Int,
)

data class MtPhotoMediaSummary(
    val id: Long,
    val md5: String,
    val type: ChatMessageType,
    val title: String,
    val subtitle: String,
    val thumbUrl: String,
)

data class ImportedMtPhotoMedia(
    val localPath: String,
    val localFilename: String,
    val dedup: Boolean,
)

data class MtPhotoFolderHistoryItem(
    val folderId: Long?,
    val folderName: String,
    val folderPath: String = "",
    val coverMd5: String = "",
    val subFolderNum: Int = 0,
)

data class MtPhotoAlbumPage(
    val items: List<MtPhotoMediaSummary>,
    val total: Int,
    val page: Int,
    val totalPages: Int,
)

data class MtPhotoFolderPage(
    val folderName: String,
    val folderPath: String,
    val folders: List<MtPhotoFolderSummary>,
    val files: List<MtPhotoMediaSummary>,
    val total: Int,
    val page: Int,
    val totalPages: Int,
)

class MtPhotoRepository @Inject constructor(
    private val mtPhotoApiService: MtPhotoApiService,
    private val systemApiService: SystemApiService,
    private val baseUrlProvider: BaseUrlProvider,
    private val preferencesStore: AppPreferencesStore,
) {
    suspend fun loadAlbums(): AppResult<List<MtPhotoAlbumSummary>> = runCatching {
        val root = mtPhotoApiService.getAlbums() as? JsonObject ?: error("mtPhoto 相册响应格式异常")
        root.errorMessage()?.let(::error)
        root["data"]?.jsonArray.orEmpty().mapNotNull { item -> item.toAlbumSummary() }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载 mtPhoto 相册失败", it) },
    )

    suspend fun loadAlbumFiles(albumId: Long, page: Int, pageSize: Int = 60): AppResult<MtPhotoAlbumPage> = runCatching {
        val root = mtPhotoApiService.getAlbumFiles(albumId = albumId, page = page, pageSize = pageSize) as? JsonObject
            ?: error("mtPhoto 相册媒体响应格式异常")
        root.errorMessage()?.let(::error)
        MtPhotoAlbumPage(
            items = root["data"]?.jsonArray.orEmpty().mapNotNull { item -> item.toMediaSummary() },
            total = root.intOrDefault("total", 0),
            page = root.intOrDefault("page", page),
            totalPages = root.intOrDefault("totalPages", 0),
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载 mtPhoto 相册媒体失败", it) },
    )

    suspend fun loadFolderRoot(): AppResult<MtPhotoFolderPage> = runCatching {
        val root = mtPhotoApiService.getFolderRoot() as? JsonObject ?: error("mtPhoto 根目录响应格式异常")
        root.errorMessage()?.let(::error)
        MtPhotoFolderPage(
            folderName = "根目录",
            folderPath = root.stringOrNull("path").orEmpty(),
            folders = root["folderList"]?.jsonArray.orEmpty().mapNotNull { item -> item.toFolderSummary() },
            files = root["fileList"]?.jsonArray.orEmpty().mapNotNull { item -> item.toMediaSummary() },
            total = root.intOrDefault("total", 0),
            page = root.intOrDefault("page", 1),
            totalPages = root.intOrDefault("totalPages", 0),
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载 mtPhoto 根目录失败", it) },
    )

    suspend fun loadFolderContent(
        folderId: Long,
        page: Int,
        includeTimeline: Boolean,
        pageSize: Int = 60,
    ): AppResult<MtPhotoFolderPage> = runCatching {
        val root = mtPhotoApiService.getFolderContent(
            folderId = folderId,
            page = page,
            pageSize = pageSize,
            includeTimeline = includeTimeline,
        ) as? JsonObject
            ?: error("mtPhoto 目录响应格式异常")
        root.errorMessage()?.let(::error)
        MtPhotoFolderPage(
            folderName = root.stringOrNull("path")?.toFolderName("目录 $folderId") ?: "目录 $folderId",
            folderPath = root.stringOrNull("path").orEmpty(),
            folders = root["folderList"]?.jsonArray.orEmpty().mapNotNull { item -> item.toFolderSummary() },
            files = root["fileList"]?.jsonArray.orEmpty().mapNotNull { item -> item.toMediaSummary() },
            total = root.intOrDefault("total", 0),
            page = root.intOrDefault("page", page),
            totalPages = root.intOrDefault("totalPages", 0),
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: if (includeTimeline) "加载 mtPhoto 时间线失败" else "加载 mtPhoto 目录失败", it) },
    )

    suspend fun loadTimelineThreshold(): Int = runCatching {
        val remoteConfig = systemApiService.getSystemConfig().data
        if (remoteConfig != null) {
            preferencesStore.saveCachedSystemConfig(remoteConfig)
        }
        remoteConfig?.mtPhotoTimelineDeferSubfolderThreshold
            ?.takeIf { it > 0 }
            ?.coerceAtMost(500)
            ?: preferencesStore.readCachedSystemConfig()?.mtPhotoTimelineDeferSubfolderThreshold
                ?.takeIf { it > 0 }
                ?.coerceAtMost(500)
            ?: DEFAULT_TIMELINE_THRESHOLD
    }.getOrElse {
        preferencesStore.readCachedSystemConfig()?.mtPhotoTimelineDeferSubfolderThreshold
            ?.takeIf { it > 0 }
            ?.coerceAtMost(500)
            ?: DEFAULT_TIMELINE_THRESHOLD
    }

    suspend fun importMedia(md5: String): AppResult<ImportedMtPhotoMedia> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份后再导入")
        val root = mtPhotoApiService.importMtPhotoMedia(userId = session.id, md5 = md5) as? JsonObject
            ?: error("mtPhoto 导入响应格式异常")
        val state = root.stringOrNull("state").orEmpty()
        if (!state.equals("OK", ignoreCase = true)) {
            error(root.stringOrNull("error") ?: root.stringOrNull("msg") ?: "导入失败")
        }
        val localPath = root.stringOrNull("localPath") ?: error("导入结果缺少 localPath")
        ImportedMtPhotoMedia(
            localPath = localPath,
            localFilename = root.stringOrNull("localFilename").orEmpty(),
            dedup = root.booleanOrFalse("dedup"),
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "导入 mtPhoto 媒体失败", it) },
    )

    private fun JsonElement.toAlbumSummary(): MtPhotoAlbumSummary? {
        val root = this as? JsonObject ?: return null
        val id = root.longOrNull("id") ?: return null
        val name = root.stringOrNull("name") ?: "相册 $id"
        val coverMd5 = root.stringOrNull("cover").orEmpty()
        return MtPhotoAlbumSummary(
            id = id,
            name = name,
            coverMd5 = coverMd5,
            coverUrl = coverMd5.takeIf { it.isNotBlank() }?.let { thumbUrl(md5 = it, size = "s260") }.orEmpty(),
            count = root.intOrDefault("count", 0),
        )
    }

    private fun JsonElement.toFolderSummary(): MtPhotoFolderSummary? {
        val root = this as? JsonObject ?: return null
        val id = root.longOrNull("id") ?: return null
        val coverMd5 = firstCoverMd5(root.stringOrNull("cover"), root.stringOrNull("s_cover"))
        return MtPhotoFolderSummary(
            id = id,
            name = root.stringOrNull("name") ?: "目录 $id",
            path = root.stringOrNull("path").orEmpty(),
            coverMd5 = coverMd5,
            coverUrl = coverMd5.takeIf { it.isNotBlank() }?.let { thumbUrl(md5 = it, size = "s260") }.orEmpty(),
            subFolderNum = root.intOrDefault("subFolderNum", 0),
            subFileNum = root.intOrDefault("subFileNum", 0),
        )
    }

    private fun JsonElement.toMediaSummary(): MtPhotoMediaSummary? {
        val root = this as? JsonObject ?: return null
        val md5 = root.stringOrNull("md5") ?: root.stringOrNull("MD5") ?: return null
        val fileType = root.stringOrNull("fileType").orEmpty()
        val type = if (fileType.equals("MP4", ignoreCase = true)) ChatMessageType.VIDEO else ChatMessageType.IMAGE
        val day = root.stringOrNull("day").orEmpty()
        val width = root.intOrDefault("width", 0)
        val height = root.intOrDefault("height", 0)
        val duration = root.stringOrNull("duration").orEmpty()
        val title = root.stringOrNull("fileName")?.takeIf { it.isNotBlank() }
            ?: root.stringOrNull("tokenAt")?.takeIf { it.isNotBlank() }
            ?: md5.take(12)
        val subtitle = buildList {
            if (day.isNotBlank()) add(day)
            if (width > 0 && height > 0) add("${width}×${height}")
            if (type == ChatMessageType.VIDEO && duration.isNotBlank()) add("${duration}s")
        }.joinToString(separator = " · ")
        return MtPhotoMediaSummary(
            id = root.longOrNull("id") ?: 0L,
            md5 = md5,
            type = type,
            title = title,
            subtitle = subtitle,
            thumbUrl = thumbUrl(md5 = md5, size = if (type == ChatMessageType.IMAGE) "h220" else "s260"),
        )
    }

    private fun thumbUrl(md5: String, size: String): String {
        val apiBaseUrl = baseUrlProvider.currentApiBaseUrl()
        val isDefaultPort = (apiBaseUrl.isHttps && apiBaseUrl.port == 443) || (!apiBaseUrl.isHttps && apiBaseUrl.port == 80)
        val portSuffix = if (isDefaultPort) "" else ":${apiBaseUrl.port}"
        val origin = "${apiBaseUrl.scheme}://${apiBaseUrl.host}$portSuffix"
        return "$origin/api/getMtPhotoThumb?size=$size&md5=$md5"
    }
}

data class MtPhotoUiState(
    val loading: Boolean = true,
    val loadingMore: Boolean = false,
    val mode: MtPhotoMode = MtPhotoMode.ALBUMS,
    val albumsLoaded: Boolean = false,
    val foldersLoaded: Boolean = false,
    val albums: List<MtPhotoAlbumSummary> = emptyList(),
    val selectedAlbum: MtPhotoAlbumSummary? = null,
    val albumItems: List<MtPhotoMediaSummary> = emptyList(),
    val albumPage: Int = 0,
    val albumTotal: Int = 0,
    val albumTotalPages: Int = 0,
    val folderHistory: List<MtPhotoFolderHistoryItem> = listOf(MtPhotoFolderHistoryItem(folderId = null, folderName = "根目录")),
    val currentFolders: List<MtPhotoFolderSummary> = emptyList(),
    val folderItems: List<MtPhotoMediaSummary> = emptyList(),
    val folderPage: Int = 0,
    val folderTotal: Int = 0,
    val folderTotalPages: Int = 0,
    val folderTimelineDeferred: Boolean = false,
    val timelineThreshold: Int = DEFAULT_TIMELINE_THRESHOLD,
    val folderFavoritesLoading: Boolean = false,
    val folderFavoriteSaving: Boolean = false,
    val folderFavorites: List<MtPhotoFolderFavorite> = emptyList(),
    val previewItem: MtPhotoMediaSummary? = null,
    val importingPreview: Boolean = false,
    val message: String? = null,
)

@HiltViewModel
class MtPhotoViewModel @Inject constructor(
    private val repository: MtPhotoRepository,
    private val folderFavoritesRepository: MtPhotoFolderFavoritesRepository,
) : ViewModel() {
    var uiState by mutableStateOf(MtPhotoUiState())
        private set

    init {
        loadAlbums()
    }

    fun switchMode(mode: MtPhotoMode) {
        if (uiState.mode == mode) return
        uiState = uiState.copy(mode = mode, previewItem = null)
        when (mode) {
            MtPhotoMode.ALBUMS -> if (!uiState.albumsLoaded) loadAlbums()
            MtPhotoMode.FOLDERS -> {
                if (!uiState.foldersLoaded) loadFolderRoot()
                loadFolderFavorites(silent = uiState.folderFavorites.isNotEmpty())
            }
        }
    }

    fun loadAlbums() {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, message = null)
            when (val result = repository.loadAlbums()) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        loading = false,
                        albumsLoaded = true,
                        albums = result.data,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, message = result.message)
            }
        }
    }

    fun openAlbum(album: MtPhotoAlbumSummary) {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, selectedAlbum = album, message = null, previewItem = null)
            when (val result = repository.loadAlbumFiles(album.id, page = 1)) {
                is AppResult.Success -> {
                    val page = result.data
                    uiState = uiState.copy(
                        loading = false,
                        selectedAlbum = album,
                        albumItems = page.items,
                        albumPage = page.page,
                        albumTotal = page.total,
                        albumTotalPages = page.totalPages,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, message = result.message)
            }
        }
    }

    fun backToAlbums() {
        uiState = uiState.copy(
            selectedAlbum = null,
            albumItems = emptyList(),
            albumPage = 0,
            albumTotal = 0,
            albumTotalPages = 0,
            previewItem = null,
        )
    }

    fun loadMoreAlbum() {
        val state = uiState
        val album = state.selectedAlbum ?: return
        if (state.loadingMore || state.albumPage >= state.albumTotalPages) return
        viewModelScope.launch {
            uiState = uiState.copy(loadingMore = true)
            when (val result = repository.loadAlbumFiles(album.id, page = state.albumPage + 1)) {
                is AppResult.Success -> {
                    val page = result.data
                    uiState = uiState.copy(
                        loadingMore = false,
                        albumItems = state.albumItems + page.items,
                        albumPage = page.page,
                        albumTotal = page.total,
                        albumTotalPages = page.totalPages,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loadingMore = false, message = result.message)
            }
        }
    }

    fun loadFolderRoot() {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, message = null, previewItem = null)
            when (val result = repository.loadFolderRoot()) {
                is AppResult.Success -> {
                    val root = result.data
                    uiState = uiState.copy(
                        loading = false,
                        foldersLoaded = true,
                        folderHistory = listOf(MtPhotoFolderHistoryItem(folderId = null, folderName = root.folderName, folderPath = root.folderPath)),
                        currentFolders = root.folders,
                        folderItems = root.files,
                        folderPage = root.page,
                        folderTotal = root.total,
                        folderTotalPages = root.totalPages,
                        folderTimelineDeferred = false,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, message = result.message)
            }
        }
    }

    fun openFolder(folder: MtPhotoFolderSummary) {
        val nextHistory = uiState.folderHistory + MtPhotoFolderHistoryItem(
            folderId = folder.id,
            folderName = folder.name,
            folderPath = folder.path,
            coverMd5 = folder.coverMd5,
            subFolderNum = folder.subFolderNum,
        )
        loadFolderByHistory(nextHistory)
    }

    fun backFolder() {
        val history = uiState.folderHistory
        if (history.size <= 1) {
            loadFolderRoot()
            return
        }
        loadFolderByHistory(history.dropLast(1))
    }

    fun loadFolderTimeline() {
        val current = uiState.folderHistory.lastOrNull() ?: return
        val folderId = current.folderId ?: return
        loadFolderByHistory(uiState.folderHistory, forceTimeline = true, explicitFolderId = folderId, explicitFolderName = current.folderName)
    }

    fun loadFolderFavorites(silent: Boolean = false) {
        if (uiState.folderFavoritesLoading) return
        viewModelScope.launch {
            if (!silent) {
                uiState = uiState.copy(folderFavoritesLoading = true, message = null)
            } else {
                uiState = uiState.copy(folderFavoritesLoading = true)
            }
            when (val result = folderFavoritesRepository.loadFavorites()) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        folderFavoritesLoading = false,
                        folderFavorites = result.data,
                    )
                }
                is AppResult.Error -> {
                    uiState = uiState.copy(
                        folderFavoritesLoading = false,
                        message = if (silent) uiState.message else result.message,
                    )
                }
            }
        }
    }

    fun saveCurrentFolderFavorite() {
        val current = uiState.folderHistory.lastOrNull() ?: return
        val folderId = current.folderId ?: run {
            uiState = uiState.copy(message = "根目录暂不支持收藏")
            return
        }
        if (uiState.folderFavoriteSaving) return
        viewModelScope.launch {
            uiState = uiState.copy(folderFavoriteSaving = true, message = null)
            when (
                val result = folderFavoritesRepository.upsertFavorite(
                    folderId = folderId,
                    folderName = current.folderName.ifBlank { "目录 $folderId" },
                    folderPath = current.folderPath,
                    coverMd5 = current.coverMd5,
                )
            ) {
                is AppResult.Success -> {
                    val favorite = result.data
                    val nextFavorites = listOf(favorite) + uiState.folderFavorites.filterNot { it.folderId == favorite.folderId }
                    uiState = uiState.copy(
                        folderFavoriteSaving = false,
                        folderFavorites = nextFavorites,
                        message = "目录收藏已保存",
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(folderFavoriteSaving = false, message = result.message)
            }
        }
    }

    fun removeCurrentFolderFavorite() {
        val folderId = uiState.folderHistory.lastOrNull()?.folderId ?: return
        removeFavoriteFolder(folderId)
    }

    fun removeFavoriteFolder(folderId: Long) {
        if (folderId <= 0 || uiState.folderFavoriteSaving) return
        viewModelScope.launch {
            uiState = uiState.copy(folderFavoriteSaving = true, message = null)
            when (val result = folderFavoritesRepository.removeFavorite(folderId)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        folderFavoriteSaving = false,
                        folderFavorites = uiState.folderFavorites.filterNot { it.folderId == folderId },
                        message = "已取消目录收藏",
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(folderFavoriteSaving = false, message = result.message)
            }
        }
    }

    fun openFavoriteFolder(favorite: MtPhotoFolderFavorite) {
        openFolderById(
            folderId = favorite.folderId,
            folderName = favorite.folderName,
            folderPath = favorite.folderPath,
            coverMd5 = favorite.coverMd5,
        )
    }

    fun openFolderById(
        folderId: Long,
        folderName: String,
        folderPath: String = "",
        coverMd5: String = "",
    ) {
        if (folderId <= 0) return
        val history = listOf(
            MtPhotoFolderHistoryItem(folderId = null, folderName = "根目录"),
            MtPhotoFolderHistoryItem(
                folderId = folderId,
                folderName = folderName.ifBlank { "目录 $folderId" },
                folderPath = folderPath,
                coverMd5 = coverMd5,
            ),
        )
        if (uiState.mode != MtPhotoMode.FOLDERS) {
            uiState = uiState.copy(mode = MtPhotoMode.FOLDERS, previewItem = null)
        }
        if (uiState.folderFavorites.isEmpty()) {
            loadFolderFavorites(silent = false)
        }
        loadFolderByHistory(history, explicitFolderId = folderId, explicitFolderName = folderName)
    }

    fun loadMoreFolder() {
        val state = uiState
        val current = state.folderHistory.lastOrNull() ?: return
        val folderId = current.folderId ?: return
        if (state.loadingMore || state.folderTimelineDeferred || state.folderPage >= state.folderTotalPages) return
        viewModelScope.launch {
            uiState = uiState.copy(loadingMore = true)
            when (val result = repository.loadFolderContent(folderId = folderId, page = state.folderPage + 1, includeTimeline = true)) {
                is AppResult.Success -> {
                    val page = result.data
                    uiState = uiState.copy(
                        loadingMore = false,
                        currentFolders = page.folders,
                        folderItems = state.folderItems + page.files,
                        folderPage = page.page,
                        folderTotal = page.total,
                        folderTotalPages = page.totalPages,
                        folderTimelineDeferred = false,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loadingMore = false, message = result.message)
            }
        }
    }

    private fun currentFolderFavorite(history: List<MtPhotoFolderHistoryItem> = uiState.folderHistory): MtPhotoFolderFavorite? {
        val folderId = history.lastOrNull()?.folderId ?: return null
        return uiState.folderFavorites.firstOrNull { it.folderId == folderId }
    }

    private fun resolvedFolderHistory(
        history: List<MtPhotoFolderHistoryItem>,
        page: MtPhotoFolderPage,
        explicitFolderName: String? = null,
    ): List<MtPhotoFolderHistoryItem> = history.mapIndexed { index, item ->
        if (index == history.lastIndex) {
            item.copy(
                folderName = explicitFolderName ?: item.folderName.ifBlank { page.folderName },
                folderPath = page.folderPath.ifBlank { item.folderPath },
            )
        } else {
            item
        }
    }

    private fun loadFolderByHistory(
        history: List<MtPhotoFolderHistoryItem>,
        forceTimeline: Boolean = false,
        explicitFolderId: Long? = null,
        explicitFolderName: String? = null,
    ) {
        val target = history.lastOrNull() ?: MtPhotoFolderHistoryItem(folderId = null, folderName = "根目录")
        val folderId = explicitFolderId ?: target.folderId
        if (folderId == null) {
            loadFolderRoot()
            return
        }
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, message = null, previewItem = null)
            val threshold = repository.loadTimelineThreshold()
            when (val detailResult = repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = false)) {
                is AppResult.Success -> {
                    val detailPage = detailResult.data
                    val resolvedHistory = resolvedFolderHistory(history, detailPage, explicitFolderName)
                    val folderCount = detailPage.folders.size
                    val shouldDeferTimeline = !forceTimeline && folderCount > threshold
                    if (shouldDeferTimeline) {
                        uiState = uiState.copy(
                            loading = false,
                            foldersLoaded = true,
                            timelineThreshold = threshold,
                            folderHistory = resolvedHistory,
                            currentFolders = detailPage.folders,
                            folderItems = emptyList(),
                            folderPage = 0,
                            folderTotal = detailPage.total,
                            folderTotalPages = detailPage.totalPages,
                            folderTimelineDeferred = true,
                        )
                        return@launch
                    }
                    when (val timelineResult = repository.loadFolderContent(folderId = folderId, page = 1, includeTimeline = true)) {
                        is AppResult.Success -> {
                            val page = timelineResult.data
                            uiState = uiState.copy(
                                loading = false,
                                foldersLoaded = true,
                                timelineThreshold = threshold,
                                folderHistory = resolvedFolderHistory(resolvedHistory, page, explicitFolderName),
                                currentFolders = page.folders,
                                folderItems = page.files,
                                folderPage = page.page,
                                folderTotal = page.total,
                                folderTotalPages = page.totalPages,
                                folderTimelineDeferred = false,
                            )
                        }
                        is AppResult.Error -> {
                            uiState = uiState.copy(
                                loading = false,
                                foldersLoaded = true,
                                timelineThreshold = threshold,
                                folderHistory = resolvedHistory,
                                currentFolders = detailPage.folders,
                                folderItems = detailPage.files,
                                folderPage = detailPage.page,
                                folderTotal = detailPage.total,
                                folderTotalPages = detailPage.totalPages,
                                folderTimelineDeferred = false,
                                message = timelineResult.message,
                            )
                        }
                    }
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, message = detailResult.message)
            }
        }
    }

    fun refreshCurrent() {
        when (uiState.mode) {
            MtPhotoMode.ALBUMS -> {
                val album = uiState.selectedAlbum
                if (album == null) loadAlbums() else openAlbum(album)
            }
            MtPhotoMode.FOLDERS -> {
                loadFolderFavorites(silent = true)
                val current = uiState.folderHistory.lastOrNull()
                if (current?.folderId == null) loadFolderRoot() else loadFolderByHistory(uiState.folderHistory)
            }
        }
    }

    fun showPreview(item: MtPhotoMediaSummary) {
        uiState = uiState.copy(previewItem = item)
    }

    fun dismissPreview() {
        uiState = uiState.copy(previewItem = null, importingPreview = false)
    }

    fun importPreview(onImported: ((ImportedMtPhotoMedia) -> Unit)? = null) {
        val preview = uiState.previewItem ?: return
        if (uiState.importingPreview) return
        viewModelScope.launch {
            uiState = uiState.copy(importingPreview = true, message = null)
            when (val result = repository.importMedia(preview.md5)) {
                is AppResult.Success -> {
                    val imported = result.data
                    uiState = uiState.copy(
                        importingPreview = false,
                        previewItem = if (onImported != null) null else uiState.previewItem,
                        message = if (imported.dedup) "已存在（去重复用）" else "已导入到本地媒体库",
                    )
                    onImported?.invoke(imported)
                }
                is AppResult.Error -> uiState = uiState.copy(importingPreview = false, message = result.message)
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
fun MtPhotoScreen(
    viewModel: MtPhotoViewModel,
    onBack: () -> Unit,
    onImportToChat: ((ImportedMtPhotoMedia) -> Unit)? = null,
    initialFolderId: Long? = null,
    initialFolderName: String = "",
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }
    var initialFolderHandled by remember(initialFolderId, initialFolderName) { mutableStateOf(false) }
    val currentFolderFavorite = state.folderHistory.lastOrNull()?.folderId?.let { folderId ->
        state.folderFavorites.firstOrNull { it.folderId == folderId }
    }

    LaunchedEffect(initialFolderId, initialFolderName, initialFolderHandled) {
        val folderId = initialFolderId
        if (!initialFolderHandled && folderId != null && folderId > 0) {
            initialFolderHandled = true
            viewModel.openFolderById(folderId = folderId, folderName = initialFolderName)
        }
    }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    state.previewItem?.let { preview ->
        AlertDialog(
            onDismissRequest = viewModel::dismissPreview,
            confirmButton = {
                Button(onClick = { viewModel.importPreview(onImportToChat) }, enabled = !state.importingPreview) {
                    Text(
                        if (state.importingPreview) {
                            "导入中..."
                        } else if (onImportToChat != null) {
                            "导入到会话"
                        } else {
                            "导入到本地"
                        }
                    )
                }
            },
            dismissButton = {
                OutlinedButton(onClick = viewModel::dismissPreview, enabled = !state.importingPreview) {
                    Text("关闭")
                }
            },
            title = { Text(preview.title) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    AsyncImage(
                        model = preview.thumbUrl,
                        contentDescription = preview.title,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(240.dp),
                        contentScale = ContentScale.Fit,
                    )
                    if (preview.subtitle.isNotBlank()) {
                        Text(preview.subtitle, style = MaterialTheme.typography.bodySmall)
                    }
                    Text(
                        text = if (preview.type == ChatMessageType.VIDEO) "当前为基础浏览模式，视频先展示缩略图与元信息。" else "当前为基础浏览模式，可查看图片缩略图与元信息。",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("mtPhoto 相册") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
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
                            text = "基础浏览已对齐：支持相册 / 目录模式切换、缩略图查看、延迟时间线加载。",
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            ToggleButton(
                                label = "相册",
                                selected = state.mode == MtPhotoMode.ALBUMS,
                                modifier = Modifier.weight(1f),
                                onClick = { viewModel.switchMode(MtPhotoMode.ALBUMS) },
                            )
                            ToggleButton(
                                label = "目录",
                                selected = state.mode == MtPhotoMode.FOLDERS,
                                modifier = Modifier.weight(1f),
                                onClick = { viewModel.switchMode(MtPhotoMode.FOLDERS) },
                            )
                        }
                        OutlinedButton(
                            onClick = viewModel::refreshCurrent,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text("刷新当前视图")
                        }
                    }
                }
            }

            if (state.loading && currentVisibleItemCount(state) == 0) {
                item {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(24.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        CircularProgressIndicator()
                    }
                }
            } else {
                when (state.mode) {
                    MtPhotoMode.ALBUMS -> {
                        if (state.selectedAlbum == null) {
                            if (state.albums.isEmpty()) {
                                item { EmptyCard(text = "暂无 mtPhoto 相册") }
                            } else {
                                items(state.albums, key = { it.id }) { album ->
                                    AlbumCard(album = album, onClick = { viewModel.openAlbum(album) })
                                }
                            }
                        } else {
                            item {
                                AlbumHeaderCard(
                                    album = state.selectedAlbum,
                                    itemCount = state.albumItems.size,
                                    total = state.albumTotal,
                                    onBack = viewModel::backToAlbums,
                                )
                            }
                            if (state.albumItems.isEmpty()) {
                                item { EmptyCard(text = "当前相册暂无媒体") }
                            } else {
                                items(state.albumItems, key = { it.md5 }) { item ->
                                    MediaCard(item = item, onClick = { viewModel.showPreview(item) })
                                }
                            }
                            if (state.albumPage < state.albumTotalPages) {
                                item {
                                    LoadMoreCard(
                                        loading = state.loadingMore,
                                        onClick = viewModel::loadMoreAlbum,
                                    )
                                }
                            }
                        }
                    }
                    MtPhotoMode.FOLDERS -> {
                        item {
                            FolderHeaderCard(
                                current = state.folderHistory.lastOrNull(),
                                deferred = state.folderTimelineDeferred,
                                threshold = state.timelineThreshold,
                                onBackFolder = if (state.folderHistory.size > 1) viewModel::backFolder else null,
                                onLoadTimeline = if (state.folderTimelineDeferred) viewModel::loadFolderTimeline else null,
                            )
                        }
                        item {
                            CurrentFolderFavoriteCard(
                                current = state.folderHistory.lastOrNull(),
                                currentFavorite = currentFolderFavorite,
                                loading = state.folderFavoritesLoading,
                                saving = state.folderFavoriteSaving,
                                onRefresh = { viewModel.loadFolderFavorites() },
                                onSave = viewModel::saveCurrentFolderFavorite,
                                onRemove = viewModel::removeCurrentFolderFavorite,
                            )
                        }
                        if (state.folderFavorites.isNotEmpty()) {
                            item {
                                SectionTitle(text = "目录收藏 (${state.folderFavorites.size})")
                            }
                            items(state.folderFavorites, key = { it.folderId }) { favorite ->
                                MtPhotoFolderFavoriteCard(
                                    item = favorite,
                                    onOpen = { viewModel.openFavoriteFolder(favorite) },
                                    onRemove = { viewModel.removeFavoriteFolder(favorite.folderId) },
                                )
                            }
                        } else if (state.folderFavoritesLoading) {
                            item { EmptyCard(text = "正在加载目录收藏...") }
                        }
                        if (state.currentFolders.isEmpty() && state.folderItems.isEmpty() && !state.folderTimelineDeferred) {
                            item { EmptyCard(text = "当前目录暂无内容") }
                        } else {
                            if (state.currentFolders.isNotEmpty()) {
                                item {
                                    SectionTitle(text = "子目录 (${state.currentFolders.size})")
                                }
                                items(state.currentFolders, key = { it.id }) { folder ->
                                    FolderCard(folder = folder, onClick = { viewModel.openFolder(folder) })
                                }
                            }
                            if (state.folderTimelineDeferred) {
                                item {
                                    EmptyCard(text = "当前目录子文件夹较多，已按阈值延迟加载时间线图片。")
                                }
                            } else {
                                if (state.folderItems.isNotEmpty()) {
                                    item {
                                        SectionTitle(text = "媒体 (${state.folderItems.size}/${state.folderTotal.coerceAtLeast(state.folderItems.size)})")
                                    }
                                    items(state.folderItems, key = { it.md5 + "-" + it.id }) { item ->
                                        MediaCard(item = item, onClick = { viewModel.showPreview(item) })
                                    }
                                }
                                if (state.folderPage < state.folderTotalPages) {
                                    item {
                                        LoadMoreCard(
                                            loading = state.loadingMore,
                                            onClick = viewModel::loadMoreFolder,
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

@Composable
private fun ToggleButton(
    label: String,
    selected: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    if (selected) {
        Button(onClick = onClick, modifier = modifier) { Text(label) }
    } else {
        OutlinedButton(onClick = onClick, modifier = modifier) { Text(label) }
    }
}

@Composable
private fun AlbumCard(
    album: MtPhotoAlbumSummary,
    onClick: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp)
            .clickable(onClick = onClick),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            MtPhotoThumb(
                url = album.coverUrl,
                label = "相册",
                modifier = Modifier.size(84.dp),
            )
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(album.name, style = MaterialTheme.typography.titleMedium)
                Text("共 ${album.count} 项", style = MaterialTheme.typography.bodySmall)
            }
            OutlinedButton(onClick = onClick) {
                Text("进入")
            }
        }
    }
}

@Composable
private fun AlbumHeaderCard(
    album: MtPhotoAlbumSummary,
    itemCount: Int,
    total: Int,
    onBack: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(album.name, style = MaterialTheme.typography.titleLarge)
            Text("已加载 $itemCount / ${total.coerceAtLeast(itemCount)} 项", style = MaterialTheme.typography.bodySmall)
            OutlinedButton(onClick = onBack) {
                Text("返回相册列表")
            }
        }
    }
}

@Composable
private fun FolderHeaderCard(
    current: MtPhotoFolderHistoryItem?,
    deferred: Boolean,
    threshold: Int,
    onBackFolder: (() -> Unit)?,
    onLoadTimeline: (() -> Unit)?,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(current?.folderName ?: "根目录", style = MaterialTheme.typography.titleLarge)
            Text("当前时间线延迟阈值：$threshold", style = MaterialTheme.typography.bodySmall)
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                if (onBackFolder != null) {
                    OutlinedButton(onClick = onBackFolder, modifier = Modifier.weight(1f)) {
                        Text("返回上级")
                    }
                }
                if (deferred && onLoadTimeline != null) {
                    Button(onClick = onLoadTimeline, modifier = Modifier.weight(1f)) {
                        Text("加载时间线图片")
                    }
                }
            }
        }
    }
}

@Composable
private fun FolderCard(
    folder: MtPhotoFolderSummary,
    onClick: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp)
            .clickable(onClick = onClick),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            MtPhotoThumb(
                url = folder.coverUrl,
                label = "目录",
                modifier = Modifier.size(72.dp),
            )
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(folder.name, style = MaterialTheme.typography.titleMedium)
                Text(
                    text = "子目录 ${folder.subFolderNum} · 文件 ${folder.subFileNum}",
                    style = MaterialTheme.typography.bodySmall,
                )
                if (folder.path.isNotBlank()) {
                    Text(
                        text = folder.path,
                        style = MaterialTheme.typography.labelSmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            OutlinedButton(onClick = onClick) {
                Text("进入")
            }
        }
    }
}

@Composable
private fun MediaCard(
    item: MtPhotoMediaSummary,
    onClick: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp)
            .clickable(onClick = onClick),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            MtPhotoThumb(
                url = item.thumbUrl,
                label = if (item.type == ChatMessageType.VIDEO) "视频" else "图片",
                modifier = Modifier.size(88.dp),
            )
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = item.title,
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (item.subtitle.isNotBlank()) {
                    Text(
                        text = item.subtitle,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Text(
                    text = if (item.type == ChatMessageType.VIDEO) "视频缩略图" else "图片缩略图",
                    style = MaterialTheme.typography.labelSmall,
                )
            }
            OutlinedButton(onClick = onClick) {
                Text("预览")
            }
        }
    }
}

@Composable
internal fun MtPhotoThumb(
    url: String,
    label: String,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier,
        contentAlignment = Alignment.Center,
    ) {
        if (url.isNotBlank()) {
            AsyncImage(
                model = url,
                contentDescription = label,
                modifier = Modifier.fillMaxSize(),
                contentScale = ContentScale.Crop,
            )
        } else {
            Text(label, style = MaterialTheme.typography.labelMedium)
        }
    }
}

@Composable
private fun LoadMoreCard(
    loading: Boolean,
    onClick: () -> Unit,
) {
    OutlinedButton(
        onClick = onClick,
        enabled = !loading,
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
    ) {
        Text(if (loading) "加载中..." else "加载更多")
    }
}

@Composable
private fun EmptyCard(text: String) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Text(
            text = text,
            modifier = Modifier.padding(16.dp),
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

@Composable
private fun SectionTitle(text: String) {
    Text(
        text = text,
        modifier = Modifier.padding(horizontal = 16.dp),
        style = MaterialTheme.typography.titleMedium,
    )
}

private fun currentVisibleItemCount(state: MtPhotoUiState): Int = when (state.mode) {
    MtPhotoMode.ALBUMS -> if (state.selectedAlbum == null) state.albums.size else state.albumItems.size
    MtPhotoMode.FOLDERS -> state.currentFolders.size + state.folderItems.size
}

private fun JsonObject.stringOrNull(key: String): String? =
    this[key]?.let { runCatching { it.jsonPrimitive.contentOrNull ?: it.jsonPrimitive.content }.getOrNull() }?.takeIf { it.isNotBlank() }

private fun JsonObject.longOrNull(key: String): Long? = stringOrNull(key)?.toLongOrNull()

private fun JsonObject.intOrDefault(key: String, defaultValue: Int): Int = stringOrNull(key)?.toIntOrNull() ?: defaultValue

private fun JsonObject.booleanOrFalse(key: String): Boolean = stringOrNull(key)?.toBooleanStrictOrNull() ?: false

private fun JsonObject.errorMessage(): String? = stringOrNull("error") ?: stringOrNull("msg")?.takeIf { this["data"] == null && this["folderList"] == null }

private fun String.toFolderName(fallback: String): String {
    val normalized = trim().replace('\\', '/')
    if (normalized.isBlank()) return fallback
    return normalized.substringAfterLast('/').ifBlank { fallback }
}

private fun firstCoverMd5(cover: String?, sCover: String?): String {
    val secondary = sCover?.trim().orEmpty()
    if (secondary.isNotBlank()) return secondary
    val primary = cover?.trim().orEmpty()
    if (primary.isBlank()) return ""
    return primary.substringBefore(',').trim()
}
