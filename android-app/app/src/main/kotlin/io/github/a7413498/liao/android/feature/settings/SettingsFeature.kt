/*
 * 设置模块承载 Android 端的基础配置、当前身份编辑与系统管理入口。
 * 本轮继续对齐 Web 端的媒体管理入口与图片端口策略配置能力。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.settings

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.app.theme.LiaoThemePreference
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.generateCookie
import io.github.a7413498.liao.android.core.database.ConversationDao
import io.github.a7413498.liao.android.core.database.MessageDao
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ConnectionStatsDto
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.network.SystemApiService
import io.github.a7413498.liao.android.core.network.SystemConfigDto
import io.github.a7413498.liao.android.core.websocket.LiaoWebSocketClient
import io.github.a7413498.liao.android.core.websocket.WebSocketState
import javax.inject.Inject
import kotlinx.coroutines.async
import kotlinx.coroutines.supervisorScope
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject

private const val DEFAULT_IMAGE_PORT_MODE = "fixed"
private const val DEFAULT_IMAGE_PORT_FIXED = "9006"
private const val DEFAULT_IMAGE_PORT_REAL_MIN_BYTES = "2048"
private const val DEFAULT_MTPHOTO_TIMELINE_THRESHOLD = "10"

data class SettingsSnapshot(
    val baseUrl: String,
    val tokenPreview: String,
    val currentIdentityId: String,
    val currentIdentityName: String,
    val currentIdentitySex: String,
    val currentIdentityPreview: String,
    val connectionStats: ConnectionStatsDto,
    val forceoutUserCount: Int,
    val systemConfig: SystemConfigDto,
    val themePreference: LiaoThemePreference,
)

data class SaveIdentityResult(
    val updatedIdentity: IdentityDto,
    val message: String,
)

class SettingsRepository @Inject constructor(
    private val preferencesStore: AppPreferencesStore,
    private val identityApiService: IdentityApiService,
    private val systemApiService: SystemApiService,
    private val webSocketClient: LiaoWebSocketClient,
    private val conversationDao: ConversationDao,
    private val messageDao: MessageDao,
) {
    suspend fun loadSnapshot(): AppResult<SettingsSnapshot> = runCatching {
        supervisorScope {
            val currentSession = preferencesStore.readCurrentSession()
            val baseUrlDeferred = async { preferencesStore.readBaseUrl() }
            val tokenDeferred = async { preferencesStore.readAuthToken() }
            val statsDeferred = async {
                runCatching { systemApiService.getConnectionStats().data ?: ConnectionStatsDto() }
                    .getOrDefault(ConnectionStatsDto())
            }
            val forceoutDeferred = async {
                runCatching { systemApiService.getForceoutUserCount().data ?: 0 }
                    .getOrDefault(0)
            }
            val systemConfigDeferred = async { loadSystemConfigWithCacheFallback() }
            val themePreferenceDeferred = async { preferencesStore.readThemePreference() }
            val baseUrl = baseUrlDeferred.await()
            val token = tokenDeferred.await()
            val connectionStats = statsDeferred.await()
            val forceoutUserCount = forceoutDeferred.await()
            val systemConfig = systemConfigDeferred.await()
            val themePreference = themePreferenceDeferred.await()
            SettingsSnapshot(
                baseUrl = baseUrl,
                tokenPreview = token?.take(16)?.plus("...") ?: "未登录",
                currentIdentityId = currentSession?.id.orEmpty(),
                currentIdentityName = currentSession?.name.orEmpty(),
                currentIdentitySex = currentSession?.sex ?: "女",
                currentIdentityPreview = currentSession?.let {
                    "${it.name} (${it.id.take(8)}) / ${it.ip} / ${it.area}"
                } ?: "未选择身份",
                connectionStats = connectionStats,
                forceoutUserCount = forceoutUserCount,
                systemConfig = systemConfig,
                themePreference = themePreference,
            )
        }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载设置失败", it) },
    )

    suspend fun saveThemePreference(preference: LiaoThemePreference): AppResult<LiaoThemePreference> = runCatching {
        preferencesStore.saveThemePreference(preference)
        preference
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "保存主题偏好失败", it) },
    )

    private suspend fun loadSystemConfigWithCacheFallback(): SystemConfigDto {
        val remoteConfig = runCatching {
            systemApiService.getSystemConfig().data ?: defaultSystemConfig()
        }.getOrNull()
        if (remoteConfig != null) {
            preferencesStore.saveCachedSystemConfig(remoteConfig)
            return remoteConfig
        }
        return preferencesStore.readCachedSystemConfig() ?: defaultSystemConfig()
    }

    suspend fun saveBaseUrl(value: String): AppResult<Unit> = runCatching {
        preferencesStore.saveBaseUrl(value.trim())
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "保存地址失败", it) },
    )

    suspend fun saveIdentity(identityId: String, name: String, sex: String): AppResult<SaveIdentityResult> = runCatching {
        val currentSession = preferencesStore.readCurrentSession() ?: error("请先选择身份")
        val nextId = identityId.trim()
        val nextName = name.trim()
        val nextSex = sex.trim()
        if (nextId.isBlank()) error("身份 ID 不能为空")
        if (nextName.isBlank()) error("昵称不能为空")
        if (nextSex.isBlank()) error("性别不能为空")

        if (nextId != currentSession.id) {
            val response = identityApiService.updateIdentityId(
                oldId = currentSession.id,
                newId = nextId,
                name = nextName,
                sex = nextSex,
            )
            val updated = response.data ?: error(response.msg ?: response.message ?: "更新身份失败")
            val nextSession = currentSession.copy(
                id = updated.id,
                name = updated.name,
                sex = updated.sex,
                cookie = generateCookie(updated.id, updated.name),
            )
            conversationDao.clearAll()
            messageDao.clearAll()
            preferencesStore.saveCurrentSession(nextSession)
            SaveIdentityResult(updatedIdentity = updated, message = "身份 ID 已更新，已重建本地会话")
        } else {
            val response = identityApiService.updateIdentity(
                id = currentSession.id,
                name = nextName,
                sex = nextSex,
            )
            val updated = response.data ?: error(response.msg ?: response.message ?: "更新身份失败")
            val nextSession = currentSession.copy(
                name = updated.name,
                sex = updated.sex,
                cookie = generateCookie(updated.id, updated.name),
            )
            preferencesStore.saveCurrentSession(nextSession)

            val syncNotes = mutableListOf<String>()
            if (currentSession.sex != updated.sex && !webSocketClient.sendModifyInfo(senderId = updated.id, userSex = updated.sex)) {
                syncNotes += "性别实时同步未发送"
            }
            if (currentSession.name != updated.name && !webSocketClient.sendChangeName(senderId = updated.id, newName = updated.name)) {
                syncNotes += "昵称实时同步未发送"
            }
            val message = if (syncNotes.isEmpty()) {
                "身份信息已保存"
            } else {
                "身份信息已保存，${syncNotes.joinToString("、")}"
            }
            SaveIdentityResult(updatedIdentity = updated, message = message)
        }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "保存身份失败", it) },
    )

    suspend fun saveSystemConfig(
        imagePortMode: String,
        imagePortFixed: String,
        imagePortRealMinBytes: String,
        mtPhotoTimelineDeferSubfolderThreshold: String,
    ): AppResult<SystemConfigDto> = runCatching {
        val normalizedMode = normalizeImagePortMode(imagePortMode)
        val normalizedFixedPort = imagePortFixed.trim().ifBlank { DEFAULT_IMAGE_PORT_FIXED }
        val normalizedRealMinBytes = imagePortRealMinBytes.trim().toLongOrNull()?.takeIf { it > 0 }
            ?: DEFAULT_IMAGE_PORT_REAL_MIN_BYTES.toLong()
        val normalizedMtPhotoThreshold = mtPhotoTimelineDeferSubfolderThreshold.trim().toIntOrNull()?.takeIf { it > 0 }
            ?: DEFAULT_MTPHOTO_TIMELINE_THRESHOLD.toInt()

        val response = systemApiService.updateSystemConfig(
            buildJsonObject {
                put("imagePortMode", JsonPrimitive(normalizedMode))
                put("imagePortFixed", JsonPrimitive(normalizedFixedPort))
                put("imagePortRealMinBytes", JsonPrimitive(normalizedRealMinBytes))
                put("mtPhotoTimelineDeferSubfolderThreshold", JsonPrimitive(normalizedMtPhotoThreshold))
            }
        )
        if (response.code != 0 && response.data == null) {
            error(response.msg ?: response.message ?: "保存图片端口策略失败")
        }
        (response.data ?: SystemConfigDto(
            imagePortMode = normalizedMode,
            imagePortFixed = normalizedFixedPort,
            imagePortRealMinBytes = normalizedRealMinBytes,
            mtPhotoTimelineDeferSubfolderThreshold = normalizedMtPhotoThreshold,
        )).also { config ->
            preferencesStore.saveCachedSystemConfig(config)
        }
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "保存图片端口策略失败", it) },
    )

    suspend fun logout(): AppResult<Unit> = runCatching {
        webSocketClient.disconnect(manual = true)
        preferencesStore.clearAuthToken()
        preferencesStore.clearCurrentSession()
        conversationDao.clearAll()
        messageDao.clearAll()
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "退出登录失败", it) },
    )

    suspend fun disconnectAllConnections(): AppResult<String> = runCatching {
        val response = systemApiService.disconnectAllConnections()
        if (response.code != 0) error(response.msg ?: response.message ?: "断开所有连接失败")
        "已请求断开全部连接"
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "断开所有连接失败", it) },
    )

    suspend fun clearForceoutUsers(): AppResult<String> = runCatching {
        val response = systemApiService.clearForceoutUsers()
        if (response.code != 0) error(response.msg ?: response.message ?: "清空禁连用户失败")
        response.msg ?: response.message ?: "已清空禁连用户"
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "清空禁连用户失败", it) },
    )
}

data class SettingsUiState(
    val loading: Boolean = true,
    val savingIdentity: Boolean = false,
    val savingBaseUrl: Boolean = false,
    val savingSystemConfig: Boolean = false,
    val baseUrl: String = "",
    val tokenPreview: String = "未登录",
    val currentIdentityId: String = "",
    val currentIdentityName: String = "",
    val currentIdentitySex: String = "女",
    val currentIdentityPreview: String = "未选择身份",
    val connectionStateLabel: String = "Closed",
    val connectionStats: ConnectionStatsDto = ConnectionStatsDto(),
    val forceoutUserCount: Int = 0,
    val imagePortMode: String = DEFAULT_IMAGE_PORT_MODE,
    val imagePortFixed: String = DEFAULT_IMAGE_PORT_FIXED,
    val imagePortRealMinBytes: String = DEFAULT_IMAGE_PORT_REAL_MIN_BYTES,
    val mtPhotoTimelineDeferSubfolderThreshold: String = DEFAULT_MTPHOTO_TIMELINE_THRESHOLD,
    val themePreference: LiaoThemePreference = LiaoThemePreference.DARK,
    val message: String? = null,
    val loggedOut: Boolean = false,
)

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val repository: SettingsRepository,
    private val webSocketClient: LiaoWebSocketClient,
) : ViewModel() {
    var uiState by mutableStateOf(SettingsUiState())
        private set

    init {
        refresh()
        observeConnectionState()
    }

    private fun observeConnectionState() {
        viewModelScope.launch {
            webSocketClient.state.collect { state ->
                uiState = uiState.copy(connectionStateLabel = state.toDisplayText())
            }
        }
    }

    private fun applySnapshot(snapshot: SettingsSnapshot) {
        uiState = uiState.copy(
            loading = false,
            baseUrl = snapshot.baseUrl,
            tokenPreview = snapshot.tokenPreview,
            currentIdentityId = snapshot.currentIdentityId,
            currentIdentityName = snapshot.currentIdentityName,
            currentIdentitySex = snapshot.currentIdentitySex,
            currentIdentityPreview = snapshot.currentIdentityPreview,
            connectionStats = snapshot.connectionStats,
            forceoutUserCount = snapshot.forceoutUserCount,
            imagePortMode = snapshot.systemConfig.imagePortMode.toUiMode(),
            imagePortFixed = snapshot.systemConfig.imagePortFixed.ifBlank { DEFAULT_IMAGE_PORT_FIXED },
            imagePortRealMinBytes = snapshot.systemConfig.imagePortRealMinBytes.toUiRealMinBytes(),
            mtPhotoTimelineDeferSubfolderThreshold = snapshot.systemConfig.mtPhotoTimelineDeferSubfolderThreshold.toUiMtPhotoThreshold(),
            themePreference = snapshot.themePreference,
        )
    }

    private fun applySystemConfig(config: SystemConfigDto, message: String? = null) {
        uiState = uiState.copy(
            savingSystemConfig = false,
            imagePortMode = config.imagePortMode.toUiMode(),
            imagePortFixed = config.imagePortFixed.ifBlank { DEFAULT_IMAGE_PORT_FIXED },
            imagePortRealMinBytes = config.imagePortRealMinBytes.toUiRealMinBytes(),
            mtPhotoTimelineDeferSubfolderThreshold = config.mtPhotoTimelineDeferSubfolderThreshold.toUiMtPhotoThreshold(),
            message = message,
        )
    }

    fun refresh() {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, message = null)
            when (val result = repository.loadSnapshot()) {
                is AppResult.Success -> applySnapshot(result.data)
                is AppResult.Error -> uiState = uiState.copy(loading = false, message = result.message)
            }
        }
    }

    fun updateBaseUrl(value: String) {
        uiState = uiState.copy(baseUrl = value)
    }

    fun updateIdentityId(value: String) {
        uiState = uiState.copy(currentIdentityId = value)
    }

    fun updateIdentityName(value: String) {
        uiState = uiState.copy(currentIdentityName = value)
    }

    fun updateIdentitySex(value: String) {
        uiState = uiState.copy(currentIdentitySex = value)
    }

    fun updateImagePortMode(value: String) {
        uiState = uiState.copy(imagePortMode = normalizeImagePortMode(value))
    }

    fun updateImagePortFixed(value: String) {
        uiState = uiState.copy(imagePortFixed = value)
    }

    fun updateImagePortRealMinBytes(value: String) {
        uiState = uiState.copy(imagePortRealMinBytes = value)
    }

    fun updateMtPhotoTimelineDeferSubfolderThreshold(value: String) {
        uiState = uiState.copy(mtPhotoTimelineDeferSubfolderThreshold = value)
    }

    fun updateThemePreference(value: LiaoThemePreference) {
        if (uiState.themePreference == value) return
        viewModelScope.launch {
            when (val result = repository.saveThemePreference(value)) {
                is AppResult.Success -> uiState = uiState.copy(themePreference = result.data, message = "已切换主题：${result.data.toDisplayLabel()}")
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun saveBaseUrl() {
        if (uiState.savingBaseUrl) return
        viewModelScope.launch {
            uiState = uiState.copy(savingBaseUrl = true, message = null)
            when (val result = repository.saveBaseUrl(uiState.baseUrl)) {
                is AppResult.Success -> uiState = uiState.copy(savingBaseUrl = false, message = "已保存 Base URL")
                is AppResult.Error -> uiState = uiState.copy(savingBaseUrl = false, message = result.message)
            }
        }
    }

    fun saveIdentity() {
        if (uiState.savingIdentity) return
        viewModelScope.launch {
            uiState = uiState.copy(savingIdentity = true, message = null)
            when (
                val result = repository.saveIdentity(
                    identityId = uiState.currentIdentityId,
                    name = uiState.currentIdentityName,
                    sex = uiState.currentIdentitySex,
                )
            ) {
                is AppResult.Success -> {
                    uiState = uiState.copy(savingIdentity = false, message = result.data.message)
                    refresh()
                }
                is AppResult.Error -> uiState = uiState.copy(savingIdentity = false, message = result.message)
            }
        }
    }

    fun saveSystemConfig() {
        if (uiState.savingSystemConfig) return
        viewModelScope.launch {
            uiState = uiState.copy(savingSystemConfig = true, message = null)
            when (
                val result = repository.saveSystemConfig(
                    imagePortMode = uiState.imagePortMode,
                    imagePortFixed = uiState.imagePortFixed,
                    imagePortRealMinBytes = uiState.imagePortRealMinBytes,
                    mtPhotoTimelineDeferSubfolderThreshold = uiState.mtPhotoTimelineDeferSubfolderThreshold,
                )
            ) {
                is AppResult.Success -> applySystemConfig(result.data, message = "已保存图片端口策略")
                is AppResult.Error -> uiState = uiState.copy(savingSystemConfig = false, message = result.message)
            }
        }
    }

    fun disconnectAllConnections() {
        viewModelScope.launch {
            when (val result = repository.disconnectAllConnections()) {
                is AppResult.Success -> {
                    uiState = uiState.copy(message = result.data)
                    refresh()
                }
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun clearForceoutUsers() {
        viewModelScope.launch {
            when (val result = repository.clearForceoutUsers()) {
                is AppResult.Success -> {
                    uiState = uiState.copy(message = result.data)
                    refresh()
                }
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun logout() {
        viewModelScope.launch {
            when (val result = repository.logout()) {
                is AppResult.Success -> uiState = uiState.copy(loggedOut = true, message = "已退出登录")
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }
}

private fun WebSocketState.toDisplayText(): String = when (this) {
    WebSocketState.Idle -> "Idle"
    WebSocketState.Connecting -> "Connecting"
    WebSocketState.Connected -> "Connected"
    WebSocketState.Reconnecting -> "Reconnecting"
    is WebSocketState.Forceout -> "Forceout"
    WebSocketState.Closed -> "Closed"
}

private fun defaultSystemConfig(): SystemConfigDto = SystemConfigDto(
    imagePortMode = DEFAULT_IMAGE_PORT_MODE,
    imagePortFixed = DEFAULT_IMAGE_PORT_FIXED,
    imagePortRealMinBytes = DEFAULT_IMAGE_PORT_REAL_MIN_BYTES.toLong(),
    mtPhotoTimelineDeferSubfolderThreshold = DEFAULT_MTPHOTO_TIMELINE_THRESHOLD.toInt(),
)

private fun normalizeImagePortMode(value: String): String = when (value.trim().lowercase()) {
    "fixed", "probe", "real" -> value.trim().lowercase()
    else -> DEFAULT_IMAGE_PORT_MODE
}

private fun String.toUiMode(): String = normalizeImagePortMode(this)

private fun Long.toUiRealMinBytes(): String = if (this > 0) this.toString() else DEFAULT_IMAGE_PORT_REAL_MIN_BYTES

private fun Int.toUiMtPhotoThreshold(): String = if (this > 0) this.toString() else DEFAULT_MTPHOTO_TIMELINE_THRESHOLD

private fun LiaoThemePreference.toDisplayLabel(): String = when (this) {
    LiaoThemePreference.AUTO -> "跟随系统"
    LiaoThemePreference.LIGHT -> "浅色"
    LiaoThemePreference.DARK -> "深色"
}

@Composable
private fun SettingsSection(
    title: String,
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    Card(modifier = modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(text = title, style = MaterialTheme.typography.titleMedium)
            content()
        }
    }
}

@Composable
private fun ModeOptionButton(
    label: String,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    if (selected) {
        Button(onClick = onClick, modifier = modifier) {
            Text(label)
        }
    } else {
        OutlinedButton(onClick = onClick, modifier = modifier) {
            Text(label)
        }
    }
}

@Composable
fun SettingsScreen(
    viewModel: SettingsViewModel,
    onBack: () -> Unit,
    onOpenGlobalFavorites: () -> Unit,
    onOpenMediaLibrary: () -> Unit,
    onOpenMtPhoto: () -> Unit,
    onOpenDouyin: () -> Unit,
    onOpenVideoExtract: () -> Unit,
    onOpenVideoExtractTasks: () -> Unit,
    onLoggedOut: () -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }
    val scrollState = rememberScrollState()
    val systemDark = isSystemInDarkTheme()
    val resolvedThemeLabel = remember(state.themePreference, systemDark) {
        when (state.themePreference) {
            LiaoThemePreference.AUTO -> if (systemDark) "深色" else "浅色"
            LiaoThemePreference.LIGHT -> "浅色"
            LiaoThemePreference.DARK -> "深色"
        }
    }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    LaunchedEffect(state.loggedOut) {
        if (state.loggedOut) {
            onLoggedOut()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("设置") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                }
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(scrollState)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            if (state.loading) {
                CircularProgressIndicator()
            }

            SettingsSection(title = "外观") {
                Text(
                    text = "主题：${state.themePreference.toDisplayLabel()} · 当前：$resolvedThemeLabel",
                    style = MaterialTheme.typography.bodySmall,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ModeOptionButton(
                        label = "系统",
                        selected = state.themePreference == LiaoThemePreference.AUTO,
                        onClick = { viewModel.updateThemePreference(LiaoThemePreference.AUTO) },
                        modifier = Modifier.weight(1f),
                    )
                    ModeOptionButton(
                        label = "浅色",
                        selected = state.themePreference == LiaoThemePreference.LIGHT,
                        onClick = { viewModel.updateThemePreference(LiaoThemePreference.LIGHT) },
                        modifier = Modifier.weight(1f),
                    )
                    ModeOptionButton(
                        label = "深色",
                        selected = state.themePreference == LiaoThemePreference.DARK,
                        onClick = { viewModel.updateThemePreference(LiaoThemePreference.DARK) },
                        modifier = Modifier.weight(1f),
                    )
                }
            }

            SettingsSection(title = "连接与地址") {
                OutlinedTextField(
                    modifier = Modifier.fillMaxWidth(),
                    value = state.baseUrl,
                    onValueChange = viewModel::updateBaseUrl,
                    label = { Text("API Base URL") },
                    enabled = !state.savingBaseUrl,
                )
                Text("Token：${state.tokenPreview}")
                Text("WebSocket：${state.connectionStateLabel}")
                Button(
                    onClick = viewModel::saveBaseUrl,
                    enabled = !state.savingBaseUrl && state.baseUrl.isNotBlank(),
                ) {
                    Text(if (state.savingBaseUrl) "保存中..." else "保存地址")
                }
            }

            SettingsSection(title = "当前身份") {
                Text(
                    text = state.currentIdentityPreview,
                    style = MaterialTheme.typography.bodyMedium,
                )
                OutlinedTextField(
                    modifier = Modifier.fillMaxWidth(),
                    value = state.currentIdentityId,
                    onValueChange = viewModel::updateIdentityId,
                    label = { Text("身份 ID") },
                    enabled = !state.savingIdentity,
                )
                OutlinedTextField(
                    modifier = Modifier.fillMaxWidth(),
                    value = state.currentIdentityName,
                    onValueChange = viewModel::updateIdentityName,
                    label = { Text("昵称") },
                    enabled = !state.savingIdentity,
                )
                OutlinedTextField(
                    modifier = Modifier.fillMaxWidth(),
                    value = state.currentIdentitySex,
                    onValueChange = viewModel::updateIdentitySex,
                    label = { Text("性别") },
                    enabled = !state.savingIdentity,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Button(
                        onClick = viewModel::saveIdentity,
                        enabled = !state.savingIdentity && state.currentIdentityId.isNotBlank() && state.currentIdentityName.isNotBlank(),
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(if (state.savingIdentity) "保存中..." else "保存身份")
                    }
                    OutlinedButton(
                        onClick = onOpenGlobalFavorites,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text("全局收藏")
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedButton(
                        onClick = onOpenMediaLibrary,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text("图片管理")
                    }
                    OutlinedButton(
                        onClick = onOpenMtPhoto,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text("mtPhoto 相册")
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedButton(
                        onClick = onOpenDouyin,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text("抖音下载")
                    }
                    OutlinedButton(
                        onClick = onOpenVideoExtract,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text("视频抽帧")
                    }
                }
                OutlinedButton(
                    onClick = onOpenVideoExtractTasks,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text("抽帧任务中心")
                }
            }

            SettingsSection(title = "系统能力") {
                Text("活动连接：${state.connectionStats.active}")
                Text("上游连接：${state.connectionStats.upstream}")
                Text("下游连接：${state.connectionStats.downstream}")
                Text("禁连用户：${state.forceoutUserCount}")
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedButton(onClick = viewModel::refresh, modifier = Modifier.weight(1f)) {
                        Text("刷新统计")
                    }
                    OutlinedButton(onClick = viewModel::disconnectAllConnections, modifier = Modifier.weight(1f)) {
                        Text("断开全部")
                    }
                }
                OutlinedButton(
                    onClick = viewModel::clearForceoutUsers,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text("清空禁连")
                }
            }

            SettingsSection(title = "图片端口策略") {
                Text(
                    text = "模式（全局共用）",
                    style = MaterialTheme.typography.labelLarge,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    ModeOptionButton(
                        label = "固定",
                        selected = state.imagePortMode == "fixed",
                        onClick = { viewModel.updateImagePortMode("fixed") },
                        modifier = Modifier.weight(1f),
                    )
                    ModeOptionButton(
                        label = "探测",
                        selected = state.imagePortMode == "probe",
                        onClick = { viewModel.updateImagePortMode("probe") },
                        modifier = Modifier.weight(1f),
                    )
                    ModeOptionButton(
                        label = "真实",
                        selected = state.imagePortMode == "real",
                        onClick = { viewModel.updateImagePortMode("real") },
                        modifier = Modifier.weight(1f),
                    )
                }
                OutlinedTextField(
                    modifier = Modifier.fillMaxWidth(),
                    value = state.imagePortFixed,
                    onValueChange = viewModel::updateImagePortFixed,
                    label = { Text("固定图片端口") },
                    enabled = !state.savingSystemConfig,
                    singleLine = true,
                )
                if (state.imagePortMode == "real") {
                    OutlinedTextField(
                        modifier = Modifier.fillMaxWidth(),
                        value = state.imagePortRealMinBytes,
                        onValueChange = viewModel::updateImagePortRealMinBytes,
                        label = { Text("最小字节阈值") },
                        enabled = !state.savingSystemConfig,
                        singleLine = true,
                    )
                }
                OutlinedTextField(
                    modifier = Modifier.fillMaxWidth(),
                    value = state.mtPhotoTimelineDeferSubfolderThreshold,
                    onValueChange = viewModel::updateMtPhotoTimelineDeferSubfolderThreshold,
                    label = { Text("mtPhoto 时间线延迟阈值") },
                    enabled = !state.savingSystemConfig,
                    singleLine = true,
                )
                Text(
                    text = "视频端口仍保持现有固定逻辑；仅图片按上述策略解析。真实图片请求会对候选端口发起小范围读取并按阈值判定，首次可能稍慢。mtPhoto 阈值用于控制子文件夹较多时延迟加载时间线预览。",
                    style = MaterialTheme.typography.bodySmall,
                )
                Button(
                    onClick = viewModel::saveSystemConfig,
                    enabled = !state.savingSystemConfig,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (state.savingSystemConfig) "保存中..." else "保存图片端口策略")
                }
            }

            OutlinedButton(
                onClick = viewModel::logout,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("退出登录")
            }
        }
    }
}
