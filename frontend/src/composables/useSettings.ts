import { ref } from 'vue'
import * as systemApi from '@/api/system'
import type { ConnectionStats } from '@/types'

export const useSettings = () => {
  const connectionStats = ref<ConnectionStats>({ active: 0, upstream: 0, downstream: 0 })
  const forceoutUserCount = ref(0)
  const disconnectAllLoading = ref(false)

  const loadConnectionStats = async () => {
    try {
      const res = await systemApi.getConnectionStats()
      if (res.code === 0 && res.data) {
        connectionStats.value = res.data
        console.log('连接统计:', res.data)
      }
    } catch (error) {
      console.error('加载连接统计失败:', error)
    }
  }

  const loadForceoutUserCount = async () => {
    try {
      console.log('🔍 开始加载被禁止用户统计...')
      const res = await systemApi.getForceoutUserCount()
      console.log('🔍 API响应数据:', res)

      if (res.code === 0 && typeof res.data === 'number') {
        forceoutUserCount.value = res.data
        console.log('✅ 被禁止用户数量:', res.data)
      }
    } catch (error) {
      console.error('❌ 加载被禁止用户数量失败:', error)
    }
  }

  const disconnectAll = async () => {
    disconnectAllLoading.value = true
    try {
      const res = await systemApi.disconnectAllConnections()
      if (res.code === 0) {
        await loadConnectionStats()
        return true
      }
      return false
    } finally {
      disconnectAllLoading.value = false
    }
  }

  const clearForceout = async () => {
    try {
      const res = await systemApi.clearForceoutUsers()
      if (res.code === 0) {
        await loadForceoutUserCount()
        return { success: true, message: res.msg || '清除成功' }
      }
      return { success: false, message: res.msg || '清除失败' }
    } catch (error) {
      console.error('清除禁止用户失败:', error)
      return { success: false, message: '清除失败，请稍后重试' }
    }
  }

  return {
    connectionStats,
    forceoutUserCount,
    disconnectAllLoading,
    loadConnectionStats,
    loadForceoutUserCount,
    disconnectAll,
    clearForceout
  }
}
