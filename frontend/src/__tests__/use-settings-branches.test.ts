import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useSettings } from '@/composables/useSettings'
import * as systemApi from '@/api/system'

vi.mock('@/api/system', () => ({
  getConnectionStats: vi.fn(),
  getForceoutUserCount: vi.fn(),
  disconnectAllConnections: vi.fn(),
  clearForceoutUsers: vi.fn()
}))

describe('composables/useSettings branch coverage', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('loadConnectionStats ignores non-success responses', async () => {
    vi.mocked(systemApi.getConnectionStats).mockResolvedValue({ code: 1, data: { active: 9 } } as any)

    const s = useSettings()
    await s.loadConnectionStats()

    expect(s.connectionStats.value).toEqual({ active: 0, upstream: 0, downstream: 0 })
  })

  it('loadForceoutUserCount ignores non-number payloads', async () => {
    vi.mocked(systemApi.getForceoutUserCount).mockResolvedValue({ code: 0, data: 'oops' } as any)

    const s = useSettings()
    await s.loadForceoutUserCount()

    expect(s.forceoutUserCount.value).toBe(0)
  })

  it('clearForceout uses fallback success message when msg is empty', async () => {
    vi.mocked(systemApi.clearForceoutUsers).mockResolvedValue({ code: 0, msg: '' } as any)
    vi.mocked(systemApi.getForceoutUserCount).mockResolvedValue({ code: 0, data: 2 } as any)

    const s = useSettings()
    const res = await s.clearForceout()

    expect(res).toEqual({ success: true, message: '清除成功' })
    expect(s.forceoutUserCount.value).toBe(2)
  })

  it('clearForceout uses fallback failure message when msg is empty', async () => {
    vi.mocked(systemApi.clearForceoutUsers).mockResolvedValue({ code: 1, msg: '' } as any)

    const s = useSettings()
    const res = await s.clearForceout()

    expect(res).toEqual({ success: false, message: '清除失败' })
  })
})

