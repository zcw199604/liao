import { afterEach, describe, expect, it, vi } from 'vitest'

import { md5Hex } from '@/utils/md5'

describe('utils/md5 md5Hex', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('hashes common test vectors', () => {
    expect(md5Hex('')).toBe('d41d8cd98f00b204e9800998ecf8427e')
    expect(md5Hex('me')).toBe('ab86a1e1ef70dff97959067b723c5c24')
    expect(md5Hex('你好')).toBe('7eca689f0d3389d9dea66ae112e5cfd7')
  })

  it('returns 32 lowercase hex chars', () => {
    expect(md5Hex('u1')).toMatch(/^[0-9a-f]{32}$/)
  })

  it('treats nullish input as empty string', () => {
    expect(md5Hex(undefined as any)).toBe(md5Hex(''))
  })

  it('falls back to manual utf-8 encoding when TextEncoder is unavailable', () => {
    const cases = ['A', '¢', '中', '😀']
    const expected = new Map(cases.map(s => [s, md5Hex(s)]))

    vi.stubGlobal('TextEncoder', undefined as any)

    for (const s of cases) {
      expect(md5Hex(s)).toBe(expected.get(s))
    }
  })

  it('handles surrogate edge cases without TextEncoder', () => {
    vi.stubGlobal('TextEncoder', undefined as any)
    expect(md5Hex('\uD83D' + 'a')).toMatch(/^[0-9a-f]{32}$/)
  })

  it('covers padding calculation branch for long inputs', () => {
    vi.stubGlobal('TextEncoder', undefined as any)
    expect(md5Hex('a'.repeat(56))).toMatch(/^[0-9a-f]{32}$/)
  })
})
