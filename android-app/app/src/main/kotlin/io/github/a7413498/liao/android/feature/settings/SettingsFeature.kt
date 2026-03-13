/*
 * 设置模块用于暴露联调地址、登录状态与当前身份快照。
 * 系统配置、连接统计和高级管理接口将在后续迭代中继续扩展。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import javax.inject.Inject
import kotlinx.coroutines.launch

@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val preferencesStore: AppPreferencesStore,
) : ViewModel() {
    var baseUrl by mutableStateOf("")
        private set
    var tokenPreview by mutableStateOf("未登录")
        private set
    var sessionPreview by mutableStateOf("未选择身份")
        private set
    var message by mutableStateOf<String?>(null)
        private set

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            baseUrl = preferencesStore.readBaseUrl()
            tokenPreview = preferencesStore.readAuthToken()?.take(16)?.plus("...") ?: "未登录"
            sessionPreview = preferencesStore.readCurrentSession()?.let {
                "${it.name} (${it.id.take(8)}) / ${it.ip} / ${it.area}"
            } ?: "未选择身份"
        }
    }

    fun updateBaseUrl(value: String) {
        baseUrl = value
    }

    fun saveBaseUrl() {
        viewModelScope.launch {
            preferencesStore.saveBaseUrl(baseUrl)
            message = "已保存 Base URL"
        }
    }

    fun logout() {
        viewModelScope.launch {
            preferencesStore.clearAuthToken()
            preferencesStore.clearCurrentSession()
            message = "已退出登录"
        }
    }
}

@Composable
fun SettingsScreen(
    viewModel: SettingsViewModel,
    onBack: () -> Unit,
    onLoggedOut: () -> Unit,
) {
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(viewModel.message) {
        viewModel.message?.let { snackbarHostState.showSnackbar(it) }
        if (viewModel.message == "已退出登录") {
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
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            OutlinedTextField(
                modifier = Modifier.fillMaxWidth(),
                value = viewModel.baseUrl,
                onValueChange = viewModel::updateBaseUrl,
                label = { Text("API Base URL") },
            )
            Text("Token：${viewModel.tokenPreview}")
            Text("当前身份：${viewModel.sessionPreview}")
            Button(onClick = viewModel::saveBaseUrl) { Text("保存地址") }
            Button(onClick = viewModel::logout) { Text("退出登录") }
        }
    }
}
