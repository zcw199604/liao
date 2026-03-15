package io.github.a7413498.liao.android

import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.inferFileName
import io.github.a7413498.liao.android.core.common.inferMessageType
import io.github.a7413498.liao.android.core.common.md5Hex
import io.github.a7413498.liao.android.core.network.ChatMessageDto
import io.github.a7413498.liao.android.core.network.ChatMessageUserDto
import io.github.a7413498.liao.android.core.network.ChatUserDto
import io.github.a7413498.liao.android.core.network.HistoryMessageDto
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.network.toFavoriteItemOrNull
import io.github.a7413498.liao.android.core.network.toHistoryMessageList
import io.github.a7413498.liao.android.core.network.toPeer
import io.github.a7413498.liao.android.core.network.toSession
import io.github.a7413498.liao.android.core.network.toTimeline
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class NetworkModelAlignmentTest {
    private val parserJson = Json { ignoreUnknownKeys = true }

    @Test
    fun `message type inference should classify media brackets`() {
        assertEquals(ChatMessageType.IMAGE, inferMessageType("[/upload/demo/test.PNG]"))
        assertEquals(ChatMessageType.VIDEO, inferMessageType("[C:/video/demo.MP4]"))
        assertEquals(ChatMessageType.FILE, inferMessageType("[/tmp/archive.zip]"))
        assertEquals(ChatMessageType.TEXT, inferMessageType("普通文本消息"))
        assertEquals("test.PNG", inferFileName("[/upload/demo/test.PNG]"))
    }

    @Test
    fun `timeline preview should align with media and self wording`() {
        val imageMessage = ChatTimelineMessage(
            id = "1",
            fromUserId = "self",
            fromUserName = "我",
            toUserId = "peer",
            content = "[/upload/a.png]",
            time = "刚刚",
            isSelf = true,
            type = ChatMessageType.IMAGE,
            fileName = "a.png",
        )
        val textMessage = ChatTimelineMessage(
            id = "2",
            fromUserId = "peer",
            fromUserName = "对端",
            toUserId = "self",
            content = "",
            time = "刚刚",
            isSelf = false,
            type = ChatMessageType.TEXT,
        )

        assertEquals("我: [图片] a.png", imageMessage.lastMessagePreview())
        assertEquals("[空消息]", textMessage.lastMessagePreview())
    }

    @Test
    fun `identity and peer mapping should preserve explicit overrides`() {
        val session = IdentityDto(id = "u-1", name = "张三", sex = "男")
            .toSession(cookie = "cookie-fixed", ip = "1.1.1.1", area = "北京")
        val peer = ChatUserDto(
            id = "peer-1",
            name = "原名",
            nickname = "备注名",
            sex = "女",
            ip = "2.2.2.2",
            address = "上海",
            isFavorite = false,
            lastMsg = "你好",
            lastTime = "2026-03-15 09:00:00",
            unreadCount = 3,
        ).toPeer(isFavoriteOverride = true)

        assertEquals("cookie-fixed", session.cookie)
        assertEquals("1.1.1.1", session.ip)
        assertEquals("北京", session.area)
        assertEquals("备注名", peer.name)
        assertTrue(peer.isFavorite)
        assertEquals(3, peer.unreadCount)
    }

    @Test
    fun `chat dto timeline should infer self and media metadata`() {
        val currentUserId = "123456"
        val timeline = ChatMessageDto(
            code = 7,
            fromUser = ChatMessageUserDto(id = md5Hex(currentUserId), nickname = "自己"),
            toUser = ChatMessageUserDto(id = "peer-1"),
            content = "[/upload/demo/video.mp4]",
            time = "2026-03-15 09:10:00",
            tid = "tid-1",
            type = null,
        ).toTimeline(currentUserId = currentUserId)

        assertTrue(timeline.isSelf)
        assertEquals(ChatMessageType.VIDEO, timeline.type)
        assertEquals("/upload/demo/video.mp4", timeline.mediaUrl)
        assertEquals("video.mp4", timeline.fileName)
        assertEquals("tid-1", timeline.id)
    }

    @Test
    fun `history dto timeline should distinguish self from peer`() {
        val selfTimeline = HistoryMessageDto(
            tidUpper = "T-1",
            id = "self-1",
            content = "[/upload/demo/pic.jpg]",
            time = "2026-03-15 09:20:00",
            type = null,
        ).toTimeline(currentUserId = "self-1", peerId = "peer-1", currentUserName = "自己")
        val peerTimeline = HistoryMessageDto(
            tidLower = "t-2",
            id = "peer-1",
            nickname = "对端昵称",
            content = "你好",
            time = "",
            type = null,
        ).toTimeline(currentUserId = "self-1", peerId = "peer-1", currentUserName = "自己")

        assertTrue(selfTimeline.isSelf)
        assertEquals("自己", selfTimeline.fromUserName)
        assertEquals(ChatMessageType.IMAGE, selfTimeline.type)
        assertEquals("peer-1", selfTimeline.toUserId)

        assertFalse(peerTimeline.isSelf)
        assertEquals("对端昵称", peerTimeline.fromUserName)
        assertEquals("刚刚", peerTimeline.time)
        assertEquals("t-2", peerTimeline.id)
    }

    @Test
    fun `history parser should accept array and object envelopes`() {
        val arrayPayload = parserJson.parseToJsonElement(
            """[{"Tid":"t-1","id":"peer-1","content":"hi","time":"2026-03-15 09:30:00"}]""",
        )
        val objectPayload = parserJson.parseToJsonElement(
            """{"code":0,"contents_list":[{"tid":"t-2","id":"peer-2","content":"hello","time":"2026-03-15 09:31:00"}]}""",
        )

        val arrayItems = arrayPayload.toHistoryMessageList()
        val objectItems = objectPayload.toHistoryMessageList()

        assertEquals(1, arrayItems.size)
        assertEquals("t-1", arrayItems.first().tidUpper)
        assertEquals(1, objectItems.size)
        assertEquals("t-2", objectItems.first().tidLower)
    }

    @Test
    fun `favorite parser should reject invalid id and keep valid payload`() {
        val validPayload = parserJson.parseToJsonElement(
            """{"id":"12","identityId":"self-1","targetUserId":"peer-1","targetUserName":"对端","createTime":"2026-03-15 09:40:00"}""",
        )
        val invalidPayload = parserJson.parseToJsonElement(
            """{"id":"not-number","identityId":"self-1"}""",
        )

        val validItem = validPayload.toFavoriteItemOrNull()
        val invalidItem = invalidPayload.toFavoriteItemOrNull()

        assertEquals(12, validItem?.id)
        assertEquals("peer-1", validItem?.targetUserId)
        assertNull(invalidItem)
    }
}
