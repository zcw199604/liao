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

class DouyinRepositoryOperationsTest {
    private val douyinApiService = mockk<DouyinApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val baseUrlProvider = mockk<BaseUrlProvider>()
    private val repository = DouyinRepository(douyinApiService, preferencesStore, baseUrlProvider)

    init {
        every { baseUrlProvider.currentApiBaseUrl() } returns "https://demo.test/api/".toHttpUrl()
    }

    @Test
    fun `upsert favorite user should build payload and parse response`() = runTest {
        val payloadSlot = slot<JsonElement>()
        coEvery { douyinApiService.addFavoriteUser(capture(payloadSlot)) } returns buildJsonObject {
            put("secUserId", JsonPrimitive("sec-1"))
            put("displayName", JsonPrimitive("Alice"))
            put("avatarUrl", JsonPrimitive("/avatar.jpg"))
            put(
                "tagIds",
                buildJsonArray {
                    add(JsonPrimitive("1"))
                    add(JsonPrimitive("2"))
                }
            )
        }

        val result = repository.upsertFavoriteUser(
            accountInput = " https://douyin.test/user/sec-1 ",
            result = DouyinAccountResult(
                secUserId = "sec-1",
                displayName = "Alice",
                signature = "签名",
                avatarUrl = "https://cdn.test/avatar.jpg",
                profileUrl = "https://douyin.test/profile/sec-1",
                followerCount = 10,
                followingCount = 5,
                awemeCount = 3,
                totalFavorited = 99,
                items = listOf(
                    DouyinAccountItem(
                        detailId = "aweme-1",
                        type = "video",
                        mediaType = "video",
                        desc = "作品",
                        coverUrl = "",
                        imageCount = 0,
                        videoDuration = 0.0,
                        isLivePhoto = false,
                        livePhotoPairs = 0,
                        isPinned = false,
                        pinnedRank = null,
                        publishAt = "",
                        status = "",
                        authorUniqueId = "",
                        authorName = "",
                    )
                ),
            )
        )

        assertTrue(result is AppResult.Success)
        val payload = payloadSlot.captured.jsonObject
        assertEquals("sec-1", payload["secUserId"]?.jsonPrimitive?.content)
        assertEquals("https://douyin.test/user/sec-1", payload["sourceInput"]?.jsonPrimitive?.content)
        assertEquals("Alice", payload["displayName"]?.jsonPrimitive?.content)
        assertEquals("https://cdn.test/avatar.jpg", payload["avatarUrl"]?.jsonPrimitive?.content)
        assertEquals("https://douyin.test/profile/sec-1", payload["profileUrl"]?.jsonPrimitive?.content)
        assertEquals("1", payload["lastParsedCount"]?.jsonPrimitive?.content)
        val raw = payload["lastParsedRaw"]?.jsonObject ?: error("missing lastParsedRaw")
        assertEquals("签名", raw["signature"]?.jsonPrimitive?.content)
        assertEquals("10", raw["followerCount"]?.jsonPrimitive?.content)
        assertEquals(listOf(1L, 2L), (result as AppResult.Success).data.tagIds)
        assertEquals("https://demo.test/avatar.jpg", result.data.avatarUrl)
    }

    @Test
    fun `favorite aweme methods should build payload from detail and account`() = runTest {
        val payloads = mutableListOf<JsonElement>()
        coEvery { douyinApiService.addFavoriteAweme(any()) } answers {
            payloads.add(invocation.args[0] as JsonElement)
            if (payloads.size == 1) {
                buildJsonObject {
                    put("awemeId", JsonPrimitive("detail-1"))
                    put("type", JsonPrimitive("image"))
                }
            } else {
                buildJsonObject {
                    put("awemeId", JsonPrimitive("detail-2"))
                    put("type", JsonPrimitive("video"))
                    put("secUserId", JsonPrimitive("sec-2"))
                }
            }
        }

        val detailResult = repository.upsertFavoriteAwemeFromDetail(
            DouyinDetailResult(
                key = "key-1",
                detailId = " detail-1 ",
                type = "image",
                mediaType = "image",
                title = "图文",
                coverUrl = "https://demo.test/cover.jpg",
                imageCount = 1,
                videoDuration = 0.0,
                isLivePhoto = false,
                livePhotoPairs = 0,
                items = emptyList(),
            )
        )
        val accountResult = repository.upsertFavoriteAwemeFromAccount(
            secUserId = "sec-2",
            item = DouyinAccountItem(
                detailId = "detail-2",
                type = "video",
                mediaType = "video",
                desc = "视频",
                coverUrl = "https://demo.test/video-cover.jpg",
                imageCount = 0,
                videoDuration = 12.0,
                isLivePhoto = false,
                livePhotoPairs = 0,
                isPinned = false,
                pinnedRank = null,
                publishAt = "",
                status = "",
                authorUniqueId = "",
                authorName = "",
            ),
        )

        assertTrue(detailResult is AppResult.Success)
        assertTrue(accountResult is AppResult.Success)
        assertEquals("detail-1", payloads[0].jsonObject["awemeId"]?.jsonPrimitive?.content)
        assertEquals("image", payloads[0].jsonObject["type"]?.jsonPrimitive?.content)
        assertEquals("图文", payloads[0].jsonObject["desc"]?.jsonPrimitive?.content)
        assertEquals("sec-2", payloads[1].jsonObject["secUserId"]?.jsonPrimitive?.content)
        assertEquals("video", payloads[1].jsonObject["type"]?.jsonPrimitive?.content)
        assertEquals("视频", payloads[1].jsonObject["desc"]?.jsonPrimitive?.content)
    }

    @Test
    fun `remove favorite methods should trim ids and reject blank values`() = runTest {
        val userPayloadSlot = slot<JsonElement>()
        val awemePayloadSlot = slot<JsonElement>()
        coEvery { douyinApiService.removeFavoriteUser(capture(userPayloadSlot)) } returns buildJsonObject {}
        coEvery { douyinApiService.removeFavoriteAweme(capture(awemePayloadSlot)) } returns buildJsonObject {}

        val userResult = repository.removeFavoriteUser(" sec-3 ")
        val awemeResult = repository.removeFavoriteAweme(" aweme-3 ")
        val blankUserResult = repository.removeFavoriteUser("   ")
        val blankAwemeResult = repository.removeFavoriteAweme("   ")

        assertTrue(userResult is AppResult.Success)
        assertTrue(awemeResult is AppResult.Success)
        assertEquals("sec-3", userPayloadSlot.captured.jsonObject["secUserId"]?.jsonPrimitive?.content)
        assertEquals("aweme-3", awemePayloadSlot.captured.jsonObject["awemeId"]?.jsonPrimitive?.content)
        assertTrue(blankUserResult is AppResult.Error)
        assertEquals("secUserId 不能为空", (blankUserResult as AppResult.Error).message)
        assertTrue(blankAwemeResult is AppResult.Error)
        assertEquals("awemeId 不能为空", (blankAwemeResult as AppResult.Error).message)
    }

    @Test
    fun `tag operations should route by kind and normalize payload`() = runTest {
        val createUserTagSlot = slot<JsonElement>()
        val createAwemeTagSlot = slot<JsonElement>()
        val removeUserTagSlot = slot<JsonElement>()
        val removeAwemeTagSlot = slot<JsonElement>()
        val applyUserTagsSlot = slot<JsonElement>()
        val applyAwemeTagsSlot = slot<JsonElement>()
        coEvery { douyinApiService.addFavoriteUserTag(capture(createUserTagSlot)) } returns buildJsonObject {
            put("id", JsonPrimitive(1))
            put("name", JsonPrimitive("人物"))
        }
        coEvery { douyinApiService.addFavoriteAwemeTag(capture(createAwemeTagSlot)) } returns buildJsonObject {
            put("id", JsonPrimitive(2))
            put("name", JsonPrimitive("作品"))
        }
        coEvery { douyinApiService.removeFavoriteUserTag(capture(removeUserTagSlot)) } returns buildJsonObject {}
        coEvery { douyinApiService.removeFavoriteAwemeTag(capture(removeAwemeTagSlot)) } returns buildJsonObject {}
        coEvery { douyinApiService.applyFavoriteUserTags(capture(applyUserTagsSlot)) } returns buildJsonObject {}
        coEvery { douyinApiService.applyFavoriteAwemeTags(capture(applyAwemeTagsSlot)) } returns buildJsonObject {}

        val createUser = repository.createTag(DouyinTagKind.USERS, " 人物 ")
        val createAweme = repository.createTag(DouyinTagKind.AWEMES, " 作品 ")
        val removeUser = repository.removeTag(DouyinTagKind.USERS, 7)
        val removeAweme = repository.removeTag(DouyinTagKind.AWEMES, 8)
        val applyUsers = repository.applyTags(DouyinTagKind.USERS, " sec-9 ", listOf(1, 1, -1, 2))
        val applyAwemes = repository.applyTags(DouyinTagKind.AWEMES, " aweme-9 ", listOf(3, 0, 3, 4))
        val invalidTag = repository.removeTag(DouyinTagKind.USERS, 0)
        val blankTarget = repository.applyTags(DouyinTagKind.AWEMES, "   ", listOf(1))

        assertTrue(createUser is AppResult.Success)
        assertTrue(createAweme is AppResult.Success)
        assertTrue(removeUser is AppResult.Success)
        assertTrue(removeAweme is AppResult.Success)
        assertTrue(applyUsers is AppResult.Success)
        assertTrue(applyAwemes is AppResult.Success)
        assertEquals("人物", createUserTagSlot.captured.jsonObject["name"]?.jsonPrimitive?.content)
        assertEquals("作品", createAwemeTagSlot.captured.jsonObject["name"]?.jsonPrimitive?.content)
        assertEquals("7", removeUserTagSlot.captured.jsonObject["id"]?.jsonPrimitive?.content)
        assertEquals("8", removeAwemeTagSlot.captured.jsonObject["id"]?.jsonPrimitive?.content)
        assertEquals(listOf("sec-9"), applyUserTagsSlot.captured.jsonObject["secUserIds"]?.jsonArray?.map { it.jsonPrimitive.content })
        assertEquals(listOf("1", "2"), applyUserTagsSlot.captured.jsonObject["tagIds"]?.jsonArray?.map { it.jsonPrimitive.content })
        assertEquals(listOf("aweme-9"), applyAwemeTagsSlot.captured.jsonObject["awemeIds"]?.jsonArray?.map { it.jsonPrimitive.content })
        assertEquals(listOf("3", "4"), applyAwemeTagsSlot.captured.jsonObject["tagIds"]?.jsonArray?.map { it.jsonPrimitive.content })
        assertTrue(invalidTag is AppResult.Error)
        assertEquals("标签 ID 非法", (invalidTag as AppResult.Error).message)
        assertTrue(blankTarget is AppResult.Error)
        assertEquals("目标 ID 不能为空", (blankTarget as AppResult.Error).message)
    }


    @Test
    fun `favorite operations should surface invalid responses and remote errors`() = runTest {
        coEvery { douyinApiService.addFavoriteUser(any()) } returns JsonPrimitive("oops")
        coEvery { douyinApiService.addFavoriteAweme(any()) } returns buildJsonObject {
            put("error", JsonPrimitive("aweme-fail"))
        }
        coEvery { douyinApiService.removeFavoriteUser(any()) } returns buildJsonObject {
            put("error", JsonPrimitive("remove-user-fail"))
        }
        coEvery { douyinApiService.removeFavoriteAweme(any()) } returns buildJsonObject {
            put("error", JsonPrimitive("remove-aweme-fail"))
        }
        coEvery { douyinApiService.addFavoriteUserTag(any()) } returns JsonPrimitive("oops")
        coEvery { douyinApiService.removeFavoriteAwemeTag(any()) } returns buildJsonObject {
            put("error", JsonPrimitive("remove-tag-fail"))
        }
        val applyPayloads = mutableListOf<JsonElement>()
        coEvery { douyinApiService.applyFavoriteUserTags(any()) } answers {
            applyPayloads.add(invocation.args[0] as JsonElement)
            buildJsonObject {
                put("msg", JsonPrimitive("apply-tag-fail"))
            }
        }

        val upsertUser = repository.upsertFavoriteUser(
            accountInput = "",
            result = DouyinAccountResult(
                secUserId = "sec-err",
                displayName = "",
                signature = "",
                avatarUrl = "",
                profileUrl = "",
                followerCount = null,
                followingCount = null,
                awemeCount = null,
                totalFavorited = null,
                items = emptyList(),
            ),
        )
        val upsertDetail = repository.upsertFavoriteAwemeFromDetail(
            DouyinDetailResult(
                key = "key-1",
                detailId = "detail-1",
                type = "video",
                mediaType = "video",
                title = "",
                coverUrl = "",
                imageCount = 0,
                videoDuration = 0.0,
                isLivePhoto = false,
                livePhotoPairs = 0,
                items = emptyList(),
            )
        )
        val blankAccountAweme = repository.upsertFavoriteAwemeFromAccount(
            secUserId = "sec-1",
            item = DouyinAccountItem(
                detailId = "   ",
                type = "video",
                mediaType = "video",
                desc = "",
                coverUrl = "",
                imageCount = 0,
                videoDuration = 0.0,
                isLivePhoto = false,
                livePhotoPairs = 0,
                isPinned = false,
                pinnedRank = null,
                publishAt = "",
                status = "",
                authorUniqueId = "",
                authorName = "",
            ),
        )
        val removeUser = repository.removeFavoriteUser("sec-1")
        val removeAweme = repository.removeFavoriteAweme("aweme-1")
        val blankTagName = repository.createTag(DouyinTagKind.USERS, "   ")
        val createTag = repository.createTag(DouyinTagKind.USERS, "人物")
        val removeTag = repository.removeTag(DouyinTagKind.AWEMES, 9)
        val applyTags = repository.applyTags(DouyinTagKind.USERS, "sec-1", listOf(-1, 0))

        assertTrue(upsertUser is AppResult.Error)
        assertEquals("收藏作者响应格式异常", (upsertUser as AppResult.Error).message)
        assertTrue(upsertDetail is AppResult.Error)
        assertEquals("aweme-fail", (upsertDetail as AppResult.Error).message)
        assertTrue(blankAccountAweme is AppResult.Error)
        assertEquals("当前作品缺少 awemeId", (blankAccountAweme as AppResult.Error).message)
        assertTrue(removeUser is AppResult.Error)
        assertEquals("remove-user-fail", (removeUser as AppResult.Error).message)
        assertTrue(removeAweme is AppResult.Error)
        assertEquals("remove-aweme-fail", (removeAweme as AppResult.Error).message)
        assertTrue(blankTagName is AppResult.Error)
        assertEquals("标签名称不能为空", (blankTagName as AppResult.Error).message)
        assertTrue(createTag is AppResult.Error)
        assertEquals("标签保存响应格式异常", (createTag as AppResult.Error).message)
        assertTrue(removeTag is AppResult.Error)
        assertEquals("remove-tag-fail", (removeTag as AppResult.Error).message)
        assertTrue(applyTags is AppResult.Error)
        assertEquals("apply-tag-fail", (applyTags as AppResult.Error).message)
        assertEquals(emptyList<String>(), applyPayloads.single().jsonObject["tagIds"]?.jsonArray?.map { it.jsonPrimitive.content })
    }

}
