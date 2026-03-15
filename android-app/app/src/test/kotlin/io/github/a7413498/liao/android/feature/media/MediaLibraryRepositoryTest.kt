package io.github.a7413498.liao.android.feature.media

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.datastore.CachedMediaLibraryItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedMediaLibrarySnapshot
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MediaApiService
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class MediaLibraryRepositoryTest {
    private val mediaApiService = mockk<MediaApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val repository = MediaLibraryRepository(mediaApiService, preferencesStore, baseUrlProvider)

    @Test
    fun `load media should normalize remote payload and cache first page`() = runTest {
        val savedSnapshot = slot<CachedMediaLibrarySnapshot>()
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test/api/".toHttpUrl()
        coEvery { mediaApiService.getAllUploadImages(page = 1, pageSize = 20) } returns buildJsonObject {
            put("page", JsonPrimitive(1))
            put("total", JsonPrimitive(2))
            put("totalPages", JsonPrimitive(3))
            put(
                "data",
                buildJsonArray {
                    add(
                        mediaItemJson(
                            url = "/upload/a.jpg",
                            originalFilename = "a.jpg",
                            type = "image",
                            updateTime = "2026-03-15 10:00:00",
                            source = "douyin",
                            fileSize = 1024,
                            posterUrl = "poster/a.jpg",
                        )
                    )
                    add(
                        mediaItemJson(
                            url = "clips/b.mp4",
                            originalFilename = "",
                            localFilename = "b.mp4",
                            type = "video",
                            updateTime = "2026-03-15 11:00:00",
                        )
                    )
                }
            )
        }
        coEvery { preferencesStore.saveCachedMediaLibrary(capture(savedSnapshot)) } just runs

        val result = repository.loadMedia(page = 1)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(1, payload.page)
        assertEquals(2, payload.total)
        assertEquals(3, payload.totalPages)
        assertEquals(2, payload.items.size)
        assertEquals("https://demo.test/upload/a.jpg", payload.items[0].url)
        assertEquals("/a.jpg", payload.items[0].localPath)
        assertEquals("图片 · a.jpg", payload.items[0].type.displayLabel(payload.items[0].title))
        assertEquals("douyin · 2026-03-15 10:00:00 · 1.0 KB", payload.items[0].subtitle)
        assertEquals("https://demo.test/upload/poster/a.jpg", payload.items[0].posterUrl)
        assertEquals("b.mp4", payload.items[1].title)
        assertEquals("/clips/b.mp4", payload.items[1].localPath)
        assertEquals("视频 · b.mp4", payload.items[1].type.displayLabel(payload.items[1].title))
        assertEquals(listOf("/a.jpg", "/clips/b.mp4"), savedSnapshot.captured.items.map { it.localPath })
    }

    @Test
    fun `load media page over one should merge existing cache and keep first duplicate`() = runTest {
        val savedSnapshot = slot<CachedMediaLibrarySnapshot>()
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test/api/".toHttpUrl()
        coEvery { preferencesStore.readCachedMediaLibrary() } returns CachedMediaLibrarySnapshot(
            items = listOf(
                CachedMediaLibraryItemSnapshot(
                    url = "https://demo.test/upload/same.jpg",
                    localPath = "/same.jpg",
                    type = "IMAGE",
                    title = "cached-same.jpg",
                    subtitle = "cached",
                ),
                CachedMediaLibraryItemSnapshot(
                    url = "https://demo.test/upload/old.jpg",
                    localPath = "/old.jpg",
                    type = "IMAGE",
                    title = "old.jpg",
                    subtitle = "old",
                ),
            ),
            page = 1,
            total = 2,
            totalPages = 1,
        )
        coEvery { mediaApiService.getAllUploadImages(page = 2, pageSize = 20) } returns buildJsonObject {
            put("page", JsonPrimitive(2))
            put("total", JsonPrimitive(4))
            put("totalPages", JsonPrimitive(2))
            put(
                "data",
                buildJsonArray {
                    add(mediaItemJson(url = "/upload/same.jpg", originalFilename = "remote-same.jpg", type = "image"))
                    add(mediaItemJson(url = "/upload/new.jpg", originalFilename = "new.jpg", type = "image"))
                }
            )
        }
        coEvery { preferencesStore.saveCachedMediaLibrary(capture(savedSnapshot)) } just runs

        val result = repository.loadMedia(page = 2)

        assertTrue(result is AppResult.Success)
        assertEquals(listOf("/same.jpg", "/old.jpg", "/new.jpg"), savedSnapshot.captured.items.map { it.localPath })
        assertEquals("cached-same.jpg", savedSnapshot.captured.items.first().title)
        assertEquals(2, savedSnapshot.captured.page)
        assertEquals(4, savedSnapshot.captured.total)
        assertEquals(2, savedSnapshot.captured.totalPages)
    }

    @Test
    fun `load media should fallback to cached snapshot when remote fails`() = runTest {
        coEvery { mediaApiService.getAllUploadImages(page = 3, pageSize = 20) } throws IllegalStateException("remote down")
        coEvery { preferencesStore.readCachedMediaLibrary() } returns CachedMediaLibrarySnapshot(
            items = listOf(
                CachedMediaLibraryItemSnapshot(
                    url = "https://cache.test/upload/video.mp4",
                    localPath = "/video.mp4",
                    type = "video",
                    title = "video.mp4",
                    subtitle = "cache",
                )
            ),
            page = 3,
            total = 1,
            totalPages = 5,
        )

        val result = repository.loadMedia(page = 3)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertTrue(payload.fromCache)
        assertEquals(3, payload.page)
        assertEquals(1, payload.items.size)
        assertEquals("/video.mp4", payload.items.single().localPath)
    }

    @Test
    fun `load media should return error when response shape invalid and no cache available`() = runTest {
        coEvery { mediaApiService.getAllUploadImages(page = 1, pageSize = 20) } returns JsonPrimitive("oops")
        coEvery { preferencesStore.readCachedMediaLibrary() } returns null

        val result = repository.loadMedia(page = 1)

        assertTrue(result is AppResult.Error)
        assertEquals("媒体库响应格式异常", (result as AppResult.Error).message)
    }

    @Test
    fun `delete media should normalize payload invoke api and prune cached items`() = runTest {
        val payloadSlot = slot<JsonElement>()
        val savedSnapshot = slot<CachedMediaLibrarySnapshot>()
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "identity-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery { mediaApiService.batchDeleteMedia(capture(payloadSlot)) } returns ApiEnvelope(code = 0, msg = "删除完成")
        coEvery { preferencesStore.readCachedMediaLibrary() } returns CachedMediaLibrarySnapshot(
            items = listOf(
                CachedMediaLibraryItemSnapshot("u1", "/a.jpg", "image", "a.jpg", ""),
                CachedMediaLibraryItemSnapshot("u2", "/b.jpg", "image", "b.jpg", ""),
            ),
            page = 1,
            total = 2,
            totalPages = 1,
        )
        coEvery { preferencesStore.saveCachedMediaLibrary(capture(savedSnapshot)) } just runs

        val result = repository.deleteMedia(listOf(" /a.jpg ", "/a.jpg", "", " /b.jpg "))

        assertTrue(result is AppResult.Success)
        assertEquals("删除完成", (result as AppResult.Success).data)
        val payload = payloadSlot.captured.jsonObject
        assertEquals("identity-1", payload["userId"]?.jsonPrimitive?.content)
        assertEquals(listOf("/a.jpg", "/b.jpg"), payload["localPaths"]?.jsonArray?.map { it.jsonPrimitive.content })
        assertTrue(savedSnapshot.captured.items.isEmpty())
        assertEquals(0, savedSnapshot.captured.total)
        assertEquals(1, savedSnapshot.captured.page)
        assertEquals(0, savedSnapshot.captured.totalPages)
    }

    @Test
    fun `delete media should reject empty normalized selection`() = runTest {
        val result = repository.deleteMedia(listOf(" ", "", "   "))

        assertTrue(result is AppResult.Error)
        assertEquals("请选择要删除的媒体", (result as AppResult.Error).message)
        coVerify(exactly = 0) { mediaApiService.batchDeleteMedia(any()) }
    }

    @Test
    fun `delete media should fail when current session missing`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns null

        val result = repository.deleteMedia(listOf("/a.jpg"))

        assertTrue(result is AppResult.Error)
        assertEquals("请先选择身份", (result as AppResult.Error).message)
    }

    @Test
    fun `delete media should surface api failure without touching cache`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns CurrentIdentitySession(
            id = "identity-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery { mediaApiService.batchDeleteMedia(any()) } returns ApiEnvelope(code = 1, msg = "删除失败")

        val result = repository.deleteMedia(listOf("/a.jpg"))

        assertTrue(result is AppResult.Error)
        assertEquals("删除失败", (result as AppResult.Error).message)
        coVerify(exactly = 0) { preferencesStore.saveCachedMediaLibrary(any()) }
    }

    private fun mediaItemJson(
        url: String,
        originalFilename: String? = null,
        localFilename: String? = null,
        type: String? = null,
        updateTime: String? = null,
        source: String? = null,
        fileSize: Long? = null,
        posterUrl: String? = null,
    ) = buildJsonObject {
        put("url", JsonPrimitive(url))
        originalFilename?.let { put("originalFilename", JsonPrimitive(it)) }
        localFilename?.let { put("localFilename", JsonPrimitive(it)) }
        type?.let { put("type", JsonPrimitive(it)) }
        updateTime?.let { put("updateTime", JsonPrimitive(it)) }
        source?.let { put("source", JsonPrimitive(it)) }
        fileSize?.let { put("fileSize", JsonPrimitive(it)) }
        posterUrl?.let { put("posterUrl", JsonPrimitive(it)) }
    }
}
