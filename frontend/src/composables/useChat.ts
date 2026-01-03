import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useMessageStore } from '@/stores/message'
import { useFavoriteStore } from '@/stores/favorite'
import { useWebSocket } from './useWebSocket'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'
import { generateRandomHexId } from '@/utils/id'
import { useToast } from '@/composables/useToast'

export const useChat = () => {
  const chatStore = useChatStore()
  const userStore = useUserStore()
  const messageStore = useMessageStore()
  const favoriteStore = useFavoriteStore()
  const { send } = useWebSocket()
  const { show } = useToast()

  // 自动匹配定时器
  let autoMatchTimer: ReturnType<typeof setTimeout> | null = null

  const loadUsers = async () => {
    const currentUser = userStore.currentUser
    if (!currentUser) return

    await Promise.all([
      chatStore.loadHistoryUsers(currentUser.id, currentUser.name),
      chatStore.loadFavoriteUsers(currentUser.id, currentUser.name)
    ])
  }

  const startMatch = (isContinuous: boolean = false) => {
    if (!userStore.currentUser) return false
    if (!chatStore.wsConnected) {
      show('WebSocket 未连接，无法匹配')
      return false
    }

    // 只在非连续模式下更新状态
    if (!isContinuous) {
      chatStore.startMatch()
    }

    const message = {
      act: 'random',
      id: userStore.currentUser.id,
      userAge: '0'
    }

    send(message)
    console.log('开始匹配，已发送random消息')
    return true
  }

  const cancelMatch = () => {
    if (!userStore.currentUser) return

    // 清除自动匹配定时器
    if (autoMatchTimer) {
      clearTimeout(autoMatchTimer)
      autoMatchTimer = null
    }

    // 取消连续匹配状态
    chatStore.cancelContinuousMatch()

    if (!chatStore.wsConnected) return

    const message = {
      act: 'randomOut',
      id: userStore.currentUser.id,
      msg: generateRandomHexId()
    }

    send(message)
    console.log('取消匹配')
  }

  // 开始连续匹配
  const startContinuousMatch = (count: number) => {
    if (!userStore.currentUser) return false
    if (!chatStore.wsConnected) {
      show('WebSocket 未连接，无法匹配')
      return false
    }

    chatStore.startContinuousMatch(count)
    return startMatch(true)
  }

  // 处理自动匹配（匹配成功后触发）
  const handleAutoMatch = () => {
    const config = chatStore.continuousMatchConfig

    if (!config.enabled) return

    if (config.current >= config.total) {
      // 完成所有匹配
      chatStore.cancelContinuousMatch()
      show(`连续匹配完成！共匹配 ${config.total} 次`)
      return
    }

    // 等待2秒后开始下一次匹配
    autoMatchTimer = setTimeout(() => {
      chatStore.incrementMatchCount()
      chatStore.setCurrentMatchedUser(null) // 清空上一个用户信息
      startMatch(true)
    }, 2000)
  }

  // 进入聊天并中断连续匹配
  const enterChatAndStopMatch = (user: any) => {
    // 清除定时器
    if (autoMatchTimer) {
      clearTimeout(autoMatchTimer)
      autoMatchTimer = null
    }

    // 取消连续匹配
    chatStore.cancelContinuousMatch()

    // 进入聊天
    enterChat(user, true)
  }

  const enterChat = (user: any, loadHistory: boolean = true) => {
    chatStore.enterChat(user)

    // 清零未读 - 使用单一数据源更新
    if (user.unreadCount && user.unreadCount > 0) {
      chatStore.updateUser(user.id, { unreadCount: 0 })
    }

    // 加载聊天历史 - 增量渲染策略
    if (loadHistory && userStore.currentUser) {
      const cachedMessages = messageStore.getMessages(user.id)

      if (cachedMessages.length > 0) {
        // 有缓存：立即显示，增量获取新消息
        console.log(`使用缓存消息 ${cachedMessages.length} 条，检查新消息中...`)

        // 获取最新消息并增量追加
        messageStore.loadHistory(userStore.currentUser.id, user.id, {
          isFirst: true,
          firstTid: '0',
          myUserName: userStore.currentUser.name,
          incremental: true  // 增量模式
        }).then(newCount => {
          if (newCount > 0) {
            console.log(`增量追加 ${newCount} 条新消息`)
          } else {
            console.log('没有新消息')
          }
        })
      } else {
        // 无缓存：正常加载（显示loading）
        messageStore.loadHistory(userStore.currentUser.id, user.id, {
          isFirst: true,
          firstTid: '0',
          myUserName: userStore.currentUser.name
        })
      }
    }
  }

  const exitChat = () => {
    chatStore.exitChat()
  }

  const toggleFavorite = async (user: any) => {
    if (!userStore.currentUser) return

    try {
      const myUserID = userStore.currentUser.id
      const cookieData = generateCookie(myUserID, userStore.currentUser.name)
      const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
      const userAgent = navigator.userAgent

      let res: any
      if (user.isFavorite) {
        res = await chatApi.cancelFavorite(myUserID, user.id, cookieData, referer, userAgent)
      } else {
        res = await chatApi.toggleFavorite(myUserID, user.id, cookieData, referer, userAgent)
      }

      const ok = res?.code === '0' || res?.status === 'true'
      if (!ok) {
        show(`操作失败: ${res?.msg || '未知错误'}`)
        return
      }

      const newFavoriteState = !user.isFavorite

      // 更新 userMap 中的收藏状态
      chatStore.updateUser(user.id, { isFavorite: newFavoriteState })

      // 更新 favoriteUserIds 数组
      if (newFavoriteState) {
        // 添加收藏 - 插入到收藏列表前面
        if (!chatStore.favoriteUserIds.includes(user.id)) {
          chatStore.favoriteUserIds.unshift(user.id)
        }
        // 同步到全局收藏
        void favoriteStore.addFavorite(myUserID, user.id, user.nickname || user.name)
      } else {
        // 取消收藏 - 从收藏列表移除
        const index = chatStore.favoriteUserIds.indexOf(user.id)
        if (index > -1) {
          chatStore.favoriteUserIds.splice(index, 1)
        }
        // 从全局收藏移除
        void favoriteStore.removeFavorite(myUserID, user.id)
      }

      show(newFavoriteState ? '收藏成功' : '取消收藏成功')

      // 本地立即更新，无需重新加载列表
    } catch (error) {
      console.error('收藏操作失败:', error)
      show('操作失败')
    }
  }

  return {
    loadUsers,
    startMatch,
    cancelMatch,
    startContinuousMatch,
    handleAutoMatch,
    enterChatAndStopMatch,
    enterChat,
    exitChat,
    toggleFavorite
  }
}
