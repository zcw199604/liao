import { ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useMessageStore } from '@/stores/message'
import { useUserStore } from '@/stores/user'
import { generateCookie } from '@/utils/cookie'
import { WS_URL } from '@/constants/config'
import type { WebSocketMessage, ChatMessage, User } from '@/types'
import * as chatApi from '@/api/chat'
import { useMediaStore } from '@/stores/media'
import { useToast } from '@/composables/useToast'
import { emojiMap } from '@/constants/emoji'
import { isImageFile, isVideoFile } from '@/utils/file'

let ws: WebSocket | null = null
let manualClose = false
const forceoutFlag = ref(false)

// 滚动到底部的方法引用（全局单例）
let scrollToBottomCallback: (() => void) | null = null

export const useWebSocket = () => {
  const chatStore = useChatStore()
  const messageStore = useMessageStore()
  const userStore = useUserStore()
  const mediaStore = useMediaStore()
  const { show } = useToast()

  const formatNow = () => {
    const now = new Date()
    const pad = (value: number, length: number = 2) => String(value).padStart(length, '0')
    return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}.${pad(now.getMilliseconds(), 3)}`
  }

  const toPlainText = (input: string) => {
    return input
      .replace(/<br\s*\/?>/gi, ' ')
      .replace(/<[^>]*>/g, '')
      .replace(/\s+/g, ' ')
      .trim()
  }

  const setScrollToBottom = (callback: () => void) => {
    scrollToBottomCallback = callback
  }

  const scrollToBottom = () => {
    if (scrollToBottomCallback) {
      scrollToBottomCallback()
    }
  }

  const connect = () => {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      console.log('WebSocket 已连接/正在连接，跳过重复连接')
      return
    }

    const token = localStorage.getItem('authToken')
    if (!token) {
      console.error('没有Token，无法连接WebSocket')
      return
    }

    const currentUser = userStore.currentUser
    if (!currentUser) {
      console.error('没有当前用户，无法连接WebSocket')
      return
    }

    // WebSocket URL中添加token参数
    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const wsUrlWithToken = `${scheme}://${window.location.host}${WS_URL}?token=${encodeURIComponent(token)}`
    console.log('正在连接 WebSocket (已携带Token):', wsUrlWithToken)

    ws = new WebSocket(wsUrlWithToken)

    ws.onopen = async () => {
      console.log('WebSocket 连接成功')
      chatStore.wsConnected = true
      forceoutFlag.value = false

      // 发送登录消息
      const signMessage = {
        "act": "sign",
        "id": currentUser.id,
        "name": currentUser.nickname,
        "userSex": currentUser.sex,
        "address_show": "false",
        "randomhealthmode": "0",
        "randomvipsex": "0",
        "randomvipaddress": "0",
        "userip": currentUser.ip,
        "useraddree": currentUser.area,
        "randomvipcode": ""
      }

      const signMsg = JSON.stringify(signMessage)
      ws?.send(signMsg)
      console.log('已发送登录消息:', signMsg)

      // 上报访问记录 -> 获取图片服务器地址 -> 刷新缓存图片
      try {
        const cookieData = generateCookie(currentUser.id, currentUser.name)
        const referer = 'http://v1.chat2019.cn/randomdeskrynewjc46ko.html?v=jc46ko'
        const userAgent = navigator.userAgent

        await chatApi.reportReferrer({
          referrerUrl: document.referrer || '',
          currUrl: window.location.href,
          userid: currentUser.id,
          cookieData,
          referer,
          userAgent
        })
      } catch (e) {
        console.warn('上报访问记录失败:', e)
      }

      try {
        await mediaStore.loadImgServer()
        if (mediaStore.imgServer) {
          await mediaStore.loadCachedImages(currentUser.id)
        }
      } catch (e) {
        console.warn('初始化图片服务器信息失败:', e)
      }
    }

    ws.onmessage = async (event) => {
      console.log('收到消息:', event.data)

      try {
        const data: WebSocketMessage = JSON.parse(event.data)
        const code = Number((data as any)?.code)

        // 检测forceout消息（code=-3, forceout=true）
        if (code === -3 && data.forceout === true) {
          console.error('收到forceout消息，停止重连:', data.content)
          forceoutFlag.value = true
          // 使用 Router 或直接跳转到登录页，带上错误信息
          chatStore.wsConnected = false
          if (ws) {
            ws.close()
          }
          // 清除Token并跳转
          localStorage.removeItem('authToken')
          // 这里使用 reload 或 href 跳转来确保彻底断开和重置状态
          window.location.href = `/?error=${encodeURIComponent(data.content || '请不要在同一个浏览器下重复登录')}`
          return
        }

        // 检测后端拒绝消息（code=-4, forceout=true）
        if (code === -4 && data.forceout === true) {
          console.error('后端拒绝连接:', data.content)
          forceoutFlag.value = true
          window.location.href = `/?error=${encodeURIComponent(data.content || '连接被拒绝，请稍后再试')}`
          return
        }

        // 过滤系统消息（code: 12, 13, 16, 19 等），用 toast 显示
        if (code === 12 || code === 13 || code === 16 || code === 19) {
          console.log('系统消息，toast显示:', data)
          if (data.content) {
            show(data.content)
          }
          return
        }

        // 忽略特定消息（code: 18）
        if (code === 18) {
          console.log('忽略消息 code=18:', data)
          return
        }

        // 处理正在输入消息
        if (data.act && (data.act.startsWith('inputStatusOn_') || data.act.startsWith('inputStatusOff_'))) {
          const isOn = data.act.startsWith('inputStatusOn_')
          const parts = data.act.split('_')
          const typingUserId = parts[1]

          console.log('正在输入状态:', isOn ? '开始' : '结束', 'userId=', typingUserId)

          // 如果当前正在和这个用户聊天，显示/隐藏正在输入提示
          if (chatStore.currentChatUser && chatStore.currentChatUser.id === typingUserId) {
            messageStore.isTyping = isOn
            console.log('更新正在输入状态:', messageStore.isTyping)

            // 如果显示正在输入，滚动到底部
            if (isOn) {
              scrollToBottom()
            }
          }
          return
        }

        // 处理匹配成功消息 (code: 15)
        if (code === 15 && (data as any).sel_userid) {
          const matchedUser: User = {
            id: String((data as any).sel_userid),
            name: String((data as any).sel_userNikename || '匿名用户'),
            nickname: String((data as any).sel_userNikename || '匿名用户'),
            sex: String((data as any).sel_userSex || '未知'),
            age: String((data as any).sel_userAge || '0'),
            area: String((data as any).sel_userAddress || '未知'),
            address: String((data as any).sel_userAddress || '未知'),
            ip: '',
            isFavorite: false,
            lastMsg: '匹配成功',
            lastTime: '刚刚',
            unreadCount: 0
          }

          // 更新单一数据源
          chatStore.upsertUser(matchedUser)

          // 移到历史列表最前面
          const historyIds = chatStore.historyUserIds
          const existingIndex = historyIds.indexOf(matchedUser.id)
          if (existingIndex > -1) {
            historyIds.splice(existingIndex, 1)
          }
          historyIds.unshift(matchedUser.id)

          // 初始化聊天记录为空
          messageStore.clearHistory(matchedUser.id)

          // 检查匹配是否已被取消（用户点击了取消按钮）
          if (!chatStore.isMatching) {
            // 匹配已取消，只更新用户列表，不自动进入聊天
            console.log('匹配已取消，忽略自动进入聊天')
            return
          }

          // 判断是否为连续匹配模式
          if (chatStore.continuousMatchConfig.enabled) {
            // 连续匹配模式：不进入聊天，只更新蒙层显示
            chatStore.setCurrentMatchedUser(matchedUser)
            // 触发自动匹配检查
            window.dispatchEvent(new CustomEvent('match-auto-check'))
          } else {
            // 单次匹配模式：进入聊天
            chatStore.isMatching = false
            chatStore.enterChat(matchedUser as any)
            window.dispatchEvent(new CustomEvent('match-success', { detail: matchedUser }))
          }

          return
        }

        // 处理在线状态查询结果 (code: 30)
        if (code === 30) {
          console.log('收到在线状态查询结果:', data)
          window.dispatchEvent(new CustomEvent('check-online-result', { detail: data }))
          return
        }

        // 处理聊天消息 (code: 7)
        if (code === 7 && data.fromuser) {
          console.log('收到聊天消息:', data)

          const currentUserId = String(currentUser.id || '')
          const fromUserId = String((data as any)?.fromuser?.id ?? '')
          const toUserId = String((data as any)?.touser?.id ?? '')
          let messageContent = String(
            (data as any)?.fromuser?.content ??
            (data as any)?.content ??
            (data as any)?.msg ??
            ''
          )
          const fromUserNickname = String((data as any)?.fromuser?.nickname ?? (data as any)?.fromuser?.name ?? '')

          console.log('解析消息 - fromUserId=', fromUserId, 'toUserId=', toUserId, 'currentUserId=', currentUser.id)
          console.log('消息内容:', messageContent)
          console.log('当前聊天对象:', chatStore.currentChatUser ? chatStore.currentChatUser.id : '无')

          // 判断是不是自己发送的消息（通过nickname判断）
          const isSelf = fromUserNickname === currentUser.nickname
          console.log('isSelf=', isSelf, '(通过nickname判断)')

          // 判断是否应该显示这条消息（通过nickname判断）
          const shouldDisplay = chatStore.currentChatUser &&
            (fromUserNickname === chatStore.currentChatUser.nickname ||
             (data.touser && data.touser.nickname === chatStore.currentChatUser.nickname))

          console.log('shouldDisplay=', shouldDisplay)

          // 解析消息类型
          let isImage = false
          let isVideo = false
          let isFile = false
          let imageUrl = ''
          let videoUrl = ''
          let fileUrl = ''

          if (messageContent && typeof messageContent === 'string') {
            const raw = messageContent

            // 检查是否是媒体消息（格式：[path/to/file.ext]），对齐 loadHistory 的解析逻辑
            if (raw.startsWith('[') && raw.endsWith(']')) {
              // 如果是表情包，不处理为文件
              if (!emojiMap[raw]) {
                const path = raw.substring(1, raw.length - 1)
                const isVideoPath = isVideoFile(path)
                const isImagePath = isImageFile(path)

                // 对齐 loadHistory：需要时尝试补全 imgServer，避免消息先到导致无法拼接URL
                if (!mediaStore.imgServer) {
                  try {
                    await mediaStore.loadImgServer()
                  } catch {
                    // ignore
                  }
                }

                if (mediaStore.imgServer) {
                  const port = isVideoPath ? '8006' : '9006'
                  const url = `http://${mediaStore.imgServer}:${port}/img/Upload/${path}`
                  messageContent = url

                  if (isVideoPath) {
                    isVideo = true
                    videoUrl = url
                  } else if (isImagePath) {
                    isImage = true
                    imageUrl = url
                  } else {
                    isFile = true
                    fileUrl = url
                  }
                }
              }
            }
          }

          // 构建聊天消息对象
          const resolvedTime = String(
            (data as any)?.fromuser?.time ??
            (data as any)?.fromuser?.Time ??
            (data as any)?.time ??
            (data as any)?.Time ??
            ''
          )
          const resolvedTid = String(
            (data as any)?.tid ??
            (data as any)?.Tid ??
            (data as any)?.fromuser?.tid ??
            (data as any)?.fromuser?.Tid ??
            ''
          )
          const time = resolvedTime || formatNow()
          const tid = resolvedTid || `${Date.now()}${Math.floor(Math.random() * 1000).toString().padStart(3, '0')}`

          const chatMessage: ChatMessage = {
            code,
            fromuser: data.fromuser,
            touser: data.touser,
            type: isImage ? 'image' : isVideo ? 'video' : isFile ? 'file' : (data.type || 'text'),
            content: messageContent,
            time,
            tid,
            isSelf,
            isImage,
            isVideo,
            isFile,
            imageUrl,
            videoUrl,
            fileUrl
          }

          // 只有与当前聊天对象相关的消息，才存储到currentChatUser.id下
          const targetUserId = shouldDisplay && chatStore.currentChatUser?.id
            ? String(chatStore.currentChatUser.id)
            : (isSelf ? toUserId : fromUserId)

          if (targetUserId) {
            // WebSocket消息去重 - 基于tid
            const existingMessages = messageStore.getMessages(targetUserId)
            const isDuplicate = existingMessages.some(msg =>
              msg.tid && chatMessage.tid && msg.tid === chatMessage.tid
            )

            if (isDuplicate) {
              console.log('WebSocket消息重复（tid已存在），跳过:', chatMessage.tid)
            } else {
              messageStore.addMessage(targetUserId, chatMessage)
              console.log('消息已添加到聊天历史')
            }
          }

          const lastMsg = isImage ? '[图片]' : (isVideo ? '[视频]' : (isFile ? '[文件]' : messageContent))

          if (shouldDisplay) {
            // 收到消息，隐藏正在输入提示
            if (!isSelf) {
              messageStore.isTyping = false
            }

            // 直接更新 userMap 中的对象
            if (chatStore.currentChatUser) {
              chatStore.updateUser(chatStore.currentChatUser.id, {
                lastMsg,
                lastTime: '刚刚',
                unreadCount: 0
              })
            }

            setTimeout(scrollToBottom, 100)
          } else if (!isSelf) {
            // 不在当前聊天界面，但收到消息 - 使用单一数据源更新
            // 使用 nickname 查找用户（与 shouldDisplay 判断逻辑一致）
            const existingUser = chatStore.getUserByNickname(fromUserNickname)

            if (existingUser) {
              // 用户已存在 - 更新状态
              chatStore.updateUser(existingUser.id, {
                lastMsg,
                lastTime: '刚刚',
                unreadCount: (existingUser.unreadCount || 0) + 1
              })

              // 移到历史列表最前面
              const historyIds = chatStore.historyUserIds
              const existingIndex = historyIds.indexOf(existingUser.id)
              if (existingIndex > -1) {
                historyIds.splice(existingIndex, 1)
              }
              historyIds.unshift(existingUser.id)

            } else if (fromUserId && fromUserId !== currentUserId) {
              // 新用户 - 创建并添加
              const newUser: User = {
                id: fromUserId,
                name: fromUserNickname || '匿名用户',
                nickname: fromUserNickname || '匿名用户',
                sex: '未知',
                age: '0',
                area: '未知',
                address: '未知',
                ip: '',
                isFavorite: false,
                lastMsg,
                lastTime: '刚刚',
                unreadCount: 1
              }

              chatStore.upsertUser(newUser)
              chatStore.historyUserIds.unshift(fromUserId)
            }
          }

          return
        }

        // 兜底：对齐旧版逻辑，未识别的消息在聊天界面内直接显示（优先显示在当前聊天窗口）
        const fallbackContent = String((data as any)?.content ?? (data as any)?.msg ?? '')
        if (fallbackContent) {
          const now = formatNow()
          if (chatStore.currentChatUser) {
            const peer = chatStore.currentChatUser as any
            const fromUser = {
              id: peer.id,
              name: peer.nickname || peer.name || '匿名用户',
              nickname: peer.nickname || peer.name || '匿名用户',
              sex: peer.sex || '未知',
              ip: peer.ip || ''
            }

            const chatMessage: ChatMessage = {
              code: Number.isFinite(code) ? code : 0,
              fromuser: fromUser,
              touser: undefined,
              type: 'text',
              content: fallbackContent,
              time: now,
              tid: String(Date.now()),
              isSelf: false,
              isImage: false,
              isVideo: false,
              imageUrl: '',
              videoUrl: ''
            }

            messageStore.addMessage(peer.id, chatMessage)
            peer.lastMsg = fallbackContent
            peer.lastTime = '刚刚'

            setTimeout(scrollToBottom, 100)
          } else {
            show(toPlainText(fallbackContent) || '系统消息')
          }

          return
        }

        console.log('未处理的消息:', data)

      } catch (e) {
        console.error('解析消息失败:', e)
        console.log('原始消息:', event.data)

        // 兜底：非 JSON 消息按旧版逻辑直接插入当前聊天窗口
        const raw = String(event.data || '')
        if (!raw) return

        const now = formatNow()
        if (chatStore.currentChatUser) {
          const peer = chatStore.currentChatUser as any
          const fromUser = {
            id: peer.id,
            name: peer.nickname || peer.name || '匿名用户',
            nickname: peer.nickname || peer.name || '匿名用户',
            sex: peer.sex || '未知',
            ip: peer.ip || ''
          }

          const chatMessage: ChatMessage = {
            code: 0,
            fromuser: fromUser,
            touser: undefined,
            type: 'text',
            content: raw,
            time: now,
            tid: String(Date.now()),
            isSelf: false,
            isImage: false,
            isVideo: false,
            imageUrl: '',
            videoUrl: ''
          }

          messageStore.addMessage(peer.id, chatMessage)
          peer.lastMsg = raw
          peer.lastTime = '刚刚'
          setTimeout(scrollToBottom, 100)
        } else {
          show(toPlainText(raw) || '系统消息')
        }
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket 错误:', error)
      chatStore.wsConnected = false
    }

    ws.onclose = () => {
      console.log('WebSocket 连接关闭')
      chatStore.wsConnected = false

      // WebSocket断开时取消连续匹配
      if (chatStore.continuousMatchConfig.enabled) {
        chatStore.cancelContinuousMatch()
        show('连接断开，连续匹配已取消')
      }

      // 如果是手动关闭（切换身份），不重连
      if (manualClose) {
        console.log('手动关闭，跳过重连')
        manualClose = false
        return
      }

      // 检查forceout标志
      if (forceoutFlag.value) {
        console.log('因forceout被禁止，跳过重连')
        return
      }

      // 尝试重连
      setTimeout(() => {
        console.log('尝试重新连接...')
        connect()
      }, 3000)
    }
  }

  const send = (message: any) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      const msg = JSON.stringify(message)
      ws.send(msg)
      console.log('发送消息:', msg)
    } else {
      console.error('WebSocket未连接，无法发送消息')
      show('连接已断开，请刷新页面重试')
    }
  }

  const disconnect = (manual: boolean = false) => {
    manualClose = manual
    if (ws) {
      ws.close()
      ws = null
    }
  }

  const checkUserOnlineStatus = (targetUserId: string) => {
    const currentUser = userStore.currentUser
    if (!currentUser) return

    const msg = {
      "act": "ShowUserLoginInfo",
      "id": currentUser.id,
      "msg": targetUserId,
      "randomvipcode": "vipali67fbff86676e361016812533"
    }
    send(msg)
  }

  return {
    connect,
    send,
    disconnect,
    setScrollToBottom,
    forceoutFlag,
    checkUserOnlineStatus
  }
}
