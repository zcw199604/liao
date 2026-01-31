import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/chat', () => ({
  getHistoryUserList: vi.fn(),
  getFavoriteUserList: vi.fn()
}))

vi.mock('@/utils/cookie', () => ({
  generateCookie: vi.fn().mockReturnValue('cookie')
}))

import { useChatStore } from '@/stores/chat'
import * as chatApi from '@/api/chat'

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
})

describe('stores/chat load list branches', () => {
  it('loadHistoryUsers merges into existing users and filters out self', async () => {
    const store = useChatStore()

    store.upsertUser({
      id: 'u1',
      name: 'Old',
      nickname: 'Old',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: true,
      lastMsg: 'old',
      lastTime: '刚刚',
      unreadCount: 5
    } as any)

    ;(chatApi.getHistoryUserList as any).mockResolvedValueOnce([
      { id: 'me', nickname: 'Me' }, // self should be filtered out
      { id: 'u1', nickname: 'NewNick', lastMsg: 'new', lastTime: 't' },
      { id: 'u2', name: 'U2', sex: '男', lastMsg: 'hi', lastTime: 't2' }
    ])

    await store.loadHistoryUsers('me', 'Me')

    expect(store.historyUserIds).toEqual(['u1', 'u2'])
    expect(store.getUser('u1')?.isFavorite).toBe(true)
    expect(store.getUser('u1')?.unreadCount).toBe(5)
    expect(store.getUser('u1')?.nickname).toBe('NewNick')
    expect(store.getUser('u2')?.nickname).toBe('U2')
  })

  it('loadHistoryUsers clears list on error', async () => {
    const store = useChatStore()
    store.historyUserIds = ['u1']

    ;(chatApi.getHistoryUserList as any).mockRejectedValueOnce(new Error('boom'))
    await store.loadHistoryUsers('me', 'Me')
    expect(store.historyUserIds).toEqual([])
  })

  it('loadHistoryUsers no-ops when backend returns non-array payload (covers else branch)', async () => {
    const store = useChatStore()
    store.historyUserIds = ['u1']

    ;(chatApi.getHistoryUserList as any).mockResolvedValueOnce({ not: 'an array' })
    await store.loadHistoryUsers('me', 'Me')

    // Should not overwrite list when payload shape is unexpected.
    expect(store.historyUserIds).toEqual(['u1'])
  })

  it('loadHistoryUsers uses fallback name/nickname when both are missing and keeps unreadCount via || 0', async () => {
    const store = useChatStore()

    // Seed an existing user with unreadCount=0 to hit (existing.unreadCount || 0) fallback branch.
    store.upsertUser({
      id: 'u1',
      name: 'Old',
      nickname: 'Old',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: 'x',
      lastTime: '刚刚',
      unreadCount: 0
    } as any)

    ;(chatApi.getHistoryUserList as any).mockResolvedValueOnce([
      { id: 'u1', nickname: 'NewNick', lastMsg: 'new', lastTime: 't' },
      { id: 'u3' } // no name/nickname -> normalizeUser fallback to '未知'
    ])

    await store.loadHistoryUsers('me', 'Me')

    expect(store.getUser('u3')?.name).toBe('未知')
    expect(store.getUser('u3')?.nickname).toBe('未知')
    expect(store.getUser('u1')?.unreadCount).toBe(0)
  })

  it('loadFavoriteUsers forces isFavorite=true and updates favorite ids', async () => {
    const store = useChatStore()

    store.upsertUser({
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: 'x',
      lastTime: '刚刚',
      unreadCount: 0
    } as any)

    ;(chatApi.getFavoriteUserList as any).mockResolvedValueOnce([
      { id: 'u1', nickname: 'Alice2' },
      { id: 'u3', nickname: 'Bob' }
    ])

    await store.loadFavoriteUsers('me', 'Me')

    expect(store.favoriteUserIds).toEqual(['u1', 'u3'])
    expect(store.getUser('u1')?.isFavorite).toBe(true)
    expect(store.getUser('u1')?.nickname).toBe('Alice2')
    expect(store.getUser('u3')?.isFavorite).toBe(true)
  })

  it('loadFavoriteUsers no-ops when backend returns non-array payload (covers else branch)', async () => {
    const store = useChatStore()
    store.favoriteUserIds = ['u1']

    ;(chatApi.getFavoriteUserList as any).mockResolvedValueOnce(null)
    await store.loadFavoriteUsers('me', 'Me')
    expect(store.favoriteUserIds).toEqual(['u1'])
  })

  it('incrementMatchCount no-ops when continuousMatch is disabled (covers else branch)', () => {
    const store = useChatStore()
    const before = store.continuousMatchConfig.current
    store.incrementMatchCount()
    expect(store.continuousMatchConfig.current).toBe(before)
  })

  it('getUserByNickname/updateUser/clearAllUsers cover utility branches', () => {
    const store = useChatStore()

    store.upsertUser({
      id: 'u1',
      name: 'Alice',
      nickname: 'Alice',
      sex: '未知',
      age: '0',
      area: '',
      address: '',
      ip: '',
      isFavorite: false,
      lastMsg: 'x',
      lastTime: '刚刚',
      unreadCount: 0
    } as any)

    expect(store.getUserByNickname('Alice')?.id).toBe('u1')
    expect(store.getUserByNickname('Missing')).toBeUndefined()

    store.updateUser('u1', { lastMsg: 'updated' } as any)
    expect(store.getUser('u1')?.lastMsg).toBe('updated')
    store.updateUser('nope', { lastMsg: 'x' } as any)

    store.enterChat(store.getUser('u1') as any)
    store.clearAllUsers()
    expect(store.historyUserIds).toEqual([])
    expect(store.favoriteUserIds).toEqual([])
    expect(store.currentChatUser).toBeNull()
  })
})
