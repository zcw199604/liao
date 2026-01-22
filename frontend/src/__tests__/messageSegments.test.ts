import { describe, expect, it, vi } from 'vitest'
import { emojiMap } from '@/constants/emoji'
import { buildLastMsgPreviewFromSegments, getSegmentsMeta, parseMessageSegments } from '@/utils/messageSegments'

describe('utils/messageSegments', () => {
  it('returns empty list for empty input', async () => {
    const resolveMediaUrl = vi.fn(async (_path: string) => 'http://x/should-not-be-called')
    const segments = await parseMessageSegments('', { emojiMap, resolveMediaUrl })
    expect(segments).toEqual([])
    expect(resolveMediaUrl).not.toHaveBeenCalled()
  })

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

  it('treats unclosed bracket as plain text', async () => {
    const resolveMediaUrl = vi.fn(async (_path: string) => 'http://x/should-not-be-called')

    const segments = await parseMessageSegments('hi [20260104/image.jpg', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([{ kind: 'text', text: 'hi [20260104/image.jpg' }])
    expect(resolveMediaUrl).not.toHaveBeenCalled()
  })

  it('does not treat token containing :// as media', async () => {
    const resolveMediaUrl = vi.fn(async (_path: string) => 'http://x/should-not-be-called')

    const segments = await parseMessageSegments('[http://x/a.png]', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([{ kind: 'text', text: '[http://x/a.png]' }])
    expect(resolveMediaUrl).not.toHaveBeenCalled()
  })

  it('does not treat token containing whitespace as media', async () => {
    const resolveMediaUrl = vi.fn(async (_path: string) => 'http://x/should-not-be-called')

    const segments = await parseMessageSegments('x[2026/ a.png]y', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([{ kind: 'text', text: 'x[2026/ a.png]y' }])
    expect(resolveMediaUrl).not.toHaveBeenCalled()
  })

  it('does not treat numeric-only extension as file', async () => {
    const resolveMediaUrl = vi.fn(async (_path: string) => 'http://x/should-not-be-called')

    const segments = await parseMessageSegments('[v1.0]', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([{ kind: 'text', text: '[v1.0]' }])
    expect(resolveMediaUrl).not.toHaveBeenCalled()
  })

  it('falls back to text when resolveMediaUrl returns empty', async () => {
    const resolveMediaUrl = vi.fn(async (_path: string) => '')

    const segments = await parseMessageSegments('[20260104/image.jpg]', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([{ kind: 'text', text: '[20260104/image.jpg]' }])
    expect(resolveMediaUrl).toHaveBeenCalledTimes(1)
  })

  it('parses adjacent media tokens into two segments', async () => {
    const resolveMediaUrl = vi.fn(async (path: string) => `http://x/media/${path}`)

    const segments = await parseMessageSegments('[a.jpg][b.mp4]', {
      emojiMap,
      resolveMediaUrl
    })

    expect(segments).toEqual([
      { kind: 'image', path: 'a.jpg', url: 'http://x/media/a.jpg' },
      { kind: 'video', path: 'b.mp4', url: 'http://x/media/b.mp4' }
    ])
    expect(resolveMediaUrl).toHaveBeenCalledTimes(2)
  })

  it('builds lastMsg preview with text + media tag', () => {
    const segments = [
      { kind: 'text', text: '喜欢吗' },
      { kind: 'image', path: '20260104/image.jpg', url: 'http://x/img/Upload/20260104/image.jpg' }
    ] as any

    expect(buildLastMsgPreviewFromSegments(segments)).toBe('喜欢吗 [图片]')
  })

  it('builds lastMsg preview fallback for empty segments', () => {
    expect(buildLastMsgPreviewFromSegments([])).toBe('[消息]')
  })

  it('builds lastMsg preview tag-only when no text', () => {
    const segments = [{ kind: 'video', path: 'a.mp4', url: 'http://x/a.mp4' }] as any
    expect(buildLastMsgPreviewFromSegments(segments)).toBe('[视频]')
  })

  it('builds lastMsg preview prefers image tag when both image and video exist', () => {
    const segments = [
      { kind: 'video', path: 'a.mp4', url: 'http://x/a.mp4' },
      { kind: 'image', path: 'b.jpg', url: 'http://x/b.jpg' }
    ] as any
    expect(buildLastMsgPreviewFromSegments(segments)).toBe('[图片]')
  })

  it('builds lastMsg preview truncates long text and keeps tag', () => {
    const segments = [
      { kind: 'text', text: '1234567890123456789012345678901' },
      { kind: 'file', path: 'a.bin', url: 'http://x/a.bin' }
    ] as any
    expect(buildLastMsgPreviewFromSegments(segments, { maxTextLength: 30 })).toBe('123456789012345678901234567890... [文件]')
  })
})
