/*
 * 通用文件承载基础结果类型、日志、会话模型与协议辅助函数。
 * 这些能力会被网络层、WebSocket、页面状态和本地缓存共同复用。
 */
package io.github.a7413498.liao.android.core.common

import android.util.Log
import java.security.MessageDigest
import kotlin.random.Random

sealed interface AppResult<out T> {
    data class Success<T>(val data: T) : AppResult<T>
    data class Error(val message: String, val cause: Throwable? = null) : AppResult<Nothing>
}

data class CurrentIdentitySession(
    val id: String,
    val name: String,
    val sex: String,
    val cookie: String,
    val ip: String,
    val area: String = "未知",
)

data class ChatPeer(
    val id: String,
    val name: String,
    val sex: String,
    val ip: String,
    val address: String,
    val isFavorite: Boolean = false,
    val lastMessage: String = "",
    val lastTime: String = "",
    val unreadCount: Int = 0,
)

enum class ChatMessageType {
    TEXT,
    IMAGE,
    VIDEO,
    FILE,
}

enum class OutgoingMessageStatus {
    SENDING,
    SENT,
    FAILED,
}

data class ChatTimelineMessage(
    val id: String,
    val fromUserId: String,
    val fromUserName: String,
    val toUserId: String,
    val content: String,
    val time: String,
    val isSelf: Boolean,
    val type: ChatMessageType = ChatMessageType.TEXT,
    val mediaUrl: String = "",
    val fileName: String = "",
    val clientId: String = "",
    val sendStatus: OutgoingMessageStatus = OutgoingMessageStatus.SENT,
    val sendError: String? = null,
) {
    val peerId: String
        get() = if (isSelf) toUserId else fromUserId

    fun lastMessagePreview(): String {
        val base = when (type) {
            ChatMessageType.IMAGE -> if (fileName.isNotBlank()) "[图片] $fileName" else "[图片]"
            ChatMessageType.VIDEO -> if (fileName.isNotBlank()) "[视频] $fileName" else "[视频]"
            ChatMessageType.FILE -> if (fileName.isNotBlank()) "[文件] $fileName" else "[文件]"
            ChatMessageType.TEXT -> content.ifBlank { "[空消息]" }
        }
        return if (isSelf) "我: $base" else base
    }
}

data class GlobalFavoriteItem(
    val id: Int,
    val identityId: String,
    val targetUserId: String,
    val targetUserName: String,
    val createTime: String,
)

object LiaoLogger {
    fun i(tag: String, message: String) = Log.i(tag, message)
    fun w(tag: String, message: String, throwable: Throwable? = null) = Log.w(tag, message, throwable)
    fun e(tag: String, message: String, throwable: Throwable? = null) = Log.e(tag, message, throwable)
}

fun generateCookie(userId: String, nickname: String): String {
    val timestamp = System.currentTimeMillis() / 1000
    val random = List(6) { ('a'..'z').random() }.joinToString(separator = "")
    return "${userId}_${nickname}_${timestamp}_${random}"
}

fun generateRandomIp(): String =
    List(4) { Random.nextInt(0, 256) }.joinToString(separator = ".")

fun md5Hex(value: String): String {
    val digest = MessageDigest.getInstance("MD5")
    return digest.digest(value.toByteArray())
        .joinToString(separator = "") { "%02x".format(it) }
}

fun inferPrivateMessageIsSelf(currentUserId: String, fromUserId: String): Boolean {
    if (currentUserId.isBlank() || fromUserId.isBlank()) return false
    return md5Hex(currentUserId).equals(fromUserId, ignoreCase = true)
}

fun inferMessageType(content: String): ChatMessageType = when {
    content.startsWith("[") && content.endsWith("]") -> {
        val path = content.removePrefix("[").removeSuffix("]").lowercase()
        when {
            path.endsWith(".png") || path.endsWith(".jpg") || path.endsWith(".jpeg") || path.endsWith(".gif") || path.endsWith(".webp") -> ChatMessageType.IMAGE
            path.endsWith(".mp4") || path.endsWith(".mov") || path.endsWith(".mkv") || path.endsWith(".webm") -> ChatMessageType.VIDEO
            else -> ChatMessageType.FILE
        }
    }

    else -> ChatMessageType.TEXT
}

fun inferFileName(content: String): String =
    content.removePrefix("[").removeSuffix("]").substringAfterLast('/').substringAfterLast('\\')

fun normalizeTextForMatch(value: String): String = value.trim().replace("\r\n", "\n")
