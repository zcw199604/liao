import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const toastShow = vi.fn()
const wsSend = vi.fn().mockReturnValue(true)

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ show: toastShow })
}))

vi.mock('@/composables/useWebSocket', () => ({
  useWebSocket: () => ({ send: wsSend })
}))

import { useChat } from '@/composables/useChat'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
})

describe('composables/useChat branch gaps', () => {
  it('startContinuousMatch proceeds when wsConnected is true (covers else branch)', () => {
    const chatStore = useChatStore()
    const userStore = useUserStore()
    userStore.currentUser = { id: 'me', name: 'Me' } as any
    chatStore.wsConnected = true

    const chat = useChat()
    expect(chat.startContinuousMatch(2)).toBe(true)
    expect(chatStore.continuousMatchConfig.enabled).toBe(true)
  })

  it('enterChatAndStopMatch does not require autoMatchTimer (covers autoMatchTimer guard else branch)', () => {
    const chatStore = useChatStore()
    const chat = useChat()

    chat.enterChatAndStopMatch({ id: 'u1', unreadCount: 0, name: 'U1', nickname: 'U1' } as any)
    expect(chatStore.currentChatUser?.id).toBe('u1')
  })
})

