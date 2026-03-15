package io.github.a7413498.liao.android.feature.douyin

import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.DouyinApiService
import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DouyinRepositoryBranchTest {
    private val douyinApiService = mockk<DouyinApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val repository = DouyinRepository(douyinApiService, preferencesStore, baseUrlProvider)

    init {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test:8443/api/".toHttpUrl()
    }

    @Test
    fun `resolve detail and account should validate input and trim payload fields`() = runTest {
        val detailPayload = slot<JsonElement>()
        val accountPayload = slot<JsonElement>()
        coEvery { douyinApiService.getDetail(capture(detailPayload)) } returns buildJsonObject {
            put("msg", JsonPrimitive("detail-fail"))
        }
        coEvery { douyinApiService.getAccount(capture(accountPayload)) } returns buildJsonObject {
            put("msg", JsonPrimitive("account-fail"))
        }

        val blankDetail = repository.resolveDetail(input = "   ", cookie = "cookie")
        val detailFailure = repository.resolveDetail(input = " share ", cookie = " cookie ")
        val blankAccount = repository.resolveAccount(input = "   ", cookie = "cookie")
        val accountFailure = repository.resolveAccount(input = " user ", cookie = " cookie ")

        assertTrue(blankDetail is AppResult.Error)
        assertEquals("请输入抖音分享文本/链接/作品ID", (blankDetail as AppResult.Error).message)
        assertTrue(detailFailure is AppResult.Error)
        assertEquals("detail-fail", (detailFailure as AppResult.Error).message)
        assertEquals("share", detailPayload.captured.jsonObject["input"]?.jsonPrimitive?.content)
        assertEquals("cookie", detailPayload.captured.jsonObject["cookie"]?.jsonPrimitive?.content)

        assertTrue(blankAccount is AppResult.Error)
        assertEquals("请输入抖音用户主页链接/分享文本/sec_uid", (blankAccount as AppResult.Error).message)
        assertTrue(accountFailure is AppResult.Error)
        assertEquals("account-fail", (accountFailure as AppResult.Error).message)
        assertEquals("user", accountPayload.captured.jsonObject["input"]?.jsonPrimitive?.content)
        assertEquals("post", accountPayload.captured.jsonObject["tab"]?.jsonPrimitive?.content)
        assertEquals("18", accountPayload.captured.jsonObject["count"]?.jsonPrimitive?.content)
        assertEquals("cookie", accountPayload.captured.jsonObject["cookie"]?.jsonPrimitive?.content)
    }

    @Test
    fun `resolve detail account and favorites snapshot should surface structural errors`() = runTest {
        coEvery { douyinApiService.getDetail(any()) } returns JsonPrimitive("oops")
        coEvery { douyinApiService.getAccount(any()) } returns buildJsonObject {
            put("displayName", JsonPrimitive("Alice"))
        }
        coEvery { douyinApiService.listFavoriteUsers() } returns JsonPrimitive("oops")

        val detailResult = repository.resolveDetail(input = "share", cookie = "")
        val accountResult = repository.resolveAccount(input = "user", cookie = "")
        val favoritesResult = repository.refreshFavoritesSnapshot()

        assertTrue(detailResult is AppResult.Error)
        assertEquals("抖音解析响应格式异常", (detailResult as AppResult.Error).message)
        assertTrue(accountResult is AppResult.Error)
        assertEquals("解析结果缺少 secUserId", (accountResult as AppResult.Error).message)
        assertTrue(favoritesResult is AppResult.Error)
        assertEquals("收藏作者列表响应格式异常", (favoritesResult as AppResult.Error).message)
    }

    @Test
    fun `refresh favorites snapshot should surface nested tag errors`() = runTest {
        coEvery { douyinApiService.listFavoriteUsers() } returns buildJsonObject {
            put("items", buildJsonArray { })
        }
        coEvery { douyinApiService.listFavoriteAwemes() } returns buildJsonObject {
            put("items", buildJsonArray { })
        }
        coEvery { douyinApiService.listFavoriteUserTags() } returns buildJsonObject {
            put("error", JsonPrimitive("tag-users-fail"))
        }

        val result = repository.refreshFavoritesSnapshot()

        assertTrue(result is AppResult.Error)
        assertEquals("tag-users-fail", (result as AppResult.Error).message)
    }

    @Test
    fun `import media should validate inputs use pre identity and surface failed states`() = runTest {
        val session = CurrentIdentitySession(
            id = "   ",
            name = "Alice",
            sex = "女",
            cookie = "cookie",
            ip = "1.1.1.1",
            area = "深圳",
        )
        val userIds = mutableListOf<String>()
        val keys = mutableListOf<String>()
        val indexes = mutableListOf<Int>()
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { douyinApiService.importMedia(any(), any(), any()) } answers {
            val userId = invocation.args[0] as String
            val key = invocation.args[1] as String
            val index = invocation.args[2] as Int
            userIds.add(userId)
            keys.add(key)
            indexes.add(index)
            when (index) {
                1 -> buildJsonObject {
                    put("state", JsonPrimitive("FAIL"))
                    put("error", JsonPrimitive("远端失败"))
                }

                else -> buildJsonObject {
                    put("state", JsonPrimitive("OK"))
                    put("localFilename", JsonPrimitive("a.jpg"))
                }
            }
        }

        val blankKey = repository.importMedia(key = "   ", index = 0)
        val negativeIndex = repository.importMedia(key = "key-0", index = -1)
        val failedState = repository.importMedia(key = " key-1 ", index = 1)
        val missingPath = repository.importMedia(key = " key-2 ", index = 2)

        assertTrue(blankKey is AppResult.Error)
        assertEquals("解析信息缺失，请重新解析", (blankKey as AppResult.Error).message)
        assertTrue(negativeIndex is AppResult.Error)
        assertEquals("媒体索引非法", (negativeIndex as AppResult.Error).message)
        assertTrue(failedState is AppResult.Error)
        assertEquals("远端失败", (failedState as AppResult.Error).message)
        assertTrue(missingPath is AppResult.Error)
        assertEquals("导入结果缺少 localPath", (missingPath as AppResult.Error).message)
        assertEquals(listOf("pre_identity", "pre_identity"), userIds)
        assertEquals(listOf("key-1", "key-2"), keys)
        assertEquals(listOf(1, 2), indexes)
    }
}
