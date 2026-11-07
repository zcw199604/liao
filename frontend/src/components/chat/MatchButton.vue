<template>
  <div class="absolute bottom-6 left-0 right-0 flex justify-center z-20 pointer-events-none">
    <div ref="rootRef" class="pointer-events-auto">
    <!-- 匹配按钮 -->
    <div
      v-if="!chatStore.isMatching"
      class="flex items-stretch rounded-full shadow-xl overflow-hidden"
    >
      <button
        type="button"
        @click="handleStartMatch"
        @touchstart.prevent="handleTouchStart"
        @touchend.prevent="handleTouchEnd"
        @touchcancel="handleTouchCancel"
        class="flex items-center px-7 py-4 bg-blue-600 hover:bg-blue-500 text-white font-bold text-lg transition active:scale-95"
      >
        <i class="fas fa-random mr-2"></i> 匹配新用户
      </button>
      <button
        type="button"
        aria-label="连续匹配选项"
        title="连续匹配选项"
        @click.stop="toggleMenu"
        class="w-14 flex items-center justify-center bg-blue-700 hover:bg-blue-600 text-white border-l border-white/20 transition active:scale-95"
      >
        <i class="fas fa-chevron-up text-sm"></i>
      </button>
    </div>

    <!-- 取消按钮（带进度显示） -->
    <button
      v-else
      @click="handleCancelMatch"
      class="flex flex-col items-center px-8 py-4 bg-red-600 rounded-full text-white font-bold text-lg shadow-2xl animate-pulse"
    >
      <div class="flex items-center">
        <i class="fas fa-stop mr-2"></i>
        <span v-if="!chatStore.continuousMatchConfig.enabled || chatStore.continuousMatchConfig.total === 1">取消匹配</span>
        <span v-else>取消连续匹配</span>
      </div>
      <span v-if="chatStore.continuousMatchConfig.enabled && chatStore.continuousMatchConfig.total > 1" class="text-xs mt-1 opacity-90">
        第 {{ chatStore.continuousMatchConfig.current }}/{{ chatStore.continuousMatchConfig.total }} 次
      </span>
    </button>

    <!-- 长按菜单 -->
    <div
      v-if="showMenu"
      class="absolute bottom-full left-1/2 -translate-x-1/2 mb-3 bg-surface rounded-xl shadow-2xl border border-line-strong overflow-hidden"
    >
      <button
        v-for="option in menuOptions"
        :key="option.count"
        @click="handleSelectCount(option.count)"
        class="w-full px-6 py-3 text-left hover:bg-surface-3 text-fg flex items-center gap-3 transition"
      >
        <i :class="option.icon" class="text-blue-400"></i>
        <span>连续匹配 {{ option.count }} 次</span>
      </button>

      <!-- 匹配偏好：仅晚风线路支持 -->
      <button
        v-if="matchPreferenceStore.supported"
        data-testid="match-preference-entry"
        @click="openPreferencePanel"
        class="w-full px-6 py-3 text-left hover:bg-surface-3 text-fg flex items-center gap-3 border-t border-line transition"
      >
        <i class="fas fa-sliders-h text-purple-400"></i>
        <span class="flex-1">匹配偏好</span>
        <span class="text-xs text-fg-muted">{{ matchPreferenceStore.summary }}</span>
      </button>
    </div>

    <!-- 匹配偏好面板 -->
    <div
      v-if="showPreferencePanel"
      data-testid="match-preference-panel"
      class="absolute bottom-full left-1/2 -translate-x-1/2 mb-3 w-72 bg-surface rounded-xl shadow-2xl border border-line-strong p-4 space-y-3"
      @click.stop
    >
      <div class="flex items-center justify-between">
        <span class="text-sm font-medium text-fg">匹配偏好</span>
        <button aria-label="关闭" @click="showPreferencePanel = false" class="text-fg-muted hover:text-fg">
          <i class="fas fa-times text-xs"></i>
        </button>
      </div>

      <div>
        <div class="text-xs text-fg-muted mb-1.5">期望对方性别</div>
        <div class="flex gap-2">
          <button
            v-for="opt in genderOptions"
            :key="opt.value"
            type="button"
            class="flex-1 py-1.5 rounded-lg text-xs border transition"
            :class="
              draft.preferGender === opt.value
                ? 'bg-purple-500/20 border-purple-500/50 text-purple-300'
                : 'bg-surface-2 border-line text-fg-muted hover:bg-surface-3'
            "
            @click="draft.preferGender = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <div>
        <div class="text-xs text-fg-muted mb-1.5">期望年龄（留空或 0 表示不限）</div>
        <div class="flex items-center gap-2">
          <input
            v-model.number="draft.preferAgeMin"
            type="number"
            min="0"
            max="120"
            placeholder="最小"
            class="w-full bg-surface-2 border border-line rounded-lg px-2 py-1.5 text-xs text-fg"
          />
          <span class="text-fg-muted text-xs">-</span>
          <input
            v-model.number="draft.preferAgeMax"
            type="number"
            min="0"
            max="120"
            placeholder="最大"
            class="w-full bg-surface-2 border border-line rounded-lg px-2 py-1.5 text-xs text-fg"
          />
        </div>
      </div>

      <label class="flex items-center gap-2 text-xs text-fg cursor-pointer">
        <input v-model="draft.preferSameProvince" type="checkbox" class="rounded" />
        <span>只匹配同省</span>
      </label>

      <button
        type="button"
        :disabled="matchPreferenceStore.saving"
        @click="savePreference"
        class="w-full py-2 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-xs font-medium transition disabled:opacity-60"
      >
        {{ matchPreferenceStore.saving ? '保存中…' : '保存' }}
      </button>
    </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useMatchPreferenceStore } from '@/stores/matchPreference'
import { useChat } from '@/composables/useChat'
import { useToast } from '@/composables/useToast'

const chatStore = useChatStore()
const userStore = useUserStore()
const { startContinuousMatch, cancelMatch, handleAutoMatch } = useChat()
const { show } = useToast()

const matchPreferenceStore = useMatchPreferenceStore()

const showMenu = ref(false)
const showPreferencePanel = ref(false)
// 编辑中的副本，点保存才提交，避免每改一项就发请求
const draft = ref({
  preferGender: 'any',
  preferAgeMin: 0,
  preferAgeMax: 0,
  preferSameProvince: false
})

const genderOptions = [
  { value: 'any', label: '不限' },
  { value: 'male', label: '男生' },
  { value: 'female', label: '女生' }
]
const longPressTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const isLongPress = ref(false)
const rootRef = ref<HTMLElement | null>(null)

const menuOptions = [
  { count: 3, icon: 'fas fa-dice-three' },
  { count: 5, icon: 'fas fa-hand-spock' },
  { count: 10, icon: 'fas fa-fire' }
]

const openMenu = () => {
  showMenu.value = true
}

// 打开匹配偏好面板：先把当前生效值拷进 draft
const openPreferencePanel = () => {
  showMenu.value = false
  const p = matchPreferenceStore.preference
  draft.value = {
    preferGender: p.preferGender || 'any',
    preferAgeMin: p.preferAgeMin || 0,
    preferAgeMax: p.preferAgeMax || 0,
    preferSameProvince: Boolean(p.preferSameProvince)
  }
  showPreferencePanel.value = true
}

const savePreference = async () => {
  const identityId = userStore.currentUser?.id
  if (!identityId) return

  const result = await matchPreferenceStore.save({ ...draft.value, identityId })
  show(result.success ? '匹配偏好已保存' : `保存失败：${result.message || '未知错误'}`)
  if (result.success) {
    showPreferencePanel.value = false
  }
}

const toggleMenu = () => {
  showMenu.value = !showMenu.value
}

const startLongPressTimer = () => {
  isLongPress.value = false
  longPressTimer.value = setTimeout(() => {
    isLongPress.value = true
    openMenu()
  }, 300)
}

const clearLongPressTimer = () => {
  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
    longPressTimer.value = null
  }
}

// 触摸长按仍保留为移动端连续匹配入口。
const handleTouchStart = (e: TouchEvent) => {
  e.preventDefault()
  startLongPressTimer()
}

const handleTouchEnd = (e: TouchEvent) => {
  e.preventDefault()
  clearLongPressTimer()
  if (!isLongPress.value) {
    handleStartMatch()
  }
}

const handleTouchCancel = () => {
  clearLongPressTimer()
}

// 单次匹配
const handleStartMatch = () => {
  const ok = startContinuousMatch(1)
  if (ok) {
    show('正在匹配...')
  }
}

// 选择连续匹配次数
const handleSelectCount = (count: number) => {
  showMenu.value = false
  const ok = startContinuousMatch(count)
  if (ok) {
    show(`开始连续匹配 ${count} 次...`)
  }
}

// 取消匹配
const handleCancelMatch = () => {
  cancelMatch()
  show('已取消匹配')
}

// 监听匹配成功事件，触发自动匹配
const handleMatchAutoCheck = () => {
  handleAutoMatch()
}

// 点击外部关闭菜单
const handleClickOutside = (e: MouseEvent) => {
  const target = e.target as Node | null
  // 偏好面板也挂在 rootRef 内，点内部不关闭
  if (target && rootRef.value?.contains(target)) return
  if (showMenu.value) showMenu.value = false
  if (showPreferencePanel.value) showPreferencePanel.value = false
}

onMounted(() => {
  window.addEventListener('match-auto-check', handleMatchAutoCheck)
  document.addEventListener('click', handleClickOutside)

  // 进入页面时加载当前身份的匹配偏好（unsupported 时入口不显示）
  const identityId = userStore.currentUser?.id
  if (identityId) {
    matchPreferenceStore.load(identityId)
  }
})

onUnmounted(() => {
  window.removeEventListener('match-auto-check', handleMatchAutoCheck)
  document.removeEventListener('click', handleClickOutside)
  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
  }
})
</script>
