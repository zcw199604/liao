package io.github.a7413498.liao.android.feature.mtphoto

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dagger.hilt.android.scopes.ViewModelScoped
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import javax.inject.Inject
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive

@ViewModelScoped
class MtPhotoFolderFavoritesRepository @Inject constructor(
    private val mtPhotoApiService: MtPhotoApiService,
    private val baseUrlProvider: BaseUrlProvider,
) {
    suspend fun loadFavorites(): AppResult<List<MtPhotoFolderFavorite>> = runCatching {
        val root = mtPhotoApiService.getFolderFavorites() as? JsonObject ?: error("目录收藏响应格式异常")
        root.stringOrNull("error")?.let(::error)
        root["items"]?.jsonArray.orEmpty().mapNotNull { it.toFolderFavorite() }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载目录收藏失败", it) },
    )

    suspend fun upsertFavorite(
        folderId: Long,
        folderName: String,
        folderPath: String,
        coverMd5: String = "",
    ): AppResult<MtPhotoFolderFavorite> = runCatching {
        val root = mtPhotoApiService.upsertFolderFavorite(
            buildJsonObject {
                put("folderId", JsonPrimitive(folderId))
                put("folderName", JsonPrimitive(folderName))
                put("folderPath", JsonPrimitive(folderPath))
                if (coverMd5.isNotBlank()) {
                    put("coverMd5", JsonPrimitive(coverMd5))
                }
                put("tags", JsonArray(emptyList()))
                put("note", JsonPrimitive(""))
            }
        ) as? JsonObject ?: error("保存目录收藏响应格式异常")
        root.stringOrNull("error")?.let(::error)
        if (root.booleanOrFalse("success").not()) {
            error(root.stringOrNull("message") ?: "保存目录收藏失败")
        }
        root["item"]?.toFolderFavorite() ?: error("保存目录收藏返回为空")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "保存目录收藏失败", it) },
    )

    suspend fun removeFavorite(folderId: Long): AppResult<Unit> = runCatching {
        val root = mtPhotoApiService.removeFolderFavorite(
            buildJsonObject {
                put("folderId", JsonPrimitive(folderId))
            }
        ) as? JsonObject ?: error("移除目录收藏响应格式异常")
        root.stringOrNull("error")?.let(::error)
        if (root.booleanOrFalse("success").not()) {
            error(root.stringOrNull("message") ?: "移除目录收藏失败")
        }
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "移除目录收藏失败", it) },
    )

    private fun JsonElement.toFolderFavorite(): MtPhotoFolderFavorite? {
        val root = this as? JsonObject ?: return null
        val folderId = root.longOrNull("folderId") ?: return null
        val coverMd5 = root.stringOrNull("coverMd5").orEmpty()
        val tags = root["tags"]?.jsonArray.orEmpty().mapNotNull { element ->
            runCatching { element.jsonPrimitive.contentOrNull ?: element.jsonPrimitive.content }.getOrNull()?.trim()?.takeIf { it.isNotBlank() }
        }
        return MtPhotoFolderFavorite(
            id = root.longOrNull("id") ?: folderId,
            folderId = folderId,
            folderName = root.stringOrNull("folderName") ?: "目录 $folderId",
            folderPath = root.stringOrNull("folderPath").orEmpty(),
            coverMd5 = coverMd5,
            coverUrl = coverMd5.takeIf { it.isNotBlank() }?.let { thumbUrl(md5 = it, size = "s260") }.orEmpty(),
            tags = tags,
            note = root.stringOrNull("note").orEmpty(),
            updateTime = root.stringOrNull("updateTime").orEmpty(),
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

data class MtPhotoFolderFavorite(
    val id: Long,
    val folderId: Long,
    val folderName: String,
    val folderPath: String,
    val coverMd5: String,
    val coverUrl: String,
    val tags: List<String>,
    val note: String,
    val updateTime: String,
)

@Composable
internal fun CurrentFolderFavoriteCard(
    current: MtPhotoFolderHistoryItem?,
    currentFavorite: MtPhotoFolderFavorite?,
    loading: Boolean,
    saving: Boolean,
    onRefresh: () -> Unit,
    onSave: () -> Unit,
    onRemove: () -> Unit,
) {
    val folderId = current?.folderId
    if (folderId == null || folderId <= 0) return

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(
                text = if (currentFavorite == null) "当前目录未收藏" else "当前目录已收藏",
                style = MaterialTheme.typography.titleMedium,
            )
            current?.folderPath?.takeIf { it.isNotBlank() }?.let { folderPath ->
                Text(
                    text = folderPath,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            if (currentFavorite != null) {
                if (currentFavorite.tags.isNotEmpty()) {
                    Text(
                        text = "标签：${currentFavorite.tags.joinToString("、")}",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
                if (currentFavorite.note.isNotBlank()) {
                    Text(
                        text = currentFavorite.note,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedButton(
                    onClick = onRefresh,
                    enabled = !loading && !saving,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(if (loading) "刷新中..." else "刷新收藏")
                }
                if (currentFavorite == null) {
                    Button(
                        onClick = onSave,
                        enabled = !saving,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(if (saving) "保存中..." else "收藏当前目录")
                    }
                } else {
                    OutlinedButton(
                        onClick = onRemove,
                        enabled = !saving,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(if (saving) "处理中..." else "取消收藏")
                    }
                }
            }
        }
    }
}

@Composable
internal fun MtPhotoFolderFavoriteCard(
    item: MtPhotoFolderFavorite,
    onOpen: () -> Unit,
    onRemove: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp)
            .clickable(onClick = onOpen),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            MtPhotoThumb(
                url = item.coverUrl,
                label = "收藏",
                modifier = Modifier.size(72.dp),
            )
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                Text(
                    text = item.folderName,
                    style = MaterialTheme.typography.titleSmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (item.folderPath.isNotBlank()) {
                    Text(
                        text = item.folderPath,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                val meta = buildList {
                    if (item.tags.isNotEmpty()) add("标签 ${item.tags.size}")
                    if (item.updateTime.isNotBlank()) add(item.updateTime)
                }.joinToString(separator = " · ")
                if (meta.isNotBlank()) {
                    Text(meta, style = MaterialTheme.typography.labelSmall)
                }
            }
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = onOpen) {
                    Text("进入")
                }
                OutlinedButton(onClick = onRemove) {
                    Text("移除")
                }
            }
        }
    }
}

