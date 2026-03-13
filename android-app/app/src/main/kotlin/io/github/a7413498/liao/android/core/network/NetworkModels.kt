/*
 * 网络模型文件描述 Android 客户端与 Go 服务端之间的主要 DTO。
 * 对于返回结构不统一的接口，客户端会保留足够宽松的字段以保证兼容性。
 */
package io.github.a7413498.liao.android.core.network

import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.inferPrivateMessageIsSelf
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

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
data class ConnectionStatsDto(
    val active: Int = 0,
    val upstream: Int = 0,
    val downstream: Int = 0,
)

@Serializable
data class SystemConfigDto(
    val imagePortMode: String = "probe",
    val imagePortFixed: String = "9006",
    val imagePortRealMinBytes: Long = 0,
    val mtPhotoTimelineDeferSubfolderThreshold: Int = 0,
)

@Serializable
data class GenericJsonResponse(
    val raw: JsonElement? = null,
)

fun IdentityDto.toSession(cookie: String, ip: String, area: String = "未知"): CurrentIdentitySession =
    CurrentIdentitySession(
        id = id,
        name = name,
        sex = sex,
        cookie = cookie,
        ip = ip,
        area = area,
    )

fun ChatUserDto.toPeer(isFavoriteOverride: Boolean? = null): ChatPeer =
    ChatPeer(
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

fun ChatMessageDto.toTimeline(currentUserId: String): ChatTimelineMessage =
    ChatTimelineMessage(
        id = tid.ifBlank { "${time}_${fromUser.id}_${content.hashCode()}" },
        fromUserId = fromUser.id,
        fromUserName = fromUser.nickname ?: fromUser.name.orEmpty(),
        toUserId = toUser?.id.orEmpty(),
        content = content,
        time = time,
        isSelf = inferPrivateMessageIsSelf(currentUserId = currentUserId, fromUserId = fromUser.id),
    )
