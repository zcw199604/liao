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
            <h3 class="text-base font-bold text-fg">从其他身份接入</h3>
            <p class="text-xs text-fg-subtle truncate">{{ sourceLabel }}</p>
          </div>
          <button class="ui-icon-btn ui-icon-btn-ghost" aria-label="关闭" @click="close">
            <i class="fas fa-times"></i>
          </button>
        </div>

        <div class="px-5 py-3 border-b border-line space-y-3">
          <div class="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
            <select
              v-model="selectedSourceIdentityId"
              class="min-w-0 rounded-xl border border-line bg-surface-2 px-3 py-2 text-sm text-fg outline-none focus:ring-2 focus:ring-blue-500/30"
              @change="loadCandidates"
            >
              <option value="">选择来源身份</option>
              <option
                v-for="identity in sourceIdentities"
                :key="identity.id"
                :value="identity.id"
              >
                {{ identity.name }} ({{ identity.id }})
              </option>
            </select>
            <button
              class="ui-btn-secondary px-3 py-2 text-sm"
              :disabled="!selectedSourceIdentityId || loading"
              @click="loadCandidates"
            >
              <i class="fas fa-sync-alt text-xs"></i>
            </button>
          </div>

          <div class="relative">
            <i class="fas fa-search absolute left-3 top-1/2 -translate-y-1/2 text-xs text-fg-muted"></i>
            <input
              v-model="keyword"
              type="text"
              placeholder="搜索用户"
              class="w-full rounded-xl border border-line bg-surface-2 py-2 pl-9 pr-9 text-sm text-fg placeholder:text-fg-muted outline-none focus:ring-2 focus:ring-blue-500/30"
              @keyup.enter="loadCandidates"
            />
            <button
              v-if="keyword"
              class="absolute right-2 top-1/2 -translate-y-1/2 w-5 h-5 rounded-full bg-surface-3 text-fg-muted hover:text-fg transition"
              aria-label="清空搜索"
              @click="keyword = ''; loadCandidates()"
            >
              <i class="fas fa-times text-[10px]"></i>
            </button>
          </div>
        </div>

        <div v-if="warnings.length > 0" class="px-5 py-2 border-b border-line bg-amber-500/10 text-amber-300 text-xs">
          {{ warningText }}
        </div>

        <div class="flex-1 min-h-0 overflow-y-auto p-4 no-scrollbar">
          <div v-if="loading" class="py-12 text-center text-fg-subtle">
            <div class="radar-spinner mx-auto mb-3"></div>
            <p class="text-sm">加载中...</p>
          </div>

          <div v-else-if="!selectedSourceIdentityId" class="py-14 text-center text-fg-subtle">
            <i class="fas fa-user-friends text-4xl opacity-40 mb-3"></i>
            <p class="text-sm">请选择来源身份</p>
          </div>

          <div v-else-if="filteredCandidates.length === 0" class="py-14 text-center text-fg-subtle">
            <i class="fas fa-search text-4xl opacity-40 mb-3"></i>
            <p class="text-sm">暂无可接入用户</p>
          </div>

          <div v-else class="space-y-2">
            <button
              v-for="candidate in filteredCandidates"
              :key="candidate.targetUserId"
              class="w-full ui-list-item flex items-center p-3 text-left"
              @click="selectCandidate(candidate)"
            >
              <div
                :class="getColorClass(candidate.targetUserId)"
                class="w-11 h-11 rounded-xl flex items-center justify-center text-white font-bold text-base shadow shrink-0"
              >
                {{ displayName(candidate).charAt(0).toUpperCase() }}
              </div>

              <div class="ml-3 flex-1 min-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="font-semibold text-sm text-fg truncate">{{ displayName(candidate) }}</span>
                  <span
                    v-if="candidate.localArchived"
                    class="chat-user-tag-local-archived shrink-0"
                    title="本地归档"
                  >
                    <i class="fas fa-database text-[9px]"></i>
                    <span>归档</span>
                  </span>
                  <span
                    v-if="candidate.sources.includes('history')"
                    class="px-1.5 py-0.5 rounded text-[10px] bg-blue-500/15 text-blue-300 shrink-0"
                  >
                    历史
                  </span>
                  <span
                    v-if="candidate.sources.includes('favorite')"
                    class="px-1.5 py-0.5 rounded text-[10px] bg-yellow-500/15 text-yellow-300 shrink-0"
                  >
                    收藏
                  </span>
                </div>
                <div class="mt-1 flex items-center gap-2 min-w-0 text-xs text-fg-subtle">
                  <span class="truncate">{{ candidate.targetUserId }}</span>
                  <span v-if="candidate.address || candidate.area" class="truncate">| {{ candidate.address || candidate.area }}</span>
                </div>
                <p class="mt-1 text-xs text-fg-muted truncate">{{ candidate.lastMsg || '暂无消息' }}</p>
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
import { useIdentityStore } from '@/stores/identity'
import { useUserStore } from '@/stores/user'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'
import { getColorClass } from '@/constants/colors'
import type { ContactCandidate } from '@/types'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'select': [candidate: ContactCandidate, sourceIdentityId: string]
}>()

const identityStore = useIdentityStore()
const userStore = useUserStore()

const selectedSourceIdentityId = ref('')
const keyword = ref('')
const candidates = ref<ContactCandidate[]>([])
const warnings = ref<string[]>([])
const loading = ref(false)

const sourceIdentities = computed(() => {
  const currentId = String(userStore.currentUser?.id || '')
  return identityStore.identityList.filter(identity => String(identity.id) !== currentId)
})

const sourceIdentity = computed(() => {
  return sourceIdentities.value.find(identity => identity.id === selectedSourceIdentityId.value)
})

const sourceLabel = computed(() => {
  if (!sourceIdentity.value) return '选择一个来源身份'
  return `${sourceIdentity.value.name} -> ${userStore.currentUser?.name || '当前身份'}`
})

const warningText = computed(() => {
  if (warnings.value.length === 0) return ''
  return '部分上游数据不可用，已显示可用归档'
})

const normalizedKeyword = computed(() => keyword.value.trim().toLowerCase())
const filteredCandidates = computed(() => {
  const kw = normalizedKeyword.value
  if (!kw) return candidates.value
  return candidates.value.filter(candidate => {
    const fields = [
      candidate.targetUserId,
      candidate.targetUserName,
      candidate.name,
      candidate.nickname,
      candidate.area,
      candidate.address,
      candidate.lastMsg
    ]
    return fields.some(field => String(field || '').toLowerCase().includes(kw))
  })
})

const displayName = (candidate: ContactCandidate) => {
  return candidate.nickname || candidate.name || candidate.targetUserName || candidate.targetUserId
}

const close = () => {
  emit('update:visible', false)
}

const loadCandidates = async () => {
  const sourceIdentityId = selectedSourceIdentityId.value
  if (!sourceIdentityId) {
    candidates.value = []
    warnings.value = []
    return
  }

  loading.value = true
  try {
    const identityName = sourceIdentity.value?.name || sourceIdentityId
    const cookieData = identityStore.getIdentityCookie(sourceIdentityId) || generateCookie(sourceIdentityId, identityName)
    const res = await chatApi.getContactCandidates({
      sourceIdentityId,
      includeUpstream: true,
      q: '',
      limit: 300,
      cookieData,
      referer: 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko',
      userAgent: navigator.userAgent
    })
    candidates.value = Array.isArray(res.data?.items) ? res.data.items : []
    warnings.value = Array.isArray(res.warnings) ? res.warnings : []
  } catch (e) {
    console.error('加载跨身份候选失败:', e)
    candidates.value = []
    warnings.value = ['load failed']
  } finally {
    loading.value = false
  }
}

const selectCandidate = (candidate: ContactCandidate) => {
  const sourceIdentityId = selectedSourceIdentityId.value
  if (!sourceIdentityId) return
  emit('select', candidate, sourceIdentityId)
  close()
}

watch(
  () => props.visible,
  async (visible) => {
    if (!visible) return
    if (identityStore.identityList.length === 0) {
      await identityStore.loadList()
    }
    if (!selectedSourceIdentityId.value && sourceIdentities.value.length > 0) {
      selectedSourceIdentityId.value = sourceIdentities.value[0]?.id || ''
      await loadCandidates()
    }
  }
)
</script>
