/*
 * DataStore 负责保存 Token、Base URL 与当前会话等轻量级首选项。
 * 这些数据会驱动登录恢复、动态联调地址与身份恢复流程。
 */
package io.github.a7413498.liao.android.core.datastore

import android.content.Context
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStoreFile
import dagger.hilt.android.qualifiers.ApplicationContext
import io.github.a7413498.liao.android.BuildConfig
import io.github.a7413498.liao.android.core.common.CurrentIdentitySession
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.Serializable

@Serializable
private data class StoredSession(
    val id: String,
    val name: String,
    val sex: String,
    val cookie: String,
    val ip: String,
    val area: String,
)

@Singleton
class AppPreferencesStore @Inject constructor(
    @ApplicationContext context: Context,
    private val json: Json,
) {
    private val dataStore = PreferenceDataStoreFactory.create {
        context.preferencesDataStoreFile(name = "liao_android_prefs")
    }

    private val authTokenKey = stringPreferencesKey("auth_token")
    private val baseUrlKey = stringPreferencesKey("base_url")
    private val currentSessionKey = stringPreferencesKey("current_session")

    val authTokenFlow: Flow<String?> = dataStore.data.map { it[authTokenKey] }
    val baseUrlFlow: Flow<String> = dataStore.data.map { it[baseUrlKey] ?: BuildConfig.DEFAULT_API_BASE_URL }
    val currentSessionFlow: Flow<CurrentIdentitySession?> = dataStore.data.map { prefs ->
        prefs[currentSessionKey]?.let { decodeSession(it) }
    }

    suspend fun saveAuthToken(token: String) {
        dataStore.edit { it[authTokenKey] = token }
    }

    suspend fun clearAuthToken() {
        dataStore.edit { it.remove(authTokenKey) }
    }

    suspend fun readAuthToken(): String? = authTokenFlow.first()

    suspend fun saveBaseUrl(url: String) {
        dataStore.edit { it[baseUrlKey] = url }
    }

    suspend fun readBaseUrl(): String = baseUrlFlow.first()

    suspend fun saveCurrentSession(session: CurrentIdentitySession) {
        val payload = StoredSession(
            id = session.id,
            name = session.name,
            sex = session.sex,
            cookie = session.cookie,
            ip = session.ip,
            area = session.area,
        )
        dataStore.edit { it[currentSessionKey] = json.encodeToString(payload) }
    }

    suspend fun readCurrentSession(): CurrentIdentitySession? = currentSessionFlow.first()

    suspend fun clearCurrentSession() {
        dataStore.edit { it.remove(currentSessionKey) }
    }

    private fun decodeSession(raw: String): CurrentIdentitySession? = runCatching {
        val stored = json.decodeFromString(StoredSession.serializer(), raw)
        CurrentIdentitySession(
            id = stored.id,
            name = stored.name,
            sex = stored.sex,
            cookie = stored.cookie,
            ip = stored.ip,
            area = stored.area,
        )
    }.getOrNull()
}
