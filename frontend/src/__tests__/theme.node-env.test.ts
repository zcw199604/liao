// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useThemeStore } from '@/stores/theme'

describe('stores/theme (node env)', () => {
  it('init/setPreference are safe without window/document (covers guard branches)', () => {
    setActivePinia(createPinia())
    const store = useThemeStore()

    store.init()
    expect(store.preference).toBe('dark')
    expect(store.resolved).toBe('dark')

    store.setPreference('light')
    expect(store.preference).toBe('light')
    expect(store.resolved).toBe('light')
  })
})

