/*
 * 纯单元测试用于校验 Android 侧协议辅助函数、WS 协议目录与 forceout 常量。
 * 当前环境未必执行这些测试，但它们为后续本地构建提供了最小回归基线。
 */
package io.github.a7413498.liao.android

import io.github.a7413498.liao.android.core.common.inferPrivateMessageIsSelf
import io.github.a7413498.liao.android.core.common.md5Hex
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.LiaoWsKnownAct
import io.github.a7413498.liao.android.core.websocket.LiaoWsKnownCode
import io.github.a7413498.liao.android.core.websocket.LiaoWsProtocolCatalog
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ProtocolAlignmentTest {
    @Test
    fun `MD5 self detection should align with web client rule`() {
        val currentUserId = "123456"
        val fromUserId = md5Hex(currentUserId)
        assertTrue(inferPrivateMessageIsSelf(currentUserId = currentUserId, fromUserId = fromUserId))
        assertFalse(inferPrivateMessageIsSelf(currentUserId = currentUserId, fromUserId = "other"))
    }

    @Test
    fun `forceout duration should remain five minutes`() {
        assertEquals(5 * 60 * 1000L, LiaoWebSocketClient.FORCEOUT_DURATION_MILLIS)
    }

    @Test
    fun `protocol catalog should resolve known code and act`() {
        assertEquals(LiaoWsKnownCode.Reject, LiaoWsProtocolCatalog.resolveKnownCode(-4))
        assertEquals(LiaoWsKnownCode.PrivateMessage, LiaoWsProtocolCatalog.resolveKnownCode(7))
        assertEquals(LiaoWsKnownAct.PrivateMessage, LiaoWsProtocolCatalog.resolveKnownAct("touser_100_张三"))
        assertEquals(LiaoWsKnownAct.WarningReport, LiaoWsProtocolCatalog.resolveKnownAct("warningreport"))
    }

    @Test
    fun `protocol parser should classify reject and private message`() {
        val rejectEnvelope = LiaoWsProtocolCatalog.parseEnvelope(
            raw = """{"code":-4,"forceout":true,"content":"由于重复登录，您的连接被暂时禁止"}""",
        )
        assertNotNull(rejectEnvelope)
        assertEquals(LiaoWsKnownCode.Reject, rejectEnvelope?.knownCode)
        assertTrue(rejectEnvelope?.isForceoutMessage() == true)

        val chatEnvelope = LiaoWsProtocolCatalog.parseEnvelope(
            raw = """{"code":7,"fromuser":{"id":"${md5Hex("123456")}","nickname":"对端","content":"你好","time":"2026-03-13 05:50:00","Tid":"t-1"},"touser":{"id":"peer-1"}}""",
        )
        assertNotNull(chatEnvelope)
        val timeline = chatEnvelope?.toTimelineMessage(currentUserId = "123456")
        assertNotNull(timeline)
        assertEquals("t-1", timeline?.id)
        assertEquals("你好", timeline?.content)
        assertTrue(timeline?.isSelf == true)
    }
}
