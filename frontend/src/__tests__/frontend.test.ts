import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/auth', () => ({
  login: vi.fn(),
  verifyToken: vi.fn()
}))

import request, { createFormData, navigation } from '@/api/request'
import * as authApi from '@/api/auth'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import { parseEmoji, escapeRegex, randomString, truncate } from '@/utils/string'
import { formatFullTime, formatTime } from '@/utils/time'

beforeEach(() => {
  localStorage.clear()
})

describe('utils/string', () => {
  it('escapeRegex escapes special characters', () => {
    expect(escapeRegex('a+b*c?')).toBe('a\\+b\\*c\\?')
  })

  it('parseEmoji replaces all emoji keys and escapes regex keys', () => {
    const emojiMap = {
      ':)': '/emoji/smile.png',
      '[ok]': '/emoji/ok.png'
    }

    const html = parseEmoji('hi :) [ok] :)', emojiMap)

    expect(html).toContain('src="/emoji/smile.png"')
    expect(html).toContain('alt=":)"')
    expect(html).toContain('src="/emoji/ok.png"')
    expect(html).toContain('alt="[ok]"')
    expect(html.match(/src="\/emoji\/smile\.png"/g)?.length).toBe(2)
  })

  it('truncate keeps short text and truncates long text', () => {
    expect(truncate('hello', 5)).toBe('hello')
    expect(truncate('hello world', 5)).toBe('hello...')
  })

  it('randomString returns the expected length and charset', () => {
    expect(randomString(0)).toBe('')
    const value = randomString(16)
    expect(value).toHaveLength(16)
    expect(value).toMatch(/^[A-Za-z0-9]+$/)
  })
})

describe('utils/time', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('formatTime returns HH:mm for today', () => {
    vi.setSystemTime(new Date(2026, 0, 4, 12, 0, 0))
    const timeStr = new Date(2026, 0, 4, 9, 5, 0).toISOString()
    expect(formatTime(timeStr)).toBe('09:05')
  })

  it('formatTime returns 昨天 for yesterday', () => {
    vi.setSystemTime(new Date(2026, 0, 4, 12, 0, 0))
    const timeStr = new Date(2026, 0, 3, 9, 5, 0).toISOString()
    expect(formatTime(timeStr)).toBe('昨天')
  })

  it('formatTime returns MM/DD for this year (excluding today/yesterday)', () => {
    vi.setSystemTime(new Date(2026, 0, 4, 12, 0, 0))
    const timeStr = new Date(2026, 0, 1, 9, 5, 0).toISOString()
    expect(formatTime(timeStr)).toBe('01/01')
  })

  it('formatTime returns YY/MM/DD for previous years', () => {
    vi.setSystemTime(new Date(2026, 0, 4, 12, 0, 0))
    const timeStr = new Date(2025, 11, 31, 23, 59, 59).toISOString()
    expect(formatTime(timeStr)).toBe('25/12/31')
  })

  it('formatFullTime returns YYYY-MM-DD HH:mm:ss', () => {
    const timeStr = new Date(2026, 0, 4, 9, 5, 6).toISOString()
    expect(formatFullTime(timeStr)).toBe('2026-01-04 09:05:06')
  })
})

describe('composables/useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    useToast().hide()
    vi.useRealTimers()
  })

  it('show displays toast then hides after duration', () => {
    const toast = useToast()
    toast.show('hello', 2000)
    expect(toast.message.value).toBe('hello')
    expect(toast.visible.value).toBe(true)

    vi.advanceTimersByTime(1999)
    expect(toast.visible.value).toBe(true)

    vi.advanceTimersByTime(1)
    expect(toast.visible.value).toBe(false)
  })

  it('error uses 3000ms and success uses 2000ms', () => {
    const toast = useToast()

    toast.error('bad')
    expect(toast.visible.value).toBe(true)
    vi.advanceTimersByTime(2999)
    expect(toast.visible.value).toBe(true)
    vi.advanceTimersByTime(1)
    expect(toast.visible.value).toBe(false)

    toast.success('ok')
    expect(toast.visible.value).toBe(true)
    vi.advanceTimersByTime(1999)
    expect(toast.visible.value).toBe(true)
    vi.advanceTimersByTime(1)
    expect(toast.visible.value).toBe(false)
  })
})

describe('api/request', () => {
  it('createFormData skips null/undefined and stringifies values', () => {
    const params = createFormData({
      a: 1,
      b: null,
      c: undefined,
      d: 'x',
      e: false
    })
    expect(params.toString()).toBe('a=1&d=x&e=false')
  })

  it('request interceptor attaches Authorization when token exists', () => {
    const fulfilled = (request as any).interceptors.request.handlers[0].fulfilled as (cfg: any) => any

    localStorage.setItem('authToken', 't-123')
    const cfgWithToken = fulfilled({ headers: {} })
    expect(cfgWithToken.headers.Authorization).toBe('Bearer t-123')

    localStorage.removeItem('authToken')
    const cfgWithoutToken = fulfilled({ headers: {} })
    expect(cfgWithoutToken.headers.Authorization).toBeUndefined()
  })

  it('response interceptor parses JSON string payloads', () => {
    const fulfilled = (request as any).interceptors.response.handlers[0].fulfilled as (resp: any) => any

    expect(fulfilled({ data: ' {\"a\": 1} ' })).toEqual({ a: 1 })
    expect(fulfilled({ data: 'not-json' })).toBe('not-json')
    expect(fulfilled({ data: '{ invalid }' })).toBe('{ invalid }')
  })

  it('response interceptor clears token and redirects on 401', async () => {
    const rejected = (request as any).interceptors.response.handlers[0].rejected as (err: any) => Promise<any>

    localStorage.setItem('authToken', 't-401')
    const toLoginSpy = vi.spyOn(navigation, 'toLogin').mockImplementation(() => {})

    await rejected({ response: { status: 401 } }).catch(() => {})

    expect(localStorage.getItem('authToken')).toBeNull()
    expect(toLoginSpy).toHaveBeenCalledOnce()
  })
})

describe('stores/auth', () => {
  it('login stores token on success', async () => {
    setActivePinia(createPinia())

    const mockedLogin = vi.mocked(authApi.login)
    mockedLogin.mockResolvedValue({ code: 0, token: 't-login' } as any)

    const store = useAuthStore()
    const ok = await store.login('code')

    expect(ok).toBe(true)
    expect(store.token).toBe('t-login')
    expect(localStorage.getItem('authToken')).toBe('t-login')
    expect(store.isAuthenticated).toBe(true)
    expect(store.loginLoading).toBe(false)
  })

  it('login returns false on failure and resets loading', async () => {
    setActivePinia(createPinia())

    const mockedLogin = vi.mocked(authApi.login)
    mockedLogin.mockResolvedValue({ code: 1 } as any)

    const store = useAuthStore()
    const ok = await store.login('code')

    expect(ok).toBe(false)
    expect(store.token).toBe('')
    expect(store.isAuthenticated).toBe(false)
    expect(store.loginLoading).toBe(false)
  })

  it('checkToken returns false when token is empty', async () => {
    setActivePinia(createPinia())

    const store = useAuthStore()
    const ok = await store.checkToken()

    expect(ok).toBe(false)
    expect(store.isAuthenticated).toBe(false)
  })

  it('checkToken validates token via API and logs out on failure', async () => {
    setActivePinia(createPinia())

    const mockedVerify = vi.mocked(authApi.verifyToken)
    mockedVerify.mockResolvedValueOnce({ code: 0 } as any)
    mockedVerify.mockResolvedValueOnce({ code: 1 } as any)

    const store = useAuthStore()
    store.token = 't'
    localStorage.setItem('authToken', 't')

    const ok1 = await store.checkToken()
    expect(ok1).toBe(true)
    expect(store.isAuthenticated).toBe(true)

    const ok2 = await store.checkToken()
    expect(ok2).toBe(false)
    expect(store.token).toBe('')
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.getItem('authToken')).toBeNull()
  })

  it('checkToken logs out when verifyToken throws', async () => {
    setActivePinia(createPinia())

    const mockedVerify = vi.mocked(authApi.verifyToken)
    mockedVerify.mockRejectedValue(new Error('network'))

    const store = useAuthStore()
    store.token = 't'
    localStorage.setItem('authToken', 't')

    const ok = await store.checkToken()
    expect(ok).toBe(false)
    expect(store.token).toBe('')
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.getItem('authToken')).toBeNull()
  })
})
