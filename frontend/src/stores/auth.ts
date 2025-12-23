import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as authApi from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('authToken') || '')
  const isAuthenticated = ref(false)
  const loginLoading = ref(false)

  const login = async (accessCode: string) => {
    loginLoading.value = true
    try {
      const res = await authApi.login(accessCode)
      if (res.code === 0 && res.token) {
        token.value = res.token
        localStorage.setItem('authToken', res.token)
        isAuthenticated.value = true
        return true
      }
      return false
    } catch (error) {
      console.error('登录失败:', error)
      return false
    } finally {
      loginLoading.value = false
    }
  }

  const checkToken = async () => {
    if (!token.value) {
      isAuthenticated.value = false
      return false
    }

    try {
      const res = await authApi.verifyToken()
      if (res.code === 0) {
        isAuthenticated.value = true
        return true
      }
      logout()
      return false
    } catch (error) {
      logout()
      return false
    }
  }

  const logout = () => {
    token.value = ''
    localStorage.removeItem('authToken')
    isAuthenticated.value = false
  }

  return {
    token,
    isAuthenticated,
    loginLoading,
    login,
    checkToken,
    logout
  }
})
