package io.github.a7413498.liao.android.feature.settings

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.assertTextContains
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.a7413498.liao.android.app.testing.SettingsTestTags
import io.github.a7413498.liao.android.app.theme.LiaoThemePreference
import io.github.a7413498.liao.android.test.setLiaoTestContent
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class SettingsScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun clicking_light_theme_should_update_summary() {
        var state by mutableStateOf(
            SettingsUiState(
                loading = false,
                themePreference = LiaoThemePreference.AUTO,
                baseUrl = "http://10.0.2.2:8080/api/",
            ),
        )

        composeRule.setLiaoTestContent {
            SettingsScreenContent(
                state = state,
                systemDark = false,
                onBack = {},
                onUpdateThemePreference = { state = state.copy(themePreference = it) },
                onBaseUrlChange = { state = state.copy(baseUrl = it) },
                onSaveBaseUrl = {},
                onIdentityIdChange = {},
                onIdentityNameChange = {},
                onIdentitySexChange = {},
                onSaveIdentity = {},
                onOpenGlobalFavorites = {},
                onOpenMediaLibrary = {},
                onOpenMtPhoto = {},
                onOpenDouyin = {},
                onOpenVideoExtract = {},
                onOpenVideoExtractTasks = {},
                onRefresh = {},
                onDisconnectAllConnections = {},
                onClearForceoutUsers = {},
                onUpdateImagePortMode = { state = state.copy(imagePortMode = it) },
                onUpdateImagePortFixed = { state = state.copy(imagePortFixed = it) },
                onUpdateImagePortRealMinBytes = { state = state.copy(imagePortRealMinBytes = it) },
                onUpdateMtPhotoTimelineDeferSubfolderThreshold = { state = state.copy(mtPhotoTimelineDeferSubfolderThreshold = it) },
                onSaveSystemConfig = {},
                onLogout = {},
            )
        }

        composeRule.onNodeWithTag(SettingsTestTags.THEME_SUMMARY).assertTextContains("当前：浅色")
        composeRule.onNodeWithTag(SettingsTestTags.THEME_LIGHT_BUTTON).performClick()
        composeRule.onNodeWithTag(SettingsTestTags.THEME_SUMMARY).assertTextContains("主题：浅色")
    }

    @Test
    fun base_url_save_button_should_enable_after_input_and_trigger_save() {
        var state by mutableStateOf(SettingsUiState(loading = false, baseUrl = ""))
        var saveClicks = 0

        composeRule.setLiaoTestContent {
            SettingsScreenContent(
                state = state,
                systemDark = false,
                onBack = {},
                onUpdateThemePreference = { state = state.copy(themePreference = it) },
                onBaseUrlChange = { state = state.copy(baseUrl = it) },
                onSaveBaseUrl = { saveClicks += 1 },
                onIdentityIdChange = {},
                onIdentityNameChange = {},
                onIdentitySexChange = {},
                onSaveIdentity = {},
                onOpenGlobalFavorites = {},
                onOpenMediaLibrary = {},
                onOpenMtPhoto = {},
                onOpenDouyin = {},
                onOpenVideoExtract = {},
                onOpenVideoExtractTasks = {},
                onRefresh = {},
                onDisconnectAllConnections = {},
                onClearForceoutUsers = {},
                onUpdateImagePortMode = { state = state.copy(imagePortMode = it) },
                onUpdateImagePortFixed = { state = state.copy(imagePortFixed = it) },
                onUpdateImagePortRealMinBytes = { state = state.copy(imagePortRealMinBytes = it) },
                onUpdateMtPhotoTimelineDeferSubfolderThreshold = { state = state.copy(mtPhotoTimelineDeferSubfolderThreshold = it) },
                onSaveSystemConfig = {},
                onLogout = {},
            )
        }

        composeRule.onNodeWithTag(SettingsTestTags.SAVE_BASE_URL_BUTTON).assertIsNotEnabled()
        composeRule.onNodeWithTag(SettingsTestTags.BASE_URL_INPUT).performTextInput("https://liao.example/api/")
        composeRule.onNodeWithTag(SettingsTestTags.SAVE_BASE_URL_BUTTON).assertIsEnabled().performClick()

        composeRule.runOnIdle {
            assertEquals("https://liao.example/api/", state.baseUrl)
            assertEquals(1, saveClicks)
        }
    }

    @Test
    fun real_image_port_mode_should_show_real_min_bytes_field() {
        var state by mutableStateOf(
            SettingsUiState(
                loading = false,
                baseUrl = "http://10.0.2.2:8080/api/",
                imagePortMode = "fixed",
            ),
        )

        composeRule.setLiaoTestContent {
            SettingsScreenContent(
                state = state,
                systemDark = false,
                onBack = {},
                onUpdateThemePreference = { state = state.copy(themePreference = it) },
                onBaseUrlChange = { state = state.copy(baseUrl = it) },
                onSaveBaseUrl = {},
                onIdentityIdChange = {},
                onIdentityNameChange = {},
                onIdentitySexChange = {},
                onSaveIdentity = {},
                onOpenGlobalFavorites = {},
                onOpenMediaLibrary = {},
                onOpenMtPhoto = {},
                onOpenDouyin = {},
                onOpenVideoExtract = {},
                onOpenVideoExtractTasks = {},
                onRefresh = {},
                onDisconnectAllConnections = {},
                onClearForceoutUsers = {},
                onUpdateImagePortMode = { state = state.copy(imagePortMode = it) },
                onUpdateImagePortFixed = { state = state.copy(imagePortFixed = it) },
                onUpdateImagePortRealMinBytes = { state = state.copy(imagePortRealMinBytes = it) },
                onUpdateMtPhotoTimelineDeferSubfolderThreshold = { state = state.copy(mtPhotoTimelineDeferSubfolderThreshold = it) },
                onSaveSystemConfig = {},
                onLogout = {},
            )
        }

        composeRule.runOnIdle {
            assertEquals(0, composeRule.onAllNodesWithText("最小字节阈值").fetchSemanticsNodes().size)
        }
        composeRule.onNodeWithTag(SettingsTestTags.IMAGE_PORT_MODE_REAL_BUTTON).performScrollTo().performClick()
        composeRule.onNodeWithText("最小字节阈值").fetchSemanticsNode()
    }

    @Test
    fun feature_buttons_and_logout_should_trigger_callbacks() {
        var mediaLibraryClicks = 0
        var douyinClicks = 0
        var taskCenterClicks = 0
        var logoutClicks = 0

        composeRule.setLiaoTestContent {
            SettingsScreenContent(
                state = SettingsUiState(
                    loading = false,
                    baseUrl = "http://10.0.2.2:8080/api/",
                    currentIdentityId = "identity-1",
                    currentIdentityName = "测试身份",
                ),
                systemDark = false,
                onBack = {},
                onUpdateThemePreference = {},
                onBaseUrlChange = {},
                onSaveBaseUrl = {},
                onIdentityIdChange = {},
                onIdentityNameChange = {},
                onIdentitySexChange = {},
                onSaveIdentity = {},
                onOpenGlobalFavorites = {},
                onOpenMediaLibrary = { mediaLibraryClicks += 1 },
                onOpenMtPhoto = {},
                onOpenDouyin = { douyinClicks += 1 },
                onOpenVideoExtract = {},
                onOpenVideoExtractTasks = { taskCenterClicks += 1 },
                onRefresh = {},
                onDisconnectAllConnections = {},
                onClearForceoutUsers = {},
                onUpdateImagePortMode = {},
                onUpdateImagePortFixed = {},
                onUpdateImagePortRealMinBytes = {},
                onUpdateMtPhotoTimelineDeferSubfolderThreshold = {},
                onSaveSystemConfig = {},
                onLogout = { logoutClicks += 1 },
            )
        }

        composeRule.onNodeWithTag(SettingsTestTags.OPEN_MEDIA_LIBRARY_BUTTON).performScrollTo().performClick()
        composeRule.onNodeWithTag(SettingsTestTags.OPEN_DOUYIN_BUTTON).performScrollTo().performClick()
        composeRule.onNodeWithTag(SettingsTestTags.OPEN_VIDEO_EXTRACT_TASKS_BUTTON).performScrollTo().performClick()
        composeRule.onNodeWithTag(SettingsTestTags.LOGOUT_BUTTON).performScrollTo().performClick()

        composeRule.runOnIdle {
            assertEquals(1, mediaLibraryClicks)
            assertEquals(1, douyinClicks)
            assertEquals(1, taskCenterClicks)
            assertEquals(1, logoutClicks)
        }
    }
}
