import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { UploadedMedia, VideoExtractFramesPage, VideoExtractMode, VideoExtractSourceType, VideoExtractTask, VideoProbeResult } from '@/types'
import { extractUploadLocalPath } from '@/utils/media'
import * as videoExtractApi from '@/api/videoExtract'

export interface VideoExtractCreateSource {
  sourceType: VideoExtractSourceType
  localPath?: string
  md5?: string
  displayName?: string
  userId?: string
  mediaUrl?: string
}

export const useVideoExtractStore = defineStore('videoExtract', () => {
  const showCreateModal = ref(false)
  const showTaskModal = ref(false)

  const createSource = ref<VideoExtractCreateSource | null>(null)
  const probeLoading = ref(false)
  const probe = ref<VideoProbeResult | null>(null)
  const probeError = ref<string>('')

  const listLoading = ref(false)
  const tasks = ref<VideoExtractTask[]>([])
  const listPage = ref(1)
  const listPageSize = ref(20)
  const listTotal = ref(0)

  const selectedTaskId = ref<string>('')
  const selectedTask = ref<VideoExtractTask | null>(null)
  const detailLoading = ref(false)
  const frames = ref<VideoExtractFramesPage>({ items: [], nextCursor: 0, hasMore: false })

  const polling = ref(false)
  let pollTimer: ReturnType<typeof setTimeout> | null = null

  const clearCreateState = () => {
    createSource.value = null
    probe.value = null
    probeError.value = ''
    showCreateModal.value = false
  }

  const isRunningStatus = (status?: string) => {
    return status === 'PENDING' || status === 'PREPARING' || status === 'RUNNING'
  }

  const isUsableUploadLocalPath = (value: string) => {
    const p = String(value || '').trim()
    return p.startsWith('/videos/') || p.startsWith('/tmp/video_extract_inputs/')
  }

  const mergeFrameItems = (existing: any[], incoming: any[]) => {
    const seenSeq = new Set<number>()
    const result: any[] = []

    for (const item of [...existing, ...incoming]) {
      const seq = Number(item?.seq)
      if (Number.isFinite(seq)) {
        if (seenSeq.has(seq)) continue
        seenSeq.add(seq)
      }
      result.push(item)
    }

    return result
  }

  const openCreateFromMedia = async (media: UploadedMedia, userId?: string) => {
    if (!media || media.type !== 'video') return false

    const url = String(media.url || '').trim()
    const md5 = String(media.md5 || '').trim()

    let sourceType: VideoExtractSourceType = 'upload'
    let localPath = ''
    let md5Value = ''

    // mtPhoto：通常带 md5，且预览 URL 为 /lsp/...（或 /api/... 下载地址）
    if (md5 && (url.startsWith('/lsp/') || url.startsWith('/api/'))) {
      sourceType = 'mtPhoto'
      md5Value = md5
    } else {
      sourceType = 'upload'
      localPath = extractUploadLocalPath(url)
    }

    if (sourceType === 'upload' && !isUsableUploadLocalPath(localPath)) {
      clearCreateState()
      return false
    }

    if (sourceType === 'mtPhoto' && !md5Value) {
      clearCreateState()
      return false
    }

    createSource.value = {
      sourceType,
      localPath: localPath || undefined,
      md5: md5Value || undefined,
      displayName: media.originalFilename || media.localFilename || undefined,
      userId: userId || undefined,
      mediaUrl: url || undefined
    }

    probe.value = null
    probeError.value = ''

    showCreateModal.value = true
    await fetchProbe()
    return true
  }

  const closeCreateModal = () => {
    showCreateModal.value = false
  }

  const openTaskCenter = async (taskId?: string) => {
    showTaskModal.value = true
    await loadTasks(1)
    if (taskId) {
      await openTaskDetail(taskId)
    }
  }

  const closeTaskModal = () => {
    showTaskModal.value = false
    stopPolling()
    selectedTaskId.value = ''
    selectedTask.value = null
    frames.value = { items: [], nextCursor: 0, hasMore: false }
  }

  const fetchProbe = async () => {
    if (!createSource.value) return
    probeLoading.value = true
    probeError.value = ''
    try {
      const params: any = { sourceType: createSource.value.sourceType }
      if (createSource.value.sourceType === 'upload') params.localPath = createSource.value.localPath
      if (createSource.value.sourceType === 'mtPhoto') params.md5 = createSource.value.md5
      const res = await videoExtractApi.probeVideo(params)
      if (res?.code === 0 && res.data) {
        probe.value = res.data
      } else {
        probe.value = null
        probeError.value = String(res?.msg || res?.message || '探测失败')
      }
    } catch (e: any) {
      probe.value = null
      probeError.value = String(e?.message || '探测失败')
    } finally {
      probeLoading.value = false
    }
  }

  const createTask = async (payload: {
    mode: VideoExtractMode
    keyframeMode?: 'iframe' | 'scene'
    sceneThreshold?: number
    fps?: number
    startSec?: number
    endSec?: number
    maxFrames: number
    outputFormat: 'jpg' | 'png'
    jpgQuality?: number
  }) => {
    if (!createSource.value) throw new Error('缺少视频来源')
    if (probeLoading.value) throw new Error('视频探测中，请稍后再试')
    if (probeError.value) throw new Error('视频探测失败，请刷新后重试')
    if (!probe.value) throw new Error('请先完成视频探测')
    const body: any = {
      userId: createSource.value.userId,
      sourceType: createSource.value.sourceType,
      localPath: createSource.value.localPath,
      md5: createSource.value.md5,
      ...payload
    }
    const res = await videoExtractApi.createVideoExtractTask(body)
    if (res?.code !== 0 || !res.data?.taskId) {
      throw new Error(String(res?.msg || res?.message || '创建任务失败'))
    }
    return res.data
  }

  const loadTasks = async (page: number = 1) => {
    listLoading.value = true
    try {
      const res = await videoExtractApi.getVideoExtractTaskList(page, listPageSize.value)
      const items = Array.isArray(res?.data?.items) ? res.data.items : []
      tasks.value = items
      listTotal.value = Number(res?.data?.total || 0)
      listPage.value = Number(res?.data?.page || page)
      listPageSize.value = Number(res?.data?.pageSize || listPageSize.value)
    } finally {
      listLoading.value = false
    }
  }

  const openTaskDetail = async (taskId: string) => {
    selectedTaskId.value = String(taskId || '').trim()
    selectedTask.value = null
    frames.value = { items: [], nextCursor: 0, hasMore: false }
    await refreshTaskDetail(true)
    startPolling()
  }

  const refreshTaskDetail = async (resetCursor: boolean = false) => {
    if (!selectedTaskId.value) return
    detailLoading.value = true
    try {
      const cursor = resetCursor ? 0 : frames.value.nextCursor
      const res = await videoExtractApi.getVideoExtractTaskDetail({
        taskId: selectedTaskId.value,
        cursor,
        pageSize: 120
      })
      if (res?.code !== 0 || !res.data?.task) return

      selectedTask.value = res.data.task

      const page = res.data.frames
      if (page && Array.isArray(page.items)) {
        if (resetCursor) {
          frames.value = page
        } else {
          const merged = mergeFrameItems(frames.value.items, page.items)
          frames.value = { ...page, items: merged }
        }
      }
    } finally {
      detailLoading.value = false
    }
  }

  const loadMoreFrames = async () => {
    if (!selectedTaskId.value) return
    if (!frames.value.hasMore) return
    await refreshTaskDetail(false)
  }

  const cancelTask = async (taskId: string) => {
    const id = String(taskId || '').trim()
    if (!id) return
    await videoExtractApi.cancelVideoExtractTask(id)
    await refreshTaskDetail(false)
    await loadTasks(listPage.value)
  }

  const continueTask = async (data: { taskId: string; endSec?: number; maxFrames?: number }) => {
    const taskId = String(data.taskId || '').trim()
    if (!taskId) throw new Error('缺少任务ID')

    const hasEndSec = data.endSec !== undefined && data.endSec !== null
    const hasMaxFrames = data.maxFrames !== undefined && data.maxFrames !== null
    if (!hasEndSec && !hasMaxFrames) throw new Error('请至少填写新的 endSec 或 maxFrames')

    const body: { taskId: string; endSec?: number; maxFrames?: number } = { taskId }

    if (hasEndSec) {
      const endSec = Number(data.endSec)
      if (!Number.isFinite(endSec) || endSec < 0) throw new Error('endSec 必须为非负数字')
      const baseline = Number(selectedTask.value?.cursorOutTimeSec ?? selectedTask.value?.startSec ?? 0)
      if (Number.isFinite(baseline) && endSec <= baseline) throw new Error('endSec 必须大于当前进度')
      body.endSec = endSec
    }

    if (hasMaxFrames) {
      const maxFrames = Number(data.maxFrames)
      if (!Number.isFinite(maxFrames) || maxFrames <= 0 || !Number.isInteger(maxFrames)) throw new Error('maxFrames 必须为正整数')
      const extracted = Number(selectedTask.value?.framesExtracted ?? 0)
      if (Number.isFinite(extracted) && maxFrames <= extracted) throw new Error('maxFrames 必须大于已输出帧数')
      body.maxFrames = maxFrames
    }

    await videoExtractApi.continueVideoExtractTask(body)
    await refreshTaskDetail(true)
    startPolling()
    await loadTasks(listPage.value)
  }

  const deleteTask = async (data: { taskId: string; deleteFiles: boolean }) => {
    await videoExtractApi.deleteVideoExtractTask(data)
    if (selectedTaskId.value === data.taskId) {
      selectedTaskId.value = ''
      selectedTask.value = null
      frames.value = { items: [], nextCursor: 0, hasMore: false }
      stopPolling()
    }
    await loadTasks(listPage.value)
  }

  const startPolling = () => {
    stopPolling()
    polling.value = true

    const tick = async () => {
      if (!polling.value) return
      const status = selectedTask.value?.status
      const visible = typeof document !== 'undefined' ? document.visibilityState !== 'hidden' : true
      const interval = visible && isRunningStatus(status) ? 1000 : 5000

      try {
        if (selectedTaskId.value) {
          await refreshTaskDetail(false)
        }
        if (!isRunningStatus(selectedTask.value?.status)) {
          stopPolling()
          return
        }
      } finally {
        if (!polling.value) return
        pollTimer = setTimeout(tick, interval)
      }
    }

    pollTimer = setTimeout(tick, 600)
  }

  const stopPolling = () => {
    polling.value = false
    if (pollTimer) {
      clearTimeout(pollTimer)
      pollTimer = null
    }
  }

  const createSourceLabel = computed(() => {
    if (!createSource.value) return ''
    const name = createSource.value.displayName || ''
    if (name) return name
    if (createSource.value.sourceType === 'mtPhoto') return `mtPhoto:${createSource.value.md5 || ''}`
    return createSource.value.localPath || ''
  })

  return {
    // ui
    showCreateModal,
    showTaskModal,
    openCreateFromMedia,
    closeCreateModal,
    openTaskCenter,
    closeTaskModal,

    // create
    createSource,
    createSourceLabel,
    probeLoading,
    probe,
    probeError,
    fetchProbe,
    createTask,

    // list
    tasks,
    listLoading,
    listPage,
    listPageSize,
    listTotal,
    loadTasks,

    // detail
    selectedTaskId,
    selectedTask,
    detailLoading,
    frames,
    openTaskDetail,
    refreshTaskDetail,
    loadMoreFrames,
    cancelTask,
    continueTask,
    deleteTask,

    // polling
    polling,
    startPolling,
    stopPolling
  }
})

