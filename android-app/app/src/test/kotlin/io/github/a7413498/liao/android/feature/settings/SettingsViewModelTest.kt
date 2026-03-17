package io.github.a7413498.liao.android.feature.settings

import io.github.a7413498.liao.android.app.theme.LiaoThemePreference
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.network.ConnectionStatsDto
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.network.SystemConfigDto
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import io.github.a7413498.liao.android.test.MainDispatcherRule
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class SettingsViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository = mockk<SettingsRepository>()
    private val webSocketClient = mockk<LiaoWebSocketClient>()
    private val stateFlow = MutableStateFlow<WebSocketState>(WebSocketState.Idle)

    @Test
    fun `init should load snapshot and observe websocket state`() = runTest(mainDispatcherRule.dispatcher) {
        every { webSocketClient.state } returns stateFlow
        coEvery { repository.loadSnapshot() } returns AppResult.Success(sampleSnapshot())

        val viewModel = SettingsViewModel(repository, webSocketClient)
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.loading)
        assertEquals("https://demo.test/api/", viewModel.uiState.baseUrl)
        assertEquals("Idle", viewModel.uiState.connectionStateLabel)

        stateFlow.value = WebSocketState.Connected
        advanceUntilIdle()
        assertEquals("Connected", viewModel.uiState.connectionStateLabel)
    }

    @Test
    fun `update theme preference should short circuit same value and handle success plus error`() = runTest(mainDispatcherRule.dispatcher) {
        every { webSocketClient.state } returns stateFlow
        coEvery { repository.loadSnapshot() } returns AppResult.Success(sampleSnapshot(themePreference = LiaoThemePreference.DARK))
        coEvery { repository.saveThemePreference(LiaoThemePreference.LIGHT) } returns AppResult.Success(LiaoThemePreference.LIGHT)
        coEvery { repository.saveThemePreference(LiaoThemePreference.AUTO) } returns AppResult.Error("主题保存失败")

        val viewModel = SettingsViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.updateThemePreference(LiaoThemePreference.DARK)
        advanceUntilIdle()
        coVerify(exactly = 0) { repository.saveThemePreference(LiaoThemePreference.DARK) }

        viewModel.updateThemePreference(LiaoThemePreference.LIGHT)
        advanceUntilIdle()
        assertEquals(LiaoThemePreference.LIGHT, viewModel.uiState.themePreference)
        assertEquals("已切换主题：浅色", viewModel.uiState.message)

        viewModel.updateThemePreference(LiaoThemePreference.AUTO)
        advanceUntilIdle()
        assertEquals("主题保存失败", viewModel.uiState.message)
    }

    @Test
    fun `save base url should handle success and error`() = runTest(mainDispatcherRule.dispatcher) {
        every { webSocketClient.state } returns stateFlow
        coEvery { repository.loadSnapshot() } returns AppResult.Success(sampleSnapshot(baseUrl = "https://initial.test/api/"))
        coEvery { repository.saveBaseUrl("https://ok.test/api/") } returns AppResult.Success(Unit)
        coEvery { repository.saveBaseUrl("https://bad.test/api/") } returns AppResult.Error("地址保存失败")

        val viewModel = SettingsViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.updateBaseUrl("https://ok.test/api/")
        viewModel.saveBaseUrl()
        advanceUntilIdle()
        assertEquals(false, viewModel.uiState.savingBaseUrl)
        assertEquals("已保存 Base URL", viewModel.uiState.message)

        viewModel.updateBaseUrl("https://bad.test/api/")
        viewModel.saveBaseUrl()
        advanceUntilIdle()
        assertEquals(false, viewModel.uiState.savingBaseUrl)
        assertEquals("地址保存失败", viewModel.uiState.message)
    }

    @Test
    fun `save system config should apply success result and surface errors`() = runTest(mainDispatcherRule.dispatcher) {
        every { webSocketClient.state } returns stateFlow
        coEvery { repository.loadSnapshot() } returns AppResult.Success(sampleSnapshot())
        coEvery {
            repository.saveSystemConfig(
                imagePortMode = "real",
                imagePortFixed = "9006",
                imagePortRealMinBytes = "4096",
                mtPhotoTimelineDeferSubfolderThreshold = "12",
            )
        } returns AppResult.Success(
            SystemConfigDto(
                imagePortMode = "real",
                imagePortFixed = "9007",
                imagePortRealMinBytes = 8192,
                mtPhotoTimelineDeferSubfolderThreshold = 15,
            ),
        )
        coEvery {
            repository.saveSystemConfig(
                imagePortMode = "probe",
                imagePortFixed = "9007",
                imagePortRealMinBytes = "8192",
                mtPhotoTimelineDeferSubfolderThreshold = "15",
            )
        } returns AppResult.Error("保存系统配置失败")

        val viewModel = SettingsViewModel(repository, webSocketClient)
        advanceUntilIdle()
        viewModel.updateImagePortMode("real")
        viewModel.updateImagePortRealMinBytes("4096")
        viewModel.updateMtPhotoTimelineDeferSubfolderThreshold("12")

        viewModel.saveSystemConfig()
        advanceUntilIdle()

        assertEquals(false, viewModel.uiState.savingSystemConfig)
        assertEquals("real", viewModel.uiState.imagePortMode)
        assertEquals("9007", viewModel.uiState.imagePortFixed)
        assertEquals("8192", viewModel.uiState.imagePortRealMinBytes)
        assertEquals("15", viewModel.uiState.mtPhotoTimelineDeferSubfolderThreshold)
        assertEquals("已保存图片端口策略", viewModel.uiState.message)

        viewModel.updateImagePortMode("probe")
        viewModel.saveSystemConfig()
        advanceUntilIdle()
        assertEquals(false, viewModel.uiState.savingSystemConfig)
        assertEquals("保存系统配置失败", viewModel.uiState.message)
    }

    @Test
    fun `consume message should clear existing message`() = runTest(mainDispatcherRule.dispatcher) {
        every { webSocketClient.state } returns stateFlow
        coEvery { repository.loadSnapshot() } returns AppResult.Success(sampleSnapshot())
        coEvery { repository.saveThemePreference(LiaoThemePreference.LIGHT) } returns AppResult.Success(LiaoThemePreference.LIGHT)

        val viewModel = SettingsViewModel(repository, webSocketClient)
        advanceUntilIdle()
        viewModel.updateThemePreference(LiaoThemePreference.LIGHT)
        advanceUntilIdle()
        assertEquals("已切换主题：浅色", viewModel.uiState.message)

        viewModel.consumeMessage()
        assertNull(viewModel.uiState.message)
    }

    @Test
    fun `save identity should refresh on success and surface repository error`() = runTest(mainDispatcherRule.dispatcher) {
        every { webSocketClient.state } returns stateFlow
        coEvery { repository.loadSnapshot() } returnsMany listOf(
            AppResult.Success(sampleSnapshot()),
            AppResult.Success(sampleSnapshot(identityName = "Bob", identitySex = "男")),
        )
        coEvery {
            repository.saveIdentity(
                identityId = "identity-1",
                name = "Bob",
                sex = "男",
            )
        } returns AppResult.Success(
            SaveIdentityResult(
                updatedIdentity = IdentityDto(id = "identity-1", name = "Bob", sex = "男"),
                message = "身份信息已保存",
            ),
        )
        coEvery {
            repository.saveIdentity(
                identityId = "identity-1",
                name = "Carol",
                sex = "女",
            )
        } returns AppResult.Error("身份保存失败")

        val viewModel = SettingsViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.updateIdentityName("Bob")
        viewModel.updateIdentitySex("男")
        viewModel.saveIdentity()
        advanceUntilIdle()
        assertEquals(false, viewModel.uiState.savingIdentity)
        assertEquals("Bob", viewModel.uiState.currentIdentityName)
        assertEquals("男", viewModel.uiState.currentIdentitySex)
        coVerify(exactly = 2) { repository.loadSnapshot() }

        viewModel.updateIdentityName("Carol")
        viewModel.updateIdentitySex("女")
        viewModel.saveIdentity()
        advanceUntilIdle()
        assertEquals(false, viewModel.uiState.savingIdentity)
        assertEquals("身份保存失败", viewModel.uiState.message)
    }

    @Test
    fun `admin actions should refresh on disconnect success surface forceout error and logout`() = runTest(mainDispatcherRule.dispatcher) {
        every { webSocketClient.state } returns stateFlow
        coEvery { repository.loadSnapshot() } returnsMany listOf(
            AppResult.Success(sampleSnapshot()),
            AppResult.Success(sampleSnapshot()),
        )
        coEvery { repository.disconnectAllConnections() } returns AppResult.Success("已请求断开全部连接")
        coEvery { repository.clearForceoutUsers() } returns AppResult.Error("清空禁连用户失败")
        coEvery { repository.logout() } returns AppResult.Success(Unit)

        val viewModel = SettingsViewModel(repository, webSocketClient)
        advanceUntilIdle()

        viewModel.disconnectAllConnections()
        advanceUntilIdle()
        assertEquals(false, viewModel.uiState.loading)
        coVerify(exactly = 2) { repository.loadSnapshot() }

        viewModel.clearForceoutUsers()
        advanceUntilIdle()
        assertEquals("清空禁连用户失败", viewModel.uiState.message)

        viewModel.logout()
        advanceUntilIdle()
        assertEquals(true, viewModel.uiState.loggedOut)
        assertEquals("已退出登录", viewModel.uiState.message)
    }

    private fun sampleSnapshot(
        baseUrl: String = "https://demo.test/api/",
        themePreference: LiaoThemePreference = LiaoThemePreference.DARK,
        identityName: String = "Alice",
        identitySex: String = "女",
    ): SettingsSnapshot = SettingsSnapshot(
        baseUrl = baseUrl,
        tokenPreview = "token-preview",
        currentIdentityId = "identity-1",
        currentIdentityName = identityName,
        currentIdentitySex = identitySex,
        currentIdentityPreview = "$identityName (identity) / 1.1.1.1 / 深圳",
        connectionStats = ConnectionStatsDto(active = 1, upstream = 2, downstream = 3),
        forceoutUserCount = 4,
        systemConfig = SystemConfigDto(
            imagePortMode = "fixed",
            imagePortFixed = "9006",
            imagePortRealMinBytes = 2048,
            mtPhotoTimelineDeferSubfolderThreshold = 10,
        ),
        themePreference = themePreference,
    )
}
