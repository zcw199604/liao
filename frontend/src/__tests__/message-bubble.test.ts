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
  template: `<div class="media-tile" @click="$emit('click')"></div>`
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
})
