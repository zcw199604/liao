package io.github.a7413498.liao.android.feature.chatroom

import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.OutgoingMessageStatus
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ChatRoomFeatureHelpersTest {
    @Test
    fun `min numeric tid should ignore non numeric ids and keep smallest numeric string`() {
        val result = listOf(
            timelineMessage(id = "abc"),
            timelineMessage(id = " 12 "),
            timelineMessage(id = "7"),
            timelineMessage(id = "-5"),
        ).minNumericTid()

        assertEquals("-5", result)
    }

    @Test
    fun `merge timeline messages should deduplicate by client id with current item winning`() {
        val current = listOf(
            timelineMessage(id = "1", clientId = "c-1", content = "current"),
            timelineMessage(id = "2", content = "existing-2"),
        )
        val incoming = listOf(
            timelineMessage(id = "remote-1", clientId = "c-1", content = "incoming"),
            timelineMessage(id = "3", content = "incoming-3"),
        )

        val result = mergeTimelineMessages(current = current, incoming = incoming)

        assertEquals(listOf("current", "incoming-3", "existing-2"), result.map { it.content })
    }

    @Test
    fun `append timeline message should replace matched entry or append new one`() {
        val current = listOf(
            timelineMessage(id = "1", clientId = "c-1", content = "pending"),
            timelineMessage(id = "2", content = "stable"),
        )

        val replaced = appendTimelineMessage(
            current = current,
            incoming = timelineMessage(id = "remote", clientId = "c-1", content = "sent"),
        )
        val appended = appendTimelineMessage(
            current = current,
            incoming = timelineMessage(id = "3", content = "brand-new"),
        )

        assertEquals(listOf("sent", "stable"), replaced.map { it.content })
        assertEquals(listOf("pending", "stable", "brand-new"), appended.map { it.content })
    }

    @Test
    fun `merge echo message should preserve optimistic client id for exact tid match`() {
        val current = listOf(
            timelineMessage(
                id = "tid-1",
                clientId = "client-1",
                content = "hello",
                sendStatus = OutgoingMessageStatus.SENDING,
            )
        )

        val result = mergeEchoMessage(
            current = current,
            incoming = timelineMessage(
                id = "tid-1",
                content = "hello",
                sendStatus = OutgoingMessageStatus.FAILED,
                sendError = "old",
            ),
        )

        val merged = result.single()
        assertEquals("client-1", merged.clientId)
        assertEquals(OutgoingMessageStatus.SENT, merged.sendStatus)
        assertEquals(null, merged.sendError)
    }

    @Test
    fun `merge echo message should match optimistic self message by normalized content and append fallback`() {
        val pending = timelineMessage(
            id = "temp-1",
            clientId = "client-1",
            content = "hello\r\nworld",
            toUserId = "peer-1",
            isSelf = true,
            type = ChatMessageType.TEXT,
            sendStatus = OutgoingMessageStatus.SENDING,
        )

        val optimistic = mergeEchoMessage(
            current = listOf(pending),
            incoming = timelineMessage(
                id = "tid-remote",
                content = " hello\nworld ",
                toUserId = "peer-1",
                isSelf = true,
                type = ChatMessageType.TEXT,
            ),
        )
        val appended = mergeEchoMessage(
            current = listOf(timelineMessage(id = "seed", content = "other")),
            incoming = timelineMessage(id = "tid-2", content = "new", sendStatus = OutgoingMessageStatus.FAILED, sendError = "x"),
        )

        assertEquals("client-1", optimistic.single().clientId)
        assertEquals(OutgoingMessageStatus.SENT, optimistic.single().sendStatus)
        assertEquals(listOf("seed", "tid-2"), appended.map { it.id })
        assertEquals(OutgoingMessageStatus.SENT, appended.last().sendStatus)
        assertEquals(null, appended.last().sendError)
    }

    @Test
    fun `display helpers should map media labels content and status text`() {
        assertEquals("[图片] cat.png", timelineMessage(type = ChatMessageType.IMAGE, fileName = "cat.png").displayContent())
        assertEquals("[视频]", timelineMessage(type = ChatMessageType.VIDEO).displayContent())
        assertEquals("[文件] report.pdf", timelineMessage(type = ChatMessageType.FILE, fileName = "report.pdf").displayContent())
        assertEquals("正文", timelineMessage(type = ChatMessageType.TEXT, content = "正文").displayContent())

        assertEquals("发送中", OutgoingMessageStatus.SENDING.displayText())
        assertEquals("已发送", OutgoingMessageStatus.SENT.displayText())
        assertEquals("发送失败", OutgoingMessageStatus.FAILED.displayText())
    }

    @Test
    fun `websocket display text should cover standard states and forceout countdown floor`() {
        assertEquals("Idle", WebSocketState.Idle.toDisplayText())
        assertEquals("Connecting", WebSocketState.Connecting.toDisplayText())
        assertEquals("Connected", WebSocketState.Connected.toDisplayText())
        assertEquals("Reconnecting", WebSocketState.Reconnecting.toDisplayText())
        assertEquals("Closed", WebSocketState.Closed.toDisplayText())
        assertEquals("Forceout(0s)", WebSocketState.Forceout(System.currentTimeMillis() - 1).toDisplayText())
    }

    @Test
    fun `media label helpers should expose mime type and display label`() {
        assertEquals("image/*", ChatMessageType.IMAGE.openMimeType())
        assertEquals("video/*", ChatMessageType.VIDEO.openMimeType())
        assertEquals("*/*", ChatMessageType.FILE.openMimeType())
        assertEquals("text/plain", ChatMessageType.TEXT.openMimeType())

        assertEquals("图片 · cat.png", ChatMessageType.IMAGE.displayLabel("cat.png"))
        assertEquals("视频", ChatMessageType.VIDEO.displayLabel(""))
        assertEquals("文件 · report.pdf", ChatMessageType.FILE.displayLabel("report.pdf"))
        assertEquals("plain.txt", ChatMessageType.TEXT.displayLabel("plain.txt"))
    }

    @Test
    fun `forceout display text should round remaining time up to seconds`() {
        val text = WebSocketState.Forceout(System.currentTimeMillis() + 50).toDisplayText()

        assertTrue(text.startsWith("Forceout("))
        assertTrue(text.endsWith("s)"))
    }

    private fun timelineMessage(
        id: String = "1",
        clientId: String = "",
        content: String = "content",
        toUserId: String = "peer",
        isSelf: Boolean = false,
        type: ChatMessageType = ChatMessageType.TEXT,
        fileName: String = "",
        sendStatus: OutgoingMessageStatus = OutgoingMessageStatus.SENT,
        sendError: String? = null,
    ): ChatTimelineMessage = ChatTimelineMessage(
        id = id,
        fromUserId = "self",
        fromUserName = "Alice",
        toUserId = toUserId,
        content = content,
        time = "10:00:00",
        isSelf = isSelf,
        type = type,
        fileName = fileName,
        clientId = clientId,
        sendStatus = sendStatus,
        sendError = sendError,
    )
}
