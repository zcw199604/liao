import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useThemeStore } from '@/stores/theme'

const STORAGE_KEY = 'liao-theme-preference'

describe('stores/theme', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    document.documentElement.classList.remove('dark')
    window.localStorage.clear()
    vi.restoreAllMocks()
  })

  it('init(): default preference is dark and applies dark class', () => {
    const theme = useThemeStore()
    theme.init()

    expect(theme.preference).toBe('dark')
    expect(theme.resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('setPreference(light): persists and removes dark class', () => {
    const theme = useThemeStore()
    theme.init()

    theme.setPreference('light')

    expect(window.localStorage.getItem(STORAGE_KEY)).toBe('light')
    expect(theme.preference).toBe('light')
    expect(theme.resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('auto: follows matchMedia changes and updates DOM class', () => {
    window.localStorage.setItem(STORAGE_KEY, 'auto')

    const listeners: Array<(event: any) => void> = []
    const mql: any = {
      matches: true,
      media: '(prefers-color-scheme: dark)',
      onchange: null,
      addEventListener: (_type: string, listener: (event: any) => void) => {
        listeners.push(listener)
      },
      removeEventListener: (_type: string, listener: (event: any) => void) => {
        const index = listeners.indexOf(listener)
        if (index >= 0) listeners.splice(index, 1)
      },
      addListener: (listener: (event: any) => void) => {
        listeners.push(listener)
      },
      removeListener: (listener: (event: any) => void) => {
        const index = listeners.indexOf(listener)
        if (index >= 0) listeners.splice(index, 1)
      },
      dispatchEvent: vi.fn()
    }

    vi.spyOn(window, 'matchMedia').mockReturnValue(mql)

    const theme = useThemeStore()
    theme.init()
    expect(theme.preference).toBe('auto')
    expect(theme.resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    mql.matches = false
    listeners.forEach((listener) => listener({ matches: false }))
    expect(theme.resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('system changes do not toggle DOM class when not in auto', () => {
    const listeners: Array<(event: any) => void> = []
    const mql: any = {
      matches: true,
      media: '(prefers-color-scheme: dark)',
      onchange: null,
      addEventListener: (_type: string, listener: (event: any) => void) => {
        listeners.push(listener)
      },
      removeEventListener: (_type: string, listener: (event: any) => void) => {
        const index = listeners.indexOf(listener)
        if (index >= 0) listeners.splice(index, 1)
      },
      addListener: (listener: (event: any) => void) => {
        listeners.push(listener)
      },
      removeListener: (listener: (event: any) => void) => {
        const index = listeners.indexOf(listener)
        if (index >= 0) listeners.splice(index, 1)
      },
      dispatchEvent: vi.fn()
    }

    vi.spyOn(window, 'matchMedia').mockReturnValue(mql)

    const theme = useThemeStore()
    theme.init()
    expect(theme.resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    theme.setPreference('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)

    mql.matches = true
    listeners.forEach((listener) => listener({ matches: true }))
    expect(theme.preference).toBe('light')
    expect(theme.resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })
})
