package io.github.a7413498.liao.android.feature.chatlist

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.a7413498.liao.android.app.testing.ChatListTestTags
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.test.setLiaoTestContent
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class ChatListScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun switching_tabs_should_update_empty_state_copy() {
        var state by mutableStateOf(ChatListUiState(tab = ConversationTab.HISTORY, loading = false))

        composeRule.setLiaoTestContent {
            ChatListScreenContent(
                state = state,
                onSwitchTab = { state = state.copy(tab = it) },
                onRefresh = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { _, _ -> },
                onMarkPeerRead = {},
            )
        }

        composeRule.onNodeWithText("暂无历史会话").fetchSemanticsNode()
        composeRule.onNodeWithTag(ChatListTestTags.FAVORITE_TAB).performClick()
        composeRule.onNodeWithText("暂无收藏会话").fetchSemanticsNode()
    }

    @Test
    fun error_state_should_render_state_card() {
        composeRule.setLiaoTestContent {
            ChatListScreenContent(
                state = ChatListUiState(
                    tab = ConversationTab.FAVORITE,
                    loading = false,
                    errorMessage = "接口暂时不可用",
                ),
                onSwitchTab = {},
                onRefresh = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { _, _ -> },
                onMarkPeerRead = {},
            )
        }

        composeRule.onNodeWithTag(ChatListTestTags.STATE_CARD).fetchSemanticsNode()
        composeRule.onNodeWithText("收藏会话加载失败").fetchSemanticsNode()
    }

    @Test
    fun clicking_peer_item_should_mark_read_and_open_chat() {
        val peer = ChatPeer(
            id = "peer-1",
            name = "测试会话",
            sex = "女",
            ip = "127.0.0.1",
            address = "上海",
            lastMessage = "你好",
            lastTime = "10:00",
            unreadCount = 2,
        )
        var markedReadId: String? = null
        var openedPeerId: String? = null
        var openedPeerName: String? = null

        composeRule.setLiaoTestContent {
            ChatListScreenContent(
                state = ChatListUiState(loading = false, items = listOf(peer)),
                onSwitchTab = {},
                onRefresh = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { peerId, peerName ->
                    openedPeerId = peerId
                    openedPeerName = peerName
                },
                onMarkPeerRead = { markedReadId = it },
            )
        }

        composeRule.onNodeWithTag(ChatListTestTags.item(peer.id)).performClick()

        composeRule.runOnIdle {
            assertEquals(peer.id, markedReadId)
            assertEquals(peer.id, openedPeerId)
            assertEquals(peer.name, openedPeerName)
        }
    }
}
