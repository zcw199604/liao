import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'

export const useChatStore = defineStore('chat', () => {
  const currentChatUser = ref<User | null>(null)
  const historyUsers = ref<User[]>([])
  const favoriteUsers = ref<User[]>([])
  const activeTab = ref<'history' | 'favorite'>('history')
  const isMatching = ref(false)
  const wsConnected = ref(false)
  const listScrollTop = ref(0)

  const displayList = computed(() => {
    return activeTab.value === 'history' ? historyUsers.value : favoriteUsers.value
  })

  const loadHistoryUsers = async (userId: string, userName: string) => {
    try {
      const cookieData = generateCookie(userId, userName)
      const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
      const userAgent = navigator.userAgent

      const data = await chatApi.getHistoryUserList(userId, cookieData, referer, userAgent)
      console.log('历史用户数据:', data)

      if (data && Array.isArray(data)) {
        historyUsers.value = data
          .filter(user => user.id !== userId)
          .map(user => ({
            id: user.id,
            name: user.nickname || user.name || '未知',
            nickname: user.nickname || user.name || '未知',
            sex: user.sex || user.userSex || '未知',
            age: user.age || user.userAge || '0',
            area: user.address || user.userAddress || '未知',
            address: user.address || user.userAddress || '未知',
            ip: user.ip || '',
            isFavorite: false,
            lastMsg: '暂无消息',
            lastTime: '刚刚',
            unreadCount: 0
          }))
        console.log('历史用户列表加载完成:', historyUsers.value.length)
      }
    } catch (error) {
      console.error('加载历史用户失败:', error)
      historyUsers.value = []
    }
  }

  const loadFavoriteUsers = async (userId: string, userName: string) => {
    try {
      const cookieData = generateCookie(userId, userName)
      const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
      const userAgent = navigator.userAgent

      const data = await chatApi.getFavoriteUserList(userId, cookieData, referer, userAgent)
      console.log('收藏用户数据:', data)

      if (data && Array.isArray(data)) {
        favoriteUsers.value = data.map(user => ({
          id: user.id,
          name: user.nickname || user.name || '未知',
          nickname: user.nickname || user.name || '未知',
          sex: user.sex || user.userSex || '未知',
          age: user.age || user.userAge || '0',
          area: user.address || user.userAddress || '未知',
          address: user.address || user.userAddress || '未知',
          ip: user.ip || '',
          isFavorite: true,
          lastMsg: '暂无消息',
          lastTime: '刚刚',
          unreadCount: 0
        }))
        console.log('收藏用户列表加载完成:', favoriteUsers.value.length)
      }
    } catch (error) {
      console.error('加载收藏用户失败:', error)
      favoriteUsers.value = []
    }
  }

  const enterChat = (user: User) => {
    currentChatUser.value = user
  }

  const exitChat = () => {
    currentChatUser.value = null
  }

  const startMatch = () => {
    isMatching.value = true
  }

  const cancelMatch = () => {
    isMatching.value = false
  }

  return {
    currentChatUser,
    historyUsers,
    favoriteUsers,
    activeTab,
    isMatching,
    wsConnected,
    listScrollTop,
    displayList,
    loadHistoryUsers,
    loadFavoriteUsers,
    enterChat,
    exitChat,
    startMatch,
    cancelMatch
  }
})
