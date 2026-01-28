import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export type ThemePreference = 'dark' | 'light' | 'auto'
export type ResolvedTheme = 'dark' | 'light'

const STORAGE_KEY = 'liao-theme-preference'
const MEDIA_QUERY = '(prefers-color-scheme: dark)'

const readPreference = (): ThemePreference => {
  if (typeof window === 'undefined') return 'dark'
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw === 'dark' || raw === 'light' || raw === 'auto') return raw
  } catch {
    // ignore
  }
  return 'dark'
}

const writePreference = (value: ThemePreference) => {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, value)
  } catch {
    // ignore
  }
}

const applyThemeToDom = (theme: ResolvedTheme) => {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('dark', theme === 'dark')
}

export const useThemeStore = defineStore('theme', () => {
  const initialized = ref(false)
  const preference = ref<ThemePreference>('dark')
  const systemPrefersDark = ref(false)

  let mediaQueryList: MediaQueryList | null = null
  let mediaListener: ((event: MediaQueryListEvent) => void) | null = null

  const resolved = computed<ResolvedTheme>(() => {
    if (preference.value === 'auto') return systemPrefersDark.value ? 'dark' : 'light'
    return preference.value
  })

  const init = () => {
    if (initialized.value) return

    preference.value = readPreference()

    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
      mediaQueryList = window.matchMedia(MEDIA_QUERY)
      systemPrefersDark.value = mediaQueryList.matches

      mediaListener = (event: MediaQueryListEvent) => {
        systemPrefersDark.value = event.matches
        if (preference.value === 'auto') {
          applyThemeToDom(event.matches ? 'dark' : 'light')
        }
      }

      if (typeof mediaQueryList.addEventListener === 'function') {
        mediaQueryList.addEventListener('change', mediaListener)
      } else if (typeof mediaQueryList.addListener === 'function') {
        mediaQueryList.addListener(mediaListener)
      }
    }

    applyThemeToDom(resolved.value)
    initialized.value = true
  }

  const setPreference = (next: ThemePreference) => {
    preference.value = next
    writePreference(next)
    applyThemeToDom(resolved.value)
  }

  return {
    initialized,
    preference,
    resolved,
    init,
    setPreference
  }
})

