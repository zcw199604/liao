<template>
  <teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-[80] bg-black/70 flex items-center justify-center p-4"
      @click="close"
    >
      <div 
        class="w-full max-w-5xl h-[85vh] bg-[#18181b] rounded-2xl shadow-2xl flex flex-col overflow-hidden border border-white/10" 
        @click.stop
      >
        <!-- Header -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-white/5 bg-[#1f1f23]">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-lg bg-blue-500/20 flex items-center justify-center">
              <i class="fas fa-search text-blue-400"></i>
            </div>
            <h3 class="text-lg font-bold text-white">图片查重工具</h3>
          </div>
          <button
            @click="close"
            class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition rounded-lg hover:bg-[#27272a]"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>

        <div class="flex flex-1 overflow-hidden flex-col md:flex-row">
          <!-- Left Panel: Input & Settings -->
          <div class="w-full md:w-1/3 md:min-w-[320px] bg-[#1f1f23] flex flex-col border-b md:border-b-0 md:border-r border-white/5 p-6 overflow-y-auto shrink-0">
            
            <!-- File Upload -->
            <div class="mb-6">
              <label class="block text-sm font-medium text-gray-400 mb-2">待检测文件</label>
              <div
                class="relative w-full aspect-square md:aspect-square h-48 md:h-auto rounded-xl border-2 border-dashed border-white/10 hover:border-blue-500 transition-colors bg-[#18181b] flex flex-col items-center justify-center overflow-hidden cursor-pointer group"
                @click="triggerFileSelect"
                @drop.prevent="handleDrop"
                @dragover.prevent
              >
                <MediaTile
                  v-if="previewUrl"
                  :src="previewUrl"
                  type="image"
                  fit="contain"
                  class="w-full h-full"
                  :show-skeleton="false"
                  :lazy="false"
                />
                <div v-else-if="selectedFile" class="text-center p-4">
                  <i class="fas fa-file-alt text-4xl text-gray-600 group-hover:text-blue-500 mb-3 transition-colors"></i>
                  <p class="text-sm text-gray-300 font-medium truncate max-w-[200px]">{{ selectedFile.name }}</p>
                  <p class="text-xs text-gray-500 mt-1">{{ (selectedFile.size / 1024).toFixed(1) }} KB</p>
                </div>
                <div v-else class="text-center p-4">
                  <i class="fas fa-cloud-upload-alt text-4xl text-gray-600 group-hover:text-blue-500 mb-3 transition-colors"></i>
                  <p class="text-sm text-gray-400">点击或拖拽文件到此处</p>
                  <p class="text-xs text-gray-600 mt-1">支持任意格式文件</p>
                </div>
                
                <!-- Overlay for change -->
                <div v-if="selectedFile" class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
                  <span class="text-white text-sm font-medium">更换文件</span>
                </div>

                <input
                  type="file"
                  ref="fileInputRef"
                  class="hidden"
                  @change="handleFileChange"
                />
              </div>
            </div>

            <!-- Settings -->
            <div class="space-y-6">
              <div>
                <div class="flex justify-between mb-2">
                  <label class="text-sm font-medium text-gray-400">相似度阈值 (Similarity)</label>
                  <span class="text-sm text-blue-400 font-bold">{{ (similarityThreshold * 100).toFixed(0) }}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.01"
                  v-model.number="similarityThreshold"
                  class="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
                />
                <p class="text-xs text-gray-500 mt-1">越接近 100% 越严格，建议 85% 以上</p>
              </div>

              <div>
                <div class="flex justify-between mb-2">
                  <label class="text-sm font-medium text-gray-400">最大返回数量 (Limit)</label>
                  <span class="text-sm text-gray-300">{{ limit }}</span>
                </div>
                <input
                  type="range"
                  min="1"
                  max="100"
                  step="1"
                  v-model.number="limit"
                  class="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
                />
              </div>
            </div>

            <!-- Action -->
            <div class="mt-8">
              <button
                @click="checkDuplicate"
                :disabled="!selectedFile || loading"
                class="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-bold transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
              >
                <i v-if="loading" class="fas fa-spinner fa-spin"></i>
                <i v-else class="fas fa-search"></i>
                <span>{{ loading ? '检测中...' : '开始查重' }}</span>
              </button>
            </div>
          </div>

          <!-- Right Panel: Results -->
          <div ref="resultsPanelRef" class="flex-1 bg-[#18181b] flex flex-col overflow-hidden relative min-h-[300px] md:min-h-0">
            
            <!-- Empty State -->
            <div v-if="!result && !loading" class="flex-1 flex flex-col items-center justify-center text-gray-600">
              <div class="w-20 h-20 rounded-full bg-[#1f1f23] flex items-center justify-center mb-4">
                <i class="fas fa-search text-3xl opacity-50"></i>
              </div>
              <p class="text-sm">上传图片并点击查重查看结果</p>
            </div>

            <!-- Loading State -->
            <div v-if="loading" class="absolute inset-0 z-10 bg-[#18181b]/80 flex flex-col items-center justify-center backdrop-blur-sm">
              <div class="radar-spinner mb-4"></div>
              <p class="text-gray-400 animate-pulse">正在比对特征库...</p>
            </div>

            <!-- Results List -->
            <div v-if="result" class="flex-1 flex flex-col h-full overflow-hidden">
              <!-- Result Header -->
              <div class="px-6 py-4 bg-[#1f1f23]/50 border-b border-white/5 shrink-0">
                <div class="flex items-center gap-3">
                  <div 
                    class="px-3 py-1 rounded-lg text-xs font-bold uppercase tracking-wider"
                    :class="{
                      'bg-green-500/20 text-green-400': result.matchType === 'none',
                      'bg-red-500/20 text-red-400': result.matchType === 'md5',
                      'bg-orange-500/20 text-orange-400': result.matchType === 'phash'
                    }"
                  >
                    {{ getMatchTypeText(result.matchType) }}
                  </div>
                  <span class="text-gray-400 text-sm">
                    共找到 <strong class="text-white">{{ (result.items || []).length }}</strong> 个相似结果
                  </span>
                  <span v-if="result.pHash" class="ml-auto text-xs text-gray-600 font-mono">
                    pHash: {{ result.pHash }}
                  </span>
                </div>
              </div>

              <!-- List -->
              <div class="flex-1 overflow-y-auto p-4 space-y-3 custom-scrollbar">
                <div 
                  v-for="(item, idx) in (result.items || [])" 
                  :key="item.id"
                  class="bg-[#1f1f23] rounded-xl p-3 flex gap-4 hover:bg-[#27272a] transition border border-transparent hover:border-white/10 group"
                >
                  <!-- Thumb -->
                  <div
                    class="w-24 h-24 shrink-0 bg-black rounded-lg overflow-hidden relative cursor-pointer hover:ring-2 hover:ring-blue-500 transition-all"
                    @click="openPreview(item)"
                    title="点击预览大图"
	                  >
	                    <MediaTile
	                      :src="getImgUrl(item.filePath)"
	                      type="image"
	                      fit="contain"
	                      class="w-full h-full"
	                      :show-skeleton="false"
	                      :lazy="false"
	                      @error="handleImgError"
	                    />
	                    <div class="absolute bottom-0 left-0 right-0 bg-black/60 text-[10px] text-white text-center py-0.5 backdrop-blur-sm">
	                      ID: {{ item.id }}
	                    </div>
                    <!-- 预览提示 -->
                    <div class="absolute inset-0 bg-black/40 opacity-0 hover:opacity-100 flex items-center justify-center transition-opacity">
                      <i class="fas fa-search-plus text-white text-lg"></i>
                    </div>
                  </div>

                  <!-- Info -->
                  <div class="flex-1 min-w-0 flex flex-col justify-center">
                    <div class="flex justify-between items-start mb-1">
                      <h4 class="text-white font-medium truncate pr-4 text-sm" :title="item.fileName">
                        {{ item.fileName }}
                      </h4>
                      <span 
                        class="text-xs font-bold"
                        :class="getSimilarityClass(item.similarity)"
                      >
                        {{ (item.similarity * 100).toFixed(1) }}% 相似
                      </span>
                    </div>

                    <div class="text-xs text-gray-500 space-y-1">
                      <p>MD5: <span class="font-mono text-gray-600">{{ item.md5Hash.substring(0, 16) }}...</span></p>
                      <p>日期: {{ item.createdAt || '未知' }}</p>
                      <p>路径: <span class="text-gray-600 truncate block max-w-xs" :title="item.filePath">{{ item.filePath }}</span></p>
                    </div>
                  </div>

                  <!-- Actions -->
                  <div class="flex flex-col justify-center gap-2">
                    <button
                      @click="openDetail(item)"
                      class="p-2 rounded-lg bg-[#27272a] hover:bg-gray-600 text-gray-400 hover:text-white transition"
                      title="查看详情"
                    >
                      <i class="fas fa-exclamation-circle text-blue-400"></i>
                    </button>
                    <a 
                      :href="getImgUrl(item.filePath)" 
                      target="_blank"
                      class="p-2 rounded-lg bg-[#27272a] hover:bg-blue-600 text-gray-400 hover:text-white transition"
                      title="打开原图"
                    >
                      <i class="fas fa-external-link-alt"></i>
                    </a>
                  </div>
                </div>

                <div v-if="!result.items || result.items.length === 0" class="text-center py-10">
                   <p class="text-green-500 font-medium mb-2">未发现重复图片</p>
                   <p class="text-gray-500 text-xs">{{ result.reason || '该图片在数据库中是唯一的' }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Detail Panel -->
    <MediaDetailPanel
      v-if="detailMedia"
      v-model:visible="showDetail"
      :media="detailMedia"
    />

    <!-- Preview Panel -->
    <MediaPreview
      v-model:visible="showPreview"
      :url="mediaPreviewUrl"
      :type="mediaPreviewType"
      :can-upload="false"
      :media-list="resultMediaList"
    />
  </teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'
import { useToast } from '@/composables/useToast'
	import { checkDuplicateMedia } from '@/api/media'
	import type { CheckDuplicateData, DuplicateCheckItem, UploadedMedia } from '@/types'
	import { useMediaStore } from '@/stores/media'
	import MediaTile from '@/components/common/MediaTile.vue'
	import MediaDetailPanel from './MediaDetailPanel.vue'
	import MediaPreview from './MediaPreview.vue'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const { show } = useToast()
const mediaStore = useMediaStore()

const fileInputRef = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const previewUrl = ref<string>('')
const loading = ref(false)

const similarityThreshold = ref(0.85)
const limit = ref(20)

const result = ref<CheckDuplicateData | null>(null)
const resultsPanelRef = ref<HTMLElement | null>(null)

// Detail Panel
const showDetail = ref(false)
const detailMedia = ref<UploadedMedia | null>(null)

// Preview Panel
const showPreview = ref(false)
const mediaPreviewUrl = ref('')
const mediaPreviewType = ref<'image' | 'video' | 'file'>('image')

// 将查重结果转换为 MediaPreview 需要的 UploadedMedia 格式
const resultMediaList = computed<UploadedMedia[]>(() => {
  if (!result.value?.items) return []
  return result.value.items.map(item => ({
    url: getImgUrl(item.filePath),
    type: 'image' as const,
    originalFilename: item.fileName,
    localFilename: item.fileName,
    fileSize: item.fileSize,
    uploadTime: item.createdAt,
    md5: item.md5Hash,
    pHash: item.pHash,
    similarity: item.similarity
  }))
})

// Ensure server address is loaded
if (!mediaStore.imgServer) {
    mediaStore.loadImgServer()
}

const close = () => {
  emit('update:visible', false)
}

const triggerFileSelect = () => {
  fileInputRef.value?.click()
}

const handleFileChange = (e: Event) => {
  const input = e.target as HTMLInputElement
  if (input.files && input.files[0]) {
    setFile(input.files[0])
  }
  input.value = ''
}

const handleDrop = (e: DragEvent) => {
  if (e.dataTransfer?.files && e.dataTransfer.files[0]) {
    const file = e.dataTransfer.files[0]
    setFile(file)
  }
}

const setFile = (file: File) => {
  selectedFile.value = file
  result.value = null // Reset result
  
  if (file.type.startsWith('image/')) {
    // Create preview
    const reader = new FileReader()
    reader.onload = (e) => {
      previewUrl.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
  } else {
    previewUrl.value = ''
  }
}

const checkDuplicate = async () => {
  if (!selectedFile.value) return

  loading.value = true
  result.value = null

  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    formData.append('similarityThreshold', String(similarityThreshold.value))
    formData.append('limit', String(limit.value))

    // Manual type assertion as api might return generic ApiResponse
    const res = await checkDuplicateMedia(formData)
    
    if (res.code === 0 && res.data) {
      result.value = res.data as CheckDuplicateData
      // 自动滚动到结果区域 (针对手机端)
      await nextTick()
      if (resultsPanelRef.value) {
        resultsPanelRef.value.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
    } else {
      show(res.msg || '查重失败')
    }
  } catch (e: any) {
    console.error('Check duplicate error:', e)
    show('查重请求出错')
  } finally {
    loading.value = false
  }
}

const openDetail = (item: DuplicateCheckItem) => {
  detailMedia.value = {
    url: getImgUrl(item.filePath),
    type: 'image', // Assuming image since it's image deduplication
    originalFilename: item.fileName,
    localFilename: item.fileName, // Using fileName as localFilename too
    fileSize: item.fileSize,
    uploadTime: item.createdAt,
    md5: item.md5Hash,
    pHash: item.pHash,
    similarity: item.similarity
  }
  showDetail.value = true
}

// 打开图片预览
const openPreview = (item: DuplicateCheckItem) => {
  mediaPreviewUrl.value = getImgUrl(item.filePath)
  mediaPreviewType.value = 'image'
  showPreview.value = true
}

const getMatchTypeText = (type: string) => {
  switch (type) {
    case 'md5': return 'MD5 精确命中'
    case 'phash': return 'pHash 相似命中'
    case 'none': return '无重复'
    default: return type
  }
}

const getSimilarityClass = (score: number) => {
  if (score >= 0.95) return 'text-red-500'
  if (score >= 0.85) return 'text-orange-500'
  return 'text-yellow-500'
}

const getImgUrl = (filePath: string) => {
  // 如果已经是完整 URL，直接返回
  if (filePath.startsWith('http')) return filePath

  // 本地文件路径（可能是 /upload/... 或 /lsp/tg/... 等），拼接当前页面的 origin
  if (filePath.startsWith('/')) {
    return `${window.location.origin}${filePath}`
  }

  // 如果路径不以 / 开头，添加前缀
  return `${window.location.origin}/${filePath}`
}

const handleImgError = (e: Event) => {
  const img = e.target as HTMLImageElement
  // img.src = '/placeholder.png' // Optional
  img.style.opacity = '0.3'
}

// Watch visibility to clear state if needed
watch(() => props.visible, (val) => {
  if (!val) {
    // Optional: clear state on close
    // selectedFile.value = null
    // previewUrl.value = ''
    // result.value = null
  }
})
</script>

<style scoped>
.radar-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid rgba(59, 130, 246, 0.3);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #3f3f46;
  border-radius: 3px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #52525b;
}
</style>
