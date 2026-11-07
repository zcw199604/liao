import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getProviderList, type ProviderInfo } from '@/api/provider'

// 线路选择持久化 key（按浏览器保存，切换身份不影响线路偏好）
export const WS_PROVIDER_STORAGE_KEY = 'liao-ws-provider'

// 默认/兜底线路（后端未返回列表时也保证能正常连接）
export const DEFAULT_PROVIDER_ID = 'chat2019'

const readPersistedProvider = (): string => {
  try {
    return String(localStorage.getItem(WS_PROVIDER_STORAGE_KEY) || '')
  } catch {
    return ''
  }
}

export const useProviderStore = defineStore('provider', () => {
  const providers = ref<ProviderInfo[]>([])
  const selectedId = ref<string>(readPersistedProvider())
  const loading = ref(false)
  const loaded = ref(false)

  // 当前实际生效的线路 id：优先用户选择，其次后端默认线路，最后兜底
  const currentId = computed(() => {
    const chosen = selectedId.value
    if (chosen && providers.value.some(p => p.id === chosen)) return chosen
    const fallback = providers.value.find(p => p.default)
    if (fallback) return fallback.id
    const first = providers.value[0]
    if (first) return first.id
    return chosen || DEFAULT_PROVIDER_ID
  })

  const currentProvider = computed(() => providers.value.find(p => p.id === currentId.value) || null)

  // 是否有多条线路可选（只有一条时不展示选择入口）
  const hasChoice = computed(() => providers.value.length > 1)

  const load = async () => {
    if (loading.value) return
    loading.value = true
    try {
      const res = await getProviderList()
      if (res?.code === 0 && Array.isArray(res.data?.providers)) {
        providers.value = res.data.providers
        // 选中的线路已下线时回落到默认线路
        if (selectedId.value && !providers.value.some(p => p.id === selectedId.value)) {
          selectedId.value = ''
          try {
            localStorage.removeItem(WS_PROVIDER_STORAGE_KEY)
          } catch {
            // ignore
          }
        }
      }
      loaded.value = true
    } catch (e) {
      console.warn('加载上游线路列表失败，将使用默认线路:', e)
    } finally {
      loading.value = false
    }
  }

  const setProvider = (id: string) => {
    selectedId.value = String(id || '')
    try {
      if (selectedId.value) localStorage.setItem(WS_PROVIDER_STORAGE_KEY, selectedId.value)
      else localStorage.removeItem(WS_PROVIDER_STORAGE_KEY)
    } catch {
      // 隐私模式下 localStorage 不可用，忽略即可
    }
  }

  return {
    providers,
    selectedId,
    currentId,
    currentProvider,
    hasChoice,
    loading,
    loaded,
    load,
    setProvider
  }
})
