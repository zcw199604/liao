<template>
  <div class="fixed inset-0 z-[200] bg-[#0f0f13] flex items-center justify-center">
    <div class="w-full max-w-md px-6">
      <!-- Logo/标题 -->
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-white mb-2">匿名匹配</h1>
        <p class="text-gray-400 text-sm">请输入访问码以继续</p>
      </div>

      <!-- 登录表单 -->
      <div class="bg-[#1a1a1f] rounded-2xl p-8 shadow-xl">
        <div class="mb-6">
          <label class="block text-gray-300 text-sm font-medium mb-2">访问码</label>
          <input
            v-model="accessCode"
            type="password"
            placeholder="请输入访问码"
            @keyup.enter="handleLogin"
            class="w-full px-4 py-3 bg-[#27272a] text-white rounded-lg border border-gray-700 focus:border-indigo-500 focus:outline-none transition"
            :disabled="loading"
            autofocus
          />
        </div>

        <button
          @click="handleLogin"
          :disabled="loading || !accessCode"
          class="w-full py-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg font-medium hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition"
        >
          <span v-if="!loading">登录</span>
          <span v-else class="flex items-center justify-center">
            <span class="radar-spinner mr-2"></span>
            登录中...
          </span>
        </button>
      </div>
    </div>

    <Toast />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import Toast from '@/components/common/Toast.vue'

const router = useRouter()
const authStore = useAuthStore()
const { show } = useToast()

const accessCode = ref('')
const loading = ref(false)

const handleLogin = async () => {
  if (!accessCode.value.trim()) {
    show('请输入访问码')
    return
  }

  loading.value = true
  try {
    const success = await authStore.login(accessCode.value)
    if (success) {
      show('登录成功')
      router.push('/identity')
    } else {
      show('访问码错误，请重试')
    }
  } catch (error) {
    console.error('登录失败:', error)
    show('登录失败，请重试')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  const isValid = await authStore.checkToken()
  if (isValid) {
    router.push('/identity')
  }
})
</script>
