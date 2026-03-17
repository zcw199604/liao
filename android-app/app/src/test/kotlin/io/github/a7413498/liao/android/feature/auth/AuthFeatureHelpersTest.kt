package io.github.a7413498.liao.android.feature.auth

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AuthFeatureHelpersTest {
    @Test
    fun `login action should require non blank access code when idle`() {
        assertFalse(isLoginActionEnabled(LoginUiState(accessCode = "", loading = false)))
        assertFalse(isLoginActionEnabled(LoginUiState(accessCode = "   ", loading = false)))
        assertTrue(isLoginActionEnabled(LoginUiState(accessCode = "access-code", loading = false)))
    }

    @Test
    fun `login action should stay disabled while loading`() {
        assertFalse(isLoginActionEnabled(LoginUiState(accessCode = "access-code", loading = true)))
    }
}
