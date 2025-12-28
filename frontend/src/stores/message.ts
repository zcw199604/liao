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

        if (isFirst) {
          if (incremental && existing.length > 0) {
            // 增量模式 + 有缓存：合并去重（以服务端数据为准）
            // 将新数据 mapped 放在前面，deduplicateMessages 会保留首次出现的版本
            const combined = [...mapped, ...existing]
            const normalized = normalizeMessages(combined)
            chatHistory.value.set(UserToID, normalized)

            const minTid = getMinTid(normalized)
            if (minTid) {
              firstTidMap.value[UserToID] = minTid
            }

            // 返回新增消息数量（近似值，通过长度变化计算）
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
          // 向前翻页：prepend到前面并去重排序
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
