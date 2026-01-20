export type VideoExtractSourceType = 'upload' | 'mtPhoto'

export type VideoExtractMode = 'keyframe' | 'fps' | 'all'

export type VideoExtractKeyframeMode = 'iframe' | 'scene'

export type VideoExtractTaskStatus =
  | 'PENDING'
  | 'PREPARING'
  | 'RUNNING'
  | 'PAUSED_USER'
  | 'PAUSED_LIMIT'
  | 'FINISHED'
  | 'FAILED'

export type VideoExtractStopReason = 'MAX_FRAMES' | 'END_SEC' | 'EOF' | 'USER' | 'ERROR'

export type VideoExtractOutputFormat = 'jpg' | 'png'

export interface VideoProbeResult {
  durationSec: number
  width: number
  height: number
  avgFps?: number
}

export interface VideoExtractRuntimeView {
  frame: number
  outTimeSec: number
  speed?: string
  logs?: string[]
}

export interface VideoExtractTask {
  taskId: string
  userId?: string

  sourceType: VideoExtractSourceType
  sourceRef: string

  outputDirLocalPath: string
  outputDirUrl?: string
  outputFormat: VideoExtractOutputFormat
  jpgQuality?: number

  mode: VideoExtractMode
  keyframeMode?: VideoExtractKeyframeMode
  fps?: number
  sceneThreshold?: number

  startSec?: number
  endSec?: number
  maxFrames: number

  framesExtracted: number
  videoWidth: number
  videoHeight: number
  durationSec?: number
  cursorOutTimeSec?: number

  status: VideoExtractTaskStatus
  stopReason?: VideoExtractStopReason
  lastError?: string

  createdAt?: string
  updatedAt?: string

  runtime?: VideoExtractRuntimeView
}

export interface VideoExtractFrame {
  seq: number
  url: string
}

export interface VideoExtractFramesPage {
  items: VideoExtractFrame[]
  nextCursor: number
  hasMore: boolean
}

