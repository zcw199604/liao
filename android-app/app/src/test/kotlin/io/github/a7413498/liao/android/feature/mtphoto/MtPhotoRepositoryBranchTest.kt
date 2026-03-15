package io.github.a7413498.liao.android.feature.mtphoto

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.MtPhotoApiService
import io.github.a7413498.liao.android.core.network.SystemApiService
import io.github.a7413498.liao.android.core.network.SystemConfigDto
import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class MtPhotoRepositoryBranchTest {
    private val mtPhotoApiService = mockk<MtPhotoApiService>()
    private val systemApiService = mockk<SystemApiService>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val preferencesStore = mockk<AppPreferencesStore>(relaxUnitFun = true)
    private val repository = MtPhotoRepository(mtPhotoApiService, systemApiService, baseUrlProvider, preferencesStore)

    init {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()
    }

    @Test
    fun `load methods should surface format remote and fallback errors`() = runTest {
        coEvery { mtPhotoApiService.getAlbums() } returns JsonPrimitive("oops")
        coEvery { mtPhotoApiService.getAlbumFiles(albumId = 3L, page = 4, pageSize = 10) } returns buildJsonObject {
            put("msg", JsonPrimitive("album-files-fail"))
        }
        coEvery { mtPhotoApiService.getFolderRoot() } returns buildJsonObject {
            put("error", JsonPrimitive("root-fail"))
        }
        coEvery { mtPhotoApiService.getFolderContent(folderId = 9L, page = 2, pageSize = 10, includeTimeline = true) } throws RuntimeException()
        coEvery { mtPhotoApiService.getFolderContent(folderId = 9L, page = 3, pageSize = 10, includeTimeline = false) } throws RuntimeException()

        val albums = repository.loadAlbums()
        val albumFiles = repository.loadAlbumFiles(albumId = 3L, page = 4, pageSize = 10)
        val root = repository.loadFolderRoot()
        val timeline = repository.loadFolderContent(folderId = 9L, page = 2, includeTimeline = true, pageSize = 10)
        val folder = repository.loadFolderContent(folderId = 9L, page = 3, includeTimeline = false, pageSize = 10)

        assertTrue(albums is AppResult.Error)
        assertEquals("mtPhoto 相册响应格式异常", (albums as AppResult.Error).message)
        assertTrue(albumFiles is AppResult.Error)
        assertEquals("album-files-fail", (albumFiles as AppResult.Error).message)
        assertTrue(root is AppResult.Error)
        assertEquals("root-fail", (root as AppResult.Error).message)
        assertTrue(timeline is AppResult.Error)
        assertEquals("加载 mtPhoto 时间线失败", (timeline as AppResult.Error).message)
        assertTrue(folder is AppResult.Error)
        assertEquals("加载 mtPhoto 目录失败", (folder as AppResult.Error).message)
    }

    @Test
    fun `load timeline threshold should clamp cached value on failures`() = runTest {
        coEvery { systemApiService.getSystemConfig() } returns ApiEnvelope(data = null) andThenThrows RuntimeException()
        coEvery { preferencesStore.readCachedSystemConfig() } returns SystemConfigDto(mtPhotoTimelineDeferSubfolderThreshold = 999)

        val remoteEmpty = repository.loadTimelineThreshold()
        val remoteFailure = repository.loadTimelineThreshold()

        assertEquals(500, remoteEmpty)
        assertEquals(500, remoteFailure)
    }

    @Test
    fun `import media should surface non json failed state and missing path`() = runTest {
        val session = CurrentIdentitySession(
            id = "identity-1",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { mtPhotoApiService.importMtPhotoMedia(userId = "identity-1", md5 = "md5-format") } returns JsonPrimitive("oops")
        coEvery { mtPhotoApiService.importMtPhotoMedia(userId = "identity-1", md5 = "md5-fail") } returns buildJsonObject {
            put("state", JsonPrimitive("FAIL"))
            put("msg", JsonPrimitive("remote-fail"))
        }
        coEvery { mtPhotoApiService.importMtPhotoMedia(userId = "identity-1", md5 = "md5-missing") } returns buildJsonObject {
            put("state", JsonPrimitive("OK"))
            put("localFilename", JsonPrimitive("a.jpg"))
        }

        val formatResult = repository.importMedia("md5-format")
        val failResult = repository.importMedia("md5-fail")
        val missingPathResult = repository.importMedia("md5-missing")

        assertTrue(formatResult is AppResult.Error)
        assertEquals("mtPhoto 导入响应格式异常", (formatResult as AppResult.Error).message)
        assertTrue(failResult is AppResult.Error)
        assertEquals("remote-fail", (failResult as AppResult.Error).message)
        assertTrue(missingPathResult is AppResult.Error)
        assertEquals("导入结果缺少 localPath", (missingPathResult as AppResult.Error).message)
    }
}
