<template>
  <teleport to="body">
    <div
      v-if="videoExtractStore.showCreateModal"
      class="fixed inset-0 z-[90] bg-black/70 flex items-center justify-center p-4"
      @click="close"
    >
      <div
        class="w-full max-w-4xl bg-[#18181b] rounded-2xl shadow-2xl overflow-hidden border border-white/10 flex flex-col max-h-[90vh]"
        @click.stop
      >
        <!-- Header -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-white/5 bg-[#1f1f23]">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-8 h-8 rounded-lg bg-emerald-500/20 flex items-center justify-center">
              <i class="fas fa-film text-emerald-400"></i>
            </div>
            <div class="min-w-0">
              <h3 class="text-lg font-bold text-white truncate">视频抽帧</h3>
              <p class="text-xs text-gray-500 truncate">
                {{ videoExtractStore.createSourceLabel || '选择一个视频后再创建任务' }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button
              class="px-3 py-2 text-xs rounded-lg bg-white/10 hover:bg-white/15 text-gray-200 transition disabled:opacity-50 disabled:cursor-not-allowed"
              :disabled="!sourcePreviewMedia"
              @click="openSourcePreview"
              title="预览源视频（支持倍速与抓帧）"
            >
              <i class="fas fa-play mr-1"></i>
              预览/抓帧
            </button>
            <button
              class="px-3 py-2 text-xs rounded-lg bg-white/10 hover:bg-white/15 text-gray-200 transition"
              :disabled="videoExtractStore.probeLoading"
              @click="refreshProbe"
              title="重新探测视频信息"
            >
              <i class="fas fa-sync-alt mr-1" :class="videoExtractStore.probeLoading ? 'fa-spin' : ''"></i>
              刷新
            </button>
            <button
              @click="close"
              class="w-8 h-8 flex items-center justify-center text-white/40 hover:text-white/90 transition-colors rounded-lg hover:bg-[#27272a]"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>
        </div>

        <div class="flex-1 overflow-y-auto p-6 space-y-6">
          <!-- Probe summary -->
          <div class="rounded-xl border border-white/5 bg-[#111113] p-4">
            <div class="flex items-center justify-between gap-3">
              <div class="text-sm text-gray-300 font-medium">视频信息</div>
              <div v-if="videoExtractStore.probeLoading" class="text-xs text-gray-500 flex items-center gap-2">
                <span class="w-3 h-3 border-2 border-gray-500 border-t-transparent rounded-full animate-spin"></span>
                探测中...
              </div>
            </div>

            <div v-if="videoExtractStore.probeError" class="mt-3 text-xs text-red-400">
              {{ videoExtractStore.probeError }}
            </div>

            <div v-else class="mt-3 grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div class="rounded-lg bg-[#18181b] border border-white/5 p-3">
                <div class="text-[11px] text-gray-500 mb-1">宽 × 高</div>
                <div class="text-sm text-white font-mono">
                  {{ probe?.width || '-' }} × {{ probe?.height || '-' }}
                </div>
              </div>
              <div class="rounded-lg bg-[#18181b] border border-white/5 p-3">
                <div class="text-[11px] text-gray-500 mb-1">时长</div>
                <div class="text-sm text-white font-mono">
                  {{ formatSec(probe?.durationSec) }}
                </div>
              </div>
              <div class="rounded-lg bg-[#18181b] border border-white/5 p-3">
                <div class="text-[11px] text-gray-500 mb-1">平均 FPS</div>
                <div class="text-sm text-white font-mono">
                  {{ probe?.avgFps ? probe.avgFps.toFixed(2) : '-' }}
                </div>
              </div>
              <div class="rounded-lg bg-[#18181b] border border-white/5 p-3">
                <div class="text-[11px] text-gray-500 mb-1">预计输出</div>
                <div class="text-sm text-white font-mono">
                  {{ estimateText }}
                </div>
              </div>
            </div>
          </div>

          <!-- Form -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="space-y-4">
              <div>
                <label class="block text-sm text-gray-400 mb-2">模式</label>
                <div class="grid grid-cols-3 gap-2">
                  <button
                    type="button"
                    class="px-3 py-2 rounded-lg border text-sm transition"
                    :class="mode === 'keyframe' ? activeBtn : idleBtn"
                    @click="mode = 'keyframe'"
                  >
                    关键帧
                  </button>
                  <button
                    type="button"
                    class="px-3 py-2 rounded-lg border text-sm transition"
                    :class="mode === 'fps' ? activeBtn : idleBtn"
                    @click="mode = 'fps'"
                  >
                    固定FPS
                  </button>
                  <button
                    type="button"
                    class="px-3 py-2 rounded-lg border text-sm transition"
                    :class="mode === 'all' ? activeBtn : idleBtn"
                    @click="mode = 'all'"
                  >
                    每一帧
                  </button>
                </div>
              </div>

              <div v-if="mode === 'keyframe'" class="space-y-3">
                <div>
                  <label class="block text-sm text-gray-400 mb-2">关键帧策略</label>
                  <div class="grid grid-cols-2 gap-2">
                    <button type="button" class="px-3 py-2 rounded-lg border text-sm transition" :class="keyframeMode === 'iframe' ? activeBtn : idleBtn" @click="keyframeMode = 'iframe'">
                      I 帧
                    </button>
                    <button type="button" class="px-3 py-2 rounded-lg border text-sm transition" :class="keyframeMode === 'scene' ? activeBtn : idleBtn" @click="keyframeMode = 'scene'">
                      场景变化
                    </button>
                  </div>
                </div>

                <div v-if="keyframeMode === 'scene'">
                  <label class="block text-sm text-gray-400 mb-2">场景阈值 (0-1)</label>
                  <input
                    v-model.number="sceneThreshold"
                    type="number"
                    min="0"
                    max="1"
                    step="0.01"
                    class="w-full bg-[#111113] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-emerald-500 focus:outline-none"
                    placeholder="默认 0.30"
                  />
                  <p class="text-xs text-gray-500 mt-1">数值越大越严格，通常 0.2~0.4</p>
                </div>
              </div>

              <div v-if="mode === 'fps'">
                <label class="block text-sm text-gray-400 mb-2">FPS</label>
                <input
                  v-model.number="fps"
                  type="number"
                  min="0.01"
                  step="0.01"
                  class="w-full bg-[#111113] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-emerald-500 focus:outline-none"
                  placeholder="例如 1 表示每秒 1 帧"
                />
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-sm text-gray-400 mb-2">起始秒</label>
                  <input
                    v-model.number="startSec"
                    type="number"
                    min="0"
                    step="0.01"
                    class="w-full bg-[#111113] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-emerald-500 focus:outline-none"
                    placeholder="可空"
                  />
                </div>
                <div>
                  <label class="block text-sm text-gray-400 mb-2">结束秒</label>
                  <input
                    v-model.number="endSec"
                    type="number"
                    min="0"
                    step="0.01"
                    class="w-full bg-[#111113] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-emerald-500 focus:outline-none"
                    placeholder="可空"
                  />
                </div>
              </div>

              <div>
                <label class="block text-sm text-gray-400 mb-2">最大帧数上限</label>
                <input
                  v-model.number="maxFrames"
                  type="number"
                  min="1"
                  step="1"
                  class="w-full bg-[#111113] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-emerald-500 focus:outline-none"
                  placeholder="必填，例如 500"
                />
                <p class="text-xs mt-1" :class="riskLevel === 'high' ? 'text-red-400' : riskLevel === 'mid' ? 'text-amber-400' : 'text-gray-500'">
                  {{ riskText }}
                </p>
              </div>
            </div>

            <div class="space-y-4">
              <div>
                <label class="block text-sm text-gray-400 mb-2">输出格式</label>
                <div class="grid grid-cols-2 gap-2">
                  <button type="button" class="px-3 py-2 rounded-lg border text-sm transition" :class="outputFormat === 'jpg' ? activeBtn : idleBtn" @click="outputFormat = 'jpg'">
                    JPG
                  </button>
                  <button type="button" class="px-3 py-2 rounded-lg border text-sm transition" :class="outputFormat === 'png' ? activeBtn : idleBtn" @click="outputFormat = 'png'">
                    PNG
                  </button>
                </div>
                <p class="text-xs text-gray-500 mt-1">JPG 体积小；PNG 无损但体积更大</p>
              </div>

              <div v-if="outputFormat === 'jpg'">
                <label class="block text-sm text-gray-400 mb-2">JPG 质量 (1-31)</label>
                <input
                  v-model.number="jpgQuality"
                  type="number"
                  min="1"
                  max="31"
                  step="1"
                  class="w-full bg-[#111113] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-emerald-500 focus:outline-none"
                  placeholder="可空，建议 3~6（数值越小质量越高）"
                />
              </div>

              <div class="rounded-xl border border-white/5 bg-[#111113] p-4">
                <div class="text-sm text-gray-300 font-medium mb-2">提交说明</div>
                <ul class="text-xs text-gray-500 space-y-1 list-disc pl-4">
                  <li>任务为异步执行，可在“任务中心”查看进度与预览。</li>
                  <li>达到 endSec / maxFrames 时会以“因限制暂停”状态结束，可继续追加抽帧。</li>
                  <li>运行中可随时终止，已生成的图片会保留在任务目录中。</li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="px-6 py-4 border-t border-white/5 bg-[#1f1f23] flex items-center justify-between gap-3">
          <button
            class="px-4 py-2 rounded-lg bg-white/10 hover:bg-white/15 text-gray-200 transition"
            @click="close"
          >
            取消
          </button>
          <button
            class="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold transition disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            :disabled="submitting || !!validationError || videoExtractStore.probeLoading"
            @click="submit"
          >
            <span v-if="submitting" class="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin"></span>
            <span>{{ submitting ? '创建中...' : '创建任务' }}</span>
          </button>
        </div>
      </div>
    </div>
  </teleport>

  <MediaPreview
    v-model:visible="showSourcePreview"
    :url="sourcePreviewMedia?.url || ''"
    :type="'video'"
    :can-upload="false"
    :media-list="sourcePreviewMediaList"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { useVideoExtractStore } from '@/stores/videoExtract'
import type { UploadedMedia, VideoProbeResult } from '@/types'
import MediaPreview from '@/components/media/MediaPreview.vue'

const videoExtractStore = useVideoExtractStore()
const { show } = useToast()

const showSourcePreview = ref(false)

const normalizeUploadLocalPath = (input: string): string => {
  let p = String(input || '').trim()
  if (!p) return ''

  p = p.replace(/\\/g, '/')
  try {
    const u = new URL(p)
    p = u.pathname || p
  } catch {
    // ignore
  }

  p = (p.split('?')[0] || '').split('#')[0] || ''
  if (p.startsWith('/upload/')) p = p.slice('/upload'.length)
  p = p.trim()
  if (!p) return ''
  if (!p.startsWith('/')) p = '/' + p
  return p
}

const buildUploadPreviewUrl = (localPath: string): string => {
  const p = normalizeUploadLocalPath(localPath)
  if (!p) return ''
  return `/upload${p}`
}

const sourcePreviewMedia = computed<UploadedMedia | null>(() => {
  const src = videoExtractStore.createSource
  if (!src) return null

  if (src.sourceType === 'upload' && src.localPath) {
    const url = buildUploadPreviewUrl(src.localPath)
    if (!url) return null
    return {
      url,
      type: 'video',
      originalFilename: src.displayName,
      localFilename: src.displayName
    }
  }

  const u = String(src.mediaUrl || '').trim()
  if (!u) return null
  return {
    url: u,
    type: 'video',
    originalFilename: src.displayName,
    localFilename: src.displayName
  }
})

const sourcePreviewMediaList = computed<UploadedMedia[]>(() => (sourcePreviewMedia.value ? [sourcePreviewMedia.value] : []))

const openSourcePreview = () => {
  if (!sourcePreviewMedia.value) return
  showSourcePreview.value = true
}

const mode = ref<'keyframe' | 'fps' | 'all'>('keyframe')
const keyframeMode = ref<'iframe' | 'scene'>('iframe')
const sceneThreshold = ref<number>(0.3)
const fps = ref<number>(1)
const startSec = ref<number | null>(null)
const endSec = ref<number | null>(null)
const maxFrames = ref<number>(500)
const outputFormat = ref<'jpg' | 'png'>('jpg')
const jpgQuality = ref<number | null>(null)

const submitting = ref(false)

const probe = computed<VideoProbeResult | null>(() => videoExtractStore.probe)

const activeBtn = 'border-emerald-500 bg-emerald-500/10 text-emerald-300'
const idleBtn = 'border-white/10 bg-[#111113] text-gray-300 hover:border-white/20'

const formatSec = (sec?: number) => {
  if (!sec || sec <= 0) return '-'
  const s = Math.floor(sec)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const r = s % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(r).padStart(2, '0')}`
  return `${m}:${String(r).padStart(2, '0')}`
}

const segmentDuration = computed(() => {
  const s = startSec.value ?? 0
  const e = endSec.value
  const dur = probe.value?.durationSec
  const end = typeof e === 'number' && e > 0 ? e : (dur || 0)
  const seg = end - s
  return seg > 0 ? seg : 0
})

const estimate = computed(() => {
  if (!probe.value) return null
  const seg = segmentDuration.value
  if (seg <= 0) return null
  if (mode.value === 'fps') {
    const f = Number(fps.value || 0)
    if (f <= 0) return null
    return Math.round(seg * f)
  }
  if (mode.value === 'all') {
    const af = Number(probe.value.avgFps || 0)
    if (af <= 0) return null
    return Math.round(seg * af)
  }
  return null
})

const estimateText = computed(() => {
  if (mode.value === 'keyframe') return '不可预估'
  const v = estimate.value
  if (v === null) return '-'
  return String(v)
})

const riskLevel = computed<'low' | 'mid' | 'high'>(() => {
  const est = estimate.value
  const limit = Number(maxFrames.value || 0)
  const predicted = est === null ? limit : Math.min(est, limit)
  if (predicted >= 5000) return 'high'
  if (predicted >= 1500) return 'mid'
  return 'low'
})

const riskText = computed(() => {
  const est = estimate.value
  const limit = Number(maxFrames.value || 0)
  if (est === null) return `将按最大帧数 ${limit || '-'} 控制输出量`
  const predicted = Math.min(est, limit || est)
  if (predicted >= 5000) return `预计输出约 ${predicted} 张，数量较大，建议缩小时间区间或降低 FPS`
  if (predicted >= 1500) return `预计输出约 ${predicted} 张，建议关注加载与存储占用`
  return `预计输出约 ${predicted} 张`
})

const validationError = computed(() => {
  if (!videoExtractStore.createSource) return '缺少视频来源'
  if (!maxFrames.value || maxFrames.value <= 0) return 'maxFrames 必须大于 0'
  if (mode.value === 'fps' && (!fps.value || fps.value <= 0)) return 'fps 必须大于 0'
  const s = startSec.value
  const e = endSec.value
  if (s !== null && s < 0) return 'startSec 不能小于 0'
  if (e !== null && e < 0) return 'endSec 不能小于 0'
  if (s !== null && e !== null && e > 0 && e <= s) return 'endSec 必须大于 startSec'
  if (outputFormat.value === 'jpg' && jpgQuality.value !== null) {
    if (jpgQuality.value < 1 || jpgQuality.value > 31) return 'jpgQuality 范围为 1-31'
  }
  if (mode.value === 'keyframe' && keyframeMode.value === 'scene') {
    if (sceneThreshold.value < 0 || sceneThreshold.value > 1) return 'sceneThreshold 范围为 0-1'
  }
  return ''
})

const refreshProbe = async () => {
  await videoExtractStore.fetchProbe()
}

const close = () => {
  videoExtractStore.closeCreateModal()
}

const submit = async () => {
  if (validationError.value) {
    show(validationError.value)
    return
  }
  submitting.value = true
  try {
    const res = await videoExtractStore.createTask({
      mode: mode.value,
      keyframeMode: mode.value === 'keyframe' ? keyframeMode.value : undefined,
      sceneThreshold: mode.value === 'keyframe' && keyframeMode.value === 'scene' ? sceneThreshold.value : undefined,
      fps: mode.value === 'fps' ? fps.value : undefined,
      startSec: startSec.value ?? undefined,
      endSec: endSec.value ?? undefined,
      maxFrames: maxFrames.value,
      outputFormat: outputFormat.value,
      jpgQuality: outputFormat.value === 'jpg' ? (jpgQuality.value ?? undefined) : undefined
    })

    show('抽帧任务已创建')
    videoExtractStore.closeCreateModal()
    await videoExtractStore.openTaskCenter(res.taskId)
  } catch (e: any) {
    console.error('create task failed', e)
    show(String(e?.message || '创建任务失败'))
  } finally {
    submitting.value = false
  }
}

watch(
  () => videoExtractStore.showCreateModal,
  (v) => {
    if (!v) return
    // 每次打开时做一次轻量默认值重置（避免上一次状态污染）
    mode.value = 'keyframe'
    keyframeMode.value = 'iframe'
    sceneThreshold.value = 0.3
    fps.value = 1
    startSec.value = null
    endSec.value = null
    maxFrames.value = 500
    outputFormat.value = 'jpg'
    jpgQuality.value = null
  }
)

watch(
  () => videoExtractStore.showCreateModal,
  (v) => {
    if (v) return
    showSourcePreview.value = false
  }
)
</script>
