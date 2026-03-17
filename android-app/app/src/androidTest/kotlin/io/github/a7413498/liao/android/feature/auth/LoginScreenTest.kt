package io.github.a7413498.liao.android.feature.auth

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.a7413498.liao.android.app.testing.LoginTestTags
import io.github.a7413498.liao.android.test.setLiaoTestContent
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class LoginScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun login_button_should_enable_after_access_code_input_and_trigger_callback() {
        var state by mutableStateOf(
            LoginUiState(
                baseUrl = "http://10.0.2.2:8080/api/",
                accessCode = "",
                loading = false,
            ),
        )
        var loginClicks = 0

        composeRule.setLiaoTestContent {
            LoginScreenContent(
                state = state,
                onBaseUrlChange = { state = state.copy(baseUrl = it) },
                onAccessCodeChange = { state = state.copy(accessCode = it) },
                onLoginClick = { loginClicks += 1 },
            )
        }

        composeRule.onNodeWithTag(LoginTestTags.LOGIN_BUTTON).assertIsNotEnabled()
        composeRule.onNodeWithTag(LoginTestTags.ACCESS_CODE_INPUT).performTextInput("code-123")
        composeRule.onNodeWithTag(LoginTestTags.LOGIN_BUTTON).assertIsEnabled().performClick()

        composeRule.runOnIdle {
            assertEquals("code-123", state.accessCode)
            assertEquals(1, loginClicks)
        }
    }

    @Test
    fun loading_state_should_disable_inputs_and_show_progress() {
        composeRule.setLiaoTestContent {
            LoginScreenContent(
                state = LoginUiState(
                    baseUrl = "http://10.0.2.2:8080/api/",
                    accessCode = "ready",
                    loading = true,
                ),
                onBaseUrlChange = {},
                onAccessCodeChange = {},
                onLoginClick = {},
            )
        }

        composeRule.onNodeWithTag(LoginTestTags.BASE_URL_INPUT).assertIsNotEnabled()
        composeRule.onNodeWithTag(LoginTestTags.ACCESS_CODE_INPUT).assertIsNotEnabled()
        composeRule.onNodeWithTag(LoginTestTags.LOGIN_BUTTON).assertIsNotEnabled()
        composeRule.onNodeWithTag(LoginTestTags.LOADING_INDICATOR).fetchSemanticsNode()
    }
}
