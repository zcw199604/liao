package io.github.a7413498.liao.android.core.datastore

import kotlinx.serialization.Serializable

@Serializable
data class CachedMediaLibraryItemSnapshot(
    val url: String,
    val localPath: String,
    val type: String,
    val title: String,
    val subtitle: String,
    val posterUrl: String = "",
    val updateTime: String = "",
    val source: String = "",
)

@Serializable
data class CachedMediaLibrarySnapshot(
    val items: List<CachedMediaLibraryItemSnapshot> = emptyList(),
    val page: Int = 1,
    val total: Int = 0,
    val totalPages: Int = 0,
)

@Serializable
data class CachedVideoExtractTaskItemSnapshot(
    val taskId: String,
    val sourceType: String,
    val sourceRef: String,
    val sourcePreviewUrl: String,
    val outputDirLocalPath: String,
    val outputDirUrl: String,
    val outputFormat: String,
    val jpgQuality: Int? = null,
    val mode: String,
    val keyframeMode: String? = null,
    val fps: Double? = null,
    val sceneThreshold: Double? = null,
    val startSec: Double? = null,
    val endSec: Double? = null,
    val maxFrames: Int,
    val framesExtracted: Int,
    val videoWidth: Int,
    val videoHeight: Int,
    val durationSec: Double? = null,
    val cursorOutTimeSec: Double? = null,
    val status: String,
    val stopReason: String,
    val lastError: String,
    val createdAt: String,
    val updatedAt: String,
    val runtimeLogs: List<String> = emptyList(),
)

@Serializable
data class CachedVideoExtractFrameItemSnapshot(
    val seq: Int,
    val url: String,
)

@Serializable
data class CachedVideoExtractTaskListSnapshot(
    val items: List<CachedVideoExtractTaskItemSnapshot> = emptyList(),
    val page: Int = 1,
    val pageSize: Int = 20,
    val total: Int = 0,
)

@Serializable
data class CachedVideoExtractTaskDetailSnapshot(
    val task: CachedVideoExtractTaskItemSnapshot,
    val frames: List<CachedVideoExtractFrameItemSnapshot> = emptyList(),
    val nextCursor: Int = 0,
    val hasMore: Boolean = false,
)

@Serializable
data class CachedVideoExtractTaskDetailsSnapshot(
    val items: List<CachedVideoExtractTaskDetailSnapshot> = emptyList(),
)
