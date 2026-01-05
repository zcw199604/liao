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
})

