package io.github.a7413498.liao.android.feature.identity

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.database.IdentityDao
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class IdentityRepositoryTest {
    private val identityApiService = mockk<IdentityApiService>()
    private val identityDao = mockk<IdentityDao>(relaxUnitFun = true)
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val repository = IdentityRepository(identityApiService, identityDao, preferencesStore)

    @Test
    fun `load identities should cache remote items`() = runTest {
        val items = listOf(IdentityDto(id = "id-1", name = "Alice", sex = "女", createdAt = "c", lastUsedAt = "l"))
        coEvery { identityApiService.getIdentityList() } returns ApiEnvelope(code = 0, data = items)

        val result = repository.loadIdentities()

        assertTrue(result is AppResult.Success)
        assertEquals(items, (result as AppResult.Success).data)
        coVerify { identityDao.replaceAll(match { it.single().id == "id-1" && it.single().name == "Alice" }) }
    }

    @Test
    fun `load identities should return error when api throws`() = runTest {
        coEvery { identityApiService.getIdentityList() } throws IllegalStateException("boom")

        val result = repository.loadIdentities()

        assertTrue(result is AppResult.Error)
        assertEquals("boom", (result as AppResult.Error).message)
    }

    @Test
    fun `create identity should return created item`() = runTest {
        val created = IdentityDto(id = "id-1", name = "Alice", sex = "女")
        coEvery { identityApiService.createIdentity("Alice", "女") } returns ApiEnvelope(code = 0, data = created)

        val result = repository.createIdentity("Alice", "女")

        assertTrue(result is AppResult.Success)
        assertEquals(created, (result as AppResult.Success).data)
    }

    @Test
    fun `create identity should return error when response data missing`() = runTest {
        coEvery { identityApiService.createIdentity("Alice", "女") } returns ApiEnvelope(code = 0, msg = "创建身份失败", data = null)

        val result = repository.createIdentity("Alice", "女")

        assertTrue(result is AppResult.Error)
        assertEquals("创建身份失败", (result as AppResult.Error).message)
    }

    @Test
    fun `update identity should sync current session when editing active identity`() = runTest {
        val updated = IdentityDto(id = "id-1", name = "Alice2", sex = "男")
        val savedSession = slot<CurrentIdentitySession>()
        coEvery { identityApiService.updateIdentity("id-1", "Alice2", "男") } returns ApiEnvelope(code = 0, data = updated)
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
        )
        coEvery { preferencesStore.saveCurrentSession(capture(savedSession)) } just runs

        val result = repository.updateIdentity("id-1", "Alice2", "男")

        assertTrue(result is AppResult.Success)
        assertEquals(updated, (result as AppResult.Success).data)
        assertEquals("Alice2", savedSession.captured.name)
        assertEquals("男", savedSession.captured.sex)
    }

    @Test
    fun `update identity should not save session when current identity differs`() = runTest {
        val updated = IdentityDto(id = "id-1", name = "Alice2", sex = "男")
        coEvery { identityApiService.updateIdentity("id-1", "Alice2", "男") } returns ApiEnvelope(code = 0, data = updated)
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-2",
            name = "Bob",
            sex = "男",
            cookie = "cookie",
            ip = "2.2.2.2",
        )

        val result = repository.updateIdentity("id-1", "Alice2", "男")

        assertTrue(result is AppResult.Success)
        coVerify(exactly = 0) { preferencesStore.saveCurrentSession(any()) }
    }

    @Test
    fun `delete identity should clear current session when deleting active identity`() = runTest {
        coEvery { identityApiService.deleteIdentity("id-1") } returns ApiEnvelope(code = 0)
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "id-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
        )
        coEvery { preferencesStore.clearCurrentSession() } just runs

        val result = repository.deleteIdentity("id-1")

        assertTrue(result is AppResult.Success)
        coVerify { preferencesStore.clearCurrentSession() }
    }

    @Test
    fun `delete identity should return error when api code not zero`() = runTest {
        coEvery { identityApiService.deleteIdentity("id-1") } returns ApiEnvelope(code = 1, msg = "删除失败")

        val result = repository.deleteIdentity("id-1")

        assertTrue(result is AppResult.Error)
        assertEquals("删除失败", (result as AppResult.Error).message)
    }

    @Test
    fun `quick create should return dto when response valid`() = runTest {
        val created = IdentityDto(id = "id-quick", name = "Quick", sex = "女")
        coEvery { identityApiService.quickCreateIdentity() } returns ApiEnvelope(code = 0, data = created)

        val result = repository.quickCreate()

        assertTrue(result is AppResult.Success)
        assertEquals(created, (result as AppResult.Success).data)
    }

    @Test
    fun `quick create should surface api error when data missing`() = runTest {
        coEvery { identityApiService.quickCreateIdentity() } returns ApiEnvelope(code = 0, msg = "快速创建失败", data = null)

        val result = repository.quickCreate()

        assertTrue(result is AppResult.Error)
        assertEquals("快速创建失败", (result as AppResult.Error).message)
    }

    @Test
    fun `select identity should persist generated session`() = runTest {
        val savedSession = slot<CurrentIdentitySession>()
        val identity = IdentityDto(id = "id-2", name = "Bob", sex = "男")
        coEvery { identityApiService.selectIdentity("id-2") } returns ApiEnvelope(code = 0, data = identity)
        coEvery { preferencesStore.saveCurrentSession(capture(savedSession)) } just runs

        val result = repository.selectIdentity(identity)

        assertTrue(result is AppResult.Success)
        assertEquals("id-2", savedSession.captured.id)
        assertEquals("Bob", savedSession.captured.name)
        assertEquals("男", savedSession.captured.sex)
        assertTrue(savedSession.captured.cookie.startsWith("id-2_Bob_"))
        assertTrue(savedSession.captured.ip.matches(Regex("\\d+\\.\\d+\\.\\d+\\.\\d+")))
        assertNotNull(savedSession.captured.area)
    }
}
