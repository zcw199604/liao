/*
 * 身份模块负责拉取身份池、维护身份并生成当前客户端会话。
 * 当前实现补齐了创建、编辑、删除、快速创建与选择的最小闭环。
 */
package io.github.a7413498.liao.android.feature.identity

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.app.testing.IdentityTestTags
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.common.generateCookie
import io.github.a7413498.liao.android.core.common.generateRandomIp
import io.github.a7413498.liao.android.core.database.IdentityDao
import io.github.a7413498.liao.android.core.database.IdentityEntity
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.IdentityApiService
import io.github.a7413498.liao.android.core.network.IdentityDto
import io.github.a7413498.liao.android.core.network.toSession
import javax.inject.Inject
import kotlinx.coroutines.launch

class IdentityRepository @Inject constructor(
    private val identityApiService: IdentityApiService,
    private val identityDao: IdentityDao,
    private val preferencesStore: AppPreferencesStore,
) {
    private suspend fun replaceIdentityCache(items: List<IdentityDto>) {
        identityDao.replaceAll(
            items.map {
                IdentityEntity(
                    id = it.id,
                    name = it.name,
                    sex = it.sex,
                    createdAt = it.createdAt.orEmpty(),
                    lastUsedAt = it.lastUsedAt.orEmpty(),
                )
            }
        )
    }

    private suspend fun syncEditedCurrentSession(identity: IdentityDto) {
        val currentSession = preferencesStore.readCurrentSession() ?: return
        if (currentSession.id != identity.id) return
        preferencesStore.saveCurrentSession(
            currentSession.copy(
                name = identity.name,
                sex = identity.sex,
            )
        )
    }

    suspend fun loadIdentities(): AppResult<List<IdentityDto>> = runCatching {
        val response = identityApiService.getIdentityList()
        val items = response.data.orEmpty()
        replaceIdentityCache(items)
        items
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载身份失败", it) },
    )

    suspend fun createIdentity(name: String, sex: String): AppResult<IdentityDto> = runCatching {
        val response = identityApiService.createIdentity(name = name, sex = sex)
        response.data ?: error(response.msg ?: "创建身份失败")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "创建身份失败", it) },
    )

    suspend fun updateIdentity(id: String, name: String, sex: String): AppResult<IdentityDto> = runCatching {
        val response = identityApiService.updateIdentity(id = id, name = name, sex = sex)
        val updatedIdentity = response.data ?: error(response.msg ?: "更新身份失败")
        syncEditedCurrentSession(updatedIdentity)
        updatedIdentity
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "更新身份失败", it) },
    )

    suspend fun deleteIdentity(id: String): AppResult<Unit> = runCatching {
        val response = identityApiService.deleteIdentity(id)
        if (response.code != 0) {
            error(response.msg ?: "删除身份失败")
        }
        if (preferencesStore.readCurrentSession()?.id == id) {
            preferencesStore.clearCurrentSession()
        }
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "删除身份失败", it) },
    )

    suspend fun quickCreate(): AppResult<IdentityDto> = runCatching {
        val response = identityApiService.quickCreateIdentity()
        response.data ?: error(response.msg ?: "快速创建失败")
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "快速创建失败", it) },
    )

    suspend fun selectIdentity(identity: IdentityDto): AppResult<Unit> = runCatching {
        identityApiService.selectIdentity(identity.id)
        val cookie = generateCookie(identity.id, identity.name)
        val session = identity.toSession(cookie = cookie, ip = generateRandomIp())
        preferencesStore.saveCurrentSession(session)
    }.fold(
        onSuccess = { AppResult.Success(Unit) },
        onFailure = { AppResult.Error(it.message ?: "选择身份失败", it) },
    )
}

data class IdentityUiState(
    val identities: List<IdentityDto> = emptyList(),
    val loading: Boolean = true,
    val saving: Boolean = false,
    val nameInput: String = "",
    val sexInput: String = "女",
    val editingIdentityId: String? = null,
    val confirmDeleteIdentity: IdentityDto? = null,
    val message: String? = null,
    val selected: Boolean = false,
)

@HiltViewModel
class IdentityViewModel @Inject constructor(
    private val repository: IdentityRepository,
) : ViewModel() {
    var uiState by mutableStateOf(IdentityUiState())
        private set

    init {
        refresh()
    }

    fun updateName(value: String) {
        uiState = uiState.copy(nameInput = value)
    }

    fun updateSex(value: String) {
        uiState = uiState.copy(sexInput = value)
    }

    fun startEditing(identity: IdentityDto) {
        uiState = uiState.copy(
            editingIdentityId = identity.id,
            nameInput = identity.name,
            sexInput = identity.sex,
            message = null,
        )
    }

    fun cancelEditing() {
        uiState = uiState.copy(
            editingIdentityId = null,
            nameInput = "",
            sexInput = "女",
            message = null,
        )
    }

    fun confirmDelete(identity: IdentityDto) {
        uiState = uiState.copy(confirmDeleteIdentity = identity)
    }

    fun dismissDeleteDialog() {
        uiState = uiState.copy(confirmDeleteIdentity = null)
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }

    fun refresh() {
        viewModelScope.launch {
            uiState = uiState.copy(loading = true, message = null)
            reloadIdentities()
        }
    }

    private suspend fun reloadIdentities(
        successMessage: String? = null,
        clearForm: Boolean = false,
        clearEditing: Boolean = false,
        dismissDeleteDialog: Boolean = false,
    ) {
        when (val result = repository.loadIdentities()) {
            is AppResult.Success -> {
                val currentState = uiState
                uiState = currentState.copy(
                    loading = false,
                    saving = false,
                    identities = result.data,
                    nameInput = if (clearForm) "" else currentState.nameInput,
                    sexInput = if (clearForm) "女" else currentState.sexInput,
                    editingIdentityId = if (clearEditing) null else currentState.editingIdentityId,
                    confirmDeleteIdentity = if (dismissDeleteDialog) null else currentState.confirmDeleteIdentity,
                    message = successMessage,
                )
            }

            is AppResult.Error -> {
                val currentState = uiState
                uiState = currentState.copy(
                    loading = false,
                    saving = false,
                    confirmDeleteIdentity = if (dismissDeleteDialog) null else currentState.confirmDeleteIdentity,
                    message = result.message,
                )
            }
        }
    }

    fun quickCreate() {
        if (!uiState.editingIdentityId.isNullOrBlank()) {
            uiState = uiState.copy(message = "请先完成或取消当前编辑")
            return
        }
        viewModelScope.launch {
            uiState = uiState.copy(saving = true, message = null)
            when (val result = repository.quickCreate()) {
                is AppResult.Success -> {
                    uiState = uiState.copy(loading = true)
                    reloadIdentities(successMessage = "已创建 ${result.data.name}")
                }

                is AppResult.Error -> uiState = uiState.copy(saving = false, message = result.message)
            }
        }
    }

    fun submitIdentity() {
        val name = uiState.nameInput.trim()
        val sex = uiState.sexInput.trim()
        if (name.isBlank()) {
            uiState = uiState.copy(message = "名字不能为空")
            return
        }
        if (sex.isBlank()) {
            uiState = uiState.copy(message = "性别不能为空")
            return
        }
        val editingId = uiState.editingIdentityId
        viewModelScope.launch {
            uiState = uiState.copy(saving = true, message = null)
            val result = if (editingId.isNullOrBlank()) {
                repository.createIdentity(name, sex)
            } else {
                repository.updateIdentity(editingId, name, sex)
            }
            when (result) {
                is AppResult.Success -> {
                    val successText = if (editingId.isNullOrBlank()) {
                        "已创建 ${result.data.name}"
                    } else {
                        "已更新 ${result.data.name}"
                    }
                    uiState = uiState.copy(loading = true)
                    reloadIdentities(
                        successMessage = successText,
                        clearForm = true,
                        clearEditing = true,
                    )
                }

                is AppResult.Error -> uiState = uiState.copy(saving = false, message = result.message)
            }
        }
    }

    fun deleteConfirmed() {
        val target = uiState.confirmDeleteIdentity ?: return
        val shouldClearEditing = uiState.editingIdentityId == target.id
        viewModelScope.launch {
            uiState = uiState.copy(saving = true, message = null)
            when (val result = repository.deleteIdentity(target.id)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(loading = true)
                    reloadIdentities(
                        successMessage = "已删除 ${target.name}",
                        clearForm = shouldClearEditing,
                        clearEditing = shouldClearEditing,
                        dismissDeleteDialog = true,
                    )
                }

                is AppResult.Error -> uiState = uiState.copy(
                    saving = false,
                    confirmDeleteIdentity = null,
                    message = result.message,
                )
            }
        }
    }

    fun select(identity: IdentityDto) {
        viewModelScope.launch {
            when (val result = repository.selectIdentity(identity)) {
                is AppResult.Success -> uiState = uiState.copy(selected = true)
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }
}

@Composable
fun IdentityScreen(
    viewModel: IdentityViewModel,
    onIdentitySelected: () -> Unit,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(state.selected) {
        if (state.selected) onIdentitySelected()
    }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    Scaffold(snackbarHost = { SnackbarHost(snackbarHostState) }) { padding ->
        IdentityScreenContent(
            state = state,
            onNameChange = viewModel::updateName,
            onSexChange = viewModel::updateSex,
            onSubmitIdentity = viewModel::submitIdentity,
            onQuickCreate = viewModel::quickCreate,
            onCancelEditing = viewModel::cancelEditing,
            onSelectIdentity = viewModel::select,
            onStartEditing = viewModel::startEditing,
            onConfirmDeleteIdentity = viewModel::confirmDelete,
            onDismissDeleteDialog = viewModel::dismissDeleteDialog,
            onDeleteConfirmed = viewModel::deleteConfirmed,
            modifier = Modifier.padding(padding),
        )
    }
}

internal fun identityIntroText(editingIdentityId: String?): String =
    if (editingIdentityId.isNullOrBlank()) {
        "可创建新身份，或选择已有身份进入聊天。"
    } else {
        "当前处于编辑模式，保存后会同步刷新列表与当前会话。"
    }

internal fun identityPrimaryActionLabel(editingIdentityId: String?): String =
    if (editingIdentityId.isNullOrBlank()) "创建身份" else "保存编辑"

internal fun identitySecondaryActionLabel(editingIdentityId: String?): String =
    if (editingIdentityId.isNullOrBlank()) "快速创建" else "取消编辑"

@Composable
fun IdentityScreenContent(
    state: IdentityUiState,
    onNameChange: (String) -> Unit,
    onSexChange: (String) -> Unit,
    onSubmitIdentity: () -> Unit,
    onQuickCreate: () -> Unit,
    onCancelEditing: () -> Unit,
    onSelectIdentity: (IdentityDto) -> Unit,
    onStartEditing: (IdentityDto) -> Unit,
    onConfirmDeleteIdentity: (IdentityDto) -> Unit,
    onDismissDeleteDialog: () -> Unit,
    onDeleteConfirmed: () -> Unit,
    modifier: Modifier = Modifier,
) {
    state.confirmDeleteIdentity?.let { identity ->
        AlertDialog(
            onDismissRequest = onDismissDeleteDialog,
            title = { Text("确认删除身份") },
            text = { Text("删除后将无法在本地身份池中继续使用 ${identity.name}。") },
            confirmButton = {
                TextButton(
                    onClick = onDeleteConfirmed,
                    enabled = !state.saving,
                    modifier = Modifier.testTag(IdentityTestTags.DELETE_DIALOG_CONFIRM),
                ) {
                    Text("删除")
                }
            },
            dismissButton = {
                TextButton(
                    onClick = onDismissDeleteDialog,
                    enabled = !state.saving,
                    modifier = Modifier.testTag(IdentityTestTags.DELETE_DIALOG_CANCEL),
                ) {
                    Text("取消")
                }
            },
        )
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(
            "选择身份",
            style = MaterialTheme.typography.headlineSmall,
            modifier = Modifier.testTag(IdentityTestTags.TITLE),
        )
        Text(
            text = identityIntroText(state.editingIdentityId),
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.testTag(IdentityTestTags.DESCRIPTION),
        )
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            OutlinedTextField(
                modifier = Modifier
                    .weight(1f)
                    .testTag(IdentityTestTags.NAME_INPUT),
                value = state.nameInput,
                onValueChange = onNameChange,
                label = { Text("名字") },
                singleLine = true,
            )
            OutlinedTextField(
                modifier = Modifier
                    .weight(1f)
                    .testTag(IdentityTestTags.SEX_INPUT),
                value = state.sexInput,
                onValueChange = onSexChange,
                label = { Text("性别（男/女）") },
                singleLine = true,
            )
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Button(
                onClick = onSubmitIdentity,
                enabled = !state.saving,
                modifier = Modifier
                    .weight(1f)
                    .testTag(IdentityTestTags.PRIMARY_ACTION_BUTTON),
            ) {
                Text(identityPrimaryActionLabel(state.editingIdentityId))
            }
            OutlinedButton(
                onClick = if (state.editingIdentityId.isNullOrBlank()) onQuickCreate else onCancelEditing,
                enabled = !state.saving,
                modifier = Modifier
                    .weight(1f)
                    .testTag(IdentityTestTags.SECONDARY_ACTION_BUTTON),
            ) {
                Text(identitySecondaryActionLabel(state.editingIdentityId))
            }
        }
        when {
            state.loading -> {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(modifier = Modifier.testTag(IdentityTestTags.LOADING_INDICATOR))
                }
            }

            state.identities.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f),
                    contentAlignment = Alignment.Center,
                ) {
                    Card(
                        modifier = Modifier
                            .fillMaxWidth()
                            .testTag(IdentityTestTags.EMPTY_STATE),
                    ) {
                        Column(
                            modifier = Modifier.padding(16.dp),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Text("暂无身份", style = MaterialTheme.typography.titleMedium)
                            Text("可先手动创建，或使用快速创建生成一个临时身份。")
                        }
                    }
                }
            }

            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f)
                        .testTag(IdentityTestTags.LIST),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    items(state.identities, key = { it.id }) { identity ->
                        val isEditing = state.editingIdentityId == identity.id
                        Card(
                            modifier = Modifier
                                .fillMaxWidth()
                                .testTag(IdentityTestTags.itemCard(identity.id)),
                        ) {
                            Column(
                                modifier = Modifier.padding(16.dp),
                                verticalArrangement = Arrangement.spacedBy(12.dp),
                            ) {
                                Column(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable(enabled = !state.saving) { onSelectIdentity(identity) },
                                    verticalArrangement = Arrangement.spacedBy(4.dp),
                                ) {
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                    ) {
                                        Text(
                                            text = identity.name,
                                            style = MaterialTheme.typography.titleMedium,
                                        )
                                        if (isEditing) {
                                            Text(
                                                text = "编辑中",
                                                color = MaterialTheme.colorScheme.primary,
                                            )
                                        }
                                    }
                                    Text(text = "${identity.sex} · ${identity.id.take(8)}")
                                    Text(text = "最近使用：${identity.lastUsedAt.orEmpty().ifBlank { "未记录" }}")
                                    Text(
                                        text = "点击上方区域即可直接选择该身份",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    OutlinedButton(
                                        onClick = { onStartEditing(identity) },
                                        enabled = !state.saving,
                                        modifier = Modifier
                                            .weight(1f)
                                            .testTag(IdentityTestTags.editButton(identity.id)),
                                    ) {
                                        Text(if (isEditing) "继续编辑" else "编辑")
                                    }
                                    OutlinedButton(
                                        onClick = { onConfirmDeleteIdentity(identity) },
                                        enabled = !state.saving,
                                        modifier = Modifier
                                            .weight(1f)
                                            .testTag(IdentityTestTags.deleteButton(identity.id)),
                                    ) {
                                        Text("删除")
                                    }
                                    TextButton(
                                        onClick = { onSelectIdentity(identity) },
                                        enabled = !state.saving,
                                        modifier = Modifier
                                            .weight(1f)
                                            .testTag(IdentityTestTags.selectButton(identity.id)),
                                    ) {
                                        Text("选择")
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
