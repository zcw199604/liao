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

