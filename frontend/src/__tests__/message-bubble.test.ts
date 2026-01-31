import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import { formatTime } from '@/utils/time'

const uploadMocks = {
  getMediaUrl: (input: string) => input
}

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => uploadMocks
}))

import MessageBubble from '@/components/chat/MessageBubble.vue'

const MediaTileStub = {
  name: 'MediaTile',
  props: ['src', 'type'],
  emits: ['click', 'error'],
  template: `<div class="media-tile" @click="$emit('click')"><slot name="center" /></div>`
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('components/chat/MessageBubble.vue', () => {
  it('renders plain text message and formats timestamp', () => {
    const wrapper = mount(MessageBubble, {
      props: {
        message: {
          tid: '1',
          time: '2026-01-01 00:00:00.000',
          content: 'hi',
          isSelf: false,
          isImage: false,
          isVideo: false,
          isFile: false,
          imageUrl: '',
          videoUrl: '',
          fileUrl: ''
        } as any
      },
      global: { stubs: { MediaTile: MediaTileStub } }
    })

    expect(wrapper.text()).toContain('hi')
    expect(wrapper.text()).toContain(formatTime('2026-01-01 00:00:00.000'))
  })

  it('handles file message: computes filename and downloads via anchor', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const wrapper = mount(MessageBubble, {
      props: {
        message: {
          tid: '1',
          time: '2026-01-01 00:00:00.000',
          content: 'https://example.com/files/%E4%B8%AD%E6%96%87.txt',
          isSelf: true,
          isImage: false,
          isVideo: false,
          isFile: true,
          imageUrl: '',
          videoUrl: '',
          fileUrl: 'https://example.com/files/%E4%B8%AD%E6%96%87.txt'
        } as any
      },
      global: { stubs: { MediaTile: MediaTileStub } }
    })

    expect(wrapper.text()).toContain('中文.txt')
    await wrapper.find('.fa-download').element.closest('div')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(clickSpy).toHaveBeenCalled()

    clickSpy.mockRestore()
  })

  it('renders segments: image/video/file branches and dispatches preview event', async () => {
    const previewSpy = vi.fn()
    window.addEventListener('preview-media', previewSpy as any)

    const wrapper = mount(MessageBubble, {
      props: {
        message: {
          tid: '1',
          time: '2026-01-01 00:00:00.000',
          content: '',
          isSelf: false,
          isImage: false,
          isVideo: false,
          isFile: false,
          imageUrl: '',
          videoUrl: '',
          fileUrl: '',
          segments: [
            { kind: 'text', text: 'hello' },
            { kind: 'image', url: '/img/a.png' },
            { kind: 'video', url: '/videos/a.mp4' },
            { kind: 'file', url: '/files/a.txt' }
          ]
        } as any
      },
      global: { stubs: { MediaTile: MediaTileStub } }
    })

    expect(wrapper.text()).toContain('hello')

    // click first media tile -> image preview event
    await wrapper.findAll('.media-tile')[0]!.trigger('click')
    expect(previewSpy).toHaveBeenCalled()

    expect(wrapper.text()).toContain('点击下载')

    window.removeEventListener('preview-media', previewSpy as any)
  })

  it('renders self segments (covers file segment class branches) and handles handleImageError fallback', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const wrapper = mount(MessageBubble, {
      props: {
        message: {
          tid: '1',
          time: '',
          content: '',
          isSelf: true,
          isImage: false,
          isVideo: false,
          isFile: false,
          imageUrl: '',
          videoUrl: '',
          fileUrl: '',
          segments: [{ kind: 'file', url: '/files/a.txt' }]
        } as any
      },
      global: { stubs: { MediaTile: MediaTileStub } }
    })

    // file segment shows the download label and uses self styles
    expect(wrapper.text()).toContain('点击下载')
    expect(wrapper.find('.msg-bubble').classes().join(' ')).toContain('msg-right')

    // handleImageError prints target.src if present, otherwise falls back to computed imageUrl
    ;(wrapper.vm as any).handleImageError({ target: { src: 'http://x/img.png' } })
    ;(wrapper.vm as any).handleImageError({ target: null })
    expect(consoleSpy).toHaveBeenCalled()

    consoleSpy.mockRestore()
  })

  it('computed media urls fall back to message.content when explicit urls are missing', () => {
    const wrapper = mount(MessageBubble, {
      props: {
        message: {
          tid: '1',
          time: '2026-01-01 00:00:00.000',
          content: '/upload/images/a.png',
          isSelf: false,
          isImage: true,
          isVideo: false,
          isFile: false,
          imageUrl: '',
          videoUrl: '',
          fileUrl: ''
        } as any
      },
      global: { stubs: { MediaTile: MediaTileStub } }
    })

    // imageUrl computed should use content fallback
    expect((wrapper.vm as any).imageUrl).toBe('/upload/images/a.png')

    // parsedContent returns '' when content is empty
    const wrapper2 = mount(MessageBubble, {
      props: { message: { tid: '1', time: 't', content: '', isSelf: false, isImage: false, isVideo: false, isFile: false } as any },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect((wrapper2.vm as any).parsedContent).toBe('')
  })

  it('getFileNameFromUrl falls back for relative paths and guards empty downloads', async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    const wrapper = mount(MessageBubble, {
      props: {
        message: {
          tid: '1',
          time: '2026-01-01 00:00:00.000',
          content: '',
          isSelf: true,
          isImage: false,
          isVideo: false,
          isFile: true,
          fileUrl: '/upload/files/a.txt'
        } as any
      },
      global: { stubs: { MediaTile: MediaTileStub } }
    })

    expect(wrapper.text()).toContain('a.txt')

    ;(wrapper.vm as any).downloadUrl('')
    expect(clickSpy).not.toHaveBeenCalled()

    clickSpy.mockRestore()
  })

  it('covers videoUrl/fileUrl fallbacks and getFileNameFromUrl unknown-name branches', () => {
    // videoUrl: falls back to content then to empty string
    const w1 = mount(MessageBubble, {
      props: {
        message: {
          tid: '1',
          time: 't',
          content: '/v.mp4',
          isSelf: false,
          isImage: false,
          isVideo: true,
          isFile: false,
          videoUrl: ''
        } as any
      },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect((w1.vm as any).videoUrl).toBe('/v.mp4')

    const w2 = mount(MessageBubble, {
      props: { message: { tid: '1', time: 't', content: '', isSelf: false, isImage: false, isVideo: true, isFile: false, videoUrl: '' } as any },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect((w2.vm as any).videoUrl).toBe('')

    // fileUrl: falls back to content then to empty string
    const w3 = mount(MessageBubble, {
      props: { message: { tid: '1', time: 't', content: '/f.txt', isSelf: false, isImage: false, isVideo: false, isFile: true, fileUrl: '' } as any },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect((w3.vm as any).fileUrl).toBe('/f.txt')

    const w4 = mount(MessageBubble, {
      props: { message: { tid: '1', time: 't', content: '', isSelf: false, isImage: false, isVideo: false, isFile: true, fileUrl: '' } as any },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect((w4.vm as any).fileUrl).toBe('')

    // fileName computed returns empty when isFile=false
    const w5 = mount(MessageBubble, {
      props: { message: { tid: '1', time: 't', content: 'x', isSelf: false, isImage: false, isVideo: false, isFile: false } as any },
      global: { stubs: { MediaTile: MediaTileStub } }
    })
    expect((w5.vm as any).fileName).toBe('')

    // getFileNameFromUrl: empty -> fallback, URL with trailing slash -> fallback, and catch-path trailing slash -> fallback
    expect((w3.vm as any).getFileNameFromUrl('')).toBe('未知文件')
    expect((w3.vm as any).getFileNameFromUrl('http://example.com/a/')).toBe('未知文件')
    expect((w3.vm as any).getFileNameFromUrl('a/')).toBe('未知文件')
  })
})
