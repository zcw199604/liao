import { afterEach, describe, expect, it, vi } from 'vitest'

import { generateRandomHexId, generateRandomIP, generateSessionId, generateUniqueId } from '@/utils/id'

describe('utils/id', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('generateSessionId combines timestamp and random suffix', () => {
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    vi.spyOn(Math, 'random').mockReturnValue(0.123456)

    const id = generateSessionId()
    expect(id).toContain((1700000000000).toString(36))
    expect(id).toContain((0.123456).toString(36).substring(2))
  })

  it('generateRandomIP joins 4 octets', () => {
    vi.spyOn(Math, 'random')
      .mockReturnValueOnce(0)
      .mockReturnValueOnce(0.5)
      .mockReturnValueOnce(0.999)
      .mockReturnValueOnce(0.1)

    expect(generateRandomIP()).toBe('0.128.255.25')
  })

  it('generateUniqueId prefixes with timestamp', () => {
    vi.spyOn(Date, 'now').mockReturnValue(1700000000000)
    vi.spyOn(Math, 'random').mockReturnValue(0.123456)

    const id = generateUniqueId()
    expect(id.startsWith('1700000000000-')).toBe(true)
    expect(id.split('-')[1]).toMatch(/^[a-z0-9]+$/)
  })

  it('generateRandomHexId uses crypto.getRandomValues when available', () => {
    const getRandomValues = vi.fn((arr: Uint8Array) => {
      arr[0] = 0
      arr[1] = 255
      return arr
    })
    vi.stubGlobal('crypto', { getRandomValues } as any)

    expect(generateRandomHexId(4)).toBe('00ff')
    expect(getRandomValues).toHaveBeenCalled()
  })

  it('generateRandomHexId falls back to Math.random when crypto is unavailable', () => {
    vi.stubGlobal('crypto', undefined as any)
    vi.spyOn(Math, 'random').mockReturnValueOnce(0).mockReturnValueOnce(0.999)

    expect(generateRandomHexId(4)).toBe('00ff')
  })

  it('generateRandomHexId defaults to 32 hex chars', () => {
    vi.stubGlobal('crypto', { getRandomValues: (arr: Uint8Array) => arr } as any)
    expect(generateRandomHexId().length).toBe(32)
  })
})

