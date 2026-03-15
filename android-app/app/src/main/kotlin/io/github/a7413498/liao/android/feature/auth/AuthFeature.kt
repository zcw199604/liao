/*
 * 登录模块负责 Base URL 输入、访问码登录与已有 Token 校验。
 * Android 客户端的所有后续 API 与 WebSocket 流程都以此为起点。
 */
package io.github.a7413498.liao.android.feature.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
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
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.LiaoLogger
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.AuthApiService
import javax.inject.Inject
import kotlinx.coroutines.launch

class AuthRepository @Inject constructor(
    private val authApiService: AuthApiService,
    private val preferencesStore: AppPreferencesStore,
) {
    suspend fun login(baseUrl: String, accessCode: String): AppResult<Unit> = runCatching {
        preferencesStore.saveBaseUrl(baseUrl)
        val response = authApiService.login(accessCode)
        if (response.code != 0 || response.token.isNullOrBlank()) {
            error(response.msg ?: response.message ?: "登录失败")
        }
        preferencesStore.saveAuthToken(response.token)
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "登录失败", it) },
    )

    suspend fun restoreSessionState(): AppResult<Boolean> = runCatching {
        val token = preferencesStore.readAuthToken().orEmpty()
        if (token.isBlank()) return@runCatching false
        val valid = authApiService.verify().valid == true
        if (!valid) {
            preferencesStore.clearAuthToken()
            preferencesStore.clearCurrentSession()
            return@runCatching false
        }
        preferencesStore.readCurrentSession() != null
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = {
            LiaoLogger.w("AuthRepository", "Token 校验失败，已清理本地会话", it)
            runCatching {
                preferencesStore.clearAuthToken()
                preferencesStore.clearCurrentSession()
            }
            AppResult.Error(it.message ?: "登录状态恢复失败", it)
        },
    )
}

data class LoginUiState(
    val baseUrl: String = "http://10.0.2.2:8080/api/",
    val accessCode: String = "",
    val loading: Boolean = true,
    val errorMessage: String? = null,
    val loggedIn: Boolean = false,
    val hasCurrentSession: Boolean = false,
)

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    private val preferencesStore: AppPreferencesStore,
) : ViewModel() {
    var uiState by mutableStateOf(LoginUiState())
        private set

    init {
        viewModelScope.launch {
            val baseUrl = preferencesStore.readBaseUrl()
            when (val result = authRepository.restoreSessionState()) {
                is AppResult.Success -> {
                    val hasToken = !preferencesStore.readAuthToken().isNullOrBlank()
                    uiState = uiState.copy(
                        baseUrl = baseUrl,
                        loading = false,
                        loggedIn = hasToken,
                        hasCurrentSession = result.data,
                    )
                }

                is AppResult.Error -> uiState = uiState.copy(
                    baseUrl = baseUrl,
                    loading = false,
                    loggedIn = false,
                    hasCurrentSession = false,
                    errorMessage = result.message,
                )
            }
        }
    }

    fun updateBaseUrl(value: String) {
        uiState = uiState.copy(baseUrl = value)
    }

    fun updateAccessCode(value: String) {
        uiState = uiState.copy(accessCode = value)
    }

    fun login() {
        if (uiState.loading) return
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, errorMessage = null)
            when (val result = authRepository.login(uiState.baseUrl, uiState.accessCode)) {
                is AppResult.Success -> uiState = uiState.copy(
                    loading = false,
                    loggedIn = true,
                    hasCurrentSession = false,
                )
                is AppResult.Error -> uiState = uiState.copy(loading = false, errorMessage = result.message)
            }
        }
    }
}

@Composable
fun LoginScreen(
    viewModel: LoginViewModel,
    onLoginSuccess: (Boolean) -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(state.loggedIn, state.hasCurrentSession) {
        if (state.loggedIn) onLoginSuccess(state.hasCurrentSession)
    }

    LaunchedEffect(state.errorMessage) {
        state.errorMessage?.let { snackbarHostState.showSnackbar(it) }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(hostState = snackbarHostState) },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .padding(padding)
                .padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(text = "Liao Android", style = MaterialTheme.typography.headlineMedium)
            Text(text = "请输入服务地址与访问码，完成后端登录。")
            OutlinedTextField(
                modifier = Modifier.fillMaxWidth(),
                value = state.baseUrl,
                onValueChange = viewModel::updateBaseUrl,
                label = { Text("API Base URL") },
                singleLine = true,
                enabled = !state.loading,
            )
            OutlinedTextField(
                modifier = Modifier.fillMaxWidth(),
                value = state.accessCode,
                onValueChange = viewModel::updateAccessCode,
                label = { Text("访问码") },
                singleLine = true,
                enabled = !state.loading,
            )
            Button(
                modifier = Modifier.fillMaxWidth(),
                enabled = !state.loading && state.accessCode.isNotBlank(),
                onClick = viewModel::login,
            ) {
                if (state.loading) {
                    CircularProgressIndicator()
                } else {
                    Text("登录")
                }
            }
        }
    }
}
