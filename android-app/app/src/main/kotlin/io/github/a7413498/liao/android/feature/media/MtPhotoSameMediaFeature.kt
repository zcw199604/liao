package io.github.a7413498.liao.android.feature.media

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import dagger.hilt.android.scopes.ViewModelScoped
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import javax.inject.Inject
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive

@ViewModelScoped
class MtPhotoSameMediaRepository @Inject constructor(
    private val mtPhotoApiService: MtPhotoApiService,
) {
    suspend fun queryByLocalPath(localPath: String): AppResult<List<MtPhotoSameMediaItem>> = runCatching {
        val normalizedPath = localPath.trim()
        if (normalizedPath.isBlank()) error("localPath 不能为空")
        val root = mtPhotoApiService.getSameMedia(localPath = normalizedPath) as? JsonObject ?: error("同媒体响应格式异常")
        root.stringOrNull("error")?.let(::error)
        root["items"]?.jsonArray.orEmpty().mapNotNull { it.toSameMediaItem() }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "查询 mtPhoto 同媒体失败", it) },
    )

    private fun JsonElement.toSameMediaItem(): MtPhotoSameMediaItem? {
        val root = this as? JsonObject ?: return null
        val md5 = root.stringOrNull("md5") ?: return null
        return MtPhotoSameMediaItem(
            id = root.longOrNull("id") ?: 0L,
            md5 = md5,
            filePath = root.stringOrNull("filePath").orEmpty(),
            fileName = root.stringOrNull("fileName").orEmpty(),
            folderId = root.longOrNull("folderId") ?: 0L,
            folderPath = root.stringOrNull("folderPath").orEmpty(),
            folderName = root.stringOrNull("folderName").orEmpty(),
            day = root.stringOrNull("day").orEmpty(),
            canOpenFolder = root.booleanOrFalse("canOpenFolder"),
        )
    }
}

data class MtPhotoSameMediaItem(
    val id: Long,
    val md5: String,
    val filePath: String,
    val fileName: String,
    val folderId: Long,
    val folderPath: String,
    val folderName: String,
    val day: String,
    val canOpenFolder: Boolean,
)

@Composable
internal fun MtPhotoSameMediaDialog(
    visible: Boolean,
    loading: Boolean,
    sourceTitle: String,
    error: String?,
    items: List<MtPhotoSameMediaItem>,
    onDismiss: () -> Unit,
    onRetry: () -> Unit,
    onOpenFolder: (MtPhotoSameMediaItem) -> Unit,
) {
    if (!visible) return

    AlertDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            OutlinedButton(onClick = onDismiss) {
                Text("关闭")
            }
        },
        dismissButton = {
            if (error != null) {
                Button(onClick = onRetry, enabled = !loading) {
                    Text(if (loading) "查询中..." else "重试")
                }
            }
        },
        title = {
            Text(
                text = if (sourceTitle.isBlank()) "mtPhoto 同媒体" else "mtPhoto 同媒体 · $sourceTitle",
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        },
        text = {
            when {
                loading -> {
                    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        CircularProgressIndicator()
                        Text("正在查询同媒体结果...")
                    }
                }
                error != null -> {
                    Text(error, style = MaterialTheme.typography.bodyMedium)
                }
                items.isEmpty() -> {
                    Text("未找到可打开的 mtPhoto 同媒体结果。", style = MaterialTheme.typography.bodyMedium)
                }
                else -> {
                    LazyColumn(
                        modifier = Modifier
                            .fillMaxWidth()
                            .heightIn(max = 360.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        items(items, key = { "${it.folderId}-${it.id}-${it.md5}" }) { item ->
                            MtPhotoSameMediaItemCard(
                                item = item,
                                onOpenFolder = { onOpenFolder(item) },
                            )
                        }
                    }
                }
            }
        },
    )
}

@Composable
private fun MtPhotoSameMediaItemCard(
    item: MtPhotoSameMediaItem,
    onOpenFolder: () -> Unit,
) {
    androidx.compose.material3.Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                text = item.fileName.ifBlank { item.filePath.ifBlank { item.md5 } },
                style = MaterialTheme.typography.titleSmall,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            val lines = buildList {
                if (item.day.isNotBlank()) add(item.day)
                if (item.folderName.isNotBlank()) add(item.folderName)
                if (item.folderPath.isNotBlank()) add(item.folderPath)
                if (item.filePath.isNotBlank()) add(item.filePath)
            }
            lines.forEach { line ->
                Text(
                    text = line,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            OutlinedButton(
                onClick = onOpenFolder,
                enabled = item.canOpenFolder && item.folderId > 0,
            ) {
                Text(if (item.canOpenFolder && item.folderId > 0) "打开所在目录" else "目录不可用")
            }
        }
    }
}

private fun JsonObject.stringOrNull(key: String): String? =
    this[key]?.let { runCatching { it.jsonPrimitive.contentOrNull ?: it.jsonPrimitive.content }.getOrNull() }?.takeIf { it.isNotBlank() }

private fun JsonObject.longOrNull(key: String): Long? = stringOrNull(key)?.toLongOrNull()

private fun JsonObject.booleanOrFalse(key: String): Boolean = stringOrNull(key)?.toBooleanStrictOrNull() ?: false
