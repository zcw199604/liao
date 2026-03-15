package io.github.a7413498.liao.android.feature.auth

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.LiaoLogger
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.AuthApiService
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkObject
import io.mockk.unmockkObject
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class AuthRepositoryTest {
    private val authApiService = mockk<AuthApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val repository = AuthRepository(authApiService, preferencesStore)

    @Before
    fun setUp() {
        mockkObject(LiaoLogger)
        every { LiaoLogger.w(any(), any(), any()) } returns 0
    }

    @After
    fun tearDown() {
        unmockkObject(LiaoLogger)
    }

    @Test
    fun `login should save base url and token when response valid`() = runTest {
        coEvery { authApiService.login("code") } returns ApiEnvelope(code = 0, token = "jwt-token")

        val result = repository.login("http://host/api/", "code")

        assertTrue(result is AppResult.Success)
        coVerify { preferencesStore.saveBaseUrl("http://host/api/") }
        coVerify { preferencesStore.saveAuthToken("jwt-token") }
    }

    @Test
    fun `login should return error when response code invalid`() = runTest {
        coEvery { authApiService.login("bad") } returns ApiEnvelope(code = 1, msg = "访问码错误", token = "jwt")

        val result = repository.login("http://host/api/", "bad")

        assertTrue(result is AppResult.Error)
        assertEquals("访问码错误", (result as AppResult.Error).message)
        coVerify { preferencesStore.saveBaseUrl("http://host/api/") }
        coVerify(exactly = 0) { preferencesStore.saveAuthToken(any()) }
    }

    @Test
    fun `login should return error when token blank even if response code is zero`() = runTest {
        coEvery { authApiService.login("blank") } returns ApiEnvelope(code = 0, msg = "缺少 token", token = "   ")

        val result = repository.login("http://host/api/", "blank")

        assertTrue(result is AppResult.Error)
        assertEquals("缺少 token", (result as AppResult.Error).message)
        coVerify(exactly = 0) { preferencesStore.saveAuthToken(any()) }
    }

    @Test
    fun `restore session should return false when no token`() = runTest {
        coEvery { preferencesStore.readAuthToken() } returns null

        val result = repository.restoreSessionState()

        assertTrue(result is AppResult.Success)
        assertFalse((result as AppResult.Success).data)
        coVerify(exactly = 0) { authApiService.verify() }
    }

    @Test
    fun `restore session should clear state when token invalid`() = runTest {
        coEvery { preferencesStore.readAuthToken() } returns "jwt"
        coEvery { authApiService.verify() } returns ApiEnvelope(code = 0, valid = false)

        val result = repository.restoreSessionState()

        assertTrue(result is AppResult.Success)
        assertFalse((result as AppResult.Success).data)
        coVerify { preferencesStore.clearAuthToken() }
        coVerify { preferencesStore.clearCurrentSession() }
        coVerify(exactly = 0) { preferencesStore.readCurrentSession() }
    }

    @Test
    fun `restore session should return true when token valid and session exists`() = runTest {
        coEvery { preferencesStore.readAuthToken() } returns "jwt"
        coEvery { authApiService.verify() } returns ApiEnvelope(code = 0, valid = true)
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
        )

        val result = repository.restoreSessionState()

        assertTrue(result is AppResult.Success)
        assertTrue((result as AppResult.Success).data)
    }

    @Test
    fun `restore session should return false when token valid but session missing`() = runTest {
        coEvery { preferencesStore.readAuthToken() } returns "jwt"
        coEvery { authApiService.verify() } returns ApiEnvelope(code = 0, valid = true)
        coEvery { preferencesStore.readCurrentSession() } returns null

        val result = repository.restoreSessionState()

        assertTrue(result is AppResult.Success)
        assertFalse((result as AppResult.Success).data)
    }

    @Test
    fun `restore session should return error and clear state when verify throws`() = runTest {
        coEvery { preferencesStore.readAuthToken() } returns "jwt"
        coEvery { authApiService.verify() } throws IllegalStateException("network down")

        val result = repository.restoreSessionState()

        assertTrue(result is AppResult.Error)
        assertEquals("network down", (result as AppResult.Error).message)
        coVerify { preferencesStore.clearAuthToken() }
        coVerify { preferencesStore.clearCurrentSession() }
    }
}
