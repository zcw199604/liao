<template>
  <div class="fixed inset-0 z-[200] bg-canvas flex items-center justify-center">
    <div class="w-full max-w-md px-6">
      <!-- Logo/标题 -->
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-fg mb-2">匿名匹配</h1>
        <p class="text-fg-muted text-sm">请输入访问码以继续</p>
      </div>

      <!-- 登录表单 -->
      <div class="bg-surface/80 backdrop-blur-xl ring-1 ring-line rounded-2xl p-8 shadow-2xl shadow-indigo-500/10">
        <div class="mb-6">
          <label class="block text-fg-muted text-sm font-medium mb-2">访问码</label>
          <input
            v-model="accessCode"
            type="password"
            placeholder="请输入访问码"
            @keyup.enter="handleLogin"
            class="w-full px-4 py-3 bg-surface-2 text-fg rounded-lg border border-line-strong focus:border-line-strong focus:ring-1 focus:ring-indigo-500/50 focus:bg-surface-hover focus:outline-none transition placeholder-fg-subtle"
            :disabled="loading"
            autofocus
          />
        </div>

        <button
          @click="handleLogin"
          :disabled="loading || !accessCode"
          class="w-full py-3 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-lg shadow-indigo-500/20 shadow-[inset_0_1px_0_rgba(255,255,255,0.2)]"
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
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import Toast from '@/components/common/Toast.vue'

const router = useRouter()
const route = useRoute()
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
  // 检查是否有错误参数（如被踢下线）
  const errorMsg = route.query.error
  if (errorMsg) {
    show(decodeURIComponent(String(errorMsg)))
    // 清除URL中的参数
    router.replace('/')
    return // 如果有错误，不自动检查Token，等待用户手动重新登录
  }

  const isValid = await authStore.checkToken()
  if (isValid) {
    router.push('/identity')
  }
})
</script>
