/*
 * 视频抽帧创建页对齐 Web 端最小可用流程：本地选择视频、上传、probe，再创建抽帧任务。
 */
@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package io.github.a7413498.liao.android.feature.videoextract

import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
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
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import io.github.a7413498.liao.android.core.common.AppResult
import io.github.a7413498.liao.android.core.datastore.AppPreferencesStore
import io.github.a7413498.liao.android.core.network.ApiEnvelope
import io.github.a7413498.liao.android.core.network.VideoExtractApiService
import io.github.a7413498.liao.android.core.network.stringOrNull
import java.net.URLConnection
import java.util.Locale
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody
import okio.source

enum class VideoExtractModeOption {
    KEYFRAME,
    FPS,
    ALL;

    fun apiValue(): String = when (this) {
        KEYFRAME -> "keyframe"
        FPS -> "fps"
        ALL -> "all"
    }
}

enum class VideoExtractKeyframeModeOption {
    IFRAME,
    SCENE;

    fun apiValue(): String = when (this) {
        IFRAME -> "iframe"
        SCENE -> "scene"
    }
}

enum class VideoExtractOutputFormatOption {
    JPG,
    PNG;

    fun apiValue(): String = when (this) {
        JPG -> "jpg"
        PNG -> "png"
    }
}

data class VideoExtractUploadedSource(
    val displayName: String,
    val mimeType: String,
    val size: Long,
    val localPath: String,
    val localFilename: String,
    val originalFilename: String,
)

data class VideoExtractProbeSummary(
    val durationSec: Double,
    val width: Int,
    val height: Int,
    val avgFps: Double?,
)

data class VideoExtractCreatedTask(
    val taskId: String,
    val probe: VideoExtractProbeSummary?,
)

data class VideoExtractCreatePayload(
    val mode: VideoExtractModeOption,
    val keyframeMode: VideoExtractKeyframeModeOption,
    val sceneThreshold: Double?,
    val fps: Double?,
    val startSec: Double?,
    val endSec: Double?,
    val maxFrames: Int,
    val outputFormat: VideoExtractOutputFormatOption,
    val jpgQuality: Int?,
)

data class VideoExtractCreateUiState(
    val uploading: Boolean = false,
    val probing: Boolean = false,
    val creating: Boolean = false,
    val source: VideoExtractUploadedSource? = null,
    val probe: VideoExtractProbeSummary? = null,
    val probeError: String? = null,
    val createdTask: VideoExtractCreatedTask? = null,
    val mode: VideoExtractModeOption = VideoExtractModeOption.KEYFRAME,
    val keyframeMode: VideoExtractKeyframeModeOption = VideoExtractKeyframeModeOption.IFRAME,
    val sceneThreshold: String = "0.3",
    val fps: String = "1",
    val startSec: String = "",
    val endSec: String = "",
    val maxFrames: String = "500",
    val outputFormat: VideoExtractOutputFormatOption = VideoExtractOutputFormatOption.JPG,
    val jpgQuality: String = "",
    val message: String? = null,
)

class VideoExtractCreateRepository @Inject constructor(
    private val videoExtractApiService: VideoExtractApiService,
    private val preferencesStore: AppPreferencesStore,
    @ApplicationContext private val appContext: Context,
) {
    private fun queryFileMeta(uri: Uri): Triple<String, String, Long> {
        val contentResolver = appContext.contentResolver
        var displayName = "video_${System.currentTimeMillis()}.mp4"
        var size = -1L
        contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE), null, null, null)?.use { cursor ->
            if (cursor.moveToFirst()) {
                val nameIndex = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                val sizeIndex = cursor.getColumnIndex(OpenableColumns.SIZE)
                if (nameIndex >= 0) displayName = cursor.getString(nameIndex) ?: displayName
                if (sizeIndex >= 0) size = cursor.getLong(sizeIndex)
            }
        }
        val mimeType = contentResolver.getType(uri)
            ?: URLConnection.guessContentTypeFromName(displayName)
            ?: "application/octet-stream"
        return Triple(displayName, mimeType, size)
    }

    private fun buildMultipartFilePart(uri: Uri, fileName: String, mimeType: String, size: Long): MultipartBody.Part {
        val contentResolver = appContext.contentResolver
        val requestBody = object : RequestBody() {
            override fun contentType() = mimeType.toMediaType()

            override fun contentLength(): Long = if (size >= 0L) size else -1L

            override fun writeTo(sink: okio.BufferedSink) {
                contentResolver.openInputStream(uri)?.use { inputStream ->
                    sink.writeAll(inputStream.source())
                } ?: error("无法读取所选视频")
            }
        }
        return MultipartBody.Part.createFormData("file", fileName, requestBody)
    }

    suspend fun uploadVideo(uri: Uri): AppResult<VideoExtractUploadedSource> = runCatching {
        val (displayName, mimeType, size) = queryFileMeta(uri)
        val response = videoExtractApiService.uploadVideoExtractInput(
            file = buildMultipartFilePart(uri, displayName, mimeType, size),
        )
        val root = requireEnvelopeDataObject(response, "上传视频失败")
        VideoExtractUploadedSource(
            displayName = displayName,
            mimeType = root.stringOrNull("contentType") ?: mimeType,
            size = root.longOrDefault("fileSize", size),
            localPath = root.stringOrNull("localPath") ?: error("上传结果缺少 localPath"),
            localFilename = root.stringOrNull("localFilename").orEmpty().ifBlank { displayName },
            originalFilename = root.stringOrNull("originalFilename").orEmpty().ifBlank { displayName },
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "上传视频失败", it) },
    )

    suspend fun probeUpload(localPath: String): AppResult<VideoExtractProbeSummary> = runCatching {
        val response = videoExtractApiService.probeVideo(sourceType = "upload", localPath = localPath)
        val root = requireEnvelopeDataObject(response, "视频探测失败")
        root.toProbeSummary()
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "视频探测失败", it) },
    )

    suspend fun createTask(localPath: String, payload: VideoExtractCreatePayload): AppResult<VideoExtractCreatedTask> = runCatching {
        val userId = preferencesStore.readCurrentSession()?.id.orEmpty()
        val response = videoExtractApiService.createTask(
            buildJsonObject {
                if (userId.isNotBlank()) put("userId", JsonPrimitive(userId))
                put("sourceType", JsonPrimitive("upload"))
                put("localPath", JsonPrimitive(localPath))
                put("mode", JsonPrimitive(payload.mode.apiValue()))
                if (payload.mode == VideoExtractModeOption.KEYFRAME) {
                    put("keyframeMode", JsonPrimitive(payload.keyframeMode.apiValue()))
                    payload.sceneThreshold?.let { put("sceneThreshold", JsonPrimitive(it)) }
                }
                if (payload.mode == VideoExtractModeOption.FPS) {
                    payload.fps?.let { put("fps", JsonPrimitive(it)) }
                }
                payload.startSec?.let { put("startSec", JsonPrimitive(it)) }
                payload.endSec?.let { put("endSec", JsonPrimitive(it)) }
                put("maxFrames", JsonPrimitive(payload.maxFrames))
                put("outputFormat", JsonPrimitive(payload.outputFormat.apiValue()))
                if (payload.outputFormat == VideoExtractOutputFormatOption.JPG) {
                    payload.jpgQuality?.let { put("jpgQuality", JsonPrimitive(it)) }
                }
            }
        )
        val root = requireEnvelopeDataObject(response, "创建抽帧任务失败")
        VideoExtractCreatedTask(
            taskId = root.stringOrNull("taskId") ?: error("创建结果缺少 taskId"),
            probe = (root["probe"] as? JsonObject)?.toProbeSummary(),
        )
    }.fold(
        onSuccess = { AppResult.Success(it) },
        onFailure = { AppResult.Error(it.message ?: "创建抽帧任务失败", it) },
    )

    private fun requireEnvelopeDataObject(response: ApiEnvelope<JsonElement>, fallbackMessage: String): JsonObject {
        if (response.code != 0 && response.data == null) {
            error(response.msg ?: response.message ?: fallbackMessage)
        }
        return response.data?.jsonObject ?: error(response.msg ?: response.message ?: fallbackMessage)
    }
}

@HiltViewModel
class VideoExtractCreateViewModel @Inject constructor(
    private val repository: VideoExtractCreateRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow(VideoExtractCreateUiState())
    val uiState: StateFlow<VideoExtractCreateUiState> = _uiState.asStateFlow()

    fun uploadSource(uri: Uri) {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(
                uploading = true,
                probing = false,
                creating = false,
                source = null,
                probe = null,
                probeError = null,
                createdTask = null,
                message = null,
            )
            when (val result = repository.uploadVideo(uri)) {
                is AppResult.Success -> {
                    _uiState.value = _uiState.value.copy(
                        uploading = false,
                        source = result.data,
                        message = "视频已上传，正在探测信息",
                    )
                    refreshProbe(autoMessage = false)
                }
                is AppResult.Error -> {
                    _uiState.value = _uiState.value.copy(uploading = false, message = result.message)
                }
            }
        }
    }

    fun refreshProbe(autoMessage: Boolean = true) {
        val source = _uiState.value.source ?: run {
            if (autoMessage) {
                _uiState.value = _uiState.value.copy(message = "请先选择本地视频")
            }
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(probing = true, probeError = null, message = null)
            when (val result = repository.probeUpload(source.localPath)) {
                is AppResult.Success -> {
                    _uiState.value = _uiState.value.copy(
                        probing = false,
                        probe = result.data,
                        probeError = null,
                        message = if (autoMessage) "视频信息已更新" else null,
                    )
                }
                is AppResult.Error -> {
                    _uiState.value = _uiState.value.copy(
                        probing = false,
                        probe = null,
                        probeError = result.message,
                        message = result.message,
                    )
                }
            }
        }
    }

    fun updateMode(value: VideoExtractModeOption) {
        _uiState.value = _uiState.value.copy(mode = value)
    }

    fun updateKeyframeMode(value: VideoExtractKeyframeModeOption) {
        _uiState.value = _uiState.value.copy(keyframeMode = value)
    }

    fun updateSceneThreshold(value: String) {
        _uiState.value = _uiState.value.copy(sceneThreshold = value)
    }

    fun updateFps(value: String) {
        _uiState.value = _uiState.value.copy(fps = value)
    }

    fun updateStartSec(value: String) {
        _uiState.value = _uiState.value.copy(startSec = value)
    }

    fun updateEndSec(value: String) {
        _uiState.value = _uiState.value.copy(endSec = value)
    }

    fun updateMaxFrames(value: String) {
        _uiState.value = _uiState.value.copy(maxFrames = value)
    }

    fun updateOutputFormat(value: VideoExtractOutputFormatOption) {
        _uiState.value = _uiState.value.copy(outputFormat = value)
    }

    fun updateJpgQuality(value: String) {
        _uiState.value = _uiState.value.copy(jpgQuality = value)
    }

    fun createTask() {
        val state = _uiState.value
        val source = state.source ?: run {
            _uiState.value = state.copy(message = "请先选择本地视频")
            return
        }
        val payload = state.toCreatePayloadOrError()
        if (payload is AppResult.Error) {
            _uiState.value = state.copy(message = payload.message)
            return
        }
        payload as AppResult.Success
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(creating = true, message = null)
            when (val result = repository.createTask(localPath = source.localPath, payload = payload.data)) {
                is AppResult.Success -> {
                    _uiState.value = _uiState.value.copy(
                        creating = false,
                        createdTask = result.data,
                        probe = result.data.probe ?: _uiState.value.probe,
                        message = "抽帧任务已创建：${result.data.taskId}",
                    )
                }
                is AppResult.Error -> {
                    _uiState.value = _uiState.value.copy(creating = false, message = result.message)
                }
            }
        }
    }

    fun consumeMessage() {
        if (_uiState.value.message != null) {
            _uiState.value = _uiState.value.copy(message = null)
        }
    }
}

@Composable
private fun VideoExtractOptionButton(
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
private fun VideoExtractSection(
    title: String,
    content: @Composable () -> Unit,
) {
    androidx.compose.material3.Card(modifier = Modifier.fillMaxWidth()) {
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
fun VideoExtractCreateScreen(
    onBack: () -> Unit,
    onOpenTaskCenter: () -> Unit,
    viewModel: VideoExtractCreateViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }
    val scrollState = rememberScrollState()
    val context = LocalContext.current
    val pickerLauncher = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        uri?.let(viewModel::uploadSource)
    }
    val validationError = remember(state) { state.validationError() }
    val estimatedFrames = remember(state) { estimateFrames(state) }

    LaunchedEffect(state.message) {
        state.message?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.consumeMessage()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("视频抽帧") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Outlined.ArrowBack, contentDescription = "返回")
                    }
                },
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
            VideoExtractSection(title = "视频来源") {
                Text(
                    text = "先选择本地视频，Android 会自动上传到 /api/uploadVideoExtractInput，并调用 /api/probeVideo 读取视频信息。",
                    style = MaterialTheme.typography.bodySmall,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Button(
                        onClick = { pickerLauncher.launch("video/*") },
                        enabled = !state.uploading && !state.probing && !state.creating,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(if (state.uploading) "上传中..." else "选择本地视频")
                    }
                    OutlinedButton(
                        onClick = { viewModel.refreshProbe() },
                        enabled = state.source != null && !state.uploading && !state.probing && !state.creating,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(if (state.probing) "探测中..." else "重新探测")
                    }
                }
                if (state.uploading || state.probing) {
                    CircularProgressIndicator()
                }
                state.source?.let { source ->
                    Text("文件：${source.displayName}")
                    Text("MIME：${source.mimeType}")
                    Text("大小：${source.size.toReadableSize()}")
                    Text("localPath：${source.localPath}", style = MaterialTheme.typography.bodySmall)
                } ?: Text("尚未选择视频", style = MaterialTheme.typography.bodySmall)
                state.probeError?.let {
                    Text(text = it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
            }

            VideoExtractSection(title = "视频信息") {
                val probe = state.probe
                if (probe == null) {
                    Text("完成上传后会在此显示宽高、时长与平均 FPS。", style = MaterialTheme.typography.bodySmall)
                } else {
                    Text("宽 × 高：${probe.width} × ${probe.height}")
                    Text("时长：${formatDuration(probe.durationSec)}")
                    Text(
                        "平均 FPS：${probe.avgFps?.let { String.format(Locale.US, "%.2f", it) } ?: "-"}",
                    )
                    Text(
                        text = estimatedFrames?.let { "预计输出：约 ${it.coerceAtMost(state.maxFrames.toIntOrNull() ?: it)} 张" }
                            ?: "预计输出：将按最大帧数限制控制",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }

            VideoExtractSection(title = "抽帧参数") {
                Text("模式", style = MaterialTheme.typography.labelLarge)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    VideoExtractOptionButton(
                        label = "关键帧",
                        selected = state.mode == VideoExtractModeOption.KEYFRAME,
                        onClick = { viewModel.updateMode(VideoExtractModeOption.KEYFRAME) },
                        modifier = Modifier.weight(1f),
                    )
                    VideoExtractOptionButton(
                        label = "FPS",
                        selected = state.mode == VideoExtractModeOption.FPS,
                        onClick = { viewModel.updateMode(VideoExtractModeOption.FPS) },
                        modifier = Modifier.weight(1f),
                    )
                    VideoExtractOptionButton(
                        label = "全部",
                        selected = state.mode == VideoExtractModeOption.ALL,
                        onClick = { viewModel.updateMode(VideoExtractModeOption.ALL) },
                        modifier = Modifier.weight(1f),
                    )
                }

                if (state.mode == VideoExtractModeOption.KEYFRAME) {
                    Text("关键帧策略", style = MaterialTheme.typography.labelLarge)
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        VideoExtractOptionButton(
                            label = "I 帧",
                            selected = state.keyframeMode == VideoExtractKeyframeModeOption.IFRAME,
                            onClick = { viewModel.updateKeyframeMode(VideoExtractKeyframeModeOption.IFRAME) },
                            modifier = Modifier.weight(1f),
                        )
                        VideoExtractOptionButton(
                            label = "场景变化",
                            selected = state.keyframeMode == VideoExtractKeyframeModeOption.SCENE,
                            onClick = { viewModel.updateKeyframeMode(VideoExtractKeyframeModeOption.SCENE) },
                            modifier = Modifier.weight(1f),
                        )
                    }
                    if (state.keyframeMode == VideoExtractKeyframeModeOption.SCENE) {
                        OutlinedTextField(
                            modifier = Modifier.fillMaxWidth(),
                            value = state.sceneThreshold,
                            onValueChange = viewModel::updateSceneThreshold,
                            label = { Text("场景阈值 (0-1)") },
                            singleLine = true,
                        )
                    }
                }

                if (state.mode == VideoExtractModeOption.FPS) {
                    OutlinedTextField(
                        modifier = Modifier.fillMaxWidth(),
                        value = state.fps,
                        onValueChange = viewModel::updateFps,
                        label = { Text("FPS") },
                        singleLine = true,
                    )
                }

                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        modifier = Modifier.weight(1f),
                        value = state.startSec,
                        onValueChange = viewModel::updateStartSec,
                        label = { Text("起始秒") },
                        singleLine = true,
                    )
                    OutlinedTextField(
                        modifier = Modifier.weight(1f),
                        value = state.endSec,
                        onValueChange = viewModel::updateEndSec,
                        label = { Text("结束秒") },
                        singleLine = true,
                    )
                }

                OutlinedTextField(
                    modifier = Modifier.fillMaxWidth(),
                    value = state.maxFrames,
                    onValueChange = viewModel::updateMaxFrames,
                    label = { Text("最大帧数上限") },
                    singleLine = true,
                )

                Text("输出格式", style = MaterialTheme.typography.labelLarge)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    VideoExtractOptionButton(
                        label = "JPG",
                        selected = state.outputFormat == VideoExtractOutputFormatOption.JPG,
                        onClick = { viewModel.updateOutputFormat(VideoExtractOutputFormatOption.JPG) },
                        modifier = Modifier.weight(1f),
                    )
                    VideoExtractOptionButton(
                        label = "PNG",
                        selected = state.outputFormat == VideoExtractOutputFormatOption.PNG,
                        onClick = { viewModel.updateOutputFormat(VideoExtractOutputFormatOption.PNG) },
                        modifier = Modifier.weight(1f),
                    )
                }
                if (state.outputFormat == VideoExtractOutputFormatOption.JPG) {
                    OutlinedTextField(
                        modifier = Modifier.fillMaxWidth(),
                        value = state.jpgQuality,
                        onValueChange = viewModel::updateJpgQuality,
                        label = { Text("JPG 质量 (可空，1-31)") },
                        singleLine = true,
                    )
                }

                validationError?.let {
                    Text(text = it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }

                Button(
                    onClick = viewModel::createTask,
                    enabled = validationError == null && state.source != null && state.probe != null && !state.uploading && !state.probing && !state.creating,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (state.creating) "创建中..." else "创建抽帧任务")
                }
            }

            state.createdTask?.let { created ->
                VideoExtractSection(title = "最近创建结果") {
                    Text("任务 ID：${created.taskId}")
                    created.probe?.let { probe ->
                        Text("视频：${probe.width} × ${probe.height} / ${formatDuration(probe.durationSec)}")
                        Text(
                            text = "平均 FPS：${probe.avgFps?.let { String.format(Locale.US, "%.2f", it) } ?: "-"}",
                            style = MaterialTheme.typography.bodySmall,
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        OutlinedButton(
                            onClick = { openTaskCreatedTip(context, created.taskId) },
                            modifier = Modifier.weight(1f),
                        ) {
                            Text("查看说明")
                        }
                        Button(
                            onClick = onOpenTaskCenter,
                            modifier = Modifier.weight(1f),
                        ) {
                            Text("任务中心")
                        }
                    }
                }
            }
        }
    }
}

private fun openTaskCreatedTip(context: Context, taskId: String) {
    android.widget.Toast.makeText(context, "任务已创建，可后续在任务中心查看：$taskId", android.widget.Toast.LENGTH_LONG).show()
}

internal fun VideoExtractCreateUiState.validationError(): String? {
    if (source == null) return "请先选择本地视频"
    if (probe == null) return probeError ?: "请先完成视频探测"
    val maxFramesValue = maxFrames.trim().toIntOrNull() ?: return "maxFrames 必须为正整数"
    if (maxFramesValue <= 0) return "maxFrames 必须大于 0"

    val start = startSec.trim().takeIf { it.isNotBlank() }?.toDoubleOrNull()
        ?: if (startSec.isBlank()) null else return "startSec 格式非法"
    val end = endSec.trim().takeIf { it.isNotBlank() }?.toDoubleOrNull()
        ?: if (endSec.isBlank()) null else return "endSec 格式非法"
    if (start != null && start < 0) return "startSec 不能小于 0"
    if (end != null && end < 0) return "endSec 不能小于 0"
    if (start != null && end != null && end <= start) return "endSec 必须大于 startSec"

    if (mode == VideoExtractModeOption.FPS) {
        val fpsValue = fps.trim().toDoubleOrNull() ?: return "fps 必须大于 0"
        if (fpsValue <= 0) return "fps 必须大于 0"
    }

    if (mode == VideoExtractModeOption.KEYFRAME && keyframeMode == VideoExtractKeyframeModeOption.SCENE) {
        val threshold = sceneThreshold.trim().toDoubleOrNull() ?: return "sceneThreshold 范围为 0-1"
        if (threshold < 0 || threshold > 1) return "sceneThreshold 范围为 0-1"
    }

    if (outputFormat == VideoExtractOutputFormatOption.JPG && jpgQuality.isNotBlank()) {
        val quality = jpgQuality.trim().toIntOrNull() ?: return "jpgQuality 范围为 1-31"
        if (quality !in 1..31) return "jpgQuality 范围为 1-31"
    }
    return null
}

internal fun VideoExtractCreateUiState.toCreatePayloadOrError(): AppResult<VideoExtractCreatePayload> {
    validationError()?.let { return AppResult.Error(it) }
    val start = startSec.trim().takeIf { it.isNotBlank() }?.toDoubleOrNull()
    val end = endSec.trim().takeIf { it.isNotBlank() }?.toDoubleOrNull()
    val fpsValue = fps.trim().takeIf { it.isNotBlank() }?.toDoubleOrNull()
    val sceneThresholdValue = sceneThreshold.trim().takeIf { it.isNotBlank() }?.toDoubleOrNull()
    val jpgQualityValue = jpgQuality.trim().takeIf { it.isNotBlank() }?.toIntOrNull()
    val maxFramesValue = maxFrames.trim().toIntOrNull() ?: return AppResult.Error("maxFrames 必须为正整数")
    return AppResult.Success(
        VideoExtractCreatePayload(
            mode = mode,
            keyframeMode = keyframeMode,
            sceneThreshold = if (mode == VideoExtractModeOption.KEYFRAME && keyframeMode == VideoExtractKeyframeModeOption.SCENE) sceneThresholdValue else null,
            fps = if (mode == VideoExtractModeOption.FPS) fpsValue else null,
            startSec = start,
            endSec = end,
            maxFrames = maxFramesValue,
            outputFormat = outputFormat,
            jpgQuality = if (outputFormat == VideoExtractOutputFormatOption.JPG) jpgQualityValue else null,
        )
    )
}

internal fun JsonObject.toProbeSummary(): VideoExtractProbeSummary = VideoExtractProbeSummary(
    durationSec = doubleOrDefault("durationSec", 0.0),
    width = intOrDefault("width", 0),
    height = intOrDefault("height", 0),
    avgFps = doubleOrNull("avgFps"),
)

internal fun JsonObject.intOrDefault(key: String, defaultValue: Int): Int =
    stringOrNull(key)?.toIntOrNull() ?: defaultValue

internal fun JsonObject.longOrDefault(key: String, defaultValue: Long): Long =
    stringOrNull(key)?.toLongOrNull() ?: defaultValue

internal fun JsonObject.doubleOrNull(key: String): Double? =
    stringOrNull(key)?.toDoubleOrNull()

internal fun JsonObject.doubleOrDefault(key: String, defaultValue: Double): Double =
    doubleOrNull(key) ?: defaultValue

internal fun estimateFrames(state: VideoExtractCreateUiState): Int? {
    val probe = state.probe ?: return null
    val start = state.startSec.trim().toDoubleOrNull() ?: 0.0
    val end = state.endSec.trim().toDoubleOrNull()?.takeIf { it > 0 } ?: probe.durationSec
    val segment = end - start
    if (segment <= 0) return null
    return when (state.mode) {
        VideoExtractModeOption.FPS -> {
            val fps = state.fps.trim().toDoubleOrNull() ?: return null
            if (fps <= 0) null else kotlin.math.round(segment * fps).toInt()
        }
        VideoExtractModeOption.ALL -> {
            val avgFps = probe.avgFps ?: return null
            if (avgFps <= 0) null else kotlin.math.round(segment * avgFps).toInt()
        }
        VideoExtractModeOption.KEYFRAME -> null
    }
}

internal fun formatDuration(seconds: Double?): String {
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

internal fun Long.toReadableSize(): String {
    if (this <= 0L) return "未知"
    val kb = 1024.0
    val mb = kb * 1024.0
    val gb = mb * 1024.0
    return when {
        this >= gb -> String.format(Locale.US, "%.2f GB", this / gb)
        this >= mb -> String.format(Locale.US, "%.2f MB", this / mb)
        this >= kb -> String.format(Locale.US, "%.2f KB", this / kb)
        else -> "$this B"
    }
}
