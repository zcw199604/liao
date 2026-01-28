<template>
  <teleport to="body">
    <div
      v-if="videoExtractStore.showTaskModal"
      class="fixed inset-0 z-[85] bg-black/70 flex items-stretch justify-stretch md:items-center md:justify-center p-0 md:p-4"
      @click="close"
    >
      <div
        class="w-full md:max-w-6xl h-[100dvh] md:h-[88vh] bg-[#18181b] rounded-none md:rounded-2xl shadow-2xl flex flex-col overflow-hidden border-0 md:border border-white/10 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] md:pt-0 md:pb-0"
        @click.stop
      >
        <!-- Header -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-white/5 bg-[#1f1f23]">
          <div class="flex items-center gap-3 min-w-0">
            <button
              v-if="isMobile && mobilePane === 'detail'"
              class="w-8 h-8 flex items-center justify-center text-white/40 hover:text-white/90 transition-colors rounded-lg hover:bg-[#27272a]"
              @click="backToList"
              title="返回任务列表"
            >
              <i class="fas fa-arrow-left"></i>
            </button>
            <div class="w-8 h-8 rounded-lg bg-purple-500/20 flex items-center justify-center">
              <i class="fas fa-tasks text-purple-400"></i>
            </div>
            <div class="min-w-0">
              <h3 class="text-lg font-bold text-white truncate">抽帧任务中心</h3>
              <p class="text-xs text-gray-500 truncate">
                {{ videoExtractStore.polling ? '实时刷新中' : '可刷新查看最新状态' }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button
              class="px-3 py-2 text-xs rounded-lg bg-white/10 hover:bg-white/15 text-gray-200 transition"
              :disabled="videoExtractStore.listLoading"
              @click="refresh"
              title="刷新任务列表"
            >
              <i class="fas fa-sync-alt mr-1" :class="videoExtractStore.listLoading ? 'fa-spin' : ''"></i>
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

        <div class="flex flex-1 overflow-hidden">
          <!-- Left: task list -->
          <div
            class="w-full md:w-[360px] bg-[#1f1f23] border-r border-white/5 flex flex-col overflow-hidden"
            :class="isMobile && mobilePane === 'detail' ? 'hidden' : ''"
          >
            <div class="px-4 py-3 border-b border-white/5 flex items-center justify-between">
              <div class="flex items-center gap-2">
                <div class="text-sm text-gray-300 font-medium">任务列表</div>
                <button
                  class="px-2.5 py-1.5 text-[11px] rounded-lg bg-purple-600 hover:bg-purple-500 text-white transition disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
                  :disabled="uploading"
                  @click="pickUploadFile"
                  title="上传视频并创建抽帧任务（不限制文件类型）"
                >
                  <span v-if="uploading" class="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin"></span>
                  <i v-else class="fas fa-upload"></i>
                  <span class="hidden sm:inline">{{ uploading ? '上传中' : '上传' }}</span>
                </button>
              </div>
              <div class="text-xs text-gray-500">{{ videoExtractStore.listTotal }} 个</div>
            </div>

            <input ref="uploadInputRef" type="file" class="hidden" @change="handleUploadInputChange" />

            <div class="flex-1 overflow-y-auto no-scrollbar">
              <div v-if="videoExtractStore.listLoading && videoExtractStore.tasks.length === 0" class="p-6 text-center text-gray-500 text-sm">
                加载中...
              </div>

              <button
                v-for="t in videoExtractStore.tasks"
                :key="t.taskId"
                class="w-full text-left px-4 py-3 border-b border-white/5 hover:bg-[#27272a] transition-colors"
                :class="t.taskId === videoExtractStore.selectedTaskId ? 'bg-[#27272a]' : ''"
                @click="openDetail(t.taskId)"
              >
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0">
                    <div class="text-sm text-white font-medium truncate">
                      {{ formatTaskTitle(t) }}
                    </div>
                    <div class="mt-1 text-xs text-gray-500 truncate">
                      {{ formatTaskSub(t) }}
                    </div>
                  </div>
                  <span class="text-[11px] px-2 py-1 rounded-full" :class="statusBadgeClass(t.status)">
                    {{ statusText(t.status) }}
                  </span>
                </div>

                <div class="mt-2 flex items-center justify-between gap-2">
                  <div class="text-[11px] text-gray-500 font-mono">
                    {{ t.framesExtracted }} / {{ t.maxFrames }}
                  </div>
                  <div class="text-[11px] text-gray-500 font-mono">
                    {{ formatPercent(progressPercent(t)) }}
                  </div>
                </div>

                <div class="mt-2 h-2 bg-black/30 rounded-full overflow-hidden">
                  <div class="h-full bg-purple-500/70" :style="{ width: `${progressPercent(t)}%` }"></div>
                </div>
              </button>

              <div v-if="!videoExtractStore.listLoading && videoExtractStore.tasks.length === 0" class="p-6 text-center text-gray-500 text-sm">
                暂无任务
              </div>
            </div>
          </div>

          <!-- Right: detail -->
          <div class="flex-1 flex flex-col overflow-hidden" :class="isMobile && mobilePane === 'list' ? 'hidden' : ''">
            <div v-if="!videoExtractStore.selectedTaskId" class="flex-1 flex items-center justify-center text-gray-600">
              <div class="text-center">
                <i class="fas fa-hand-pointer text-4xl mb-3 opacity-30"></i>
                <div class="text-sm">选择一个任务查看详情</div>
              </div>
            </div>

            <div v-else-if="videoExtractStore.detailLoading && !videoExtractStore.selectedTask" class="flex-1 flex items-center justify-center text-gray-600">
              <div class="text-center">
                <span class="w-6 h-6 border-2 border-gray-500 border-t-transparent rounded-full animate-spin inline-block mb-3"></span>
                <div class="text-sm">加载任务详情...</div>
              </div>
            </div>

            <div v-else-if="!videoExtractStore.selectedTask" class="flex-1 flex items-center justify-center text-gray-600">
              <div class="text-center">
                <i class="fas fa-exclamation-circle text-4xl mb-3 opacity-30"></i>
                <div class="text-sm">任务不存在或已删除</div>
              </div>
            </div>

            <div v-else class="flex-1 flex flex-col overflow-hidden">
              <div class="px-6 py-4 border-b border-white/5 bg-[#18181b] max-h-[46vh] overflow-y-auto no-scrollbar md:max-h-none md:overflow-visible">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="text-base text-white font-bold truncate">
                      {{ formatTaskTitle(videoExtractStore.selectedTask) }}
                    </div>
                    <div class="mt-1 text-xs text-gray-500 font-mono break-all">
                      taskId: {{ videoExtractStore.selectedTask.taskId }}
                    </div>
                  </div>

                  <div class="flex items-center gap-2">
                    <button
                      v-if="canPreviewSourceVideo(videoExtractStore.selectedTask)"
                      class="px-3 py-2 text-xs rounded-lg bg-white/10 hover:bg-white/15 text-gray-200 transition"
                      @click="openSourcePreview(videoExtractStore.selectedTask)"
                      title="预览源视频（支持倍速与抓帧）"
                    >
                      <i class="fas fa-play mr-1"></i>
                      源视频
                    </button>
                    <button
                      v-if="isRunning(videoExtractStore.selectedTask.status)"
                      class="px-3 py-2 text-xs rounded-lg bg-amber-600 hover:bg-amber-500 text-white transition"
                      @click="confirmCancel"
                    >
                      <i class="fas fa-stop mr-1"></i>
                      终止
                    </button>
                    <button
                      class="px-3 py-2 text-xs rounded-lg bg-red-600 hover:bg-red-500 text-white transition"
                      @click="confirmDelete"
                    >
                      <i class="fas fa-trash mr-1"></i>
                      删除
                    </button>
                  </div>
                </div>

                <div class="mt-4 grid grid-cols-2 md:grid-cols-4 gap-3">
                  <div class="rounded-lg bg-[#111113] border border-white/5 p-3">
                    <div class="text-[11px] text-gray-500 mb-1">状态</div>
                    <div class="text-sm text-white">{{ statusText(videoExtractStore.selectedTask.status) }}</div>
                  </div>
                  <div class="rounded-lg bg-[#111113] border border-white/5 p-3">
                    <div class="text-[11px] text-gray-500 mb-1">模式</div>
                    <div class="text-sm text-white">{{ modeText(videoExtractStore.selectedTask) }}</div>
                  </div>
                  <div class="rounded-lg bg-[#111113] border border-white/5 p-3">
                    <div class="text-[11px] text-gray-500 mb-1">帧数</div>
                    <div class="text-sm text-white font-mono">
                      {{ videoExtractStore.selectedTask.framesExtracted }} / {{ videoExtractStore.selectedTask.maxFrames }}
                    </div>
                  </div>
                  <div class="rounded-lg bg-[#111113] border border-white/5 p-3">
                    <div class="text-[11px] text-gray-500 mb-1">宽 × 高</div>
                    <div class="text-sm text-white font-mono">
                      {{ videoExtractStore.selectedTask.videoWidth }} × {{ videoExtractStore.selectedTask.videoHeight }}
                    </div>
                  </div>
                </div>

                <div class="mt-3 grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div class="rounded-lg bg-[#111113] border border-white/5 p-3">
                    <div class="text-[11px] text-gray-500 mb-1">限制摘要</div>
                    <div class="text-xs text-gray-300">
                      {{ limitText(videoExtractStore.selectedTask) }}
                    </div>
                  </div>
                  <div class="rounded-lg bg-[#111113] border border-white/5 p-3">
                    <div class="text-[11px] text-gray-500 mb-1">输出目录</div>
                    <div class="text-xs text-gray-300 font-mono break-all">
                      {{ videoExtractStore.selectedTask.outputDirLocalPath }}
                    </div>
                    <div v-if="videoExtractStore.selectedTask.outputDirUrl" class="mt-1 text-[11px] text-gray-500 break-all">
                      {{ videoExtractStore.selectedTask.outputDirUrl }}
                    </div>
                  </div>
                </div>

                <div v-if="videoExtractStore.selectedTask.lastError" class="mt-3 text-xs text-red-400">
                  {{ videoExtractStore.selectedTask.lastError }}
                </div>

                <div class="mt-3">
                  <div class="flex items-center justify-between text-xs text-gray-500 mb-1">
                    <span>进度</span>
                    <span class="font-mono">{{ formatPercent(progressPercent(videoExtractStore.selectedTask)) }}</span>
                  </div>
                  <div class="h-2 bg-black/30 rounded-full overflow-hidden">
                    <div class="h-full bg-purple-500/70" :style="{ width: `${progressPercent(videoExtractStore.selectedTask)}%` }"></div>
                  </div>
                  <div class="mt-2 flex items-center justify-between text-[11px] text-gray-500 font-mono">
                    <span>{{ formatCursor(videoExtractStore.selectedTask) }}</span>
                    <span v-if="videoExtractStore.selectedTask.runtime?.speed">speed={{ videoExtractStore.selectedTask.runtime.speed }}</span>
                  </div>
                </div>

                <!-- Continue section -->
                <div v-if="canContinue(videoExtractStore.selectedTask)" class="mt-4 rounded-xl border border-white/5 bg-[#111113] p-4">
                  <div class="flex items-center justify-between gap-3">
                    <div class="text-sm text-gray-200 font-medium">继续抽帧</div>
                    <span class="text-[11px] px-2 py-1 rounded-full bg-amber-500/15 text-amber-300 border border-amber-500/30">
                      {{ videoExtractStore.selectedTask.status === 'PAUSED_LIMIT' ? '因限制暂停' : '已终止' }}
                    </span>
                  </div>
                  <div class="mt-3 grid grid-cols-1 md:grid-cols-3 gap-3">
                    <div>
                      <label class="block text-xs text-gray-500 mb-1">新的 endSec（可空）</label>
                      <input
                        v-model.number="continueEndSec"
                        type="number"
                        min="0"
                        step="0.01"
                        class="w-full bg-[#18181b] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-purple-500 focus:outline-none text-sm"
                        placeholder="不修改则留空"
                      />
                    </div>
                    <div>
                      <label class="block text-xs text-gray-500 mb-1">新的 maxFrames（可空）</label>
                      <input
                        v-model.number="continueMaxFrames"
                        type="number"
                        min="1"
                        step="1"
                        class="w-full bg-[#18181b] text-white px-3 py-2 rounded-lg border border-white/10 focus:border-purple-500 focus:outline-none text-sm"
                        placeholder="不修改则留空"
                      />
                    </div>
                    <div class="flex items-end">
                      <button
                        class="w-full px-4 py-2 rounded-lg bg-purple-600 hover:bg-purple-500 text-white font-semibold transition disabled:opacity-50 disabled:cursor-not-allowed"
                        :disabled="continueSubmitting"
                        @click="doContinue"
                      >
                        {{ continueSubmitting ? '提交中...' : '继续' }}
                      </button>
                    </div>
                  </div>
                  <p class="mt-2 text-[11px] text-gray-500">
                    提示：maxFrames 不能小于已输出帧数；endSec 必须大于 startSec。
                  </p>
                </div>
              </div>

              <!-- Content: frames + logs -->
              <div class="flex-1 flex overflow-hidden">
                <div class="flex-1 flex flex-col overflow-hidden">
                  <div class="px-4 py-3 border-b border-white/5 flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <span class="text-sm text-gray-300 font-medium">帧图预览</span>
                      <span class="text-xs text-gray-500 font-mono">{{ videoExtractStore.frames.items.length }} 张</span>
                    </div>
                    <div class="flex items-center gap-2">
                      <button
                        class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
                        @click="toggleLayout"
                        :title="layoutMode === 'masonry' ? '切换到网格视图' : '切换到瀑布流视图'"
                      >
                        <i :class="layoutMode === 'masonry' ? 'fas fa-th' : 'fas fa-stream'"></i>
                      </button>
                    </div>
                  </div>

                  <div ref="framesScrollerHost" class="flex-1 overflow-hidden">
                    <!-- Grid: 虚拟滚动（优先） -->
                    <div
                      v-if="layoutMode === 'grid' && videoExtractStore.detailLoading && videoExtractStore.frames.items.length === 0"
                      class="flex h-full items-center justify-center"
                    >
                      <div class="text-center">
                        <div class="radar-spinner mx-auto mb-3"></div>
                        <p class="text-gray-500 text-sm">加载中...</p>
                      </div>
                    </div>

                    <RecycleScroller
                      v-else-if="layoutMode === 'grid'"
                      ref="framesScrollerRef"
                      :key="videoExtractStore.selectedTaskId"
                      class="h-full overflow-y-auto p-2 no-scrollbar"
                      :items="videoExtractStore.frames.items"
                      :item-size="gridItemSize"
                      :grid-items="gridColumns"
                      :item-secondary-size="gridItemSize"
                      key-field="seq"
                      :buffer="600"
                      :prerender="Math.min(120, gridColumns * 12)"
                      :emit-update="true"
                      @update="handleFramesScrollerUpdate"
                    >
	                      <template #default="{ item }">
	                        <div class="w-full h-full p-1">
	                          <MediaTile
	                            :src="item.url"
	                            type="image"
	                            class="w-full h-full rounded-xl overflow-hidden cursor-pointer border border-white/10 hover:border-purple-500 transition-colors bg-gray-800"
	                            :show-skeleton="false"
	                            @click="openPreview(item)"
	                          />
	                        </div>
	                      </template>

                      <template #empty>
                        <div class="flex items-center justify-center text-gray-500 text-sm h-full">
                          暂无帧图
                        </div>
                      </template>

                      <template #after>
                        <div v-if="videoExtractStore.detailLoading" class="flex justify-center py-6 text-gray-500 text-sm w-full">
                          <div class="flex items-center gap-2 bg-[#27272a] px-4 py-2 rounded-full shadow-lg">
                            <span class="w-4 h-4 border-2 border-purple-500 border-t-transparent rounded-full animate-spin"></span>
                            <span>加载更多...</span>
                          </div>
                        </div>
                        <div v-else-if="!videoExtractStore.frames.hasMore && videoExtractStore.frames.items.length > 0" class="flex justify-center py-8 text-gray-600 text-xs w-full">
                          <span class="px-3 py-1 bg-[#27272a]/50 rounded-full">已加载全部</span>
                        </div>
                      </template>
                    </RecycleScroller>

                    <!-- Masonry: 兼容保留（大量帧图时建议切换到网格虚拟滚动） -->
                    <InfiniteMediaGrid
                      v-else
                      :items="frameMasonryItems"
                      :loading="videoExtractStore.detailLoading"
                      :finished="!videoExtractStore.frames.hasMore"
                      :layout-mode="layoutMode"
                      :item-key="(item) => item.seq"
                      @load-more="videoExtractStore.loadMoreFrames"
                    >
	                      <template #default="{ item }">
	                        <MediaTile
	                          :src="item.url"
	                          type="image"
	                          class="w-full h-full rounded-xl overflow-hidden cursor-pointer border border-white/10 hover:border-purple-500 transition-colors bg-gray-800"
	                          :show-skeleton="false"
	                          @click="openPreview(item)"
	                        />
	                      </template>

                      <template #empty>
                        <div class="flex items-center justify-center text-gray-500 text-sm h-full">
                          暂无帧图
                        </div>
                      </template>

                      <template #finished-text>
                        已加载全部
                      </template>
                    </InfiniteMediaGrid>
                  </div>
                </div>

                <div class="w-[360px] hidden lg:flex flex-col border-l border-white/5 bg-[#1f1f23] overflow-hidden">
                  <div class="px-4 py-3 border-b border-white/5 flex items-center justify-between">
                    <div class="text-sm text-gray-300 font-medium">实时日志</div>
                    <div class="text-xs text-gray-500 font-mono">{{ (videoExtractStore.selectedTask.runtime?.logs || []).length }}</div>
                  </div>
                  <div class="flex-1 overflow-y-auto no-scrollbar p-3">
                    <div v-if="!videoExtractStore.selectedTask.runtime?.logs?.length" class="text-xs text-gray-500 text-center mt-6">
                      暂无日志
                    </div>
                    <div v-else class="space-y-1 font-mono text-[11px] text-gray-400">
                      <div v-for="(line, idx) in videoExtractStore.selectedTask.runtime.logs" :key="idx" class="break-all">
                        {{ line }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <Dialog
          v-model:visible="showCancelConfirm"
          title="确认终止任务"
          content="终止后可继续追加抽帧，已生成的图片会保留。确定终止？"
          show-warning
          confirm-button-class="bg-amber-600"
          @confirm="doCancel"
        />

        <Dialog
          v-model:visible="showDeleteConfirm"
          title="确认删除任务"
          :content="deleteConfirmContent"
          show-warning
          confirm-button-class="bg-red-600"
          @confirm="doDelete"
        />

        <MediaPreview
          v-model:visible="showPreview"
          :url="previewUrl"
          :type="previewType"
          :can-upload="false"
          :media-list="previewMediaList"
          @media-change="handlePreviewMediaChange"
        />
      </div>
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { breakpointsTailwind, useBreakpoints, useElementSize } from '@vueuse/core'
import { RecycleScroller } from 'vue-virtual-scroller'
import { useToast } from '@/composables/useToast'
import { useUserStore } from '@/stores/user'
import { useVideoExtractStore } from '@/stores/videoExtract'
import InfiniteMediaGrid from '@/components/common/InfiniteMediaGrid.vue'
import MediaTile from '@/components/common/MediaTile.vue'
import Dialog from '@/components/common/Dialog.vue'
import MediaPreview from '@/components/media/MediaPreview.vue'
import * as videoExtractApi from '@/api/videoExtract'
import type { UploadedMedia, VideoExtractTask } from '@/types'

const videoExtractStore = useVideoExtractStore()
const { show } = useToast()
const userStore = useUserStore()

const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = computed(() => !breakpoints.md.value)
const mobilePane = ref<'list' | 'detail'>('list')

const layoutMode = ref<'masonry' | 'grid'>((localStorage.getItem('video_extract_layout') as any) || 'grid')

const uploadInputRef = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const tempInputLocalPaths = ref<string[]>([])

const showCancelConfirm = ref(false)
const showDeleteConfirm = ref(false)
const deleteFiles = ref(true)

const continueEndSec = ref<number | null>(null)
const continueMaxFrames = ref<number | null>(null)
const continueSubmitting = ref(false)

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewMediaList = ref<UploadedMedia[]>([])

const framesScrollerHost = ref<HTMLElement | null>(null)
const framesScrollerRef = ref<any>(null)
const { width: framesScrollerWidth } = useElementSize(framesScrollerHost)

const gridColumns = computed(() => {
  const w = framesScrollerWidth.value || 0
  if (w < 420) return 2
  if (w < 640) return 3
  if (w < 900) return 4
  if (w < 1200) return 5
  return 6
})

const gridItemSize = computed(() => {
  const w = framesScrollerWidth.value || 0
  const cols = gridColumns.value || 1
  // RecycleScroller 不会自动扣除 padding，这里按 p-2(8px*2) 预留
  const available = Math.max(0, w - 16)
  const size = Math.floor(available / cols)
  return Math.max(84, size)
})

const cleanupTempInputs = async () => {
  const raw = Array.isArray(tempInputLocalPaths.value) ? tempInputLocalPaths.value : []
  if (raw.length === 0) return

  const unique = Array.from(new Set(raw.map(p => String(p || '').trim()).filter(Boolean))).filter(p => p.startsWith('/tmp/video_extract_inputs/'))
  tempInputLocalPaths.value = []

  if (unique.length === 0) return

  const results = await Promise.allSettled(unique.map(p => videoExtractApi.cleanupVideoExtractInput(p)))
  for (const r of results) {
    if (r.status === 'rejected') {
      console.warn('cleanupVideoExtractInput failed', r.reason)
    }
  }
}

const close = () => {
  void cleanupTempInputs()
  videoExtractStore.closeTaskModal()
}

const backToList = () => {
  mobilePane.value = 'list'
}

const refresh = async () => {
  await videoExtractStore.loadTasks(videoExtractStore.listPage)
}

const pickUploadFile = () => {
  if (uploading.value) return
  uploadInputRef.value?.click()
}

const handleUploadInputChange = async (e: Event) => {
  const input = e.target as HTMLInputElement | null
  const file = input?.files?.[0]
  if (input) input.value = ''
  if (!file) return

  uploading.value = true
  try {
    const form = new FormData()
    form.append('file', file)
    const res = await videoExtractApi.uploadVideoExtractInput(form)
    if (res?.code !== 0 || !res.data?.localPath) {
      throw new Error(String(res?.msg || '上传失败'))
    }

    const localPath = String(res.data.localPath || '').trim()
    if (localPath.startsWith('/tmp/video_extract_inputs/')) {
      tempInputLocalPaths.value = Array.from(new Set([...tempInputLocalPaths.value, localPath]))
    }

    const url = String(res.data.url || '').trim() || `${window.location.origin}/upload${localPath}`
    const media: UploadedMedia = {
      url,
      type: 'video',
      originalFilename: file.name,
      fileSize: file.size,
      fileType: file.type || res.data.contentType || undefined
    }

    const ok = await videoExtractStore.openCreateFromMedia(media, userStore.currentUser?.id)
    if (!ok) {
      show('当前文件不支持抽帧')
    }
  } catch (err: any) {
    console.error('uploadVideoExtractInput failed', err)
    show(String(err?.message || '上传失败'))
  } finally {
    uploading.value = false
  }
}

const openDetail = async (taskId: string) => {
  await videoExtractStore.openTaskDetail(taskId)
  continueEndSec.value = null
  continueMaxFrames.value = null
  if (isMobile.value) {
    mobilePane.value = 'detail'
  }
}

const isRunning = (status?: string) => status === 'PENDING' || status === 'PREPARING' || status === 'RUNNING'

const canContinue = (t: VideoExtractTask) => {
  return t.status === 'PAUSED_LIMIT' || t.status === 'PAUSED_USER'
}

const statusText = (status: string) => {
  switch (status) {
    case 'PENDING':
      return '排队中'
    case 'PREPARING':
      return '准备中'
    case 'RUNNING':
      return '运行中'
    case 'PAUSED_USER':
      return '已终止'
    case 'PAUSED_LIMIT':
      return '因限制暂停'
    case 'FINISHED':
      return '已完成'
    case 'FAILED':
      return '失败'
    default:
      return status
  }
}

const statusBadgeClass = (status: string) => {
  switch (status) {
    case 'RUNNING':
      return 'bg-emerald-500/15 text-emerald-300 border border-emerald-500/30'
    case 'PAUSED_LIMIT':
      return 'bg-amber-500/15 text-amber-300 border border-amber-500/30'
    case 'PAUSED_USER':
      return 'bg-gray-500/15 text-gray-300 border border-gray-500/30'
    case 'FAILED':
      return 'bg-red-500/15 text-red-300 border border-red-500/30'
    case 'FINISHED':
      return 'bg-blue-500/15 text-blue-300 border border-blue-500/30'
    default:
      return 'bg-white/10 text-gray-200 border border-white/10'
  }
}

const modeText = (t: VideoExtractTask) => {
  if (t.mode === 'fps') return `固定FPS ${t.fps || ''}`.trim()
  if (t.mode === 'all') return '每一帧'
  if (t.mode === 'keyframe') return t.keyframeMode === 'scene' ? `关键帧(场景 ${t.sceneThreshold ?? ''})`.trim() : '关键帧(I帧)'
  return t.mode
}

const formatTaskTitle = (t: VideoExtractTask) => {
  if (t.sourceType === 'mtPhoto') return `mtPhoto:${(t.sourceRef || '').slice(0, 8)}…`
  const ref = t.sourceRef || ''
  return ref.split('/').pop() || ref || t.taskId
}

const formatTaskSub = (t: VideoExtractTask) => {
  const timeRange = (t.startSec !== undefined || t.endSec !== undefined) ? `${t.startSec ?? 0}~${t.endSec ?? 'end'}s` : '全程'
  return `${modeText(t)} · ${timeRange}`
}

const progressPercent = (t: VideoExtractTask) => {
  // 优先以时间进度计算；缺失则按帧数上限计算
  const dur = t.durationSec || 0
  const cur = t.cursorOutTimeSec || 0
  if (dur > 0 && cur > 0) {
    const p = Math.round((cur / dur) * 100)
    return Math.max(0, Math.min(100, p))
  }
  if (t.maxFrames > 0) {
    const p = Math.round((t.framesExtracted / t.maxFrames) * 100)
    return Math.max(0, Math.min(100, p))
  }
  return 0
}

const formatPercent = (p: number) => `${p}%`

const formatCursor = (t: VideoExtractTask) => {
  const cur = t.cursorOutTimeSec
  if (!cur) return ''
  const dur = t.durationSec
  if (dur) return `t=${cur.toFixed(2)}s / ${dur.toFixed(2)}s`
  return `t=${cur.toFixed(2)}s`
}

const limitText = (t: VideoExtractTask) => {
  const time = (t.startSec !== undefined || t.endSec !== undefined) ? `时间区间: ${t.startSec ?? 0} ~ ${t.endSec ?? 'end'} (s)` : '时间区间: 全程'
  return `${time}；maxFrames=${t.maxFrames}`
}

const MAX_MASONRY_SAFE_ITEMS = 800
const MAX_PREVIEW_MEDIA_LIST_ITEMS = 10000

const toggleLayout = () => {
  if (layoutMode.value === 'grid') {
    if (videoExtractStore.frames.items.length > MAX_MASONRY_SAFE_ITEMS) {
      show(`帧数较多（>${MAX_MASONRY_SAFE_ITEMS}），已禁用瀑布流，请使用网格虚拟滚动`)
      return
    }
    layoutMode.value = 'masonry'
  } else {
    layoutMode.value = 'grid'
  }
  localStorage.setItem('video_extract_layout', layoutMode.value)
}

const frameMasonryItems = computed(() => {
  const t = videoExtractStore.selectedTask
  const w = Number(t?.videoWidth || 0) || undefined
  const h = Number(t?.videoHeight || 0) || undefined
  return videoExtractStore.frames.items.map(f => ({
    seq: f.seq,
    url: f.url,
    width: w,
    height: h
  }))
})

const handleFramesScrollerUpdate = (_startIndex: number, _endIndex: number, _visibleStartIndex: number, visibleEndIndex: number) => {
  if (videoExtractStore.detailLoading) return
  if (!videoExtractStore.frames.hasMore) return
  const total = videoExtractStore.frames.items.length
  if (!total) return

  // 接近底部时预取下一页（按 2~3 行阈值）
  const threshold = Math.max(0, total - gridColumns.value * 3)
  if (visibleEndIndex >= threshold) {
    videoExtractStore.loadMoreFrames()
  }
}

const openPreview = (item: any) => {
  previewType.value = 'image'
  previewUrl.value = item.url
  const t = videoExtractStore.selectedTask
  const w = Number(t?.videoWidth || 0) || undefined
  const h = Number(t?.videoHeight || 0) || undefined

  const frames = videoExtractStore.frames.items
  if (frames.length > MAX_PREVIEW_MEDIA_LIST_ITEMS) {
    previewMediaList.value = [{ url: item.url, type: 'image' }]
  } else {
    previewMediaList.value = frames.map(f => ({
      url: f.url,
      type: 'image' as const,
      width: w,
      height: h
    }))
  }
  showPreview.value = true
}

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

const canPreviewSourceVideo = (t?: VideoExtractTask | null) => {
  if (!t) return false
  if (t.sourceType !== 'upload') return false
  return !!buildUploadPreviewUrl(t.sourceRef || '')
}

const openSourcePreview = (t: VideoExtractTask) => {
  const url = buildUploadPreviewUrl(t.sourceRef || '')
  if (!url) {
    show('源视频不可预览')
    return
  }

  previewType.value = 'video'
  previewUrl.value = url
  previewMediaList.value = [{ url, type: 'video', originalFilename: t.sourceRef.split('/').pop() || undefined }]
  showPreview.value = true
}

const handlePreviewMediaChange = (media: UploadedMedia) => {
  previewUrl.value = media.url || previewUrl.value
  if (media.type === 'image' || media.type === 'video' || media.type === 'file') {
    previewType.value = media.type
  }
}

const confirmCancel = () => {
  showCancelConfirm.value = true
}

const doCancel = async () => {
  const t = videoExtractStore.selectedTask
  if (!t) return
  try {
    await videoExtractStore.cancelTask(t.taskId)
    show('已终止任务')
  } catch (e: any) {
    console.error(e)
    show(String(e?.message || '终止失败'))
  }
}

const deleteConfirmContent = computed(() => {
  return `删除任务记录，并${deleteFiles.value ? '同时删除' : '保留'}输出目录内的文件。确定删除？`
})

const confirmDelete = () => {
  deleteFiles.value = true
  showDeleteConfirm.value = true
}

const doDelete = async () => {
  const t = videoExtractStore.selectedTask
  if (!t) return
  try {
    await videoExtractStore.deleteTask({ taskId: t.taskId, deleteFiles: deleteFiles.value })
    show('已删除任务')
  } catch (e: any) {
    console.error(e)
    show(String(e?.message || '删除失败'))
  }
}

const doContinue = async () => {
  const t = videoExtractStore.selectedTask
  if (!t) return
  continueSubmitting.value = true
  try {
    const payload: any = { taskId: t.taskId }
    if (continueEndSec.value !== null) payload.endSec = continueEndSec.value
    if (continueMaxFrames.value !== null) payload.maxFrames = continueMaxFrames.value
    await videoExtractStore.continueTask(payload)
    show('已提交继续抽帧')
    continueEndSec.value = null
    continueMaxFrames.value = null
  } catch (e: any) {
    console.error(e)
    show(String(e?.message || '继续失败'))
  } finally {
    continueSubmitting.value = false
  }
}

watch(
  () => videoExtractStore.showTaskModal,
  async (v) => {
    if (!v) {
      mobilePane.value = 'list'
      return
    }
    tempInputLocalPaths.value = []
    mobilePane.value = isMobile.value && videoExtractStore.selectedTaskId ? 'detail' : 'list'
    await videoExtractStore.loadTasks(1)
  }
)

watch([gridItemSize, gridColumns], async () => {
  await nextTick()
  framesScrollerRef.value?.updateVisibleItems?.(true)
})

watch([isMobile, () => videoExtractStore.selectedTaskId], ([mobile, id]) => {
  if (!videoExtractStore.showTaskModal) return
  if (mobile) {
    mobilePane.value = id ? 'detail' : 'list'
  }
})
</script>
