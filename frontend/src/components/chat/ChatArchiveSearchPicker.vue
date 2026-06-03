<template>
  <teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-[80] bg-black/60 backdrop-blur-sm flex items-center justify-center p-4"
      @click="close"
    >
      <div class="w-full max-w-xl max-h-[82vh] bg-surface border border-line rounded-2xl shadow-2xl flex flex-col overflow-hidden" @click.stop>
        <div class="px-5 py-4 border-b border-line flex items-center justify-between gap-3">
          <div class="min-w-0">
            <h3 class="text-base font-bold text-fg">全局归档搜索</h3>
            <p class="text-xs text-fg-subtle truncate">按用户 ID 或归档名称查找所有身份下的会话</p>
          </div>
          <button class="ui-icon-btn ui-icon-btn-ghost" aria-label="关闭" @click="close">
            <i class="fas fa-times"></i>
          </button>
        </div>

        <div class="px-5 py-3 border-b border-line">
          <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
            <div class="relative min-w-0">
              <i class="fas fa-search absolute left-3 top-1/2 -translate-y-1/2 text-xs text-fg-muted"></i>
              <input
                v-model="keyword"
                type="text"
                placeholder="搜索归档用户 ID 或名称"
                class="w-full rounded-xl border border-line bg-surface-2 py-2 pl-9 pr-9 text-sm text-fg placeholder:text-fg-muted outline-none focus:ring-2 focus:ring-blue-500/30"
                @keyup.enter="search"
              />
              <button
                v-if="keyword"
                class="absolute right-2 top-1/2 -translate-y-1/2 w-5 h-5 rounded-full bg-surface-3 text-fg-muted hover:text-fg transition"
                aria-label="清空搜索"
                @click="clearSearch"
              >
                <i class="fas fa-times text-[10px]"></i>
              </button>
            </div>
            <button
              class="ui-btn-secondary px-3 py-2 text-sm"
              :disabled="!normalizedKeyword || loading"
              @click="search"
            >
              <i class="fas fa-search text-xs"></i>
            </button>
          </div>
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto p-4 no-scrollbar">
          <div v-if="loading" class="py-12 text-center text-fg-subtle">
            <div class="radar-spinner mx-auto mb-3"></div>
            <p class="text-sm">加载中...</p>
          </div>

          <div v-else-if="!searched" class="py-14 text-center text-fg-subtle">
            <i class="fas fa-database text-4xl opacity-40 mb-3"></i>
            <p class="text-sm">输入用户 ID 或名称开始搜索</p>
          </div>

          <div v-else-if="items.length === 0" class="py-14 text-center text-fg-subtle">
            <i class="fas fa-search text-4xl opacity-40 mb-3"></i>
            <p class="text-sm">未找到归档用户</p>
          </div>

          <div v-else class="space-y-2">
            <button
              v-for="item in items"
              :key="`${item.ownerUserId}:${item.targetUserId}`"
              class="w-full ui-list-item flex items-center p-3 text-left"
              @click="selectItem(item)"
            >
              <div
                :class="getColorClass(item.targetUserId)"
                class="w-11 h-11 rounded-xl flex items-center justify-center text-white font-bold text-base shadow shrink-0"
              >
                {{ displayName(item).charAt(0).toUpperCase() }}
              </div>

              <div class="ml-3 flex-1 min-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="font-semibold text-sm text-fg truncate">
                    <template v-for="(part, index) in getHighlightParts(displayName(item))" :key="`${item.ownerUserId}-${item.targetUserId}-name-${index}`">
                      <span :class="part.match ? 'search-highlight rounded px-0.5' : ''">{{ part.text }}</span>
                    </template>
                  </span>
                  <span class="chat-user-tag-local-archived shrink-0">
                    <i class="fas fa-database text-[9px]"></i>
                    <span>归档</span>
                  </span>
                  <span
                    v-if="item.sources.includes('history')"
                    class="px-1.5 py-0.5 rounded text-[10px] bg-blue-500/15 text-blue-300 shrink-0"
                  >
                    历史
                  </span>
                  <span
                    v-if="item.sources.includes('favorite')"
                    class="px-1.5 py-0.5 rounded text-[10px] bg-yellow-500/15 text-yellow-300 shrink-0"
                  >
                    收藏
                  </span>
                </div>
                <div class="mt-1 flex items-center gap-2 min-w-0 text-xs text-fg-subtle">
                  <span class="truncate">
                    <template v-for="(part, index) in getHighlightParts(item.targetUserId)" :key="`${item.ownerUserId}-${item.targetUserId}-id-${index}`">
                      <span :class="part.match ? 'search-highlight rounded px-0.5' : ''">{{ part.text }}</span>
                    </template>
                  </span>
                  <span class="truncate">| {{ item.ownerUserId }}</span>
                  <span v-if="item.address || item.area" class="truncate">| {{ item.address || item.area }}</span>
                </div>
                <p class="mt-1 text-xs text-fg-muted truncate">{{ item.lastMsg || '暂无消息' }}</p>
              </div>

              <i class="fas fa-chevron-right text-xs text-fg-muted ml-2"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import * as chatApi from '@/api/chat'
import { getColorClass } from '@/constants/colors'
import type { ChatArchiveSearchItem } from '@/types'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'select': [item: ChatArchiveSearchItem]
}>()

const keyword = ref('')
const items = ref<ChatArchiveSearchItem[]>([])
const loading = ref(false)
const searched = ref(false)

const normalizedKeyword = computed(() => keyword.value.trim())

const displayName = (item: ChatArchiveSearchItem) => {
  return item.nickname || item.name || item.targetUserName || item.targetUserId
}

interface HighlightPart {
  text: string
  match: boolean
}

const getHighlightParts = (raw: string | null | undefined): HighlightPart[] => {
  const text = String(raw || '')
  const keywordText = normalizedKeyword.value.toLowerCase()
  if (!text || !keywordText) {
    return [{ text, match: false }]
  }

  const lowerText = text.toLowerCase()
  if (!lowerText.includes(keywordText)) {
    return [{ text, match: false }]
  }

  const parts: HighlightPart[] = []
  let lastIndex = 0
  let index = lowerText.indexOf(keywordText)
  while (index !== -1) {
    if (index > lastIndex) {
      parts.push({ text: text.slice(lastIndex, index), match: false })
    }
    parts.push({ text: text.slice(index, index + keywordText.length), match: true })
    lastIndex = index + keywordText.length
    index = lowerText.indexOf(keywordText, lastIndex)
  }
  if (lastIndex < text.length) {
    parts.push({ text: text.slice(lastIndex), match: false })
  }
  return parts
}

const close = () => {
  emit('update:visible', false)
}

const clearSearch = () => {
  keyword.value = ''
  items.value = []
  searched.value = false
}

const search = async () => {
  const q = normalizedKeyword.value
  if (!q) {
    clearSearch()
    return
  }

  loading.value = true
  searched.value = true
  try {
    const res = await chatApi.searchChatArchive({ q, limit: 100 })
    items.value = Array.isArray(res.data?.items) ? res.data.items : []
  } catch (error) {
    console.error('搜索归档用户失败:', error)
    items.value = []
  } finally {
    loading.value = false
  }
}

const selectItem = (item: ChatArchiveSearchItem) => {
  emit('select', item)
  close()
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) return
    clearSearch()
  }
)
</script>
