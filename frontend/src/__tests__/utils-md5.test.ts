import { describe, expect, it } from 'vitest'

import { md5Hex } from '@/utils/md5'

describe('utils/md5 md5Hex', () => {
  it('hashes common test vectors', () => {
    expect(md5Hex('')).toBe('d41d8cd98f00b204e9800998ecf8427e')
    expect(md5Hex('me')).toBe('ab86a1e1ef70dff97959067b723c5c24')
    expect(md5Hex('你好')).toBe('7eca689f0d3389d9dea66ae112e5cfd7')
  })

  it('returns 32 lowercase hex chars', () => {
    expect(md5Hex('u1')).toMatch(/^[0-9a-f]{32}$/)
  })
})

