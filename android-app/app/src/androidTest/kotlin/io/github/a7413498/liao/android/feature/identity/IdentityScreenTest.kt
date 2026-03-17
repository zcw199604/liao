package io.github.a7413498.liao.android.feature.identity

import androidx.compose.ui.test.assertTextContains
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.a7413498.liao.android.app.testing.IdentityTestTags
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.test.setLiaoTestContent
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class IdentityScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun empty_state_should_show_and_secondary_action_should_quick_create_in_create_mode() {
        var quickCreateClicks = 0
        var cancelClicks = 0

        composeRule.setLiaoTestContent {
            IdentityScreenContent(
                state = IdentityUiState(identities = emptyList(), loading = false),
                onNameChange = {},
                onSexChange = {},
                onSubmitIdentity = {},
                onQuickCreate = { quickCreateClicks += 1 },
                onCancelEditing = { cancelClicks += 1 },
                onSelectIdentity = {},
                onStartEditing = {},
                onConfirmDeleteIdentity = {},
                onDismissDeleteDialog = {},
                onDeleteConfirmed = {},
            )
        }

        composeRule.onNodeWithTag(IdentityTestTags.EMPTY_STATE).fetchSemanticsNode()
        composeRule.onNodeWithTag(IdentityTestTags.SECONDARY_ACTION_BUTTON).performClick()

        composeRule.runOnIdle {
            assertEquals(1, quickCreateClicks)
            assertEquals(0, cancelClicks)
        }
    }

    @Test
    fun edit_mode_should_switch_labels_and_allow_cancel() {
        val identity = sampleIdentity(id = "identity-1")
        var cancelClicks = 0

        composeRule.setLiaoTestContent {
            IdentityScreenContent(
                state = IdentityUiState(
                    identities = listOf(identity),
                    loading = false,
                    editingIdentityId = identity.id,
                ),
                onNameChange = {},
                onSexChange = {},
                onSubmitIdentity = {},
                onQuickCreate = {},
                onCancelEditing = { cancelClicks += 1 },
                onSelectIdentity = {},
                onStartEditing = {},
                onConfirmDeleteIdentity = {},
                onDismissDeleteDialog = {},
                onDeleteConfirmed = {},
            )
        }

        composeRule.onNodeWithTag(IdentityTestTags.DESCRIPTION).assertTextContains("编辑模式")
        composeRule.onNodeWithText("保存编辑").fetchSemanticsNode()
        composeRule.onNodeWithText("取消编辑").fetchSemanticsNode()
        composeRule.onNodeWithTag(IdentityTestTags.SECONDARY_ACTION_BUTTON).performClick()

        composeRule.runOnIdle {
            assertEquals(1, cancelClicks)
        }
    }

    @Test
    fun select_button_should_forward_identity() {
        val identity = sampleIdentity(id = "identity-select")
        var selectedId: String? = null

        composeRule.setLiaoTestContent {
            IdentityScreenContent(
                state = IdentityUiState(identities = listOf(identity), loading = false),
                onNameChange = {},
                onSexChange = {},
                onSubmitIdentity = {},
                onQuickCreate = {},
                onCancelEditing = {},
                onSelectIdentity = { selectedId = it.id },
                onStartEditing = {},
                onConfirmDeleteIdentity = {},
                onDismissDeleteDialog = {},
                onDeleteConfirmed = {},
            )
        }

        composeRule.onNodeWithTag(IdentityTestTags.selectButton(identity.id)).performClick()

        composeRule.runOnIdle {
            assertEquals(identity.id, selectedId)
        }
    }

    @Test
    fun delete_dialog_should_trigger_confirm_and_cancel_callbacks() {
        val identity = sampleIdentity(id = "identity-delete")
        var confirmClicked = false
        var cancelClicked = false

        composeRule.setLiaoTestContent {
            IdentityScreenContent(
                state = IdentityUiState(
                    identities = listOf(identity),
                    loading = false,
                    confirmDeleteIdentity = identity,
                ),
                onNameChange = {},
                onSexChange = {},
                onSubmitIdentity = {},
                onQuickCreate = {},
                onCancelEditing = {},
                onSelectIdentity = {},
                onStartEditing = {},
                onConfirmDeleteIdentity = {},
                onDismissDeleteDialog = { cancelClicked = true },
                onDeleteConfirmed = { confirmClicked = true },
            )
        }

        composeRule.onNodeWithTag(IdentityTestTags.DELETE_DIALOG_CONFIRM).performClick()
        composeRule.onNodeWithTag(IdentityTestTags.DELETE_DIALOG_CANCEL).performClick()

        composeRule.runOnIdle {
            assertTrue(confirmClicked)
            assertTrue(cancelClicked)
        }
    }

    private fun sampleIdentity(id: String) = IdentityDto(
        id = id,
        name = "测试身份",
        sex = "女",
        lastUsedAt = "2026-03-16 10:00:00",
    )
}
