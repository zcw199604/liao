import { afterEach, describe, expect, it, vi } from 'vitest'

import { copyToClipboard } from '@/utils/clipboard'

const setClipboard = (clipboard: any) => {
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: clipboard
  })
}

describe('utils/clipboard', () => {
  const originalClipboard = (navigator as any).clipboard
  const originalExecCommand = (document as any).execCommand

  afterEach(() => {
    setClipboard(originalClipboard)
    ;(document as any).execCommand = originalExecCommand
    vi.restoreAllMocks()
  })

  it('uses navigator.clipboard.writeText when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    setClipboard({ writeText })

    const ok = await copyToClipboard(' hello ')
    expect(ok).toBe(true)
    expect(writeText).toHaveBeenCalledWith('hello')
  })

  it('falls back to document.execCommand when writeText fails', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('denied'))
    setClipboard({ writeText })
    ;(document as any).execCommand = vi.fn().mockReturnValue(true)

    const ok = await copyToClipboard('hi')
    expect(ok).toBe(true)
    expect((document as any).execCommand).toHaveBeenCalledWith('copy')
  })

  it('returns false when no method is available', async () => {
    setClipboard(undefined)
    ;(document as any).execCommand = undefined

    const ok = await copyToClipboard('hi')
    expect(ok).toBe(false)
  })
})

