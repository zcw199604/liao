import { describe, expect, it, vi } from 'vitest'

import { generateCookie, parseCookie } from '@/utils/cookie'
import { extractFileName, formatFileSize, getFileExtension, isImageFile, isVideoFile } from '@/utils/file'
import { extractRemoteFilePathFromImgUploadUrl, extractUploadLocalPath, inferMediaTypeFromUrl } from '@/utils/media'

describe('utils/cookie', () => {
  it('generateCookie includes userid/nickname/timestamp/random', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 5, 0, 0, 10))
    vi.spyOn(Math, 'random').mockReturnValue(0.123456)

    const cookie = generateCookie('u1', 'Alice')
    expect(cookie).toMatch(/^u1_Alice_\d+_[a-z0-9]{6}$/)

    const parts = cookie.split('_')
    expect(parts[0]).toBe('u1')
    expect(parts[1]).toBe('Alice')
    expect(parts[2]).toBe(String(Math.floor(Date.now() / 1000)))

    vi.useRealTimers()
  })

  it('parseCookie returns userid and nickname', () => {
    expect(parseCookie('u1_Alice_1_abcd12')).toEqual({ userid: 'u1', nickname: 'Alice' })
    // empty userid / nickname branches
    expect(parseCookie('_Nick_1_abcd12')).toEqual({ userid: '', nickname: 'Nick' })
    expect(parseCookie('u1__1_abcd12')).toEqual({ userid: 'u1', nickname: '' })
    expect(parseCookie('bad')).toBeNull()
  })
})

describe('utils/file', () => {
  it('extractFileName returns last segment', () => {
    expect(extractFileName('/a/b/c.txt')).toBe('c.txt')
    expect(extractFileName('')).toBe('')
  })

  it('getFileExtension handles missing extension and case', () => {
    expect(getFileExtension('a.JPG')).toBe('jpg')
    expect(getFileExtension('noext')).toBe('')
    expect(getFileExtension('')).toBe('')
  })

  it('isImageFile and isVideoFile match common extensions', () => {
    expect(isImageFile('a.png')).toBe(true)
    expect(isImageFile('a.mp4')).toBe(false)

    expect(isVideoFile('a.mp4')).toBe(true)
    expect(isVideoFile('a.png')).toBe(false)
  })

  it('formatFileSize formats bytes into KB/MB/GB', () => {
    expect(formatFileSize(1)).toBe('1 B')
    expect(formatFileSize(1024)).toBe('1.00 KB')
    expect(formatFileSize(1024 * 1024)).toBe('1.00 MB')
    expect(formatFileSize(1024 * 1024 * 1024)).toBe('1.00 GB')
  })
})

describe('utils/media', () => {
  it('extractUploadLocalPath extracts /images/... from /upload/images/...', () => {
    expect(extractUploadLocalPath('http://localhost:8080/upload/images/2026/01/a.jpg')).toBe('/images/2026/01/a.jpg')
    expect(extractUploadLocalPath('/images/2026/01/a.jpg')).toBe('/images/2026/01/a.jpg')
    expect(extractUploadLocalPath('/videos/2026/01/a.mp4')).toBe('/videos/2026/01/a.mp4')
    expect(extractUploadLocalPath('http://localhost:8080/upload/videos/2026/01/a.mp4')).toBe('/videos/2026/01/a.mp4')
    expect(extractUploadLocalPath('')).toBe('')
    expect(extractUploadLocalPath('not a url')).toBe('not a url')
  })

  it('extractRemoteFilePathFromImgUploadUrl extracts after /img/Upload/', () => {
    expect(extractRemoteFilePathFromImgUploadUrl('http://s:9006/img/Upload/2026/01/a.jpg')).toBe('2026/01/a.jpg')
    expect(extractRemoteFilePathFromImgUploadUrl('')).toBe('')
    expect(extractRemoteFilePathFromImgUploadUrl('http://x/other')).toBe('http://x/other')
  })

  it('inferMediaTypeFromUrl infers type ignoring query/hash', () => {
    expect(inferMediaTypeFromUrl('http://x/a.png?x=1')).toBe('image')
    expect(inferMediaTypeFromUrl('http://x/a.mp4#t=1')).toBe('video')
    expect(inferMediaTypeFromUrl('http://x/a.bin')).toBe('file')
    expect(inferMediaTypeFromUrl('')).toBe('file')
    // cover cleanUrl fallbacks when split yields empty prefix
    expect(inferMediaTypeFromUrl('?x=1')).toBe('file')
    expect(inferMediaTypeFromUrl('#t=1')).toBe('file')
  })
})
