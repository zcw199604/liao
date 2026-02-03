import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSystemConfigStore } from '@/stores/systemConfig'
import * as systemApi from '@/api/system'

vi.mock('@/api/system', () => ({
  getSystemConfig: vi.fn(),
  updateSystemConfig: vi.fn(),
  resolveImagePort: vi.fn()
}))

describe('stores/systemConfig', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
  })

  it('loadSystemConfig returns early when already loading', async () => {
    const store = useSystemConfigStore()
    store.loading = true

    await store.loadSystemConfig()
    expect(systemApi.getSystemConfig).not.toHaveBeenCalled()
  })

  it('saveSystemConfig returns early when already saving', async () => {
    const store = useSystemConfigStore()
    store.saving = true

    const ok = await store.saveSystemConfig({ imagePortFixed: '9007' } as any)
    expect(ok).toBe(false)
    expect(systemApi.updateSystemConfig).not.toHaveBeenCalled()
  })

  it('saveSystemConfig applies fallback defaults when backend returns partial config', async () => {
    vi.mocked(systemApi.updateSystemConfig).mockResolvedValue({ code: 0, data: {} } as any)

    const store = useSystemConfigStore()
    store.imagePortMode = 'real' as any
    store.imagePortFixed = '9999'
    store.imagePortRealMinBytes = 123

    const ok = await store.saveSystemConfig({} as any)
    expect(ok).toBe(true)
    expect(store.loaded).toBe(true)
    // Empty config -> defaults.
    expect(store.imagePortMode).toBe('fixed')
    expect(store.imagePortFixed).toBe('9006')
    expect(store.imagePortRealMinBytes).toBe(2048)
  })

  it('saveSystemConfig returns false when backend does not return data', async () => {
    vi.mocked(systemApi.updateSystemConfig).mockResolvedValue({ code: 0, data: null } as any)

    const store = useSystemConfigStore()
    const ok = await store.saveSystemConfig({ imagePortFixed: '9007' } as any)
    expect(ok).toBe(false)
  })

  it('resolveImagePort returns default fixed port when fixed mode or port missing', async () => {
    const store = useSystemConfigStore()
    store.loaded = true
    store.imagePortMode = 'fixed' as any
    store.imagePortFixed = '' // exercise fallback to DEFAULT_FIXED_PORT

    const port = await store.resolveImagePort('/img/Upload/a.png', 'srv')
    expect(port).toBe('9006')
    expect(systemApi.resolveImagePort).not.toHaveBeenCalled()
  })

  it('resolveImagePort caches result, preserves cache when clearResolvedCache is called for another server, and refreshes on server change', async () => {
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1_000)
    vi.mocked(systemApi.resolveImagePort)
      .mockResolvedValueOnce({ code: 0, data: { port: 9010 } } as any)
      .mockResolvedValueOnce({ code: 0, data: { port: 9011 } } as any)

    const store = useSystemConfigStore()
    store.loaded = true
    store.imagePortMode = 'real' as any
    store.imagePortFixed = '9009'

    const p1 = await store.resolveImagePort('/img/Upload/a.png', 'srv1')
    expect(p1).toBe('9010')
    expect(systemApi.resolveImagePort).toHaveBeenCalledTimes(1)

    // Calling clearResolvedCache for a different server should NOT clear cached value.
    store.clearResolvedCache('srv2')
    const p2 = await store.resolveImagePort('/img/Upload/a.png', 'srv1')
    expect(p2).toBe('9010')
    expect(systemApi.resolveImagePort).toHaveBeenCalledTimes(1)

    // Server changed -> cache should be cleared -> calls resolve again.
    const p3 = await store.resolveImagePort('/img/Upload/a.png', 'srv2')
    expect(p3).toBe('9011')
    expect(systemApi.resolveImagePort).toHaveBeenCalledTimes(2)

    nowSpy.mockRestore()
  })

  it('resolveImagePort returns fixed port when resolve api returns no usable port, and covers no-imgServer branch', async () => {
    vi.mocked(systemApi.resolveImagePort).mockResolvedValue({ code: 0, data: {} } as any)

    const store = useSystemConfigStore()
    store.loaded = true
    store.imagePortMode = 'real' as any
    store.imagePortFixed = '9009'

    const port = await store.resolveImagePort('/img/Upload/a.png')
    expect(port).toBe('9009')
  })

  it('resolveImagePort stores cached port even when imgServer is not provided (covers resolvedForServer else branch)', async () => {
    vi.mocked(systemApi.resolveImagePort).mockResolvedValue({ code: 0, data: { port: 9012 } } as any)

    const store = useSystemConfigStore()
    store.loaded = true
    store.imagePortMode = 'real' as any
    store.imagePortFixed = '9009'

    const port = await store.resolveImagePort('/img/Upload/a.png')
    expect(port).toBe('9012')
    expect(systemApi.resolveImagePort).toHaveBeenCalledTimes(1)
  })
})
