package io.github.a7413498.liao.android.feature.auth

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.mockk
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LoginViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val authRepository = mockk<AuthRepository>()
    private val preferencesStore = mockk<AppPreferencesStore>()

    @Test
    fun `init should restore valid session state`() = runTest(mainDispatcherRule.dispatcher) {
        coEvery { preferencesStore.readBaseUrl() } returns "https://demo.test/api/"
        coEvery { authRepository.restoreSessionState() } returns AppResult.Success(true)
        coEvery { preferencesStore.readAuthToken() } returns "jwt-token"

        val viewModel = LoginViewModel(authRepository, preferencesStore)
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.loading)
        assertTrue(viewModel.uiState.loggedIn)
        assertTrue(viewModel.uiState.hasCurrentSession)
        assertEquals("https://demo.test/api/", viewModel.uiState.baseUrl)
    }

    @Test
    fun `init should surface restore error`() = runTest(mainDispatcherRule.dispatcher) {
        coEvery { preferencesStore.readBaseUrl() } returns "https://demo.test/api/"
        coEvery { authRepository.restoreSessionState() } returns AppResult.Error("恢复失败")

        val viewModel = LoginViewModel(authRepository, preferencesStore)
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.loading)
        assertFalse(viewModel.uiState.loggedIn)
        assertEquals("恢复失败", viewModel.uiState.errorMessage)
    }

    @Test
    fun `login should update state on success and error`() = runTest(mainDispatcherRule.dispatcher) {
        coEvery { preferencesStore.readBaseUrl() } returns "https://demo.test/api/"
        coEvery { authRepository.restoreSessionState() } returns AppResult.Success(false)
        coEvery { preferencesStore.readAuthToken() } returns null
        coEvery { authRepository.login("https://demo.test/api/", "code-ok") } returns AppResult.Success(Unit)
        coEvery { authRepository.login("https://demo.test/api/", "code-bad") } returns AppResult.Error("登录失败")

        val viewModel = LoginViewModel(authRepository, preferencesStore)
        advanceUntilIdle()

        viewModel.updateAccessCode("code-ok")
        viewModel.login()
        advanceUntilIdle()
        assertTrue(viewModel.uiState.loggedIn)
        assertEquals(false, viewModel.uiState.loading)
        assertEquals(null, viewModel.uiState.errorMessage)

        viewModel.updateAccessCode("code-bad")
        viewModel.login()
        advanceUntilIdle()
        assertEquals(false, viewModel.uiState.loading)
        assertEquals("登录失败", viewModel.uiState.errorMessage)
    }
}
