import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage } from '@/types'
import * as chatApi from '@/api/chat'
import { generateCookie } from '@/utils/cookie'
import { useMediaStore } from '@/stores/media'
import { useSystemConfigStore } from '@/stores/systemConfig'
import { emojiMap } from '@/constants/emoji'
import { getSegmentsMeta, parseMessageSegments } from '@/utils/messageSegments'

export const useMessageStore = defineStore('message', () => {
  const chatHistory = ref<Map<string, ChatMessage[]>>(new Map())
  const isTyping = ref(false)
  const firstTidMap = ref<Record<string, string>>({})
  const loadingMore = ref(false)
  const isLoadingHistory = ref(false)

  // 语义去重窗口：用于合并 WebSocket 推送与历史拉取时，避免同一媒体消息短时间内重复渲染
  const MEDIA_DEDUP_WINDOW_MS = 5_000

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

  const stripQueryAndHash = (input: string): string => {
    const s = String(input || '')
    return (s.split('?')[0] || '').split('#')[0] || ''
  }

  // 提取聊天媒体消息的 remotePath：
  // - "[path/to/file.ext]" -> "path/to/file.ext"
  // - "http://x:9006/img/Upload/path/to/file.ext" -> "path/to/file.ext"
  const extractRemotePathFromMediaString = (input: string): string => {
    const raw = String(input || '').trim()
    if (!raw) return ''

    if (raw.startsWith('[') && raw.endsWith(']')) {
      // 表情包（如 "[doge]"）不应作为媒体去重Key的一部分
      if (emojiMap[raw]) return ''
      return raw.substring(1, raw.length - 1)
    }

    const clean = stripQueryAndHash(raw)
    const marker = '/img/Upload/'
    const idx = clean.indexOf(marker)
    if (idx < 0) return ''
    return clean.substring(idx + marker.length)
  }

  const getMessageRemoteMediaPath = (msg: ChatMessage): string => {
    if (Array.isArray(msg.segments)) {
      for (const seg of msg.segments) {
        if (seg.kind !== 'text' && seg.path) return String(seg.path)
      }
    }

    const candidates = [
      String(msg.imageUrl || ''),
      String(msg.videoUrl || ''),
      String(msg.fileUrl || ''),
      String(msg.content || '')
    ]

    for (const c of candidates) {
      const path = extractRemotePathFromMediaString(c)
      if (path) return path
    }

    return ''
  }

  const getTimeBucketKey = (timeStr: string): string => {
    const raw = String(timeStr || '').trim()
    const ms = parseMessageTime(raw)
    if (ms == null) return `raw:${raw}`
    return `b:${Math.floor(ms / MEDIA_DEDUP_WINDOW_MS)}`
  }

  const getMessageDedupKey = (msg: ChatMessage): string => {
    // 媒体消息：使用 remotePath + isSelf + 时间桶作为语义去重 key，
    // 避免上游 tid 缺失/不一致时出现“同一张图显示两条”的问题。
    const mediaPath = getMessageRemoteMediaPath(msg)
    if (mediaPath) {
      const direction = msg.isSelf ? 'out' : 'in'
      return `media:${direction}|${mediaPath}|${getTimeBucketKey(msg.time)}`
    }

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
        const systemConfigStore = useSystemConfigStore()
        if (!systemConfigStore.loaded) {
          await systemConfigStore.loadSystemConfig()
        }

        const resolveMediaUrl = async (path: string): Promise<string> => {
          if (!mediaStore.imgServer) return ''
          const port = await systemConfigStore.resolveImagePort(path, mediaStore.imgServer)
          return `http://${mediaStore.imgServer}:${port}/img/Upload/${path}`
        }

        const list = data.contents_list.slice().reverse()
        const mapped: ChatMessage[] = await Promise.all(list.map(async (msg: any) => {
          const rawContent = String(msg?.content || '')
          const msgTid = String(msg?.Tid || msg?.tid || '')
          const msgTime = String(msg?.time || '')

          const isSelf = String(msg?.id || '') !== String(UserToID)
          const segments = await parseMessageSegments(rawContent, { emojiMap, resolveMediaUrl })
          const meta = getSegmentsMeta(segments)
          const type = meta.hasImage ? 'image' : meta.hasVideo ? 'video' : meta.hasFile ? 'file' : 'text'

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
            // Keep raw content; media URLs live in segments/imageUrl/videoUrl/fileUrl.
            content: rawContent,
            time: msgTime,
            tid: msgTid,
            isSelf,
            isImage: meta.hasImage,
            isVideo: meta.hasVideo,
            isFile: meta.hasFile,
            imageUrl: meta.imageUrl,
            videoUrl: meta.videoUrl,
            fileUrl: meta.fileUrl,
            segments
          } as ChatMessage
        }))

        const existing = chatHistory.value.get(UserToID) || []

        if (isFirst) {
          if (incremental && existing.length > 0) {
            // 增量模式 + 有缓存：合并去重
            // 特殊处理：清理本地缓存中可能是 WebSocket 推送的临时消息
            // (这些消息内容相同、时间相近，但 TID 不同，会导致常规去重失效)
            const cleanExisting = existing.filter(oldMsg => {
              const oldTime = parseMessageTime(oldMsg.time) || 0
              // 检查 mapped 中是否有对应的“正式版”消息
              const hasDuplicateInMapped = mapped.some(newMsg => {
                const newTime = parseMessageTime(newMsg.time) || 0
                const oldKey = getMessageRemoteMediaPath(oldMsg) || String(oldMsg.content || '')
                const newKey = getMessageRemoteMediaPath(newMsg) || String(newMsg.content || '')
                return newKey === oldKey &&
                       Math.abs(newTime - oldTime) < MEDIA_DEDUP_WINDOW_MS // 时间窗口误差容忍
              })
              // 如果有对应正式版，则移除本地临时版
              return !hasDuplicateInMapped
            })

            // 将新数据 mapped 放在前面，deduplicateMessages 会保留首次出现的版本
            const combined = [...mapped, ...cleanExisting]
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
