package io.github.a7413498.liao.android.feature.mtphoto

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import io.mockk.coEvery
import io.mockk.mockk
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MtPhotoFolderFavoritesRepositoryTest {
    private val mtPhotoApiService = mockk<MtPhotoApiService>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val repository = MtPhotoFolderFavoritesRepository(mtPhotoApiService, baseUrlProvider)

    init {
        io.mockk.every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()
    }

    @Test
    fun `load favorites should parse items apply defaults and skip invalid entries`() = runTest {
        coEvery { mtPhotoApiService.getFolderFavorites() } returns buildJsonObject {
            put(
                "items",
                buildJsonArray {
                    add(
                        buildJsonObject {
                            put("id", JsonPrimitive(7))
                            put("folderId", JsonPrimitive("5"))
                            put("folderPath", JsonPrimitive("/root/a"))
                            put("coverMd5", JsonPrimitive("cover-1"))
                            put(
                                "tags",
                                buildJsonArray {
                                    add(JsonPrimitive(" tag1 "))
                                    add(JsonPrimitive(" "))
                                    add(JsonPrimitive(123))
                                }
                            )
                            put("note", JsonPrimitive("备注"))
                            put("updateTime", JsonPrimitive("2026-03-15"))
                        }
                    )
                    add(buildJsonObject { put("folderName", JsonPrimitive("缺失 folderId")) })
                    add(
                        buildJsonObject {
                            put("folderId", JsonPrimitive("8"))
                            put("folderName", JsonPrimitive("   "))
                        }
                    )
                }
            )
        }

        val result = repository.loadFavorites()

        assertTrue(result is AppResult.Success)
        val payload = (result as AppResult.Success).data
        assertEquals(2, payload.size)
        assertEquals(7L, payload[0].id)
        assertEquals(5L, payload[0].folderId)
        assertEquals("目录 5", payload[0].folderName)
        assertEquals(listOf("tag1", "123"), payload[0].tags)
        assertEquals("https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=cover-1", payload[0].coverUrl)
        assertEquals(8L, payload[1].id)
        assertEquals("目录 8", payload[1].folderName)
        assertEquals("", payload[1].coverUrl)
    }

    @Test
    fun `load favorites should surface error field`() = runTest {
        coEvery { mtPhotoApiService.getFolderFavorites() } returns buildJsonObject {
            put("error", JsonPrimitive("服务异常"))
        }

        val result = repository.loadFavorites()

        assertTrue(result is AppResult.Error)
        assertEquals("服务异常", (result as AppResult.Error).message)
    }

    @Test
    fun `upsert favorite should include cover md5 in payload and parse item`() = runTest {
        val payloadSlot = slot<JsonElement>()
        coEvery { mtPhotoApiService.upsertFolderFavorite(capture(payloadSlot)) } returns buildJsonObject {
            put("success", JsonPrimitive(true))
            put(
                "item",
                buildJsonObject {
                    put("id", JsonPrimitive(11))
                    put("folderId", JsonPrimitive(11))
                    put("folderName", JsonPrimitive("目录 11"))
                    put("folderPath", JsonPrimitive("/root/11"))
                    put("coverMd5", JsonPrimitive("cover-11"))
                }
            )
        }

        val result = repository.upsertFavorite(folderId = 11L, folderName = "目录 11", folderPath = "/root/11", coverMd5 = "cover-11")

        assertTrue(result is AppResult.Success)
        val payload = payloadSlot.captured.jsonObject
        assertEquals("11", payload["folderId"]?.jsonPrimitive?.content)
        assertEquals("cover-11", payload["coverMd5"]?.jsonPrimitive?.content)
        assertTrue(payload["tags"]?.jsonArray?.isEmpty() == true)
        assertEquals("", payload["note"]?.jsonPrimitive?.content)
        assertEquals(
            "https://demo.test:8443/api/getMtPhotoThumb?size=s260&md5=cover-11",
            (result as AppResult.Success).data.coverUrl
        )
    }

    @Test
    fun `upsert favorite should omit blank cover md5 and return message when save fails`() = runTest {
        val payloadSlot = slot<JsonElement>()
        coEvery { mtPhotoApiService.upsertFolderFavorite(capture(payloadSlot)) } returns buildJsonObject {
            put("success", JsonPrimitive(false))
            put("message", JsonPrimitive("保存失败"))
        }

        val result = repository.upsertFavorite(folderId = 12L, folderName = "目录 12", folderPath = "/root/12", coverMd5 = "")

        assertTrue(result is AppResult.Error)
        assertEquals("保存失败", (result as AppResult.Error).message)
        assertFalse(payloadSlot.captured.jsonObject.containsKey("coverMd5"))
    }

    @Test
    fun `remove favorite should send folder id payload on success`() = runTest {
        val payloadSlot = slot<JsonElement>()
        coEvery { mtPhotoApiService.removeFolderFavorite(capture(payloadSlot)) } returns buildJsonObject {
            put("success", JsonPrimitive(true))
        }

        val result = repository.removeFavorite(folderId = 21L)

        assertTrue(result is AppResult.Success)
        assertEquals("21", payloadSlot.captured.jsonObject["folderId"]?.jsonPrimitive?.content)
    }

    @Test
    fun `remove favorite should return default message when response marks failure`() = runTest {
        coEvery { mtPhotoApiService.removeFolderFavorite(any()) } returns buildJsonObject {
            put("success", JsonPrimitive(false))
        }

        val result = repository.removeFavorite(folderId = 22L)

        assertTrue(result is AppResult.Error)
        assertEquals("移除目录收藏失败", (result as AppResult.Error).message)
    }
}
