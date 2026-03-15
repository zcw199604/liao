package io.github.a7413498.liao.android.feature.douyin

import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DouyinFeatureHelpersTest {
    @Test
    fun `infer douyin item type should detect image video and fallback`() {
        assertEquals("image", inferDouyinItemType(downloadUrl = "https://cdn.test/a.JPG", sourceUrl = "", originalFilename = ""))
        assertEquals("video", inferDouyinItemType(downloadUrl = "", sourceUrl = "https://www.douyin.com/aweme/v1/play/?id=1", originalFilename = ""))
        assertEquals("video", inferDouyinItemType(downloadUrl = "https://cdn.test/file.bin", sourceUrl = "", originalFilename = "unknown"))
    }

    @Test
    fun `resolve douyin media type label should respect live photo album and video`() {
        assertEquals("实况图（2 对）", resolveDouyinMediaTypeLabel(mediaType = "", type = "", imageCount = 0, isLivePhoto = true, livePhotoPairs = 2))
        assertEquals("图集 3 张", resolveDouyinMediaTypeLabel(mediaType = "album", type = "", imageCount = 3, isLivePhoto = false, livePhotoPairs = 0))
        assertEquals("图片", resolveDouyinMediaTypeLabel(mediaType = "image", type = "", imageCount = 1, isLivePhoto = false, livePhotoPairs = 0))
        assertEquals("视频", resolveDouyinMediaTypeLabel(mediaType = "video", type = "", imageCount = 0, isLivePhoto = false, livePhotoPairs = 0))
    }

    @Test
    fun `normalize and resolve favorite media type should cover aliases and count branches`() {
        assertEquals("livePhoto", normalizeDouyinMediaType(" 实况 "))
        assertEquals("imageAlbum", normalizeDouyinMediaType("IMAGE"))
        assertEquals("video", normalizeDouyinMediaType("clip"))

        assertEquals("livePhoto", resolveFavoriteMediaType(type = "video", mediaType = "", isLivePhoto = true, imageCount = 0))
        assertEquals("imageAlbum", resolveFavoriteMediaType(type = "image", mediaType = "album", isLivePhoto = false, imageCount = 4))
        assertEquals("image", resolveFavoriteMediaType(type = "image", mediaType = "image", isLivePhoto = false, imageCount = 1))
        assertEquals("video", resolveFavoriteMediaType(type = "video", mediaType = "", isLivePhoto = false, imageCount = 0))
    }

    @Test
    fun `default file name and import action text should reflect item state`() {
        assertEquals(
            "douyin_1.mp4",
            defaultDouyinFileName(
                DouyinMediaItem(
                    index = 0,
                    type = "video",
                    url = "",
                    downloadUrl = "",
                    originalFilename = "",
                    thumbUrl = "",
                )
            )
        )
        assertEquals(
            "cover.jpg",
            defaultDouyinFileName(
                DouyinMediaItem(
                    index = 1,
                    type = "image",
                    url = "",
                    downloadUrl = "",
                    originalFilename = "cover.jpg",
                    thumbUrl = "",
                )
            )
        )
        assertEquals(
            "still.jpg",
            defaultDouyinFileName(
                DouyinMediaItem(
                    index = 2,
                    type = "image",
                    url = "",
                    downloadUrl = "",
                    originalFilename = "still",
                    thumbUrl = "",
                )
            )
        )

        assertEquals("导入中...", resolveDouyinImportActionText(importing = true, status = null, defaultText = "导入"))
        assertEquals("已存在（去重）", resolveDouyinImportActionText(importing = false, status = DouyinImportStatus.EXISTS, defaultText = "导入"))
        assertEquals("已导入", resolveDouyinImportActionText(importing = false, status = DouyinImportStatus.IMPORTED, defaultText = "导入"))
        assertEquals("导入", resolveDouyinImportActionText(importing = false, status = null, defaultText = "导入"))
    }

    @Test
    fun `json helpers should parse scalar values and defaults`() {
        val root = buildJsonObject {
            put("text", JsonPrimitive("hello"))
            put("blank", JsonPrimitive("   "))
            put("number", JsonPrimitive(42))
            put("badInt", JsonPrimitive("oops"))
            put("validInt", JsonPrimitive("7"))
            put("longValue", JsonPrimitive("922337203685477580"))
            put("doubleValue", JsonPrimitive("1.5"))
            put("boolTrue", JsonPrimitive("true"))
            put("boolBad", JsonPrimitive("yes"))
            put("msg", JsonPrimitive("来自 msg 的错误"))
        }

        assertEquals("hello", root.stringOrNull("text"))
        assertNull(root.stringOrNull("blank"))
        assertEquals("42", root.stringOrNull("number"))
        assertEquals(7, root.intOrDefault("validInt", defaultValue = 0))
        assertEquals(9, root.intOrDefault("badInt", defaultValue = 9))
        assertEquals(7, root.intOrNull("validInt"))
        assertNull(root.intOrNull("badInt"))
        assertEquals(922337203685477580L, root.longOrNull("longValue"))
        assertEquals(1.5, root.doubleOrDefault("doubleValue", defaultValue = 0.0), 0.0001)
        assertEquals(3.2, root.doubleOrDefault("missingDouble", defaultValue = 3.2), 0.0001)
        assertEquals(true, root.booleanOrFalse("boolTrue"))
        assertEquals(false, root.booleanOrFalse("boolBad"))
        assertEquals("来自 msg 的错误", root.errorMessage())
        assertEquals(
            "优先 error",
            buildJsonObject {
                put("error", JsonPrimitive("优先 error"))
                put("msg", JsonPrimitive("忽略 msg"))
            }.errorMessage()
        )
    }

    @Test
    fun `long list should keep only valid numeric entries`() {
        val root = buildJsonObject {
            put(
                "tagIds",
                buildJsonArray {
                    add(JsonPrimitive("1"))
                    add(JsonPrimitive("bad"))
                    add(JsonPrimitive(2))
                    add(buildJsonObject { put("nested", JsonPrimitive("3")) })
                }
            )
        }

        assertEquals(listOf(1L, 2L), root.longList("tagIds"))
        assertEquals(emptyList<Long>(), root.longList("missing"))
    }
}
