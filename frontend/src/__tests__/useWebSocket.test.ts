import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const toastShow = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

vi.mock('@/api/system', () => ({
  getSystemConfig: vi.fn(),
  updateSystemConfig: vi.fn(),
  resolveImagePort: vi.fn()
}))

vi.mock('@/api/chat', () => ({
  reportReferrer: vi.fn().mockResolvedValue({}),
  getHistoryUserList: vi.fn(),
  getFavoriteUserList: vi.fn(),
  getMessageHistory: vi.fn(),
  toggleFavorite: vi.fn(),
  cancelFavorite: vi.fn()
}))

import { useWebSocket } from '@/composables/useWebSocket'
import router from '@/router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useMediaStore } from '@/stores/media'
import { useMessageStore } from '@/stores/message'
import { useUserStore } from '@/stores/user'
import { md5Hex } from '@/utils/md5'
import * as systemApi from '@/api/system'

class FakeWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  static instances: FakeWebSocket[] = []

  readonly url: string
  readyState = FakeWebSocket.CONNECTING
  sent: string[] = []

  onopen: ((ev?: any) => any) | null = null
  onmessage: ((ev: any) => any) | null = null
  onerror: ((ev?: any) => any) | null = null
  onclose: ((ev?: any) => any) | null = null

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.readyState = FakeWebSocket.CLOSED
    this.onclose?.({})
  }

  async triggerOpen() {
    this.readyState = FakeWebSocket.OPEN
    const ret = this.onopen?.({})
    if (ret && typeof (ret as Promise<any>).then === 'function') {
      await ret
    }
  }

  async triggerMessage(payload: any) {
    const ret = this.onmessage?.({ data: JSON.stringify(payload) })
    if (ret && typeof (ret as Promise<any>).then === 'function') {
      await ret
    }
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())

  // ensure module-scoped singleton connections are cleared before each test
  try {
    useWebSocket().disconnect(true)
  } catch {
    // ignore
  }

  FakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', FakeWebSocket as any)

  vi.mocked(systemApi.getSystemConfig).mockResolvedValue({
    code: 0,
    data: { imagePortMode: 'fixed', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
  } as any)
  vi.mocked(systemApi.resolveImagePort).mockResolvedValue({ code: 0, data: { port: '9006' } } as any)
})

afterEach(() => {
  // best-effort cleanup for module-scoped ws singletons
  try {
    useWebSocket().disconnect(true)
  } catch {
    // ignore
  }
  vi.useRealTimers()
})

describe('composables/useWebSocket', () => {
  it('connect returns early when token is missing', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const socket = useWebSocket()
    socket.connect()

    expect(FakeWebSocket.instances).toHaveLength(0)
  })

  it('connect returns early when currentUser is missing', () => {
    localStorage.setItem('authToken', 't-1')
    const socket = useWebSocket()
    socket.connect()
    expect(FakeWebSocket.instances).toHaveLength(0)
  })

  it('connect resets stale closed connection before reconnecting', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    expect(FakeWebSocket.instances).toHaveLength(1)

    FakeWebSocket.instances[0]!.readyState = FakeWebSocket.CLOSED
    socket.connect()
    expect(FakeWebSocket.instances).toHaveLength(2)
  })

  it('onopen ignores outdated connection after identity switch', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'a', name: 'A', nickname: 'A', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    const ws1 = FakeWebSocket.instances[0]!

    userStore.currentUser = { id: 'b', name: 'B', nickname: 'B', sex: '女', ip: '', area: '' } as any
    socket.connect()
    const ws2 = FakeWebSocket.instances[1]!

    await ws1.triggerOpen()
    expect(chatStore.wsConnected).toBe(false)

    await ws2.triggerOpen()
    expect(chatStore.wsConnected).toBe(true)
  })

  it('onmessage ignores outdated connection after identity switch', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'a', name: 'A', nickname: 'A', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    const ws1 = FakeWebSocket.instances[0]!

    userStore.currentUser = { id: 'b', name: 'B', nickname: 'B', sex: '女', ip: '', area: '' } as any
    socket.connect()
    const ws2 = FakeWebSocket.instances[1]!
    await ws2.triggerOpen()

    const messageStore = useMessageStore()
    await ws1.triggerMessage({
      code: 7,
      fromuser: { id: 'u1', name: 'U1', nickname: 'U1', sex: '未知', ip: '', content: 'x', time: '2026-01-01 00:00:00.000', tid: 't1' },
      touser: { id: 'b', name: 'B', nickname: 'B', sex: '未知', ip: '' },
      tid: 't1'
    })
    expect(messageStore.getMessages('u1')).toHaveLength(0)
  })

  it('connect skips duplicate connect for same identity', async () => {
    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'Me',
      nickname: 'Me',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN'
    } as any

    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    socket.connect()
    expect(FakeWebSocket.instances).toHaveLength(1)

    await FakeWebSocket.instances[0]!.triggerOpen()
    expect(useChatStore().wsConnected).toBe(true)
  })

  it('connect sends sign message and sets wsConnected on open', async () => {
    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'Me',
      nickname: 'Me',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN'
    } as any

    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    const mediaStore = useMediaStore()

    vi.spyOn(mediaStore, 'loadImgServer').mockImplementation(async () => {
      mediaStore.imgServer = '1.2.3.4'
    })
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()

    expect(FakeWebSocket.instances).toHaveLength(1)
    expect(FakeWebSocket.instances[0]?.url).toContain('/ws?token=')

    await FakeWebSocket.instances[0]!.triggerOpen()

    expect(chatStore.wsConnected).toBe(true)
    expect(mediaStore.loadImgServer).toHaveBeenCalledOnce()
    expect(mediaStore.loadCachedImages).toHaveBeenCalledWith('me')

    const sign = JSON.parse(FakeWebSocket.instances[0]!.sent[0] || '{}')
    expect(sign.act).toBe('sign')
    expect(sign.id).toBe('me')

    socket.send({ a: 1 })
    expect(FakeWebSocket.instances[0]!.sent.some(s => s === JSON.stringify({ a: 1 }))).toBe(true)
  })

  it('reconnects and re-signs when identity changes', async () => {
    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'a',
      name: 'A',
      nickname: 'A',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN'
    } as any

    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    expect(FakeWebSocket.instances).toHaveLength(1)

    await FakeWebSocket.instances[0]!.triggerOpen()
    expect(chatStore.wsConnected).toBe(true)

    const signA = JSON.parse(FakeWebSocket.instances[0]!.sent[0] || '{}')
    expect(signA.act).toBe('sign')
    expect(signA.id).toBe('a')

    userStore.currentUser = {
      id: 'b',
      name: 'B',
      nickname: 'B',
      sex: '女',
      ip: '127.0.0.2',
      area: 'CN'
    } as any

    socket.connect()
    expect(FakeWebSocket.instances).toHaveLength(2)

    await FakeWebSocket.instances[1]!.triggerOpen()
    const signB = JSON.parse(FakeWebSocket.instances[1]!.sent[0] || '{}')
    expect(signB.act).toBe('sign')
    expect(signB.id).toBe('b')
  })

  it('auto reconnects after unexpected close', async () => {
    vi.useFakeTimers()

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'Me',
      nickname: 'Me',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN'
    } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const chatStore = useChatStore()

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    expect(chatStore.wsConnected).toBe(true)

    FakeWebSocket.instances[0]!.close()
    expect(chatStore.wsConnected).toBe(false)

    await vi.advanceTimersByTimeAsync(3000)
    expect(FakeWebSocket.instances).toHaveLength(2)

    await FakeWebSocket.instances[1]!.triggerOpen()
    const sign = JSON.parse(FakeWebSocket.instances[1]!.sent[0] || '{}')
    expect(sign.act).toBe('sign')
    expect(sign.id).toBe('me')
  })

  it('manual disconnect prevents auto reconnect', async () => {
    vi.useFakeTimers()

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'Me',
      nickname: 'Me',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN'
    } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    socket.disconnect(true)
    await vi.advanceTimersByTimeAsync(3000)

    expect(FakeWebSocket.instances).toHaveLength(1)
  })

  it('forceout clears token and prevents auto reconnect', async () => {
    vi.useFakeTimers()

    const userStore = useUserStore()
    userStore.currentUser = {
      id: 'me',
      name: 'Me',
      nickname: 'Me',
      sex: '男',
      ip: '127.0.0.1',
      area: 'CN'
    } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const chatStore = useChatStore()
    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    expect(chatStore.wsConnected).toBe(true)

    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      await FakeWebSocket.instances[0]!.triggerMessage({ code: -3, forceout: true, content: 'x' })
    } finally {
      errorSpy.mockRestore()
    }

    expect(chatStore.wsConnected).toBe(false)
    expect(localStorage.getItem('authToken')).toBeNull()

    await vi.advanceTimersByTimeAsync(3000)
    expect(FakeWebSocket.instances).toHaveLength(1)
  })

  it('handles typing status messages for current chat user and triggers scroll callback', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.currentChatUser = {
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
    } as any

    const messageStore = useMessageStore()
    const scrollSpy = vi.fn()
    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.setScrollToBottom(scrollSpy)
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ act: 'inputStatusOn_u2_x' })
    expect(messageStore.isTyping).toBe(true)
    expect(scrollSpy).toHaveBeenCalledOnce()

    await FakeWebSocket.instances[0]!.triggerMessage({ act: 'inputStatusOff_u2_x' })
    expect(messageStore.isTyping).toBe(false)
  })

  it('increments unreadCount on /list even when currentChatUser is set (double insurance)', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/list')
    await router.isReady()

    const chatStore = useChatStore()
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
    } as any)
    chatStore.enterChat(chatStore.getUser('u2') as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '', content: 'hello', time: '2026-01-01 00:00:00.000', tid: 't-1' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 't-1'
    })

    const user = chatStore.getUser('u2') as any
    expect(user.unreadCount).toBe(1)
    expect(user.lastMsg).toBe('hello')
  })

  it('does not duplicate history list when peer nickname changes (id-first match)', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/list')
    await router.isReady()

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u2',
      name: 'OldName',
      nickname: 'OldName',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    } as any)
    chatStore.historyUserIds = ['u2']

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u2', name: 'NewName', nickname: 'NewName', sex: '未知', ip: '', content: 'hi', time: '2026-01-01 00:00:00.000', tid: 't-nick' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 't-nick'
    })

    const occurrences = chatStore.historyUserIds.filter(id => id === 'u2')
    expect(occurrences).toHaveLength(1)

    const user = chatStore.getUser('u2') as any
    expect(user.nickname).toBe('NewName')
    expect(user.lastMsg).toBe('hi')
    expect(user.unreadCount).toBe(1)
  })

  it('clears unreadCount on /chat when message belongs to current chat', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/chat/u2')
    await router.isReady()

    const chatStore = useChatStore()
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
      unreadCount: 3
    } as any)
    chatStore.enterChat(chatStore.getUser('u2') as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '', content: 'hello', time: '2026-01-01 00:00:00.000', tid: 't-2' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 't-2'
    })

    const user = chatStore.getUser('u2') as any
    expect(user.unreadCount).toBe(0)
    expect(user.lastMsg).toBe('hello')
  })

  it('promotes both history/favorite lists and prefixes lastMsg for self-sent echo on /chat', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/chat/u2')
    await router.isReady()

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u1',
      name: 'U1',
      nickname: 'U1',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: true,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    } as any)
    chatStore.upsertUser({
      id: 'u2',
      name: 'U2',
      nickname: 'U2',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: true,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    } as any)
    chatStore.historyUserIds = ['u1', 'u2']
    chatStore.favoriteUserIds = ['u1', 'u2']
    chatStore.enterChat(chatStore.getUser('u2') as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: {
        id: md5Hex('me'),
        name: 'Me',
        nickname: 'Me',
        sex: '未知',
        ip: '',
        content: 'hi',
        time: '2026-01-01 00:00:00.000',
        tid: 't-self'
      },
      touser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' },
      tid: 't-self'
    })

    expect(chatStore.historyUserIds[0]).toBe('u2')
    expect(chatStore.favoriteUserIds[0]).toBe('u2')

    const user = chatStore.getUser('u2') as any
    expect(user.unreadCount).toBe(0)
    expect(user.lastMsg).toBe('我: hi')
  })

  it('merges self-sent echo into optimistic message and avoids duplicates', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/chat/u2')
    await router.isReady()

    const chatStore = useChatStore()
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
    } as any)
    chatStore.enterChat(chatStore.getUser('u2') as any)

    const messageStore = useMessageStore()
    messageStore.addMessage('u2', {
      code: 7,
      fromuser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      touser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' },
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
      clientId: 'cid-1',
      sendStatus: 'sending',
      optimistic: true
    } as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: {
        id: md5Hex('me'),
        name: 'Me',
        nickname: 'Me',
        sex: '未知',
        ip: '',
        content: 'hello',
        time: '2026-01-01 00:00:00.000',
        tid: 't-echo'
      },
      touser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' },
      tid: 't-echo'
    })

    const msgs = messageStore.getMessages('u2') as any[]
    expect(msgs).toHaveLength(1)
    expect(msgs[0].clientId).toBe('cid-1')
    expect(msgs[0].sendStatus).toBe('sent')
    expect(msgs[0].optimistic).toBe(false)
    expect(msgs[0].tid).toBe('t-echo')
  })

  it('handles match success (single match) by entering chat and dispatching event', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.isMatching = true
    chatStore.continuousMatchConfig.enabled = false

    const messageStore = useMessageStore()
    const clearSpy = vi.spyOn(messageStore, 'clearHistory')
    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const onMatchSuccess = vi.fn()
    window.addEventListener('match-success', onMatchSuccess as any)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 15,
      sel_userid: 'u3',
      sel_userNikename: 'Bob',
      sel_userSex: '男',
      sel_userAge: '20',
      sel_userAddress: 'BJ'
    })

    expect(clearSpy).toHaveBeenCalledWith('u3')
    expect(chatStore.historyUserIds[0]).toBe('u3')
    expect(chatStore.isMatching).toBe(false)
    expect(chatStore.currentChatUser?.id).toBe('u3')

    expect(onMatchSuccess).toHaveBeenCalled()
    expect((onMatchSuccess.mock.calls[0]?.[0] as CustomEvent).detail.id).toBe('u3')

    window.removeEventListener('match-success', onMatchSuccess as any)
  })

  it('handles match success (continuous match) by setting currentMatchedUser and dispatching auto-check', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.startContinuousMatch(3)
    chatStore.isMatching = true

    const onAutoCheck = vi.fn()
    window.addEventListener('match-auto-check', onAutoCheck as any)
    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 15,
      sel_userid: 'u3',
      sel_userNikename: 'Bob',
      sel_userSex: '男',
      sel_userAge: '20',
      sel_userAddress: 'BJ'
    })

    expect(chatStore.currentMatchedUser?.id).toBe('u3')
    expect(onAutoCheck).toHaveBeenCalledOnce()

    window.removeEventListener('match-auto-check', onAutoCheck as any)
  })

  it('send shows toast when ws is not open', () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const socket = useWebSocket()
    socket.connect()

    // do not trigger open -> CONNECTING, not OPEN
    socket.send({ a: 1 })
    expect(toastShow).toHaveBeenCalledWith('连接已断开，请刷新页面重试')
  })

  it('shows toast on code=12 tip message', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 12, content: '连接成功' })
    expect(toastShow).toHaveBeenCalledWith('连接成功')
  })

  it('silently ignores system messages (code=13/14/16/19/18)', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const messageStore = useMessageStore()
    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 13, content: 'x' })
    await FakeWebSocket.instances[0]!.triggerMessage({ code: 14, content: 'x' })
    await FakeWebSocket.instances[0]!.triggerMessage({ code: 16, content: 'x' })
    await FakeWebSocket.instances[0]!.triggerMessage({ code: 19, content: 'x' })
    await FakeWebSocket.instances[0]!.triggerMessage({ code: 18, content: 'x' })

    expect(toastShow).not.toHaveBeenCalled()
    expect(messageStore.getMessages('me') || []).toHaveLength(0)
  })

  it('dispatches check-online-result event on code=30', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const onResult = vi.fn()
    window.addEventListener('check-online-result', onResult as any)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 30, userId: 'u2', online: true })
    expect(onResult).toHaveBeenCalledOnce()
    expect((onResult.mock.calls[0]?.[0] as CustomEvent).detail.code).toBe(30)

    window.removeEventListener('check-online-result', onResult as any)
  })

  it('marks forceoutFlag on code=-4 reject message', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      await FakeWebSocket.instances[0]!.triggerMessage({ code: -4, forceout: true, content: 'x' })
    } finally {
      errorSpy.mockRestore()
    }

    expect(socket.forceoutFlag.value).toBe(true)
  })

  it('reject (code=-4) uses default error message when content is empty', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      await FakeWebSocket.instances[0]!.triggerMessage({ code: -4, forceout: true })
    } finally {
      errSpy.mockRestore()
    }

    expect(socket.forceoutFlag.value).toBe(true)
  })

  it('forceout (code=-3) uses default error message when content is empty and clears token', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      await FakeWebSocket.instances[0]!.triggerMessage({ code: -3, forceout: true })
    } finally {
      errSpy.mockRestore()
    }

    expect(localStorage.getItem('authToken')).toBeNull()
    expect(socket.forceoutFlag.value).toBe(true)
  })

  it('match message uses default fields and does not auto-enter when matching is cancelled', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.isMatching = false

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 15, sel_userid: 'u3' })
    expect(chatStore.currentChatUser).toBeNull()

    const u3 = chatStore.getUser('u3') as any
    expect(u3.nickname).toBe('匿名用户')
    expect(u3.sex).toBe('未知')
    expect(u3.age).toBe('0')
    expect(u3.address).toBe('未知')
  })

  it('shouldDisplay falls back to nickname match and supports messageContent from msg field', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/chat/u2')
    await router.isReady()

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u2',
      name: 'Peer',
      nickname: 'SameNick',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: '',
      lastTime: '',
      unreadCount: 0
    } as any)
    chatStore.enterChat(chatStore.getUser('u2') as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'uX', name: 'SameNick', sex: '未知', ip: '', time: '2026-01-01 00:00:00.000', tid: 't-nick' },
      touser: { id: 'me', name: 'Me', sex: '未知', ip: '' },
      msg: 'hello',
      tid: 't-nick'
    })

    const messageStore = useMessageStore()
    expect(messageStore.getMessages('u2').some((m: any) => m.tid === 't-nick')).toBe(true)
  })

  it('non-json raw message shows toast when no current chat user', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    const ret = FakeWebSocket.instances[0]!.onmessage?.({ data: 'raw<br>msg' })
    if (ret && typeof (ret as Promise<any>).then === 'function') await ret

    expect(toastShow).toHaveBeenCalledWith('raw msg')
  })

  it('fallbackContent supports msg field and coerces non-finite code to 0', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.currentChatUser = { id: 'u2', name: '', nickname: '', sex: '', age: '0', area: '', address: '', ip: '', isFavorite: false, lastMsg: '', lastTime: '', unreadCount: 0 } as any

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 'oops', msg: 'sys' })
    const msg = useMessageStore().getMessages('u2')[0] as any
    expect(msg.code).toBe(0)
    expect(msg.content).toBe('sys')
  })

  it('creates new user and increments unread when receiving message on non-chat route', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/list')
    await router.isReady()

    const chatStore = useChatStore()
    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u_new', name: 'New', nickname: 'New', sex: '未知', ip: '', content: 'hello', time: '2026-01-01 00:00:00.000', tid: 't-new' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 't-new'
    })

    const user = chatStore.getUser('u_new') as any
    expect(user).toBeTruthy()
    expect(user.unreadCount).toBe(1)

    const messageStore = useMessageStore()
    expect(messageStore.getMessages('u_new')).toHaveLength(1)
  })

  it('skips duplicate ws message when tid already exists', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/list')
    await router.isReady()

    const chatStore = useChatStore()
    chatStore.upsertUser({ id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', age: '0', area: '', address: '', ip: '', isFavorite: false, lastMsg: '', lastTime: '', unreadCount: 0 } as any)

    const messageStore = useMessageStore()
    messageStore.addMessage('u2', {
      code: 7,
      fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      type: 'text',
      content: 'hello',
      time: '2026-01-01 00:00:00.000',
      tid: 'tdup',
      isSelf: false,
      isImage: false,
      isVideo: false,
      isFile: false,
      imageUrl: '',
      videoUrl: '',
      fileUrl: '',
      segments: []
    } as any)

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '', content: 'hello', time: '2026-01-01 00:00:00.000', tid: 'tdup' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 'tdup'
    })

    expect(messageStore.getMessages('u2')).toHaveLength(1)
  })

  it('parses image/video/file segments and sets message types accordingly', async () => {
    const authStore = useAuthStore()
    const userStore = useUserStore()
    authStore.isAuthenticated = true
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    await router.push('/chat/u2')
    await router.isReady()

    const chatStore = useChatStore()
    chatStore.upsertUser({ id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', age: '0', area: '', address: '', ip: '', isFavorite: false, lastMsg: '', lastTime: '', unreadCount: 0 } as any)
    chatStore.enterChat(chatStore.getUser('u2') as any)

    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local'
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '', content: '[2026/01/a.png]', time: '2026-01-01 00:00:00.000', tid: 't-img' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 't-img'
    })
    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '', content: '[2026/01/a.mp4]', time: '2026-01-01 00:00:00.100', tid: 't-vid' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 't-vid'
    })
    await FakeWebSocket.instances[0]!.triggerMessage({
      code: 7,
      fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '', content: '[2026/01/a.txt]', time: '2026-01-01 00:00:00.200', tid: 't-file' },
      touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
      tid: 't-file'
    })

    const messages = useMessageStore().getMessages('u2') as any[]
    expect(messages.some(m => m.tid === 't-img' && m.type === 'image')).toBe(true)
    expect(messages.some(m => m.tid === 't-vid' && m.type === 'video')).toBe(true)
    expect(messages.some(m => m.tid === 't-file' && m.type === 'file')).toBe(true)
  })

  it('handles unknown code by showing toast when no currentChatUser', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 999, content: 'a<br>b' })
    expect(toastShow).toHaveBeenCalledWith('a b')
  })

  it('handles unknown code by inserting system message into current chat', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.currentChatUser = { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', age: '0', area: '', address: '', ip: '', isFavorite: false, lastMsg: '', lastTime: '', unreadCount: 0 } as any

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 999, content: 'sys-msg' })

    const messageStore = useMessageStore()
    expect(messageStore.getMessages('u2')[0]?.content).toBe('sys-msg')
  })

  it('falls back to raw text when message payload is not JSON', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.currentChatUser = { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', age: '0', area: '', address: '', ip: '', isFavorite: false, lastMsg: '', lastTime: '', unreadCount: 0 } as any

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    const ret = FakeWebSocket.instances[0]!.onmessage?.({ data: 'raw-msg' })
    if (ret && typeof (ret as Promise<any>).then === 'function') await ret

    const messageStore = useMessageStore()
    expect(messageStore.getMessages('u2')[0]?.content).toBe('raw-msg')
  })

  it('onclose cancels continuous match and does not reconnect when forceoutFlag is set', async () => {
    vi.useFakeTimers()
    try {
      const userStore = useUserStore()
      userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
      localStorage.setItem('authToken', 't-1')

      const chatStore = useChatStore()
      chatStore.startContinuousMatch(2)

      const mediaStore = useMediaStore()
      vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
      vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

      const socket = useWebSocket()
      socket.connect()
      await FakeWebSocket.instances[0]!.triggerOpen()

      socket.forceoutFlag.value = true
      FakeWebSocket.instances[0]!.close()

      expect(chatStore.continuousMatchConfig.enabled).toBe(false)
      expect(toastShow).toHaveBeenCalledWith('连接断开，连续匹配已取消')

      await vi.advanceTimersByTimeAsync(3000)
      expect(FakeWebSocket.instances).toHaveLength(1)
    } finally {
      vi.useRealTimers()
    }
  })

  it('code=12 without content does not toast', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    await FakeWebSocket.instances[0]!.triggerMessage({ code: 12 })
    expect(toastShow).not.toHaveBeenCalled()
  })

  it('typing status scrolls even when no scroll callback is registered (no-op branch)', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()
    chatStore.currentChatUser = {
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
    } as any

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    const messageStore = useMessageStore()
    await FakeWebSocket.instances[0]!.triggerMessage({ act: 'inputStatusOn_u2_x' })
    expect(messageStore.isTyping).toBe(true)
  })

  it('checkUserOnlineStatus returns early when currentUser is missing and sends when present', async () => {
    const socket = useWebSocket()

    const userStore = useUserStore()
    userStore.currentUser = null as any
    socket.checkUserOnlineStatus('u2')

    // Now connect and verify it sends.
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    socket.checkUserOnlineStatus('u2')
    const sent = FakeWebSocket.instances[0]!.sent.join('\n')
    expect(sent).toContain('ShowUserLoginInfo')
    expect(sent).toContain('"msg":"u2"')
  })

  it('socket.onerror updates wsConnected only for active connection', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const chatStore = useChatStore()

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    expect(chatStore.wsConnected).toBe(true)

    FakeWebSocket.instances[0]!.onerror?.({})
    expect(chatStore.wsConnected).toBe(false)

    // Stale handler should no-op (branch: activeConnection !== connection).
    socket.disconnect(true)
    FakeWebSocket.instances[0]!.onerror?.({})
    expect(chatStore.wsConnected).toBe(false)
  })

  it('covers message payload fallbacks (content/time/tid/id/nickname) and targetUserId empty path', async () => {
    vi.useFakeTimers()
    try {
      const authStore = useAuthStore()
      const userStore = useUserStore()
      authStore.isAuthenticated = true
      userStore.currentUser = { id: 'me', name: 'Me', nickname: '' } as any
      localStorage.setItem('authToken', 't-1')

      // Not in chat page -> shouldDisplay=false for all messages.
      await router.push('/list')
      await router.isReady()

      const chatStore = useChatStore()
      chatStore.currentChatUser = null as any

      const mediaStore = useMediaStore()
      mediaStore.imgServer = 'img.local' as any
      vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
      vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

      const socket = useWebSocket()
      socket.connect()
      await FakeWebSocket.instances[0]!.triggerOpen()

      // content: fromuser.content
      await FakeWebSocket.instances[0]!.triggerMessage({
        code: 7,
        fromuser: { id: 'u2', name: 'FromName', sex: '未知', ip: '', content: 'c1', time: 't1', tid: 'tid1' },
        touser: { id: 'me', name: 'ToName', sex: '未知', ip: '' }
      })

      // content: data.content, time: fromuser.Time, tid: data.Tid
      await FakeWebSocket.instances[0]!.triggerMessage({
        code: 7,
        fromuser: { id: 'u2', name: 'FromName', sex: '未知', ip: '', Time: 't2' },
        touser: { id: 'me', name: 'ToName', sex: '未知', ip: '' },
        Tid: 'tid2',
        content: 'c2'
      })

      // content: data.msg, time: data.Time, tid: fromuser.Tid
      await FakeWebSocket.instances[0]!.triggerMessage({
        code: 7,
        fromuser: { id: 'u2', name: 'FromName', sex: '未知', ip: '', Tid: 'tid3' },
        touser: { id: 'me', name: 'ToName', sex: '未知', ip: '' },
        Time: 't3',
        msg: 'c3'
      })

      // content: fallback '', id fallback '' -> targetUserId empty -> should not add history.
      await FakeWebSocket.instances[0]!.triggerMessage({
        code: 7,
        fromuser: { name: 'FromName', sex: '未知', ip: '' },
        touser: { name: 'ToName', sex: '未知', ip: '' }
      })

      // cover isSelf=true with shouldDisplay=false (else-if !isSelf branch false)
      await FakeWebSocket.instances[0]!.triggerMessage({
        code: 7,
        fromuser: { id: md5Hex('me'), name: 'Me', nickname: 'Me', sex: '未知', ip: '', content: 'echo', time: '', tid: '' },
        touser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' }
      })

      // allow any deferred scroll timers to flush
      await vi.runOnlyPendingTimersAsync()

      const messageStore = useMessageStore()
      expect(messageStore.getMessages('u2').length).toBeGreaterThan(0)
    } finally {
      vi.useRealTimers()
    }
  })

  it('uses default toast text when fallback content becomes blank after stripping tags', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    // Unknown code but content only contains tags -> should toast default.
    await FakeWebSocket.instances[0]!.triggerMessage({ code: 999, content: '<br>' })
    expect(toastShow).toHaveBeenCalledWith('系统消息')

    // Invalid JSON raw message with only tags -> should also toast default.
    const ret = FakeWebSocket.instances[0]!.onmessage?.({ data: '<br>' })
    if (ret && typeof (ret as Promise<any>).then === 'function') await ret
    expect(toastShow).toHaveBeenCalledWith('系统消息')
  })

  it('raw empty message returns early in catch branch', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()

    const ret = FakeWebSocket.instances[0]!.onmessage?.({ data: '' })
    if (ret && typeof (ret as Promise<any>).then === 'function') await ret
  })
})
