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
                onOpenCrossIdentity = {},
                onOpenArchiveSearch = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { _, _ -> },
                onMarkPeerRead = {},
                onClearPeerUnread = {},
                onToggleGlobalFavorite = { _, _ -> },
                onRequestDeletePeer = {},
                onCheckOnlineStatus = {},
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
                onOpenCrossIdentity = {},
                onOpenArchiveSearch = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { _, _ -> },
                onMarkPeerRead = {},
                onClearPeerUnread = {},
                onToggleGlobalFavorite = { _, _ -> },
                onRequestDeletePeer = {},
                onCheckOnlineStatus = {},
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
                onOpenCrossIdentity = {},
                onOpenArchiveSearch = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { peerId, peerName ->
                    openedPeerId = peerId
                    openedPeerName = peerName
                },
                onMarkPeerRead = { markedReadId = it },
                onClearPeerUnread = {},
                onToggleGlobalFavorite = { _, _ -> },
                onRequestDeletePeer = {},
                onCheckOnlineStatus = {},
            )
        }

        composeRule.onNodeWithTag(ChatListTestTags.item(peer.id)).performClick()

        composeRule.runOnIdle {
            assertEquals(peer.id, markedReadId)
            assertEquals(peer.id, openedPeerId)
            assertEquals(peer.name, openedPeerName)
        }
    }

    @Test
    fun peer_item_actions_should_clear_unread_and_request_delete() {
        val peer = ChatPeer(
            id = "peer-actions",
            name = "动作会话",
            sex = "女",
            ip = "127.0.0.1",
            address = "上海",
            lastMessage = "你好",
            lastTime = "10:00",
            unreadCount = 3,
        )
        var clearedPeerId: String? = null
        var toggledFavorite: Pair<String, Boolean>? = null
        var requestedDeleteId: String? = null
        var checkedOnlineId: String? = null

        composeRule.setLiaoTestContent {
            ChatListScreenContent(
                state = ChatListUiState(
                    loading = false,
                    items = listOf(peer),
                    globalFavoriteTargetIds = setOf(peer.id),
                ),
                onSwitchTab = {},
                onRefresh = {},
                onOpenCrossIdentity = {},
                onOpenArchiveSearch = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { _, _ -> },
                onMarkPeerRead = {},
                onClearPeerUnread = { clearedPeerId = it.id },
                onToggleGlobalFavorite = { target, isGlobalFavorite -> toggledFavorite = target.id to isGlobalFavorite },
                onRequestDeletePeer = { requestedDeleteId = it.id },
                onCheckOnlineStatus = { checkedOnlineId = it.id },
            )
        }

        composeRule.onNodeWithText("取消全局收藏").fetchSemanticsNode()
        composeRule.onNodeWithTag(ChatListTestTags.clearUnreadButton(peer.id)).performClick()
        composeRule.onNodeWithTag(ChatListTestTags.globalFavoriteButton(peer.id)).performClick()
        composeRule.onNodeWithTag(ChatListTestTags.checkOnlineButton(peer.id)).performClick()
        composeRule.onNodeWithTag(ChatListTestTags.deleteButton(peer.id)).performClick()

        composeRule.runOnIdle {
            assertEquals(peer.id, clearedPeerId)
            assertEquals(peer.id to true, toggledFavorite)
            assertEquals(peer.id, checkedOnlineId)
            assertEquals(peer.id, requestedDeleteId)
        }
    }

    @Test
    fun online_status_dialog_should_render_status_and_last_time() {
        composeRule.setLiaoTestContent {
            ChatListScreenContent(
                state = ChatListUiState(
                    loading = false,
                    onlineStatusVisible = true,
                    onlineStatusPeerName = "在线用户",
                    onlineStatusOnline = true,
                    onlineStatusLastTime = "2026-07-07 13:00",
                ),
                onSwitchTab = {},
                onRefresh = {},
                onOpenCrossIdentity = {},
                onOpenArchiveSearch = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { _, _ -> },
                onMarkPeerRead = {},
                onClearPeerUnread = {},
                onToggleGlobalFavorite = { _, _ -> },
                onRequestDeletePeer = {},
                onCheckOnlineStatus = {},
                onDismissOnlineStatus = {},
            )
        }

        composeRule.onNodeWithText("在线状态").fetchSemanticsNode()
        composeRule.onNodeWithText("在线用户").fetchSemanticsNode()
        composeRule.onNodeWithText("在线").fetchSemanticsNode()
        composeRule.onNodeWithText("2026-07-07 13:00").fetchSemanticsNode()
    }

    @Test
    fun batch_selection_should_toggle_peer_and_confirm_delete() {
        val peer = ChatPeer(
            id = "peer-batch",
            name = "批量会话",
            sex = "女",
            ip = "127.0.0.1",
            address = "上海",
            lastMessage = "你好",
            lastTime = "10:00",
            unreadCount = 0,
        )
        var selectionEntered = false
        var toggledPeerId: String? = null
        var deleteRequested = false
        var deleteConfirmed = false
        var state by mutableStateOf(ChatListUiState(loading = false, items = listOf(peer)))

        composeRule.setLiaoTestContent {
            ChatListScreenContent(
                state = state,
                onSwitchTab = {},
                onRefresh = {},
                onOpenCrossIdentity = {},
                onOpenArchiveSearch = {},
                onOpenGlobalFavorites = {},
                onOpenSettings = {},
                onOpenChat = { _, _ -> },
                onMarkPeerRead = {},
                onClearPeerUnread = {},
                onToggleGlobalFavorite = { _, _ -> },
                onRequestDeletePeer = {},
                onCheckOnlineStatus = {},
                onEnterSelectionMode = {
                    selectionEntered = true
                    state = state.copy(selectionMode = true)
                },
                onTogglePeerSelection = {
                    toggledPeerId = it
                    state = state.copy(selectedPeerIds = setOf(it))
                },
                onRequestBatchDelete = {
                    deleteRequested = true
                    state = state.copy(batchDeleteConfirmVisible = true)
                },
                onConfirmBatchDelete = { deleteConfirmed = true },
            )
        }

        composeRule.onNodeWithTag(ChatListTestTags.BATCH_MANAGE_BUTTON).performClick()
        composeRule.onNodeWithTag(ChatListTestTags.item(peer.id)).performClick()
        composeRule.onNodeWithTag(ChatListTestTags.BATCH_DELETE_BUTTON).performClick()
        composeRule.onNodeWithTag(ChatListTestTags.BATCH_DELETE_DIALOG_CONFIRM).performClick()

        composeRule.runOnIdle {
            assertEquals(true, selectionEntered)
            assertEquals(peer.id, toggledPeerId)
            assertEquals(true, deleteRequested)
            assertEquals(true, deleteConfirmed)
        }
    }
}
