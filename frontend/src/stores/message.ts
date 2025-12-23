import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage } from '@/types'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'
import { useMediaStore } from '@/stores/media'

export const useMessageStore = defineStore('message', () => {
  const chatHistory = ref<Map<string, ChatMessage[]>>(new Map())
  const isTyping = ref(false)
  const firstTidMap = ref<Record<string, string>>({})
  const loadingMore = ref(false)

  // 消息去重 - 基于tid
  const deduplicateMessages = (messages: ChatMessage[]): ChatMessage[] => {
    const seen = new Set<string>()
    return messages.filter(msg => {
      if (!msg.tid) return true // 没有tid的消息保留
      if (seen.has(msg.tid)) return false
      seen.add(msg.tid)
      return true
    })
  }

  // 消息排序 - 按tid数字大小排序
  const sortMessages = (messages: ChatMessage[]): ChatMessage[] => {
    return messages.sort((a, b) => {
      const tidA = parseInt(a.tid) || 0
      const tidB = parseInt(b.tid) || 0
      return tidA - tidB
    })
  }

  // 获取增量新消息 - 只返回比缓存最后一条更新的消息
  const getIncrementalMessages = (
    cached: ChatMessage[],
    fresh: ChatMessage[]
  ): ChatMessage[] => {
    if (cached.length === 0) return fresh // 无缓存，返回所有新消息

    // 获取缓存中最后一条消息的tid（最大的tid）
    const maxCachedTid = Math.max(...cached.map(m => parseInt(m.tid) || 0))

    // 只返回tid比缓存更大的消息
    const newMessages = fresh.filter(msg => {
      const msgTid = parseInt(msg.tid) || 0
      return msgTid > maxCachedTid
    })

    return newMessages
  }

  const getMessages = (userId: string): ChatMessage[] => {
    return chatHistory.value.get(userId) || []
  }

  const addMessage = (userId: string, message: ChatMessage) => {
    const messages = chatHistory.value.get(userId) || []
    messages.push(message)
    chatHistory.value.set(userId, messages)
  }

  const loadHistory = async (
    myUserID: string,
    UserToID: string,
    options?: { isFirst?: boolean; firstTid?: string; myUserName?: string; incremental?: boolean }
  ): Promise<number> => {
    loadingMore.value = true
    try {
      const mediaStore = useMediaStore()
      const isFirst = options?.isFirst ?? true
      const firstTid = options?.firstTid ?? '0'
      const myUserName = options?.myUserName || 'User'
      const incremental = options?.incremental ?? false

      if (!mediaStore.imgServer) {
        try {
          await mediaStore.loadImgServer()
        } catch {
          // ignore
        }
      }

      const cookieData = generateCookie(myUserID, myUserName)
      const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
      const userAgent = navigator.userAgent

      const data = await chatApi.getMessageHistory(
        myUserID,
        UserToID,
        isFirst ? '1' : '0',
        firstTid,
        cookieData,
        referer,
        userAgent
      )
      console.log(isFirst ? '聊天历史数据:' : '更多历史消息数据:', data)

      if (data?.error) {
        console.warn('聊天历史加载失败:', data.error)
        if (!chatHistory.value.get(UserToID)) {
          chatHistory.value.set(UserToID, [])
        }
        return 0
      }

      if (data && data.code === 0 && Array.isArray(data.contents_list)) {
        const mapped: ChatMessage[] = data.contents_list.reverse().map((msg: any) => {
          const rawContent = String(msg?.content || '')
          const msgTid = String(msg?.Tid || msg?.tid || '')
          const msgTime = String(msg?.time || '')

          const isSelf = String(msg?.id || '') !== String(UserToID)
          let content = rawContent
          let type = 'text'

          if (rawContent.startsWith('[') && rawContent.endsWith(']')) {
            const path = rawContent.substring(1, rawContent.length - 1)
            const isVideo = path.toLowerCase().includes('.mp4')
            const isImage = !isVideo && /\.(jpg|jpeg|png|gif|webp)$/i.test(path)

            if (mediaStore.imgServer && (isVideo || isImage)) {
              const port = isVideo ? '8006' : '9006'
              content = `http://${mediaStore.imgServer}:${port}/img/Upload/${path}`
              type = isVideo ? 'video' : 'image'
            }
          }

          const nickname = String(msg?.nickname || (isSelf ? myUserName : ''))
          const fromuser = {
            id: String(msg?.id || ''),
            name: nickname,
            nickname,
            sex: '未知',
            ip: ''
          }

          return {
            code: 7,
            fromuser,
            touser: undefined,
            type,
            content,
            time: msgTime,
            tid: msgTid,
            isSelf,
            isImage: type === 'image',
            isVideo: type === 'video',
            imageUrl: type === 'image' ? content : '',
            videoUrl: type === 'video' ? content : ''
          } as ChatMessage
        })

        const existing = chatHistory.value.get(UserToID) || []

        if (isFirst) {
          if (incremental && existing.length > 0) {
            // 增量模式 + 有缓存：只追加新消息
            const newMessages = getIncrementalMessages(existing, mapped)

            if (newMessages.length > 0) {
              console.log(`增量追加 ${newMessages.length} 条新消息`)
              const updated = [...existing, ...newMessages]
              chatHistory.value.set(UserToID, updated)
            } else {
              console.log('没有新消息，保持缓存不变')
            }

            return newMessages.length  // 返回新增消息数量
          } else {
            // 首次加载 或 无缓存：直接设置
            chatHistory.value.set(UserToID, mapped)
          }
        } else {
          // 向前翻页：prepend到前面并去重排序
          const combined = [...mapped, ...existing]
          const deduplicated = deduplicateMessages(combined)
          const sorted = sortMessages(deduplicated)
          chatHistory.value.set(UserToID, sorted)
        }

        if (mapped.length > 0) {
          firstTidMap.value[UserToID] = mapped[0]!.tid
        }

        return mapped.length
      }

      if (!chatHistory.value.get(UserToID)) {
        chatHistory.value.set(UserToID, [])
      }
      return 0
    } catch (error) {
      console.error('加载聊天历史失败:', error)
      return -1
    } finally {
      loadingMore.value = false
    }
  }

  const clearHistory = (userId: string) => {
    chatHistory.value.delete(userId)
    delete firstTidMap.value[userId]
  }

  const resetAll = () => {
    chatHistory.value = new Map()
    isTyping.value = false
    firstTidMap.value = {}
    loadingMore.value = false
  }

  return {
    chatHistory,
    isTyping,
    firstTidMap,
    loadingMore,
    getMessages,
    addMessage,
    loadHistory,
    clearHistory,
    resetAll
  }
})
