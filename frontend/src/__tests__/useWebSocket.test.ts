import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const toastShow = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
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
import { useChatStore } from '@/stores/chat'
import { useMediaStore } from '@/stores/media'
import { useMessageStore } from '@/stores/message'
import { useUserStore } from '@/stores/user'

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

  FakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', FakeWebSocket as any)
})

afterEach(() => {
  // best-effort cleanup for module-scoped ws singletons
  try {
    useWebSocket().disconnect(true)
  } catch {
    // ignore
  }
})

describe('composables/useWebSocket', () => {
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
})
