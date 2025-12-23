import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useMessageStore } from '@/stores/message'
import { useWebSocket } from './useWebSocket'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'
import { generateRandomHexId } from '@/utils/id'
import { useToast } from '@/composables/useToast'

export const useChat = () => {
  const chatStore = useChatStore()
  const userStore = useUserStore()
  const messageStore = useMessageStore()
  const { send } = useWebSocket()
  const { show } = useToast()

  const loadUsers = async () => {
    const currentUser = userStore.currentUser
    if (!currentUser) return

    await Promise.all([
      chatStore.loadHistoryUsers(currentUser.id, currentUser.name),
      chatStore.loadFavoriteUsers(currentUser.id, currentUser.name)
    ])
  }

  const startMatch = () => {
    if (!userStore.currentUser) return false
    if (!chatStore.wsConnected) {
      show('WebSocket 未连接，无法匹配')
      return false
    }

    chatStore.startMatch()

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

    chatStore.cancelMatch()

    if (!chatStore.wsConnected) return

    const message = {
      act: 'randomOut',
      id: userStore.currentUser.id,
      msg: generateRandomHexId()
    }

    send(message)
    console.log('取消匹配')
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
      } else {
        // 取消收藏 - 从收藏列表移除
        const index = chatStore.favoriteUserIds.indexOf(user.id)
        if (index > -1) {
          chatStore.favoriteUserIds.splice(index, 1)
        }
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
    enterChat,
    exitChat,
    toggleFavorite
  }
}
