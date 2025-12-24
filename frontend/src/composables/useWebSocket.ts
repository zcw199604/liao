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

    ws.onmessage = (event) => {
      console.log('收到消息:', event.data)

      try {
        const data: WebSocketMessage = JSON.parse(event.data)
        const code = Number((data as any)?.code)

        // 检测forceout消息（code=-3, forceout=true）
        if (code === -3 && data.forceout === true) {
          console.error('收到forceout消息，停止重连:', data.content)
          forceoutFlag.value = true
          alert(data.content || '请不要在同一个浏览器下重复登录')
          chatStore.wsConnected = false
          if (ws) {
            ws.close()
          }
          return
        }

        // 检测后端拒绝消息（code=-4, forceout=true）
        if (code === -4 && data.forceout === true) {
          console.error('后端拒绝连接:', data.content)
          forceoutFlag.value = true
          alert(data.content || '连接被拒绝，请稍后再试')
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

          chatStore.isMatching = false

          // 更新单一数据源
          chatStore.upsertUser(matchedUser)

          // 移到历史列表最前面
          const historyIds = chatStore.historyUserIds
          const existingIndex = historyIds.indexOf(matchedUser.id)
          if (existingIndex > -1) {
            historyIds.splice(existingIndex, 1)
          }
          historyIds.unshift(matchedUser.id)

          // 初始化聊天记录为空，并进入聊天（不加载历史）
          messageStore.clearHistory(matchedUser.id)
          chatStore.enterChat(matchedUser as any)

          window.dispatchEvent(new CustomEvent('match-success', { detail: matchedUser }))
          return
        }

        // 处理聊天消息 (code: 7)
        if (code === 7 && data.fromuser) {
          console.log('收到聊天消息:', data)

          const fromUserId = data.fromuser.id
          const toUserId = data.touser ? data.touser.id : null
          let messageContent = data.fromuser.content
          const fromUserNickname = data.fromuser.nickname

          console.log('解析消息 - fromUserId=', fromUserId, 'toUserId=', toUserId, 'currentUserId=', currentUser.id)
          console.log('消息内容:', messageContent)
          console.log('当前聊天对象:', chatStore.currentChatUser ? chatStore.currentChatUser.id : '无')

          // 判断是不是自己发送的消息（通过nickname判断）
          const isSelf = data.fromuser.nickname === currentUser.nickname
          console.log('isSelf=', isSelf, '(通过nickname判断)')

          // 判断是否应该显示这条消息（通过nickname判断）
          const shouldDisplay = chatStore.currentChatUser &&
            (data.fromuser.nickname === chatStore.currentChatUser.nickname ||
             (data.touser && data.touser.nickname === chatStore.currentChatUser.nickname))

          console.log('shouldDisplay=', shouldDisplay)

          // 解析消息类型
          let isImage = false
          let isVideo = false
          let imageUrl = ''
          let videoUrl = ''

          if (messageContent && typeof messageContent === 'string') {
            const raw = messageContent

            // 检查是否是图片/视频消息（格式：[path/to/file.jpg]）
            if (raw.startsWith('[') && raw.endsWith(']')) {
              const path = raw.substring(1, raw.length - 1)
              const isMp4 = path.toLowerCase().includes('.mp4')
              const isImg = !isMp4 && /\.(jpg|jpeg|png|gif|webp)$/i.test(path)

              if (mediaStore.imgServer && (isMp4 || isImg)) {
                const port = isMp4 ? '8006' : '9006'
                const url = `http://${mediaStore.imgServer}:${port}/img/Upload/${path}`
                messageContent = url

                if (isMp4) {
                  isVideo = true
                  videoUrl = url
                } else {
                  isImage = true
                  imageUrl = url
                }
              }
            }
          }

          // 构建聊天消息对象
          const chatMessage: ChatMessage = {
            code,
            fromuser: data.fromuser,
            touser: data.touser,
            type: isImage ? 'image' : isVideo ? 'video' : (data.type || 'text'),
            content: messageContent,
            time: (data as any).fromuser?.time || (data as any).time || '',
            tid: (data as any).tid || (data as any).fromuser?.tid || '',
            isSelf,
            isImage,
            isVideo,
            imageUrl,
            videoUrl
          }

          const targetUserId = isSelf
            ? (chatStore.currentChatUser?.id || data.touser?.id || '')
            : fromUserId

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

          const lastMsg = isImage ? '[图片]' : (isVideo ? '[视频]' : messageContent)

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
            const existingUser = chatStore.getUser(fromUserId)

            if (existingUser) {
              // 用户已存在 - 更新状态
              chatStore.updateUser(fromUserId, {
                lastMsg,
                lastTime: '刚刚',
                unreadCount: (existingUser.unreadCount || 0) + 1
              })

              // 移到历史列表最前面
              const historyIds = chatStore.historyUserIds
              const existingIndex = historyIds.indexOf(fromUserId)
              if (existingIndex > -1) {
                historyIds.splice(existingIndex, 1)
              }
              historyIds.unshift(fromUserId)

            } else if (fromUserId !== currentUser.id) {
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
          const now = new Date().toISOString().replace('T', ' ').substring(0, 23)
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

        const now = new Date().toISOString().replace('T', ' ').substring(0, 23)
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

  return {
    connect,
    send,
    disconnect,
    setScrollToBottom,
    forceoutFlag
  }
}
