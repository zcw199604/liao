import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage } from '@/types'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'
import { useMediaStore } from '@/stores/media'
import { emojiMap } from '@/constants/emoji'

export const useMessageStore = defineStore('message', () => {
  const chatHistory = ref<Map<string, ChatMessage[]>>(new Map())
  const isTyping = ref(false)
  const firstTidMap = ref<Record<string, string>>({})
  const loadingMore = ref(false)
  const isLoadingHistory = ref(false)

  const parseMessageTime = (timeStr: string): number | null => {
    if (!timeStr) return null

    // 常见格式：2025-12-18 03:02:11.721 / 2025/12/18 03:02:11
    const match = timeStr.match(
      /(\d{4})[-\/](\d{2})[-\/](\d{2})\s+(\d{2}):(\d{2})(?::(\d{2}))?(?:\.(\d{1,3}))?/
    )
    if (match) {
      const year = Number(match[1])
      const month = Number(match[2]) - 1
      const day = Number(match[3])
      const hour = Number(match[4])
      const minute = Number(match[5])
      const second = Number(match[6] || '0')
      const millisecond = Number(String(match[7] || '0').padEnd(3, '0'))
      return new Date(year, month, day, hour, minute, second, millisecond).getTime()
    }

    const parsed = Date.parse(timeStr)
    if (!Number.isNaN(parsed)) return parsed
    return null
  }

  const getMessageDedupKey = (msg: ChatMessage): string => {
    const tid = String(msg.tid || '').trim()
    if (tid) return `tid:${tid}`

    const fromUserId = String(msg.fromuser?.id || '')
    const toUserId = String(msg.touser?.id || '')
    const type = String(msg.type || '')
    const time = String(msg.time || '')
    const content = String(msg.content || '')
    return `fallback:${fromUserId}|${toUserId}|${type}|${time}|${content}`
  }

  // 消息去重：优先基于 tid；tid 缺失时使用内容+时间的兜底 key
  const deduplicateMessages = (messages: ChatMessage[]): ChatMessage[] => {
    const seen = new Set<string>()
    const result: ChatMessage[] = []
    for (const msg of messages) {
      const key = getMessageDedupKey(msg)
      if (seen.has(key)) continue
      seen.add(key)
      result.push(msg)
    }
    return result
  }

  // 消息排序：按时间（主）+ tid（辅）升序，保证渲染稳定
  const sortMessages = (messages: ChatMessage[]): ChatMessage[] => {
    return messages.sort((a, b) => {
      const timeA = parseMessageTime(a.time)
      const timeB = parseMessageTime(b.time)
      if (timeA != null && timeB != null && timeA !== timeB) return timeA - timeB

      const tidA = Number.parseInt(a.tid, 10)
      const tidB = Number.parseInt(b.tid, 10)
      const tidAValid = Number.isFinite(tidA)
      const tidBValid = Number.isFinite(tidB)
      if (tidAValid && tidBValid && tidA !== tidB) return tidA - tidB
      if (tidAValid && !tidBValid) return 1
      if (!tidAValid && tidBValid) return -1

      const timeStrA = String(a.time || '')
      const timeStrB = String(b.time || '')
      if (timeStrA !== timeStrB) return timeStrA.localeCompare(timeStrB)

      const fromIdA = String(a.fromuser?.id || '')
      const fromIdB = String(b.fromuser?.id || '')
      if (fromIdA !== fromIdB) return fromIdA.localeCompare(fromIdB)

      return String(a.content || '').localeCompare(String(b.content || ''))
    })
  }

  const normalizeMessages = (messages: ChatMessage[]): ChatMessage[] => {
    return sortMessages(deduplicateMessages(messages))
  }

  const getMinTid = (messages: ChatMessage[]): string | null => {
    let minTidValue: number | null = null
    let minTidRaw: string | null = null

    for (const msg of messages) {
      const rawTid = String(msg.tid || '').trim()
      if (!rawTid) continue
      const numericTid = Number.parseInt(rawTid, 10)
      if (!Number.isFinite(numericTid)) continue

      if (minTidValue == null || numericTid < minTidValue) {
        minTidValue = numericTid
        minTidRaw = rawTid
      }
    }

    return minTidRaw
  }

  const getMessages = (userId: string): ChatMessage[] => {
    return chatHistory.value.get(userId) || []
  }

  const addMessage = (userId: string, message: ChatMessage) => {
    const messages = chatHistory.value.get(userId) || []
    const updated = normalizeMessages([...messages, message])
    chatHistory.value.set(userId, updated)

    const minTid = getMinTid(updated)
    if (minTid) {
      firstTidMap.value[userId] = minTid
    }
  }

  const loadHistory = async (
    myUserID: string,
    UserToID: string,
    options?: { isFirst?: boolean; firstTid?: string; myUserName?: string; incremental?: boolean }
  ): Promise<number> => {
    loadingMore.value = true
    isLoadingHistory.value = true
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
            // 如果是表情包，不处理为文件
            if (emojiMap[rawContent]) {
              type = 'text'
              content = rawContent
            } else {
              const path = rawContent.substring(1, rawContent.length - 1)
              const isVideo = path.toLowerCase().includes('.mp4')
              const isImage = !isVideo && /\.(jpg|jpeg|png|gif|webp)$/i.test(path)

              if (mediaStore.imgServer) {
                const port = isVideo ? '8006' : '9006'
                content = `http://${mediaStore.imgServer}:${port}/img/Upload/${path}`
                if (isVideo) type = 'video'
                else if (isImage) type = 'image'
                else type = 'file'
              }
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
            isFile: type === 'file',
            imageUrl: type === 'image' ? content : '',
            videoUrl: type === 'video' ? content : '',
            fileUrl: type === 'file' ? content : ''
          } as ChatMessage
        })

        const existing = chatHistory.value.get(UserToID) || []

        if (isFirst) {
          if (incremental && existing.length > 0) {
            // 增量模式 + 有缓存：合并去重（以服务端数据为准）
            const combined = [...mapped, ...existing]
            const normalized = normalizeMessages(combined)
            chatHistory.value.set(UserToID, normalized)

            const minTid = getMinTid(normalized)
            if (minTid) {
              firstTidMap.value[UserToID] = minTid
            }

            return Math.max(0, normalized.length - existing.length)
          } else {
            // 首次加载 或 无缓存：直接设置
            const normalized = normalizeMessages(mapped)
            chatHistory.value.set(UserToID, normalized)

            const minTid = getMinTid(normalized)
            if (minTid) {
              firstTidMap.value[UserToID] = minTid
            }
          }
        } else {
          // 向前翻页
          const combined = [...mapped, ...existing]
          const normalized = normalizeMessages(combined)
          chatHistory.value.set(UserToID, normalized)

          const minTid = getMinTid(normalized)
          if (minTid) {
            firstTidMap.value[UserToID] = minTid
          }
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
      isLoadingHistory.value = false
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
    isLoadingHistory,
    getMessages,
    addMessage,
    loadHistory,
    clearHistory,
    resetAll
  }
})