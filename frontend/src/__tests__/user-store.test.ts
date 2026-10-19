import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useIdentityStore } from '@/stores/identity'
import { useUserStore } from '@/stores/user'

const makeUser = (overrides?: Partial<any>) => ({
  id: 'u1',
  name: 'Alice',
  nickname: 'Alice',
  sex: '女',
  color: 'c1',
  created_at: 't1',
  cookie: 'ck-1',
  ip: '127.0.0.1',
  area: 'CN',
  ...(overrides || {})
})

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
})

describe('stores/user', () => {
  it('setCurrentUser saves identity cookie when id and cookie exist', () => {
    const identityStore = useIdentityStore()
    const saveSpy = vi.spyOn(identityStore, 'saveIdentityCookie')

    const userStore = useUserStore()
    userStore.setCurrentUser(makeUser())

    expect(userStore.currentUser?.id).toBe('u1')
    expect(saveSpy).toHaveBeenCalledWith('u1', 'ck-1')
  })

  it('setCurrentUser does not save cookie when cookie is missing', () => {
    const identityStore = useIdentityStore()
    const saveSpy = vi.spyOn(identityStore, 'saveIdentityCookie')

    const userStore = useUserStore()
    userStore.setCurrentUser(makeUser({ cookie: '' }))

    expect(saveSpy).not.toHaveBeenCalled()
  })

  it('setCurrentUser does not save cookie when id is missing', () => {
    const identityStore = useIdentityStore()
    const saveSpy = vi.spyOn(identityStore, 'saveIdentityCookie')

    const userStore = useUserStore()
    userStore.setCurrentUser(makeUser({ id: '', cookie: 'ck-1' }))

    expect(saveSpy).not.toHaveBeenCalled()
  })

  it('updateUserInfo only applies when currentUser exists', () => {
    const userStore = useUserStore()
    userStore.updateUserInfo({ nickname: 'B' })
    expect(userStore.currentUser).toBeNull()

    userStore.setCurrentUser(makeUser())
    userStore.updateUserInfo({ nickname: 'B', area: 'US' })
    expect(userStore.currentUser?.nickname).toBe('B')
    expect(userStore.currentUser?.area).toBe('US')
  })

  it('startEdit/saveEdit toggles editMode and applies edits', () => {
    const userStore = useUserStore()
    userStore.startEdit()
    expect(userStore.editMode).toBe(false)

    userStore.setCurrentUser(makeUser())
    userStore.startEdit()
    expect(userStore.editMode).toBe(true)
    expect(userStore.editUserInfo).toMatchObject({ id: 'u1', nickname: 'Alice' })

    userStore.editUserInfo.nickname = 'Edited'
    userStore.saveEdit()

    expect(userStore.editMode).toBe(false)
    expect(userStore.currentUser?.nickname).toBe('Edited')
  })

  it('saveEdit is a no-op when currentUser is missing', () => {
    const userStore = useUserStore()
    userStore.saveEdit()
    expect(userStore.currentUser).toBeNull()
    expect(userStore.editMode).toBe(false)
  })

  it('cancelEdit and clearCurrentUser reset state', () => {
    const userStore = useUserStore()
    userStore.setCurrentUser(makeUser())
    userStore.startEdit()
    expect(userStore.editMode).toBe(true)

    userStore.cancelEdit()
    expect(userStore.editMode).toBe(false)
    expect(userStore.editUserInfo).toEqual({})

    userStore.clearCurrentUser()
    expect(userStore.currentUser).toBeNull()
    expect(userStore.editMode).toBe(false)
    expect(userStore.editUserInfo).toEqual({})
  })
})

describe('stores/user persistence', () => {
  it('persists current user to sessionStorage and restores it in a fresh store', async () => {
    const { CURRENT_USER_STORAGE_KEY } = await import('@/stores/user')
    const userStore = useUserStore()
    userStore.setCurrentUser(makeUser())
    expect(JSON.parse(sessionStorage.getItem(CURRENT_USER_STORAGE_KEY) || '{}').id).toBe('u1')

    userStore.updateUserInfo({ nickname: 'Bob' })
    expect(JSON.parse(sessionStorage.getItem(CURRENT_USER_STORAGE_KEY) || '{}').nickname).toBe('Bob')

    setActivePinia(createPinia())
    expect(useUserStore().currentUser?.nickname).toBe('Bob')
  })

  it('clearCurrentUser removes persisted user and ignores corrupted storage', async () => {
    const { CURRENT_USER_STORAGE_KEY } = await import('@/stores/user')
    const userStore = useUserStore()
    userStore.setCurrentUser(makeUser())
    userStore.clearCurrentUser()
    expect(sessionStorage.getItem(CURRENT_USER_STORAGE_KEY)).toBeNull()

    sessionStorage.setItem(CURRENT_USER_STORAGE_KEY, '{bad json')
    setActivePinia(createPinia())
    expect(useUserStore().currentUser).toBeNull()

    sessionStorage.setItem(CURRENT_USER_STORAGE_KEY, JSON.stringify({ name: 'no-id' }))
    setActivePinia(createPinia())
    expect(useUserStore().currentUser).toBeNull()
  })
})
