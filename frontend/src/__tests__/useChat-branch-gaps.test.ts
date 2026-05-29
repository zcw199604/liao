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
import { useMessageStore } from '@/stores/message'
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

  it('enterGlobalFavoriteChat prepares owner-scoped user and loads history', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'owner-b', name: 'Owner B', nickname: 'Owner B' } as any

    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    const loadSpy = vi.spyOn(messageStore, 'loadHistory').mockResolvedValue(0)

    const user = await useChat().enterGlobalFavoriteChat({
      targetUserId: 'target-c',
      targetUserName: 'Target C'
    })

    expect(user?.id).toBe('target-c')
    expect(chatStore.listOwnerUserId).toBe('owner-b')
    expect(chatStore.getUser('target-c')?.nickname).toBe('Target C')
    expect(chatStore.currentChatUser?.id).toBe('target-c')
    expect(loadSpy).toHaveBeenCalledWith('owner-b', 'target-c', {
      isFirst: true,
      firstTid: '0',
      myUserName: 'Owner B'
    })
  })

  it('enterGlobalFavoriteChat keeps currentChatUser when history loading fails', async () => {
    const userStore = useUserStore()
    userStore.currentUser = { id: 'owner-b', name: 'Owner B', nickname: 'Owner B' } as any

    const chatStore = useChatStore()
    const messageStore = useMessageStore()
    vi.spyOn(messageStore, 'loadHistory').mockRejectedValue(new Error('history failed'))

    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      const user = await useChat().enterGlobalFavoriteChat({
        targetUserId: 'target-c',
        targetUserName: 'Target C'
      })

      expect(user?.id).toBe('target-c')
      expect(chatStore.currentChatUser?.id).toBe('target-c')
      expect(chatStore.getUser('target-c')?.nickname).toBe('Target C')
    } finally {
      warnSpy.mockRestore()
    }
  })
})
