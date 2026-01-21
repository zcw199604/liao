<template>
  <teleport to="body">
    <div
      v-if="douyinStore.showModal"
      class="fixed inset-0 z-[75] bg-black/70 flex items-center justify-center"
      @click="close"
    >
      <div
        class="w-[95%] max-w-3xl h-[85vh] bg-[#18181b] rounded-2xl shadow-2xl flex flex-col"
        @click.stop
      >
        <!-- 头部 -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
          <div class="flex items-center gap-2 min-w-0">
            <i class="fas fa-cloud-download-alt text-emerald-400 flex-shrink-0"></i>
            <h3 class="text-lg font-bold text-white truncate">抖音下载</h3>
          </div>

          <button
            @click="close"
            class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-6 no-scrollbar">
          <div class="space-y-3">
            <div class="text-xs text-gray-500">
              支持直接粘贴整段分享文本/短链/URL/作品ID；无需手动提取链接。
            </div>

            <textarea
              v-model="inputText"
              class="w-full min-h-[110px] bg-[#111113] border border-gray-700 rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500"
              placeholder="粘贴抖音分享文本/短链/完整URL/作品ID"
            ></textarea>

            <div class="flex flex-col sm:flex-row gap-2">
              <input
                v-model="proxy"
                :class="[
                  'flex-1 bg-[#111113] border rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500',
                  highlightConfig ? 'border-red-500' : 'border-gray-700'
                ]"
                placeholder="proxy（可选，例如 http://127.0.0.1:7890）"
              />
              <button
                class="px-4 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                @click="showAdvanced = !showAdvanced"
              >
                {{ showAdvanced ? '隐藏 Cookie' : '填写 Cookie' }}
              </button>
              <button
                v-if="showAdvanced && cookie"
                class="px-4 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                @click="clearCookie"
                title="清除本地保存的 Cookie"
              >
                清除
              </button>
            </div>

            <div v-if="showAdvanced" class="space-y-2">
              <textarea
                v-model="cookie"
                :class="[
                  'w-full min-h-[90px] bg-[#111113] border rounded-xl px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500',
                  highlightConfig ? 'border-red-500' : 'border-gray-700'
                ]"
                placeholder="抖音 Cookie（可选；仅本地保存，不会写入服务端）"
              ></textarea>
              <div class="text-xs text-red-400">
                Cookie 属敏感信息：仅保存在本地浏览器（localStorage），请勿在公共设备使用；如怀疑泄露请立即失效/更新。
              </div>
            </div>

            <div class="flex gap-2">
              <button
                class="flex-1 py-3 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition disabled:opacity-60 disabled:cursor-not-allowed"
                :disabled="loading || batchImport.running || batchDownload.running"
                @click="handleResolve"
              >
                {{ loading ? '解析中…' : '解析' }}
              </button>
              <button
                class="px-5 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                :disabled="loading || batchImport.running || batchDownload.running"
                @click="pasteFromClipboard"
              >
                粘贴
              </button>
              <button
                class="px-5 py-3 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                :disabled="loading || batchImport.running || batchDownload.running"
                @click="handleClear"
              >
                清空
              </button>
            </div>

            <div class="flex flex-col sm:flex-row gap-2 sm:items-center sm:justify-between text-xs text-gray-500">
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input type="checkbox" v-model="autoClipboard" class="accent-emerald-500" />
                <span>打开时自动读取剪贴板</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input type="checkbox" v-model="autoResolveClipboard" class="accent-emerald-500" />
                <span>读取后自动解析</span>
              </label>
            </div>

            <div v-if="cookieHint" class="text-xs text-amber-300">
              {{ cookieHint }}
            </div>

            <div v-if="error" class="text-sm text-red-400">
              {{ error }}
            </div>

            <div v-if="detail" class="pt-2">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-white font-medium text-sm truncate">
                    {{ detail.title }}
                  </div>
                  <div class="text-xs text-gray-500 mt-1">
                    类型: {{ detail.type }} · ID: {{ detail.detailId }}
                    <template v-if="detail.duration"> · 时长: {{ detail.duration }}</template>
                    <template v-if="detail.width && detail.height && detail.width > 0 && detail.height > 0">
                      · 分辨率: {{ detail.width }}×{{ detail.height }}
                    </template>
                  </div>
                </div>

                <button
                  v-if="userStore.currentUser"
                  class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-xs flex items-center gap-2"
                  @click="openUploadMenu"
                  title="打开聊天页上传菜单"
                >
                  <i class="fas fa-paper-plane"></i>
                  <span>上传菜单</span>
                </button>
              </div>

              <div v-if="detail.items && detail.items.length > 0" class="mt-4 space-y-2">
                <div class="flex flex-col sm:flex-row gap-2 sm:items-center sm:justify-between">
                  <div class="flex flex-wrap items-center gap-2 text-xs text-gray-400">
                    <button
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                      @click="toggleSelectionMode"
                      :disabled="batchImport.running || batchDownload.running"
                    >
                      {{ selectionMode ? '退出多选' : '多选模式' }}
                    </button>
                    <span v-if="selectionMode">已选 {{ selectedCount }} / {{ detail.items.length }}</span>
                    <button
                      v-if="selectionMode"
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                      @click="selectAll"
                      :disabled="batchImport.running || batchDownload.running"
                    >
                      全选
                    </button>
                    <button
                      v-if="selectionMode"
                      class="px-3 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition"
                      @click="clearSelection"
                      :disabled="batchImport.running || batchDownload.running"
                    >
                      清空选择
                    </button>
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <button
                      class="px-4 py-2 bg-[#27272a] hover:bg-gray-700 text-white rounded-xl border border-gray-700 transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                      :disabled="batchDownload.running || batchImport.running"
                      @click="handleBatchDownload"
                      title="浏览器可能会提示允许多个文件下载"
                    >
                      {{ selectionMode ? '下载选中' : '下载全部' }}
                    </button>
                    <button
                      class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-xl transition text-sm disabled:opacity-60 disabled:cursor-not-allowed"
                      :disabled="batchImport.running || batchDownload.running || !userStore.currentUser"
                      @click="handleBatchImport"
                    >
                      {{ selectionMode ? '导入选中' : '导入全部' }}
                    </button>
                  </div>
                </div>

                <div v-if="batchDownload.running" class="space-y-1">
                  <div class="text-xs text-gray-400">
                    下载中：{{ batchDownload.done }} / {{ batchDownload.total }}
                  </div>
                  <div class="h-2 bg-[#111113] rounded-full overflow-hidden border border-gray-800">
                    <div class="h-full bg-indigo-500/70" :style="{ width: `${batchDownloadPercent}%` }"></div>
                  </div>
                </div>

                <div v-if="batchImport.running" class="space-y-1">
                  <div class="text-xs text-gray-400">
                    导入中：{{ batchImport.done }} / {{ batchImport.total }}
                  </div>
                  <div class="h-2 bg-[#111113] rounded-full overflow-hidden border border-gray-800">
                    <div class="h-full bg-emerald-500/70" :style="{ width: `${batchImportPercent}%` }"></div>
                  </div>
                </div>
              </div>

              <div v-if="detail.items && detail.items.length > 0" class="mt-3 grid grid-cols-3 sm:grid-cols-4 gap-3">
                <button
                  v-for="item in detail.items"
                  :key="`douyin-item-${item.index}`"
                  class="aspect-square rounded-xl overflow-hidden border border-gray-700 transition relative bg-black/20"
                  :class="[
                    selectionMode && isSelected(item.index) ? 'border-emerald-500 ring-2 ring-emerald-500/30' : 'hover:border-emerald-500',
                    itemStateByIndex[item.index]?.status === 'error' ? 'border-red-500' : ''
                  ]"
                  @click="handleItemClick(item.index)"
                  :title="item.originalFilename || ''"
                >
                  <!-- 多选勾选 -->
                  <div
                    v-if="selectionMode"
                    class="absolute top-2 left-2 z-10 w-6 h-6 rounded-full border border-white/30 bg-black/40 backdrop-blur-md flex items-center justify-center"
                  >
                    <i v-if="isSelected(item.index)" class="fas fa-check text-emerald-300 text-xs"></i>
                  </div>

                  <!-- 状态 -->
                  <div v-if="itemStateByIndex[item.index]?.status && itemStateByIndex[item.index]?.status !== 'idle'" class="absolute top-2 right-2 z-10">
                    <span
                      v-if="itemStateByIndex[item.index]?.status === 'importing'"
                      class="px-2 py-1 rounded-full text-[10px] bg-emerald-600/80 text-white border border-emerald-400/30"
                    >
                      导入中…
                    </span>
                    <span
                      v-else-if="itemStateByIndex[item.index]?.status === 'imported'"
                      class="px-2 py-1 rounded-full text-[10px] bg-emerald-600/80 text-white border border-emerald-400/30"
                    >
                      已导入
                    </span>
                    <span
                      v-else-if="itemStateByIndex[item.index]?.status === 'exists'"
                      class="px-2 py-1 rounded-full text-[10px] bg-indigo-600/80 text-white border border-indigo-400/30"
                    >
                      已存在
                    </span>
                    <span
                      v-else-if="itemStateByIndex[item.index]?.status === 'error'"
                      class="px-2 py-1 rounded-full text-[10px] bg-red-600/80 text-white border border-red-400/30"
                      :title="itemStateByIndex[item.index]?.message || ''"
                    >
                      失败
                    </span>
                  </div>

                  <img
                    v-if="item.type === 'image'"
                    :src="item.url"
                    class="w-full h-full object-cover"
                    loading="lazy"
                  />
                  <video
                    v-else
                    :src="item.url"
                    class="w-full h-full object-cover"
                    muted
                    playsinline
                    preload="metadata"
                  ></video>
                  <div v-if="item.type === 'video'" class="absolute inset-0 flex items-center justify-center bg-black/30">
                    <i class="fas fa-play-circle text-white text-3xl"></i>
                  </div>

                  <!-- 文件大小（最佳努力） -->
	                  <div v-if="itemMetaByIndex[item.index]?.size" class="absolute bottom-2 left-2 z-10">
	                    <span class="px-2 py-1 rounded-full text-[10px] bg-black/50 text-white border border-white/10 backdrop-blur-md">
	                      {{ formatBytes(itemMetaByIndex[item.index]?.size || 0) }}
	                    </span>
	                  </div>
                </button>
              </div>

              <div v-else class="mt-4 text-sm text-gray-500">
                暂无可下载资源
              </div>
            </div>
          </div>
        </div>
      </div>

      <MediaPreview
        v-model:visible="showPreview"
        :url="previewUrl"
        :type="previewType"
        :can-upload="canUpload"
        :upload-disabled="previewUploadDisabled"
        :upload-loading="previewUploadLoading"
        :upload-text="previewUploadText"
        :media-list="previewMediaList"
        @upload="confirmImportUpload"
        @media-change="handlePreviewMediaChange"
      />
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useDouyinStore } from '@/stores/douyin'
import { useUserStore } from '@/stores/user'
import { useMediaStore } from '@/stores/media'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { useToast } from '@/composables/useToast'
import { generateCookie } from '@/utils/cookie'
import * as douyinApi from '@/api/douyin'
import MediaPreview from '@/components/media/MediaPreview.vue'
import type { UploadedMedia } from '@/types'

interface DouyinDetailItem {
  index: number
  type: 'image' | 'video'
  url: string
  downloadUrl: string
  originalFilename?: string
}

interface DouyinDetailResponse {
  key: string
  detailId: string
  type: string
  title: string
  coverUrl?: string
  duration?: string
  width?: number
  height?: number
  items: DouyinDetailItem[]
}

const douyinStore = useDouyinStore()
const userStore = useUserStore()
const mediaStore = useMediaStore()
const systemConfigStore = useSystemConfigStore()
const { show } = useToast()

const inputText = ref('')
const proxy = ref('')
const cookie = ref('')
const showAdvanced = ref(false)
const cookieHint = ref('')
const highlightConfig = ref(false)

const autoClipboard = ref(true)
const autoResolveClipboard = ref(false)

const selectionMode = ref(false)
const selectedIndices = ref<Set<number>>(new Set())

const loading = ref(false)
const error = ref('')
const detail = ref<DouyinDetailResponse | null>(null)

const showPreview = ref(false)
const previewUrl = ref('')
const previewType = ref<'image' | 'video' | 'file'>('image')
const previewMediaList = ref<UploadedMedia[]>([])
const previewIndex = ref(0)

const canUpload = computed(() => !!userStore.currentUser)

type ItemStatus = 'idle' | 'importing' | 'imported' | 'exists' | 'error'
type ItemState = { status: ItemStatus; message?: string }

const itemStateByIndex = reactive<Record<number, ItemState>>({})
const itemMetaByIndex = reactive<Record<number, { size?: number; mime?: string }>>({})

const batchImport = reactive({ running: false, total: 0, done: 0, success: 0, fail: 0 })
const batchDownload = reactive({ running: false, total: 0, done: 0, success: 0, fail: 0 })

const batchImportPercent = computed(() => (batchImport.total > 0 ? Math.round((batchImport.done / batchImport.total) * 100) : 0))
const batchDownloadPercent = computed(() => (batchDownload.total > 0 ? Math.round((batchDownload.done / batchDownload.total) * 100) : 0))

const restoreLocalConfig = () => {
  proxy.value = localStorage.getItem('douyin_proxy') || ''
  cookie.value = localStorage.getItem('douyin_cookie') || ''

  autoClipboard.value = localStorage.getItem('douyin_auto_clipboard') !== '0'
  autoResolveClipboard.value = localStorage.getItem('douyin_auto_resolve_clipboard') === '1'
}

const persistLocalConfig = () => {
  localStorage.setItem('douyin_proxy', proxy.value || '')
  localStorage.setItem('douyin_cookie', cookie.value || '')
  localStorage.setItem('douyin_auto_clipboard', autoClipboard.value ? '1' : '0')
  localStorage.setItem('douyin_auto_resolve_clipboard', autoResolveClipboard.value ? '1' : '0')
}

const isLikelyDouyinText = (value: string) => {
  const v = String(value || '').trim()
  if (!v) return false
  if (v.includes('v.douyin.com') || v.includes('www.douyin.com') || v.includes('douyin.com')) return true
  if (v.includes('modal_id=') || v.includes('aweme_id=')) return true
  if (/^\\d{6,}$/.test(v)) return true
  return false
}

const clearCookie = () => {
  cookie.value = ''
  localStorage.removeItem('douyin_cookie')
  show('已清除本地 Cookie')
}

const pasteFromClipboard = async () => {
  try {
    const txt = await navigator.clipboard?.readText?.()
    if (!txt || !isLikelyDouyinText(txt)) {
      show('剪贴板未识别到抖音内容')
      return
    }
    inputText.value = txt
    show('已从剪贴板填充')
    if (autoResolveClipboard.value) {
      await handleResolve()
    }
  } catch (e) {
    console.warn('read clipboard failed:', e)
    show('无法读取剪贴板（可能需要浏览器授权或 https）')
  }
}

const resetDetailStates = () => {
  Object.keys(itemStateByIndex).forEach((k) => delete itemStateByIndex[Number(k)])
  Object.keys(itemMetaByIndex).forEach((k) => delete itemMetaByIndex[Number(k)])
  selectionMode.value = false
  selectedIndices.value = new Set()
  batchImport.running = false
  batchImport.total = 0
  batchImport.done = 0
  batchImport.success = 0
  batchImport.fail = 0
  batchDownload.running = false
  batchDownload.total = 0
  batchDownload.done = 0
  batchDownload.success = 0
  batchDownload.fail = 0
}

watch(
  () => douyinStore.showModal,
  async (v) => {
    if (v) {
      restoreLocalConfig()
      error.value = ''
      detail.value = null
      cookieHint.value = ''
      highlightConfig.value = false
      resetDetailStates()
      showPreview.value = false
      previewUrl.value = ''
      previewMediaList.value = []
      previewIndex.value = 0

      // 优先使用调用方传入的预填内容；否则按设置尝试读取剪贴板
      if (!inputText.value && douyinStore.draftInput && isLikelyDouyinText(douyinStore.draftInput)) {
        inputText.value = douyinStore.draftInput
      } else if (!inputText.value && autoClipboard.value) {
        try {
          const txt = await navigator.clipboard?.readText?.()
          if (txt && isLikelyDouyinText(txt)) {
            inputText.value = txt
            show('已从剪贴板读取抖音内容')
          }
        } catch {
          // ignore
        }
      }

      if (autoResolveClipboard.value && inputText.value) {
        await handleResolve()
      }
    }
  }
)

const close = () => {
  persistLocalConfig()
  douyinStore.close()
  showPreview.value = false
  previewUrl.value = ''
  previewMediaList.value = []
  previewIndex.value = 0
  error.value = ''
  cookieHint.value = ''
  highlightConfig.value = false
  detail.value = null
  resetDetailStates()
}

const handleClear = () => {
  inputText.value = ''
  error.value = ''
  cookieHint.value = ''
  highlightConfig.value = false
  detail.value = null
  resetDetailStates()
}

const openUploadMenu = () => {
  mediaStore.requestOpenUploadMenu()
  show('已打开上传菜单（可在“已上传的文件”中发送）')
}

const toggleSelectionMode = () => {
  selectionMode.value = !selectionMode.value
  selectedIndices.value = new Set()
}

const isSelected = (idx: number) => selectedIndices.value.has(Number(idx))
const selectedCount = computed(() => selectedIndices.value.size)

const selectAll = () => {
  if (!detail.value?.items) return
  selectedIndices.value = new Set(detail.value.items.map((i) => Number(i.index)))
}

const clearSelection = () => {
  selectedIndices.value = new Set()
}

const toggleSelect = (idx: number) => {
  const next = new Set(selectedIndices.value)
  const v = Number(idx)
  if (next.has(v)) next.delete(v)
  else next.add(v)
  selectedIndices.value = next
}

const handleItemClick = (idx: number) => {
  if (selectionMode.value) {
    toggleSelect(idx)
    return
  }
  openPreview(idx)
}

const formatBytes = (bytes: number) => {
  const n = Number(bytes || 0)
  if (!Number.isFinite(n) || n <= 0) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let v = n
  let u = 0
  while (v >= 1024 && u < units.length - 1) {
    v /= 1024
    u += 1
  }
  return `${v.toFixed(u === 0 ? 0 : 1)}${units[u]}`
}

const getAuthToken = () => String(localStorage.getItem('authToken') || '').trim()

const prefetchMetas = async (items: DouyinDetailItem[]) => {
  const token = getAuthToken()
  if (!token) return

  const queue = items.map((i) => i.index)
  const concurrency = 4

  const worker = async () => {
    while (queue.length > 0) {
      const idx = Number(queue.shift())
      const item = items.find((i) => Number(i.index) === idx)
      if (!item) continue
      try {
        const resp = await fetch(item.downloadUrl, {
          method: 'HEAD',
          headers: { Authorization: `Bearer ${token}` }
        })
        if (!resp.ok) continue
        const len = Number(resp.headers.get('Content-Length') || 0)
        const mime = String(resp.headers.get('Content-Type') || '').trim()
        if (len > 0) itemMetaByIndex[idx] = { ...(itemMetaByIndex[idx] || {}), size: len, mime }
      } catch {
        // ignore
      }
    }
  }

  await Promise.all(Array.from({ length: concurrency }, () => worker()))
}

const handleResolve = async () => {
  const input = String(inputText.value || '').trim()
  if (!input) {
    show('请输入抖音分享文本/链接/作品ID')
    return
  }

  loading.value = true
  error.value = ''
  detail.value = null
  cookieHint.value = ''
  highlightConfig.value = false
  resetDetailStates()
  persistLocalConfig()

  try {
    const res = await douyinApi.getDouyinDetail({
      input,
      proxy: String(proxy.value || '').trim(),
      cookie: String(cookie.value || '').trim()
    })

    if (!res?.key || !Array.isArray(res?.items)) {
      error.value = res?.error || '解析失败'
      return
    }

    detail.value = res as DouyinDetailResponse

    // 预取文件大小（最佳努力）
    void prefetchMetas(detail.value.items || [])
  } catch (e: any) {
    console.error('解析抖音失败:', e)
    const msg = e?.response?.data?.error || e?.message || '解析失败'
    error.value = msg

    // 经验判断：Cookie/代理问题更常见，解析失败时引导用户填写
    if (String(msg).includes('获取数据失败') || String(msg).toLowerCase().includes('cookie') || String(msg).includes('风控')) {
      showAdvanced.value = true
      highlightConfig.value = true
      cookieHint.value = '提示：可能需要更新 Cookie 或设置代理（proxy）后重试。'
    }
  } finally {
    loading.value = false
  }
}

const buildPreviewMediaList = (items: DouyinDetailItem[]): UploadedMedia[] => {
  return items
    .slice()
    .sort((a, b) => Number(a.index) - Number(b.index))
    .map((it) => ({
      url: it.url,
      type: it.type,
      downloadUrl: it.downloadUrl,
      originalFilename: it.originalFilename
    }))
}

const openPreview = (idx: number) => {
  if (!detail.value) return
  const item = detail.value.items.find((i) => Number(i.index) === Number(idx))
  if (!item) return

  previewIndex.value = Number(item.index) || 0
  previewType.value = item.type
  previewUrl.value = item.url

  if (item.type === 'image') {
    const images = detail.value.items.filter((i) => i.type === 'image')
    previewMediaList.value = buildPreviewMediaList(images)
  } else {
    previewMediaList.value = buildPreviewMediaList([item])
  }

  showPreview.value = true
}

const handlePreviewMediaChange = (media: UploadedMedia) => {
  if (!detail.value) return
  const url = String(media?.url || '').trim()
  if (!url) return
  const item = detail.value.items.find((i) => i.url === url)
  if (item) {
    previewIndex.value = Number(item.index) || previewIndex.value
  }
  previewUrl.value = url
  previewType.value = media.type || previewType.value
}

const previewUploadLoading = computed(() => itemStateByIndex[previewIndex.value]?.status === 'importing')
const previewUploadDisabled = computed(() => {
  if (!userStore.currentUser) return true
  const st = itemStateByIndex[previewIndex.value]?.status
  return st === 'importing' || st === 'imported' || st === 'exists'
})
const previewUploadText = computed(() => {
  if (!userStore.currentUser) return ''
  const st = itemStateByIndex[previewIndex.value]?.status
  if (st === 'importing') return '导入中…'
  if (st === 'imported') return '已导入'
  if (st === 'exists') return '已存在（去重）'
  return `导入此${previewType.value === 'image' ? '图片' : (previewType.value === 'video' ? '视频' : '文件')}`
})

const ensureImgServer = async () => {
  if (!mediaStore.imgServer) {
    await mediaStore.loadImgServer()
  }
  return !!mediaStore.imgServer
}

const importIndex = async (idx: number) => {
  if (!userStore.currentUser) {
    show('请先选择身份后再导入上传')
    return { ok: false, dedup: false, error: '未选择身份' }
  }
  if (!detail.value?.key) return { ok: false, dedup: false, error: '解析信息缺失' }
  if (!await ensureImgServer()) return { ok: false, dedup: false, error: '图片服务器地址未获取' }

  const current = itemStateByIndex[idx]?.status
  if (current === 'imported' || current === 'exists') {
    return { ok: true, dedup: current === 'exists' }
  }
  if (current === 'importing') {
    return { ok: false, dedup: false, error: '导入进行中' }
  }

  itemStateByIndex[idx] = { status: 'importing' }

  const cookieData = generateCookie(userStore.currentUser.id, userStore.currentUser.name)
  const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
  const userAgent = navigator.userAgent

  try {
    const res = await douyinApi.importDouyinMedia({
      userid: userStore.currentUser.id,
      key: detail.value.key,
      index: idx,
      cookieData,
      referer,
      userAgent
    })

    if (res?.state === 'OK' && res.msg) {
      const port = String(res.port || await systemConfigStore.resolveImagePort(res.msg, mediaStore.imgServer))
      const remoteUrl = `http://${mediaStore.imgServer}:${port}/img/Upload/${res.msg}`

      if (!mediaStore.uploadedMedia.some((m) => m.url === remoteUrl)) {
        mediaStore.addUploadedMedia({
          url: remoteUrl,
          type: previewType.value,
          localFilename: res.localFilename
        })
      }

      const dedup = !!res.dedup
      itemStateByIndex[idx] = { status: dedup ? 'exists' : 'imported' }
      show(dedup ? '已存在（去重复用）' : '已导入上传（可在上传菜单发送）')
      return { ok: true, dedup }
    }

    itemStateByIndex[idx] = { status: 'error', message: String(res?.error || res?.msg || '导入失败') }
    return { ok: false, dedup: false, error: String(res?.error || res?.msg || '导入失败') }
  } catch (e: any) {
    console.error('导入失败:', e)
    const msg = e?.response?.data?.error || e?.message || '导入失败'
    itemStateByIndex[idx] = { status: 'error', message: String(msg) }
    return { ok: false, dedup: false, error: String(msg) }
  }
}

const handleBatchImport = async () => {
  if (!detail.value?.items?.length) return
  const targets = selectionMode.value && selectedIndices.value.size > 0
    ? Array.from(selectedIndices.value)
    : detail.value.items.map((i) => Number(i.index))

  if (!targets.length) return
  if (!userStore.currentUser) {
    show('请先选择身份后再导入上传')
    return
  }

  batchImport.running = true
  batchImport.total = targets.length
  batchImport.done = 0
  batchImport.success = 0
  batchImport.fail = 0

  try {
    for (const idx of targets.sort((a, b) => a - b)) {
      const r = await importIndex(idx)
      batchImport.done += 1
      if (r.ok) batchImport.success += 1
      else batchImport.fail += 1
    }
  } finally {
    batchImport.running = false
    show(`导入完成：成功 ${batchImport.success}，失败 ${batchImport.fail}`)
  }
}

const decodeMaybeEscapedFilename = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  try {
    return decodeURIComponent(raw)
  } catch {
    return raw
  }
}

const getFilenameFromContentDisposition = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  const m5987 = raw.match(/filename\\*\\s*=\\s*UTF-8''([^;]+)/i)
  if (m5987 && m5987[1]) return decodeMaybeEscapedFilename(m5987[1])
  const m = raw.match(/filename\\s*=\\s*\"([^\"]+)\"/i) || raw.match(/filename\\s*=\\s*([^;]+)/i)
  if (m && m[1]) return decodeMaybeEscapedFilename(m[1])
  return ''
}

const sanitizeFilename = (value: string): string => {
  const raw = String(value || '').trim()
  if (!raw) return ''
  return raw
    .replace(/[\\\\/:*?\"<>|]/g, '_')
    .replace(/[\\r\\n\\t]/g, ' ')
    .replace(/\\s+/g, ' ')
    .trim()
    .slice(0, 120)
}

const triggerBlobDownload = (blob: Blob, filename: string) => {
  const blobUrl = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = blobUrl
  link.download = filename || 'download'
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(blobUrl), 1000)
}

const downloadIndex = async (idx: number) => {
  if (!detail.value?.items) return { ok: false, error: '解析信息缺失' }
  const item = detail.value.items.find((i) => Number(i.index) === Number(idx))
  if (!item) return { ok: false, error: '资源不存在' }

  const token = getAuthToken()
  if (!token) return { ok: false, error: '未登录或Token缺失' }

  const resp = await fetch(item.downloadUrl, {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` }
  })
  if (!resp.ok) {
    let msg = `下载失败: ${resp.status}`
    try {
      const data = await resp.json()
      msg = data?.error || data?.msg || msg
    } catch {
      // ignore
    }
    return { ok: false, error: msg }
  }

  const blob = await resp.blob()
  const cd = resp.headers.get('Content-Disposition') || ''
  const filenameFromHeader = getFilenameFromContentDisposition(cd)
  const filename = filenameFromHeader || item.originalFilename || `download_${idx}`
  triggerBlobDownload(blob, sanitizeFilename(filename) || 'download')
  return { ok: true }
}

const handleBatchDownload = async () => {
  if (!detail.value?.items?.length) return
  const targets = selectionMode.value && selectedIndices.value.size > 0
    ? Array.from(selectedIndices.value)
    : detail.value.items.map((i) => Number(i.index))
  if (!targets.length) return

  if (targets.length >= 3) {
    const ok = window.confirm(`即将下载 ${targets.length} 个文件，浏览器可能会提示允许多个下载，是否继续？`)
    if (!ok) return
  }

  batchDownload.running = true
  batchDownload.total = targets.length
  batchDownload.done = 0
  batchDownload.success = 0
  batchDownload.fail = 0

  try {
    for (const idx of targets.sort((a, b) => a - b)) {
      const r = await downloadIndex(idx)
      batchDownload.done += 1
      if (r.ok) batchDownload.success += 1
      else batchDownload.fail += 1
    }
  } finally {
    batchDownload.running = false
    show(`下载完成：成功 ${batchDownload.success}，失败 ${batchDownload.fail}`)
  }
}

const confirmImportUpload = async () => {
  const idx = Number(previewIndex.value) || 0
  const r = await importIndex(idx)
  if (!r.ok) {
    show(`导入失败: ${r.error || '未知错误'}`)
  }
}
</script>
