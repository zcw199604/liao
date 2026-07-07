package io.github.a7413498.liao.android.feature.chatlist

import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.ConversationEntity
import io.github.a7413498.liao.android.core.database.IdentityDao
import io.github.a7413498.liao.android.core.database.IdentityEntity
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ChatApiService
import io.github.a7413498.liao.android.core.network.ChatArchiveSearchItemDto
import io.github.a7413498.liao.android.core.network.ChatArchiveSearchResponseDto
import io.github.a7413498.liao.android.core.network.ChatUserDto
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.ContactCandidateDto
import io.github.a7413498.liao.android.core.network.ContactCandidatesResponseDto
import io.github.a7413498.liao.android.core.network.FavoriteApiService
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
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
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ChatListRepositoryTest {
    private val chatApiService = mockk<ChatApiService>()
    private val conversationDao = mockk<ConversationDao>(relaxUnitFun = true)
    private val messageDao = mockk<MessageDao>(relaxUnitFun = true)
    private val identityApiService = mockk<IdentityApiService>()
    private val identityDao = mockk<IdentityDao>(relaxUnitFun = true)
    private val favoriteApiService = mockk<FavoriteApiService>()
    private val preferencesStore = mockk<AppPreferencesStore>()
    private val webSocketClient = mockk<LiaoWebSocketClient>()
    private val repository = ChatListRepository(chatApiService, conversationDao, messageDao, identityApiService, identityDao, favoriteApiService, preferencesStore, webSocketClient)

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
    fun `search archive should trim keyword and return remote items`() = runTest {
        val item = ChatArchiveSearchItemDto(
            ownerUserId = "owner-1",
            targetUserId = "peer-5",
            nickname = "归档用户",
            lastMsg = "旧消息",
            sources = listOf("archive", "history"),
            localArchived = true,
        )
        coEvery { chatApiService.searchChatArchive("peer", 100) } returns ApiEnvelope(
            code = 0,
            data = ChatArchiveSearchResponseDto(items = listOf(item)),
        )

        val result = repository.searchArchive("  peer  ")

        assertTrue(result is AppResult.Success)
        assertEquals(listOf(item), (result as AppResult.Success).data)
    }

    @Test
    fun `search archive should not call api when keyword blank`() = runTest {
        val result = repository.searchArchive(" ")

        assertTrue(result is AppResult.Success)
        assertEquals(emptyList<ChatArchiveSearchItemDto>(), (result as AppResult.Success).data)
        coVerify(exactly = 0) { chatApiService.searchChatArchive(any(), any()) }
    }

    @Test
    fun `load source identities should filter current identity from cache`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { identityDao.getAll() } returns listOf(
            IdentityEntity(session.id, "Alice", "女", "", ""),
            IdentityEntity("source-1", "Source", "男", "", ""),
        )

        val result = repository.loadSourceIdentities()

        assertTrue(result is AppResult.Success)
        assertEquals(listOf("source-1"), (result as AppResult.Success).data.map { it.id })
        coVerify(exactly = 0) { identityApiService.getIdentityList() }
    }

    @Test
    fun `load source identities should refresh remote when cache empty`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { identityDao.getAll() } returns emptyList()
        coEvery { identityApiService.getIdentityList() } returns ApiEnvelope(
            code = 0,
            data = listOf(
                IdentityDto(session.id, "Alice", "女"),
                IdentityDto("source-2", "Remote", "男", createdAt = "c", lastUsedAt = "l"),
            ),
        )
        coEvery { identityDao.replaceAll(any()) } just runs

        val result = repository.loadSourceIdentities()

        assertTrue(result is AppResult.Success)
        assertEquals(listOf("source-2"), (result as AppResult.Success).data.map { it.id })
        coVerify { identityDao.replaceAll(match { it.any { entity -> entity.id == "source-2" && entity.name == "Remote" } }) }
    }

    @Test
    fun `prepare archived conversation should upsert temporary archived peer`() = runTest {
        val captured = slot<ConversationEntity>()
        val item = ChatArchiveSearchItemDto(
            ownerUserId = "owner-1",
            targetUserId = "peer-6",
            targetUserName = "",
            nickname = "归档昵称",
            sex = "女",
            area = "杭州",
            lastMsg = "",
            lastTime = "",
            sources = listOf("favorite"),
            localArchived = true,
        )
        coEvery { conversationDao.upsert(capture(captured)) } just runs

        val result = repository.prepareArchivedConversation(item)

        assertTrue(result is AppResult.Success)
        val peer = (result as AppResult.Success).data
        assertEquals("peer-6", peer.id)
        assertEquals("归档昵称", peer.name)
        assertEquals(true, peer.isFavorite)
        assertEquals("临时接入", captured.captured.lastMessage)
        assertEquals("刚刚", captured.captured.lastTime)
        assertEquals("杭州", captured.captured.address)
    }

    @Test
    fun `load contact candidates should call api with source identity context`() = runTest {
        val candidate = ContactCandidateDto(
            targetUserId = "peer-7",
            nickname = "跨身份用户",
            sources = listOf("history"),
        )
        coEvery {
            chatApiService.getContactCandidates(
                sourceIdentityId = "source-1",
                includeUpstream = "1",
                query = "",
                limit = 300,
                cookieData = match { it.startsWith("source-1_Source_") },
                referer = BuildConfig.DEFAULT_REFERER,
                userAgent = BuildConfig.DEFAULT_USER_AGENT,
            )
        } returns ApiEnvelope(
            code = 0,
            data = ContactCandidatesResponseDto(sourceIdentityId = "source-1", items = listOf(candidate)),
        )

        val result = repository.loadContactCandidates(sourceIdentity = IdentityDto("source-1", "Source", "男"))

        assertTrue(result is AppResult.Success)
        assertEquals(listOf(candidate), (result as AppResult.Success).data)
    }

    @Test
    fun `prepare contact candidate should upsert temporary peer`() = runTest {
        val captured = slot<ConversationEntity>()
        val candidate = ContactCandidateDto(
            targetUserId = "peer-8",
            name = "候选用户",
            sex = "男",
            address = "广州",
            lastMsg = "历史消息",
            lastTime = "昨天",
            sources = listOf("history", "archive"),
            localArchived = true,
        )
        coEvery { conversationDao.upsert(capture(captured)) } just runs

        val result = repository.prepareContactCandidate(candidate)

        assertTrue(result is AppResult.Success)
        assertEquals("peer-8", (result as AppResult.Success).data.id)
        assertEquals("候选用户", captured.captured.name)
        assertEquals(false, captured.captured.isFavorite)
        assertEquals("历史消息", captured.captured.lastMessage)
        assertEquals("昨天", captured.captured.lastTime)
    }

    @Test
    fun `mark peer read should delegate to dao`() = runTest {
        coEvery { conversationDao.markAsRead("peer-4") } just runs

        repository.markPeerRead("peer-4")

        coVerify { conversationDao.markAsRead("peer-4") }
    }

    @Test
    fun `request online status should send websocket action with current identity`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns session
        every { webSocketClient.sendShowUserLoginInfo(senderId = session.id, targetUserId = "peer-online") } returns true

        val result = repository.requestOnlineStatus(" peer-online ")

        assertTrue(result is AppResult.Success)
        coVerify { preferencesStore.readCurrentSession() }
        io.mockk.verify { webSocketClient.sendShowUserLoginInfo(senderId = session.id, targetUserId = "peer-online") }
    }

    @Test
    fun `request online status should surface missing session and disconnected websocket`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns null

        val missingSession = repository.requestOnlineStatus("peer-online")

        assertTrue(missingSession is AppResult.Error)
        assertEquals("请先选择身份", (missingSession as AppResult.Error).message)

        coEvery { preferencesStore.readCurrentSession() } returns session
        every { webSocketClient.sendShowUserLoginInfo(senderId = session.id, targetUserId = "peer-online") } returns false

        val disconnected = repository.requestOnlineStatus("peer-online")

        assertTrue(disconnected is AppResult.Error)
        assertEquals("WebSocket 未连接，暂时无法查询在线状态", (disconnected as AppResult.Error).message)
    }

    @Test
    fun `load global favorite target ids should filter current identity`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { favoriteApiService.listAllFavorites() } returns ApiEnvelope(
            code = 0,
            data = listOf(
                Json.parseToJsonElement("""{"id":"1","identityId":"self-1","targetUserId":"peer-global","targetUserName":"全局用户","createTime":"2026"}"""),
                Json.parseToJsonElement("""{"id":"2","identityId":"other","targetUserId":"peer-other","targetUserName":"其它身份","createTime":"2026"}"""),
                Json.parseToJsonElement("""{"id":"bad","identityId":"self-1","targetUserId":"bad"}"""),
            ),
        )

        val result = repository.loadGlobalFavoriteTargetIds()

        assertTrue(result is AppResult.Success)
        assertEquals(setOf("peer-global"), (result as AppResult.Success).data)
    }

    @Test
    fun `toggle global favorite should add when peer is not favorite`() = runTest {
        val peer = io.github.a7413498.liao.android.core.common.ChatPeer(
            id = "peer-global",
            name = "全局用户",
            sex = "",
            ip = "",
            address = "",
            isFavorite = false,
            lastMessage = "",
            lastTime = "",
            unreadCount = 0,
        )
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { favoriteApiService.addFavorite("self-1", "peer-global", "全局用户") } returns ApiEnvelope<kotlinx.serialization.json.JsonElement>(code = 0)

        val result = repository.toggleGlobalFavorite(peer, isGlobalFavorite = false)

        assertTrue(result is AppResult.Success)
        assertEquals(true, (result as AppResult.Success).data)
        coVerify { favoriteApiService.addFavorite("self-1", "peer-global", "全局用户") }
        coVerify(exactly = 0) { favoriteApiService.removeFavorite(any(), any()) }
    }

    @Test
    fun `toggle global favorite should remove when peer is already favorite`() = runTest {
        val peer = io.github.a7413498.liao.android.core.common.ChatPeer(
            id = "peer-global",
            name = "全局用户",
            sex = "",
            ip = "",
            address = "",
            isFavorite = false,
            lastMessage = "",
            lastTime = "",
            unreadCount = 0,
        )
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { favoriteApiService.removeFavorite("self-1", "peer-global") } returns ApiEnvelope<Unit>(code = 0)

        val result = repository.toggleGlobalFavorite(peer, isGlobalFavorite = true)

        assertTrue(result is AppResult.Success)
        assertEquals(false, (result as AppResult.Success).data)
        coVerify { favoriteApiService.removeFavorite("self-1", "peer-global") }
        coVerify(exactly = 0) { favoriteApiService.addFavorite(any(), any(), any()) }
    }

    @Test
    fun `toggle global favorite should return error when remote fails`() = runTest {
        val peer = io.github.a7413498.liao.android.core.common.ChatPeer(
            id = "peer-global",
            name = "全局用户",
            sex = "",
            ip = "",
            address = "",
            isFavorite = false,
            lastMessage = "",
            lastTime = "",
            unreadCount = 0,
        )
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { favoriteApiService.addFavorite("self-1", "peer-global", "全局用户") } returns ApiEnvelope<kotlinx.serialization.json.JsonElement>(code = 1, msg = "添加失败")

        val result = repository.toggleGlobalFavorite(peer, isGlobalFavorite = false)

        assertTrue(result is AppResult.Error)
        assertEquals("添加失败", (result as AppResult.Error).message)
    }

    @Test
    fun `delete peer should call remote then remove local conversation and messages`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { chatApiService.deleteUpstreamUser(session.id, "peer-delete") } returns ApiEnvelope<kotlinx.serialization.json.JsonElement>(code = 0)
        coEvery { conversationDao.deleteById("peer-delete") } just runs
        coEvery { messageDao.clearByPeer("peer-delete") } just runs

        val result = repository.deletePeer("peer-delete")

        assertTrue(result is AppResult.Success)
        coVerify { chatApiService.deleteUpstreamUser(session.id, "peer-delete") }
        coVerify { conversationDao.deleteById("peer-delete") }
        coVerify { messageDao.clearByPeer("peer-delete") }
    }

    @Test
    fun `delete peer should keep local cache when remote fails`() = runTest {
        coEvery { preferencesStore.readCurrentSession() } returns session
        coEvery { chatApiService.deleteUpstreamUser(session.id, "peer-delete") } returns ApiEnvelope<kotlinx.serialization.json.JsonElement>(code = 1, msg = "删除失败")

        val result = repository.deletePeer("peer-delete")

        assertTrue(result is AppResult.Error)
        assertEquals("删除失败", (result as AppResult.Error).message)
        coVerify(exactly = 0) { conversationDao.deleteById(any()) }
        coVerify(exactly = 0) { messageDao.clearByPeer(any()) }
    }
}
