package io.github.a7413498.liao.android.feature.douyin

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.DouyinApiService
import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class DouyinRepositoryTest {
    private val douyinApiService = mockk<DouyinApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val repository = DouyinRepository(douyinApiService, preferencesStore, baseUrlProvider)

    init {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()
    }

    @Test
    fun `resolve detail should normalize urls and infer media items`() = runTest {
        coEvery { douyinApiService.getDetail(any()) } returns buildJsonObject {
            put("key", JsonPrimitive("key-1"))
            put("detailId", JsonPrimitive("detail-1"))
            put("type", JsonPrimitive("image"))
            put("title", JsonPrimitive("作品"))
            put("coverUrl", JsonPrimitive("/cover/main.jpg"))
            put("imageCount", JsonPrimitive("2"))
            put(
                "items",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("index", JsonPrimitive(0))
                            put("type", JsonPrimitive("unknown"))
                            put("url", JsonPrimitive("gallery/1"))
                            put("downloadUrl", JsonPrimitive("/download/1.jpg"))
                            put("originalFilename", JsonPrimitive("still"))
                        }
                    )
                    add(
                        buildJsonObject {
                            put("index", JsonPrimitive(1))
                            put("url", JsonPrimitive("/aweme/v1/play/?id=9"))
                            put("downloadUrl", JsonPrimitive("https://cdn.test/play"))
                        }
                    )
                }
            )
        }

        val result = repository.resolveDetail(input = " share text ", cookie = "")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("imageAlbum", payload.mediaType)
        assertEquals("https://demo.test:8443/cover/main.jpg", payload.coverUrl)
        assertEquals(2, payload.items.size)
        assertEquals("image", payload.items[0].type)
        assertEquals("https://demo.test:8443/download/1.jpg", payload.items[0].downloadUrl)
        assertEquals("https://demo.test:8443/download/1.jpg", payload.items[0].thumbUrl)
        assertEquals("video", payload.items[1].type)
        assertEquals("https://demo.test:8443/aweme/v1/play/?id=9", payload.items[1].url)
        assertEquals("https://demo.test:8443/cover/main.jpg", payload.items[1].thumbUrl)
    }

    @Test
    fun `resolve account should parse counts skip invalid items and derive live photo media type`() = runTest {
        coEvery { douyinApiService.getAccount(any()) } returns buildJsonObject {
            put("secUserId", JsonPrimitive("sec-1"))
            put("displayName", JsonPrimitive("Alice"))
            put("followerCount", JsonPrimitive("12"))
            put("followingCount", JsonPrimitive("4"))
            put(
                "items",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("detailId", JsonPrimitive("detail-1"))
                            put("type", JsonPrimitive("image"))
                            put("isLivePhoto", JsonPrimitive("true"))
                            put("imageCount", JsonPrimitive("1"))
                            put("livePhotoPairs", JsonPrimitive("2"))
                            put("isPinned", JsonPrimitive("true"))
                            put("pinnedRank", JsonPrimitive("bad"))
                            put("coverUrl", JsonPrimitive("covers/item.jpg"))
                            put("authorUniqueId", JsonPrimitive("author-1"))
                            put("authorName", JsonPrimitive("作者"))
                        }
                    )
                    add(buildJsonObject { put("detailId", JsonPrimitive("   ")) })
                }
            )
        }

        val result = repository.resolveAccount(input = " https://douyin.test/user ", cookie = "cookie")

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(12L, payload.followerCount)
        assertEquals(4L, payload.followingCount)
        assertEquals(1, payload.items.size)
        assertEquals("livePhoto", payload.items.single().mediaType)
        assertEquals(true, payload.items.single().isPinned)
        assertNull(payload.items.single().pinnedRank)
        assertEquals("https://demo.test:8443/covers/item.jpg", payload.items.single().coverUrl)
    }

    @Test
    fun `refresh favorites snapshot should aggregate favorites and tags`() = runTest {
        coEvery { douyinApiService.listFavoriteUsers() } returns buildJsonObject {
            put(
                "items",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("secUserId", JsonPrimitive("sec-1"))
                            put("displayName", JsonPrimitive("收藏作者"))
                            put(
                                "tagIds",
                                buildJsonArray {
                                    add(JsonPrimitive("1"))
                                    add(JsonPrimitive("bad"))
                                    add(JsonPrimitive(3))
                                }
                            )
                        }
                    )
                    add(buildJsonObject { put("secUserId", JsonPrimitive("")) })
                }
            )
        }
        coEvery { douyinApiService.listFavoriteAwemes() } returns buildJsonObject {
            put(
                "items",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("awemeId", JsonPrimitive("aweme-1"))
                            put("type", JsonPrimitive("video"))
                        }
                    )
                    add(buildJsonObject { put("awemeId", JsonPrimitive(" ")) })
                }
            )
        }
        coEvery { douyinApiService.listFavoriteUserTags() } returns buildJsonObject {
            put(
                "items",
                buildJsonArray {
                    add(buildJsonObject { put("id", JsonPrimitive(1)); put("name", JsonPrimitive("人物")) })
                    add(buildJsonObject { put("name", JsonPrimitive("缺失ID")) })
                }
            )
        }
        coEvery { douyinApiService.listFavoriteAwemeTags() } returns buildJsonObject {
            put(
                "items",
                buildJsonArray {
                    add(buildJsonObject { put("id", JsonPrimitive(9)); put("name", JsonPrimitive("视频")); put("count", JsonPrimitive("2")) })
                }
            )
        }

        val result = repository.refreshFavoritesSnapshot()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(1, payload.users.size)
        assertEquals(listOf(1L, 3L), payload.users.single().tagIds)
        assertEquals(1, payload.awemes.size)
        assertEquals("aweme-1", payload.awemes.single().awemeId)
        assertEquals(1, payload.userTags.size)
        assertEquals(1L, payload.userTags.single().id)
        assertEquals(1, payload.awemeTags.size)
        assertEquals(2L, payload.awemeTags.single().count)
    }

    @Test
    fun `import media should fallback to pre identity and return parsed result`() = runTest {
        val userIdSlot = slot<String>()
        val keySlot = slot<String>()
        val indexSlot = slot<Int>()
        coEvery { preferencesStore.readCurrentSession() } returns null
        coEvery { douyinApiService.importMedia(capture(userIdSlot), capture(keySlot), capture(indexSlot)) } returns buildJsonObject {
            put("state", JsonPrimitive("OK"))
            put("localPath", JsonPrimitive("/upload/douyin/a.jpg"))
            put("localFilename", JsonPrimitive("a.jpg"))
            put("dedup", JsonPrimitive("true"))
        }

        val result = repository.importMedia(key = " key-1 ", index = 2)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals("/upload/douyin/a.jpg", payload.localPath)
        assertEquals("a.jpg", payload.localFilename)
        assertEquals(true, payload.dedup)
        assertEquals("pre_identity", userIdSlot.captured)
        assertEquals("key-1", keySlot.captured)
        assertEquals(2, indexSlot.captured)
    }
}
