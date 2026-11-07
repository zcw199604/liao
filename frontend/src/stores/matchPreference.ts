import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  getMatchPreference,
  updateMatchPreference,
  type MatchPreference
} from '@/api/matchPreference'

// 匹配偏好状态：按当前身份保存，切换身份后重新加载。
// 只有晚风线路支持，`supported` 为 false 时 UI 不展示入口。
export const useMatchPreferenceStore = defineStore('matchPreference', () => {
  const preference = ref<MatchPreference>({
    identityId: '',
    preferGender: 'any',
    preferAgeMin: 0,
    preferAgeMax: 0,
    preferSameProvince: false
  })
  const supported = ref(false)
  const loading = ref(false)
  const saving = ref(false)

  // 是否有实质偏好（用于按钮上显示小圆点提示）
  const hasPreference = computed(() => {
    const p = preference.value
    return p.preferGender !== 'any' || p.preferAgeMin > 0 || p.preferAgeMax > 0 || p.preferSameProvince
  })

  // 供 UI 展示的一句话摘要
  const summary = computed(() => {
    const p = preference.value
    const parts: string[] = []
    if (p.preferGender === 'male') parts.push('男生')
    else if (p.preferGender === 'female') parts.push('女生')
    if (p.preferAgeMin > 0 && p.preferAgeMax > 0) parts.push(`${p.preferAgeMin}-${p.preferAgeMax}岁`)
    else if (p.preferAgeMin > 0) parts.push(`${p.preferAgeMin}岁以上`)
    else if (p.preferAgeMax > 0) parts.push(`${p.preferAgeMax}岁以下`)
    if (p.preferSameProvince) parts.push('同省')
    return parts.length > 0 ? parts.join(' · ') : '不限'
  })

  const load = async (identityId: string) => {
    if (!identityId) return
    loading.value = true
    try {
      const res = await getMatchPreference(identityId)
      supported.value = Boolean(res?.supported)
      if (res?.data) {
        preference.value = { ...res.data, identityId }
      }
    } catch {
      // 读取失败保持默认值，不阻塞聊天
      supported.value = false
    } finally {
      loading.value = false
    }
  }

  const save = async (next: Partial<MatchPreference>) => {
    const merged: MatchPreference = { ...preference.value, ...next }
    saving.value = true
    try {
      const res = await updateMatchPreference(merged)
      if (res?.data) {
        preference.value = res.data
      } else {
        preference.value = merged
      }
      return { success: res?.code === 0, message: res?.msg || '' }
    } catch (err) {
      return { success: false, message: String(err) }
    } finally {
      saving.value = false
    }
  }

  return {
    preference,
    supported,
    loading,
    saving,
    hasPreference,
    summary,
    load,
    save
  }
})
