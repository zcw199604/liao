import { describe, expect, it } from 'vitest'

import { API_BASE, IMG_SERVER_ADDRESS, IMG_SERVER_IMAGE_PORT, WS_URL, setImgServerAddress } from '@/constants/config'

describe('constants/config', () => {
  it('exports defaults and allows updating IMG_SERVER_ADDRESS', () => {
    expect(API_BASE).toBe('/api')
    expect(WS_URL).toBe('/ws')
    expect(IMG_SERVER_IMAGE_PORT).toBe(9006)

    const before = IMG_SERVER_ADDRESS
    setImgServerAddress('img.local')
    expect(IMG_SERVER_ADDRESS).toBe('img.local')

    // restore to keep test isolation
    setImgServerAddress(before)
  })
})
