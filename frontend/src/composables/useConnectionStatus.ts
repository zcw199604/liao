import { computed, onScopeDispose, ref, watch } from 'vue'
import { useChatStore } from '@/stores/chat'
import type { ConnectionState } from '@/components/chat/ChatHeader.vue'

// WebSocket 连接状态展示：在线 / 重连中 / 暂时禁止（带倒计时）/ 离线
export const useConnectionStatus = () => {
  const chatStore = useChatStore()
  const now = ref(Date.now())
  let timer: ReturnType<typeof setInterval> | null = null

  const stopTimer = () => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  // 被禁止期间每秒刷新一次，用于倒计时展示
  watch(
    () => chatStore.wsForbiddenUntil,
    (until) => {
      stopTimer()
      now.value = Date.now()
      if (until <= now.value) return
      timer = setInterval(() => {
        now.value = Date.now()
        if (now.value >= until) stopTimer()
      }, 1000)
    },
    { immediate: true }
  )

  onScopeDispose(stopTimer)

  const state = computed<ConnectionState>(() => {
    if (chatStore.wsConnected) return 'online'
    if (chatStore.wsForbiddenUntil > now.value) return 'forbidden'
    if (chatStore.wsReconnecting) return 'reconnecting'
    return 'offline'
  })

  const text = computed(() => {
    switch (state.value) {
      case 'online':
        return '在线'
      case 'forbidden':
        return `暂停连接 ${Math.max(1, Math.ceil((chatStore.wsForbiddenUntil - now.value) / 1000))}s`
      case 'reconnecting':
        return '重连中…'
      default:
        return '离线'
    }
  })

  return { state, text }
}
