package io.github.a7413498.liao.android.feature.chatlist

import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.ChatUserDto
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import io.mockk.slot
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ChatListRepositoryTest {
    private val chatApiService = mockk<ChatApiService>()
    private val conversationDao = mockk<ConversationDao>(relaxUnitFun = true)
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val repository = ChatListRepository(chatApiService, conversationDao, preferencesStore)

    private val session = CurrentIdentitySession(
        id = "self-1",
        name = "Alice",
        sex = "女",
        cookie = "cookie",
        ip = "1.1.1.1",
    )

    @Test
    fun `observe conversations should return all items for history tab`() = runTest {
        every { conversationDao.observeAll() } returns flowOf(
            listOf(
                ConversationEntity("1", "A", "女", "", "", false, "m1", "t1", 0),
                ConversationEntity("2", "B", "男", "", "", true, "m2", "t2", 1),
            )
        )

        val items = repository.observeConversations(ConversationTab.HISTORY).first()

        assertEquals(listOf("1", "2"), items.map { it.id })
    }

    @Test
    fun `observe conversations should filter favorite tab`() = runTest {
        every { conversationDao.observeAll() } returns flowOf(
            listOf(
                ConversationEntity("1", "A", "女", "", "", false, "", "", 0),
                ConversationEntity("2", "B", "男", "", "", true, "", "", 0),
            )
        )

        val items = repository.observeConversations(ConversationTab.FAVORITE).first()

        assertEquals(listOf("2"), items.map { it.id })
    }

    @Test
    fun `load history should fail when no current session`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns null

        val result = repository.loadHistory()

        assertTrue(result is AppResult.Error)
        assertEquals("请先选择身份", (result as AppResult.Error).message)
    }

    @Test
    fun `load history should filter self and merge cached fields`() = runTest {
        val captured = slot<List<ConversationEntity>>()
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery {
            chatApiService.getHistoryUserList(
                session.id,
                any(),
                any(),
                session.cookie,
                BuildConfig.DEFAULT_REFERER,
                BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns listOf(
            ChatUserDto(id = session.id, nickname = "自己"),
            ChatUserDto(id = "peer-1", nickname = "对端", sex = "", ip = "", address = "", lastMsg = "", lastTime = "", unreadCount = 2),
        )
        coEvery { conversationDao.getById("peer-1") } returns ConversationEntity(
            id = "peer-1",
            name = "旧名",
            sex = "男",
            ip = "2.2.2.2",
            address = "上海",
            isFavorite = true,
            lastMessage = "旧消息",
            lastTime = "旧时间",
            unreadCount = 9,
        )
        coEvery { conversationDao.upsert(capture(captured)) } just runs

        val result = repository.loadHistory()

        assertTrue(result is AppResult.Success)
        val entity = captured.captured.single()
        assertEquals("peer-1", entity.id)
        assertEquals("对端", entity.name)
        assertEquals("男", entity.sex)
        assertEquals("2.2.2.2", entity.ip)
        assertEquals("上海", entity.address)
        assertEquals(true, entity.isFavorite)
        assertEquals("旧消息", entity.lastMessage)
        assertEquals("旧时间", entity.lastTime)
        assertEquals(9, entity.unreadCount)
    }

    @Test
    fun `load history should keep remote values when no cache exists`() = runTest {
        val captured = slot<List<ConversationEntity>>()
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery {
            chatApiService.getHistoryUserList(
                session.id,
                any(),
                any(),
                session.cookie,
                BuildConfig.DEFAULT_REFERER,
                BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns listOf(
            ChatUserDto(
                id = "peer-2",
                nickname = "新会话",
                sex = "女",
                ip = "3.3.3.3",
                address = "杭州",
                isFavorite = false,
                lastMsg = "你好",
                lastTime = "2026-03-15 10:00:00",
                unreadCount = 4,
            ),
        )
        coEvery { conversationDao.getById("peer-2") } returns null
        coEvery { conversationDao.upsert(capture(captured)) } just runs

        val result = repository.loadHistory()

        assertTrue(result is AppResult.Success)
        val entity = captured.captured.single()
        assertEquals("新会话", entity.name)
        assertEquals("女", entity.sex)
        assertEquals("3.3.3.3", entity.ip)
        assertEquals("杭州", entity.address)
        assertEquals(false, entity.isFavorite)
        assertEquals("你好", entity.lastMessage)
        assertEquals("2026-03-15 10:00:00", entity.lastTime)
        assertEquals(4, entity.unreadCount)
    }

    @Test
    fun `load favorite should force favorite flag true`() = runTest {
        val captured = slot<List<ConversationEntity>>()
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery {
            chatApiService.getFavoriteUserList(
                session.id,
                any(),
                any(),
                session.cookie,
                BuildConfig.DEFAULT_REFERER,
                BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns listOf(ChatUserDto(id = "peer-3", nickname = "收藏对象", isFavorite = false))
        coEvery { conversationDao.getById("peer-3") } returns null
        coEvery { conversationDao.upsert(capture(captured)) } just runs

        val result = repository.loadFavorite()

        assertTrue(result is AppResult.Success)
        assertTrue(captured.captured.single().isFavorite)
    }

    @Test
    fun `mark peer read should delegate to dao`() = runTest {
        coEvery { conversationDao.markAsRead("peer-4") } just runs

        repository.markPeerRead("peer-4")

        coVerify { conversationDao.markAsRead("peer-4") }
    }
}
