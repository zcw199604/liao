import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const toastShow = vi.fn()

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ show: toastShow })
}))

vi.mock('@/api/system', () => ({
  getSystemConfig: vi.fn().mockResolvedValue({ code: 0, data: {} }),
  updateSystemConfig: vi.fn(),
  resolveImagePort: vi.fn(),
  getRuntimeConfig: vi.fn().mockResolvedValue({ code: 0, data: {} })
}))

vi.mock('@/api/chat', () => ({
  reportReferrer: vi.fn().mockResolvedValue({}),
  getHistoryUserList: vi.fn(),
  getFavoriteUserList: vi.fn(),
  getMessageHistory: vi.fn(),
  toggleFavorite: vi.fn(),
  cancelFavorite: vi.fn()
}))

import { parseForbiddenSeconds, RECONNECT_DELAYS_MS, useWebSocket } from '@/composables/useWebSocket'
import { useConnectionStatus } from '@/composables/useConnectionStatus'
import { useChatStore } from '@/stores/chat'
import { useMediaStore } from '@/stores/media'
import { useUserStore } from '@/stores/user'
import { effectScope } from 'vue'

class FakeWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3
  static instances: FakeWebSocket[] = []

  readyState = FakeWebSocket.CONNECTING
  sent: string[] = []
  onopen: ((ev?: any) => any) | null = null
  onmessage: ((ev: any) => any) | null = null
  onerror: ((ev?: any) => any) | null = null
  onclose: ((ev?: any) => any) | null = null

  constructor(public readonly url: string) {
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
    await this.onopen?.({})
  }

  async triggerMessage(payload: any) {
    await this.onmessage?.({ data: JSON.stringify(payload) })
  }
}

const setupUser = () => {
  useUserStore().currentUser = { id: 'me', name: 'Me', nickname: 'Me', sex: '男', ip: '1.1.1.1', area: 'CN' } as any
  localStorage.setItem('authToken', 't-1')
  const mediaStore = useMediaStore()
  vi.spyOn(mediaStore, 'loadImgServer').mockResolvedValue(undefined)
  vi.spyOn(mediaStore, 'loadCachedImages').mockResolvedValue(undefined)
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
  try {
    useWebSocket().disconnect(true)
  } catch {
    // ignore
  }
  FakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', FakeWebSocket as any)
})

afterEach(() => {
  try {
    useWebSocket().disconnect(true)
  } catch {
    // ignore
  }
  vi.useRealTimers()
})

describe('useWebSocket reconnect', () => {
  it('parseForbiddenSeconds extracts seconds from backend message', () => {
    expect(parseForbiddenSeconds('由于重复登录，您的连接被暂时禁止，请288秒后再试')).toBe(288)
    expect(parseForbiddenSeconds('请稍后再试')).toBe(0)
    expect(parseForbiddenSeconds(undefined)).toBe(0)
    expect(parseForbiddenSeconds('请0秒后再试')).toBe(0)
  })

  it('backs off exponentially between failed attempts and resets after a successful open', async () => {
    vi.useFakeTimers()
    setupUser()
    const chatStore = useChatStore()
    const ws = useWebSocket()

    ws.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    FakeWebSocket.instances[0]!.close()
    expect(chatStore.wsReconnecting).toBe(true)

    // 第 1 次：1s
    await vi.advanceTimersByTimeAsync(RECONNECT_DELAYS_MS[0]! - 1)
    expect(FakeWebSocket.instances).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(FakeWebSocket.instances).toHaveLength(2)

    // 第 2 次失败：2s
    FakeWebSocket.instances[1]!.close()
    await vi.advanceTimersByTimeAsync(RECONNECT_DELAYS_MS[1]! - 1)
    expect(FakeWebSocket.instances).toHaveLength(2)
    await vi.advanceTimersByTimeAsync(1)
    expect(FakeWebSocket.instances).toHaveLength(3)

    // 成功后计数归零，下次断开重新从 1s 开始
    await FakeWebSocket.instances[2]!.triggerOpen()
    expect(chatStore.wsReconnecting).toBe(false)
    FakeWebSocket.instances[2]!.close()
    await vi.advanceTimersByTimeAsync(RECONNECT_DELAYS_MS[0]!)
    expect(FakeWebSocket.instances).toHaveLength(4)
  })

  it('reconnectNow skips the backoff wait, and is a no-op when connected', async () => {
    vi.useFakeTimers()
    setupUser()
    const ws = useWebSocket()

    ws.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    expect(ws.reconnectNow()).toBe(false)

    FakeWebSocket.instances[0]!.close()
    expect(ws.reconnectNow()).toBe(true)
    expect(FakeWebSocket.instances).toHaveLength(2)

    // 原定的退避定时器已取消，不会再多建一个连接
    await vi.advanceTimersByTimeAsync(RECONNECT_DELAYS_MS[0]!)
    expect(FakeWebSocket.instances).toHaveLength(2)
  })

  it('reconnects immediately on online / visible events, but not after manual disconnect', async () => {
    vi.useFakeTimers()
    setupUser()
    const ws = useWebSocket()

    ws.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    FakeWebSocket.instances[0]!.close()

    window.dispatchEvent(new Event('online'))
    expect(FakeWebSocket.instances).toHaveLength(2)

    FakeWebSocket.instances[1]!.close()
    document.dispatchEvent(new Event('visibilitychange'))
    expect(FakeWebSocket.instances).toHaveLength(3)

    // 页面在后台时不重连
    FakeWebSocket.instances[2]!.close()
    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden')
    document.dispatchEvent(new Event('visibilitychange'))
    expect(FakeWebSocket.instances).toHaveLength(3)
    visibility.mockRestore()

    ws.disconnect(true)
    window.dispatchEvent(new Event('online'))
    await vi.advanceTimersByTimeAsync(60_000)
    expect(FakeWebSocket.instances).toHaveLength(3)
  })

  it('code=-4 with remaining seconds stays on page, counts down and reconnects when allowed', async () => {
    vi.useFakeTimers()
    setupUser()
    const chatStore = useChatStore()
    const ws = useWebSocket()
    const scope = effectScope()
    const status = scope.run(() => useConnectionStatus())!

    ws.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    expect(status.state.value).toBe('online')
    expect(status.text.value).toBe('在线')

    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    await FakeWebSocket.instances[0]!.triggerMessage({
      code: -4,
      forceout: true,
      content: '由于重复登录，您的连接被暂时禁止，请5秒后再试'
    })
    errorSpy.mockRestore()

    expect(toastShow).toHaveBeenCalledWith('由于重复登录，您的连接被暂时禁止，请5秒后再试')
    expect(chatStore.wsForbiddenUntil).toBeGreaterThan(Date.now())
    expect(status.state.value).toBe('forbidden')
    expect(status.text.value).toBe('暂停连接 5s')
    // 禁止期间不能手动重连
    expect(ws.reconnectNow()).toBe(false)

    await vi.advanceTimersByTimeAsync(2000)
    expect(status.text.value).toBe('暂停连接 3s')
    expect(FakeWebSocket.instances).toHaveLength(1)

    await vi.advanceTimersByTimeAsync(4000)
    expect(FakeWebSocket.instances).toHaveLength(2)
    expect(chatStore.wsForbiddenUntil).toBe(0)
    expect(status.state.value).toBe('offline')

    chatStore.wsReconnecting = true
    expect(status.text.value).toBe('重连中…')
    chatStore.wsReconnecting = false
    expect(status.text.value).toBe('离线')
    scope.stop()
  })

  it('manual disconnect during forbidden period cancels the scheduled reconnect', async () => {
    vi.useFakeTimers()
    setupUser()
    const ws = useWebSocket()

    ws.connect()
    await FakeWebSocket.instances[0]!.triggerOpen()
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    await FakeWebSocket.instances[0]!.triggerMessage({ code: -4, forceout: true, content: '请3秒后再试' })
    errorSpy.mockRestore()

    ws.disconnect(true)
    expect(useChatStore().wsForbiddenUntil).toBe(0)
    await vi.advanceTimersByTimeAsync(10_000)
    expect(FakeWebSocket.instances).toHaveLength(1)
  })
})
