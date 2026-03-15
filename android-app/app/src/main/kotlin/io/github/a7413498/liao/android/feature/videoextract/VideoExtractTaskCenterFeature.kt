/*
 * 视频抽帧任务中心对齐 Web 端最小主流程：查看任务列表、详情、帧结果，并补齐取消 / 继续 / 删除操作。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.videoextract

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import coil.compose.AsyncImage
import dagger.hilt.android.lifecycle.HiltViewModel
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractFrameItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskDetailSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskItemSnapshot
import io.github.a7413498.liao.android.core.datastore.CachedVideoExtractTaskListSnapshot
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.BaseUrlProvider
import io.github.a7413498.liao.android.core.network.VideoExtractApiService
import io.github.a7413498.liao.android.core.network.stringOrNull
import java.util.Locale
import javax.inject.Inject
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

data class VideoExtractTaskItem(
    val taskId: String,
    val sourceType: String,
    val sourceRef: String,
    val sourcePreviewUrl: String,
    val outputDirLocalPath: String,
    val outputDirUrl: String,
    val outputFormat: String,
    val jpgQuality: Int?,
    val mode: String,
    val keyframeMode: String?,
    val fps: Double?,
    val sceneThreshold: Double?,
    val startSec: Double?,
    val endSec: Double?,
    val maxFrames: Int,
    val framesExtracted: Int,
    val videoWidth: Int,
    val videoHeight: Int,
    val durationSec: Double?,
    val cursorOutTimeSec: Double?,
    val status: String,
    val stopReason: String,
    val lastError: String,
    val createdAt: String,
    val updatedAt: String,
    val runtimeLogs: List<String> = emptyList(),
)

data class VideoExtractFrameItem(
    val seq: Int,
    val url: String,
)

data class VideoExtractFramesUiPage(
    val items: List<VideoExtractFrameItem> = emptyList(),
    val nextCursor: Int = 0,
    val hasMore: Boolean = false,
)

private const val MAX_CACHED_VIDEO_EXTRACT_TASKS = 100
private const val MAX_CACHED_VIDEO_EXTRACT_FRAMES = 200

data class VideoExtractTaskListPage(
    val items: List<VideoExtractTaskItem>,
    val page: Int,
    val pageSize: Int,
    val total: Int,
    val fromCache: Boolean = false,
)

data class VideoExtractTaskDetailResult(
    val task: VideoExtractTaskItem,
    val frames: VideoExtractFramesUiPage,
    val fromCache: Boolean = false,
)

data class VideoExtractTaskCenterUiState(
    val listLoading: Boolean = true,
    val detailLoading: Boolean = false,
    val framesLoadingMore: Boolean = false,
    val actionLoading: Boolean = false,
    val tasks: List<VideoExtractTaskItem> = emptyList(),
    val page: Int = 1,
    val pageSize: Int = 20,
    val total: Int = 0,
    val selectedTaskId: String? = null,
    val selectedTask: VideoExtractTaskItem? = null,
    val frames: VideoExtractFramesUiPage = VideoExtractFramesUiPage(),
    val continueEndSec: String = "",
    val continueMaxFrames: String = "",
    val deleteFiles: Boolean = true,
    val message: String? = null,
)

class VideoExtractTaskCenterRepository @Inject constructor(
    private val videoExtractApiService: VideoExtractApiService,
    private val preferencesStore: AppPreferencesStore,
    private val baseUrlProvider: BaseUrlProvider,
) {
    suspend fun loadTasks(page: Int, pageSize: Int = 20): AppResult<VideoExtractTaskListPage> = runCatching {
        val response = videoExtractApiService.getTaskList(page = page, pageSize = pageSize)
        val root = requireEnvelopeDataObject(response, "加载抽帧任务失败")
        val items = root["items"]?.jsonArray.orEmpty().mapNotNull { element ->
            (element as? JsonObject)?.toTaskItem(currentApiOrigin())
        }
        VideoExtractTaskListPage(
            items = items,
            page = root.intOrDefault("page", page),
            pageSize = root.intOrDefault("pageSize", pageSize),
            total = root.intOrDefault("total", items.size),
        ).also { pageData ->
            updateCachedTaskList(requestedPage = page, pageData = pageData)
        }
    }.recoverCatching { throwable ->
        preferencesStore.readCachedVideoExtractTaskList()?.toUiPage(fromCache = true) ?: throw throwable
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载抽帧任务失败", it) },
    )

    suspend fun loadTaskDetail(taskId: String, cursor: Int = 0, pageSize: Int = 80): AppResult<VideoExtractTaskDetailResult> = runCatching {
        val response = videoExtractApiService.getTaskDetail(taskId = taskId, cursor = cursor, pageSize = pageSize)
        val root = requireEnvelopeDataObject(response, "加载任务详情失败")
        val task = root["task"]?.jsonObject?.toTaskItem(currentApiOrigin()) ?: error("任务详情缺少 task")
        val framesRoot = root["frames"]?.jsonObject ?: JsonObject(emptyMap())
        VideoExtractTaskDetailResult(
            task = task,
            frames = VideoExtractFramesUiPage(
                items = framesRoot["items"]?.jsonArray.orEmpty().mapNotNull { element ->
                    (element as? JsonObject)?.toFrameItem()
                },
                nextCursor = framesRoot.intOrDefault("nextCursor", cursor),
                hasMore = framesRoot.booleanOrDefault("hasMore", false),
            ),
        ).also { detail ->
            updateCachedTaskDetail(cursor = cursor, detail = detail)
        }
    }.recoverCatching { throwable ->
        preferencesStore.readCachedVideoExtractTaskDetail(taskId.trim())?.toUiResult(fromCache = true) ?: throw throwable
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "加载任务详情失败", it) },
    )

    suspend fun cancelTask(taskId: String): AppResult<String> = runCatching {
        val response = videoExtractApiService.cancelTask(
            buildJsonObject { put("taskId", JsonPrimitive(taskId.trim())) }
        )
        if (response.code != 0) error(response.msg ?: response.message ?: "终止任务失败")
        response.msg ?: response.message ?: "已终止任务"
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "终止任务失败", it) },
    )

    suspend fun continueTask(taskId: String, endSec: Double?, maxFrames: Int?): AppResult<String> = runCatching {
        val response = videoExtractApiService.continueTask(
            buildJsonObject {
                put("taskId", JsonPrimitive(taskId.trim()))
                endSec?.let { put("endSec", JsonPrimitive(it)) }
                maxFrames?.let { put("maxFrames", JsonPrimitive(it)) }
            }
        )
        if (response.code != 0) error(response.msg ?: response.message ?: "继续任务失败")
        response.msg ?: response.message ?: "已提交继续抽帧"
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "继续任务失败", it) },
    )

    suspend fun deleteTask(taskId: String, deleteFiles: Boolean): AppResult<String> = runCatching {
        val normalizedTaskId = taskId.trim()
        val response = videoExtractApiService.deleteTask(
            buildJsonObject {
                put("taskId", JsonPrimitive(normalizedTaskId))
                put("deleteFiles", JsonPrimitive(deleteFiles))
            }
        )
        if (response.code != 0) error(response.msg ?: response.message ?: "删除任务失败")
        removeCachedTask(normalizedTaskId)
        response.msg ?: response.message ?: if (deleteFiles) "已删除任务与输出文件" else "已删除任务记录"
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "删除任务失败", it) },
    )

    private suspend fun updateCachedTaskList(requestedPage: Int, pageData: VideoExtractTaskListPage) {
        val existingItems = if (requestedPage > 1) {
            preferencesStore.readCachedVideoExtractTaskList()?.items.orEmpty().map { it.toUiModel() }
        } else {
            emptyList()
        }
        val mergedItems = (existingItems + pageData.items)
            .distinctBy { it.taskId }
            .take(MAX_CACHED_VIDEO_EXTRACT_TASKS)
        preferencesStore.saveCachedVideoExtractTaskList(
            CachedVideoExtractTaskListSnapshot(
                items = mergedItems.map { it.toSnapshot() },
                page = pageData.page,
                pageSize = pageData.pageSize,
                total = pageData.total,
            )
        )
    }

    private suspend fun updateCachedTaskDetail(cursor: Int, detail: VideoExtractTaskDetailResult) {
        val mergedFrames = if (cursor > 0) {
            val existingFrames = preferencesStore.readCachedVideoExtractTaskDetail(detail.task.taskId)
                ?.toUiResult()
                ?.frames
                ?.items
                .orEmpty()
            (existingFrames + detail.frames.items)
                .distinctBy { it.seq }
                .take(MAX_CACHED_VIDEO_EXTRACT_FRAMES)
        } else {
            detail.frames.items.take(MAX_CACHED_VIDEO_EXTRACT_FRAMES)
        }
        val mergedDetail = detail.copy(
            frames = detail.frames.copy(items = mergedFrames)
        )
        preferencesStore.saveCachedVideoExtractTaskDetail(mergedDetail.toSnapshot())
        mergeTaskIntoCachedList(mergedDetail.task)
    }

    private suspend fun mergeTaskIntoCachedList(task: VideoExtractTaskItem) {
        val snapshot = preferencesStore.readCachedVideoExtractTaskList() ?: return
        val currentItems = snapshot.items.map { it.toUiModel() }
        val mergedItems = (listOf(task) + currentItems)
            .distinctBy { it.taskId }
            .take(MAX_CACHED_VIDEO_EXTRACT_TASKS)
        preferencesStore.saveCachedVideoExtractTaskList(
            snapshot.copy(
                items = mergedItems.map { it.toSnapshot() },
                total = snapshot.total.coerceAtLeast(mergedItems.size),
            )
        )
    }

    private suspend fun removeCachedTask(taskId: String) {
        preferencesStore.removeCachedVideoExtractTaskDetail(taskId)
        val snapshot = preferencesStore.readCachedVideoExtractTaskList() ?: return
        val filteredItems = snapshot.items.filterNot { it.taskId == taskId }
        val removedCount = snapshot.items.size - filteredItems.size
        preferencesStore.saveCachedVideoExtractTaskList(
            snapshot.copy(
                items = filteredItems,
                total = if (removedCount > 0) {
                    (snapshot.total - removedCount).coerceAtLeast(filteredItems.size)
                } else {
                    snapshot.total.coerceAtLeast(filteredItems.size)
                },
                page = if (filteredItems.isEmpty()) 1 else snapshot.page,
            )
        )
    }

    private fun CachedVideoExtractTaskListSnapshot.toUiPage(fromCache: Boolean): VideoExtractTaskListPage = VideoExtractTaskListPage(
        items = items.map { it.toUiModel() },
        page = page,
        pageSize = pageSize,
        total = total,
        fromCache = fromCache,
    )

    private fun CachedVideoExtractTaskDetailSnapshot.toUiResult(fromCache: Boolean = false): VideoExtractTaskDetailResult = VideoExtractTaskDetailResult(
        task = task.toUiModel(),
        frames = VideoExtractFramesUiPage(
            items = frames.map { it.toUiModel() },
            nextCursor = nextCursor,
            hasMore = hasMore,
        ),
        fromCache = fromCache,
    )

    private fun CachedVideoExtractTaskItemSnapshot.toUiModel(): VideoExtractTaskItem = VideoExtractTaskItem(
        taskId = taskId,
        sourceType = sourceType,
        sourceRef = sourceRef,
        sourcePreviewUrl = sourcePreviewUrl,
        outputDirLocalPath = outputDirLocalPath,
        outputDirUrl = outputDirUrl,
        outputFormat = outputFormat,
        jpgQuality = jpgQuality,
        mode = mode,
        keyframeMode = keyframeMode,
        fps = fps,
        sceneThreshold = sceneThreshold,
        startSec = startSec,
        endSec = endSec,
        maxFrames = maxFrames,
        framesExtracted = framesExtracted,
        videoWidth = videoWidth,
        videoHeight = videoHeight,
        durationSec = durationSec,
        cursorOutTimeSec = cursorOutTimeSec,
        status = status,
        stopReason = stopReason,
        lastError = lastError,
        createdAt = createdAt,
        updatedAt = updatedAt,
        runtimeLogs = runtimeLogs,
    )

    private fun VideoExtractTaskItem.toSnapshot(): CachedVideoExtractTaskItemSnapshot = CachedVideoExtractTaskItemSnapshot(
        taskId = taskId,
        sourceType = sourceType,
        sourceRef = sourceRef,
        sourcePreviewUrl = sourcePreviewUrl,
        outputDirLocalPath = outputDirLocalPath,
        outputDirUrl = outputDirUrl,
        outputFormat = outputFormat,
        jpgQuality = jpgQuality,
        mode = mode,
        keyframeMode = keyframeMode,
        fps = fps,
        sceneThreshold = sceneThreshold,
        startSec = startSec,
        endSec = endSec,
        maxFrames = maxFrames,
        framesExtracted = framesExtracted,
        videoWidth = videoWidth,
        videoHeight = videoHeight,
        durationSec = durationSec,
        cursorOutTimeSec = cursorOutTimeSec,
        status = status,
        stopReason = stopReason,
        lastError = lastError,
        createdAt = createdAt,
        updatedAt = updatedAt,
        runtimeLogs = runtimeLogs,
    )

    private fun CachedVideoExtractFrameItemSnapshot.toUiModel(): VideoExtractFrameItem = VideoExtractFrameItem(
        seq = seq,
        url = url,
    )

    private fun VideoExtractFrameItem.toSnapshot(): CachedVideoExtractFrameItemSnapshot = CachedVideoExtractFrameItemSnapshot(
        seq = seq,
        url = url,
    )

    private fun VideoExtractTaskDetailResult.toSnapshot(): CachedVideoExtractTaskDetailSnapshot = CachedVideoExtractTaskDetailSnapshot(
        task = task.toSnapshot(),
        frames = frames.items.map { it.toSnapshot() },
        nextCursor = frames.nextCursor,
        hasMore = frames.hasMore,
    )

    private fun requireEnvelopeDataObject(response: ApiEnvelope<JsonElement>, fallbackMessage: String): JsonObject {
        if (response.code != 0 && response.data == null) {
            error(response.msg ?: response.message ?: fallbackMessage)
        }
        return response.data?.jsonObject ?: error(response.msg ?: response.message ?: fallbackMessage)
    }

    private fun currentApiOrigin(): String {
        val apiBaseUrl = baseUrlProvider.currentApiBaseUrl()
        val isDefaultPort = (apiBaseUrl.isHttps && apiBaseUrl.port == 443) || (!apiBaseUrl.isHttps && apiBaseUrl.port == 80)
        val portSuffix = if (isDefaultPort) "" else ":${apiBaseUrl.port}"
        return "${apiBaseUrl.scheme}://${apiBaseUrl.host}$portSuffix"
    }
}

@HiltViewModel
class VideoExtractTaskCenterViewModel @Inject constructor(
    private val repository: VideoExtractTaskCenterRepository,
) : ViewModel() {
    var uiState by mutableStateOf(VideoExtractTaskCenterUiState())
        private set

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            loadTasksInternal(page = 1, preserveSelection = true)
        }
    }

    fun loadMoreTasks() {
        if (uiState.listLoading) return
        val nextPage = uiState.page + 1
        val loadedCount = uiState.tasks.size
        if (loadedCount >= uiState.total && uiState.total > 0) return
        viewModelScope.launch {
            when (val result = repository.loadTasks(page = nextPage, pageSize = uiState.pageSize)) {
                is AppResult.Success -> {
                    val page = result.data
                    uiState = uiState.copy(
                        tasks = (uiState.tasks + page.items).distinctBy { it.taskId },
                        page = page.page,
                        pageSize = page.pageSize,
                        total = page.total,
                        message = if (page.fromCache) uiState.message ?: "网络不可用，已展示最近缓存的抽帧任务" else uiState.message,
                    )
                }
                is AppResult.Error -> uiState = uiState.copy(message = result.message)
            }
        }
    }

    fun toggleTask(taskId: String) {
        if (uiState.selectedTaskId == taskId) {
            uiState = uiState.copy(
                selectedTaskId = null,
                selectedTask = null,
                frames = VideoExtractFramesUiPage(),
                continueEndSec = "",
                continueMaxFrames = "",
            )
            return
        }
        viewModelScope.launch {
            uiState = uiState.copy(selectedTaskId = taskId, detailLoading = true, frames = VideoExtractFramesUiPage())
            when (val result = repository.loadTaskDetail(taskId = taskId, cursor = 0)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        detailLoading = false,
                        selectedTask = result.data.task,
                        frames = result.data.frames,
                        continueEndSec = "",
                        continueMaxFrames = "",
                        message = if (result.data.fromCache) uiState.message ?: "网络不可用，已展示最近缓存的任务详情" else uiState.message,
                    )
                    mergeSelectedTaskIntoList(result.data.task)
                }
                is AppResult.Error -> {
                    uiState = uiState.copy(detailLoading = false, message = result.message)
                }
            }
        }
    }

    fun loadMoreFrames() {
        val taskId = uiState.selectedTaskId ?: return
        if (uiState.framesLoadingMore || !uiState.frames.hasMore) return
        viewModelScope.launch {
            uiState = uiState.copy(framesLoadingMore = true)
            when (val result = repository.loadTaskDetail(taskId = taskId, cursor = uiState.frames.nextCursor)) {
                is AppResult.Success -> {
                    val mergedFrames = (uiState.frames.items + result.data.frames.items).distinctBy { it.seq }
                    uiState = uiState.copy(
                        framesLoadingMore = false,
                        selectedTask = result.data.task,
                        frames = result.data.frames.copy(items = mergedFrames),
                        message = if (result.data.fromCache) uiState.message ?: "网络不可用，已展示最近缓存的任务详情" else uiState.message,
                    )
                    mergeSelectedTaskIntoList(result.data.task)
                }
                is AppResult.Error -> uiState = uiState.copy(framesLoadingMore = false, message = result.message)
            }
        }
    }

    fun refreshSelectedTask() {
        val taskId = uiState.selectedTaskId ?: return
        viewModelScope.launch {
            uiState = uiState.copy(detailLoading = true)
            when (val result = repository.loadTaskDetail(taskId = taskId, cursor = 0)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        detailLoading = false,
                        selectedTask = result.data.task,
                        frames = result.data.frames,
                        message = if (result.data.fromCache) uiState.message ?: "网络不可用，已展示最近缓存的任务详情" else uiState.message,
                    )
                    mergeSelectedTaskIntoList(result.data.task)
                }
                is AppResult.Error -> uiState = uiState.copy(detailLoading = false, message = result.message)
            }
        }
    }

    fun updateContinueEndSec(value: String) {
        uiState = uiState.copy(continueEndSec = value)
    }

    fun updateContinueMaxFrames(value: String) {
        uiState = uiState.copy(continueMaxFrames = value)
    }

    fun updateDeleteFiles(value: Boolean) {
        uiState = uiState.copy(deleteFiles = value)
    }

    fun cancelSelectedTask() {
        val task = uiState.selectedTask ?: return
        if (!task.canCancel()) return
        viewModelScope.launch {
            uiState = uiState.copy(actionLoading = true, message = null)
            when (val result = repository.cancelTask(task.taskId)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(actionLoading = false, message = result.data)
                    refreshAfterAction(task.taskId)
                }
                is AppResult.Error -> uiState = uiState.copy(actionLoading = false, message = result.message)
            }
        }
    }

    fun continueSelectedTask() {
        val task = uiState.selectedTask ?: return
        if (!task.canContinue()) return
        val endValue = uiState.continueEndSec.trim().takeIf { it.isNotBlank() }?.toDoubleOrNull()
        if (uiState.continueEndSec.isNotBlank() && endValue == null) {
            uiState = uiState.copy(message = "继续 endSec 格式非法")
            return
        }
        val maxFramesValue = uiState.continueMaxFrames.trim().takeIf { it.isNotBlank() }?.toIntOrNull()
        if (uiState.continueMaxFrames.isNotBlank() && maxFramesValue == null) {
            uiState = uiState.copy(message = "继续 maxFrames 格式非法")
            return
        }
        viewModelScope.launch {
            uiState = uiState.copy(actionLoading = true, message = null)
            when (val result = repository.continueTask(task.taskId, endValue, maxFramesValue)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        actionLoading = false,
                        continueEndSec = "",
                        continueMaxFrames = "",
                        message = result.data,
                    )
                    refreshAfterAction(task.taskId)
                }
                is AppResult.Error -> uiState = uiState.copy(actionLoading = false, message = result.message)
            }
        }
    }

    fun deleteSelectedTask() {
        val task = uiState.selectedTask ?: return
        viewModelScope.launch {
            uiState = uiState.copy(actionLoading = true, message = null)
            when (val result = repository.deleteTask(task.taskId, uiState.deleteFiles)) {
                is AppResult.Success -> {
                    uiState = uiState.copy(
                        actionLoading = false,
                        message = result.data,
                        selectedTaskId = null,
                        selectedTask = null,
                        frames = VideoExtractFramesUiPage(),
                        continueEndSec = "",
                        continueMaxFrames = "",
                    )
                    loadTasksInternal(page = 1, preserveSelection = false, clearMessage = false)
                }
                is AppResult.Error -> uiState = uiState.copy(actionLoading = false, message = result.message)
            }
        }
    }

    fun consumeMessage() {
        if (uiState.message != null) {
            uiState = uiState.copy(message = null)
        }
    }

    private suspend fun refreshAfterAction(taskId: String) {
        loadTasksInternal(page = 1, preserveSelection = false, clearMessage = false)
        when (val detail = repository.loadTaskDetail(taskId = taskId, cursor = 0)) {
            is AppResult.Success -> {
                uiState = uiState.copy(
                    selectedTaskId = taskId,
                    selectedTask = detail.data.task,
                    frames = detail.data.frames,
                    detailLoading = false,
                    message = if (detail.data.fromCache) uiState.message ?: "网络不可用，已展示最近缓存的任务详情" else uiState.message,
                )
                mergeSelectedTaskIntoList(detail.data.task)
            }
            is AppResult.Error -> uiState = uiState.copy(message = detail.message)
        }
    }

    private suspend fun loadTasksInternal(page: Int, preserveSelection: Boolean, clearMessage: Boolean = true) {
        uiState = uiState.copy(listLoading = true, message = if (clearMessage) null else uiState.message)
        when (val result = repository.loadTasks(page = page, pageSize = uiState.pageSize)) {
            is AppResult.Success -> {
                val taskPage = result.data
                val selectedTaskId = if (preserveSelection) uiState.selectedTaskId else null
                uiState = uiState.copy(
                    listLoading = false,
                    tasks = taskPage.items,
                    page = taskPage.page,
                    pageSize = taskPage.pageSize,
                    total = taskPage.total,
                    selectedTaskId = selectedTaskId,
                    message = if (taskPage.fromCache) uiState.message ?: "网络不可用，已展示最近缓存的抽帧任务" else if (clearMessage) null else uiState.message,
                )
                if (selectedTaskId != null && taskPage.items.any { it.taskId == selectedTaskId }) {
                    uiState = uiState.copy(selectedTask = taskPage.items.firstOrNull { it.taskId == selectedTaskId })
                } else if (!preserveSelection) {
                    uiState = uiState.copy(selectedTask = null, frames = VideoExtractFramesUiPage())
                }
            }
            is AppResult.Error -> uiState = uiState.copy(listLoading = false, message = result.message)
        }
    }

    private fun mergeSelectedTaskIntoList(task: VideoExtractTaskItem) {
        uiState = uiState.copy(
            tasks = uiState.tasks.map { existing -> if (existing.taskId == task.taskId) task else existing },
            selectedTask = task,
        )
    }
}

@Composable
fun VideoExtractTaskCenterScreen(
    onBack: () -> Unit,
    onOpenCreate: () -> Unit,
    viewModel: VideoExtractTaskCenterViewModel,
) {
    val state = viewModel.uiState
    val snackbarHostState = remember { SnackbarHostState() }
    val context = LocalContext.current

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("抽帧任务中心") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Text(
                        text = "查看抽帧任务列表、详情与帧结果，并支持终止 / 继续 / 删除任务。",
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Button(onClick = onOpenCreate, modifier = Modifier.weight(1f)) {
                            Text("新建任务")
                        }
                        OutlinedButton(onClick = viewModel::refresh, modifier = Modifier.weight(1f)) {
                            Text("刷新列表")
                        }
                    }
                    Text(
                        text = "总任务数：${state.total}",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }

            if (state.listLoading && state.tasks.isEmpty()) {
                item {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp),
                        horizontalArrangement = Arrangement.Center,
                    ) {
                        CircularProgressIndicator()
                    }
                }
            }

            if (!state.listLoading && state.tasks.isEmpty()) {
                item {
                    VideoExtractTaskCardContainer {
                        Text("暂无抽帧任务，可先新建一个视频抽帧任务。")
                    }
                }
            }

            items(state.tasks, key = { it.taskId }) { task ->
                Column(
                    modifier = Modifier.padding(horizontal = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    VideoExtractTaskSummaryCard(
                        task = task,
                        selected = state.selectedTaskId == task.taskId,
                        onToggle = { viewModel.toggleTask(task.taskId) },
                    )
                    if (state.selectedTaskId == task.taskId) {
                        when {
                            state.detailLoading -> {
                                VideoExtractTaskCardContainer {
                                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                        CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                                        Text("正在加载任务详情...")
                                    }
                                }
                            }
                            state.selectedTask != null -> {
                                VideoExtractTaskDetailCard(
                                    task = state.selectedTask,
                                    frames = state.frames,
                                    actionLoading = state.actionLoading,
                                    continueEndSec = state.continueEndSec,
                                    continueMaxFrames = state.continueMaxFrames,
                                    deleteFiles = state.deleteFiles,
                                    onRefresh = viewModel::refreshSelectedTask,
                                    onOpenSource = { url -> openUrlExternally(context, url) },
                                    onOpenFrame = { url -> openUrlExternally(context, url) },
                                    onUpdateContinueEndSec = viewModel::updateContinueEndSec,
                                    onUpdateContinueMaxFrames = viewModel::updateContinueMaxFrames,
                                    onUpdateDeleteFiles = viewModel::updateDeleteFiles,
                                    onCancel = viewModel::cancelSelectedTask,
                                    onContinue = viewModel::continueSelectedTask,
                                    onDelete = viewModel::deleteSelectedTask,
                                    onLoadMoreFrames = viewModel::loadMoreFrames,
                                    framesLoadingMore = state.framesLoadingMore,
                                )
                            }
                        }
                    }
                }
            }

            if (state.tasks.isNotEmpty() && state.tasks.size < state.total) {
                item {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.Center,
                    ) {
                        OutlinedButton(onClick = viewModel::loadMoreTasks) {
                            Text("加载更多任务")
                        }
                    }
                }
            }

            item {
                androidx.compose.foundation.layout.Spacer(modifier = Modifier.height(8.dp))
            }
        }
    }
}

@Composable
private fun VideoExtractTaskCardContainer(content: @Composable () -> Unit) {
    androidx.compose.material3.Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            content()
        }
    }
}

@Composable
private fun VideoExtractTaskSummaryCard(
    task: VideoExtractTaskItem,
    selected: Boolean,
    onToggle: () -> Unit,
) {
    VideoExtractTaskCardContainer {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(
                    text = task.displayTitle(),
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = task.displaySubtitle(),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            }
            Text(
                text = task.statusText(),
                style = MaterialTheme.typography.labelLarge,
                color = task.statusColor(),
            )
        }
        Text(
            text = "模式：${task.modeText()} · 输出：${task.framesExtracted}/${task.maxFrames} 张 · ${task.progressText()}",
            style = MaterialTheme.typography.bodySmall,
        )
        OutlinedButton(onClick = onToggle, modifier = Modifier.fillMaxWidth()) {
            Text(if (selected) "收起详情" else "查看详情")
        }
    }
}

@Composable
private fun VideoExtractTaskDetailCard(
    task: VideoExtractTaskItem,
    frames: VideoExtractFramesUiPage,
    actionLoading: Boolean,
    continueEndSec: String,
    continueMaxFrames: String,
    deleteFiles: Boolean,
    onRefresh: () -> Unit,
    onOpenSource: (String) -> Unit,
    onOpenFrame: (String) -> Unit,
    onUpdateContinueEndSec: (String) -> Unit,
    onUpdateContinueMaxFrames: (String) -> Unit,
    onUpdateDeleteFiles: (Boolean) -> Unit,
    onCancel: () -> Unit,
    onContinue: () -> Unit,
    onDelete: () -> Unit,
    onLoadMoreFrames: () -> Unit,
    framesLoadingMore: Boolean,
) {
    VideoExtractTaskCardContainer {
        Text("任务 ID：${task.taskId}", style = MaterialTheme.typography.titleSmall)
        Text("来源：${task.sourceType} · ${task.sourceRef.ifBlank { "-" }}", style = MaterialTheme.typography.bodySmall)
        Text("输出目录：${task.outputDirLocalPath.ifBlank { task.outputDirUrl.ifBlank { "-" } }}", style = MaterialTheme.typography.bodySmall)
        Text("尺寸：${task.videoWidth} × ${task.videoHeight} · 时长：${formatDuration(task.durationSec)}", style = MaterialTheme.typography.bodySmall)
        Text("模式：${task.modeText()}", style = MaterialTheme.typography.bodySmall)
        Text("范围：${task.limitText()}", style = MaterialTheme.typography.bodySmall)
        Text("状态：${task.statusText()} · ${task.progressText()}", style = MaterialTheme.typography.bodySmall)
        if (task.lastError.isNotBlank()) {
            Text(
                text = "错误：${task.lastError}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            OutlinedButton(onClick = onRefresh, enabled = !actionLoading, modifier = Modifier.weight(1f)) {
                Text("刷新详情")
            }
            if (task.sourcePreviewUrl.isNotBlank()) {
                OutlinedButton(onClick = { onOpenSource(task.sourcePreviewUrl) }, enabled = !actionLoading, modifier = Modifier.weight(1f)) {
                    Text("源视频")
                }
            }
        }
        if (task.canCancel()) {
            Button(onClick = onCancel, enabled = !actionLoading, modifier = Modifier.fillMaxWidth()) {
                Text(if (actionLoading) "处理中..." else "终止任务")
            }
        }
        if (task.canContinue()) {
            OutlinedTextField(
                modifier = Modifier.fillMaxWidth(),
                value = continueEndSec,
                onValueChange = onUpdateContinueEndSec,
                label = { Text("继续到结束秒（可空）") },
                singleLine = true,
                enabled = !actionLoading,
            )
            OutlinedTextField(
                modifier = Modifier.fillMaxWidth(),
                value = continueMaxFrames,
                onValueChange = onUpdateContinueMaxFrames,
                label = { Text("新的最大帧数（可空）") },
                singleLine = true,
                enabled = !actionLoading,
            )
            Button(onClick = onContinue, enabled = !actionLoading, modifier = Modifier.fillMaxWidth()) {
                Text(if (actionLoading) "处理中..." else "继续抽帧")
            }
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text("删除时同时删除输出文件", style = MaterialTheme.typography.bodySmall)
                Text(
                    text = if (deleteFiles) "当前会删除任务记录与输出目录内容" else "当前仅删除任务记录",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.outline,
                )
            }
            Switch(checked = deleteFiles, onCheckedChange = onUpdateDeleteFiles, enabled = !actionLoading)
        }
        OutlinedButton(onClick = onDelete, enabled = !actionLoading, modifier = Modifier.fillMaxWidth()) {
            Text(if (actionLoading) "处理中..." else "删除任务")
        }
        if (task.runtimeLogs.isNotEmpty()) {
            Text("最近日志", style = MaterialTheme.typography.titleSmall)
            task.runtimeLogs.takeLast(8).forEach { line ->
                Text(line, style = MaterialTheme.typography.bodySmall)
            }
        }
        Text("帧结果（${frames.items.size}${if (frames.hasMore) "+" else ""}）", style = MaterialTheme.typography.titleSmall)
        if (frames.items.isEmpty()) {
            Text("暂无已生成帧。", style = MaterialTheme.typography.bodySmall)
        } else {
            LazyRow(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                items(frames.items, key = { it.seq }) { frame ->
                    androidx.compose.material3.Card(
                        modifier = Modifier
                            .fillParentMaxWidth(0.56f)
                            .clickable { onOpenFrame(frame.url) },
                    ) {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            AsyncImage(
                                model = frame.url,
                                contentDescription = "帧 ${frame.seq}",
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(140.dp),
                                contentScale = ContentScale.Crop,
                            )
                            Column(modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp)) {
                                Text("帧 #${frame.seq}", style = MaterialTheme.typography.titleSmall)
                                Text(
                                    text = frame.url,
                                    style = MaterialTheme.typography.bodySmall,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                        }
                    }
                }
            }
            if (frames.hasMore) {
                OutlinedButton(onClick = onLoadMoreFrames, enabled = !framesLoadingMore, modifier = Modifier.fillMaxWidth()) {
                    Text(if (framesLoadingMore) "加载中..." else "加载更多帧")
                }
            }
        }
    }
}

private fun VideoExtractTaskItem.displayTitle(): String = when {
    sourceRef.isNotBlank() -> sourceRef.substringAfterLast('/').ifBlank { taskId }
    else -> taskId
}

private fun VideoExtractTaskItem.displaySubtitle(): String = listOf(
    createdAt.substringBefore('T').takeIf { it.isNotBlank() },
    "${videoWidth}×${videoHeight}".takeIf { videoWidth > 0 && videoHeight > 0 },
    formatDuration(durationSec).takeIf { it != "-" },
).joinToString(separator = " · ")

private fun VideoExtractTaskItem.statusText(): String = when (status.uppercase(Locale.ROOT)) {
    "PENDING" -> "排队中"
    "PREPARING" -> "准备中"
    "RUNNING" -> "运行中"
    "PAUSED_USER" -> "已终止"
    "PAUSED_LIMIT" -> "因限制暂停"
    "FINISHED" -> "已完成"
    "FAILED" -> "失败"
    else -> status.ifBlank { "未知" }
}

@Composable
private fun VideoExtractTaskItem.statusColor() = when (status.uppercase(Locale.ROOT)) {
    "RUNNING" -> MaterialTheme.colorScheme.primary
    "PAUSED_LIMIT" -> MaterialTheme.colorScheme.tertiary
    "FAILED" -> MaterialTheme.colorScheme.error
    else -> MaterialTheme.colorScheme.outline
}

private fun VideoExtractTaskItem.modeText(): String = when (mode.lowercase(Locale.ROOT)) {
    "fps" -> "固定 FPS ${fps ?: ""}".trim()
    "all" -> "逐帧输出"
    "keyframe" -> if (keyframeMode.equals("scene", ignoreCase = true)) {
        "关键帧(场景 ${sceneThreshold ?: ""})".trim()
    } else {
        "关键帧(I 帧)"
    }
    else -> mode
}

private fun VideoExtractTaskItem.progressText(): String {
    val duration = durationSec
    val cursor = cursorOutTimeSec
    return if (duration != null && duration > 0 && cursor != null && cursor > 0) {
        val percent = ((cursor / duration) * 100.0).coerceIn(0.0, 100.0)
        "进度 ${String.format(Locale.US, "%.0f", percent)}%"
    } else {
        "已输出 ${framesExtracted}/${maxFrames} 张"
    }
}

private fun VideoExtractTaskItem.limitText(): String {
    val range = when {
        startSec != null || endSec != null -> "${startSec ?: 0.0} ~ ${endSec ?: "end"}s"
        else -> "全程"
    }
    return "时间区间：$range；maxFrames=$maxFrames；格式=${outputFormat.uppercase(Locale.ROOT)}${jpgQuality?.let { " / Q=$it" } ?: ""}"
}

private fun VideoExtractTaskItem.canCancel(): Boolean = status.uppercase(Locale.ROOT) in setOf("PENDING", "PREPARING", "RUNNING")

private fun VideoExtractTaskItem.canContinue(): Boolean = status.uppercase(Locale.ROOT) in setOf("PAUSED_LIMIT", "PAUSED_USER")

private fun JsonObject.toTaskItem(origin: String): VideoExtractTaskItem? {
    val taskId = stringOrNull("taskId") ?: return null
    val sourceType = stringOrNull("sourceType").orEmpty()
    val sourceRef = stringOrNull("sourceRef").orEmpty()
    val runtimeLogs = (this["runtime"] as? JsonObject)?.get("logs")?.jsonArray.orEmpty().mapNotNull {
        runCatching { it.jsonPrimitive.contentOrNull ?: it.jsonPrimitive.content }.getOrNull()
    }
    return VideoExtractTaskItem(
        taskId = taskId,
        sourceType = sourceType,
        sourceRef = sourceRef,
        sourcePreviewUrl = buildUploadPreviewUrl(origin, sourceType, sourceRef),
        outputDirLocalPath = stringOrNull("outputDirLocalPath").orEmpty(),
        outputDirUrl = stringOrNull("outputDirUrl").orEmpty(),
        outputFormat = stringOrNull("outputFormat").orEmpty(),
        jpgQuality = intOrNull("jpgQuality"),
        mode = stringOrNull("mode").orEmpty(),
        keyframeMode = stringOrNull("keyframeMode"),
        fps = doubleOrNull("fps"),
        sceneThreshold = doubleOrNull("sceneThreshold"),
        startSec = doubleOrNull("startSec"),
        endSec = doubleOrNull("endSec"),
        maxFrames = intOrDefault("maxFrames", 0),
        framesExtracted = intOrDefault("framesExtracted", 0),
        videoWidth = intOrDefault("videoWidth", 0),
        videoHeight = intOrDefault("videoHeight", 0),
        durationSec = doubleOrNull("durationSec"),
        cursorOutTimeSec = doubleOrNull("cursorOutTimeSec"),
        status = stringOrNull("status").orEmpty(),
        stopReason = stringOrNull("stopReason").orEmpty(),
        lastError = stringOrNull("lastError").orEmpty(),
        createdAt = stringOrNull("createdAt").orEmpty(),
        updatedAt = stringOrNull("updatedAt").orEmpty(),
        runtimeLogs = runtimeLogs,
    )
}

private fun JsonObject.toFrameItem(): VideoExtractFrameItem? {
    val seq = intOrNull("seq") ?: return null
    val url = stringOrNull("url") ?: return null
    return VideoExtractFrameItem(seq = seq, url = url)
}

private fun JsonObject.intOrNull(key: String): Int? = stringOrNull(key)?.toIntOrNull()
private fun JsonObject.intOrDefault(key: String, defaultValue: Int): Int = intOrNull(key) ?: defaultValue
private fun JsonObject.doubleOrNull(key: String): Double? = stringOrNull(key)?.toDoubleOrNull()
private fun JsonObject.booleanOrDefault(key: String, defaultValue: Boolean): Boolean =
    this[key]?.jsonPrimitive?.booleanOrNull ?: defaultValue

private fun buildUploadPreviewUrl(origin: String, sourceType: String, sourceRef: String): String {
    if (!sourceType.equals("upload", ignoreCase = true)) return ""
    var path = sourceRef.trim()
    if (path.isBlank()) return ""
    if (path.startsWith("http://") || path.startsWith("https://")) return path
    if (path.startsWith("/upload/")) return origin + path
    if (!path.startsWith('/')) path = "/$path"
    return origin + "/upload" + path
}

private fun openUrlExternally(context: Context, rawUrl: String) {
    val url = rawUrl.trim()
    if (url.isBlank()) return
    runCatching {
        context.startActivity(
            Intent(Intent.ACTION_VIEW, Uri.parse(url)).apply {
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
        )
    }.onFailure {
        android.widget.Toast.makeText(context, "无法打开: $url", android.widget.Toast.LENGTH_SHORT).show()
    }
}

private fun formatDuration(seconds: Double?): String {
    if (seconds == null || seconds <= 0) return "-"
    val total = kotlin.math.round(seconds).toLong()
    val h = total / 3600
    val m = (total % 3600) / 60
    val s = total % 60
    return if (h > 0) {
        String.format(Locale.US, "%d:%02d:%02d", h, m, s)
    } else {
        String.format(Locale.US, "%d:%02d", m, s)
    }
}
