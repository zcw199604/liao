import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const sendMock = vi.fn()
const toastShow = vi.fn()

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => ({
    send: sendMock
  })
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/utils/id', () => ({
  generateRandomHexId: () => 'hex-1'
}))

vi.mock('@/api/chat', () => ({
  toggleFavorite: vi.fn(),
  cancelFavorite: vi.fn()
}))

vi.mock('@/api/media', () => ({
  recordImageSend: vi.fn()
}))

import { useChat } from '@/composables/useChat'
import { useMessage } from '@/composables/useMessage'
import * as chatApi from '@/api/chat'
import * as mediaApi from '@/api/media'

import { useChatStore } from '@/stores/chat'
import { useFavoriteStore } from '@/stores/favorite'
import { useMessageStore } from '@/stores/message'
import { useUserStore } from '@/stores/user'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('composables/useMessage', () => {
  it('sendText is a no-op when currentUser or targetUser is missing', () => {
    // no current user
    useMessage().sendText('hi', { id: 'u1', nickname: 'U1' })
    expect(sendMock).not.toHaveBeenCalled()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    // no target user
    useMessage().sendText('hi', null as any)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('sendText constructs act and sends message', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendText('hi', { id: 'u1', nickname: 'U1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u1_U1',
      id: 'me',
      msg: 'hi'
    })
  })

  it('sendText uses name when nickname is missing', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendText('hi', { id: 'u1', name: 'NameOnly' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u1_NameOnly',
      id: 'me',
      msg: 'hi'
    })
  })

  it('sendText buildAct falls back to empty name when targetUser has no name/nickname', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendText('hi', { id: 'u1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u1_',
      id: 'me',
      msg: 'hi'
    })
  })

  it('sendText inserts optimistic message and marks failed on timeout', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const messageStore = useMessageStore()
      sendMock.mockReturnValue(true)

      useMessage().sendText('hello', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-1' })

      const first = messageStore.getMessages('u1')[0] as any
      expect(first.clientId).toBe('cid-1')
      expect(first.sendStatus).toBe('sending')

      vi.advanceTimersByTime(15000)

      const updated = messageStore.getMessages('u1')[0] as any
      expect(updated.sendStatus).toBe('failed')
      expect(updated.sendError).toBe('发送超时')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('sendText marks optimistic message failed when ws send returns false', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const messageStore = useMessageStore()
    sendMock.mockReturnValue(false)

    useMessage().sendText('oops', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-2' })

    const msg = messageStore.getMessages('u1')[0] as any
    expect(msg.clientId).toBe('cid-2')
    expect(msg.sendStatus).toBe('failed')
    expect(msg.sendError).toBe('发送失败')
  })

  it('retryMessage resets failed message to sending and re-sends', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const messageStore = useMessageStore()
      sendMock.mockReturnValueOnce(false).mockReturnValueOnce(true)

      useMessage().sendText('hello', { id: 'u1', nickname: 'U1' }, { clientId: 'cid-3' })
      expect(sendMock).toHaveBeenCalledTimes(1)

      const failed = messageStore.getMessages('u1')[0] as any
      expect(failed.sendStatus).toBe('failed')

      useMessage().retryMessage(failed)
      expect(sendMock).toHaveBeenCalledTimes(2)

      const retrying = messageStore.getMessages('u1')[0] as any
      expect(retrying.clientId).toBe('cid-3')
      expect(retrying.sendStatus).toBe('sending')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('sendTypingStatus toggles inputStatusOn/off', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().sendTypingStatus(true, { id: 'u1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'inputStatusOn_me_Me',
      destuserid: 'u1'
    })

    useMessage().sendTypingStatus(false, { id: 'u1' })
    expect(sendMock).toHaveBeenCalledWith({
      act: 'inputStatusOff_me_Me',
      destuserid: 'u1'
    })
  })

  it('sendImage wraps remote file path and records send relation', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const mockedRecord = vi.mocked(mediaApi.recordImageSend)
    mockedRecord.mockResolvedValue({ code: 0 } as any)

    await useMessage().sendImage('http://s:9006/img/Upload/a.png', { id: 'u2', nickname: 'U2' }, 'a.png')

    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u2_U2',
      id: 'me',
      msg: '[a.png]'
    })
    expect(mockedRecord).toHaveBeenCalledWith({
      remoteUrl: 'http://s:9006/img/Upload/a.png',
      fromUserId: 'me',
      toUserId: 'u2',
      localFilename: 'a.png'
    })
  })

  it('sendImage does not throw when recordImageSend fails', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(true)
    vi.mocked(mediaApi.recordImageSend).mockRejectedValue(new Error('boom'))

    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      await useMessage().sendImage('http://s:9006/img/Upload/a.png', { id: 'u2', nickname: 'U2' })
      expect(sendMock).toHaveBeenCalled()
      expect(warnSpy).toHaveBeenCalled()
    } finally {
      warnSpy.mockRestore()
    }
  })

  it('sendImage/sendVideo return early when mediaUrl is empty', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    await useMessage().sendImage('', { id: 'u2', nickname: 'U2' })
    await useMessage().sendVideo('', { id: 'u2', nickname: 'U2' })
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('sendVideo sends bracketed remote file path and records send relation', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(true)
    vi.mocked(mediaApi.recordImageSend).mockResolvedValue({ code: 0 } as any)

    const messageStore = useMessageStore()
    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', { id: 'u2', nickname: 'U2' }, 'a.mp4', { clientId: 'cid-v1' })

    expect(sendMock).toHaveBeenCalledWith({
      act: 'touser_u2_U2',
      id: 'me',
      msg: '[a.mp4]'
    })

    const msg = (messageStore.getMessages('u2') as any[]).find(m => m.clientId === 'cid-v1')
    expect(msg.type).toBe('video')
    expect(msg.sendStatus).toBe('sending')

    expect(mediaApi.recordImageSend).toHaveBeenCalledWith({
      remoteUrl: 'http://s:9006/img/Upload/a.mp4',
      fromUserId: 'me',
      toUserId: 'u2',
      localFilename: 'a.mp4'
    })
  })

  it('sendVideo marks optimistic message failed when ws send returns false', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    sendMock.mockReturnValue(false)
    vi.mocked(mediaApi.recordImageSend).mockResolvedValue({ code: 0 } as any)

    const messageStore = useMessageStore()
    await useMessage().sendVideo('http://s:9006/img/Upload/a.mp4', { id: 'u2', nickname: 'U2' }, 'a.mp4', { clientId: 'cid-v2' })

    const msg = (messageStore.getMessages('u2') as any[]).find(m => m.clientId === 'cid-v2')
    expect(msg.sendStatus).toBe('failed')
    expect(msg.sendError).toBe('发送失败')
  })

  it('retryMessage is a no-op when required fields are missing', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    useMessage().retryMessage({} as any)
    useMessage().retryMessage({ touser: { id: 'u1', nickname: 'U1' } } as any)
    useMessage().retryMessage({ touser: { id: 'u1', nickname: 'U1' }, clientId: '' } as any)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('retryMessage marks message failed when resend fails', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const messageStore = useMessageStore()
      messageStore.addMessage('u1', {
        code: 7,
        fromuser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
        touser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '' },
        type: 'text',
        content: 'hello',
        time: '2026-01-01 00:00:00.000',
        tid: '',
        isSelf: true,
        isImage: false,
        isVideo: false,
        isFile: false,
        imageUrl: '',
        videoUrl: '',
        fileUrl: '',
        segments: [],
        clientId: 'cid-r',
        sendStatus: 'failed',
        sendError: 'x',
        optimistic: true
      } as any)

      sendMock.mockReturnValue(false)

      const msg = (messageStore.getMessages('u1') as any[])[0]
      useMessage().retryMessage(msg)

      const updated = (messageStore.getMessages('u1') as any[])[0]
      expect(updated.sendStatus).toBe('failed')
      expect(updated.sendError).toBe('发送失败')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('sendTypingStatus is a no-op when currentUser/targetUser is missing', () => {
    useMessage().sendTypingStatus(true, { id: 'u1' })
    expect(sendMock).not.toHaveBeenCalled()

    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    useMessage().sendTypingStatus(true, null as any)
    expect(sendMock).not.toHaveBeenCalled()
  })
})

describe('composables/useChat', () => {
  it('startMatch fails when WebSocket is disconnected', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false

    const ok = useChat().startMatch()
    expect(ok).toBe(false)
    expect(toastShow).toHaveBeenCalledWith('WebSocket 未连接，无法匹配')
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('startMatch sends random message and updates matching state', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true

    const ok = useChat().startMatch()
    expect(ok).toBe(true)
    expect(chatStore.isMatching).toBe(true)
    expect(sendMock).toHaveBeenCalledWith({
      act: 'random',
      id: 'me',
      userAge: '0'
    })
  })

  it('startMatch in continuous mode does not toggle isMatching via startMatch()', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.isMatching = false

    const ok = useChat().startMatch(true)
    expect(ok).toBe(true)
    expect(chatStore.isMatching).toBe(false)
    expect(sendMock).toHaveBeenCalled()
  })

  it('cancelMatch cancels continuous matching and sends randomOut when connected', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = true
    chatStore.startContinuousMatch(3)

    useChat().cancelMatch()
    expect(chatStore.continuousMatchConfig.enabled).toBe(false)
    expect(sendMock).toHaveBeenCalledWith({
      act: 'randomOut',
      id: 'me',
      msg: 'hex-1'
    })
  })

  it('cancelMatch does not send randomOut when ws is disconnected', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.wsConnected = false
    chatStore.startContinuousMatch(2)

    useChat().cancelMatch()
    expect(chatStore.continuousMatchConfig.enabled).toBe(false)
    expect(sendMock).not.toHaveBeenCalled()
  })

  it('loadUsers loads history and favorite lists for current user', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    vi.spyOn(chatStore, 'loadHistoryUsers').mockResolvedValue(undefined as any)
    vi.spyOn(chatStore, 'loadFavoriteUsers').mockResolvedValue(undefined as any)

    await useChat().loadUsers()
    expect(chatStore.loadHistoryUsers).toHaveBeenCalledWith('me', 'Me')
    expect(chatStore.loadFavoriteUsers).toHaveBeenCalledWith('me', 'Me')
  })

  it('handleAutoMatch shows success immediately when total=1', () => {
    const chatStore = useChatStore()
    chatStore.startContinuousMatch(1)

    useChat().handleAutoMatch()
    expect(toastShow).toHaveBeenCalledWith('匹配成功！')
  })

  it('handleAutoMatch cancels after last match when total>1', () => {
    vi.useFakeTimers()
    try {
      const chatStore = useChatStore()
      chatStore.startContinuousMatch(2)
      chatStore.continuousMatchConfig.current = 2

      useChat().handleAutoMatch()
      expect(chatStore.continuousMatchConfig.enabled).toBe(true)

      vi.advanceTimersByTime(2000)
      expect(chatStore.continuousMatchConfig.enabled).toBe(false)
      expect(toastShow).toHaveBeenCalledWith('连续匹配完成！共匹配 2 次')
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('handleAutoMatch schedules next match when not finished', () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

      const chatStore = useChatStore()
      chatStore.wsConnected = true
      chatStore.startContinuousMatch(3)

      useChat().handleAutoMatch()
      vi.advanceTimersByTime(2000)

      expect(chatStore.continuousMatchConfig.current).toBe(2)
      expect(sendMock).toHaveBeenCalledWith({ act: 'random', id: 'me', userAge: '0' })
    } finally {
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  it('enterChat uses incremental load when cached messages exist', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    const loadSpy = vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(0)

    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 2
    })

    messageStore.addMessage('u2', {
      code: 7,
      fromuser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      touser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' },
      type: 'text',
      content: 'x',
      time: '2026-01-01 00:00:00.000',
      tid: '1',
      isSelf: true,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: ''
    } as any)

    const user = chatStore.getUser('u2') as any
    const updateSpy = vi.spyOn(chatStore, 'updateUser')

    useChat().enterChat(user, true)
    expect(updateSpy).toHaveBeenCalledWith('u2', { unreadCount: 0 })
    expect(loadSpy).toHaveBeenCalledWith('me', 'u2', expect.objectContaining({ incremental: true }))
  })

  it('enterChat loads history normally when no cached messages exist', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    const loadSpy = vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(0)

    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    const user = chatStore.getUser('u2') as any
    useChat().enterChat(user, true)
    expect(loadSpy).toHaveBeenCalledWith('me', 'u2', expect.not.objectContaining({ incremental: true }))
  })

  it('toggleFavorite updates store and list ids', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'U3',
      nickname: 'U3',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    const favoriteStore = useFavoriteStore()
    vi.spyOn(favoriteStore, 'addFavorite').mockResolvedValue(true)
    vi.spyOn(favoriteStore, 'removeFavorite').mockResolvedValue(true)

    const mockedToggle = vi.mocked(chatApi.toggleFavorite)
    mockedToggle.mockResolvedValue({ code: '0' } as any)

    const user = chatStore.getUser('u3') as any
    await useChat().toggleFavorite(user)
    expect(user.isFavorite).toBe(true)
    expect(chatStore.favoriteUserIds[0]).toBe('u3')
    expect(toastShow).toHaveBeenCalledWith('收藏成功')

    const mockedCancel = vi.mocked(chatApi.cancelFavorite)
    mockedCancel.mockResolvedValue({ code: '0' } as any)

    await useChat().toggleFavorite(user)
    expect(user.isFavorite).toBe(false)
    expect(chatStore.favoriteUserIds).not.toContain('u3')
    expect(toastShow).toHaveBeenCalledWith('取消收藏成功')
  })

  it('toggleFavorite shows failure message when backend returns non-ok', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'U3',
      nickname: 'U3',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    vi.mocked(chatApi.toggleFavorite).mockResolvedValue({ code: '1', msg: 'bad' } as any)
    await useChat().toggleFavorite(chatStore.getUser('u3') as any)
    expect(toastShow).toHaveBeenCalledWith('操作失败: bad')
    expect((chatStore.getUser('u3') as any).isFavorite).toBe(false)
  })

  it('toggleFavorite catches unexpected errors', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u3',
      name: 'U3',
      nickname: 'U3',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    })

    vi.mocked(chatApi.toggleFavorite).mockRejectedValue(new Error('boom'))
    await useChat().toggleFavorite(chatStore.getUser('u3') as any)
    expect(toastShow).toHaveBeenCalledWith('操作失败')
  })
})
