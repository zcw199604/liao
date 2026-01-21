import request from './request'
import type { ApiResponse, PaginationResponse } from '@/types'
import type { VideoExtractFramesPage, VideoExtractTask, VideoProbeResult } from '@/types'

export interface VideoExtractUploadInputResult {
  localPath: string
  url?: string
  originalFilename?: string
  localFilename?: string
  fileSize?: number
  contentType?: string
  suggestSourceType?: string
}

export const uploadVideoExtractInput = (formData: FormData) => {
  return request.post<any, ApiResponse<VideoExtractUploadInputResult>>('/uploadVideoExtractInput', formData)
}

export const cleanupVideoExtractInput = (localPath: string) => {
  return request.post<any, ApiResponse<{ deleted: boolean }>>('/cleanupVideoExtractInput', { localPath })
}

export const probeVideo = (params: { sourceType: 'upload' | 'mtPhoto'; localPath?: string; md5?: string }) => {
  return request.get<any, ApiResponse<VideoProbeResult>>('/probeVideo', { params })
}

export const createVideoExtractTask = (data: any) => {
  return request.post<any, ApiResponse<{ taskId: string; probe: VideoProbeResult }>>('/createVideoExtractTask', data)
}

export const getVideoExtractTaskList = (page: number, pageSize: number) => {
  return request.get<any, PaginationResponse<VideoExtractTask>>('/getVideoExtractTaskList', {
    params: { page, pageSize }
  })
}

export const getVideoExtractTaskDetail = (params: { taskId: string; cursor?: number; pageSize?: number }) => {
  return request.get<any, ApiResponse<{ task: VideoExtractTask; frames: VideoExtractFramesPage }>>('/getVideoExtractTaskDetail', { params })
}

export const cancelVideoExtractTask = (taskId: string) => {
  return request.post<any, ApiResponse>('/cancelVideoExtractTask', { taskId })
}

export const continueVideoExtractTask = (data: { taskId: string; endSec?: number; maxFrames?: number }) => {
  return request.post<any, ApiResponse>('/continueVideoExtractTask', data)
}

export const deleteVideoExtractTask = (data: { taskId: string; deleteFiles: boolean }) => {
  return request.post<any, ApiResponse>('/deleteVideoExtractTask', data)
}
