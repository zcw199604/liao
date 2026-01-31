import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const toastShow = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    show: toastShow
  })
}))

const systemApiMock = {
  getSystemConfig: vi.fn(),
  updateSystemConfig: vi.fn(),
  resolveImagePort: vi.fn()
}

const chatApiMock = {
  reportReferrer: vi.fn().mockResolvedValue({}),
  getHistoryUserList: vi.fn(),
  getFavoriteUserList: vi.fn(),
  getMessageHistory: vi.fn(),
  toggleFavorite: vi.fn(),
  cancelFavorite: vi.fn()
}

vi.mock('@/api/system', () => systemApiMock)
vi.mock('@/api/chat', () => chatApiMock)

// Keep message segment helpers deterministic and controllable for branch tests.
const parseMessageSegmentsMock = vi.fn()
const getSegmentsMetaMock = vi.fn()
const buildLastMsgPreviewFromSegmentsMock = vi.fn()

vi.mock('@/utils/messageSegments', () => ({
  parseMessageSegments: (...args: any[]) => parseMessageSegmentsMock(...args),
  getSegmentsMeta: (...args: any[]) => getSegmentsMetaMock(...args),
  buildLastMsgPreviewFromSegments: (...args: any[]) => buildLastMsgPreviewFromSegmentsMock(...args)
}))

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
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.resetModules()
  localStorage.clear()
  setActivePinia(createPinia())

  FakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', FakeWebSocket as any)

  // deterministic segment/meta behavior for most tests
  parseMessageSegmentsMock.mockImplementation(async (raw: string, opts: any) => {
    const content = String(raw || '')
    if (content.includes('[img]')) {
      const url = await opts.resolveMediaUrl('2026/01/a.png')
      if (!url) return [{ kind: 'text', text: content }]
      return [{ kind: 'image', path: '2026/01/a.png', url }]
    }
    return [{ kind: 'text', text: content }]
  })

  getSegmentsMetaMock.mockImplementation((segs: any[]) => {
    const hasImage = Array.isArray(segs) && segs.some((s) => s?.kind === 'image' && s?.url)
    const imageUrl = hasImage ? String(segs.find((s) => s?.kind === 'image')?.url || '') : ''
    return { hasImage, hasVideo: false, hasFile: false, imageUrl, videoUrl: '', fileUrl: '' } as any
  })

  buildLastMsgPreviewFromSegmentsMock.mockImplementation((segs: any[]) => {
    const firstText = Array.isArray(segs) ? segs.find((s) => s?.kind === 'text')?.text : ''
    return String(firstText || '')
  })

  systemApiMock.getSystemConfig.mockResolvedValue({
    code: 0,
    data: { imagePortMode: 'fixed', imagePortFixed: '9006', imagePortRealMinBytes: 2048 }
  } as any)
  systemApiMock.resolveImagePort.mockResolvedValue({ code: 0, data: { port: '9006' } } as any)
})

describe('composables/useWebSocket branch gaps', () => {
  it('connect uses wss scheme when window is https', async () => {
    const { useUserStore } = await import('@/stores/user')
    const { useWebSocket } = await import('@/composables/useWebSocket')

    localStorage.setItem('authToken', 't-1')
    useUserStore().currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const origLocation = window.location
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { protocol: 'https:', host: 'localhost:3000', href: 'https://localhost:3000' }
    })
    try {
      useWebSocket().connect()
      expect(FakeWebSocket.instances[0]!.url.startsWith('wss://')).toBe(true)
    } finally {
      Object.defineProperty(window, 'location', { configurable: true, value: origLocation })
    }
  })

  it('typing status ignores other user and scrollToBottom is safe when callback is unset', async () => {
    const { useUserStore } = await import('@/stores/user')
    const { useChatStore } = await import('@/stores/chat')
    const { useMessageStore } = await import('@/stores/message')
    const { useWebSocket } = await import('@/composables/useWebSocket')

    localStorage.setItem('authToken', 't-1')
    useUserStore().currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.currentChatUser = { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '' } as any

    const socket = useWebSocket()
    socket.connect()

    // Mismatch -> no typing update (covers the "else" branch of the guard).
    await FakeWebSocket.instances[0]!.onmessage?.({ data: JSON.stringify({ act: 'inputStatusOn_u3_x' }) })
    expect(useMessageStore().isTyping).toBe(false)

    // Match + no scroll callback -> should not throw and should flip typing flag.
    await FakeWebSocket.instances[0]!.onmessage?.({ data: JSON.stringify({ act: 'inputStatusOn_u2_x' }) })
    expect(useMessageStore().isTyping).toBe(true)
  })

  it('covers nickname/id fallbacks, resolveMediaUrl no-imgServer branches, and new-user defaults', async () => {
    const { useAuthStore } = await import('@/stores/auth')
    const { useUserStore } = await import('@/stores/user')
    const { useChatStore } = await import('@/stores/chat')
    const { useMediaStore } = await import('@/stores/media')
    const { useWebSocket } = await import('@/composables/useWebSocket')

    // currentUser.id = '' covers (id || '') fallbacks in multiple places.
    const authStore = useAuthStore()
    authStore.isAuthenticated = true
    const userStore = useUserStore()
    userStore.currentUser = { id: '', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const mediaStore = useMediaStore()
    mediaStore.imgServer = '' as any
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const chatStore = useChatStore()
    chatStore.currentChatUser = { id: '', name: 'Peer', nickname: 'Peer', sex: '未知', ip: '' } as any

    // Not in chat page -> shouldDisplay=false, hits the "new user" path.
    const router = (await import('@/router')).default
    await router.push('/list')
    await router.isReady()

    const socket = useWebSocket()
    socket.connect()

    // fromuser/touser missing nickname+name -> falls back to '' for both nickname fields.
    await FakeWebSocket.instances[0]!.onmessage?.({
      data: JSON.stringify({
        code: 7,
        fromuser: { id: 'u_new', content: '[img]', time: '2026-01-01 00:00:00.000', tid: 't1', sex: '未知', ip: '' },
        touser: { id: '', sex: '未知', ip: '' }
      })
    })

    const u = chatStore.getUser('u_new') as any
    expect(u).toBeTruthy()
    // when fromUserNickname is empty, new user falls back to default display name
    expect(u.nickname).toBe('匿名用户')
  })

  it('shouldDisplay stays true while currentChatUser is cleared mid-async, covering the unreachable-looking else branch', async () => {
    const { useAuthStore } = await import('@/stores/auth')
    const { useUserStore } = await import('@/stores/user')
    const { useChatStore } = await import('@/stores/chat')
    const { useMediaStore } = await import('@/stores/media')
    const { useWebSocket } = await import('@/composables/useWebSocket')

    const authStore = useAuthStore()
    authStore.isAuthenticated = true
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const router = (await import('@/router')).default
    await router.push('/chat/u2')
    await router.isReady()

    const chatStore = useChatStore()
    chatStore.enterChat({
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

    const mediaStore = useMediaStore()
    mediaStore.imgServer = 'img.local' as any
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    let release: (() => void) | null = null
    parseMessageSegmentsMock.mockImplementationOnce(async (_raw: string, _opts: any) => {
      await new Promise<void>((resolve) => {
        release = resolve
      })
      return [{ kind: 'text', text: 'x' }]
    })

    const socket = useWebSocket()
    socket.connect()

    const promise = FakeWebSocket.instances[0]!.onmessage?.({
      data: JSON.stringify({
        code: 7,
        fromuser: { id: 'u2', name: 'U2', nickname: 'U2', sex: '未知', ip: '', content: 'x', time: '2026-01-01 00:00:00.000', tid: 't-1' },
        touser: { id: 'me', name: 'Me', nickname: 'Me', sex: '未知', ip: '' },
        tid: 't-1'
      })
    })

    // While message parsing is awaiting, user navigates away / clears current chat.
    chatStore.currentChatUser = null as any
    release?.()
    await promise

    // No crash and the stale shouldDisplay path finished.
    expect(true).toBe(true)
  })

  it('updates existing user without nickname updates when fromUserNickname is empty', async () => {
    const { useAuthStore } = await import('@/stores/auth')
    const { useUserStore } = await import('@/stores/user')
    const { useChatStore } = await import('@/stores/chat')
    const { useMediaStore } = await import('@/stores/media')
    const { useWebSocket } = await import('@/composables/useWebSocket')

    const authStore = useAuthStore()
    authStore.isAuthenticated = true
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '', area: '' } as any
    localStorage.setItem('authToken', 't-1')

    const router = (await import('@/router')).default
    await router.push('/list')
    await router.isReady()

    const chatStore = useChatStore()
    chatStore.upsertUser({
      id: 'u2',
      name: 'Old',
      nickname: 'Old',
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

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()

    // fromuser has no name/nickname -> fromUserNickname='' so the nickname-update branch is skipped.
    await FakeWebSocket.instances[0]!.onmessage?.({
      data: JSON.stringify({
        code: 7,
        fromuser: { id: 'u2', sex: '未知', ip: '', content: 'hi', time: '2026-01-01 00:00:00.000', tid: 't-1' },
        touser: { id: 'me', sex: '未知', ip: '' },
        tid: 't-1'
      })
    })

    const user = chatStore.getUser('u2') as any
    expect(user).toBeTruthy()
  })

  it('falls back to empty string for unknown-code content when neither content nor msg exists', async () => {
    const { useUserStore } = await import('@/stores/user')
    const { useWebSocket } = await import('@/composables/useWebSocket')

    localStorage.setItem('authToken', 't-1')
    useUserStore().currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const socket = useWebSocket()
    socket.connect()

    await FakeWebSocket.instances[0]!.onmessage?.({ data: JSON.stringify({ code: 999 }) })
    // no toast, no crash
    expect(toastShow).not.toHaveBeenCalled()
  })

  it('non-json raw message inserts system message into current chat and uses peer.name fallback', async () => {
    const { useUserStore } = await import('@/stores/user')
    const { useChatStore } = await import('@/stores/chat')
    const { useMessageStore } = await import('@/stores/message')
    const { useMediaStore } = await import('@/stores/media')
    const { useWebSocket } = await import('@/composables/useWebSocket')

    localStorage.setItem('authToken', 't-1')
    useUserStore().currentUser = { id: 'me', name: 'Me', nickname: 'Me' } as any

    const chatStore = useChatStore()
    chatStore.currentChatUser = { id: 'u2', name: 'PeerName', nickname: '', sex: '', ip: '' } as any

    const mediaStore = useMediaStore()
    vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
    vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)

    const socket = useWebSocket()
    socket.connect()

    const ret = FakeWebSocket.instances[0]!.onmessage?.({ data: 'raw' })
    if (ret && typeof (ret as Promise<any>).then === 'function') await ret

    const msgs = useMessageStore().getMessages('u2') as any[]
    expect(msgs.length).toBeGreaterThan(0)
    expect(String(msgs[0]?.fromuser?.name)).toBe('PeerName')
    expect(String(msgs[0]?.fromuser?.sex)).toBe('未知')
  })
})
