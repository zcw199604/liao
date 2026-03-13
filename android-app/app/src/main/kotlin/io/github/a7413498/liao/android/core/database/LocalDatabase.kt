/*
 * 本地数据库用于承接身份、会话与消息的离线缓存骨架。
 * 当前结构先覆盖高频链路，后续再逐步扩展媒体、任务和系统配置表。
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
import io.github.a7413498.liao.android.core.common.ChatPeer
import io.github.a7413498.liao.android.core.common.ChatTimelineMessage
import javax.inject.Singleton

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
)

@Dao
interface IdentityDao {
    @Query("SELECT * FROM identity_cache ORDER BY lastUsedAt DESC")
    suspend fun getAll(): List<IdentityEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun replaceAll(items: List<IdentityEntity>)
}

@Dao
interface ConversationDao {
    @Query("SELECT * FROM conversation_cache ORDER BY lastTime DESC")
    suspend fun getAll(): List<ConversationEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(items: List<ConversationEntity>)
}

@Dao
interface MessageDao {
    @Query("SELECT * FROM message_cache WHERE peerId = :peerId ORDER BY time ASC")
    suspend fun listByPeer(peerId: String): List<MessageEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(items: List<MessageEntity>)
}

@Database(
    entities = [IdentityEntity::class, ConversationEntity::class, MessageEntity::class],
    version = 1,
    exportSchema = false,
)
abstract class LiaoDatabase : RoomDatabase() {
    abstract fun identityDao(): IdentityDao
    abstract fun conversationDao(): ConversationDao
    abstract fun messageDao(): MessageDao
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
)

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {
    @Provides
    @Singleton
    fun provideDatabase(@ApplicationContext context: Context): LiaoDatabase =
        Room.databaseBuilder(context, LiaoDatabase::class.java, "liao_android.db").build()

    @Provides
    fun provideIdentityDao(database: LiaoDatabase): IdentityDao = database.identityDao()

    @Provides
    fun provideConversationDao(database: LiaoDatabase): ConversationDao = database.conversationDao()

    @Provides
    fun provideMessageDao(database: LiaoDatabase): MessageDao = database.messageDao()
}
