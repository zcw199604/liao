import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFavoriteStore } from '@/stores/favorite'
import * as favoriteApi from '@/api/favorite'

vi.mock('@/api/favorite', () => ({
  listAllFavorites: vi.fn(),
  addFavorite: vi.fn(),
  removeFavorite: vi.fn(),
  removeFavoriteById: vi.fn()
}))

describe('stores/favorite', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
  })

  it('loadAllFavorites sets list on success and keeps list on non-success', async () => {
    const store = useFavoriteStore()

    vi.mocked(favoriteApi.listAllFavorites).mockResolvedValueOnce({
      code: 0,
      data: [{ id: 1, identityId: 'me', targetUserId: 'u1', targetUserName: 'U1' }]
    } as any)

    await store.loadAllFavorites()
    expect(store.allFavorites).toHaveLength(1)

    vi.mocked(favoriteApi.listAllFavorites).mockResolvedValueOnce({ code: 1, data: [] } as any)
    await store.loadAllFavorites()
    // code != 0 -> do not overwrite with data
    expect(store.allFavorites).toHaveLength(1)
  })

  it('addFavorite returns true on success (and reloads), false otherwise', async () => {
    const store = useFavoriteStore()

    vi.mocked(favoriteApi.listAllFavorites).mockResolvedValue({ code: 0, data: [] } as any)

    vi.mocked(favoriteApi.addFavorite).mockResolvedValueOnce({ code: 0 } as any)
    const ok = await store.addFavorite('me', 'u1', 'U1')
    expect(ok).toBe(true)
    expect(favoriteApi.listAllFavorites).toHaveBeenCalledTimes(1)

    vi.mocked(favoriteApi.addFavorite).mockResolvedValueOnce({ code: 1 } as any)
    const bad = await store.addFavorite('me', 'u2', 'U2')
    expect(bad).toBe(false)
    expect(favoriteApi.listAllFavorites).toHaveBeenCalledTimes(1)
  })

  it('removeFavorite returns true on success and updates list, false otherwise', async () => {
    const store = useFavoriteStore()
    store.allFavorites = [
      { id: 1, identityId: 'me', targetUserId: 'u1', targetUserName: 'U1' } as any,
      { id: 2, identityId: 'me', targetUserId: 'u2', targetUserName: 'U2' } as any
    ]

    vi.mocked(favoriteApi.removeFavorite).mockResolvedValueOnce({ code: 0 } as any)
    const ok = await store.removeFavorite('me', 'u1')
    expect(ok).toBe(true)
    expect(store.allFavorites.some(f => f.targetUserId === 'u1')).toBe(false)

    vi.mocked(favoriteApi.removeFavorite).mockResolvedValueOnce({ code: 1 } as any)
    const bad = await store.removeFavorite('me', 'u2')
    expect(bad).toBe(false)
  })

  it('removeFavoriteById returns true on success and updates list, false otherwise', async () => {
    const store = useFavoriteStore()
    store.allFavorites = [
      { id: 1, identityId: 'me', targetUserId: 'u1', targetUserName: 'U1' } as any,
      { id: 2, identityId: 'me', targetUserId: 'u2', targetUserName: 'U2' } as any
    ]

    vi.mocked(favoriteApi.removeFavoriteById).mockResolvedValueOnce({ code: 0 } as any)
    const ok = await store.removeFavoriteById(2)
    expect(ok).toBe(true)
    expect(store.allFavorites.some(f => f.id === 2)).toBe(false)

    vi.mocked(favoriteApi.removeFavoriteById).mockResolvedValueOnce({ code: 1 } as any)
    const bad = await store.removeFavoriteById(1)
    expect(bad).toBe(false)
  })
})
