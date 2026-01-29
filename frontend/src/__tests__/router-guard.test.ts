import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'

const routerHarness = vi.hoisted(() => {
  let guard: any
  return {
    setGuard(cb: any) {
      guard = cb
    },
    getGuard() {
      return guard
    }
  }
})

vi.mock('vue-router', () => ({
  createWebHistory: vi.fn(),
  createRouter: vi.fn(() => ({
    beforeEach: (cb: any) => routerHarness.setGuard(cb)
  }))
}))

import '@/router'

const makeUser = () => ({
  id: 'u1',
  name: 'Alice',
  nickname: 'Alice',
  sex: '女',
  color: 'c1',
  created_at: 't1',
  cookie: 'ck-1',
  ip: '127.0.0.1',
  area: 'CN'
})

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('router guard', () => {
  it('redirects to /login when requiresAuth and token is invalid', async () => {
    const guard = routerHarness.getGuard()

    const authStore = useAuthStore()
    authStore.isAuthenticated = false
    vi.spyOn(authStore, 'checkToken').mockResolvedValue(false as any)

    const next = vi.fn()
    await guard({ meta: { requiresAuth: true } } as any, {} as any, next)
    expect(next).toHaveBeenCalledWith('/login')
  })

  it('redirects to /identity when requiresIdentity and no currentUser', async () => {
    const guard = routerHarness.getGuard()

    const authStore = useAuthStore()
    authStore.isAuthenticated = false
    vi.spyOn(authStore, 'checkToken').mockResolvedValue(true as any)

    const userStore = useUserStore()
    expect(userStore.currentUser).toBeNull()

    const next = vi.fn()
    await guard({ meta: { requiresAuth: true, requiresIdentity: true } } as any, {} as any, next)
    expect(next).toHaveBeenCalledWith('/identity')
  })

  it('calls next() when auth and identity requirements are satisfied', async () => {
    const guard = routerHarness.getGuard()

    const authStore = useAuthStore()
    authStore.isAuthenticated = true

    const userStore = useUserStore()
    userStore.setCurrentUser(makeUser())

    const next = vi.fn()
    await guard({ meta: { requiresAuth: true, requiresIdentity: true } } as any, {} as any, next)
    expect(next).toHaveBeenCalledWith()
  })

  it('skips auth/identity checks when route does not require them', async () => {
    const guard = routerHarness.getGuard()

    const authStore = useAuthStore()
    authStore.isAuthenticated = false
    const tokenSpy = vi.spyOn(authStore, 'checkToken').mockResolvedValue(true as any)

    const next = vi.fn()
    await guard({ meta: {} } as any, {} as any, next)
    expect(tokenSpy).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledWith()
  })
})

