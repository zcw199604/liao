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

    // 清零未读
    if (user.unreadCount && user.unreadCount > 0) {
      user.unreadCount = 0
    }

    // 加载聊天历史
    if (loadHistory && userStore.currentUser) {
      messageStore.loadHistory(userStore.currentUser.id, user.id, {
        isFirst: true,
        firstTid: '0',
        myUserName: userStore.currentUser.name
      })
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

      user.isFavorite = !user.isFavorite
      show(user.isFavorite ? '收藏成功' : '取消收藏成功')

      // 重新加载列表
      await loadUsers()
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
