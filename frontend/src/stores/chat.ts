import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'

// 辅助函数：标准化用户数据
const normalizeUser = (user: any, isFavorite: boolean = false): User => {
  return {
    id: user.id,
    name: user.nickname || user.name || '未知',
    nickname: user.nickname || user.name || '未知',
    sex: user.sex || user.userSex || '未知',
    age: user.age || user.userAge || '0',
    area: user.address || user.userAddress || '未知',
    address: user.address || user.userAddress || '未知',
    ip: user.ip || '',
    isFavorite: isFavorite,
    lastMsg: '暂无消息',
    lastTime: '刚刚',
    unreadCount: 0
  }
}

export const useChatStore = defineStore('chat', () => {
  // === 单一数据源架构 ===
  const userMap = ref<Map<string, User>>(new Map())  // userId -> User对象（唯一数据源）
  const historyUserIds = ref<string[]>([])           // 历史列表的用户ID顺序
  const favoriteUserIds = ref<string[]>([])          // 收藏列表的用户ID顺序

  // === 其他状态 ===
  const currentChatUser = ref<User | null>(null)
  const activeTab = ref<'history' | 'favorite'>('history')
  const isMatching = ref(false)
  const wsConnected = ref(false)
  const listScrollTop = ref(0)

  // === Computed：从单一数据源派生列表 ===
  const historyUsers = computed(() =>
    historyUserIds.value
      .map(id => userMap.value.get(id))
      .filter((user): user is User => user !== undefined)
  )

  const favoriteUsers = computed(() =>
    favoriteUserIds.value
      .map(id => userMap.value.get(id))
      .filter((user): user is User => user !== undefined)
  )

  const displayList = computed(() => {
    return activeTab.value === 'history' ? historyUsers.value : favoriteUsers.value
  })

  // === 工具方法：更新或插入用户 ===
  const upsertUser = (user: User) => {
    const existingUser = userMap.value.get(user.id)
    if (existingUser) {
      // 合并更新
      Object.assign(existingUser, user)
    } else {
      // 新增用户
      userMap.value.set(user.id, user)
    }
  }

  // === 工具方法：获取用户 ===
  const getUser = (userId: string): User | undefined => {
    return userMap.value.get(userId)
  }

  // === 工具方法：更新用户的部分字段 ===
  const updateUser = (userId: string, updates: Partial<User>) => {
    const user = userMap.value.get(userId)
    if (user) {
      Object.assign(user, updates)
    }
  }

  // === 工具方法：清空所有用户数据 ===
  const clearAllUsers = () => {
    userMap.value.clear()
    historyUserIds.value = []
    favoriteUserIds.value = []
    currentChatUser.value = null
  }

  // === 加载历史用户列表 ===
  const loadHistoryUsers = async (userId: string, userName: string) => {
    try {
      const cookieData = generateCookie(userId, userName)
      const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
      const userAgent = navigator.userAgent

      const data = await chatApi.getHistoryUserList(userId, cookieData, referer, userAgent)

      if (data && Array.isArray(data)) {
        const users: User[] = data
          .filter(user => user.id !== userId)
          .map(user => normalizeUser(user, false))

        // 更新 userMap 和 historyUserIds
        const newHistoryIds: string[] = []
        users.forEach(user => {
          const existing = userMap.value.get(user.id)
          if (existing) {
            // 保留现有的收藏状态和未读数
            upsertUser({
              ...user,
              isFavorite: existing.isFavorite,
              unreadCount: existing.unreadCount || 0
            })
          } else {
            upsertUser(user)
          }
          newHistoryIds.push(user.id)
        })

        historyUserIds.value = newHistoryIds
      }
    } catch (error) {
      console.error('加载历史用户失败:', error)
      historyUserIds.value = []
    }
  }

  // === 加载收藏用户列表 ===
  const loadFavoriteUsers = async (userId: string, userName: string) => {
    try {
      const cookieData = generateCookie(userId, userName)
      const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
      const userAgent = navigator.userAgent

      const data = await chatApi.getFavoriteUserList(userId, cookieData, referer, userAgent)

      if (data && Array.isArray(data)) {
        const users: User[] = data.map(user => normalizeUser(user, true))

        // 更新 userMap 和 favoriteUserIds
        const newFavoriteIds: string[] = []
        users.forEach(user => {
          const existing = userMap.value.get(user.id)
          if (existing) {
            // 合并更新，强制设置isFavorite=true
            upsertUser({
              ...existing,
              ...user,
              isFavorite: true
            })
          } else {
            upsertUser(user)
          }
          newFavoriteIds.push(user.id)
        })

        favoriteUserIds.value = newFavoriteIds
      }
    } catch (error) {
      console.error('加载收藏用户失败:', error)
      favoriteUserIds.value = []
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
    // 状态
    currentChatUser,
    activeTab,
    isMatching,
    wsConnected,
    listScrollTop,

    // Computed（向后兼容）
    historyUsers,
    favoriteUsers,
    displayList,

    // 新增：直接访问底层数据（供高级操作）
    historyUserIds,
    favoriteUserIds,

    // 方法
    loadHistoryUsers,
    loadFavoriteUsers,
    enterChat,
    exitChat,
    startMatch,
    cancelMatch,

    // 新增工具方法
    upsertUser,
    getUser,
    updateUser,
    clearAllUsers
  }
})
