package io.github.a7413498.liao.android

import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.inferMessageType
import io.github.a7413498.liao.android.core.common.inferPrivateMessageIsSelf
import io.github.a7413498.liao.android.core.common.md5Hex
import io.github.a7413498.liao.android.core.network.ChatMessageDto
import io.github.a7413498.liao.android.core.network.ChatMessageUserDto
import io.github.a7413498.liao.android.core.network.ChatUserDto
import io.github.a7413498.liao.android.core.network.HistoryMessageDto
import io.github.a7413498.liao.android.core.network.stringOrNull
import io.github.a7413498.liao.android.core.network.toFavoriteItemOrNull
import io.github.a7413498.liao.android.core.network.toHistoryMessageList
import io.github.a7413498.liao.android.core.network.toPeer
import io.github.a7413498.liao.android.core.network.toTimeline
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class NetworkModelBranchTest {
    private val parserJson = Json { ignoreUnknownKeys = true }

    @Test
    fun `message type inference should cover remaining suffix branches and private message matcher`() {
        assertEquals(ChatMessageType.IMAGE, inferMessageType("[/tmp/pic.jpeg]"))
        assertEquals(ChatMessageType.IMAGE, inferMessageType("[/tmp/pic.GIF]"))
        assertEquals(ChatMessageType.IMAGE, inferMessageType("[/tmp/pic.webp]"))
        assertEquals(ChatMessageType.VIDEO, inferMessageType("[/tmp/a.mov]"))
        assertEquals(ChatMessageType.VIDEO, inferMessageType("[/tmp/b.mkv]"))
        assertEquals(ChatMessageType.VIDEO, inferMessageType("[/tmp/c.webm]"))
        assertEquals(ChatMessageType.FILE, inferMessageType("[/tmp/d.pdf]"))

        assertFalse(inferPrivateMessageIsSelf(currentUserId = "", fromUserId = "abc"))
        assertFalse(inferPrivateMessageIsSelf(currentUserId = "user-1", fromUserId = ""))
        assertFalse(inferPrivateMessageIsSelf(currentUserId = "user-1", fromUserId = "plain-id"))
        assertTrue(inferPrivateMessageIsSelf(currentUserId = "user-1", fromUserId = md5Hex("user-1")))
    }

    @Test
    fun `peer mapping should fallback to source values when override absent`() {
        val peer = ChatUserDto(
            id = "peer-1",
            nickname = null,
            name = null,
            sex = null,
            ip = null,
            address = null,
            isFavorite = true,
            lastMsg = null,
            lastTime = null,
            unreadCount = null,
        ).toPeer()

        assertEquals("", peer.name)
        assertEquals("", peer.sex)
        assertEquals("", peer.ip)
        assertEquals("", peer.address)
        assertTrue(peer.isFavorite)
        assertEquals(0, peer.unreadCount)
    }

    @Test
    fun `chat dto timeline should honor explicit type overrides and fallback generated id`() {
        val explicitImage = ChatMessageDto(
            fromUser = ChatMessageUserDto(id = "peer-1", name = "原名"),
            content = "plain-image-path",
            time = "2026-03-16 18:00:00",
            tid = "",
            type = "image",
        ).toTimeline(currentUserId = "self")
        val explicitText = ChatMessageDto(
            fromUser = ChatMessageUserDto(id = "peer-2", nickname = "文本用户"),
            content = "plain-text-message",
            time = "2026-03-16 18:01:00",
            tid = "",
            type = "unknown",
        ).toTimeline(currentUserId = "self")

        assertEquals(ChatMessageType.IMAGE, explicitImage.type)
        assertEquals("plain-image-path", explicitImage.mediaUrl)
        assertEquals("plain-image-path", explicitImage.fileName)
        assertEquals("原名", explicitImage.fromUserName)
        assertTrue(explicitImage.id.startsWith("2026-03-16 18:00:00_peer-1_"))
        assertFalse(explicitImage.isSelf)

        assertEquals(ChatMessageType.TEXT, explicitText.type)
        assertEquals("", explicitText.mediaUrl)
        assertEquals("", explicitText.fileName)
    }

    @Test
    fun `history dto timeline should cover display fallback and explicit file type`() {
        val nameFallback = HistoryMessageDto(
            id = "peer-1",
            name = "来自 name",
            nickname = null,
            content = "[/upload/archive.zip]",
            time = "2026-03-16 18:10:00",
            type = "file",
        ).toTimeline(currentUserId = "self", peerId = "peer-1", currentUserName = "自己")
        val peerIdFallback = HistoryMessageDto(
            id = "",
            nickname = null,
            name = null,
            content = "普通文本",
            time = "",
            type = null,
        ).toTimeline(currentUserId = "self", peerId = "peer-long-id", currentUserName = "自己")

        assertFalse(nameFallback.isSelf)
        assertEquals("来自 name", nameFallback.fromUserName)
        assertEquals(ChatMessageType.FILE, nameFallback.type)
        assertEquals("archive.zip", nameFallback.fileName)
        assertTrue(nameFallback.id.startsWith("2026-03-16 18:10:00_peer-1_"))

        assertFalse(peerIdFallback.isSelf)
        assertEquals("peer-lon", peerIdFallback.fromUserName)
        assertEquals("刚刚", peerIdFallback.time)
        assertEquals("self", peerIdFallback.toUserId)
    }

    @Test
    fun `history parser should ignore invalid shapes and retain valid items only`() {
        val mixedArray = parserJson.parseToJsonElement(
            """[{"Tid":"t-1","id":"peer-1","content":"hi","time":"2026-03-16"},123,{"Tid":{"bad":1}}]""",
        )
        val invalidObject = parserJson.parseToJsonElement("""{"contents_list":1}""")
        val primitive = parserJson.parseToJsonElement("\"oops\"")

        val mixedItems = mixedArray.toHistoryMessageList()

        assertEquals(1, mixedItems.size)
        assertEquals("t-1", mixedItems.first().tidUpper)
        assertTrue(invalidObject.toHistoryMessageList().isEmpty())
        assertTrue(primitive.toHistoryMessageList().isEmpty())
    }

    @Test
    fun `favorite parser and string helper should cover numeric and blank cases`() {
        val objectPayload = parserJson.parseToJsonElement(
            """{"id":15,"identityId":"   ","targetUserId":"peer-1","targetUserName":"对端","createTime":"2026-03-16 18:20:00","num":123,"blank":"   ","nullValue":null}""",
        ).jsonObject

        val item = objectPayload.toFavoriteItemOrNull()

        assertEquals(15, item?.id)
        assertEquals("", item?.identityId)
        assertEquals("peer-1", item?.targetUserId)
        assertEquals("123", objectPayload.stringOrNull("num"))
        assertNull(objectPayload.stringOrNull("blank"))
        assertNull(parserJson.parseToJsonElement("123").toFavoriteItemOrNull())
    }
}
