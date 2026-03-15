/*
 * 本地数据库用于承接身份、会话、消息与收藏等缓存。
 * 当前结构优先支持跨端能力对齐所需的会话实时更新、消息回显合并与全局收藏基础缓存。
 */
package io.github.a7413498.liao.android.core.database

import android.content.Context
import androidx.room.Dao
import androidx.room.Database
import androidx.room.Entity
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.PrimaryKey
import androidx.room.Query
import androidx.room.Room
import androidx.room.RoomDatabase
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import io.github.a7413498.liao.android.core.common.ChatMessageType
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import io.github.a7413498.liao.android.core.common.GlobalFavoriteItem
import io.github.a7413498.liao.android.core.common.OutgoingMessageStatus
import javax.inject.Singleton
import kotlinx.coroutines.flow.Flow

@Entity(tableName = "identity_cache")
data class IdentityEntity(
    @PrimaryKey val id: String,
    val name: String,
    val sex: String,
    val createdAt: String,
    val lastUsedAt: String,
)

@Entity(tableName = "conversation_cache")
data class ConversationEntity(
    @PrimaryKey val id: String,
    val name: String,
    val sex: String,
    val ip: String,
    val address: String,
    val isFavorite: Boolean,
    val lastMessage: String,
    val lastTime: String,
    val unreadCount: Int,
)

@Entity(tableName = "message_cache")
data class MessageEntity(
    @PrimaryKey val id: String,
    val peerId: String,
    val fromUserId: String,
    val fromUserName: String,
    val toUserId: String,
    val content: String,
    val time: String,
    val isSelf: Boolean,
    val type: String,
    val mediaUrl: String,
    val fileName: String,
)

@Entity(tableName = "favorite_cache")
data class FavoriteEntity(
    @PrimaryKey val id: Int,
    val identityId: String,
    val targetUserId: String,
    val targetUserName: String,
    val createTime: String,
)

@Dao
interface IdentityDao {
    @Query("SELECT * FROM identity_cache ORDER BY lastUsedAt DESC")
    suspend fun getAll(): List<IdentityEntity>

    @Query("SELECT * FROM identity_cache WHERE id = :id LIMIT 1")
    suspend fun getById(id: String): IdentityEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun replaceAll(items: List<IdentityEntity>)
}

@Dao
interface ConversationDao {
    @Query("SELECT * FROM conversation_cache ORDER BY lastTime DESC")
    suspend fun getAll(): List<ConversationEntity>

    @Query("SELECT * FROM conversation_cache ORDER BY lastTime DESC")
    fun observeAll(): Flow<List<ConversationEntity>>

    @Query("SELECT * FROM conversation_cache WHERE id = :id LIMIT 1")
    suspend fun getById(id: String): ConversationEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(items: List<ConversationEntity>)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(item: ConversationEntity)

    @Query("UPDATE conversation_cache SET unreadCount = 0 WHERE id = :id")
    suspend fun markAsRead(id: String)

    @Query("DELETE FROM conversation_cache")
    suspend fun clearAll()
}

@Dao
interface MessageDao {
    @Query("SELECT * FROM message_cache WHERE peerId = :peerId ORDER BY time ASC")
    suspend fun listByPeer(peerId: String): List<MessageEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(items: List<MessageEntity>)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(item: MessageEntity)

    @Query("DELETE FROM message_cache WHERE peerId = :peerId")
    suspend fun clearByPeer(peerId: String)

    @Query("DELETE FROM message_cache")
    suspend fun clearAll()
}

@Dao
interface FavoriteDao {
    @Query("SELECT * FROM favorite_cache ORDER BY createTime DESC, id DESC")
    fun observeAll(): Flow<List<FavoriteEntity>>

    @Query("SELECT * FROM favorite_cache ORDER BY createTime DESC, id DESC")
    suspend fun getAll(): List<FavoriteEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun replaceAll(items: List<FavoriteEntity>)

    @Query("DELETE FROM favorite_cache WHERE id = :id")
    suspend fun deleteById(id: Int)

    @Query("DELETE FROM favorite_cache")
    suspend fun clearAll()
}

@Database(
    entities = [IdentityEntity::class, ConversationEntity::class, MessageEntity::class, FavoriteEntity::class],
    version = 2,
    exportSchema = false,
)
abstract class LiaoDatabase : RoomDatabase() {
    abstract fun identityDao(): IdentityDao
    abstract fun conversationDao(): ConversationDao
    abstract fun messageDao(): MessageDao
    abstract fun favoriteDao(): FavoriteDao
}

fun IdentityEntity.toDisplayName(): String = name

fun ConversationEntity.toPeer(): ChatPeer = ChatPeer(
    id = id,
    name = name,
    sex = sex,
    ip = ip,
    address = address,
    isFavorite = isFavorite,
    lastMessage = lastMessage,
    lastTime = lastTime,
    unreadCount = unreadCount,
)

fun MessageEntity.toTimelineMessage(): ChatTimelineMessage = ChatTimelineMessage(
    id = id,
    fromUserId = fromUserId,
    fromUserName = fromUserName,
    toUserId = toUserId,
    content = content,
    time = time,
    isSelf = isSelf,
    type = runCatching { ChatMessageType.valueOf(type) }.getOrDefault(ChatMessageType.TEXT),
    mediaUrl = mediaUrl,
    fileName = fileName,
    sendStatus = OutgoingMessageStatus.SENT,
)

fun FavoriteEntity.toFavoriteItem(): GlobalFavoriteItem = GlobalFavoriteItem(
    id = id,
    identityId = identityId,
    targetUserId = targetUserId,
    targetUserName = targetUserName,
    createTime = createTime,
)

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {
    @Provides
    @Singleton
    fun provideDatabase(@ApplicationContext context: Context): LiaoDatabase =
        Room.databaseBuilder(context, LiaoDatabase::class.java, "liao_android.db")
            .fallbackToDestructiveMigration()
            .build()

    @Provides
    fun provideIdentityDao(database: LiaoDatabase): IdentityDao = database.identityDao()

    @Provides
    fun provideConversationDao(database: LiaoDatabase): ConversationDao = database.conversationDao()

    @Provides
    fun provideMessageDao(database: LiaoDatabase): MessageDao = database.messageDao()

    @Provides
    fun provideFavoriteDao(database: LiaoDatabase): FavoriteDao = database.favoriteDao()
}
