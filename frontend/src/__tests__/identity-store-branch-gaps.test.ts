import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/identity', () => ({
  getIdentityList: vi.fn(),
  createIdentity: vi.fn(),
  deleteIdentity: vi.fn(),
  selectIdentity: vi.fn()
}))

describe('stores/identity branch gaps', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.resetModules()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('saveIdentityCookie no-ops on empty input and persists when id+cookie are present', async () => {
    const { useIdentityStore } = await import('@/stores/identity')

    const store = useIdentityStore()

    store.saveIdentityCookie('', 'c1')
    store.saveIdentityCookie('i1', '')
    expect(store.getIdentityCookie('i1')).toBe('')
    expect(localStorage.getItem('identityCookies')).toBeNull()

    store.saveIdentityCookie('i1', 'cookie-1')
    expect(store.getIdentityCookie('i1')).toBe('cookie-1')
    expect(String(localStorage.getItem('identityCookies') || '')).toContain('cookie-1')
  })

  it('loadList does not update identityList when response is not usable', async () => {
    const identityApi = await import('@/api/identity')
    const { useIdentityStore } = await import('@/stores/identity')

    vi.mocked(identityApi.getIdentityList).mockResolvedValue({ code: 1, data: [{ id: 'x' }] } as any)

    const store = useIdentityStore()
    await store.loadList()

    expect(store.identityList).toEqual([])
    expect(store.loading).toBe(false)
  })
})

