import { describe, expect, it, vi } from 'vitest'
import { emojiMap } from '@/constants/emoji'
import { buildLastMsgPreviewFromSegments, getSegmentsMeta, parseMessageSegments } from '@/utils/messageSegments'

describe('utils/messageSegments', () => {
  it('parses inline image with text into segments', async () => {
    const resolveMediaUrl = vi.fn(async (path: string) => `http://x/img/Upload/${path}`)

    const segments = await parseMessageSegments('喜欢吗[20260104/image.jpg]', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([
      { kind: 'text', text: '喜欢吗' },
      { kind: 'image', path: '20260104/image.jpg', url: 'http://x/img/Upload/20260104/image.jpg' }
    ])
    expect(resolveMediaUrl).toHaveBeenCalledTimes(1)

    const meta = getSegmentsMeta(segments)
    expect(meta.hasImage).toBe(true)
    expect(meta.imageUrl).toBe('http://x/img/Upload/20260104/image.jpg')
  })

  it('does not treat emoji token as media', async () => {
    const resolveMediaUrl = vi.fn(async (_path: string) => 'http://x/should-not-be-called')

    const segments = await parseMessageSegments('[doge]', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([{ kind: 'text', text: '[doge]' }])
    expect(resolveMediaUrl).not.toHaveBeenCalled()
  })

  it('builds lastMsg preview with text + media tag', () => {
    const segments = [
      { kind: 'text', text: '喜欢吗' },
      { kind: 'image', path: '20260104/image.jpg', url: 'http://x/img/Upload/20260104/image.jpg' }
    ] as any

    expect(buildLastMsgPreviewFromSegments(segments)).toBe('喜欢吗 [图片]')
  })
})

