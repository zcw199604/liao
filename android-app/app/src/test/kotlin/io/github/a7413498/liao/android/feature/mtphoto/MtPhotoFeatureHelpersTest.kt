package io.github.a7413498.liao.android.feature.mtphoto

import io.github.a7413498.liao.android.core.common.ChatMessageType
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class MtPhotoFeatureHelpersTest {
    @Test
    fun `current visible item count should respect mode and selection`() {
        val album = MtPhotoAlbumSummary(id = 1L, name = "相册", coverMd5 = "", coverUrl = "", count = 1)
        val media = MtPhotoMediaSummary(id = 2L, md5 = "m1", type = ChatMessageType.IMAGE, title = "图", subtitle = "", thumbUrl = "")
        val folder = MtPhotoFolderSummary(id = 3L, name = "目录", path = "/a", coverMd5 = "", coverUrl = "", subFolderNum = 0, subFileNum = 0)

        assertEquals(1, currentVisibleItemCount(MtPhotoUiState(mode = MtPhotoMode.ALBUMS, albums = listOf(album))))
        assertEquals(1, currentVisibleItemCount(MtPhotoUiState(mode = MtPhotoMode.ALBUMS, selectedAlbum = album, albumItems = listOf(media))))
        assertEquals(2, currentVisibleItemCount(MtPhotoUiState(mode = MtPhotoMode.FOLDERS, currentFolders = listOf(folder), folderItems = listOf(media))))
    }

    @Test
    fun `json helpers should parse scalars and defaults`() {
        val root = buildJsonObject {
            put("text", JsonPrimitive("hello"))
            put("blank", JsonPrimitive("   "))
            put("longValue", JsonPrimitive("123"))
            put("intValue", JsonPrimitive("7"))
            put("badInt", JsonPrimitive("oops"))
            put("boolTrue", JsonPrimitive("true"))
            put("boolBad", JsonPrimitive("yes"))
        }

        assertEquals("hello", root.stringOrNull("text"))
        assertNull(root.stringOrNull("blank"))
        assertEquals(123L, root.longOrNull("longValue"))
        assertEquals(7, root.intOrDefault("intValue", 0))
        assertEquals(9, root.intOrDefault("badInt", 9))
        assertEquals(true, root.booleanOrFalse("boolTrue"))
        assertEquals(false, root.booleanOrFalse("boolBad"))
    }

    @Test
    fun `error message should only use msg when payload data absent`() {
        assertEquals(
            "error 优先",
            buildJsonObject {
                put("error", JsonPrimitive("error 优先"))
                put("msg", JsonPrimitive("忽略 msg"))
            }.errorMessage()
        )
        assertEquals(
            "使用 msg",
            buildJsonObject { put("msg", JsonPrimitive("使用 msg")) }.errorMessage()
        )
        assertNull(
            buildJsonObject {
                put("msg", JsonPrimitive("被 data 覆盖"))
                put("data", buildJsonArray { add(JsonPrimitive(1)) })
            }.errorMessage()
        )
        assertNull(
            buildJsonObject {
                put("msg", JsonPrimitive("被 folderList 覆盖"))
                put("folderList", buildJsonArray { add(buildJsonObject { }) })
            }.errorMessage()
        )
    }

    @Test
    fun `folder name and cover helpers should normalize fallback and priority`() {
        assertEquals("cats", " C:\\photos\\cats ".toFolderName("fallback"))
        assertEquals("fallback", " / ".toFolderName("fallback"))

        assertEquals("secondary-md5", firstCoverMd5(cover = "primary-md5", sCover = " secondary-md5 "))
        assertEquals("first-md5", firstCoverMd5(cover = " first-md5 , second-md5 ", sCover = null))
        assertEquals("", firstCoverMd5(cover = "   ", sCover = " "))
    }
}
