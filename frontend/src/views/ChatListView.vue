<template>
  <div class="h-full w-full">
    <ChatSidebar
      @select="handleSelectUser"
      @match-success="handleMatchSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useChat } from '@/composables/useChat'
import ChatSidebar from '@/components/chat/ChatSidebar.vue'
import type { User } from '@/types'

const router = useRouter()
const userStore = useUserStore()
const { enterChat } = useChat()

const handleSelectUser = (user: User) => {
  enterChat(user, true)
  router.push(`/chat/${user.id}`)
}

const handleMatchSuccess = (user: User) => {
  enterChat(user, false) // 匹配进入不强制加载历史（新会话）
  router.push(`/chat/${user.id}`)
}

onMounted(() => {
  if (!userStore.currentUser) {
    router.push('/identity')
  }
})
</script>