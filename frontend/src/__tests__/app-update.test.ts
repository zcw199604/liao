import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { watchAppUpdates } from '@/utils/appUpdate'

describe('app update detection', () => {
  let stop: (() => void) | undefined
  const fetchMock = vi.fn()
  const clearStoredData = vi.fn()
  const removeStoredData = vi.fn()
  const response = (version: string, entry = '/assets/index-current.js') => ({ ok: true, json: async () => ({ version, entry }) })
  const flush = async () => { await Promise.resolve(); await Promise.resolve(); await Promise.resolve() }

  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('localStorage', { getItem: () => 'keep', clear: clearStoredData, removeItem: removeStoredData })
    fetchMock.mockReset()
    document.head.innerHTML = '<script type="module" src="/assets/index-current.js"></script>'
  })
  afterEach(() => { stop?.(); vi.useRealTimers(); vi.unstubAllGlobals() })

  it('detects an already outdated loaded entry without clearing stored data', async () => {
    const notify = vi.fn()
    fetchMock.mockResolvedValue(response('new', '/assets/index-new.js'))
    stop = watchAppUpdates(notify)
    await flush()
    expect(notify).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/version.json'), expect.objectContaining({ cache: 'no-store' }))
    expect(window.localStorage.getItem('authToken')).toBe('keep')
    expect(clearStoredData).not.toHaveBeenCalled()
    expect(removeStoredData).not.toHaveBeenCalled()
  })

  it('checks after returning to foreground and during long visits, notifying only once', async () => {
    const notify = vi.fn()
    fetchMock.mockResolvedValueOnce(response('old')).mockResolvedValue(response('new'))
    stop = watchAppUpdates(notify)
    await flush()
    expect(notify).not.toHaveBeenCalled()
    window.dispatchEvent(new Event('focus'))
    await flush()
    expect(notify).toHaveBeenCalledOnce()
    await vi.advanceTimersByTimeAsync(60_000)
    expect(notify).toHaveBeenCalledOnce()
    stop()
    const calls = fetchMock.mock.calls.length
    window.dispatchEvent(new Event('focus'))
    await vi.advanceTimersByTimeAsync(60_000)
    expect(fetchMock).toHaveBeenCalledTimes(calls)
  })

  it('does not treat offline errors or non-JSON development responses as updates', async () => {
    const notify = vi.fn()
    fetchMock.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ ok: true, json: async () => { throw new Error('html') } }).mockResolvedValue(response('same'))
    stop = watchAppUpdates(notify)
    await flush()
    await vi.advanceTimersByTimeAsync(120_000)
    expect(notify).not.toHaveBeenCalled()
  })

  it('prompts refresh on missing lazy chunks without automatic reload loops', () => {
    fetchMock.mockResolvedValue(response('same'))
    const notify = vi.fn()
    stop = watchAppUpdates(notify)
    const event = new Event('vite:preloadError', { cancelable: true })
    window.dispatchEvent(event)
    expect(notify).toHaveBeenCalledOnce()
    expect(event.defaultPrevented).toBe(true)
  })
})
