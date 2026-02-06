<template>
  <div class="fixed inset-0 z-[200] bg-canvas">
    <div class="page-container items-center justify-center overflow-y-auto p-6">
      <div class="w-full max-w-md">
        <!-- Logo/标题 -->
        <div class="text-center mb-8">
          <h1 class="text-3xl font-bold text-fg mb-2">匿名匹配</h1>
          <p class="text-fg-muted text-sm">请输入访问码以继续</p>
        </div>

        <!-- 登录表单 -->
        <div class="ui-card p-8">
          <div class="mb-6">
            <label class="block text-fg-muted text-sm font-medium mb-2">访问码</label>
            <input
              v-model="accessCode"
              type="password"
              placeholder="请输入访问码"
              @keyup.enter="handleLogin"
              class="ui-input"
              :disabled="loading"
              autofocus
            />
          </div>

          <button
            @click="handleLogin"
            :disabled="loading || !accessCode"
            class="ui-btn-primary w-full py-3"
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
