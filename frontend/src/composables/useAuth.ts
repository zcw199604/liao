import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

export const useAuth = () => {
  const authStore = useAuthStore()
  const router = useRouter()

  const login = async (accessCode: string) => {
    const success = await authStore.login(accessCode)
    if (success) {
      router.push('/identity')
    }
    return success
  }

  const logout = () => {
    authStore.logout()
    router.push('/login')
  }

  const checkAuth = async () => {
    const isValid = await authStore.checkToken()
    if (!isValid) {
      router.push('/login')
    }
    return isValid
  }

  return {
    login,
    logout,
    checkAuth
  }
}
