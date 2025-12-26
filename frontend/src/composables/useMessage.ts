import { useUserStore } from '@/stores/user'
import { useWebSocket } from './useWebSocket'
import * as mediaApi from '@/api/media'
import { extractRemoteFilePathFromImgUploadUrl } from '@/utils/media'

export const useMessage = () => {
  const userStore = useUserStore()
  const { send } = useWebSocket()

  // 发送文本消息
  const sendText = (content: string, targetUser: any) => {
    if (!userStore.currentUser || !targetUser) return

    const message = {
      act: `touser_${targetUser.id}_${targetUser.nickname || targetUser.name || ''}`,
      id: userStore.currentUser.id,
      msg: content
    }

    send(message)
  }

  // 发送图片消息
  const sendImage = async (mediaUrl: string, targetUser: any, localFilename?: string) => {
    if (!userStore.currentUser || !targetUser) return

    const filePath = extractRemoteFilePathFromImgUploadUrl(mediaUrl)
    if (!filePath) return

    const message = {
      act: `touser_${targetUser.id}_${targetUser.nickname || targetUser.name || ''}`,
      id: userStore.currentUser.id,
      msg: `[${filePath}]`
    }

    send(message)

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
  const sendVideo = async (mediaUrl: string, targetUser: any, localFilename?: string) => {
    if (!userStore.currentUser || !targetUser) return

    const filePath = extractRemoteFilePathFromImgUploadUrl(mediaUrl)
    if (!filePath) return

    const message = {
      act: `touser_${targetUser.id}_${targetUser.nickname || targetUser.name || ''}`,
      id: userStore.currentUser.id,
      msg: `[${filePath}]`
    }

    send(message)

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
    sendTypingStatus
  }
}
