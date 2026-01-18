// 消息发送能力：封装 WebSocket 发送，并提供乐观 UI（sending/failed）与重试入口。
import { useUserStore } from '@/stores/user'
import { useMessageStore } from '@/stores/message'
import { useWebSocket } from './useWebSocket'
import * as mediaApi from '@/api/media'
import { extractRemoteFilePathFromImgUploadUrl } from '@/utils/media'
import type { ChatMessage, MessageSegment } from '@/types'

export const useMessage = () => {
  const userStore = useUserStore()
  const messageStore = useMessageStore()
  const { send } = useWebSocket()

  const formatNow = () => {
    const now = new Date()
    const pad = (value: number, length: number = 2) => String(value).padStart(length, '0')
    return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}.${pad(now.getMilliseconds(), 3)}`
  }

  const generateClientId = (): string => {
    try {
      if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
        return (crypto as any).randomUUID()
      }
    } catch {
      // ignore
    }
    return `c_${Date.now()}_${Math.random().toString(16).slice(2)}`
  }

  const buildAct = (targetUser: any) => {
    return `touser_${targetUser.id}_${targetUser.nickname || targetUser.name || ''}`
  }

  const upsertOptimisticMessage = (params: {
    targetUserId: string
    clientId: string
    nextMessage: ChatMessage
  }) => {
    const updated = messageStore.updateMessageByClientId(
      params.targetUserId,
      params.clientId,
      msg => {
        Object.assign(msg, params.nextMessage)
      }
    )
    if (!updated) {
      messageStore.addMessage(params.targetUserId, params.nextMessage)
    }
  }

  // 发送文本消息
  const sendText = (content: string, targetUser: any, options?: { clientId?: string }) => {
    if (!userStore.currentUser || !targetUser) return

    const clientId = options?.clientId || generateClientId()
    const now = formatNow()

    const optimisticMessage: ChatMessage = {
      code: 7,
      fromuser: userStore.currentUser as any,
      touser: targetUser,
      type: 'text',
      content,
      time: now,
      tid: '',
      isSelf: true,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: '',
      segments: [],
      clientId,
      sendStatus: 'sending',
      sendError: undefined,
      optimistic: true
    }

    upsertOptimisticMessage({
      targetUserId: String(targetUser.id),
      clientId,
      nextMessage: optimisticMessage
    })
    messageStore.startOptimisticTimeout(String(targetUser.id), clientId)

    const message = {
      act: buildAct(targetUser),
      id: userStore.currentUser.id,
      msg: content
    }

    const ok = send(message) !== false
    if (!ok) {
      messageStore.updateMessageByClientId(String(targetUser.id), clientId, msg => {
        if (msg.sendStatus === 'sending') {
          msg.sendStatus = 'failed'
          msg.sendError = msg.sendError || '发送失败'
        }
      })
    }
  }

  // 发送图片消息
  const sendImage = async (mediaUrl: string, targetUser: any, localFilename?: string, options?: { clientId?: string }) => {
    if (!userStore.currentUser || !targetUser) return

    const filePath = extractRemoteFilePathFromImgUploadUrl(mediaUrl)
    if (!filePath) return

    const clientId = options?.clientId || generateClientId()
    const now = formatNow()
    const segments: MessageSegment[] = [{ kind: 'image', path: filePath, url: mediaUrl }]

    const optimisticMessage: ChatMessage = {
      code: 7,
      fromuser: userStore.currentUser as any,
      touser: targetUser,
      type: 'image',
      content: `[${filePath}]`,
      time: now,
      tid: '',
      isSelf: true,
      isImage: true,
      isVideo: false,
      isFile: false,
      imageUrl: mediaUrl,
      videoUrl: '',
      fileUrl: '',
      segments,
      clientId,
      sendStatus: 'sending',
      sendError: undefined,
      optimistic: true
    }

    upsertOptimisticMessage({
      targetUserId: String(targetUser.id),
      clientId,
      nextMessage: optimisticMessage
    })
    messageStore.startOptimisticTimeout(String(targetUser.id), clientId)

    const message = {
      act: buildAct(targetUser),
      id: userStore.currentUser.id,
      msg: `[${filePath}]`
    }

    const ok = send(message) !== false
    if (!ok) {
      messageStore.updateMessageByClientId(String(targetUser.id), clientId, msg => {
        if (msg.sendStatus === 'sending') {
          msg.sendStatus = 'failed'
          msg.sendError = msg.sendError || '发送失败'
        }
      })
    }

    // 记录发送关系到数据库（用于历史图片查询）
    try {
      await mediaApi.recordImageSend({
        remoteUrl: mediaUrl,
        fromUserId: userStore.currentUser.id,
        toUserId: targetUser.id,
        localFilename
      })
    } catch (e) {
      console.warn('记录发送关系失败:', e)
    }
  }

  // 发送视频消息
  const sendVideo = async (mediaUrl: string, targetUser: any, localFilename?: string, options?: { clientId?: string }) => {
    if (!userStore.currentUser || !targetUser) return

    const filePath = extractRemoteFilePathFromImgUploadUrl(mediaUrl)
    if (!filePath) return

    const clientId = options?.clientId || generateClientId()
    const now = formatNow()
    const segments: MessageSegment[] = [{ kind: 'video', path: filePath, url: mediaUrl }]

    const optimisticMessage: ChatMessage = {
      code: 7,
      fromuser: userStore.currentUser as any,
      touser: targetUser,
      type: 'video',
      content: `[${filePath}]`,
      time: now,
      tid: '',
      isSelf: true,
      isImage: false,
      isVideo: true,
      isFile: false,
      imageUrl: '',
      videoUrl: mediaUrl,
      fileUrl: '',
      segments,
      clientId,
      sendStatus: 'sending',
      sendError: undefined,
      optimistic: true
    }

    upsertOptimisticMessage({
      targetUserId: String(targetUser.id),
      clientId,
      nextMessage: optimisticMessage
    })
    messageStore.startOptimisticTimeout(String(targetUser.id), clientId)

    const message = {
      act: buildAct(targetUser),
      id: userStore.currentUser.id,
      msg: `[${filePath}]`
    }

    const ok = send(message) !== false
    if (!ok) {
      messageStore.updateMessageByClientId(String(targetUser.id), clientId, msg => {
        if (msg.sendStatus === 'sending') {
          msg.sendStatus = 'failed'
          msg.sendError = msg.sendError || '发送失败'
        }
      })
    }

    // 记录发送关系到数据库
    try {
      await mediaApi.recordImageSend({
        remoteUrl: mediaUrl,
        fromUserId: userStore.currentUser.id,
        toUserId: targetUser.id,
        localFilename
      })
    } catch (e) {
      console.warn('记录发送关系失败:', e)
    }
  }

  const retryMessage = (message: ChatMessage) => {
    if (!userStore.currentUser) return
    const targetUser = message.touser
    if (!targetUser) return

    const targetUserId = String(targetUser.id || '')
    const clientId = String(message.clientId || '')
    if (!targetUserId || !clientId) return

    const now = formatNow()
    messageStore.updateMessageByClientId(targetUserId, clientId, msg => {
      msg.sendStatus = 'sending'
      msg.sendError = undefined
      msg.time = now
      msg.tid = ''
    })
    messageStore.startOptimisticTimeout(targetUserId, clientId)

    const payload = {
      act: buildAct(targetUser),
      id: userStore.currentUser.id,
      msg: String(message.content || '')
    }

    const ok = send(payload) !== false
    if (!ok) {
      messageStore.updateMessageByClientId(targetUserId, clientId, msg => {
        if (msg.sendStatus === 'sending') {
          msg.sendStatus = 'failed'
          msg.sendError = msg.sendError || '发送失败'
        }
      })
    }
  }

  // 发送正在输入状态
  const sendTypingStatus = (isTyping: boolean, targetUser: any) => {
    if (!userStore.currentUser || !targetUser) return

    const act = isTyping
      ? `inputStatusOn_${userStore.currentUser.id}_${userStore.currentUser.nickname}`
      : `inputStatusOff_${userStore.currentUser.id}_${userStore.currentUser.nickname}`

    const message = {
      act,
      destuserid: targetUser.id
    }

    send(message)
  }

  return {
    sendText,
    sendImage,
    sendVideo,
    retryMessage,
    sendTypingStatus
  }
}
