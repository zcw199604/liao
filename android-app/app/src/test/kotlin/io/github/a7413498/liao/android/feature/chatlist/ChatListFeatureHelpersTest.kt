package io.github.a7413498.liao.android.feature.chatlist

import org.junit.Assert.assertEquals
import org.junit.Test

class ChatListFeatureHelpersTest {
    @Test
    fun `error titles should follow tab type`() {
        assertEquals("历史会话加载失败", chatListErrorTitle(ConversationTab.HISTORY))
        assertEquals("收藏会话加载失败", chatListErrorTitle(ConversationTab.FAVORITE))
    }

    @Test
    fun `empty titles should follow tab type`() {
        assertEquals("暂无历史会话", chatListEmptyTitle(ConversationTab.HISTORY))
        assertEquals("暂无收藏会话", chatListEmptyTitle(ConversationTab.FAVORITE))
    }

    @Test
    fun `empty descriptions should follow tab type`() {
        assertEquals("开始聊天后，这里会显示最近联系的人。", chatListEmptyDescription(ConversationTab.HISTORY))
        assertEquals(
            "你还没有收藏任何会话，可以通过全局收藏查看不同身份下的收藏对象。",
            chatListEmptyDescription(ConversationTab.FAVORITE),
        )
    }
}
