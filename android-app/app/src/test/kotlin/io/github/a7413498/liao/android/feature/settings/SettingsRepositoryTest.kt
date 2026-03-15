package io.github.a7413498.liao.android.feature.settings

import io.github.a7413498.liao.android.app.theme.LiaoThemePreference
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.ConnectionStatsDto
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.network.SystemApiService
import io.github.a7413498.liao.android.core.network.SystemConfigDto
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import io.mockk.verify
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class SettingsRepositoryTest {
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val identityApiService = mockk<IdentityApiService>()
    private val systemApiService = mockk<SystemApiService>()
    private val webSocketClient = mockk<LiaoWebSocketClient>()
    private val conversationDao = mockk<ConversationDao>(relaxUnitFun = true)
    private val messageDao = mockk<MessageDao>(relaxUnitFun = true)
    private val repository = SettingsRepository(
        preferencesStore = preferencesStore,
        identityApiService = identityApiService,
        systemApiService = systemApiService,
        webSocketClient = webSocketClient,
        conversationDao = conversationDao,
        messageDao = messageDao,
    )

    @Test
    fun `load snapshot should aggregate remote values and cache remote config`() = runTest {
        val remoteConfig = SystemConfigDto(
            imagePortMode = "probe",
            imagePortFixed = "9010",
            imagePortRealMinBytes = 4096,
            mtPhotoTimelineDeferSubfolderThreshold = 12,
        )
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "1234567890",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery { preferencesStore.readBaseUrl() } returns "https://demo.test"
        coEvery { preferencesStore.readAuthToken() } returns "1234567890abcdefghijklmn"
        coEvery { systemApiService.getConnectionStats() } returns ApiEnvelope(code = 0, data = ConnectionStatsDto(active = 2, upstream = 3, downstream = 4))
        coEvery { systemApiService.getForceoutUserCount() } returns ApiEnvelope(code = 0, data = 6)
        coEvery { systemApiService.getSystemConfig() } returns ApiEnvelope(code = 0, data = remoteConfig)
        coEvery { preferencesStore.readThemePreference() } returns LiaoThemePreference.AUTO
        coEvery { preferencesStore.saveCachedSystemConfig(remoteConfig) } just runs

        val result = repository.loadSnapshot()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("https://demo.test", payload.baseUrl)
        assertEquals("1234567890abcdef...", payload.tokenPreview)
        assertEquals("1234567890", payload.currentIdentityId)
        assertEquals("Alice", payload.currentIdentityName)
        assertEquals("Alice (12345678) / 1.1.1.1 / 深圳", payload.currentIdentityPreview)
        assertEquals(ConnectionStatsDto(active = 2, upstream = 3, downstream = 4), payload.connectionStats)
        assertEquals(6, payload.forceoutUserCount)
        assertEquals(remoteConfig, payload.systemConfig)
        assertEquals(LiaoThemePreference.AUTO, payload.themePreference)
        coVerify { preferencesStore.saveCachedSystemConfig(remoteConfig) }
    }

    @Test
    fun `load snapshot should fallback to cached config and default counters when remote calls fail`() = runTest {
        val cachedConfig = SystemConfigDto(
            imagePortMode = "real",
            imagePortFixed = "9008",
            imagePortRealMinBytes = 8192,
            mtPhotoTimelineDeferSubfolderThreshold = 24,
        )
        coEvery { preferencesStore.readCurrentSession() } returns null
        coEvery { preferencesStore.readBaseUrl() } returns "https://demo.test"
        coEvery { preferencesStore.readAuthToken() } returns null
        coEvery { systemApiService.getConnectionStats() } throws IllegalStateException("stats down")
        coEvery { systemApiService.getForceoutUserCount() } throws IllegalStateException("forceout down")
        coEvery { systemApiService.getSystemConfig() } throws IllegalStateException("config down")
        coEvery { preferencesStore.readCachedSystemConfig() } returns cachedConfig
        coEvery { preferencesStore.readThemePreference() } returns LiaoThemePreference.DARK

        val result = repository.loadSnapshot()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("未登录", payload.tokenPreview)
        assertEquals("", payload.currentIdentityId)
        assertEquals("未选择身份", payload.currentIdentityPreview)
        assertEquals(ConnectionStatsDto(), payload.connectionStats)
        assertEquals(0, payload.forceoutUserCount)
        assertEquals(cachedConfig, payload.systemConfig)
        assertEquals(LiaoThemePreference.DARK, payload.themePreference)
        coVerify(exactly = 0) { preferencesStore.saveCachedSystemConfig(any()) }
    }

    @Test
    fun `save theme preference should persist selection`() = runTest {
        coEvery { preferencesStore.saveThemePreference(LiaoThemePreference.LIGHT) } just runs

        val result = repository.saveThemePreference(LiaoThemePreference.LIGHT)

        assertTrue(result is AppResult.Success)
        assertEquals(LiaoThemePreference.LIGHT, (result as AppResult.Success).data)
        coVerify { preferencesStore.saveThemePreference(LiaoThemePreference.LIGHT) }
    }

    @Test
    fun `save base url should trim value before persisting`() = runTest {
        coEvery { preferencesStore.saveBaseUrl("https://demo.test") } just runs

        val result = repository.saveBaseUrl("  https://demo.test  ")

        assertTrue(result is AppResult.Success)
        coVerify { preferencesStore.saveBaseUrl("https://demo.test") }
    }

    @Test
    fun `save identity should reject blank nickname`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )

        val result = repository.saveIdentity(identityId = "id-1", name = "   ", sex = "女")

        assertTrue(result is AppResult.Error)
        assertEquals("昵称不能为空", (result as AppResult.Error).message)
    }

    @Test
    fun `save identity should update id clear local caches and rebuild session`() = runTest {
        val savedSession = slot<CurrentIdentitySession>()
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery {
            identityApiService.updateIdentityId(
                oldId = "id-1",
                newId = "id-2",
                name = "Bob",
                sex = "男",
            )
        } returns ApiEnvelope(code = 0, data = IdentityDto(id = "id-2", name = "Bob", sex = "男"))
        coEvery { preferencesStore.saveCurrentSession(capture(savedSession)) } just runs

        val result = repository.saveIdentity(identityId = "id-2", name = "Bob", sex = "男")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("身份 ID 已更新，已重建本地会话", payload.message)
        assertEquals("id-2", payload.updatedIdentity.id)
        assertEquals("id-2", savedSession.captured.id)
        assertEquals("Bob", savedSession.captured.name)
        assertEquals("男", savedSession.captured.sex)
        assertTrue(savedSession.captured.cookie.startsWith("id-2_Bob_"))
        coVerify { conversationDao.clearAll() }
        coVerify { messageDao.clearAll() }
    }

    @Test
    fun `save identity should append sync notes when websocket updates fail`() = runTest {
        val savedSession = slot<CurrentIdentitySession>()
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery { identityApiService.updateIdentity(id = "id-1", name = "Bob", sex = "男") } returns ApiEnvelope(
            code = 0,
            data = IdentityDto(id = "id-1", name = "Bob", sex = "男"),
        )
        coEvery { preferencesStore.saveCurrentSession(capture(savedSession)) } just runs
        every { webSocketClient.sendModifyInfo(senderId = "id-1", userSex = "男") } returns false
        every { webSocketClient.sendChangeName(senderId = "id-1", newName = "Bob") } returns false

        val result = repository.saveIdentity(identityId = "id-1", name = "Bob", sex = "男")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("身份信息已保存，性别实时同步未发送、昵称实时同步未发送", payload.message)
        assertEquals("Bob", savedSession.captured.name)
        assertEquals("男", savedSession.captured.sex)
        verify { webSocketClient.sendModifyInfo(senderId = "id-1", userSex = "男") }
        verify { webSocketClient.sendChangeName(senderId = "id-1", newName = "Bob") }
    }

    @Test
    fun `save identity should return simple success message when nothing changed`() = runTest {
        val savedSession = slot<CurrentIdentitySession>()
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery { identityApiService.updateIdentity(id = "id-1", name = "Alice", sex = "女") } returns ApiEnvelope(
            code = 0,
            data = IdentityDto(id = "id-1", name = "Alice", sex = "女"),
        )
        coEvery { preferencesStore.saveCurrentSession(capture(savedSession)) } just runs

        val result = repository.saveIdentity(identityId = "id-1", name = "Alice", sex = "女")

        assertTrue(result is AppResult.Success)
        assertEquals("身份信息已保存", (result as AppResult.Success).data.message)
        assertEquals("Alice", savedSession.captured.name)
        verify(exactly = 0) { webSocketClient.sendModifyInfo(any(), any()) }
        verify(exactly = 0) { webSocketClient.sendChangeName(any(), any()) }
    }

    @Test
    fun `save system config should normalize inputs and cache fallback dto when response data missing`() = runTest {
        val payloadSlot = slot<JsonElement>()
        val cachedConfig = slot<SystemConfigDto>()
        coEvery { systemApiService.updateSystemConfig(capture(payloadSlot)) } returns ApiEnvelope(code = 0, data = null)
        coEvery { preferencesStore.saveCachedSystemConfig(capture(cachedConfig)) } just runs

        val result = repository.saveSystemConfig(
            imagePortMode = " unknown ",
            imagePortFixed = "   ",
            imagePortRealMinBytes = "0",
            mtPhotoTimelineDeferSubfolderThreshold = "-3",
        )

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("fixed", payload.imagePortMode)
        assertEquals("9006", payload.imagePortFixed)
        assertEquals(2048, payload.imagePortRealMinBytes)
        assertEquals(10, payload.mtPhotoTimelineDeferSubfolderThreshold)
        assertEquals(payload, cachedConfig.captured)
        val requestBody = payloadSlot.captured.jsonObject
        assertEquals("fixed", requestBody["imagePortMode"]?.jsonPrimitive?.content)
        assertEquals("9006", requestBody["imagePortFixed"]?.jsonPrimitive?.content)
        assertEquals("2048", requestBody["imagePortRealMinBytes"]?.jsonPrimitive?.content)
        assertEquals("10", requestBody["mtPhotoTimelineDeferSubfolderThreshold"]?.jsonPrimitive?.content)
    }

    @Test
    fun `save system config should surface api error when response has no data`() = runTest {
        coEvery { systemApiService.updateSystemConfig(any()) } returns ApiEnvelope(code = 1, msg = "保存失败", data = null)

        val result = repository.saveSystemConfig(
            imagePortMode = "probe",
            imagePortFixed = "9007",
            imagePortRealMinBytes = "4096",
            mtPhotoTimelineDeferSubfolderThreshold = "12",
        )

        assertTrue(result is AppResult.Error)
        assertEquals("保存失败", (result as AppResult.Error).message)
    }

    @Test
    fun `logout should disconnect websocket and clear local auth state`() = runTest {
        every { webSocketClient.disconnect(manual = true) } just runs
        coEvery { preferencesStore.clearAuthToken() } just runs
        coEvery { preferencesStore.clearCurrentSession() } just runs
        coEvery { conversationDao.clearAll() } just runs
        coEvery { messageDao.clearAll() } just runs

        val result = repository.logout()

        assertTrue(result is AppResult.Success)
        verify { webSocketClient.disconnect(manual = true) }
        coVerify { preferencesStore.clearAuthToken() }
        coVerify { preferencesStore.clearCurrentSession() }
        coVerify { conversationDao.clearAll() }
        coVerify { messageDao.clearAll() }
    }

    @Test
    fun `disconnect all connections should surface api error`() = runTest {
        coEvery { systemApiService.disconnectAllConnections() } returns ApiEnvelope(code = 1, msg = "断开失败")

        val result = repository.disconnectAllConnections()

        assertTrue(result is AppResult.Error)
        assertEquals("断开失败", (result as AppResult.Error).message)
    }

    @Test
    fun `clear forceout users should use fallback success message when response text missing`() = runTest {
        coEvery { systemApiService.clearForceoutUsers() } returns ApiEnvelope(code = 0, msg = null, message = null)

        val result = repository.clearForceoutUsers()

        assertTrue(result is AppResult.Success)
        assertEquals("已清空禁连用户", (result as AppResult.Success).data)
    }
}
