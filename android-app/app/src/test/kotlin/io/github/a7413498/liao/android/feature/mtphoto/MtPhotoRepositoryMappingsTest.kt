package io.github.a7413498.liao.android.feature.mtphoto

import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import io.github.a7413498.liao.android.core.network.SystemApiService
import io.mockk.every
import io.mockk.mockk
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class MtPhotoRepositoryMappingsTest {
    private val mtPhotoApiService = mockk<MtPhotoApiService>()
    private val systemApiService = mockk<SystemApiService>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val repository = MtPhotoRepository(mtPhotoApiService, systemApiService, baseUrlProvider, preferencesStore)

    init {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()
    }

    @Test
    fun `album and folder mapping helpers should cover null guards fallbacks and covers`() {
        assertNull(repository.mapAlbumSummaryForTest(JsonPrimitive("oops")))
        assertNull(repository.mapAlbumSummaryForTest(buildJsonObject { put("name", JsonPrimitive("missing id")) }))

        val fallbackAlbum = repository.mapAlbumSummaryForTest(
            buildJsonObject {
                put("id", JsonPrimitive(7))
                put("name", JsonPrimitive("   "))
                put("count", JsonPrimitive("4"))
            }
        ) ?: error("missing album")
        val coveredAlbum = repository.mapAlbumSummaryForTest(
            buildJsonObject {
                put("id", JsonPrimitive(8))
                put("name", JsonPrimitive("旅行"))
                put("cover", JsonPrimitive("cover-8"))
            }
        ) ?: error("missing covered album")
        val folder = repository.mapFolderSummaryForTest(
            buildJsonObject {
                put("id", JsonPrimitive(9))
                put("name", JsonPrimitive("   "))
                put("path", JsonPrimitive(""))
                put("cover", JsonPrimitive(" first-md5 , second-md5 "))
                put("s_cover", JsonPrimitive(" secondary-md5 "))
                put("subFolderNum", JsonPrimitive("2"))
                put("subFileNum", JsonPrimitive("5"))
            }
        ) ?: error("missing folder")

        assertEquals("相册 7", fallbackAlbum.name)
        assertEquals("", fallbackAlbum.coverUrl)
        assertEquals(4, fallbackAlbum.count)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=cover-8", coveredAlbum.coverUrl)
        assertEquals("目录 9", folder.name)
        assertEquals("", folder.path)
        assertEquals("secondary-md5", folder.coverMd5)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=secondary-md5", folder.coverUrl)
        assertEquals(2, folder.subFolderNum)
        assertEquals(5, folder.subFileNum)
    }

    @Test
    fun `media mapping helper should cover md5 alias title fallbacks subtitle and type`() {
        assertNull(repository.mapMediaSummaryForTest(JsonPrimitive("oops")))
        assertNull(repository.mapMediaSummaryForTest(buildJsonObject { put("fileType", JsonPrimitive("jpg")) }))

        val image = repository.mapMediaSummaryForTest(
            buildJsonObject {
                put("MD5", JsonPrimitive("0123456789abcdef"))
                put("day", JsonPrimitive("2026-03-15"))
                put("width", JsonPrimitive("1080"))
                put("height", JsonPrimitive("1920"))
            }
        ) ?: error("missing image summary")
        val video = repository.mapMediaSummaryForTest(
            buildJsonObject {
                put("id", JsonPrimitive(5))
                put("md5", JsonPrimitive("video-md5"))
                put("fileType", JsonPrimitive("MP4"))
                put("tokenAt", JsonPrimitive("clip-token"))
                put("width", JsonPrimitive("1920"))
                put("height", JsonPrimitive("1080"))
                put("duration", JsonPrimitive("8"))
            }
        ) ?: error("missing video summary")

        assertEquals(0L, image.id)
        assertEquals(ChatMessageType.IMAGE, image.type)
        assertEquals("0123456789ab", image.title)
        assertEquals("2026-03-15 · 1080×1920", image.subtitle)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=h220&md5=0123456789abcdef", image.thumbUrl)
        assertEquals(5L, video.id)
        assertEquals(ChatMessageType.VIDEO, video.type)
        assertEquals("clip-token", video.title)
        assertEquals("1920×1080 · 8s", video.subtitle)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=video-md5", video.thumbUrl)
    }

    @Test
    fun `thumb url helper should omit default port`() {
        assertEquals(
            "https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=cover-1",
            repository.thumbUrlForTest(md5 = "cover-1", size = "s260"),
        )

        val defaultPortProvider = mockk<BaseUrlProvider>()
        every { defaultPortProvider.currentApiBaseUrl() } returns "https://demo.test/api/".toHttpUrl()
        val defaultPortRepository = MtPhotoRepository(mtPhotoApiService, systemApiService, defaultPortProvider, preferencesStore)
        assertEquals(
            "https://demo.test/api/getMtPhotoThumb?size=h220&md5=cover-2",
            defaultPortRepository.thumbUrlForTest(md5 = "cover-2", size = "h220"),
        )
    }
}
