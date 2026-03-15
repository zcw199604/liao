package io.github.a7413498.liao.android.core.websocket

import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.md5Hex
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class WebSocketProtocolCatalogTest {
    @Test
    fun `known act resolver should support trim and private message prefix`() {
        assertEquals(LiaoWsKnownAct.Sign, LiaoWsProtocolCatalog.resolveKnownAct(" sign "))
        assertEquals(LiaoWsKnownAct.PrivateMessage, LiaoWsProtocolCatalog.resolveKnownAct("touser_100_张三"))
        assertNull(LiaoWsProtocolCatalog.resolveKnownAct("inputStatusOn_100_张三"))
        assertEquals("touser_100_张三", LiaoWsProtocolCatalog.Acts.privateMessage("100", "张三"))
    }

    @Test
    fun `protocol user display name should fallback to name then anonymous`() {
        val nicknameUser = LiaoWsProtocolUser(id = "1", name = "原名", nickname = "备注")
        val nameOnlyUser = LiaoWsProtocolUser(id = "2", name = "原名", nickname = "")
        val anonymousUser = LiaoWsProtocolUser(id = "3", name = "", nickname = "")

        assertEquals("备注", nicknameUser.displayName)
        assertEquals("原名", nameOnlyUser.displayName)
        assertEquals("匿名用户", anonymousUser.displayName)
    }

    @Test
    fun `envelope parser should prefer nested fromuser fields`() {
        val envelope = LiaoWsProtocolCatalog.parseEnvelope(
            raw =
                """{"code":7,"act":"touser_peer_对端","content":"root-content","time":"root-time","Tid":"root-tid","fromuser":{"id":"sender-1","nickname":"发送者","content":"nested-content","Time":"nested-time","Tid":"nested-tid"},"touser":{"id":"peer-1"}}""",
        )

        assertNotNull(envelope)
        assertEquals(LiaoWsKnownCode.PrivateMessage, envelope?.knownCode)
        assertEquals("nested-content", envelope?.content)
        assertEquals("nested-time", envelope?.time)
        assertEquals("root-tid", envelope?.tid)
        assertEquals("peer-1", envelope?.toUser?.id)
    }

    @Test
    fun `envelope parser should return null for invalid json`() {
        assertNull(LiaoWsProtocolCatalog.parseEnvelope("not-json"))
        assertNull(LiaoWsProtocolCatalog.parseRoot("not-json"))
    }

    @Test
    fun `forceout detection should require both flag and reject or forceout code`() {
        val reject = LiaoWsProtocolCatalog.parseEnvelope("""{"code":-4,"forceout":true,"content":"x"}""")
        val forceout = LiaoWsProtocolCatalog.parseEnvelope("""{"code":-3,"forceout":true,"content":"x"}""")
        val notice = LiaoWsProtocolCatalog.parseEnvelope("""{"code":12,"forceout":true,"content":"x"}""")
        val missingFlag = LiaoWsProtocolCatalog.parseEnvelope("""{"code":-4,"forceout":false,"content":"x"}""")

        assertTrue(reject?.isForceoutMessage() == true)
        assertTrue(forceout?.isForceoutMessage() == true)
        assertFalse(notice?.isForceoutMessage() == true)
        assertFalse(missingFlag?.isForceoutMessage() == true)
    }

    @Test
    fun `timeline conversion should infer file metadata and fallback tid time`() {
        val currentUserId = "123456"
        val envelope = LiaoWsProtocolCatalog.parseEnvelope(
            raw =
                """{"code":7,"fromuser":{"id":"${md5Hex(currentUserId)}","name":"自己","content":"[/upload/demo/report.pdf]"},"touser":{"id":"peer-1"}}""",
        )

        val timeline = envelope?.toTimelineMessage(currentUserId = currentUserId)

        assertNotNull(timeline)
        assertTrue(timeline?.isSelf == true)
        assertEquals(ChatMessageType.FILE, timeline?.type)
        assertEquals("/upload/demo/report.pdf", timeline?.mediaUrl)
        assertEquals("report.pdf", timeline?.fileName)
        assertEquals("刚刚", timeline?.time)
        assertTrue(timeline?.id?.contains(md5Hex(currentUserId)) == true)
    }

    @Test
    fun `timeline conversion should return null for non private or blank sender`() {
        val connectNotice = LiaoWsProtocolCatalog.parseEnvelope("""{"code":12,"content":"连接成功"}""")
        val blankSender = LiaoWsProtocolCatalog.parseEnvelope(
            raw = """{"code":7,"fromuser":{"id":"","content":"你好"}}""",
        )
        val blankContent = LiaoWsProtocolCatalog.parseEnvelope(
            raw = """{"code":7,"fromuser":{"id":"sender-1","content":""}}""",
        )

        assertNull(connectNotice?.toTimelineMessage(currentUserId = "123"))
        assertNull(blankSender?.toTimelineMessage(currentUserId = "123"))
        assertNull(blankContent?.toTimelineMessage(currentUserId = "123"))
    }

    @Test
    fun `match candidate and reconnect delay should keep web aligned defaults`() {
        val peer = MatchCandidate(
            id = "peer-1",
            name = "对端",
            sex = "女",
            age = "20",
            address = "上海",
        ).toPeer()

        assertEquals("匹配成功", peer.lastMessage)
        assertEquals("刚刚", peer.lastTime)
        assertEquals(0, peer.unreadCount)
        assertEquals(3_000L, LiaoWebSocketClient.computeReconnectDelayMillis(0))
        assertEquals(6_000L, LiaoWebSocketClient.computeReconnectDelayMillis(2))
        assertEquals(15_000L, LiaoWebSocketClient.computeReconnectDelayMillis(9))
    }
}
