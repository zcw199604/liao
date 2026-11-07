import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  getLocationSetting,
  updateLocationSetting,
  type LocationSetting
} from '@/api/locationSetting'

// 位置公开状态：按当前身份保存，切换身份后重新加载。
// 默认不公开（匹配成功时对方看不到我所在的城市），用户可在身份设置里改成公开。
export const useLocationSettingStore = defineStore('locationSetting', () => {
  const setting = ref<LocationSetting>({ identityId: '', showCity: false })
  // 当前线路能否真正生效（目前只有晚风线路；chat2019 等同固定不公开）
  const supported = ref(false)
  const loading = ref(false)
  const saving = ref(false)

  const summary = computed(() => (setting.value.showCity ? '公开' : '不公开'))

  const load = async (identityId: string) => {
    if (!identityId) return
    loading.value = true
    try {
      const res = await getLocationSetting(identityId)
      supported.value = Boolean(res?.supported)
      if (res?.data) {
        setting.value = { ...res.data, identityId }
      } else {
        setting.value = { identityId, showCity: false }
      }
    } catch {
      // 读取失败保持默认（不公开），不阻塞聊天
      supported.value = false
      setting.value = { identityId, showCity: false }
    } finally {
      loading.value = false
    }
  }

  const save = async (showCity: boolean) => {
    const identityId = setting.value.identityId
    if (!identityId) return { success: false, message: '缺少身份ID' }
    saving.value = true
    try {
      const res = await updateLocationSetting({ identityId, showCity })
      setting.value = { identityId, showCity: res?.data?.showCity ?? showCity }
      return { success: res?.code === 0, message: res?.msg || '' }
    } catch (err) {
      return { success: false, message: String(err) }
    } finally {
      saving.value = false
    }
  }

  return {
    setting,
    supported,
    loading,
    saving,
    summary,
    load,
    save
  }
})
