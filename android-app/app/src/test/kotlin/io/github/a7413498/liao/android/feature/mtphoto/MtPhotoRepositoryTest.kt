package io.github.a7413498.liao.android.feature.mtphoto

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import io.github.a7413498.liao.android.core.network.SystemApiService
import io.github.a7413498.liao.android.core.network.SystemConfigDto
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class MtPhotoRepositoryTest {
    private val mtPhotoApiService = mockk<MtPhotoApiService>()
    private val systemApiService = mockk<SystemApiService>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val repository = MtPhotoRepository(mtPhotoApiService, systemApiService, baseUrlProvider, preferencesStore)

    init {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()
    }

    @Test
    fun `load albums should parse valid items apply defaults and thumb urls`() = runTest {
        coEvery { mtPhotoApiService.getAlbums() } returns buildJsonObject {
            put(
                "data",
                buildJsonArray {
                    add(buildJsonObject { put("id", JsonPrimitive(1)); put("cover", JsonPrimitive("cover-1")); put("count", JsonPrimitive("9")) })
                    add(buildJsonObject { put("name", JsonPrimitive("缺失 id")) })
                }
            )
        }

        val result = repository.loadAlbums()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(1, payload.size)
        assertEquals("相册 1", payload.single().name)
        assertEquals(9, payload.single().count)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=cover-1", payload.single().coverUrl)
    }

    @Test
    fun `load album files should parse image and video summaries`() = runTest {
        coEvery { mtPhotoApiService.getAlbumFiles(albumId = 1L, page = 2, pageSize = 60) } returns buildJsonObject {
            put("total", JsonPrimitive("2"))
            put("page", JsonPrimitive("2"))
            put("totalPages", JsonPrimitive("5"))
            put(
                "data",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("md5", JsonPrimitive("image-md5"))
                            put("fileType", JsonPrimitive("jpg"))
                            put("day", JsonPrimitive("2026-03-15"))
                            put("width", JsonPrimitive("1080"))
                            put("height", JsonPrimitive("1920"))
                        }
                    )
                    add(
                        buildJsonObject {
                            put("MD5", JsonPrimitive("video-md5"))
                            put("fileType", JsonPrimitive("MP4"))
                            put("tokenAt", JsonPrimitive("clip-token"))
                            put("duration", JsonPrimitive("12"))
                        }
                    )
                }
            )
        }

        val result = repository.loadAlbumFiles(albumId = 1L, page = 2)

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(2, payload.items.size)
        assertEquals(ChatMessageType.IMAGE, payload.items[0].type)
        assertEquals("image-md5", payload.items[0].title)
        assertEquals("2026-03-15 · 1080×1920", payload.items[0].subtitle)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=h220&md5=image-md5", payload.items[0].thumbUrl)
        assertEquals(ChatMessageType.VIDEO, payload.items[1].type)
        assertEquals("clip-token", payload.items[1].title)
        assertEquals("12s", payload.items[1].subtitle)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=video-md5", payload.items[1].thumbUrl)
        assertEquals(5, payload.totalPages)
    }

    @Test
    fun `load folder pages should normalize names covers and fallback pagination`() = runTest {
        coEvery { mtPhotoApiService.getFolderRoot() } returns buildJsonObject {
            put("path", JsonPrimitive("/root"))
            put(
                "folderList",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("id", JsonPrimitive(7))
                            put("name", JsonPrimitive("   "))
                            put("s_cover", JsonPrimitive("secondary-cover"))
                            put("subFolderNum", JsonPrimitive("2"))
                        }
                    )
                }
            )
            put(
                "fileList",
                buildJsonArray {
                    add(buildJsonObject { put("md5", JsonPrimitive("root-file")); put("fileType", JsonPrimitive("jpg")) })
                }
            )
        }
        coEvery { mtPhotoApiService.getFolderContent(folderId = 9L, page = 3, pageSize = 60, includeTimeline = false) } returns buildJsonObject {
            put("path", JsonPrimitive("C:\\photos\\cats"))
            put(
                "folderList",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("id", JsonPrimitive(10))
                            put("cover", JsonPrimitive(" first-md5 , second-md5 "))
                        }
                    )
                }
            )
            put(
                "fileList",
                buildJsonArray {
                    add(buildJsonObject { put("MD5", JsonPrimitive("folder-video")); put("fileType", JsonPrimitive("MP4")); put("duration", JsonPrimitive("8")) })
                }
            )
            put("total", JsonPrimitive("1"))
        }

        val rootResult = repository.loadFolderRoot()
        val folderResult = repository.loadFolderContent(folderId = 9L, page = 3, includeTimeline = false)

        assertTrue(rootResult is AppResult.Success)
        assertTrue(folderResult is AppResult.Success)
        val rootPage = (rootResult as AppResult.Success).data
        val folderPage = (folderResult as AppResult.Success).data
        assertEquals("根目录", rootPage.folderName)
        assertEquals("目录 7", rootPage.folders.single().name)
        assertEquals("secondary-cover", rootPage.folders.single().coverMd5)
        assertEquals("cats", folderPage.folderName)
        assertEquals("first-md5", folderPage.folders.single().coverMd5)
        assertEquals(3, folderPage.page)
        assertEquals(0, folderPage.totalPages)
        assertEquals(ChatMessageType.VIDEO, folderPage.files.single().type)
    }

    @Test
    fun `load timeline threshold should prefer remote then cached then default`() = runTest {
        coEvery { systemApiService.getSystemConfig() } returns ApiEnvelope(data = SystemConfigDto(mtPhotoTimelineDeferSubfolderThreshold = 600))
        coEvery { preferencesStore.saveCachedSystemConfig(any()) } just runs

        assertEquals(500, repository.loadTimelineThreshold())
        coVerify { preferencesStore.saveCachedSystemConfig(any()) }

        coEvery { systemApiService.getSystemConfig() } returns ApiEnvelope(data = null)
        coEvery { preferencesStore.readCachedSystemConfig() } returns SystemConfigDto(mtPhotoTimelineDeferSubfolderThreshold = 24)
        assertEquals(24, repository.loadTimelineThreshold())

        coEvery { systemApiService.getSystemConfig() } throws IllegalStateException("offline")
        coEvery { preferencesStore.readCachedSystemConfig() } returns SystemConfigDto(mtPhotoTimelineDeferSubfolderThreshold = 0)
        assertEquals(10, repository.loadTimelineThreshold())
    }

    @Test
    fun `import media should validate session and parse success payload`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns null andThen CurrentIdentitySession(
            id = "identity-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery { mtPhotoApiService.importMtPhotoMedia(userId = "identity-1", md5 = "md5-1") } returns buildJsonObject {
            put("state", JsonPrimitive("OK"))
            put("localPath", JsonPrimitive("/upload/mtphoto/a.jpg"))
            put("localFilename", JsonPrimitive("a.jpg"))
            put("dedup", JsonPrimitive("true"))
        }

        val missingSession = repository.importMedia("md5-1")
        val success = repository.importMedia("md5-1")

        assertTrue(missingSession is AppResult.Error)
        assertEquals("请先选择身份后再导入", (missingSession as AppResult.Error).message)
        assertTrue(success is AppResult.Success)
        assertEquals("/upload/mtphoto/a.jpg", (success as AppResult.Success).data.localPath)
        assertEquals(true, success.data.dedup)
    }
}
