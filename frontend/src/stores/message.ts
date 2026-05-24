// 消息状态与历史管理：负责聊天消息缓存、去重排序、历史拉取，以及乐观发送状态流转。
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage, SendStatus } from '@/types'
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

  // 乐观发送：回显确认超时窗口（超时后显示“重试”）
  const OPTIMISTIC_SEND_TIMEOUT_MS = 15_000
  const OPTIMISTIC_MATCH_WINDOW_MS = 30_000
  const pendingSendTimers = new Map<string, ReturnType<typeof setTimeout>>()

  const conversationKey = (ownerUserId: string | undefined, targetUserId?: string): string => {
    const owner = String(ownerUserId || '').trim()
    const target = String(targetUserId || '').trim()
    if (!target) return ''
    return owner ? `${owner}:${target}` : target
  }

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

  const normalizeTextForMatch = (input: string): string => {
    return String(input || '')
      .replace(/<br\s*\/?>/gi, ' ')
      .replace(/<[^>]*>/g, '')
      .replace(/\s+/g, ' ')
      .trim()
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

  const getMessageDedupRank = (msg: ChatMessage): number => {
    // 数值越大越“可信”，在去重 key 冲突时优先保留
    const hasTid = !!String(msg.tid || '').trim()
    if (hasTid) return 3
    if (msg.sendStatus === 'sent') return 2
    if (msg.sendStatus === 'sending' || msg.sendStatus === 'failed') return 1
    return 0
  }

  // 消息去重：优先基于 tid；tid 缺失时使用内容+时间的兜底 key。
  // 当 key 冲突时，优先保留“更可信”的版本（例如带 tid 的回显消息优先于本地 sending）。
  const deduplicateMessages = (messages: ChatMessage[]): ChatMessage[] => {
    const bestByKey = new Map<string, ChatMessage>()

    for (const msg of messages) {
      const key = getMessageDedupKey(msg)
      const existing = bestByKey.get(key)
      if (!existing) {
        bestByKey.set(key, msg)
        continue
      }

      const rankA = getMessageDedupRank(existing)
      const rankB = getMessageDedupRank(msg)
      if (rankB > rankA) {
        bestByKey.set(key, msg)
      }
    }

    return Array.from(bestByKey.values())
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

  const resolveMessageKey = (userIdOrOwnerId: string, targetUserId?: string): string => {
    return targetUserId == null ? String(userIdOrOwnerId || '') : conversationKey(userIdOrOwnerId, targetUserId)
  }

  const splitConversationKey = (key: string): { isConversationKey: boolean; targetUserId: string } => {
    const normalized = String(key || '')
    const idx = normalized.indexOf(':')
    if (idx < 0) return { isConversationKey: false, targetUserId: normalized }
    return {
      isConversationKey: true,
      targetUserId: normalized.slice(idx + 1)
    }
  }

  const ownerKeysForTarget = (targetUserId: string): string[] => {
    const suffix = `:${targetUserId}`
    return Array.from(chatHistory.value.keys()).filter(storedKey => storedKey.endsWith(suffix))
  }

  const syncLegacyFirstTidAlias = (targetUserId: string) => {
    const ownerKeys = ownerKeysForTarget(targetUserId)
    const tids = ownerKeys
      .map(key => firstTidMap.value[key])
      .filter((tid): tid is string => !!tid)

    const onlyTid = tids[0]
    if (tids.length === 1 && onlyTid) {
      firstTidMap.value[targetUserId] = onlyTid
      return
    }

    delete firstTidMap.value[targetUserId]
  }

  const updateFirstTid = (key: string, messages: ChatMessage[]) => {
    const minTid = getMinTid(messages)
    const { isConversationKey, targetUserId } = splitConversationKey(key)

    if (minTid) {
      firstTidMap.value[key] = minTid
      if (isConversationKey) {
        firstTidMap.value[targetUserId] = minTid
      }
      return
    }

    delete firstTidMap.value[key]
    if (isConversationKey) {
      syncLegacyFirstTidAlias(targetUserId)
    }
  }

  const commitMessages = (key: string, messages: ChatMessage[]) => {
    const normalized = normalizeMessages(messages)
    chatHistory.value.set(key, normalized)

    const { isConversationKey, targetUserId } = splitConversationKey(key)
    if (isConversationKey) {
      chatHistory.value.delete(targetUserId)
      delete firstTidMap.value[targetUserId]
    }

    updateFirstTid(key, normalized)
    return normalized
  }

  const resolveSingleArgumentKey = (key: string): string => {
    if (chatHistory.value.has(key)) return key

    const { isConversationKey, targetUserId } = splitConversationKey(key)
    if (isConversationKey) {
      if (chatHistory.value.has(targetUserId)) return targetUserId
      return key
    }

    const ownerKeys = ownerKeysForTarget(key)
    const onlyKey = ownerKeys[0]
    return ownerKeys.length === 1 && onlyKey ? onlyKey : key
  }

  const getMessages = (userIdOrOwnerId: string, targetUserId?: string): ChatMessage[] => {
    const key = resolveMessageKey(userIdOrOwnerId, targetUserId)
    const direct = chatHistory.value.get(key)
    if (direct) return direct

    const { isConversationKey, targetUserId: legacyTargetUserId } = splitConversationKey(key)
    if (isConversationKey) {
      return chatHistory.value.get(legacyTargetUserId) || []
    }

    if (targetUserId != null) return []

    const suffix = `:${key}`
    const matches: ChatMessage[][] = []
    for (const [storedKey, messages] of chatHistory.value.entries()) {
      if (storedKey.endsWith(suffix)) {
        matches.push(messages)
      }
    }
    if (matches.length === 1) return matches[0] || []
    return []
  }

  const setMessages = (userIdOrOwnerId: string, messagesOrTargetUserId: ChatMessage[] | string, maybeMessages?: ChatMessage[]) => {
    const rawKey = Array.isArray(messagesOrTargetUserId)
      ? resolveMessageKey(userIdOrOwnerId)
      : resolveMessageKey(userIdOrOwnerId, messagesOrTargetUserId)
    const key = Array.isArray(messagesOrTargetUserId) ? resolveSingleArgumentKey(rawKey) : rawKey
    const messages = Array.isArray(messagesOrTargetUserId) ? messagesOrTargetUserId : (maybeMessages || [])
    commitMessages(key, messages)
  }

  const addMessage = (userIdOrOwnerId: string, messageOrTargetUserId: ChatMessage | string, maybeMessage?: ChatMessage) => {
    const rawKey = typeof messageOrTargetUserId === 'string'
      ? resolveMessageKey(userIdOrOwnerId, messageOrTargetUserId)
      : resolveMessageKey(userIdOrOwnerId)
    const key = typeof messageOrTargetUserId === 'string' ? rawKey : resolveSingleArgumentKey(rawKey)
    const message = typeof messageOrTargetUserId === 'string' ? maybeMessage : messageOrTargetUserId
    if (!message) return
    const messages = getMessages(key)
    setMessages(key, [...messages, message])
  }

  const clearPendingTimer = (clientId: string) => {
    const t = pendingSendTimers.get(clientId)
    if (t) {
      clearTimeout(t)
      pendingSendTimers.delete(clientId)
    }
  }

  const updateMessageByClientId = (
    userIdOrOwnerId: string,
    clientIdOrTargetUserId: string,
    updaterOrClientId: ((msg: ChatMessage) => void) | string,
    maybeUpdater?: ((msg: ChatMessage) => void) | { normalize?: boolean },
    maybeOptions?: { normalize?: boolean },
  ): boolean => {
    const hasTarget = typeof updaterOrClientId === 'string' && typeof maybeUpdater === 'function'
    const key = hasTarget
      ? resolveMessageKey(userIdOrOwnerId, clientIdOrTargetUserId)
      : resolveSingleArgumentKey(resolveMessageKey(userIdOrOwnerId))
    const clientId = hasTarget ? updaterOrClientId : clientIdOrTargetUserId
    const updater = hasTarget ? maybeUpdater : updaterOrClientId
    const options = hasTarget ? maybeOptions : (maybeUpdater as { normalize?: boolean } | undefined)
    if (typeof updater !== 'function') return false
    const list = getMessages(key)
    if (!list.length) return false

    const idx = list.findIndex(m => String(m.clientId || '') === String(clientId))
    if (idx < 0) return false

    const msg = list[idx]
    if (!msg) return false

    updater(msg)

    if (options?.normalize !== false) {
      setMessages(key, list.slice())
    }

    return true
  }

  const startOptimisticTimeout = (userIdOrOwnerId: string, clientIdOrTargetUserId: string, timeoutMsOrClientId?: number | string, maybeTimeoutMs?: number) => {
    const hasTarget = typeof timeoutMsOrClientId === 'string'
    const key = hasTarget
      ? resolveMessageKey(userIdOrOwnerId, clientIdOrTargetUserId)
      : resolveMessageKey(userIdOrOwnerId)
    const clientId = hasTarget ? timeoutMsOrClientId : clientIdOrTargetUserId
    clearPendingTimer(clientId)
    const ms = typeof (hasTarget ? maybeTimeoutMs : timeoutMsOrClientId) === 'number'
      ? Number(hasTarget ? maybeTimeoutMs : timeoutMsOrClientId)
      : OPTIMISTIC_SEND_TIMEOUT_MS

    const t = setTimeout(() => {
      updateMessageByClientId(
        key,
        clientId,
        msg => {
          if (msg.sendStatus !== 'sending') return
          msg.sendStatus = 'failed'
          msg.sendError = msg.sendError || '发送超时'
        },
        { normalize: true }
      )
      pendingSendTimers.delete(clientId)
    }, ms)
    // 在 Node 测试环境中避免定时器阻塞进程退出
    if (typeof (t as any)?.unref === 'function') {
      ;(t as any).unref()
    }

    pendingSendTimers.set(clientId, t)
  }

  const getOptimisticMatchKey = (msg: ChatMessage): { kind: 'media' | 'text'; key: string } => {
    const mediaPath = getMessageRemoteMediaPath(msg)
    if (mediaPath) return { kind: 'media', key: mediaPath }
    return { kind: 'text', key: normalizeTextForMatch(msg.content) }
  }

  const confirmOutgoingEcho = (userIdOrOwnerId: string, echoedOrTargetUserId: ChatMessage | string, maybeEchoed?: ChatMessage): boolean => {
    const key = typeof echoedOrTargetUserId === 'string'
      ? resolveMessageKey(userIdOrOwnerId, echoedOrTargetUserId)
      : resolveMessageKey(userIdOrOwnerId)
    const echoed = typeof echoedOrTargetUserId === 'string' ? maybeEchoed : echoedOrTargetUserId
    if (!echoed?.isSelf) return false

    const list = getMessages(key)
    if (!list.length) return false

    const echoedKey = getOptimisticMatchKey(echoed)
    const echoedTime = parseMessageTime(echoed.time) ?? Date.now()

    let best: { idx: number; score: number; statusRank: number } | null = null

    for (let i = 0; i < list.length; i++) {
      const m = list[i]
      if (!m) continue
      if (!m.isSelf) continue
      if (!m.clientId) continue
      if (m.sendStatus !== 'sending' && m.sendStatus !== 'failed') continue

      const mk = getOptimisticMatchKey(m)
      if (mk.kind !== echoedKey.kind) continue
      if (mk.key !== echoedKey.key) continue

      const mt = parseMessageTime(m.time) ?? echoedTime
      const diff = Math.abs(echoedTime - mt)
      if (diff > OPTIMISTIC_MATCH_WINDOW_MS) continue

      // prefer sending over failed, then smaller time diff, then earlier message
      const statusRank = m.sendStatus === 'sending' ? 1 : 0
      const score = -diff
      if (!best || statusRank > best.statusRank || (statusRank === best.statusRank && score > best.score)) {
        best = { idx: i, score, statusRank }
      }
    }

    if (!best) return false

    const target = list[best.idx]
    if (!target?.clientId) return false

    clearPendingTimer(target.clientId)

    Object.assign(target, {
      code: echoed.code,
      fromuser: echoed.fromuser,
      touser: echoed.touser,
      type: echoed.type,
      content: echoed.content,
      time: echoed.time,
      tid: echoed.tid,
      isSelf: echoed.isSelf,
      isImage: echoed.isImage,
      isVideo: echoed.isVideo,
      isFile: echoed.isFile,
      imageUrl: echoed.imageUrl,
      videoUrl: echoed.videoUrl,
      fileUrl: echoed.fileUrl,
      segments: echoed.segments,
      sendStatus: 'sent' as SendStatus,
      sendError: undefined,
      optimistic: false
    })

    setMessages(key, list.slice())
    return true
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
      const key = conversationKey(myUserID, UserToID)
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
        if (!chatHistory.value.get(key) && !chatHistory.value.get(UserToID)) {
          commitMessages(key, [])
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

        const existing = chatHistory.value.get(key) || chatHistory.value.get(UserToID) || []

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
            const normalized = commitMessages(key, combined)

            // 返回新增消息数量（近似值，通过长度变化计算）
            return Math.max(0, normalized.length - existing.length)
          } else {
            // 首次加载 或 无缓存：直接设置
            commitMessages(key, mapped)
          }
        } else {
          // 向前翻页
          const combined = [...mapped, ...existing]
          commitMessages(key, combined)
        }

        return mapped.length
      }

      if (!chatHistory.value.get(key)) {
        commitMessages(key, [])
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

  const clearHistory = (userIdOrOwnerId: string, targetUserId?: string) => {
    const key = resolveMessageKey(userIdOrOwnerId, targetUserId)
    const keys = targetUserId == null
      ? [key, ...Array.from(chatHistory.value.keys()).filter(storedKey => storedKey.endsWith(`:${key}`))]
      : [key]
    const uniqueKeys = Array.from(new Set(keys))
    const existing = uniqueKeys.flatMap(storedKey => chatHistory.value.get(storedKey) || [])
    for (const msg of existing) {
      const cid = String(msg?.clientId || '')
      if (cid) clearPendingTimer(cid)
    }
    uniqueKeys.forEach(storedKey => {
      chatHistory.value.delete(storedKey)
      delete firstTidMap.value[storedKey]
    })
  }

  const resetAll = () => {
    for (const cid of pendingSendTimers.keys()) {
      clearPendingTimer(cid)
    }
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
    conversationKey,
    getMessages,
    addMessage,
    setMessages,
    updateMessageByClientId,
    startOptimisticTimeout,
    confirmOutgoingEcho,
    loadHistory,
    clearHistory,
    resetAll
  }
})
