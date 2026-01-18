import type { MessageSegment } from '@/types'
import { isImageFile, isVideoFile } from '@/utils/file'

type MediaKind = 'image' | 'video' | 'file'

export type ResolveMediaUrl = (path: string) => Promise<string>

const hasOwn = (obj: Record<string, any>, key: string): boolean =>
  Object.prototype.hasOwnProperty.call(obj, key)

const looksLikeFileExt = (ext: string): boolean => {
  const e = String(ext || '').toLowerCase()
  if (!e) return false
  if (e.length > 10) return false
  if (!/^[a-z0-9]+$/.test(e)) return false
  // Require at least one letter to avoid false positives like "[v1.0]" -> ext "0"
  return /[a-z]/.test(e)
}

const inferMediaKind = (path: string): MediaKind | '' => {
  const raw = String(path || '').trim()
  if (!raw) return ''
  if (raw.includes('://')) return ''
  if (/\s/.test(raw)) return ''

  const clean = raw.split('?')[0]?.split('#')[0] || raw

  if (isImageFile(clean)) return 'image'
  if (isVideoFile(clean)) return 'video'

  const dot = clean.lastIndexOf('.')
  if (dot < 0) return ''
  const ext = clean.slice(dot + 1)
  if (!looksLikeFileExt(ext)) return ''
  return 'file'
}

const pushText = (segments: MessageSegment[], text: string) => {
  if (!text) return
  const last = segments[segments.length - 1]
  if (last && last.kind === 'text') {
    last.text += text
    return
  }
  segments.push({ kind: 'text', text })
}

export const parseMessageSegments = async (
  rawContent: string,
  options: {
    emojiMap: Record<string, string>
    resolveMediaUrl: ResolveMediaUrl
  }
): Promise<MessageSegment[]> => {
  const input = String(rawContent || '')
  if (!input) return []

  const segments: MessageSegment[] = []
  let i = 0

  while (i < input.length) {
    const open = input.indexOf('[', i)
    if (open < 0) {
      pushText(segments, input.slice(i))
      break
    }

    const close = input.indexOf(']', open + 1)
    if (close < 0) {
      pushText(segments, input.slice(i))
      break
    }

    if (open > i) {
      pushText(segments, input.slice(i, open))
    }

    const token = input.slice(open, close + 1)

    // Emoji token (e.g. "[doge]") should stay as text; it will be rendered via parseEmoji.
    if (hasOwn(options.emojiMap, token)) {
      pushText(segments, token)
      i = close + 1
      continue
    }

    const body = input.slice(open + 1, close)
    const path = String(body || '').trim()
    const kind = inferMediaKind(path)

    if (!kind) {
      pushText(segments, token)
      i = close + 1
      continue
    }

    const url = await options.resolveMediaUrl(path)
    if (!url) {
      // If we cannot resolve a URL (imgServer not ready, etc.), keep original text to avoid broken media nodes.
      pushText(segments, token)
      i = close + 1
      continue
    }

    segments.push({ kind, path, url })
    i = close + 1
  }

  return segments
}

export const getSegmentsMeta = (segments: MessageSegment[]) => {
  let imageUrl = ''
  let videoUrl = ''
  let fileUrl = ''
  let firstMediaPath = ''

  for (const seg of segments) {
    if (seg.kind === 'text') continue

    if (!firstMediaPath) firstMediaPath = seg.path
    if (!imageUrl && seg.kind === 'image') imageUrl = seg.url
    if (!videoUrl && seg.kind === 'video') videoUrl = seg.url
    if (!fileUrl && seg.kind === 'file') fileUrl = seg.url
  }

  return {
    hasImage: !!imageUrl,
    hasVideo: !!videoUrl,
    hasFile: !!fileUrl,
    imageUrl,
    videoUrl,
    fileUrl,
    firstMediaPath
  }
}

export const buildLastMsgPreviewFromSegments = (
  segments: MessageSegment[],
  options?: { maxTextLength?: number }
): string => {
  const maxTextLength = options?.maxTextLength ?? 30

  const text = segments
    .filter(s => s.kind === 'text')
    .map(s => s.text)
    .join('')
    .trim()

  const hasImage = segments.some(s => s.kind === 'image')
  const hasVideo = !hasImage && segments.some(s => s.kind === 'video')
  const hasFile = !hasImage && !hasVideo && segments.some(s => s.kind === 'file')
  const tag = hasImage ? '[图片]' : hasVideo ? '[视频]' : hasFile ? '[文件]' : ''

  let t = text
  if (t && t.length > maxTextLength) {
    t = t.slice(0, maxTextLength) + '...'
  }

  if (!t) return tag || '[消息]'
  return tag ? `${t} ${tag}` : t
}

