import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import type { CurrentUser } from '@/types'
import { useIdentityStore } from './identity'

// 当前身份按标签页保存在 sessionStorage，刷新页面后可恢复，不同标签页可使用不同身份
export const CURRENT_USER_STORAGE_KEY = 'liao-current-user'

const loadPersistedUser = (): CurrentUser | null => {
  try {
    const raw = sessionStorage.getItem(CURRENT_USER_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' && parsed.id ? (parsed as CurrentUser) : null
  } catch {
    return null
  }
}

const persistUser = (user: CurrentUser | null) => {
  try {
    if (user) sessionStorage.setItem(CURRENT_USER_STORAGE_KEY, JSON.stringify(user))
    else sessionStorage.removeItem(CURRENT_USER_STORAGE_KEY)
  } catch {
    // 隐私模式等场景下 sessionStorage 不可用，忽略即可
  }
}

export const useUserStore = defineStore('user', () => {
  const currentUser = ref<CurrentUser | null>(loadPersistedUser())
  const editMode = ref(false)
  const editUserInfo = ref<Partial<CurrentUser>>({})
  const identityStore = useIdentityStore()

  watch(currentUser, persistUser, { deep: true, flush: 'sync' })

  const setCurrentUser = (user: CurrentUser) => {
    currentUser.value = user

    // 保存Cookie用于后续预览
    if (user.id && user.cookie) {
      identityStore.saveIdentityCookie(user.id, user.cookie)
    }
  }

  const updateUserInfo = (info: Partial<CurrentUser>) => {
    if (currentUser.value) {
      Object.assign(currentUser.value, info)
    }
  }

  const startEdit = () => {
    if (currentUser.value) {
      editUserInfo.value = { ...currentUser.value }
      editMode.value = true
    }
  }

  const saveEdit = () => {
    if (currentUser.value && editUserInfo.value) {
      Object.assign(currentUser.value, editUserInfo.value)
      editMode.value = false
    }
  }

  const cancelEdit = () => {
    editMode.value = false
    editUserInfo.value = {}
  }

  const clearCurrentUser = () => {
    currentUser.value = null
    editMode.value = false
    editUserInfo.value = {}
  }

  return {
    currentUser,
    editMode,
    editUserInfo,
    setCurrentUser,
    updateUserInfo,
    startEdit,
    saveEdit,
    cancelEdit,
    clearCurrentUser
  }
})
