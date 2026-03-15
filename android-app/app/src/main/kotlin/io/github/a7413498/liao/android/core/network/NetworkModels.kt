/*
 * 网络模型文件描述 Android 客户端与 Go 服务端之间的主要 DTO。
 * 对于返回结构不统一的接口，客户端会保留足够宽松的字段以保证兼容性。
 */
package io.github.a7413498.liao.android.core.network

import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.GlobalFavoriteItem
import io.github.a7413498.liao.android.core.common.OutgoingMessageStatus
import io.github.a7413498.liao.android.core.common.generateCookie
import io.github.a7413498.liao.android.core.common.generateRandomIp
import io.github.a7413498.liao.android.core.common.inferFileName
import io.github.a7413498.liao.android.core.common.inferMessageType
import io.github.a7413498.liao.android.core.common.inferPrivateMessageIsSelf
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

private val historyParserJson = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
    isLenient = true
}

@Serializable
data class ApiEnvelope<T>(
    val code: Int = -1,
    val msg: String? = null,
    val message: String? = null,
    val data: T? = null,
    val token: String? = null,
    val valid: Boolean? = null,
)

@Serializable
data class IdentityDto(
    val id: String,
    val name: String,
    val sex: String,
    val createdAt: String? = null,
    val lastUsedAt: String? = null,
)

@Serializable
data class ChatUserDto(
    val id: String,
    val name: String? = null,
    val nickname: String? = null,
    val sex: String? = null,
    val ip: String? = null,
    @SerialName("address") val address: String? = null,
    val isFavorite: Boolean? = null,
    val lastMsg: String? = null,
    val lastTime: String? = null,
    val unreadCount: Int? = null,
)

@Serializable
data class ChatMessageUserDto(
    val id: String = "",
    val name: String? = null,
    val nickname: String? = null,
    val sex: String? = null,
    val ip: String? = null,
    @SerialName("address") val address: String? = null,
)

@Serializable
data class ChatMessageDto(
    val code: Int = 0,
    @SerialName("fromuser") val fromUser: ChatMessageUserDto = ChatMessageUserDto(),
    @SerialName("touser") val toUser: ChatMessageUserDto? = null,
    val content: String = "",
    val time: String = "",
    val tid: String = "",
    val act: String? = null,
    val type: String? = null,
    val forceout: Boolean? = null,
)

@Serializable
data class HistoryMessageDto(
    @SerialName("Tid") val tidUpper: String? = null,
    @SerialName("tid") val tidLower: String? = null,
    val id: String = "",
    @SerialName("toid") val toId: String? = null,
    val nickname: String? = null,
    val name: String? = null,
    val content: String = "",
    val time: String = "",
    val type: String? = null,
)

@Serializable
data class HistoryMessageEnvelope(
    val code: Int = 0,
    @SerialName("contents_list") val contentsList: List<HistoryMessageDto> = emptyList(),
)

@Serializable
data class ConnectionStatsDto(
    val active: Int = 0,
    val upstream: Int = 0,
    val downstream: Int = 0,
)

@Serializable
data class SystemConfigDto(
    val imagePortMode: String = "fixed",
    val imagePortFixed: String = "9006",
    val imagePortRealMinBytes: Long = 2048,
    val mtPhotoTimelineDeferSubfolderThreshold: Int = 10,
)

@Serializable
data class GenericJsonResponse(
    val raw: JsonElement? = null,
)

fun IdentityDto.toSession(cookie: String = generateCookie(id, name), ip: String = generateRandomIp(), area: String = "未知"): CurrentIdentitySession =
    CurrentIdentitySession(
        id = id,
        name = name,
        sex = sex,
        cookie = cookie,
        ip = ip,
        area = area,
    )

fun ChatUserDto.toPeer(isFavoriteOverride: Boolean? = null): ChatPeer = ChatPeer(
    id = id,
    name = nickname ?: name.orEmpty(),
    sex = sex.orEmpty(),
    ip = ip.orEmpty(),
    address = address.orEmpty(),
    isFavorite = isFavoriteOverride ?: isFavorite ?: false,
    lastMessage = lastMsg.orEmpty(),
    lastTime = lastTime.orEmpty(),
    unreadCount = unreadCount ?: 0,
)

fun ChatMessageDto.toTimeline(currentUserId: String): ChatTimelineMessage {
    val rawContent = content
    val inferredType = when (type?.lowercase()) {
        "image" -> ChatMessageType.IMAGE
        "video" -> ChatMessageType.VIDEO
        "file" -> ChatMessageType.FILE
        else -> inferMessageType(rawContent)
    }
    return ChatTimelineMessage(
        id = tid.ifBlank { "${time}_${fromUser.id}_${rawContent.hashCode()}" },
        fromUserId = fromUser.id,
        fromUserName = fromUser.nickname ?: fromUser.name.orEmpty(),
        toUserId = toUser?.id.orEmpty(),
        content = rawContent,
        time = time,
        isSelf = inferPrivateMessageIsSelf(currentUserId = currentUserId, fromUserId = fromUser.id),
        type = inferredType,
        mediaUrl = if (inferredType == ChatMessageType.TEXT) "" else rawContent.removePrefix("[").removeSuffix("]"),
        fileName = if (inferredType == ChatMessageType.TEXT) "" else inferFileName(rawContent),
        sendStatus = OutgoingMessageStatus.SENT,
    )
}

fun HistoryMessageDto.toTimeline(currentUserId: String, peerId: String, currentUserName: String): ChatTimelineMessage {
    val resolvedTid = tidUpper.orEmpty().ifBlank { tidLower.orEmpty() }
    val rawContent = content
    val inferredType = when (type?.lowercase()) {
        "image" -> ChatMessageType.IMAGE
        "video" -> ChatMessageType.VIDEO
        "file" -> ChatMessageType.FILE
        else -> inferMessageType(rawContent)
    }
    val isSelf = id.isNotBlank() && id != peerId
    val displayName = when {
        isSelf -> currentUserName
        !nickname.isNullOrBlank() -> nickname
        !name.isNullOrBlank() -> name
        else -> peerId.take(8)
    }
    return ChatTimelineMessage(
        id = resolvedTid.ifBlank { "${time}_${id}_${rawContent.hashCode()}" },
        fromUserId = if (isSelf) currentUserId else id,
        fromUserName = displayName,
        toUserId = if (isSelf) peerId else currentUserId,
        content = rawContent,
        time = time.ifBlank { "刚刚" },
        isSelf = isSelf,
        type = inferredType,
        mediaUrl = if (inferredType == ChatMessageType.TEXT) "" else rawContent.removePrefix("[").removeSuffix("]"),
        fileName = if (inferredType == ChatMessageType.TEXT) "" else inferFileName(rawContent),
        sendStatus = OutgoingMessageStatus.SENT,
    )
}

fun JsonElement.toHistoryMessageList(json: Json = historyParserJson): List<HistoryMessageDto> = when (this) {
    is JsonArray -> mapNotNull { element -> runCatching { json.decodeFromJsonElement<HistoryMessageDto>(element) }.getOrNull() }
    is JsonObject -> {
        val contents = this["contents_list"]
        when (contents) {
            is JsonArray -> contents.mapNotNull { element -> runCatching { json.decodeFromJsonElement<HistoryMessageDto>(element) }.getOrNull() }
            else -> runCatching { json.decodeFromJsonElement<HistoryMessageEnvelope>(this) }.getOrNull()?.contentsList.orEmpty()
        }
    }
    else -> emptyList()
}

fun JsonElement.toFavoriteItemOrNull(): GlobalFavoriteItem? {
    val root = this as? JsonObject ?: return null
    val id = root.stringOrNull("id")?.toIntOrNull() ?: return null
    return GlobalFavoriteItem(
        id = id,
        identityId = root.stringOrNull("identityId").orEmpty(),
        targetUserId = root.stringOrNull("targetUserId").orEmpty(),
        targetUserName = root.stringOrNull("targetUserName").orEmpty(),
        createTime = root.stringOrNull("createTime").orEmpty(),
    )
}

fun JsonObject.stringOrNull(key: String): String? =
    this[key]?.let { runCatching { it.jsonPrimitive.contentOrNull ?: it.jsonPrimitive.content }.getOrNull() }?.takeIf { it.isNotBlank() }
