import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import UploadMenu from '@/components/chat/UploadMenu.vue'

describe('components/chat/UploadMenu.vue (more branches)', () => {
  it('renders nothing when visible=false', () => {
    const wrapper = mount(UploadMenu, {
      props: {
        visible: false,
        uploadedMedia: []
      }
    })
    expect(wrapper.text()).toBe('')
  })

  it('shows empty state when uploadedMedia is empty', () => {
    const wrapper = mount(UploadMenu, {
      props: {
        visible: true,
        uploadedMedia: [],
        canOpenChatHistory: false
      }
    })
    expect(wrapper.text()).toContain('暂无已上传的文件')
    expect(wrapper.text()).not.toContain('历史聊天图片')
  })

  it('renders MediaTile props for video poster / video / image and emits all action buttons', async () => {
    const MediaTile = {
      name: 'MediaTile',
      props: ['src', 'type', 'poster', 'fit'],
      template:
        '<div class=\"media-tile-stub\" :data-src=\"src\" :data-type=\"type\" :data-poster=\"poster\" :data-fit=\"fit\"><slot /><slot name=\"center\" /><slot name=\"file\" /></div>'
    }

    const wrapper = mount(UploadMenu, {
      props: {
        visible: true,
        canOpenChatHistory: true,
        uploadedMedia: [
          { url: 'http://x/v.mp4', type: 'video', posterUrl: 'http://x/v.poster.jpg' } as any,
          { url: 'http://x/v2.mp4', type: 'video' } as any,
          { url: 'http://x/a.png', type: 'image' } as any,
          { url: 'http://x/a.bin', type: 'file' } as any
        ]
      },
      global: {
        stubs: { MediaTile }
      }
    })

    const tiles = wrapper.findAll('.media-tile-stub')
    expect(tiles).toHaveLength(4)

    expect(tiles[0]!.attributes('data-src')).toBe('http://x/v.poster.jpg')
    expect(tiles[0]!.attributes('data-type')).toBe('image')
    expect(tiles[0]!.attributes('data-poster')).toBe('http://x/v.poster.jpg')
    expect(tiles[0]!.attributes('data-fit')).toBe('contain')

    expect(tiles[1]!.attributes('data-src')).toBe('http://x/v2.mp4')
    expect(tiles[1]!.attributes('data-type')).toBe('video')
    expect(tiles[1]!.attributes('data-fit')).toBe('contain')

    expect(tiles[2]!.attributes('data-src')).toBe('http://x/a.png')
    expect(tiles[2]!.attributes('data-type')).toBe('image')
    expect(tiles[2]!.attributes('data-fit')).toBe('cover')

    // Action buttons: uploadFile, openChatHistory, openAllUploads, openMtPhoto, openDouyinFavoriteAuthors
    const buttons = wrapper.findAll('button')
    expect(buttons.length).toBeGreaterThanOrEqual(5)
    await buttons[0]!.trigger('click')
    await buttons[1]!.trigger('click')
    await buttons[2]!.trigger('click')
    await buttons[3]!.trigger('click')
    await buttons[4]!.trigger('click')

    expect(wrapper.emitted('uploadFile')).toBeTruthy()
    expect(wrapper.emitted('openChatHistory')).toBeTruthy()
    expect(wrapper.emitted('openAllUploads')).toBeTruthy()
    expect(wrapper.emitted('openMtPhoto')).toBeTruthy()
    expect(wrapper.emitted('openDouyinFavoriteAuthors')).toBeTruthy()
  })
})

