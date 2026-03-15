/*
 * 抖音入口页对齐 Web 端最小主流程：作品解析 / 用户作品 / 收藏与标签管理。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.douyin

import android.app.DownloadManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Environment
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
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
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import coil.compose.AsyncImage
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.DouyinApiService
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

enum class DouyinScreenMode {
    DETAIL,
    ACCOUNT,
    FAVORITES,
}

enum class DouyinTagKind {
    USERS,
    AWEMES,
}

data class DouyinMediaItem(
    val index: Int,
    val type: String,
    val url: String,
    val downloadUrl: String,
    val originalFilename: String,
    val thumbUrl: String,
)

data class DouyinDetailResult(
    val key: String,
    val detailId: String,
    val type: String,
    val mediaType: String,
    val title: String,
    val coverUrl: String,
    val imageCount: Int,
    val videoDuration: Double,
    val isLivePhoto: Boolean,
    val livePhotoPairs: Int,
    val items: List<DouyinMediaItem>,
)

data class ImportedDouyinMedia(
    val localPath: String,
    val localFilename: String,
    val dedup: Boolean,
)

enum class DouyinImportStatus {
    IMPORTED,
    EXISTS,
}

data class DouyinAccountItem(
    val detailId: String,
    val type: String,
    val mediaType: String,
    val desc: String,
    val coverUrl: String,
    val imageCount: Int,
    val videoDuration: Double,
    val isLivePhoto: Boolean,
    val livePhotoPairs: Int,
    val isPinned: Boolean,
    val pinnedRank: Int?,
    val publishAt: String,
    val status: String,
    val authorUniqueId: String,
    val authorName: String,
)

data class DouyinAccountResult(
    val secUserId: String,
    val displayName: String,
    val signature: String,
    val avatarUrl: String,
    val profileUrl: String,
    val followerCount: Long?,
    val followingCount: Long?,
    val awemeCount: Long?,
    val totalFavorited: Long?,
    val items: List<DouyinAccountItem>,
)

data class DouyinFavoriteUser(
    val secUserId: String,
    val sourceInput: String,
    val displayName: String,
    val signature: String,
    val avatarUrl: String,
    val profileUrl: String,
    val followerCount: Long?,
    val followingCount: Long?,
    val awemeCount: Long?,
    val totalFavorited: Long?,
    val lastParsedAt: String,
    val lastParsedCount: Int,
    val createTime: String,
    val updateTime: String,
    val tagIds: List<Long>,
)

data class DouyinFavoriteAweme(
    val awemeId: String,
    val secUserId: String,
    val type: String,
    val desc: String,
    val coverUrl: String,
    val createTime: String,
    val updateTime: String,
    val tagIds: List<Long>,
)

data class DouyinFavoriteTag(
    val id: Long,
    val name: String,
    val sortOrder: Long,
    val count: Long,
    val createTime: String,
    val updateTime: String,
)

data class DouyinFavoritesSnapshot(
    val users: List<DouyinFavoriteUser>,
    val awemes: List<DouyinFavoriteAweme>,
    val userTags: List<DouyinFavoriteTag>,
    val awemeTags: List<DouyinFavoriteTag>,
)

data class DouyinTagDialogState(
    val kind: DouyinTagKind,
    val targetId: String,
    val targetTitle: String,
    val selectedTagIds: Set<Long> = emptySet(),
    val saving: Boolean = false,
    val error: String? = null,
)

data class DouyinTagManagerState(
    val kind: DouyinTagKind,
    val nameInput: String = "",
    val creating: Boolean = false,
    val removingTagId: Long? = null,
    val error: String? = null,
)

class DouyinRepository @Inject constructor(
    private val douyinApiService: DouyinApiService,
    private val preferencesStore: AppPreferencesStore,
    private val baseUrlProvider: BaseUrlProvider,
) {
    suspend fun resolveDetail(input: String, cookie: String): AppResult<DouyinDetailResult> = runCatching {
        val normalizedInput = input.trim()
        if (normalizedInput.isBlank()) error("请输入抖音分享文本/链接/作品ID")
        val root = douyinApiService.getDetail(
            buildJsonObject {
                put("input", JsonPrimitive(normalizedInput))
                if (cookie.isNotBlank()) {
                    put("cookie", JsonPrimitive(cookie.trim()))
                }
            }
        ) as? JsonObject ?: error("抖音解析响应格式异常")
        root.errorMessage()?.let(::error)
        val coverUrl = root.stringOrNull("coverUrl").orEmpty().let(::normalizeUrl)
        DouyinDetailResult(
            key = root.stringOrNull("key") ?: error("解析结果缺少 key"),
            detailId = root.stringOrNull("detailId").orEmpty(),
            type = root.stringOrNull("type").orEmpty(),
            mediaType = root.stringOrNull("mediaType").orEmpty().ifBlank { normalizeDouyinMediaType(root.stringOrNull("type").orEmpty()) },
            title = root.stringOrNull("title").orEmpty(),
            coverUrl = coverUrl,
            imageCount = root.intOrDefault("imageCount", 0),
            videoDuration = root.doubleOrDefault("videoDuration", 0.0),
            isLivePhoto = root.booleanOrFalse("isLivePhoto"),
            livePhotoPairs = root.intOrDefault("livePhotoPairs", 0),
            items = root["items"]?.jsonArray.orEmpty().mapNotNull { it.toMediaItem(coverUrl) },
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "抖音解析失败", it) },
    )

    suspend fun resolveAccount(input: String, cookie: String): AppResult<DouyinAccountResult> = runCatching {
        val normalizedInput = input.trim()
        if (normalizedInput.isBlank()) error("请输入抖音用户主页链接/分享文本/sec_uid")
        val root = douyinApiService.getAccount(
            buildJsonObject {
                put("input", JsonPrimitive(normalizedInput))
                put("tab", JsonPrimitive("post"))
                put("count", JsonPrimitive(18))
                if (cookie.isNotBlank()) {
                    put("cookie", JsonPrimitive(cookie.trim()))
                }
            }
        ) as? JsonObject ?: error("抖音用户作品响应格式异常")
        root.errorMessage()?.let(::error)
        DouyinAccountResult(
            secUserId = root.stringOrNull("secUserId") ?: error("解析结果缺少 secUserId"),
            displayName = root.stringOrNull("displayName").orEmpty(),
            signature = root.stringOrNull("signature").orEmpty(),
            avatarUrl = root.stringOrNull("avatarUrl").orEmpty().let(::normalizeUrl),
            profileUrl = root.stringOrNull("profileUrl").orEmpty().let(::normalizeUrl),
            followerCount = root.longOrNull("followerCount"),
            followingCount = root.longOrNull("followingCount"),
            awemeCount = root.longOrNull("awemeCount"),
            totalFavorited = root.longOrNull("totalFavorited"),
            items = root["items"]?.jsonArray.orEmpty().mapNotNull { it.toAccountItem() },
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "获取用户作品失败", it) },
    )

    suspend fun importMedia(key: String, index: Int): AppResult<ImportedDouyinMedia> = runCatching {
        val normalizedKey = key.trim()
        if (normalizedKey.isBlank()) error("解析信息缺失，请重新解析")
        if (index < 0) error("媒体索引非法")
        val userId = preferencesStore.readCurrentSession()?.id?.takeIf { it.isNotBlank() } ?: "pre_identity"
        val root = douyinApiService.importMedia(
            userId = userId,
            key = normalizedKey,
            index = index,
        ) as? JsonObject ?: error("抖音导入响应格式异常")
        val state = root.stringOrNull("state").orEmpty()
        if (!state.equals("OK", ignoreCase = true)) {
            error(root.errorMessage() ?: "导入失败")
        }
        val localPath = root.stringOrNull("localPath") ?: error("导入结果缺少 localPath")
        ImportedDouyinMedia(
            localPath = localPath,
            localFilename = root.stringOrNull("localFilename").orEmpty(),
            dedup = root.booleanOrFalse("dedup"),
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "导入抖音媒体失败", it) },
    )

    suspend fun refreshFavoritesSnapshot(): AppResult<DouyinFavoritesSnapshot> = runCatching {
        DouyinFavoritesSnapshot(
            users = listFavoriteUsersOrThrow(),
            awemes = listFavoriteAwemesOrThrow(),
            userTags = listTagsOrThrow(DouyinTagKind.USERS),
            awemeTags = listTagsOrThrow(DouyinTagKind.AWEMES),
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载抖音收藏失败", it) },
    )

    suspend fun upsertFavoriteUser(accountInput: String, result: DouyinAccountResult): AppResult<DouyinFavoriteUser> = runCatching {
        val root = douyinApiService.addFavoriteUser(
            buildJsonObject {
                put("secUserId", JsonPrimitive(result.secUserId))
                if (accountInput.isNotBlank()) put("sourceInput", JsonPrimitive(accountInput.trim()))
                if (result.displayName.isNotBlank()) put("displayName", JsonPrimitive(result.displayName))
                if (result.avatarUrl.isNotBlank()) put("avatarUrl", JsonPrimitive(result.avatarUrl))
                if (result.profileUrl.isNotBlank()) put("profileUrl", JsonPrimitive(result.profileUrl))
                put("lastParsedCount", JsonPrimitive(result.items.size))
                put(
                    "lastParsedRaw",
                    buildJsonObject {
                        if (result.signature.isNotBlank()) put("signature", JsonPrimitive(result.signature))
                        result.followerCount?.let { put("followerCount", JsonPrimitive(it)) }
                        result.followingCount?.let { put("followingCount", JsonPrimitive(it)) }
                        result.awemeCount?.let { put("awemeCount", JsonPrimitive(it)) }
                        result.totalFavorited?.let { put("totalFavorited", JsonPrimitive(it)) }
                    },
                )
            }
        ) as? JsonObject ?: error("收藏作者响应格式异常")
        root.errorMessage()?.let(::error)
        root.toFavoriteUser() ?: error("收藏作者响应缺少 secUserId")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "收藏作者失败", it) },
    )

    suspend fun removeFavoriteUser(secUserId: String): AppResult<Unit> = runCatching {
        val normalizedId = secUserId.trim()
        if (normalizedId.isBlank()) error("secUserId 不能为空")
        val root = douyinApiService.removeFavoriteUser(
            buildJsonObject { put("secUserId", JsonPrimitive(normalizedId)) }
        ) as? JsonObject
        root?.errorMessage()?.let(::error)
        Unit
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "取消收藏作者失败", it) },
    )

    suspend fun upsertFavoriteAwemeFromDetail(result: DouyinDetailResult): AppResult<DouyinFavoriteAweme> = runCatching {
        val awemeId = result.detailId.trim()
        if (awemeId.isBlank()) error("当前作品缺少 awemeId")
        val root = douyinApiService.addFavoriteAweme(
            buildJsonObject {
                put("awemeId", JsonPrimitive(awemeId))
                put("type", JsonPrimitive(resolveFavoriteMediaType(result.type, result.mediaType, result.isLivePhoto, result.imageCount)))
                if (result.title.isNotBlank()) put("desc", JsonPrimitive(result.title))
                if (result.coverUrl.isNotBlank()) put("coverUrl", JsonPrimitive(result.coverUrl))
            }
        ) as? JsonObject ?: error("收藏作品响应格式异常")
        root.errorMessage()?.let(::error)
        root.toFavoriteAweme() ?: error("收藏作品响应缺少 awemeId")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "收藏作品失败", it) },
    )

    suspend fun upsertFavoriteAwemeFromAccount(secUserId: String, item: DouyinAccountItem): AppResult<DouyinFavoriteAweme> = runCatching {
        val awemeId = item.detailId.trim()
        if (awemeId.isBlank()) error("当前作品缺少 awemeId")
        val root = douyinApiService.addFavoriteAweme(
            buildJsonObject {
                put("awemeId", JsonPrimitive(awemeId))
                if (secUserId.isNotBlank()) put("secUserId", JsonPrimitive(secUserId))
                put("type", JsonPrimitive(resolveFavoriteMediaType(item.type, item.mediaType, item.isLivePhoto, item.imageCount)))
                if (item.desc.isNotBlank()) put("desc", JsonPrimitive(item.desc))
                if (item.coverUrl.isNotBlank()) put("coverUrl", JsonPrimitive(item.coverUrl))
            }
        ) as? JsonObject ?: error("收藏作品响应格式异常")
        root.errorMessage()?.let(::error)
        root.toFavoriteAweme() ?: error("收藏作品响应缺少 awemeId")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "收藏作品失败", it) },
    )

    suspend fun removeFavoriteAweme(awemeId: String): AppResult<Unit> = runCatching {
        val normalizedId = awemeId.trim()
        if (normalizedId.isBlank()) error("awemeId 不能为空")
        val root = douyinApiService.removeFavoriteAweme(
            buildJsonObject { put("awemeId", JsonPrimitive(normalizedId)) }
        ) as? JsonObject
        root?.errorMessage()?.let(::error)
        Unit
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "取消收藏作品失败", it) },
    )

    suspend fun createTag(kind: DouyinTagKind, name: String): AppResult<DouyinFavoriteTag> = runCatching {
        val normalizedName = name.trim()
        if (normalizedName.isBlank()) error("标签名称不能为空")
        val root = when (kind) {
            DouyinTagKind.USERS -> douyinApiService.addFavoriteUserTag(buildJsonObject { put("name", JsonPrimitive(normalizedName)) })
            DouyinTagKind.AWEMES -> douyinApiService.addFavoriteAwemeTag(buildJsonObject { put("name", JsonPrimitive(normalizedName)) })
        } as? JsonObject ?: error("标签保存响应格式异常")
        root.errorMessage()?.let(::error)
        root.toFavoriteTag() ?: error("标签保存失败")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "保存标签失败", it) },
    )

    suspend fun removeTag(kind: DouyinTagKind, tagId: Long): AppResult<Unit> = runCatching {
        if (tagId <= 0L) error("标签 ID 非法")
        val root = when (kind) {
            DouyinTagKind.USERS -> douyinApiService.removeFavoriteUserTag(buildJsonObject { put("id", JsonPrimitive(tagId)) })
            DouyinTagKind.AWEMES -> douyinApiService.removeFavoriteAwemeTag(buildJsonObject { put("id", JsonPrimitive(tagId)) })
        } as? JsonObject
        root?.errorMessage()?.let(::error)
        Unit
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "删除标签失败", it) },
    )

    suspend fun applyTags(kind: DouyinTagKind, targetId: String, tagIds: List<Long>): AppResult<Unit> = runCatching {
        val normalizedTargetId = targetId.trim()
        if (normalizedTargetId.isBlank()) error("目标 ID 不能为空")
        val normalizedTagIds = tagIds.map { it }.filter { it > 0L }.distinct()
        val payload = when (kind) {
            DouyinTagKind.USERS -> buildJsonObject {
                put("secUserIds", JsonArray(listOf(JsonPrimitive(normalizedTargetId))))
                put("tagIds", JsonArray(normalizedTagIds.map { JsonPrimitive(it) }))
                put("mode", JsonPrimitive("set"))
            }
            DouyinTagKind.AWEMES -> buildJsonObject {
                put("awemeIds", JsonArray(listOf(JsonPrimitive(normalizedTargetId))))
                put("tagIds", JsonArray(normalizedTagIds.map { JsonPrimitive(it) }))
                put("mode", JsonPrimitive("set"))
            }
        }
        val root = when (kind) {
            DouyinTagKind.USERS -> douyinApiService.applyFavoriteUserTags(payload)
            DouyinTagKind.AWEMES -> douyinApiService.applyFavoriteAwemeTags(payload)
        } as? JsonObject
        root?.errorMessage()?.let(::error)
        Unit
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "更新标签失败", it) },
    )

    private suspend fun listFavoriteUsersOrThrow(): List<DouyinFavoriteUser> {
        val root = douyinApiService.listFavoriteUsers() as? JsonObject ?: error("收藏作者列表响应格式异常")
        root.errorMessage()?.let(::error)
        return root["items"]?.jsonArray.orEmpty().mapNotNull { it.toFavoriteUser() }
    }

    private suspend fun listFavoriteAwemesOrThrow(): List<DouyinFavoriteAweme> {
        val root = douyinApiService.listFavoriteAwemes() as? JsonObject ?: error("收藏作品列表响应格式异常")
        root.errorMessage()?.let(::error)
        return root["items"]?.jsonArray.orEmpty().mapNotNull { it.toFavoriteAweme() }
    }

    private suspend fun listTagsOrThrow(kind: DouyinTagKind): List<DouyinFavoriteTag> {
        val root = when (kind) {
            DouyinTagKind.USERS -> douyinApiService.listFavoriteUserTags()
            DouyinTagKind.AWEMES -> douyinApiService.listFavoriteAwemeTags()
        } as? JsonObject ?: error("标签列表响应格式异常")
        root.errorMessage()?.let(::error)
        return root["items"]?.jsonArray.orEmpty().mapNotNull { it.toFavoriteTag() }
    }

    private fun JsonElement.toMediaItem(fallbackCoverUrl: String): DouyinMediaItem? {
        val root = this as? JsonObject ?: return null
        val downloadUrl = root.stringOrNull("downloadUrl")?.let(::normalizeUrl).orEmpty()
        if (downloadUrl.isBlank()) return null
        val rawType = root.stringOrNull("type").orEmpty().lowercase()
        val sourceUrl = root.stringOrNull("url").orEmpty()
        val originalFilename = root.stringOrNull("originalFilename").orEmpty()
        val type = when (rawType) {
            "image", "video" -> rawType
            else -> inferDouyinItemType(downloadUrl, sourceUrl, originalFilename)
        }
        return DouyinMediaItem(
            index = root.intOrDefault("index", 0),
            type = type,
            url = root.stringOrNull("url")?.let(::normalizeUrl).orEmpty(),
            downloadUrl = downloadUrl,
            originalFilename = originalFilename,
            thumbUrl = if (type == "image") downloadUrl else fallbackCoverUrl,
        )
    }

    private fun JsonElement.toAccountItem(): DouyinAccountItem? {
        val root = this as? JsonObject ?: return null
        val detailId = root.stringOrNull("detailId").orEmpty()
        if (detailId.isBlank()) return null
        return DouyinAccountItem(
            detailId = detailId,
            type = root.stringOrNull("type").orEmpty(),
            mediaType = root.stringOrNull("mediaType").orEmpty().ifBlank {
                resolveFavoriteMediaType(
                    type = root.stringOrNull("type").orEmpty(),
                    mediaType = "",
                    isLivePhoto = root.booleanOrFalse("isLivePhoto"),
                    imageCount = root.intOrDefault("imageCount", 0),
                )
            },
            desc = root.stringOrNull("desc").orEmpty(),
            coverUrl = root.stringOrNull("coverUrl").orEmpty().let(::normalizeUrl),
            imageCount = root.intOrDefault("imageCount", 0),
            videoDuration = root.doubleOrDefault("videoDuration", 0.0),
            isLivePhoto = root.booleanOrFalse("isLivePhoto"),
            livePhotoPairs = root.intOrDefault("livePhotoPairs", 0),
            isPinned = root.booleanOrFalse("isPinned"),
            pinnedRank = root.intOrNull("pinnedRank"),
            publishAt = root.stringOrNull("publishAt").orEmpty(),
            status = root.stringOrNull("status").orEmpty(),
            authorUniqueId = root.stringOrNull("authorUniqueId").orEmpty(),
            authorName = root.stringOrNull("authorName").orEmpty(),
        )
    }

    private fun JsonElement.toFavoriteUser(): DouyinFavoriteUser? {
        val root = this as? JsonObject ?: return null
        val secUserId = root.stringOrNull("secUserId").orEmpty()
        if (secUserId.isBlank()) return null
        return DouyinFavoriteUser(
            secUserId = secUserId,
            sourceInput = root.stringOrNull("sourceInput").orEmpty(),
            displayName = root.stringOrNull("displayName").orEmpty(),
            signature = root.stringOrNull("signature").orEmpty(),
            avatarUrl = root.stringOrNull("avatarUrl").orEmpty().let(::normalizeUrl),
            profileUrl = root.stringOrNull("profileUrl").orEmpty().let(::normalizeUrl),
            followerCount = root.longOrNull("followerCount"),
            followingCount = root.longOrNull("followingCount"),
            awemeCount = root.longOrNull("awemeCount"),
            totalFavorited = root.longOrNull("totalFavorited"),
            lastParsedAt = root.stringOrNull("lastParsedAt").orEmpty(),
            lastParsedCount = root.intOrDefault("lastParsedCount", 0),
            createTime = root.stringOrNull("createTime").orEmpty(),
            updateTime = root.stringOrNull("updateTime").orEmpty(),
            tagIds = root.longList("tagIds"),
        )
    }

    private fun JsonElement.toFavoriteAweme(): DouyinFavoriteAweme? {
        val root = this as? JsonObject ?: return null
        val awemeId = root.stringOrNull("awemeId").orEmpty()
        if (awemeId.isBlank()) return null
        return DouyinFavoriteAweme(
            awemeId = awemeId,
            secUserId = root.stringOrNull("secUserId").orEmpty(),
            type = root.stringOrNull("type").orEmpty(),
            desc = root.stringOrNull("desc").orEmpty(),
            coverUrl = root.stringOrNull("coverUrl").orEmpty().let(::normalizeUrl),
            createTime = root.stringOrNull("createTime").orEmpty(),
            updateTime = root.stringOrNull("updateTime").orEmpty(),
            tagIds = root.longList("tagIds"),
        )
    }

    private fun JsonElement.toFavoriteTag(): DouyinFavoriteTag? {
        val root = this as? JsonObject ?: return null
        val id = root.longOrNull("id") ?: return null
        return DouyinFavoriteTag(
            id = id,
            name = root.stringOrNull("name").orEmpty(),
            sortOrder = root.longOrNull("sortOrder") ?: 0L,
            count = root.longOrNull("count") ?: 0L,
            createTime = root.stringOrNull("createTime").orEmpty(),
            updateTime = root.stringOrNull("updateTime").orEmpty(),
        )
    }

    private fun normalizeUrl(raw: String): String {
        val value = raw.trim()
        if (value.isBlank()) return ""
        if (value.startsWith("http://") || value.startsWith("https://")) return value
        val apiBaseUrl = baseUrlProvider.currentApiBaseUrl()
        val isDefaultPort = (apiBaseUrl.isHttps && apiBaseUrl.port == 443) || (!apiBaseUrl.isHttps && apiBaseUrl.port == 80)
        val portSuffix = if (isDefaultPort) "" else ":${apiBaseUrl.port}"
        val origin = "${apiBaseUrl.scheme}://${apiBaseUrl.host}$portSuffix"
        return when {
            value.startsWith("/") -> origin + value
            else -> "$origin/$value"
        }
    }
}

data class DouyinUiState(
    val mode: DouyinScreenMode = DouyinScreenMode.DETAIL,
    val input: String = "",
    val accountInput: String = "",
    val cookie: String = "",
    val showCookieEditor: Boolean = false,
    val loading: Boolean = false,
    val accountLoading: Boolean = false,
    val favoritesLoading: Boolean = false,
    val importingIndex: Int? = null,
    val importedItems: Map<Int, DouyinImportStatus> = emptyMap(),
    val result: DouyinDetailResult? = null,
    val accountResult: DouyinAccountResult? = null,
    val favoriteUsers: List<DouyinFavoriteUser> = emptyList(),
    val favoriteAwemes: List<DouyinFavoriteAweme> = emptyList(),
    val favoriteUserTags: List<DouyinFavoriteTag> = emptyList(),
    val favoriteAwemeTags: List<DouyinFavoriteTag> = emptyList(),
    val tagDialog: DouyinTagDialogState? = null,
    val tagManager: DouyinTagManagerState? = null,
    val previewItem: DouyinMediaItem? = null,
    val message: String? = null,
)

@HiltViewModel
class DouyinViewModel @Inject constructor(
    private val repository: DouyinRepository,
) : ViewModel() {
    var uiState by mutableStateOf(DouyinUiState())
        private set

    init {
        refreshFavorites(notifyError = false)
    }

    fun switchMode(mode: DouyinScreenMode) {
        if (uiState.mode == mode) return
        uiState = uiState.copy(mode = mode, message = null)
        if (mode == DouyinScreenMode.FAVORITES &&
            uiState.favoriteUsers.isEmpty() &&
            uiState.favoriteAwemes.isEmpty() &&
            !uiState.favoritesLoading
        ) {
            refreshFavorites(notifyError = true)
        }
    }

    fun updateInput(value: String) {
        uiState = uiState.copy(input = value)
    }

    fun updateAccountInput(value: String) {
        uiState = uiState.copy(accountInput = value)
    }

    fun updateCookie(value: String) {
        uiState = uiState.copy(cookie = value)
    }

    fun toggleCookieEditor() {
        uiState = uiState.copy(showCookieEditor = !uiState.showCookieEditor)
    }

    fun clearCurrentMode() {
        when (uiState.mode) {
            DouyinScreenMode.DETAIL -> {
                uiState = uiState.copy(
                    input = "",
                    loading = false,
                    importingIndex = null,
                    importedItems = emptyMap(),
                    result = null,
                    previewItem = null,
                    message = null,
                )
            }
            DouyinScreenMode.ACCOUNT -> {
                uiState = uiState.copy(
                    accountInput = "",
                    accountLoading = false,
                    accountResult = null,
                    message = null,
                )
            }
            DouyinScreenMode.FAVORITES -> refreshFavorites(notifyError = true)
        }
    }

    fun resolveDetail() {
        if (uiState.loading || uiState.importingIndex != null) return
        viewModelScope.launch {
            uiState = uiState.copy(
                mode = DouyinScreenMode.DETAIL,
                loading = true,
                importingIndex = null,
                importedItems = emptyMap(),
                previewItem = null,
                message = null,
            )
            when (val result = repository.resolveDetail(uiState.input, uiState.cookie)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        loading = false,
                        result = result.data,
                        message = if (result.data.items.isEmpty()) "解析成功，但未返回可预览媒体" else "解析成功，共 ${result.data.items.size} 项",
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(
                    loading = false,
                    importingIndex = null,
                    result = null,
                    message = result.message,
                )
            }
        }
    }

    fun resolveAccount() {
        if (uiState.accountLoading) return
        viewModelScope.launch {
            uiState = uiState.copy(
                mode = DouyinScreenMode.ACCOUNT,
                accountLoading = true,
                accountResult = null,
                message = null,
            )
            when (val result = repository.resolveAccount(uiState.accountInput, uiState.cookie)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        accountLoading = false,
                        accountResult = result.data,
                        message = if (result.data.items.isEmpty()) "获取成功，但当前页暂无作品" else "已获取 ${result.data.items.size} 个作品",
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(
                    accountLoading = false,
                    accountResult = null,
                    message = result.message,
                )
            }
        }
    }

    fun refreshFavorites(notifyError: Boolean = true) {
        viewModelScope.launch {
            uiState = uiState.copy(favoritesLoading = true)
            when (val result = repository.refreshFavoritesSnapshot()) {
                is AppResult.Success -> {
                    val currentDialog = uiState.tagDialog
                    val validTagIds = when (currentDialog?.kind) {
                        DouyinTagKind.USERS -> result.data.userTags.map { it.id }.toSet()
                        DouyinTagKind.AWEMES -> result.data.awemeTags.map { it.id }.toSet()
                        null -> emptySet()
                    }
                    uiState = uiState.copy(
                        favoritesLoading = false,
                        favoriteUsers = result.data.users,
                        favoriteAwemes = result.data.awemes,
                        favoriteUserTags = result.data.userTags,
                        favoriteAwemeTags = result.data.awemeTags,
                        tagDialog = currentDialog?.copy(selectedTagIds = currentDialog.selectedTagIds.filter { it in validTagIds }.toSet()),
                    )
                }
                is AppResult.Error -> {
                    uiState = uiState.copy(
                        favoritesLoading = false,
                        message = if (notifyError) result.message else uiState.message,
                    )
                }
            }
        }
    }

    fun importItem(item: DouyinMediaItem, onImported: ((ImportedDouyinMedia) -> Unit)? = null) {
        val detail = uiState.result ?: run {
            uiState = uiState.copy(message = "请先解析抖音作品")
            return
        }
        val alreadyImported = uiState.importedItems[item.index]
        if (alreadyImported != null) {
            uiState = uiState.copy(
                message = when (alreadyImported) {
                    DouyinImportStatus.EXISTS -> "该媒体已存在于本地库，可直接在图片管理中查看"
                    DouyinImportStatus.IMPORTED -> "该媒体已导入到本地库"
                },
            )
            return
        }
        if (uiState.loading || uiState.importingIndex != null) return
        viewModelScope.launch {
            uiState = uiState.copy(importingIndex = item.index, message = null)
            when (val result = repository.importMedia(detail.key, item.index)) {
                is AppResult.Success -> {
                    val imported = result.data
                    uiState = uiState.copy(
                        importingIndex = null,
                        importedItems = uiState.importedItems + (
                            item.index to if (imported.dedup) DouyinImportStatus.EXISTS else DouyinImportStatus.IMPORTED
                        ),
                        previewItem = if (uiState.previewItem?.index == item.index) null else uiState.previewItem,
                        message = if (onImported == null) {
                            if (imported.dedup) {
                                "已存在（去重复用），可到图片管理查看"
                            } else {
                                "已导入到本地，可到图片管理查看或在聊天页手动发送"
                            }
                        } else {
                            null
                        },
                    )
                    onImported?.invoke(imported)
                }
                is AppResult.Error -> uiState = uiState.copy(importingIndex = null, message = result.message)
            }
        }
    }

    fun toggleFavoriteCurrentDetail() {
        val detail = uiState.result ?: run {
            uiState = uiState.copy(message = "请先解析抖音作品")
            return
        }
        val awemeId = detail.detailId.trim()
        if (awemeId.isBlank()) {
            uiState = uiState.copy(message = "当前解析结果缺少 awemeId")
            return
        }
        viewModelScope.launch {
            val action = if (isFavoriteAweme(awemeId)) {
                repository.removeFavoriteAweme(awemeId)
            } else {
                repository.upsertFavoriteAwemeFromDetail(detail).mapToUnit()
            }
            when (action) {
                is AppResult.Success -> {
                    refreshFavorites(notifyError = false)
                    uiState = uiState.copy(message = if (isFavoriteAweme(awemeId)) "已取消收藏作品" else "已收藏作品")
                }
                is AppResult.Error -> uiState = uiState.copy(message = action.message)
            }
        }
    }

    fun toggleFavoriteCurrentUser() {
        val account = uiState.accountResult ?: run {
            uiState = uiState.copy(message = "请先获取用户作品")
            return
        }
        val secUserId = account.secUserId.trim()
        if (secUserId.isBlank()) {
            uiState = uiState.copy(message = "当前用户缺少 secUserId")
            return
        }
        viewModelScope.launch {
            val action = if (isFavoriteUser(secUserId)) {
                repository.removeFavoriteUser(secUserId)
            } else {
                repository.upsertFavoriteUser(uiState.accountInput, account).mapToUnit()
            }
            when (action) {
                is AppResult.Success -> {
                    refreshFavorites(notifyError = false)
                    uiState = uiState.copy(message = if (isFavoriteUser(secUserId)) "已取消收藏作者" else "已收藏作者")
                }
                is AppResult.Error -> uiState = uiState.copy(message = action.message)
            }
        }
    }

    fun toggleFavoriteAccountAweme(item: DouyinAccountItem) {
        val account = uiState.accountResult ?: run {
            uiState = uiState.copy(message = "请先获取用户作品")
            return
        }
        val awemeId = item.detailId.trim()
        if (awemeId.isBlank()) {
            uiState = uiState.copy(message = "当前作品缺少 awemeId")
            return
        }
        viewModelScope.launch {
            val action = if (isFavoriteAweme(awemeId)) {
                repository.removeFavoriteAweme(awemeId)
            } else {
                repository.upsertFavoriteAwemeFromAccount(account.secUserId, item).mapToUnit()
            }
            when (action) {
                is AppResult.Success -> {
                    refreshFavorites(notifyError = false)
                    uiState = uiState.copy(message = if (isFavoriteAweme(awemeId)) "已取消收藏作品" else "已收藏作品")
                }
                is AppResult.Error -> uiState = uiState.copy(message = action.message)
            }
        }
    }

    fun removeFavoriteUser(secUserId: String) {
        val normalizedId = secUserId.trim()
        if (normalizedId.isBlank()) return
        viewModelScope.launch {
            when (val result = repository.removeFavoriteUser(normalizedId)) {
                is AppResult.Success -> {
                    if (uiState.tagDialog?.kind == DouyinTagKind.USERS && uiState.tagDialog?.targetId == normalizedId) {
                        uiState = uiState.copy(tagDialog = null)
                    }
                    refreshFavorites(notifyError = false)
                    uiState = uiState.copy(message = "已取消收藏作者")
                }
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun removeFavoriteAweme(awemeId: String) {
        val normalizedId = awemeId.trim()
        if (normalizedId.isBlank()) return
        viewModelScope.launch {
            when (val result = repository.removeFavoriteAweme(normalizedId)) {
                is AppResult.Success -> {
                    if (uiState.tagDialog?.kind == DouyinTagKind.AWEMES && uiState.tagDialog?.targetId == normalizedId) {
                        uiState = uiState.copy(tagDialog = null)
                    }
                    refreshFavorites(notifyError = false)
                    uiState = uiState.copy(message = "已取消收藏作品")
                }
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun reparseFavoriteUser(user: DouyinFavoriteUser) {
        uiState = uiState.copy(mode = DouyinScreenMode.ACCOUNT, accountInput = user.secUserId, accountResult = null, message = null)
        resolveAccount()
    }

    fun reparseFavoriteAweme(aweme: DouyinFavoriteAweme) {
        uiState = uiState.copy(mode = DouyinScreenMode.DETAIL, input = aweme.awemeId, result = null, importedItems = emptyMap(), message = null)
        resolveDetail()
    }

    fun openAccountItemDetail(item: DouyinAccountItem) {
        uiState = uiState.copy(mode = DouyinScreenMode.DETAIL, input = item.detailId, result = null, importedItems = emptyMap(), message = null)
        resolveDetail()
    }

    fun openTagEditor(kind: DouyinTagKind, targetId: String, targetTitle: String, presetTagIds: List<Long>) {
        val normalizedId = targetId.trim()
        if (normalizedId.isBlank()) return
        uiState = uiState.copy(
            tagManager = null,
            tagDialog = DouyinTagDialogState(
                kind = kind,
                targetId = normalizedId,
                targetTitle = targetTitle.ifBlank { normalizedId },
                selectedTagIds = presetTagIds.filter { it > 0L }.toSet(),
            ),
        )
    }

    fun closeTagEditor() {
        if (uiState.tagDialog != null) {
            uiState = uiState.copy(tagDialog = null)
        }
    }

    fun toggleTagSelection(tagId: Long) {
        val dialog = uiState.tagDialog ?: return
        if (tagId <= 0L || dialog.saving) return
        val next = dialog.selectedTagIds.toMutableSet()
        if (next.contains(tagId)) next.remove(tagId) else next.add(tagId)
        uiState = uiState.copy(tagDialog = dialog.copy(selectedTagIds = next))
    }

    fun saveTagSelection() {
        val dialog = uiState.tagDialog ?: return
        if (dialog.saving) return
        viewModelScope.launch {
            uiState = uiState.copy(tagDialog = dialog.copy(saving = true, error = null))
            when (val result = repository.applyTags(dialog.kind, dialog.targetId, dialog.selectedTagIds.toList())) {
                is AppResult.Success -> {
                    uiState = uiState.copy(tagDialog = null, message = "已更新标签")
                    refreshFavorites(notifyError = false)
                }
                is AppResult.Error -> uiState = uiState.copy(tagDialog = dialog.copy(saving = false, error = result.message), message = result.message)
            }
        }
    }

    fun openTagManager(kind: DouyinTagKind) {
        uiState = uiState.copy(
            tagDialog = null,
            tagManager = DouyinTagManagerState(kind = kind),
        )
    }

    fun closeTagManager() {
        if (uiState.tagManager != null) {
            uiState = uiState.copy(tagManager = null)
        }
    }

    fun updateTagManagerName(value: String) {
        val manager = uiState.tagManager ?: return
        uiState = uiState.copy(tagManager = manager.copy(nameInput = value))
    }

    fun createTag() {
        val manager = uiState.tagManager ?: return
        val name = manager.nameInput.trim()
        if (name.isBlank() || manager.creating) return
        viewModelScope.launch {
            uiState = uiState.copy(tagManager = manager.copy(creating = true, error = null))
            when (val result = repository.createTag(manager.kind, name)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(tagManager = manager.copy(nameInput = "", creating = false, error = null), message = "已创建标签")
                    refreshFavorites(notifyError = false)
                }
                is AppResult.Error -> uiState = uiState.copy(tagManager = manager.copy(creating = false, error = result.message), message = result.message)
            }
        }
    }

    fun removeTag(tagId: Long) {
        val manager = uiState.tagManager ?: return
        if (tagId <= 0L || manager.creating || manager.removingTagId != null) return
        viewModelScope.launch {
            uiState = uiState.copy(tagManager = manager.copy(removingTagId = tagId, error = null))
            when (val result = repository.removeTag(manager.kind, tagId)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(tagManager = manager.copy(removingTagId = null, error = null), message = "已删除标签")
                    refreshFavorites(notifyError = false)
                }
                is AppResult.Error -> uiState = uiState.copy(tagManager = manager.copy(removingTagId = null, error = result.message), message = result.message)
            }
        }
    }

    fun showPreview(item: DouyinMediaItem) {
        uiState = uiState.copy(previewItem = item)
    }

    fun dismissPreview() {
        if (uiState.previewItem != null) {
            uiState = uiState.copy(previewItem = null)
        }
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }

    private fun isFavoriteUser(secUserId: String): Boolean =
        uiState.favoriteUsers.any { it.secUserId.equals(secUserId.trim(), ignoreCase = false) }

    private fun isFavoriteAweme(awemeId: String): Boolean =
        uiState.favoriteAwemes.any { it.awemeId.equals(awemeId.trim(), ignoreCase = false) }

    private fun <T> AppResult<T>.mapToUnit(): AppResult<Unit> = when (this) {
        is AppResult.Success -> AppResult.Success(Unit)
        is AppResult.Error -> this
    }
}

@Composable
fun DouyinScreen(
    viewModel: DouyinViewModel,
    onBack: () -> Unit,
    onImportToChat: ((ImportedDouyinMedia) -> Unit)? = null,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }
    val context = LocalContext.current
    val importActionLabel = if (onImportToChat != null) "导入到会话" else "导入到本地"
    val favoriteUserMap = state.favoriteUsers.associateBy { it.secUserId }
    val favoriteAwemeMap = state.favoriteAwemes.associateBy { it.awemeId }
    val favoriteUserTagMap = state.favoriteUserTags.associate { it.id to it.name }
    val favoriteAwemeTagMap = state.favoriteAwemeTags.associate { it.id to it.name }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    state.previewItem?.let { item ->
        AlertDialog(
            onDismissRequest = viewModel::dismissPreview,
            confirmButton = {
                Button(
                    onClick = { viewModel.importItem(item, onImportToChat) },
                    enabled = state.importingIndex == null && state.importedItems[item.index] == null,
                ) {
                    Text(
                        resolveDouyinImportActionText(
                            importing = state.importingIndex == item.index,
                            status = state.importedItems[item.index],
                            defaultText = importActionLabel,
                        ),
                    )
                }
            },
            dismissButton = {
                OutlinedButton(onClick = {
                    openDouyinExternally(context, item.downloadUrl, item.type)
                    viewModel.dismissPreview()
                }) {
                    Text("外部打开")
                }
            },
            title = {
                Text(
                    text = item.originalFilename.ifBlank { "抖音媒体 ${item.index + 1}" },
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    if (item.thumbUrl.isNotBlank()) {
                        AsyncImage(
                            model = item.thumbUrl,
                            contentDescription = item.originalFilename,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(260.dp),
                            contentScale = ContentScale.Fit,
                        )
                    }
                    Text(
                        text = if (item.type == "video") {
                            "当前为视频结果，可先外部打开预览，再选择${importActionLabel}；如需下载，请返回列表使用“下载”按钮。"
                        } else {
                            "当前为图片结果，可直接查看，并选择${importActionLabel}；如需下载，请返回列表使用“下载”按钮。"
                        },
                        style = MaterialTheme.typography.bodySmall,
                    )
                    if (item.downloadUrl.isNotBlank()) {
                        Text(
                            text = item.downloadUrl,
                            style = MaterialTheme.typography.labelSmall,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            },
        )
    }

    state.tagDialog?.let { dialog ->
        DouyinTagDialog(
            state = dialog,
            tags = if (dialog.kind == DouyinTagKind.USERS) state.favoriteUserTags else state.favoriteAwemeTags,
            onDismiss = viewModel::closeTagEditor,
            onToggle = viewModel::toggleTagSelection,
            onSave = viewModel::saveTagSelection,
            onOpenManager = { viewModel.openTagManager(dialog.kind) },
        )
    }

    state.tagManager?.let { manager ->
        DouyinTagManagerDialog(
            state = manager,
            tags = if (manager.kind == DouyinTagKind.USERS) state.favoriteUserTags else state.favoriteAwemeTags,
            onDismiss = viewModel::closeTagManager,
            onNameChange = viewModel::updateTagManagerName,
            onCreate = viewModel::createTag,
            onRemove = viewModel::removeTag,
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("抖音下载") },
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
                DouyinModeCard(
                    mode = state.mode,
                    onSwitchMode = viewModel::switchMode,
                )
            }

            when (state.mode) {
                DouyinScreenMode.DETAIL -> {
                    item {
                        DouyinInputCard(
                            title = "作品解析",
                            description = if (onImportToChat != null) {
                                "支持粘贴抖音分享文本、短链、完整 URL 或作品 ID；当前页面会把导入结果直接回灌到聊天待发送列表。"
                            } else {
                                "支持粘贴抖音分享文本、短链、完整 URL 或作品 ID；当前页面可将作品导入本地媒体库，并保留下载入口。"
                            },
                            input = state.input,
                            inputLabel = "分享文本 / 链接 / 作品ID",
                            cookie = state.cookie,
                            showCookieEditor = state.showCookieEditor,
                            loading = state.loading,
                            onInputChange = viewModel::updateInput,
                            onCookieChange = viewModel::updateCookie,
                            onToggleCookie = viewModel::toggleCookieEditor,
                            onSubmit = viewModel::resolveDetail,
                            submitText = if (state.loading) "解析中..." else "解析",
                            onClear = viewModel::clearCurrentMode,
                        )
                    }

                    if (state.loading && state.result == null) {
                        item { LoadingCard(text = "正在解析抖音作品...") }
                    }

                    state.result?.let { result ->
                        val currentFavorite = favoriteAwemeMap[result.detailId]
                        item {
                            DouyinResultCard(
                                result = result,
                                favorite = currentFavorite != null,
                                tagNames = currentFavorite?.tagIds?.let { resolveTagNames(it, favoriteAwemeTagMap) }.orEmpty(),
                                onOpenCover = {
                                    if (result.coverUrl.isNotBlank()) {
                                        openDouyinExternally(context, result.coverUrl, "image")
                                    }
                                },
                                onToggleFavorite = viewModel::toggleFavoriteCurrentDetail,
                                onEditTags = if (currentFavorite != null) {
                                    {
                                        viewModel.openTagEditor(
                                            kind = DouyinTagKind.AWEMES,
                                            targetId = currentFavorite.awemeId,
                                            targetTitle = result.title.ifBlank { currentFavorite.awemeId },
                                            presetTagIds = currentFavorite.tagIds,
                                        )
                                    }
                                } else {
                                    null
                                },
                            )
                        }
                        if (result.items.isEmpty()) {
                            item { EmptyCard(text = "解析结果暂未返回可下载媒体") }
                        } else {
                            items(result.items, key = { "${result.key}-${it.index}" }) { item ->
                                DouyinMediaCard(
                                    item = item,
                                    importLabel = resolveDouyinImportActionText(
                                        importing = state.importingIndex == item.index,
                                        status = state.importedItems[item.index],
                                        defaultText = importActionLabel,
                                    ),
                                    importEnabled = state.importingIndex == null && state.importedItems[item.index] == null,
                                    onPreview = {
                                        if (item.type == "image") {
                                            viewModel.showPreview(item)
                                        } else {
                                            openDouyinExternally(context, item.downloadUrl, item.type)
                                        }
                                    },
                                    onDownload = {
                                        enqueueDouyinDownload(
                                            context = context,
                                            url = item.downloadUrl,
                                            filename = item.originalFilename.ifBlank { defaultDouyinFileName(item) },
                                        )
                                    },
                                    onImport = { viewModel.importItem(item, onImportToChat) },
                                )
                            }
                        }
                    }
                }

                DouyinScreenMode.ACCOUNT -> {
                    item {
                        DouyinInputCard(
                            title = "用户作品",
                            description = "支持粘贴用户主页链接、分享文本或 sec_uid，抓取该作者当前页作品；可直接收藏作者或收藏其中的作品。",
                            input = state.accountInput,
                            inputLabel = "用户主页链接 / 分享文本 / sec_uid",
                            cookie = state.cookie,
                            showCookieEditor = state.showCookieEditor,
                            loading = state.accountLoading,
                            onInputChange = viewModel::updateAccountInput,
                            onCookieChange = viewModel::updateCookie,
                            onToggleCookie = viewModel::toggleCookieEditor,
                            onSubmit = viewModel::resolveAccount,
                            submitText = if (state.accountLoading) "获取中..." else "获取作品",
                            onClear = viewModel::clearCurrentMode,
                        )
                    }

                    if (state.accountLoading && state.accountResult == null) {
                        item { LoadingCard(text = "正在获取用户作品...") }
                    }

                    state.accountResult?.let { account ->
                        val currentFavorite = favoriteUserMap[account.secUserId]
                        item {
                            DouyinAccountCard(
                                result = account,
                                favorite = currentFavorite != null,
                                tagNames = currentFavorite?.tagIds?.let { resolveTagNames(it, favoriteUserTagMap) }.orEmpty(),
                                onToggleFavorite = viewModel::toggleFavoriteCurrentUser,
                                onEditTags = if (currentFavorite != null) {
                                    {
                                        viewModel.openTagEditor(
                                            kind = DouyinTagKind.USERS,
                                            targetId = currentFavorite.secUserId,
                                            targetTitle = account.displayName.ifBlank { currentFavorite.secUserId },
                                            presetTagIds = currentFavorite.tagIds,
                                        )
                                    }
                                } else {
                                    null
                                },
                            )
                        }
                        if (account.items.isEmpty()) {
                            item { EmptyCard(text = "当前作者暂无可展示作品") }
                        } else {
                            items(account.items, key = { it.detailId }) { item ->
                                val favoriteAweme = favoriteAwemeMap[item.detailId]
                                DouyinAccountItemCard(
                                    item = item,
                                    favorite = favoriteAweme != null,
                                    tagNames = favoriteAweme?.tagIds?.let { resolveTagNames(it, favoriteAwemeTagMap) }.orEmpty(),
                                    onOpenDetail = { viewModel.openAccountItemDetail(item) },
                                    onToggleFavorite = { viewModel.toggleFavoriteAccountAweme(item) },
                                    onEditTags = if (favoriteAweme != null) {
                                        {
                                            viewModel.openTagEditor(
                                                kind = DouyinTagKind.AWEMES,
                                                targetId = favoriteAweme.awemeId,
                                                targetTitle = item.desc.ifBlank { favoriteAweme.awemeId },
                                                presetTagIds = favoriteAweme.tagIds,
                                            )
                                        }
                                    } else {
                                        null
                                    },
                                )
                            }
                        }
                    }
                }

                DouyinScreenMode.FAVORITES -> {
                    item {
                        DouyinFavoritesHeaderCard(
                            loading = state.favoritesLoading,
                            onRefresh = { viewModel.refreshFavorites(notifyError = true) },
                            onManageUserTags = { viewModel.openTagManager(DouyinTagKind.USERS) },
                            onManageAwemeTags = { viewModel.openTagManager(DouyinTagKind.AWEMES) },
                        )
                    }

                    if (state.favoritesLoading && state.favoriteUsers.isEmpty() && state.favoriteAwemes.isEmpty()) {
                        item { LoadingCard(text = "正在加载抖音收藏...") }
                    }

                    item {
                        SectionTitle(
                            text = "收藏作者 (${state.favoriteUsers.size})",
                            subtitle = if (state.favoriteUserTags.isEmpty()) "可通过“管理作者标签”创建标签" else "已配置 ${state.favoriteUserTags.size} 个作者标签",
                        )
                    }
                    if (state.favoriteUsers.isEmpty()) {
                        item { EmptyCard(text = "暂无收藏作者") }
                    } else {
                        items(state.favoriteUsers, key = { it.secUserId }) { user ->
                            FavoriteUserCard(
                                item = user,
                                tagNames = resolveTagNames(user.tagIds, favoriteUserTagMap),
                                onOpen = { viewModel.reparseFavoriteUser(user) },
                                onEditTags = {
                                    viewModel.openTagEditor(
                                        kind = DouyinTagKind.USERS,
                                        targetId = user.secUserId,
                                        targetTitle = user.displayName.ifBlank { user.secUserId },
                                        presetTagIds = user.tagIds,
                                    )
                                },
                                onRemove = { viewModel.removeFavoriteUser(user.secUserId) },
                            )
                        }
                    }

                    item {
                        SectionTitle(
                            text = "收藏作品 (${state.favoriteAwemes.size})",
                            subtitle = if (state.favoriteAwemeTags.isEmpty()) "可通过“管理作品标签”创建标签" else "已配置 ${state.favoriteAwemeTags.size} 个作品标签",
                        )
                    }
                    if (state.favoriteAwemes.isEmpty()) {
                        item { EmptyCard(text = "暂无收藏作品") }
                    } else {
                        items(state.favoriteAwemes, key = { it.awemeId }) { aweme ->
                            FavoriteAwemeCard(
                                item = aweme,
                                tagNames = resolveTagNames(aweme.tagIds, favoriteAwemeTagMap),
                                onOpen = { viewModel.reparseFavoriteAweme(aweme) },
                                onEditTags = {
                                    viewModel.openTagEditor(
                                        kind = DouyinTagKind.AWEMES,
                                        targetId = aweme.awemeId,
                                        targetTitle = aweme.desc.ifBlank { aweme.awemeId },
                                        presetTagIds = aweme.tagIds,
                                    )
                                },
                                onRemove = { viewModel.removeFavoriteAweme(aweme.awemeId) },
                            )
                        }
                    }
                }
            }

            item { Spacer(modifier = Modifier.height(8.dp)) }
        }
    }
}

@Composable
private fun DouyinModeCard(
    mode: DouyinScreenMode,
    onSwitchMode: (DouyinScreenMode) -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 4.dp),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                DouyinModeButton(
                    text = "作品解析",
                    selected = mode == DouyinScreenMode.DETAIL,
                    onClick = { onSwitchMode(DouyinScreenMode.DETAIL) },
                )
                DouyinModeButton(
                    text = "用户作品",
                    selected = mode == DouyinScreenMode.ACCOUNT,
                    onClick = { onSwitchMode(DouyinScreenMode.ACCOUNT) },
                )
                DouyinModeButton(
                    text = "收藏",
                    selected = mode == DouyinScreenMode.FAVORITES,
                    onClick = { onSwitchMode(DouyinScreenMode.FAVORITES) },
                )
            }
            Text(
                text = when (mode) {
                    DouyinScreenMode.DETAIL -> "从作品分享文本进入，继续进行预览、下载、导入或收藏作品。"
                    DouyinScreenMode.ACCOUNT -> "从用户主页链接或 sec_uid 进入，继续收藏作者、查看作品并收藏其中作品。"
                    DouyinScreenMode.FAVORITES -> "查看已收藏的作者 / 作品，并管理两类标签。"
                },
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

@Composable
private fun RowScope.DouyinModeButton(
    text: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    if (selected) {
        Button(onClick = onClick, modifier = Modifier.weight(1f)) {
            Text(text)
        }
    } else {
        OutlinedButton(onClick = onClick, modifier = Modifier.weight(1f)) {
            Text(text)
        }
    }
}

@Composable
private fun DouyinInputCard(
    title: String,
    description: String,
    input: String,
    inputLabel: String,
    cookie: String,
    showCookieEditor: Boolean,
    loading: Boolean,
    onInputChange: (String) -> Unit,
    onCookieChange: (String) -> Unit,
    onToggleCookie: () -> Unit,
    onSubmit: () -> Unit,
    submitText: String,
    onClear: () -> Unit,
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
            Text(title, style = MaterialTheme.typography.titleMedium)
            Text(description, style = MaterialTheme.typography.bodyMedium)
            OutlinedTextField(
                value = input,
                onValueChange = onInputChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text(inputLabel) },
                minLines = 3,
                enabled = !loading,
            )
            OutlinedButton(
                onClick = onToggleCookie,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (showCookieEditor) "隐藏 Cookie" else "填写 Cookie（可选）")
            }
            if (showCookieEditor) {
                OutlinedTextField(
                    value = cookie,
                    onValueChange = onCookieChange,
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text("抖音 Cookie（可选）") },
                    minLines = 3,
                    enabled = !loading,
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(
                    onClick = onSubmit,
                    enabled = !loading && input.isNotBlank(),
                    modifier = Modifier.weight(1f),
                ) {
                    Text(submitText)
                }
                OutlinedButton(
                    onClick = onClear,
                    enabled = !loading,
                    modifier = Modifier.weight(1f),
                ) {
                    Text("清空")
                }
            }
        }
    }
}

@Composable
private fun LoadingCard(text: String) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            Text(text, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

@Composable
private fun DouyinResultCard(
    result: DouyinDetailResult,
    favorite: Boolean,
    tagNames: String,
    onOpenCover: () -> Unit,
    onToggleFavorite: () -> Unit,
    onEditTags: (() -> Unit)?,
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
            Text(
                text = result.title.ifBlank { "抖音作品 ${result.detailId}" },
                style = MaterialTheme.typography.titleMedium,
            )
            val summary = buildList {
                if (result.detailId.isNotBlank()) add("作品ID ${result.detailId}")
                add(resolveDouyinMediaTypeLabel(result.mediaType, result.type, result.imageCount, result.isLivePhoto, result.livePhotoPairs))
                if (result.items.isNotEmpty()) add("${result.items.size} 项")
            }.joinToString(separator = " · ")
            if (summary.isNotBlank()) {
                Text(summary, style = MaterialTheme.typography.bodySmall)
            }
            if (tagNames.isNotBlank()) {
                Text("标签：$tagNames", style = MaterialTheme.typography.bodySmall)
            }
            if (result.coverUrl.isNotBlank()) {
                AsyncImage(
                    model = result.coverUrl,
                    contentDescription = result.title,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(220.dp)
                        .clickable(onClick = onOpenCover),
                    contentScale = ContentScale.Fit,
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onToggleFavorite, modifier = Modifier.weight(1f)) {
                    Text(if (favorite) "取消收藏作品" else "收藏作品")
                }
                if (favorite && onEditTags != null) {
                    OutlinedButton(onClick = onEditTags, modifier = Modifier.weight(1f)) {
                        Text("标签")
                    }
                }
            }
        }
    }
}

@Composable
private fun DouyinAccountCard(
    result: DouyinAccountResult,
    favorite: Boolean,
    tagNames: String,
    onToggleFavorite: () -> Unit,
    onEditTags: (() -> Unit)?,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (result.avatarUrl.isNotBlank()) {
                AsyncImage(
                    model = result.avatarUrl,
                    contentDescription = result.displayName,
                    modifier = Modifier.size(72.dp),
                    contentScale = ContentScale.Crop,
                )
            }
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    text = result.displayName.ifBlank { result.secUserId },
                    style = MaterialTheme.typography.titleMedium,
                )
                Text("sec_uid：${result.secUserId}", style = MaterialTheme.typography.bodySmall)
                val stats = buildList {
                    result.followerCount?.let { add("粉丝 $it") }
                    result.followingCount?.let { add("关注 $it") }
                    result.awemeCount?.let { add("作品 $it") }
                    result.totalFavorited?.let { add("获赞 $it") }
                }.joinToString(separator = " · ")
                if (stats.isNotBlank()) {
                    Text(stats, style = MaterialTheme.typography.bodySmall)
                }
                if (result.signature.isNotBlank()) {
                    Text(result.signature, style = MaterialTheme.typography.bodySmall, maxLines = 3, overflow = TextOverflow.Ellipsis)
                }
                if (tagNames.isNotBlank()) {
                    Text("标签：$tagNames", style = MaterialTheme.typography.bodySmall)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(onClick = onToggleFavorite, modifier = Modifier.weight(1f)) {
                        Text(if (favorite) "取消收藏作者" else "收藏作者")
                    }
                    if (favorite && onEditTags != null) {
                        OutlinedButton(onClick = onEditTags, modifier = Modifier.weight(1f)) {
                            Text("标签")
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun DouyinAccountItemCard(
    item: DouyinAccountItem,
    favorite: Boolean,
    tagNames: String,
    onOpenDetail: () -> Unit,
    onToggleFavorite: () -> Unit,
    onEditTags: (() -> Unit)?,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier.size(88.dp),
                contentAlignment = Alignment.Center,
            ) {
                if (item.coverUrl.isNotBlank()) {
                    AsyncImage(
                        model = item.coverUrl,
                        contentDescription = item.desc,
                        modifier = Modifier.fillMaxSize(),
                        contentScale = ContentScale.Crop,
                    )
                } else {
                    Text(if (normalizeDouyinMediaType(item.mediaType.ifBlank { item.type }) == "video") "视频" else "作品")
                }
            }
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    text = item.desc.ifBlank { item.detailId },
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                val summary = buildList {
                    add(resolveDouyinMediaTypeLabel(item.mediaType, item.type, item.imageCount, item.isLivePhoto, item.livePhotoPairs))
                    if (item.publishAt.isNotBlank()) add(item.publishAt)
                    if (item.isPinned) add("置顶${item.pinnedRank?.let { " #$it" }.orEmpty()}")
                    if (item.status.isNotBlank()) add(item.status)
                }.joinToString(separator = " · ")
                if (summary.isNotBlank()) {
                    Text(summary, style = MaterialTheme.typography.bodySmall)
                }
                if (tagNames.isNotBlank()) {
                    Text("标签：$tagNames", style = MaterialTheme.typography.bodySmall)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = onOpenDetail, modifier = Modifier.weight(1f)) {
                        Text("查看详情")
                    }
                    Button(onClick = onToggleFavorite, modifier = Modifier.weight(1f)) {
                        Text(if (favorite) "取消收藏" else "收藏作品")
                    }
                }
                if (favorite && onEditTags != null) {
                    OutlinedButton(onClick = onEditTags, modifier = Modifier.fillMaxWidth()) {
                        Text("设置标签")
                    }
                }
            }
        }
    }
}

@Composable
private fun DouyinFavoritesHeaderCard(
    loading: Boolean,
    onRefresh: () -> Unit,
    onManageUserTags: () -> Unit,
    onManageAwemeTags: () -> Unit,
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
            Text("收藏与标签管理", style = MaterialTheme.typography.titleMedium)
            Text(
                "可查看已收藏的抖音作者 / 作品，并分别维护两类标签；点击条目可重新获取作者作品或重新解析作品详情。",
                style = MaterialTheme.typography.bodySmall,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onRefresh, enabled = !loading, modifier = Modifier.weight(1f)) {
                    Text(if (loading) "刷新中..." else "刷新收藏")
                }
                OutlinedButton(onClick = onManageUserTags, modifier = Modifier.weight(1f)) {
                    Text("作者标签")
                }
            }
            OutlinedButton(onClick = onManageAwemeTags, modifier = Modifier.fillMaxWidth()) {
                Text("作品标签")
            }
        }
    }
}

@Composable
private fun FavoriteUserCard(
    item: DouyinFavoriteUser,
    tagNames: String,
    onOpen: () -> Unit,
    onEditTags: () -> Unit,
    onRemove: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (item.avatarUrl.isNotBlank()) {
                AsyncImage(
                    model = item.avatarUrl,
                    contentDescription = item.displayName,
                    modifier = Modifier.size(72.dp),
                    contentScale = ContentScale.Crop,
                )
            }
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(item.displayName.ifBlank { item.secUserId }, style = MaterialTheme.typography.titleSmall)
                Text("sec_uid：${item.secUserId}", style = MaterialTheme.typography.bodySmall)
                if (item.signature.isNotBlank()) {
                    Text(item.signature, style = MaterialTheme.typography.bodySmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
                }
                val summary = buildList {
                    item.followerCount?.let { add("粉丝 $it") }
                    item.awemeCount?.let { add("作品 $it") }
                    if (item.lastParsedCount > 0) add("上次抓取 ${item.lastParsedCount} 个")
                }.joinToString(separator = " · ")
                if (summary.isNotBlank()) {
                    Text(summary, style = MaterialTheme.typography.bodySmall)
                }
                if (tagNames.isNotBlank()) {
                    Text("标签：$tagNames", style = MaterialTheme.typography.bodySmall)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = onOpen, modifier = Modifier.weight(1f)) {
                        Text("重新获取")
                    }
                    OutlinedButton(onClick = onEditTags, modifier = Modifier.weight(1f)) {
                        Text("标签")
                    }
                    Button(onClick = onRemove, modifier = Modifier.weight(1f)) {
                        Text("取消收藏")
                    }
                }
            }
        }
    }
}

@Composable
private fun FavoriteAwemeCard(
    item: DouyinFavoriteAweme,
    tagNames: String,
    onOpen: () -> Unit,
    onEditTags: () -> Unit,
    onRemove: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier.size(88.dp),
                contentAlignment = Alignment.Center,
            ) {
                if (item.coverUrl.isNotBlank()) {
                    AsyncImage(
                        model = item.coverUrl,
                        contentDescription = item.desc,
                        modifier = Modifier.fillMaxSize(),
                        contentScale = ContentScale.Crop,
                    )
                } else {
                    Text(if (normalizeDouyinMediaType(item.type) == "video") "视频" else "作品")
                }
            }
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    item.desc.ifBlank { item.awemeId },
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                val summary = buildList {
                    add(resolveDouyinMediaTypeLabel(item.type, item.type, 0, normalizeDouyinMediaType(item.type) == "livePhoto", 0))
                    if (item.secUserId.isNotBlank()) add("作者 ${item.secUserId}")
                    if (item.updateTime.isNotBlank()) add(item.updateTime)
                }.joinToString(separator = " · ")
                if (summary.isNotBlank()) {
                    Text(summary, style = MaterialTheme.typography.bodySmall)
                }
                if (tagNames.isNotBlank()) {
                    Text("标签：$tagNames", style = MaterialTheme.typography.bodySmall)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = onOpen, modifier = Modifier.weight(1f)) {
                        Text("重新解析")
                    }
                    OutlinedButton(onClick = onEditTags, modifier = Modifier.weight(1f)) {
                        Text("标签")
                    }
                    Button(onClick = onRemove, modifier = Modifier.weight(1f)) {
                        Text("取消收藏")
                    }
                }
            }
        }
    }
}

@Composable
private fun DouyinTagDialog(
    state: DouyinTagDialogState,
    tags: List<DouyinFavoriteTag>,
    onDismiss: () -> Unit,
    onToggle: (Long) -> Unit,
    onSave: () -> Unit,
    onOpenManager: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = {
            if (!state.saving) onDismiss()
        },
        confirmButton = {
            Button(onClick = onSave, enabled = !state.saving) {
                Text(if (state.saving) "保存中..." else "保存")
            }
        },
        dismissButton = {
            OutlinedButton(onClick = onDismiss, enabled = !state.saving) {
                Text("关闭")
            }
        },
        title = {
            Text(if (state.kind == DouyinTagKind.USERS) "设置作者标签" else "设置作品标签")
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(
                    text = state.targetTitle.ifBlank { state.targetId },
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (state.error != null) {
                    Text(state.error, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
                }
                if (tags.isEmpty()) {
                    Text("暂无标签，可先去管理页创建", style = MaterialTheme.typography.bodySmall)
                } else {
                    LazyColumn(modifier = Modifier.heightIn(max = 260.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        items(tags, key = { it.id }) { tag ->
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(tag.name, style = MaterialTheme.typography.bodyMedium)
                                    Text("已关联 ${tag.count} 个条目", style = MaterialTheme.typography.labelSmall)
                                }
                                OutlinedButton(onClick = { onToggle(tag.id) }, enabled = !state.saving) {
                                    Text(if (state.selectedTagIds.contains(tag.id)) "取消" else "选择")
                                }
                            }
                        }
                    }
                }
                OutlinedButton(onClick = onOpenManager, modifier = Modifier.fillMaxWidth(), enabled = !state.saving) {
                    Text(if (state.kind == DouyinTagKind.USERS) "管理作者标签" else "管理作品标签")
                }
            }
        },
    )
}

@Composable
private fun DouyinTagManagerDialog(
    state: DouyinTagManagerState,
    tags: List<DouyinFavoriteTag>,
    onDismiss: () -> Unit,
    onNameChange: (String) -> Unit,
    onCreate: () -> Unit,
    onRemove: (Long) -> Unit,
) {
    AlertDialog(
        onDismissRequest = {
            if (!state.creating && state.removingTagId == null) onDismiss()
        },
        confirmButton = {
            Button(onClick = onDismiss, enabled = !state.creating && state.removingTagId == null) {
                Text("完成")
            }
        },
        dismissButton = {},
        title = {
            Text(if (state.kind == DouyinTagKind.USERS) "管理作者标签" else "管理作品标签")
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("标签全局共享；删除标签会从所有收藏条目移除。", style = MaterialTheme.typography.bodySmall)
                OutlinedTextField(
                    value = state.nameInput,
                    onValueChange = onNameChange,
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text("新建标签名称") },
                    enabled = !state.creating && state.removingTagId == null,
                )
                Button(
                    onClick = onCreate,
                    enabled = state.nameInput.isNotBlank() && !state.creating && state.removingTagId == null,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (state.creating) "创建中..." else "创建标签")
                }
                if (state.error != null) {
                    Text(state.error, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
                }
                if (tags.isEmpty()) {
                    Text("暂无标签", style = MaterialTheme.typography.bodySmall)
                } else {
                    LazyColumn(modifier = Modifier.heightIn(max = 280.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        items(tags, key = { it.id }) { tag ->
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(tag.name, style = MaterialTheme.typography.bodyMedium)
                                    Text("已关联 ${tag.count} 个条目", style = MaterialTheme.typography.labelSmall)
                                }
                                TextButton(
                                    onClick = { onRemove(tag.id) },
                                    enabled = !state.creating && state.removingTagId == null,
                                ) {
                                    Text(if (state.removingTagId == tag.id) "删除中..." else "删除")
                                }
                            }
                        }
                    }
                }
            }
        },
    )
}

@Composable
private fun DouyinMediaCard(
    item: DouyinMediaItem,
    importLabel: String,
    importEnabled: Boolean,
    onPreview: () -> Unit,
    onDownload: () -> Unit,
    onImport: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp)
            .clickable(onClick = onPreview),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier.size(88.dp),
                contentAlignment = Alignment.Center,
            ) {
                if (item.thumbUrl.isNotBlank()) {
                    AsyncImage(
                        model = item.thumbUrl,
                        contentDescription = item.originalFilename,
                        modifier = Modifier.fillMaxSize(),
                        contentScale = ContentScale.Crop,
                    )
                } else {
                    Text(if (item.type == "video") "视频" else "图片")
                }
            }
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    text = item.originalFilename.ifBlank { "媒体 ${item.index + 1}" },
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = if (item.type == "video") "视频结果" else "图片结果",
                    style = MaterialTheme.typography.labelMedium,
                )
                Text(
                    text = item.downloadUrl,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = onPreview, modifier = Modifier.weight(1f)) {
                        Text(if (item.type == "video") "打开" else "预览")
                    }
                    Button(onClick = onDownload, modifier = Modifier.weight(1f)) {
                        Text("下载")
                    }
                }
                Button(
                    onClick = onImport,
                    enabled = importEnabled,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(importLabel)
                }
            }
        }
    }
}

@Composable
private fun SectionTitle(text: String, subtitle: String = "") {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text(text, style = MaterialTheme.typography.titleSmall)
        if (subtitle.isNotBlank()) {
            Text(subtitle, style = MaterialTheme.typography.bodySmall)
        }
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

private fun inferDouyinItemType(downloadUrl: String, sourceUrl: String, originalFilename: String): String {
    val candidates = listOf(downloadUrl, sourceUrl, originalFilename)
    val lower = candidates.joinToString(separator = "\n").lowercase()
    return when {
        listOf(".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".bmp").any { lower.contains(it) } -> "image"
        listOf(".mp4", ".mov", ".webm", ".m4v", "video_id=", "/aweme/v1/play/").any { lower.contains(it) } -> "video"
        else -> "video"
    }
}

private fun resolveDouyinMediaTypeLabel(
    mediaType: String,
    type: String,
    imageCount: Int,
    isLivePhoto: Boolean,
    livePhotoPairs: Int,
): String {
    val normalized = when {
        isLivePhoto -> "livePhoto"
        else -> normalizeDouyinMediaType(mediaType.ifBlank { type })
    }
    return when (normalized) {
        "livePhoto" -> if (livePhotoPairs > 0) "实况图（$livePhotoPairs 对）" else "实况图"
        "imageAlbum" -> if (imageCount > 1) "图集 ${imageCount} 张" else "图片"
        else -> "视频"
    }
}

private fun normalizeDouyinMediaType(raw: String): String = when (raw.trim().lowercase()) {
    "livephoto", "live", "motionphoto", "实况" -> "livePhoto"
    "imagealbum", "album", "image", "图集", "图片" -> "imageAlbum"
    else -> "video"
}

private fun resolveFavoriteMediaType(type: String, mediaType: String, isLivePhoto: Boolean, imageCount: Int): String {
    if (isLivePhoto) return "livePhoto"
    return when (normalizeDouyinMediaType(mediaType.ifBlank { type })) {
        "livePhoto" -> "livePhoto"
        "imageAlbum" -> if (imageCount > 1) "imageAlbum" else "image"
        else -> "video"
    }
}

private fun resolveTagNames(tagIds: List<Long>, tagMap: Map<Long, String>): String =
    tagIds.mapNotNull { tagMap[it] }.distinct().joinToString(separator = "、")

private fun defaultDouyinFileName(item: DouyinMediaItem): String {
    val baseName = item.originalFilename.ifBlank { "douyin_${item.index + 1}" }
    return if (baseName.contains('.')) baseName else baseName + if (item.type == "video") ".mp4" else ".jpg"
}

private fun resolveDouyinImportActionText(
    importing: Boolean,
    status: DouyinImportStatus?,
    defaultText: String,
): String = when {
    importing -> "导入中..."
    status == DouyinImportStatus.EXISTS -> "已存在（去重）"
    status == DouyinImportStatus.IMPORTED -> "已导入"
    else -> defaultText
}

private fun enqueueDouyinDownload(context: Context, url: String, filename: String) {
    if (url.isBlank()) return
    val request = DownloadManager.Request(Uri.parse(url)).apply {
        setTitle(filename)
        setDescription("抖音媒体下载")
        setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
        setAllowedOverMetered(true)
        setAllowedOverRoaming(true)
        setMimeType(if (filename.endsWith(".mp4", ignoreCase = true)) "video/mp4" else "image/*")
        setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, filename)
    }
    runCatching {
        val manager = context.getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
        manager.enqueue(request)
    }.onFailure {
        openDouyinExternally(context, url, if (filename.endsWith(".mp4", ignoreCase = true)) "video" else "image")
    }
}

private fun openDouyinExternally(context: Context, url: String, type: String) {
    if (url.isBlank()) return
    val uri = Uri.parse(url)
    val mimeType = if (type == "video") "video/*" else "image/*"
    val typedIntent = Intent(Intent.ACTION_VIEW).apply {
        data = uri
        setDataAndType(uri, mimeType)
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }
    val fallbackIntent = Intent(Intent.ACTION_VIEW, uri).apply {
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }
    runCatching { context.startActivity(typedIntent) }
        .onFailure { runCatching { context.startActivity(fallbackIntent) } }
}

private fun JsonObject.errorMessage(): String? =
    stringOrNull("error") ?: stringOrNull("msg")

private fun JsonObject.stringOrNull(key: String): String? =
    this[key]?.let { runCatching { it.jsonPrimitive.contentOrNull ?: it.jsonPrimitive.content }.getOrNull() }?.takeIf { it.isNotBlank() }

private fun JsonObject.intOrDefault(key: String, defaultValue: Int): Int = stringOrNull(key)?.toIntOrNull() ?: defaultValue

private fun JsonObject.intOrNull(key: String): Int? = stringOrNull(key)?.toIntOrNull()

private fun JsonObject.longOrNull(key: String): Long? = stringOrNull(key)?.toLongOrNull()

private fun JsonObject.doubleOrDefault(key: String, defaultValue: Double): Double = stringOrNull(key)?.toDoubleOrNull() ?: defaultValue

private fun JsonObject.booleanOrFalse(key: String): Boolean = stringOrNull(key)?.toBooleanStrictOrNull() ?: false

private fun JsonObject.longList(key: String): List<Long> =
    (this[key] as? JsonArray)?.mapNotNull { element ->
        runCatching { element.jsonPrimitive.contentOrNull ?: element.jsonPrimitive.content }.getOrNull()?.toLongOrNull()
    }.orEmpty()
