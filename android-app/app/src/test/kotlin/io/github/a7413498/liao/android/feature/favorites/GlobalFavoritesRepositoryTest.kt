package io.github.a7413498.liao.android.feature.favorites

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.GlobalFavoriteItem
import io.github.a7413498.liao.android.core.database.FavoriteDao
import io.github.a7413498.liao.android.core.database.FavoriteEntity
import io.github.a7413498.liao.android.core.database.IdentityDao
import io.github.a7413498.liao.android.core.database.IdentityEntity
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.FavoriteApiService
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class GlobalFavoritesRepositoryTest {
    private val favoriteApiService = mockk<FavoriteApiService>()
    private val favoriteDao = mockk<FavoriteDao>(relaxUnitFun = true)
    private val identityApiService = mockk<IdentityApiService>()
    private val identityDao = mockk<IdentityDao>(relaxUnitFun = true)
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val repository = GlobalFavoritesRepository(favoriteApiService, favoriteDao, identityApiService, identityDao, preferencesStore)

    @Test
    fun `load favorites should cache valid remote items and resolve names from cache`() = runTest {
        val captured = slot<List<FavoriteEntity>>()
        coEvery { favoriteApiService.listAllFavorites() } returns ApiEnvelope(
            code = 0,
            data = listOf(
                Json.parseToJsonElement("""{"id":"1","identityId":"id-1","targetUserId":"peer-1","targetUserName":"对端","createTime":"2026"}"""),
                Json.parseToJsonElement("""{"id":"not-number","identityId":"id-1"}"""),
            ),
        )
        coEvery { favoriteDao.clearAll() } just runs
        coEvery { favoriteDao.replaceAll(capture(captured)) } just runs
        coEvery { identityDao.getAll() } returns listOf(IdentityEntity("id-1", "Alice", "女", "", ""))

        val result = repository.loadFavorites()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(1, payload.items.size)
        assertEquals("Alice", payload.identityNames["id-1"])
        assertEquals("peer-1", captured.captured.single().targetUserId)
        coVerify(exactly = 0) { identityApiService.getIdentityList() }
    }

    @Test
    fun `load favorites should clear cache and skip replace all when remote list empty`() = runTest {
        coEvery { favoriteApiService.listAllFavorites() } returns ApiEnvelope(code = 0, data = emptyList())
        coEvery { favoriteDao.clearAll() } just runs
        coEvery { identityDao.getAll() } returns emptyList()

        val result = repository.loadFavorites()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertTrue(payload.items.isEmpty())
        assertTrue(payload.identityNames.isEmpty())
        coVerify { favoriteDao.clearAll() }
        coVerify(exactly = 0) { favoriteDao.replaceAll(any()) }
        coVerify(exactly = 0) { identityApiService.getIdentityList() }
    }

    @Test
    fun `load favorites should fallback to dao when remote response invalid`() = runTest {
        coEvery { favoriteApiService.listAllFavorites() } returns ApiEnvelope(code = 1, msg = "remote down", data = null)
        coEvery { favoriteDao.getAll() } returns listOf(FavoriteEntity(1, "id-1", "peer-1", "对端", "2026"))
        coEvery { identityDao.getAll() } returns emptyList()
        coEvery { identityApiService.getIdentityList() } returns ApiEnvelope(
            code = 0,
            data = listOf(IdentityDto("id-1", "Alice", "女")),
        )
        coEvery { identityDao.replaceAll(any()) } just runs

        val result = repository.loadFavorites()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(1, payload.items.size)
        assertEquals("Alice", payload.identityNames["id-1"])
    }

    @Test
    fun `remove favorite should delete dao row when remote success`() = runTest {
        coEvery { favoriteApiService.removeFavoriteById(1) } returns ApiEnvelope(code = 0)
        coEvery { favoriteDao.deleteById(1) } just runs

        val result = repository.removeFavoriteById(1)

        assertTrue(result is AppResult.Success)
        coVerify { favoriteDao.deleteById(1) }
    }

    @Test
    fun `remove favorite should return error when api code invalid`() = runTest {
        coEvery { favoriteApiService.removeFavoriteById(1) } returns ApiEnvelope(code = 1, msg = "取消失败")

        val result = repository.removeFavoriteById(1)

        assertTrue(result is AppResult.Error)
        assertEquals("取消失败", (result as AppResult.Error).message)
        coVerify(exactly = 0) { favoriteDao.deleteById(any()) }
    }

    @Test
    fun `switch identity should use cached identity and fallback peer name`() = runTest {
        val saved = slot<CurrentIdentitySession>()
        val item = GlobalFavoriteItem(
            id = 1,
            identityId = "id-1",
            targetUserId = "peer-1234",
            targetUserName = "",
            createTime = "2026",
        )
        coEvery { identityDao.getById("id-1") } returns IdentityEntity("id-1", "Alice", "女", "", "")
        coEvery { identityApiService.selectIdentity("id-1") } returns ApiEnvelope(code = 0, data = null)
        coEvery { preferencesStore.saveCurrentSession(capture(saved)) } just runs

        val result = repository.switchIdentityAndPrepareChat(item)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("peer-1234", payload.peerId)
        assertEquals("用户peer", payload.peerName)
        assertEquals("id-1", saved.captured.id)
        assertEquals("Alice", saved.captured.name)
        assertTrue(saved.captured.cookie.startsWith("id-1_Alice_"))
        assertTrue(saved.captured.ip.matches(Regex("\\d+\\.\\d+\\.\\d+\\.\\d+")))
    }

    @Test
    fun `switch identity should refresh cache when dao misses and use selected response`() = runTest {
        val cachedIdentities = slot<List<IdentityEntity>>()
        val saved = slot<CurrentIdentitySession>()
        val item = GlobalFavoriteItem(
            id = 2,
            identityId = "id-2",
            targetUserId = "peer-2",
            targetUserName = "对端二号",
            createTime = "2026",
        )
        coEvery { identityDao.getById("id-2") } returns null
        coEvery { identityApiService.getIdentityList() } returns ApiEnvelope(
            code = 0,
            data = listOf(IdentityDto("id-2", "CacheBob", "男", createdAt = "c", lastUsedAt = "l")),
        )
        coEvery { identityDao.replaceAll(capture(cachedIdentities)) } just runs
        coEvery { identityApiService.selectIdentity("id-2") } returns ApiEnvelope(
            code = 0,
            data = IdentityDto("id-2", "RemoteBob", "男"),
        )
        coEvery { preferencesStore.saveCurrentSession(capture(saved)) } just runs

        val result = repository.switchIdentityAndPrepareChat(item)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("对端二号", payload.peerName)
        assertEquals("CacheBob", cachedIdentities.captured.single().name)
        assertEquals("RemoteBob", saved.captured.name)
    }

    @Test
    fun `switch identity should return error when identity missing`() = runTest {
        val item = GlobalFavoriteItem(1, "id-x", "peer", "对端", "2026")
        coEvery { identityDao.getById("id-x") } returns null
        coEvery { identityApiService.getIdentityList() } returns ApiEnvelope(code = 0, data = emptyList())

        val result = repository.switchIdentityAndPrepareChat(item)

        assertTrue(result is AppResult.Error)
        assertEquals("身份不存在，无法切换", (result as AppResult.Error).message)
    }
}
