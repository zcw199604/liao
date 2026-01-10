import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ImagePortMode, SystemConfig } from '@/types'
import * as systemApi from '@/api/system'
import { IMG_SERVER_IMAGE_PORT } from '@/constants/config'

const DEFAULT_FIXED_PORT = String(IMG_SERVER_IMAGE_PORT)
const DEFAULT_REAL_MIN_BYTES = 2048
const RESOLVE_CACHE_TTL_MS = 5 * 60 * 1000

export const useSystemConfigStore = defineStore('systemConfig', () => {
  const loaded = ref(false)
  const loading = ref(false)
  const saving = ref(false)

  const imagePortMode = ref<ImagePortMode>('fixed')
  const imagePortFixed = ref<string>(DEFAULT_FIXED_PORT)
  const imagePortRealMinBytes = ref<number>(DEFAULT_REAL_MIN_BYTES)

  // 端口解析缓存（避免 WS/历史消息频繁调用）
  const resolvedImagePort = ref<string>('')
  const resolvedForServer = ref<string>('')
  const resolvedAt = ref<number>(0)

  const applyConfig = (cfg: SystemConfig) => {
    imagePortMode.value = (cfg.imagePortMode || 'fixed') as ImagePortMode
    imagePortFixed.value = String(cfg.imagePortFixed || DEFAULT_FIXED_PORT)
    imagePortRealMinBytes.value = Number(cfg.imagePortRealMinBytes || DEFAULT_REAL_MIN_BYTES)
  }

  const clearResolvedCache = (imgServer?: string) => {
    if (!imgServer || resolvedForServer.value === imgServer) {
      resolvedImagePort.value = ''
      resolvedForServer.value = ''
      resolvedAt.value = 0
    }
  }

  const loadSystemConfig = async () => {
    if (loading.value) return
    loading.value = true
    try {
      const res = await systemApi.getSystemConfig()
      if (res.code === 0 && res.data) {
        applyConfig(res.data)
        loaded.value = true
        clearResolvedCache()
      }
    } catch (e) {
      console.error('加载系统配置失败:', e)
    } finally {
      loading.value = false
    }
  }

  const saveSystemConfig = async (next: Partial<SystemConfig>) => {
    if (saving.value) return false
    saving.value = true
    try {
      const res = await systemApi.updateSystemConfig(next)
      if (res.code === 0 && res.data) {
        applyConfig(res.data)
        loaded.value = true
        clearResolvedCache()
        return true
      }
      return false
    } catch (e) {
      console.error('保存系统配置失败:', e)
      return false
    } finally {
      saving.value = false
    }
  }

  const resolveImagePort = async (path: string, imgServer?: string): Promise<string> => {
    if (!loaded.value) {
      await loadSystemConfig()
    }

    const fixed = imagePortFixed.value || DEFAULT_FIXED_PORT
    if (imagePortMode.value === 'fixed') return fixed

    if (imgServer && resolvedForServer.value && imgServer !== resolvedForServer.value) {
      clearResolvedCache()
    }

    const now = Date.now()
    if (resolvedImagePort.value && now - resolvedAt.value < RESOLVE_CACHE_TTL_MS) {
      return resolvedImagePort.value
    }

    try {
      const res = await systemApi.resolveImagePort(path)
      if (res.code === 0 && res.data?.port) {
        const port = String(res.data.port)
        resolvedImagePort.value = port
        resolvedAt.value = now
        if (imgServer) resolvedForServer.value = imgServer
        return port
      }
    } catch (e) {
      console.warn('解析图片端口失败:', e)
    }

    return fixed
  }

  return {
    loaded,
    loading,
    saving,
    imagePortMode,
    imagePortFixed,
    imagePortRealMinBytes,
    loadSystemConfig,
    saveSystemConfig,
    resolveImagePort,
    clearResolvedCache
  }
})

