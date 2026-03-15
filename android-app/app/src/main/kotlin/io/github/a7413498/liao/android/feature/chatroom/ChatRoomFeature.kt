/*
 * 聊天模块对齐 Web 端的最小可用房间能力：
 * - Room + HTTP 历史记录恢复
 * - typed WebSocket 事件（typing / online status / forceout / notice）
 * - optimistic 发送、echo merge、失败重试
 * - 收藏、拉黑、清空重载、在线状态查询
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.chatroom

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.OpenableColumns
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import coil.compose.AsyncImage
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.LiaoLogger
import io.github.a7413498.liao.android.core.common.OutgoingMessageStatus
import io.github.a7413498.liao.android.core.common.inferFileName
import io.github.a7413498.liao.android.core.common.inferMessageType
import io.github.a7413498.liao.android.core.common.normalizeTextForMatch
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.database.MessageEntity
import io.github.a7413498.liao.android.core.database.toTimelineMessage
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.MediaApiService
import io.github.a7413498.liao.android.core.network.SystemApiService
import io.github.a7413498.liao.android.core.network.toHistoryMessageList
import io.github.a7413498.liao.android.core.network.toTimeline
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.LiaoWsEvent
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import java.net.URLConnection
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.UUID
import javax.inject.Inject
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okio.source

private const val HISTORY_PAGE_SIZE = 20

data class HistoryPageResult(
    val messages: List<ChatTimelineMessage>,
    val historyCursor: String?,
    val hasMoreHistory: Boolean,
)

data class ChatMediaAsset(
    val id: String,
    val url: String,
    val type: ChatMessageType,
    val localFilename: String = "",
    val remotePath: String = "",
    val posterUrl: String = "",
)

class ChatRoomRepository @Inject constructor(
    private val chatApiService: ChatApiService,
    private val mediaApiService: MediaApiService,
    private val systemApiService: SystemApiService,
    private val baseUrlProvider: BaseUrlProvider,
    private val preferencesStore: AppPreferencesStore,
    private val messageDao: MessageDao,
    private val conversationDao: ConversationDao,
    private val webSocketClient: LiaoWebSocketClient,
    @ApplicationContext private val appContext: Context,
) {
    private var cachedImageServerHost: String? = null
    private var cachedSystemConfig: io.github.a7413498.liao.android.core.network.SystemConfigDto? = null
    private var cachedResolvedPort: String? = null

    suspend fun ensureConnected(): AppResult<WebSocketState> = runCatching {
        val token = preferencesStore.readAuthToken().orEmpty()
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        if (token.isBlank()) error("请先登录")
        webSocketClient.connect(token = token, session = session)
        chatApiService.reportReferrer(
            referrerUrl = "",
            currUrl = "android://chatroom",
            userId = session.id,
            cookieData = session.cookie,
            referer = BuildConfig.DEFAULT_REFERER,
            userAgent = BuildConfig.DEFAULT_USER_AGENT,
        )
        webSocketClient.state.value
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "建立聊天连接失败", it) },
    )

    private fun currentApiOrigin(): String {
        val apiBaseUrl = baseUrlProvider.currentApiBaseUrl()
        val isDefaultPort = (apiBaseUrl.isHttps && apiBaseUrl.port == 443) || (!apiBaseUrl.isHttps && apiBaseUrl.port == 80)
        val portSuffix = if (isDefaultPort) "" else ":${apiBaseUrl.port}"
        return "${apiBaseUrl.scheme}://${apiBaseUrl.host}$portSuffix"
    }

    private suspend fun getImageServerHost(): String {
        cachedImageServerHost?.takeIf { it.isNotBlank() }?.let { return it }
        val raw = mediaApiService.getImgServer().string().trim()
        val host = parseImageServerHost(raw)
        cachedImageServerHost = host
        return host
    }

    private fun parseImageServerHost(raw: String): String {
        val trimmed = raw.trim().removePrefix(""").removeSuffix(""")
        if (trimmed.isBlank()) error("图片服务器地址为空")
        val parsedFromJson = runCatching {
            val root = kotlinx.serialization.json.Json.parseToJsonElement(trimmed)
            when (root) {
                is JsonObject -> {
                    val msgValue = root["msg"]
                    when (msgValue) {
                        is JsonObject -> msgValue["server"]?.jsonPrimitive?.contentOrNull
                        else -> root["server"]?.jsonPrimitive?.contentOrNull ?: msgValue?.jsonPrimitive?.contentOrNull
                    }
                }
                else -> null
            }
        }.getOrNull().orEmpty()
        val candidate = parsedFromJson.ifBlank { trimmed }
            .removePrefix("http://")
            .removePrefix("https://")
            .substringBefore('/')
            .substringBefore(':')
            .trim()
        if (candidate.isBlank()) error("无法解析图片服务器地址")
        return candidate
    }

    private suspend fun getSystemConfig(): io.github.a7413498.liao.android.core.network.SystemConfigDto {
        cachedSystemConfig?.let { return it }
        val config = runCatching {
            systemApiService.getSystemConfig().data ?: io.github.a7413498.liao.android.core.network.SystemConfigDto()
        }.getOrElse {
            preferencesStore.readCachedSystemConfig() ?: io.github.a7413498.liao.android.core.network.SystemConfigDto()
        }
        cachedSystemConfig = config
        preferencesStore.saveCachedSystemConfig(config)
        return config
    }

    private suspend fun resolveImagePort(uploadPath: String): String {
        val systemConfig = getSystemConfig()
        val fixedPort = systemConfig.imagePortFixed.ifBlank { "9006" }
        if (systemConfig.imagePortMode.equals("fixed", ignoreCase = true)) {
            return fixedPort
        }
        cachedResolvedPort?.takeIf { it.isNotBlank() }?.let { return it }
        val response = systemApiService.resolveImagePort(
            buildJsonObject {
                put("path", JsonPrimitive(uploadPath))
            }
        )
        val resolvedPort = (response.data as? JsonObject)
            ?.get("port")
            ?.jsonPrimitive
            ?.contentOrNull
            ?.ifBlank { null }
            ?: fixedPort
        cachedResolvedPort = resolvedPort
        return resolvedPort
    }

    private fun buildPosterUrlFromLocalPath(localPath: String): String {
        val normalized = localPath.trim().substringBefore('?').substringBefore('#')
        if (!normalized.startsWith("/videos/")) return ""
        val posterPath = normalized.replace(Regex("\\.[^.]+$"), ".poster.jpg")
        return currentApiOrigin() + "/upload" + posterPath
    }

    private suspend fun resolveMediaUrl(rawValue: String): String {
        val normalized = rawValue.trim().removePrefix("[").removeSuffix("]")
        if (normalized.isBlank()) return ""
        if (normalized.startsWith("http://") || normalized.startsWith("https://")) return normalized
        if (normalized.startsWith("/upload/")) return currentApiOrigin() + normalized
        if (normalized.startsWith("/images/") || normalized.startsWith("/videos/")) return currentApiOrigin() + "/upload" + normalized
        val imageHost = getImageServerHost()
        val port = resolveImagePort(normalized)
        return "http://$imageHost:$port/img/Upload/$normalized"
    }

    private fun inferMediaTypeFromPath(path: String, mimeType: String = ""): ChatMessageType {
        val normalizedMime = mimeType.lowercase(Locale.ROOT)
        return when {
            normalizedMime.startsWith("image/") -> ChatMessageType.IMAGE
            normalizedMime.startsWith("video/") -> ChatMessageType.VIDEO
            normalizedMime.isNotBlank() -> ChatMessageType.FILE
            else -> inferMessageType("[$path]")
        }
    }

    suspend fun hydrateTimelineMessage(message: ChatTimelineMessage): ChatTimelineMessage {
        if (message.type == ChatMessageType.TEXT) return message
        val rawValue = message.mediaUrl.ifBlank { message.content }
        val resolvedUrl = runCatching { resolveMediaUrl(rawValue) }.getOrDefault(rawValue.trim().removePrefix("[").removeSuffix("]"))
        return message.copy(
            mediaUrl = resolvedUrl,
            fileName = message.fileName.ifBlank { inferFileName(rawValue) },
        )
    }

    private suspend fun hydrateTimelineMessages(messages: List<ChatTimelineMessage>): List<ChatTimelineMessage> =
        messages.map { hydrateTimelineMessage(it) }

    private fun queryFileMeta(uri: Uri): Triple<String, String, Long> {
        val contentResolver = appContext.contentResolver
        var displayName = "upload_${System.currentTimeMillis()}"
        var size = -1L
        contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE), null, null, null)?.use { cursor ->
            if (cursor.moveToFirst()) {
                val nameIndex = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                val sizeIndex = cursor.getColumnIndex(OpenableColumns.SIZE)
                if (nameIndex >= 0) displayName = cursor.getString(nameIndex) ?: displayName
                if (sizeIndex >= 0) size = cursor.getLong(sizeIndex)
            }
        }
        val mimeType = contentResolver.getType(uri)
            ?: URLConnection.guessContentTypeFromName(displayName)
            ?: "application/octet-stream"
        return Triple(displayName, mimeType, size)
    }

    private fun buildMultipartFilePart(uri: Uri, fileName: String, mimeType: String, size: Long): MultipartBody.Part {
        val contentResolver = appContext.contentResolver
        val requestBody = object : RequestBody() {
            override fun contentType() = mimeType.toMediaType()
            override fun contentLength(): Long = size

            override fun writeTo(sink: okio.BufferedSink) {
                contentResolver.openInputStream(uri)?.use { inputStream ->
                    sink.writeAll(inputStream.source())
                } ?: error("无法读取所选文件")
            }
        }
        return MultipartBody.Part.createFormData("file", fileName, requestBody)
    }

    suspend fun uploadMedia(uri: Uri): AppResult<ChatMediaAsset> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val (displayName, mimeType, size) = queryFileMeta(uri)
        val payload = mediaApiService.uploadMedia(
            file = buildMultipartFilePart(uri, displayName, mimeType, size),
            userId = session.id.toRequestBody("text/plain".toMediaType()),
            cookieData = session.cookie.toRequestBody("text/plain".toMediaType()),
            referer = BuildConfig.DEFAULT_REFERER.toRequestBody("text/plain".toMediaType()),
            userAgent = BuildConfig.DEFAULT_USER_AGENT.toRequestBody("text/plain".toMediaType()),
            source = "local".toRequestBody("text/plain".toMediaType()),
        )
        val root = payload as? JsonObject ?: error("上传响应格式异常")
        val state = root["state"]?.jsonPrimitive?.contentOrNull.orEmpty()
        if (!state.equals("OK", ignoreCase = true)) {
            error(root["error"]?.jsonPrimitive?.contentOrNull ?: root["msg"]?.jsonPrimitive?.contentOrNull ?: "上传失败")
        }
        val remotePath = root["msg"]?.jsonPrimitive?.contentOrNull?.trim().orEmpty().ifBlank { error("上传结果缺少远端路径") }
        val localFilename = root["localFilename"]?.jsonPrimitive?.contentOrNull.orEmpty().ifBlank { displayName }
        val posterUrl = root["posterUrl"]?.jsonPrimitive?.contentOrNull.orEmpty().ifBlank {
            root["posterLocalPath"]?.jsonPrimitive?.contentOrNull?.let { localPath ->
                when {
                    localPath.isBlank() -> ""
                    localPath.startsWith("http://") || localPath.startsWith("https://") -> localPath
                    localPath.startsWith("/upload/") -> currentApiOrigin() + localPath
                    localPath.startsWith("/") -> currentApiOrigin() + "/upload" + localPath
                    else -> currentApiOrigin() + "/upload/" + localPath
                }
            }.orEmpty()
        }
        ChatMediaAsset(
            id = remotePath,
            url = resolveMediaUrl(remotePath),
            type = inferMediaTypeFromPath(remotePath, mimeType),
            localFilename = localFilename,
            remotePath = remotePath,
            posterUrl = posterUrl,
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "上传媒体失败", it) },
    )

    suspend fun reuploadLocalMedia(localPath: String, localFilename: String = ""): AppResult<ChatMediaAsset> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val payload = mediaApiService.reuploadHistoryImage(
            userId = session.id,
            localPath = localPath,
            cookieData = session.cookie,
            referer = BuildConfig.DEFAULT_REFERER,
            userAgent = BuildConfig.DEFAULT_USER_AGENT,
        )
        val root = payload as? JsonObject ?: error("重传媒体响应格式异常")
        val state = root["state"]?.jsonPrimitive?.contentOrNull.orEmpty()
        if (!state.equals("OK", ignoreCase = true)) {
            error(root["error"]?.jsonPrimitive?.contentOrNull ?: root["msg"]?.jsonPrimitive?.contentOrNull ?: "重传媒体失败")
        }
        val remotePath = root["msg"]?.jsonPrimitive?.contentOrNull?.trim().orEmpty().ifBlank { error("重传结果缺少远端路径") }
        val inferredType = inferMediaTypeFromPath(remotePath)
        ChatMediaAsset(
            id = remotePath,
            url = resolveMediaUrl(remotePath),
            type = inferredType,
            localFilename = localFilename.ifBlank { inferFileName(remotePath) },
            remotePath = remotePath,
            posterUrl = if (inferredType == ChatMessageType.VIDEO) buildPosterUrlFromLocalPath(localPath) else "",
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "重传媒体失败", it) },
    )

    suspend fun loadChatHistoryMedia(peerId: String, limit: Int = 20): AppResult<List<ChatMediaAsset>> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val raw = mediaApiService.getChatImages(userId1 = session.id, userId2 = peerId, limit = limit)
        val urls = when (raw) {
            is JsonArray -> raw.mapNotNull { element -> element.jsonPrimitive.contentOrNull?.takeIf { it.isNotBlank() } }
            is JsonObject -> raw["data"]?.jsonArray?.mapNotNull { element -> element.jsonPrimitive.contentOrNull?.takeIf { it.isNotBlank() } }.orEmpty()
            else -> emptyList()
        }
        urls.map { url ->
            ChatMediaAsset(
                id = url,
                url = url,
                type = inferMediaTypeFromPath(url),
                localFilename = inferFileName(url),
            )
        }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载聊天历史媒体失败", it) },
    )

    suspend fun sendUploadedMedia(
        peerId: String,
        peerName: String,
        media: ChatMediaAsset,
        clientId: String,
    ): AppResult<ChatTimelineMessage> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val remotePath = media.remotePath.ifBlank { error("媒体缺少远端路径，无法发送") }
        val content = "[$remotePath]"
        val sent = webSocketClient.sendPrivateMessage(
            targetUserId = peerId,
            targetUserName = peerName,
            senderId = session.id,
            content = content,
        )
        if (!sent) error("WebSocket 未连接")
        runCatching {
            mediaApiService.recordImageSend(
                remoteUrl = media.url,
                fromUserId = session.id,
                toUserId = peerId,
                localFilename = media.localFilename,
            )
        }
        ChatTimelineMessage(
            id = clientId,
            fromUserId = session.id,
            fromUserName = session.name,
            toUserId = peerId,
            content = content,
            time = nowTimeLabel(),
            isSelf = true,
            type = media.type,
            mediaUrl = media.url,
            fileName = media.localFilename.ifBlank { inferFileName(remotePath) },
            clientId = clientId,
            sendStatus = OutgoingMessageStatus.SENDING,
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "发送媒体失败", it) },
    )

    suspend fun loadHistory(
        peerId: String,
        isFirst: Boolean = true,
        firstTid: String = "0",
    ): AppResult<HistoryPageResult> = try {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val historyItems = chatApiService.getMessageHistory(
            myUserId = session.id,
            userToId = peerId,
            isFirst = if (isFirst) "1" else "0",
            firstTid = firstTid,
            cookieData = session.cookie,
            referer = BuildConfig.DEFAULT_REFERER,
            userAgent = BuildConfig.DEFAULT_USER_AGENT,
        ).toHistoryMessageList()
        val items = hydrateTimelineMessages(
            historyItems
                .asReversed()
                .map { historyItem ->
                    historyItem.toTimeline(
                        currentUserId = session.id,
                        peerId = peerId,
                        currentUserName = session.name,
                    )
                }
        )
        if (isFirst) {
            messageDao.clearByPeer(peerId)
        }
        if (items.isNotEmpty()) {
            messageDao.upsert(items.map { it.toEntity(peerId) })
        }
        AppResult.Success(
            HistoryPageResult(
                messages = items,
                historyCursor = items.minNumericTid(),
                hasMoreHistory = historyItems.size >= HISTORY_PAGE_SIZE,
            )
        )
    } catch (throwable: Throwable) {
        if (isFirst) {
            val cachedItems = hydrateTimelineMessages(
                messageDao.listByPeer(peerId).map { entity -> entity.toTimelineMessage() }
            )
            AppResult.Success(
                HistoryPageResult(
                    messages = cachedItems,
                    historyCursor = cachedItems.minNumericTid(),
                    hasMoreHistory = cachedItems.size >= HISTORY_PAGE_SIZE,
                )
            )
        } else {
            AppResult.Error(throwable.message ?: "加载聊天记录失败", throwable)
        }
    }

    suspend fun sendText(
        peerId: String,
        peerName: String,
        content: String,
        clientId: String,
    ): AppResult<ChatTimelineMessage> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val sent = webSocketClient.sendPrivateMessage(
            targetUserId = peerId,
            targetUserName = peerName,
            senderId = session.id,
            content = content,
        )
        if (!sent) error("WebSocket 未连接")
        buildOutgoingMessage(
            fromUserId = session.id,
            fromUserName = session.name,
            peerId = peerId,
            content = content,
            clientId = clientId,
            sendStatus = OutgoingMessageStatus.SENDING,
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "发送失败", it) },
    )

    suspend fun clearAndReload(peerId: String): AppResult<HistoryPageResult> = runCatching {
        messageDao.clearByPeer(peerId)
        when (val result = loadHistory(peerId = peerId, isFirst = true, firstTid = "0")) {
            is AppResult.Success -> result.data
            is AppResult.Error -> error(result.message)
        }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "重新加载失败", it) },
    )

    suspend fun requestUserInfo(peerId: String): AppResult<Unit> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val sent = webSocketClient.sendShowUserLoginInfo(senderId = session.id, targetUserId = peerId)
        if (!sent) error("WebSocket 未连接，暂时无法查询对方信息")
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "请求用户信息失败", it) },
    )

    suspend fun toggleFavorite(peerId: String, favorite: Boolean): AppResult<Boolean> = runCatching {
        val session = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val response = if (favorite) {
            chatApiService.cancelFavorite(
                myUserId = session.id,
                userToId = peerId,
                cookieData = session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } else {
            chatApiService.toggleFavorite(
                myUserId = session.id,
                userToId = peerId,
                cookieData = session.cookie,
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        }
        if (response.code != 0) error(response.msg ?: response.message ?: "收藏操作失败")
        val newValue = !favorite
        conversationDao.getById(peerId)?.let { current ->
            conversationDao.upsert(current.copy(isFavorite = newValue))
        }
        newValue
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "收藏操作失败", it) },
    )

    suspend fun blacklist(peerId: String): AppResult<Unit> = runCatching {
        val sent = webSocketClient.sendWarningReport(
            targetUserId = peerId,
            warningId = UUID.randomUUID().toString(),
        )
        if (!sent) error("连接已断开，请刷新页面重试")
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "发送拉黑请求失败", it) },
    )

    suspend fun getFavoriteState(peerId: String): Boolean = conversationDao.getById(peerId)?.isFavorite ?: false

    fun connectionState() = webSocketClient.state
    fun inboundEvents() = webSocketClient.events

    private fun buildOutgoingMessage(
        fromUserId: String,
        fromUserName: String,
        peerId: String,
        content: String,
        clientId: String,
        sendStatus: OutgoingMessageStatus,
        sendError: String? = null,
    ): ChatTimelineMessage {
        val type = inferMessageType(content)
        val mediaUrl = if (type == ChatMessageType.TEXT) "" else content.removePrefix("[").removeSuffix("]")
        return ChatTimelineMessage(
            id = clientId,
            fromUserId = fromUserId,
            fromUserName = fromUserName,
            toUserId = peerId,
            content = content,
            time = nowTimeLabel(),
            isSelf = true,
            type = type,
            mediaUrl = mediaUrl,
            fileName = if (type == ChatMessageType.TEXT) "" else inferFileName(content),
            clientId = clientId,
            sendStatus = sendStatus,
            sendError = sendError,
        )
    }
}

data class ChatRoomUiState(
    val loading: Boolean = true,
    val loadingMore: Boolean = false,
    val mediaSheetVisible: Boolean = false,
    val mediaUploading: Boolean = false,
    val historyMediaLoading: Boolean = false,
    val connectionStateLabel: String = "Idle",
    val messages: List<ChatTimelineMessage> = emptyList(),
    val draft: String = "",
    val peerFavorite: Boolean = false,
    val isPeerTyping: Boolean = false,
    val peerStatusMessage: String = "",
    val historyCursor: String? = null,
    val hasMoreHistory: Boolean = false,
    val uploadedMedia: List<ChatMediaAsset> = emptyList(),
    val historyMedia: List<ChatMediaAsset> = emptyList(),
    val scrollToBottomVersion: Int = 0,
    val message: String? = null,
)

@HiltViewModel
class ChatRoomViewModel @Inject constructor(
    private val repository: ChatRoomRepository,
) : ViewModel() {
    var uiState by mutableStateOf(ChatRoomUiState())
        private set

    private var boundPeerId: String? = null
    private var observersStarted = false
    private var pendingUserInfoPeerId: String? = null

    fun bind(peerId: String) {
        val isPeerChanged = boundPeerId != peerId
        boundPeerId = peerId
        if (isPeerChanged) {
            pendingUserInfoPeerId = peerId
            uiState = uiState.copy(
                loading = true,
                loadingMore = false,
                mediaSheetVisible = false,
                mediaUploading = false,
                historyMediaLoading = false,
                messages = emptyList(),
                message = null,
                isPeerTyping = false,
                peerStatusMessage = "",
                historyCursor = null,
                hasMoreHistory = false,
                historyMedia = emptyList(),
            )
        }

        viewModelScope.launch {
            when (val result = repository.ensureConnected()) {
                is AppResult.Success -> uiState = uiState.copy(connectionStateLabel = result.data.toDisplayText())
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }

        if (isPeerChanged) {
            refreshFavoriteState(peerId)
            loadHistory(peerId)
        }

        if (!observersStarted) {
            observersStarted = true
            observeConnectionState()
            observeInboundEvents()
        }
    }

    private fun loadHistory(peerId: String) {
        viewModelScope.launch {
            when (val result = repository.loadHistory(peerId = peerId, isFirst = true, firstTid = "0")) {
                is AppResult.Success -> {
                    val mergedMessages = mergeTimelineMessages(
                        current = uiState.messages,
                        incoming = result.data.messages,
                    )
                    val nextCursor = mergedMessages.minNumericTid() ?: result.data.historyCursor
                    uiState = uiState.copy(
                        loading = false,
                        loadingMore = false,
                        messages = mergedMessages,
                        historyCursor = nextCursor,
                        hasMoreHistory = result.data.hasMoreHistory && nextCursor != null,
                        scrollToBottomVersion = if (mergedMessages.isNotEmpty()) {
                            uiState.scrollToBottomVersion + 1
                        } else {
                            uiState.scrollToBottomVersion
                        },
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, loadingMore = false, message = result.message)
            }
        }
    }

    fun loadMoreHistory(peerId: String) {
        val cursor = uiState.historyCursor
        if (uiState.loading || uiState.loadingMore || cursor.isNullOrBlank()) {
            if (!uiState.loading && !uiState.loadingMore && cursor.isNullOrBlank()) {
                uiState = uiState.copy(hasMoreHistory = false)
            }
            return
        }
        viewModelScope.launch {
            uiState = uiState.copy(loadingMore = true, message = null)
            when (val result = repository.loadHistory(peerId = peerId, isFirst = false, firstTid = cursor)) {
                is AppResult.Success -> {
                    val mergedMessages = mergeTimelineMessages(
                        current = uiState.messages,
                        incoming = result.data.messages,
                    )
                    val nextCursor = mergedMessages.minNumericTid() ?: result.data.historyCursor
                    uiState = uiState.copy(
                        loadingMore = false,
                        messages = mergedMessages,
                        historyCursor = nextCursor,
                        hasMoreHistory = result.data.hasMoreHistory && nextCursor != null,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loadingMore = false, message = result.message)
            }
        }
    }

    fun openMediaSheet(peerId: String) {
        uiState = uiState.copy(mediaSheetVisible = true)
        if (!uiState.historyMediaLoading && uiState.historyMedia.isEmpty()) {
            loadHistoryMedia(peerId)
        }
    }

    fun closeMediaSheet() {
        if (uiState.mediaSheetVisible) {
            uiState = uiState.copy(mediaSheetVisible = false)
        }
    }

    fun importMtPhotoMedia(localPath: String, localFilename: String) {
        importExternalMedia(
            localPath = localPath,
            localFilename = localFilename,
            successMessage = "mtPhoto 已导入并加入待发送列表",
        )
    }

    fun importDouyinMedia(localPath: String, localFilename: String) {
        importExternalMedia(
            localPath = localPath,
            localFilename = localFilename,
            successMessage = "抖音媒体已导入并加入待发送列表",
        )
    }

    private fun importExternalMedia(localPath: String, localFilename: String, successMessage: String) {
        if (uiState.mediaUploading) return
        viewModelScope.launch {
            uiState = uiState.copy(mediaUploading = true, message = null)
            when (val result = repository.reuploadLocalMedia(localPath = localPath, localFilename = localFilename)) {
                is AppResult.Success -> {
                    val mergedUploads = listOf(result.data) + uiState.uploadedMedia.filterNot { it.id == result.data.id }
                    uiState = uiState.copy(
                        mediaUploading = false,
                        mediaSheetVisible = true,
                        uploadedMedia = mergedUploads.take(20),
                        message = successMessage,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(mediaUploading = false, message = result.message)
            }
        }
    }

    fun uploadMedia(uri: Uri) {
        viewModelScope.launch {
            uiState = uiState.copy(mediaUploading = true, message = null)
            when (val result = repository.uploadMedia(uri)) {
                is AppResult.Success -> {
                    val mergedUploads = listOf(result.data) + uiState.uploadedMedia.filterNot { it.id == result.data.id }
                    uiState = uiState.copy(
                        mediaUploading = false,
                        uploadedMedia = mergedUploads.take(20),
                        message = when (result.data.type) {
                            ChatMessageType.IMAGE -> "图片上传成功，点击可发送"
                            ChatMessageType.VIDEO -> "视频上传成功，点击可发送"
                            else -> "文件上传成功，点击可发送"
                        },
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(mediaUploading = false, message = result.message)
            }
        }
    }

    fun loadHistoryMedia(peerId: String) {
        viewModelScope.launch {
            uiState = uiState.copy(historyMediaLoading = true)
            when (val result = repository.loadChatHistoryMedia(peerId)) {
                is AppResult.Success -> uiState = uiState.copy(
                    historyMediaLoading = false,
                    historyMedia = result.data,
                )
                is AppResult.Error -> uiState = uiState.copy(historyMediaLoading = false, message = result.message)
            }
        }
    }

    fun sendUploadedMedia(peerId: String, peerName: String, media: ChatMediaAsset) {
        val clientId = "media_${System.currentTimeMillis()}"
        viewModelScope.launch {
            when (val result = repository.sendUploadedMedia(peerId = peerId, peerName = peerName, media = media, clientId = clientId)) {
                is AppResult.Success -> {
                    val displayMessage = repository.hydrateTimelineMessage(result.data)
                    uiState = uiState.copy(
                        mediaSheetVisible = false,
                        messages = appendTimelineMessage(uiState.messages, displayMessage),
                        scrollToBottomVersion = uiState.scrollToBottomVersion + 1,
                    )
                    watchOutgoingTimeout(clientId)
                }
                is AppResult.Error -> {
                    val failedMessage = ChatTimelineMessage(
                        id = clientId,
                        fromUserId = "self",
                        fromUserName = "我",
                        toUserId = peerId,
                        content = "[${media.remotePath.ifBlank { media.localFilename }}]",
                        time = nowTimeLabel(),
                        isSelf = true,
                        type = media.type,
                        mediaUrl = media.url,
                        fileName = media.localFilename.ifBlank { inferFileName(media.remotePath) },
                        clientId = clientId,
                        sendStatus = OutgoingMessageStatus.FAILED,
                        sendError = result.message,
                    )
                    uiState = uiState.copy(
                        messages = appendTimelineMessage(uiState.messages, failedMessage),
                        scrollToBottomVersion = uiState.scrollToBottomVersion + 1,
                        message = result.message,
                    )
                }
            }
        }
    }

    private fun refreshFavoriteState(peerId: String) {
        viewModelScope.launch {
            uiState = uiState.copy(peerFavorite = repository.getFavoriteState(peerId))
        }
    }

    private fun observeConnectionState() {
        viewModelScope.launch {
            repository.connectionState().collectLatest { state ->
                uiState = uiState.copy(connectionStateLabel = state.toDisplayText())
                if (state == WebSocketState.Connected) {
                    val peerId = pendingUserInfoPeerId
                    if (!peerId.isNullOrBlank()) {
                        requestPeerInfo(peerId = peerId, silent = true)
                    }
                }
            }
        }
    }

    private fun observeInboundEvents() {
        viewModelScope.launch {
            repository.inboundEvents().collectLatest { event ->
                when (event) {
                    is LiaoWsEvent.ChatMessage -> handleIncomingChat(event.timelineMessage)
                    is LiaoWsEvent.Typing -> handleTypingEvent(event)
                    is LiaoWsEvent.OnlineStatus -> handleOnlineStatus(event)
                    is LiaoWsEvent.Forceout -> uiState = uiState.copy(message = event.reason)
                    is LiaoWsEvent.ConnectNotice -> {
                        if (event.message.isNotBlank()) {
                            uiState = uiState.copy(message = event.message)
                        }
                    }
                    is LiaoWsEvent.MatchCancelled -> uiState = uiState.copy(message = event.message)
                    is LiaoWsEvent.Notice -> {
                        if (event.message.isNotBlank()) {
                            uiState = uiState.copy(message = event.message)
                        }
                    }
                    is LiaoWsEvent.Unknown -> LiaoLogger.i("ChatRoomViewModel", "收到未识别 WS 消息: code=${event.envelope?.code ?: -1}, act=${event.envelope?.act.orEmpty()}, bytes=${event.raw.length}")
                    is LiaoWsEvent.MatchSuccess -> Unit
                }
            }
        }
    }

    private fun handleTypingEvent(event: LiaoWsEvent.Typing) {
        val currentPeerId = boundPeerId.orEmpty()
        if (currentPeerId.isBlank() || currentPeerId != event.peerId) return
        uiState = uiState.copy(isPeerTyping = event.typing)
    }

    private fun handleOnlineStatus(event: LiaoWsEvent.OnlineStatus) {
        val currentPeerId = boundPeerId.orEmpty()
        if (currentPeerId.isBlank()) return
        pendingUserInfoPeerId = null
        uiState = uiState.copy(peerStatusMessage = event.message, message = event.message)
    }

    private fun handleIncomingChat(message: ChatTimelineMessage) {
        val currentPeerId = boundPeerId.orEmpty()
        if (currentPeerId.isBlank()) return
        val isCurrentPeerMessage = message.fromUserId == currentPeerId || message.toUserId == currentPeerId
        if (!isCurrentPeerMessage) {
            LiaoLogger.i("ChatRoomViewModel", "忽略非当前会话消息: peerId=$currentPeerId, rawMessageId=${message.id}")
            return
        }

        viewModelScope.launch {
            val displayMessage = repository.hydrateTimelineMessage(message)
            val mergedMessages = if (displayMessage.isSelf) {
                mergeEchoMessage(uiState.messages, displayMessage)
            } else {
                appendTimelineMessage(uiState.messages, displayMessage)
            }
            uiState = uiState.copy(
                messages = mergedMessages,
                isPeerTyping = if (displayMessage.isSelf) uiState.isPeerTyping else false,
                scrollToBottomVersion = uiState.scrollToBottomVersion + 1,
            )
        }
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }

    fun requestPeerInfo(peerId: String, silent: Boolean = false) {
        viewModelScope.launch {
            when (val result = repository.requestUserInfo(peerId)) {
                is AppResult.Success -> {
                    pendingUserInfoPeerId = peerId
                    if (!silent) {
                        uiState = uiState.copy(message = "已请求对方在线状态")
                    }
                }
                is AppResult.Error -> {
                    if (silent) {
                        pendingUserInfoPeerId = peerId
                    } else {
                        uiState = uiState.copy(message = result.message)
                    }
                }
            }
        }
    }

    fun updateDraft(value: String) {
        uiState = uiState.copy(draft = value)
    }

    fun sendText(peerId: String, peerName: String) {
        if (uiState.draft.isBlank()) return
        val draft = uiState.draft
        val clientId = "local_${System.currentTimeMillis()}"
        viewModelScope.launch {
            when (val result = repository.sendText(peerId, peerName, draft, clientId)) {
                is AppResult.Success -> {
                    val displayMessage = repository.hydrateTimelineMessage(result.data)
                    uiState = uiState.copy(
                        draft = "",
                        messages = appendTimelineMessage(uiState.messages, displayMessage),
                        scrollToBottomVersion = uiState.scrollToBottomVersion + 1,
                    )
                    watchOutgoingTimeout(clientId)
                }
                is AppResult.Error -> {
                    val failed = ChatTimelineMessage(
                        id = clientId,
                        fromUserId = "self",
                        fromUserName = "我",
                        toUserId = peerId,
                        content = draft,
                        time = nowTimeLabel(),
                        isSelf = true,
                        type = inferMessageType(draft),
                        mediaUrl = if (inferMessageType(draft) == ChatMessageType.TEXT) "" else draft.removePrefix("[").removeSuffix("]"),
                        fileName = if (inferMessageType(draft) == ChatMessageType.TEXT) "" else inferFileName(draft),
                        clientId = clientId,
                        sendStatus = OutgoingMessageStatus.FAILED,
                        sendError = result.message,
                    )
                    uiState = uiState.copy(
                        messages = appendTimelineMessage(uiState.messages, failed),
                        scrollToBottomVersion = uiState.scrollToBottomVersion + 1,
                        message = result.message,
                    )
                }
            }
        }
    }

    fun retryFailedMessage(peerId: String, peerName: String, clientId: String) {
        val existing = uiState.messages.firstOrNull { it.clientId == clientId || it.id == clientId } ?: return
        uiState = uiState.copy(
            messages = uiState.messages.map { message ->
                if (message.clientId == clientId || message.id == clientId) {
                    message.copy(sendStatus = OutgoingMessageStatus.SENDING, sendError = null)
                } else {
                    message
                }
            }
        )
        viewModelScope.launch {
            when (val result = repository.sendText(peerId, peerName, existing.content, existing.clientId.ifBlank { clientId })) {
                is AppResult.Success -> {
                    val retryClientId = existing.clientId.ifBlank { clientId }
                    val displayMessage = repository.hydrateTimelineMessage(result.data)
                    uiState = uiState.copy(
                        messages = appendTimelineMessage(uiState.messages, displayMessage),
                    )
                    watchOutgoingTimeout(retryClientId)
                }
                is AppResult.Error -> {
                    markMessageFailed(existing.clientId.ifBlank { clientId }, result.message)
                    uiState = uiState.copy(message = result.message)
                }
            }
        }
    }

    private fun watchOutgoingTimeout(clientId: String) {
        viewModelScope.launch {
            delay(10_000L)
            val target = uiState.messages.firstOrNull { it.clientId == clientId }
            if (target?.sendStatus == OutgoingMessageStatus.SENDING) {
                markMessageFailed(clientId, "发送超时，请重试")
            }
        }
    }

    private fun markMessageFailed(clientId: String, errorMessage: String) {
        uiState = uiState.copy(
            messages = uiState.messages.map { message ->
                if (message.clientId == clientId) {
                    message.copy(sendStatus = OutgoingMessageStatus.FAILED, sendError = errorMessage)
                } else {
                    message
                }
            }
        )
    }

    fun toggleFavorite(peerId: String) {
        viewModelScope.launch {
            when (val result = repository.toggleFavorite(peerId = peerId, favorite = uiState.peerFavorite)) {
                is AppResult.Success -> uiState = uiState.copy(
                    peerFavorite = result.data,
                    message = if (result.data) "收藏成功" else "已取消收藏",
                )
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun blacklist(peerId: String) {
        viewModelScope.launch {
            when (val result = repository.blacklist(peerId)) {
                is AppResult.Success -> uiState = uiState.copy(message = "已发送拉黑请求")
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun clearAndReload(peerId: String) {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, messages = emptyList(), message = "正在重新加载聊天记录...")
            when (val result = repository.clearAndReload(peerId)) {
                is AppResult.Success -> {
                    val nextCursor = result.data.messages.minNumericTid() ?: result.data.historyCursor
                    uiState = uiState.copy(
                        loading = false,
                        loadingMore = false,
                        messages = result.data.messages,
                        historyCursor = nextCursor,
                        hasMoreHistory = result.data.hasMoreHistory && nextCursor != null,
                        scrollToBottomVersion = if (result.data.messages.isNotEmpty()) {
                            uiState.scrollToBottomVersion + 1
                        } else {
                            uiState.scrollToBottomVersion
                        },
                        message = if (result.data.messages.isEmpty()) "暂无聊天记录" else "已重新加载 ${result.data.messages.size} 条消息",
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(loading = false, loadingMore = false, message = result.message)
            }
        }
    }
}

private fun ChatTimelineMessage.toEntity(peerId: String): MessageEntity = MessageEntity(
    id = id,
    peerId = peerId,
    fromUserId = fromUserId,
    fromUserName = fromUserName,
    toUserId = toUserId,
    content = content,
    time = time,
    isSelf = isSelf,
    type = type.name,
    mediaUrl = mediaUrl,
    fileName = fileName,
)

private fun List<ChatTimelineMessage>.minNumericTid(): String? =
    asSequence()
        .mapNotNull { message ->
            val rawTid = message.id.trim()
            rawTid.toLongOrNull()?.let { numericTid -> rawTid to numericTid }
        }
        .minByOrNull { (_, numericTid) -> numericTid }
        ?.first

private fun mergeTimelineMessages(
    current: List<ChatTimelineMessage>,
    incoming: List<ChatTimelineMessage>,
): List<ChatTimelineMessage> {
    val merged = linkedMapOf<String, ChatTimelineMessage>()
    (incoming + current).forEach { message ->
        val key = message.clientId.ifBlank { message.id }
        merged[key] = message
    }
    return merged.values.toList()
}

private fun appendTimelineMessage(
    current: List<ChatTimelineMessage>,
    incoming: ChatTimelineMessage,
): List<ChatTimelineMessage> {
    val targetKey = incoming.clientId.ifBlank { incoming.id }
    if (current.any { it.clientId.ifBlank { it.id } == targetKey || it.id == incoming.id }) {
        return current.map { existing ->
            if (existing.clientId.ifBlank { existing.id } == targetKey || existing.id == incoming.id) incoming else existing
        }
    }
    return current + incoming
}

private fun mergeEchoMessage(
    current: List<ChatTimelineMessage>,
    incoming: ChatTimelineMessage,
): List<ChatTimelineMessage> {
    val exactTidIndex = current.indexOfFirst { it.id == incoming.id }
    if (exactTidIndex >= 0) {
        return current.mapIndexed { index, message ->
            if (index == exactTidIndex) incoming.copy(
                clientId = message.clientId,
                sendStatus = OutgoingMessageStatus.SENT,
                sendError = null,
            ) else message
        }
    }

    val optimisticIndex = current.indexOfLast { message ->
        message.isSelf &&
            message.sendStatus != OutgoingMessageStatus.SENT &&
            message.toUserId == incoming.toUserId &&
            message.type == incoming.type &&
            normalizeTextForMatch(message.content) == normalizeTextForMatch(incoming.content)
    }
    if (optimisticIndex >= 0) {
        return current.mapIndexed { index, message ->
            if (index == optimisticIndex) incoming.copy(
                clientId = message.clientId.ifBlank { message.id },
                sendStatus = OutgoingMessageStatus.SENT,
                sendError = null,
            ) else message
        }
    }
    return appendTimelineMessage(current, incoming.copy(sendStatus = OutgoingMessageStatus.SENT, sendError = null))
}

private fun ChatTimelineMessage.displayContent(): String = when (type) {
    ChatMessageType.IMAGE -> if (fileName.isNotBlank()) "[图片] $fileName" else "[图片]"
    ChatMessageType.VIDEO -> if (fileName.isNotBlank()) "[视频] $fileName" else "[视频]"
    ChatMessageType.FILE -> if (fileName.isNotBlank()) "[文件] $fileName" else "[文件]"
    ChatMessageType.TEXT -> content
}

private fun OutgoingMessageStatus.displayText(): String = when (this) {
    OutgoingMessageStatus.SENDING -> "发送中"
    OutgoingMessageStatus.SENT -> "已发送"
    OutgoingMessageStatus.FAILED -> "发送失败"
}

private fun WebSocketState.toDisplayText(): String = when (this) {
    WebSocketState.Idle -> "Idle"
    WebSocketState.Connecting -> "Connecting"
    WebSocketState.Connected -> "Connected"
    WebSocketState.Reconnecting -> "Reconnecting"
    is WebSocketState.Forceout -> {
        val remainingMillis = (forbiddenUntilMillis - System.currentTimeMillis()).coerceAtLeast(0L)
        val remainingSeconds = (remainingMillis + 999L) / 1000L
        "Forceout(${remainingSeconds}s)"
    }
    WebSocketState.Closed -> "Closed"
}

private fun nowTimeLabel(): String =
    SimpleDateFormat("HH:mm:ss", Locale.getDefault()).format(Date())

private fun ChatMessageType.openMimeType(): String = when (this) {
    ChatMessageType.IMAGE -> "image/*"
    ChatMessageType.VIDEO -> "video/*"
    ChatMessageType.FILE -> "*/*"
    ChatMessageType.TEXT -> "text/plain"
}

private fun ChatMessageType.displayLabel(fileName: String): String = when (this) {
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

@Composable
private fun ChatMessageContent(
    message: ChatTimelineMessage,
    onOpenMedia: (String, ChatMessageType) -> Unit,
) {
    when (message.type) {
        ChatMessageType.TEXT -> Text(text = message.displayContent())
        ChatMessageType.IMAGE -> {
            if (message.mediaUrl.isNotBlank()) {
                AsyncImage(
                    model = message.mediaUrl,
                    contentDescription = message.fileName.ifBlank { "图片消息" },
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(max = 220.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .clickable { onOpenMedia(message.mediaUrl, ChatMessageType.IMAGE) },
                    contentScale = ContentScale.Crop,
                )
                if (message.fileName.isNotBlank()) {
                    Text(
                        text = message.fileName,
                        style = MaterialTheme.typography.labelSmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            } else {
                Text(text = message.displayContent())
            }
        }
        ChatMessageType.VIDEO,
        ChatMessageType.FILE -> {
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(enabled = message.mediaUrl.isNotBlank()) { onOpenMedia(message.mediaUrl, message.type) },
            ) {
                Row(
                    modifier = Modifier.padding(12.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(
                        modifier = Modifier.size(44.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(if (message.type == ChatMessageType.VIDEO) "视频" else "文件", style = MaterialTheme.typography.labelMedium)
                    }
                    Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text(
                            text = message.type.displayLabel(message.fileName),
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        Text(
                            text = if (message.mediaUrl.isNotBlank()) "点击打开" else "暂无法打开",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.outline,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun MediaAssetCard(
    media: ChatMediaAsset,
    actionLabel: String,
    onClick: () -> Unit,
) {
    Card(
        modifier = Modifier
            .size(width = 96.dp, height = 112.dp)
            .clickable(onClick = onClick),
    ) {
        Column(
            modifier = Modifier.padding(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(64.dp),
                contentAlignment = Alignment.Center,
            ) {
                if (media.type == ChatMessageType.IMAGE && media.url.isNotBlank()) {
                    AsyncImage(
                        model = media.url,
                        contentDescription = media.localFilename.ifBlank { "图片" },
                        modifier = Modifier
                            .fillMaxSize()
                            .clip(RoundedCornerShape(10.dp)),
                        contentScale = ContentScale.Crop,
                    )
                } else {
                    Text(
                        text = when (media.type) {
                            ChatMessageType.VIDEO -> "视频"
                            ChatMessageType.FILE -> "文件"
                            else -> "图片"
                        },
                        style = MaterialTheme.typography.labelMedium,
                    )
                }
            }
            Text(
                text = media.localFilename.ifBlank { media.type.displayLabel("") },
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.labelSmall,
            )
            Text(actionLabel, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
        }
    }
}

@Composable
private fun ChatRoomMediaSheet(
    state: ChatRoomUiState,
    onDismiss: () -> Unit,
    onPickImage: () -> Unit,
    onPickVideo: () -> Unit,
    onPickFile: () -> Unit,
    onOpenMtPhoto: () -> Unit,
    onOpenDouyin: () -> Unit,
    onSendUploaded: (ChatMediaAsset) -> Unit,
    onOpenHistoryMedia: (ChatMediaAsset) -> Unit,
    onReloadHistoryMedia: () -> Unit,
) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("媒体发送", style = MaterialTheme.typography.titleMedium)
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                OutlinedButton(onClick = onPickImage, enabled = !state.mediaUploading, modifier = Modifier.weight(1f)) {
                    Text("图片")
                }
                OutlinedButton(onClick = onPickVideo, enabled = !state.mediaUploading, modifier = Modifier.weight(1f)) {
                    Text("视频")
                }
                OutlinedButton(onClick = onPickFile, enabled = !state.mediaUploading, modifier = Modifier.weight(1f)) {
                    Text("文件")
                }
            }
            OutlinedButton(
                onClick = onOpenMtPhoto,
                enabled = !state.mediaUploading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("mtPhoto 相册")
            }
            OutlinedButton(
                onClick = onOpenDouyin,
                enabled = !state.mediaUploading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("抖音下载")
            }
            if (state.mediaUploading) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                    CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                    Text("正在上传媒体...", style = MaterialTheme.typography.bodySmall)
                }
            }
            HorizontalDivider()
            Text("已上传文件（点击发送）", style = MaterialTheme.typography.titleSmall)
            if (state.uploadedMedia.isEmpty()) {
                Text("暂无已上传文件", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
            } else {
                LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    items(state.uploadedMedia, key = { it.id }) { media ->
                        MediaAssetCard(media = media, actionLabel = "点击发送", onClick = { onSendUploaded(media) })
                    }
                }
            }
            HorizontalDivider()
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("聊天历史媒体", style = MaterialTheme.typography.titleSmall)
                TextButton(onClick = onReloadHistoryMedia, enabled = !state.historyMediaLoading) {
                    Text("刷新")
                }
            }
            when {
                state.historyMediaLoading -> {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                        Text("正在加载聊天历史媒体...", style = MaterialTheme.typography.bodySmall)
                    }
                }
                state.historyMedia.isEmpty() -> {
                    Text("暂无聊天历史媒体", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
                }
                else -> {
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        items(state.historyMedia, key = { it.id }) { media ->
                            MediaAssetCard(media = media, actionLabel = "点击查看", onClick = { onOpenHistoryMedia(media) })
                        }
                    }
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
        }
    }
}

@Composable
fun ChatRoomScreen(
    peerId: String,
    peerName: String,
    viewModel: ChatRoomViewModel,
    onBack: () -> Unit,
    onOpenMtPhoto: () -> Unit,
    onOpenDouyin: () -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }
    val listState = rememberLazyListState()
    val context = LocalContext.current
    val imagePickerLauncher = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        uri?.let(viewModel::uploadMedia)
    }
    val videoPickerLauncher = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        uri?.let(viewModel::uploadMedia)
    }
    val filePickerLauncher = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        uri?.let(viewModel::uploadMedia)
    }

    LaunchedEffect(peerId) {
        viewModel.bind(peerId)
    }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    LaunchedEffect(state.scrollToBottomVersion) {
        if (state.scrollToBottomVersion > 0 && state.messages.isNotEmpty()) {
            listState.animateScrollToItem(state.messages.lastIndex)
        }
    }

    if (state.mediaSheetVisible) {
        ChatRoomMediaSheet(
            state = state,
            onDismiss = viewModel::closeMediaSheet,
            onPickImage = { imagePickerLauncher.launch("image/*") },
            onPickVideo = { videoPickerLauncher.launch("video/*") },
            onPickFile = { filePickerLauncher.launch("*/*") },
            onOpenMtPhoto = {
                viewModel.closeMediaSheet()
                onOpenMtPhoto()
            },
            onOpenDouyin = {
                viewModel.closeMediaSheet()
                onOpenDouyin()
            },
            onSendUploaded = { media -> viewModel.sendUploadedMedia(peerId = peerId, peerName = peerName, media = media) },
            onOpenHistoryMedia = { media -> openMediaExternally(context, media.url, media.type) },
            onReloadHistoryMedia = { viewModel.loadHistoryMedia(peerId) },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(peerName) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(
                    modifier = Modifier.padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text("连接状态：${state.connectionStateLabel}")
                    if (state.peerStatusMessage.isNotBlank()) {
                        Text(
                            text = state.peerStatusMessage,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.secondary,
                        )
                    }
                    if (state.isPeerTyping) {
                        Text(
                            text = "对方正在输入...",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.primary,
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedButton(onClick = { viewModel.toggleFavorite(peerId) }, modifier = Modifier.weight(1f)) {
                            Text(if (state.peerFavorite) "取消收藏" else "收藏")
                        }
                        OutlinedButton(onClick = { viewModel.requestPeerInfo(peerId) }, modifier = Modifier.weight(1f)) {
                            Text("在线状态")
                        }
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedButton(onClick = { viewModel.blacklist(peerId) }, modifier = Modifier.weight(1f)) {
                            Text("拉黑")
                        }
                        OutlinedButton(onClick = { viewModel.clearAndReload(peerId) }, modifier = Modifier.weight(1f)) {
                            Text("清空重载")
                        }
                    }
                }
            }
            if (state.loading) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator()
                }
            } else {
                LazyColumn(
                    modifier = Modifier.weight(1f),
                    state = listState,
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    item(key = "load_more_history") {
                        Box(
                            modifier = Modifier.fillMaxWidth(),
                            contentAlignment = Alignment.Center,
                        ) {
                            OutlinedButton(
                                onClick = { viewModel.loadMoreHistory(peerId) },
                                enabled = !state.loadingMore && state.hasMoreHistory,
                            ) {
                                Text(
                                    when {
                                        state.loadingMore -> "加载中..."
                                        state.hasMoreHistory -> "查看历史消息"
                                        else -> "暂无更多历史消息"
                                    }
                                )
                            }
                        }
                    }
                    items(state.messages, key = { it.clientId.ifBlank { it.id } }) { message ->
                        Card(modifier = Modifier.fillMaxWidth()) {
                            Column(
                                modifier = Modifier.padding(12.dp),
                                verticalArrangement = Arrangement.spacedBy(4.dp),
                            ) {
                                Text(
                                    text = if (message.isSelf) "我" else message.fromUserName.ifBlank { peerName },
                                    style = MaterialTheme.typography.labelMedium,
                                    color = if (message.isSelf) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.secondary,
                                )
                                ChatMessageContent(
                                    message = message,
                                    onOpenMedia = { url, type -> openMediaExternally(context, url, type) },
                                )
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Text(text = message.time, style = MaterialTheme.typography.labelSmall)
                                    if (message.isSelf) {
                                        Row(verticalAlignment = Alignment.CenterVertically) {
                                            Text(
                                                text = message.sendStatus.displayText(),
                                                style = MaterialTheme.typography.labelSmall,
                                                color = if (message.sendStatus == OutgoingMessageStatus.FAILED) {
                                                    MaterialTheme.colorScheme.error
                                                } else {
                                                    MaterialTheme.colorScheme.outline
                                                },
                                            )
                                            if (message.sendStatus == OutgoingMessageStatus.FAILED) {
                                                TextButton(
                                                    onClick = {
                                                        viewModel.retryFailedMessage(
                                                            peerId = peerId,
                                                            peerName = peerName,
                                                            clientId = message.clientId.ifBlank { message.id },
                                                        )
                                                    }
                                                ) {
                                                    Text("重试")
                                                }
                                            }
                                        }
                                    }
                                }
                                if (!message.sendError.isNullOrBlank()) {
                                    Text(
                                        text = message.sendError,
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.error,
                                    )
                                }
                            }
                        }
                    }
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(
                    modifier = Modifier.weight(1f),
                    value = state.draft,
                    onValueChange = viewModel::updateDraft,
                    label = { Text("输入消息") },
                )
                OutlinedButton(onClick = { viewModel.openMediaSheet(peerId) }) {
                    Text("媒体")
                }
                Button(onClick = { viewModel.sendText(peerId, peerName) }) {
                    Text("发送")
                }
            }
        }
    }
}
