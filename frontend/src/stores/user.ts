import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { CurrentUser } from '@/types'
import { useIdentityStore } from './identity'

export const useUserStore = defineStore('user', () => {
  const currentUser = ref<CurrentUser | null>(null)
  const editMode = ref(false)
  const editUserInfo = ref<Partial<CurrentUser>>({})
  const identityStore = useIdentityStore()

  const setCurrentUser = (user: CurrentUser) => {
    currentUser.value = user
    console.log('当前用户已设置:', currentUser.value)
    
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
