package io.github.a7413498.liao.android.feature.douyin

import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.DouyinApiService
import io.mockk.every
import io.mockk.mockk
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DouyinRepositoryMappingsTest {
    private val douyinApiService = mockk<DouyinApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val repository = DouyinRepository(douyinApiService, preferencesStore, baseUrlProvider)

    init {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()
    }

    @Test
    fun `media mapping helper should cover null blank and explicit type branches`() {
        assertNull(repository.mapMediaItemForTest(JsonPrimitive("oops"), fallbackCoverUrl = "https://demo.test/fallback.jpg"))
        assertNull(
            repository.mapMediaItemForTest(
                buildJsonObject {
                    put("downloadUrl", JsonPrimitive("   "))
                },
                fallbackCoverUrl = "https://demo.test/fallback.jpg",
            )
        )

        val image = repository.mapMediaItemForTest(
            buildJsonObject {
                put("index", JsonPrimitive(2))
                put("type", JsonPrimitive("IMAGE"))
                put("url", JsonPrimitive("gallery/1"))
                put("downloadUrl", JsonPrimitive("/download/1.jpg"))
                put("originalFilename", JsonPrimitive("still"))
            },
            fallbackCoverUrl = "https://demo.test/fallback.jpg",
        ) ?: error("missing image item")
        val video = repository.mapMediaItemForTest(
            buildJsonObject {
                put("type", JsonPrimitive("video"))
                put("url", JsonPrimitive("/play?id=1"))
                put("downloadUrl", JsonPrimitive("https://cdn.test/play.mp4"))
            },
            fallbackCoverUrl = "https://demo.test/fallback.jpg",
        ) ?: error("missing video item")

        assertEquals(2, image.index)
        assertEquals("image", image.type)
        assertEquals("https://demo.test:8443/gallery/1", image.url)
        assertEquals("https://demo.test:8443/download/1.jpg", image.downloadUrl)
        assertEquals("https://demo.test:8443/download/1.jpg", image.thumbUrl)
        assertEquals("video", video.type)
        assertEquals("https://demo.test:8443/play?id=1", video.url)
        assertEquals("https://demo.test/fallback.jpg", video.thumbUrl)
    }

    @Test
    fun `account and favorite mapping helpers should parse fallbacks and invalid records`() {
        assertNull(repository.mapAccountItemForTest(buildJsonObject { put("detailId", JsonPrimitive(" ")) }))
        val account = repository.mapAccountItemForTest(
            buildJsonObject {
                put("detailId", JsonPrimitive("aweme-1"))
                put("type", JsonPrimitive("image"))
                put("imageCount", JsonPrimitive("3"))
                put("coverUrl", JsonPrimitive("covers/item.jpg"))
                put("isPinned", JsonPrimitive("true"))
                put("pinnedRank", JsonPrimitive("3"))
                put("authorName", JsonPrimitive("作者"))
            }
        ) ?: error("missing account item")
        assertEquals("imageAlbum", account.mediaType)
        assertEquals("https://demo.test:8443/covers/item.jpg", account.coverUrl)
        assertEquals(true, account.isPinned)
        assertEquals(3, account.pinnedRank)
        assertEquals("作者", account.authorName)

        assertNull(repository.mapFavoriteUserForTest(buildJsonObject { put("secUserId", JsonPrimitive(" ")) }))
        val favoriteUser = repository.mapFavoriteUserForTest(
            buildJsonObject {
                put("secUserId", JsonPrimitive("sec-1"))
                put("avatarUrl", JsonPrimitive("/avatar.jpg"))
                put("profileUrl", JsonPrimitive("profile/sec-1"))
                put(
                    "tagIds",
                    buildJsonArray {
                        add(JsonPrimitive("1"))
                        add(JsonPrimitive("bad"))
                        add(JsonPrimitive(3))
                    },
                )
            }
        ) ?: error("missing favorite user")
        assertEquals("https://demo.test:8443/avatar.jpg", favoriteUser.avatarUrl)
        assertEquals("https://demo.test:8443/profile/sec-1", favoriteUser.profileUrl)
        assertEquals(listOf(1L, 3L), favoriteUser.tagIds)

        assertNull(repository.mapFavoriteAwemeForTest(buildJsonObject { put("awemeId", JsonPrimitive(" ")) }))
        val favoriteAweme = repository.mapFavoriteAwemeForTest(
            buildJsonObject {
                put("awemeId", JsonPrimitive("aweme-9"))
                put("coverUrl", JsonPrimitive("/cover.jpg"))
            }
        ) ?: error("missing favorite aweme")
        assertEquals("https://demo.test:8443/cover.jpg", favoriteAweme.coverUrl)

        assertNull(repository.mapFavoriteTagForTest(buildJsonObject { put("name", JsonPrimitive("人物")) }))
        val favoriteTag = repository.mapFavoriteTagForTest(
            buildJsonObject {
                put("id", JsonPrimitive(9))
                put("name", JsonPrimitive("人物"))
            }
        ) ?: error("missing favorite tag")
        assertEquals(0L, favoriteTag.sortOrder)
        assertEquals(0L, favoriteTag.count)
    }

    @Test
    fun `normalize url helper should cover blank absolute relative and default port`() {
        assertEquals("", repository.normalizeUrlForTest("   "))
        assertEquals("https://full.test/a.jpg", repository.normalizeUrlForTest(" https://full.test/a.jpg "))
        assertEquals("https://demo.test:8443/assets/a.jpg", repository.normalizeUrlForTest("/assets/a.jpg"))
        assertEquals("https://demo.test:8443/assets/a.jpg", repository.normalizeUrlForTest("assets/a.jpg"))

        val defaultPortProvider = mockk<BaseUrlProvider>()
        every { defaultPortProvider.currentApiBaseUrl() } returns "http://demo.test/api/".toHttpUrl()
        val defaultPortRepository = DouyinRepository(douyinApiService, preferencesStore, defaultPortProvider)
        assertEquals("http://demo.test/assets/a.jpg", defaultPortRepository.normalizeUrlForTest("assets/a.jpg"))
    }
}
