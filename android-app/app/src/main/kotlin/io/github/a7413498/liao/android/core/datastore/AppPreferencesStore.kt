/*
 * DataStore 负责保存 Base URL、主题偏好与离线快照；敏感认证态改走加密首选项。
 * 这些数据会驱动登录恢复、动态联调地址、主题偏好与关键页面的离线兜底流程。
 */
package io.github.a7413498.liao.android.core.datastore

import android.content.Context
import android.content.SharedPreferences
import androidx.datastore.preferences.core.MutablePreferences
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStoreFile
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import dagger.hilt.android.qualifiers.ApplicationContext
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.app.theme.LiaoThemePreference
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import io.github.a7413498.liao.android.core.common.generateCookie
import io.github.a7413498.liao.android.core.common.generateRandomIp
import io.github.a7413498.liao.android.core.network.SystemConfigDto
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.KSerializer
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private const val MAX_CACHED_VIDEO_EXTRACT_TASK_DETAILS = 8
private const val SECURE_PREFS_NAME = "liao_android_secure_prefs"
private const val SECURE_AUTH_TOKEN_KEY = "auth_token"
private const val SECURE_CURRENT_SESSION_KEY = "current_session"

@Serializable
private data class StoredSession(
    val id: String,
    val name: String,
    val sex: String,
    val cookie: String = "",
    val ip: String = "",
    val area: String = "",
)

@Singleton
class AppPreferencesStore @Inject constructor(
    @ApplicationContext context: Context,
    private val json: Json,
) {
    private val dataStore = PreferenceDataStoreFactory.create {
        context.preferencesDataStoreFile(name = "liao_android_prefs")
    }

    private val securePreferences: SharedPreferences = EncryptedSharedPreferences.create(
        context,
        SECURE_PREFS_NAME,
        MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    private val authTokenKey = stringPreferencesKey("auth_token")
    private val baseUrlKey = stringPreferencesKey("base_url")
    private val currentSessionKey = stringPreferencesKey("current_session")
    private val themePreferenceKey = stringPreferencesKey("theme_preference")
    private val cachedSystemConfigKey = stringPreferencesKey("cached_system_config")
    private val cachedMediaLibraryKey = stringPreferencesKey("cached_media_library")
    private val cachedVideoExtractTaskListKey = stringPreferencesKey("cached_video_extract_task_list")
    private val cachedVideoExtractTaskDetailsKey = stringPreferencesKey("cached_video_extract_task_details")

    private val authTokenState = MutableStateFlow<String?>(null)
    private val currentSessionState = MutableStateFlow<CurrentIdentitySession?>(null)

    init {
        migrateLegacySensitivePrefsIfNeeded()
        authTokenState.value = securePreferences.getString(SECURE_AUTH_TOKEN_KEY, null)
        currentSessionState.value = securePreferences.getString(SECURE_CURRENT_SESSION_KEY, null)?.let(::decodeSession)
    }

    val authTokenFlow: Flow<String?> = authTokenState
    val baseUrlFlow: Flow<String> = dataStore.data.map { it[baseUrlKey] ?: BuildConfig.DEFAULT_API_BASE_URL }
    val currentSessionFlow: Flow<CurrentIdentitySession?> = currentSessionState
    val themePreferenceFlow: Flow<LiaoThemePreference> = dataStore.data.map { prefs ->
        LiaoThemePreference.fromStorage(prefs[themePreferenceKey])
    }

    suspend fun saveAuthToken(token: String) {
        securePreferences.edit().putString(SECURE_AUTH_TOKEN_KEY, token).apply()
        authTokenState.value = token
    }

    suspend fun clearAuthToken() {
        securePreferences.edit().remove(SECURE_AUTH_TOKEN_KEY).apply()
        authTokenState.value = null
    }

    suspend fun readAuthToken(): String? = authTokenFlow.first()

    suspend fun saveBaseUrl(url: String) {
        val previous = readBaseUrl()
        dataStore.edit { it[baseUrlKey] = url }
        if (previous != url) {
            clearNetworkScopedCaches()
        }
    }

    suspend fun readBaseUrl(): String = baseUrlFlow.first()

    suspend fun saveThemePreference(preference: LiaoThemePreference) {
        dataStore.edit { it[themePreferenceKey] = preference.storageValue }
    }

    suspend fun readThemePreference(): LiaoThemePreference = themePreferenceFlow.first()

    suspend fun saveCurrentSession(session: CurrentIdentitySession) {
        val previousSessionId = readCurrentSession()?.id
        if (previousSessionId != session.id) {
            clearSessionScopedCaches()
        }
        val payload = StoredSession(
            id = session.id,
            name = session.name,
            sex = session.sex,
        )
        val raw = json.encodeToString(payload)
        securePreferences.edit().putString(SECURE_CURRENT_SESSION_KEY, raw).apply()
        currentSessionState.value = decodeSession(raw)
    }

    suspend fun readCurrentSession(): CurrentIdentitySession? = currentSessionFlow.first()

    suspend fun clearCurrentSession() {
        securePreferences.edit().remove(SECURE_CURRENT_SESSION_KEY).apply()
        currentSessionState.value = null
        dataStore.edit { clearSessionScopedCaches(it) }
    }

    suspend fun saveCachedSystemConfig(config: SystemConfigDto) {
        saveJson(cachedSystemConfigKey, config, SystemConfigDto.serializer())
    }

    suspend fun readCachedSystemConfig(): SystemConfigDto? =
        readJson(cachedSystemConfigKey, SystemConfigDto.serializer())

    suspend fun clearCachedSystemConfig() {
        dataStore.edit { it.remove(cachedSystemConfigKey) }
    }

    suspend fun saveCachedMediaLibrary(snapshot: CachedMediaLibrarySnapshot) {
        saveJson(cachedMediaLibraryKey, snapshot, CachedMediaLibrarySnapshot.serializer())
    }

    suspend fun readCachedMediaLibrary(): CachedMediaLibrarySnapshot? =
        readJson(cachedMediaLibraryKey, CachedMediaLibrarySnapshot.serializer())

    suspend fun clearCachedMediaLibrary() {
        dataStore.edit { it.remove(cachedMediaLibraryKey) }
    }

    suspend fun saveCachedVideoExtractTaskList(snapshot: CachedVideoExtractTaskListSnapshot) {
        saveJson(cachedVideoExtractTaskListKey, snapshot, CachedVideoExtractTaskListSnapshot.serializer())
    }

    suspend fun readCachedVideoExtractTaskList(): CachedVideoExtractTaskListSnapshot? =
        readJson(cachedVideoExtractTaskListKey, CachedVideoExtractTaskListSnapshot.serializer())

    suspend fun clearCachedVideoExtractTaskList() {
        dataStore.edit { it.remove(cachedVideoExtractTaskListKey) }
    }

    suspend fun saveCachedVideoExtractTaskDetail(snapshot: CachedVideoExtractTaskDetailSnapshot) {
        val currentItems = readCachedVideoExtractTaskDetailsSnapshot()?.items.orEmpty()
        val mergedItems = listOf(snapshot) + currentItems.filterNot { it.task.taskId == snapshot.task.taskId }
        saveJson(
            cachedVideoExtractTaskDetailsKey,
            CachedVideoExtractTaskDetailsSnapshot(items = mergedItems.take(MAX_CACHED_VIDEO_EXTRACT_TASK_DETAILS)),
            CachedVideoExtractTaskDetailsSnapshot.serializer(),
        )
    }

    suspend fun readCachedVideoExtractTaskDetail(taskId: String): CachedVideoExtractTaskDetailSnapshot? =
        readCachedVideoExtractTaskDetailsSnapshot()?.items?.firstOrNull { it.task.taskId == taskId.trim() }

    suspend fun removeCachedVideoExtractTaskDetail(taskId: String) {
        val normalizedTaskId = taskId.trim()
        val currentItems = readCachedVideoExtractTaskDetailsSnapshot()?.items.orEmpty()
        saveJson(
            cachedVideoExtractTaskDetailsKey,
            CachedVideoExtractTaskDetailsSnapshot(items = currentItems.filterNot { it.task.taskId == normalizedTaskId }),
            CachedVideoExtractTaskDetailsSnapshot.serializer(),
        )
    }

    suspend fun clearCachedVideoExtractTaskDetails() {
        dataStore.edit { it.remove(cachedVideoExtractTaskDetailsKey) }
    }

    private suspend fun readCachedVideoExtractTaskDetailsSnapshot(): CachedVideoExtractTaskDetailsSnapshot? =
        readJson(cachedVideoExtractTaskDetailsKey, CachedVideoExtractTaskDetailsSnapshot.serializer())

    private fun migrateLegacySensitivePrefsIfNeeded() {
        runBlocking {
            val prefs = dataStore.data.first()
            val legacyAuthToken = prefs[authTokenKey]
            val legacyCurrentSession = prefs[currentSessionKey]
            var changed = false
            if (securePreferences.getString(SECURE_AUTH_TOKEN_KEY, null).isNullOrBlank() && !legacyAuthToken.isNullOrBlank()) {
                securePreferences.edit().putString(SECURE_AUTH_TOKEN_KEY, legacyAuthToken).apply()
                changed = true
            }
            if (securePreferences.getString(SECURE_CURRENT_SESSION_KEY, null).isNullOrBlank() && !legacyCurrentSession.isNullOrBlank()) {
                securePreferences.edit().putString(SECURE_CURRENT_SESSION_KEY, legacyCurrentSession).apply()
                changed = true
            }
            if (changed || !legacyAuthToken.isNullOrBlank() || !legacyCurrentSession.isNullOrBlank()) {
                dataStore.edit {
                    it.remove(authTokenKey)
                    it.remove(currentSessionKey)
                }
            }
        }
    }

    private suspend fun clearNetworkScopedCaches() {
        dataStore.edit {
            it.remove(cachedSystemConfigKey)
            clearSessionScopedCaches(it)
        }
    }

    private suspend fun clearSessionScopedCaches() {
        dataStore.edit { clearSessionScopedCaches(it) }
    }

    private fun clearSessionScopedCaches(preferences: MutablePreferences) {
        preferences.remove(cachedMediaLibraryKey)
        preferences.remove(cachedVideoExtractTaskListKey)
        preferences.remove(cachedVideoExtractTaskDetailsKey)
    }

    private suspend fun <T> saveJson(key: Preferences.Key<String>, value: T, serializer: KSerializer<T>) {
        dataStore.edit { it[key] = json.encodeToString(serializer, value) }
    }

    private suspend fun <T> readJson(key: Preferences.Key<String>, serializer: KSerializer<T>): T? {
        val raw = dataStore.data.first()[key] ?: return null
        return runCatching { json.decodeFromString(serializer, raw) }.getOrNull()
    }

    private fun decodeSession(raw: String): CurrentIdentitySession? = runCatching {
        val stored = json.decodeFromString(StoredSession.serializer(), raw)
        CurrentIdentitySession(
            id = stored.id,
            name = stored.name,
            sex = stored.sex,
            cookie = stored.cookie.ifBlank { generateCookie(stored.id, stored.name) },
            ip = stored.ip.ifBlank { generateRandomIp() },
            area = stored.area.ifBlank { "未知" },
        )
    }.getOrNull()
}
