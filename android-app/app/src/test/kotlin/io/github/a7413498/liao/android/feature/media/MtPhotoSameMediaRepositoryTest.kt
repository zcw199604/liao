package io.github.a7413498.liao.android.feature.media

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MtPhotoSameMediaRepositoryTest {
    private val mtPhotoApiService = mockk<MtPhotoApiService>()
    private val repository = MtPhotoSameMediaRepository(mtPhotoApiService)
    private val parserJson = Json { ignoreUnknownKeys = true }

    @Test
    fun `query should reject blank local path before calling api`() = runTest {
        val result = repository.queryByLocalPath("   ")

        assertTrue(result is AppResult.Error)
        assertEquals("localPath 不能为空", (result as AppResult.Error).message)
        coVerify(exactly = 0) { mtPhotoApiService.getSameMedia(any(), any()) }
    }

    @Test
    fun `query should reject non object response and top level error`() = runTest {
        coEvery { mtPhotoApiService.getSameMedia(localPath = "/tmp/a.jpg") } returns parserJson.parseToJsonElement("[]")

        val malformed = repository.queryByLocalPath("/tmp/a.jpg")

        assertTrue(malformed is AppResult.Error)
        assertEquals("同媒体响应格式异常", (malformed as AppResult.Error).message)

        coEvery { mtPhotoApiService.getSameMedia(localPath = "/tmp/b.jpg") } returns parserJson.parseToJsonElement(
            """{"error":"上游检索失败"}""",
        )

        val errored = repository.queryByLocalPath("/tmp/b.jpg")

        assertTrue(errored is AppResult.Error)
        assertEquals("上游检索失败", (errored as AppResult.Error).message)
    }

    @Test
    fun `query should keep only valid items and fallback invalid fields to defaults`() = runTest {
        coEvery { mtPhotoApiService.getSameMedia(localPath = "/tmp/c.jpg") } returns parserJson.parseToJsonElement(
            """
            {
              "items": [
                {
                  "id": "12",
                  "md5": "md5-a",
                  "filePath": "/upload/a.jpg",
                  "fileName": "a.jpg",
                  "folderId": "34",
                  "folderPath": "/folder/a",
                  "folderName": "相册A",
                  "day": "2026-03-16",
                  "canOpenFolder": "true"
                },
                {"id":"99","fileName":"missing-md5.jpg"},
                123,
                {
                  "md5": "md5-b",
                  "id": "oops",
                  "folderId": "bad",
                  "canOpenFolder": "not-bool",
                  "filePath": "   "
                }
              ]
            }
            """.trimIndent(),
        )

        val result = repository.queryByLocalPath("/tmp/c.jpg")

        assertTrue(result is AppResult.Success)
        val items = (result as AppResult.Success).data
        assertEquals(2, items.size)
        assertEquals(12L, items[0].id)
        assertEquals("md5-a", items[0].md5)
        assertEquals(34L, items[0].folderId)
        assertTrue(items[0].canOpenFolder)

        assertEquals(0L, items[1].id)
        assertEquals("md5-b", items[1].md5)
        assertEquals("", items[1].filePath)
        assertEquals(0L, items[1].folderId)
        assertFalse(items[1].canOpenFolder)
    }

    @Test
    fun `query should treat non array items field as repository error`() = runTest {
        coEvery { mtPhotoApiService.getSameMedia(localPath = "/tmp/d.jpg") } returns parserJson.parseToJsonElement(
            """{"items":"oops"}""",
        )

        val result = repository.queryByLocalPath("/tmp/d.jpg")

        assertTrue(result is AppResult.Error)
        assertTrue((result as AppResult.Error).message.contains("JsonArray"))
    }
}
