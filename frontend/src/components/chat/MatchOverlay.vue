<template>
  <teleport to="body">
    <div
      v-if="chatStore.isMatching"
      class="fixed inset-0 z-[60] bg-black/90 flex flex-col items-center justify-center text-center px-4"
    >
      <!-- 进度显示 -->
      <div v-if="chatStore.continuousMatchConfig.enabled && chatStore.continuousMatchConfig.total > 1" class="mb-4">
        <p class="text-blue-400 text-sm font-medium">
          连续匹配进行中：第 {{ chatStore.continuousMatchConfig.current }}/{{ chatStore.continuousMatchConfig.total }} 次
        </p>
      </div>

      <!-- 匹配动画 -->
      <div v-if="!chatStore.currentMatchedUser" class="relative w-32 h-32 mb-8">
        <div class="absolute inset-0 border-4 border-blue-500/30 rounded-full animate-ping"></div>
        <div class="absolute inset-0 border-4 border-blue-500 rounded-full flex items-center justify-center">
          <i class="fas fa-satellite-dish text-4xl text-blue-400"></i>
        </div>
      </div>

      <!-- 匹配成功：显示用户信息 -->
      <div v-else class="bg-[#18181b] rounded-2xl p-6 max-w-sm w-full mb-8">
        <div class="flex items-center gap-4 mb-4">
          <!-- 用户头像 -->
          <div :class="getColorClass(chatStore.currentMatchedUser.id)"
               class="w-16 h-16 rounded-xl flex items-center justify-center text-white font-bold text-2xl shadow-lg shrink-0">
            {{ chatStore.currentMatchedUser.nickname.charAt(0).toUpperCase() }}
          </div>

          <!-- 用户信息 -->
          <div class="flex-1 text-left">
            <h3 class="text-xl font-bold text-white mb-1">{{ chatStore.currentMatchedUser.nickname }}</h3>
            <div class="flex items-center gap-2 text-sm text-gray-400">
              <span>{{ chatStore.currentMatchedUser.sex }}</span>
              <span>·</span>
              <span>{{ chatStore.currentMatchedUser.age }}岁</span>
              <span>·</span>
              <span>{{ chatStore.currentMatchedUser.area }}</span>
            </div>
          </div>
        </div>

        <!-- 聊天按钮 -->
        <button
          @click="handleEnterChat"
          class="w-full py-3 bg-blue-600 hover:bg-blue-500 rounded-xl text-white font-bold transition active:scale-95 mb-3"
        >
          <i class="fas fa-comment-dots mr-2"></i>
          进入聊天
        </button>

        <!-- 继续匹配按钮 (仅在单次匹配模式下显示) -->
        <button
          v-if="chatStore.continuousMatchConfig.total === 1"
          @click="handleContinueMatch"
          class="w-full py-3 bg-gray-700 hover:bg-gray-600 rounded-xl text-white font-bold transition active:scale-95"
        >
          <i class="fas fa-random mr-2"></i>
          继续匹配
        </button>
      </div>

      <!-- 提示文本 -->
      <h2 v-if="!chatStore.currentMatchedUser" class="text-xl font-bold mb-2 text-white">
        正在寻找有缘人...
      </h2>
      <p v-if="!chatStore.currentMatchedUser" class="text-gray-400 text-sm mb-4">
        匹配完全匿名
      </p>

      <!-- 倒计时提示（连续匹配且已匹配到用户，且不是单次匹配模式） -->
      <p v-if="chatStore.currentMatchedUser && chatStore.continuousMatchConfig.enabled && chatStore.continuousMatchConfig.total > 1"
         class="text-gray-400 text-sm mb-4">
        {{ isLastMatch ? '2秒后返回列表...' : '2秒后自动开始下一次匹配...' }}
      </p>

      <!-- 取消按钮 -->
      <button
        type="button"
        @click="handleCancelMatch"
        class="mt-4 px-6 py-2 border border-gray-600 rounded-full text-gray-200 text-sm hover:bg-white/5 transition"
      >
        取消{{ chatStore.continuousMatchConfig.enabled && chatStore.continuousMatchConfig.total > 1 ? '连续匹配' : '' }}
      </button>
    </div>
  </teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useChat } from '@/composables/useChat'
import { useToast } from '@/composables/useToast'
import { getColorClass } from '@/constants/colors'

const router = useRouter()
const chatStore = useChatStore()
const { cancelMatch, enterChatAndStopMatch, startContinuousMatch } = useChat()
const { show } = useToast()

// 判断是否是最后一个匹配用户
const isLastMatch = computed(() => {
  const config = chatStore.continuousMatchConfig
  return config.current >= config.total
})

// 进入聊天并中断连续匹配
const handleEnterChat = () => {
  if (!chatStore.currentMatchedUser) return

  const user = chatStore.currentMatchedUser

  // 进入聊天并中断连续匹配
  enterChatAndStopMatch(user)

  // 跳转到聊天页面
  router.push(`/chat/${user.id}`)

  show('已进入聊天')
}

// 继续匹配
const handleContinueMatch = () => {
  chatStore.setCurrentMatchedUser(null)
  startContinuousMatch(1)
}

// 取消匹配
const handleCancelMatch = () => {
  cancelMatch()
  show('已取消匹配')
}
</script>
