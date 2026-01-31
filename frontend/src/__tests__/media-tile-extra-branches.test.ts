import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import MediaTile from '@/components/common/MediaTile.vue'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('components/common/MediaTile.vue (extra branches)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders corner slots and applies reveal classes when enabled', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/a.png',
        type: 'image',
        lazy: false,
        showSkeleton: false,
        revealTopRight: true,
        revealBottomLeft: true,
        revealBottomRight: true
      },
      slots: {
        'top-right': '<span class="tr">TR</span>',
        'bottom-left': '<span class="bl">BL</span>',
        'bottom-right': '<span class="br">BR</span>'
      }
    })
    await flushAsync()

    expect(wrapper.find('.tr').exists()).toBe(true)
    expect(wrapper.find('.bl').exists()).toBe(true)
    expect(wrapper.find('.br').exists()).toBe(true)

    // reveal* -> slot wrappers get media-tile-reveal class
    const wrappers = wrapper.findAll('.media-tile-reveal')
    expect(wrappers.length).toBeGreaterThanOrEqual(3)
  })

  it('applies aspectRatio from props', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/aspect-prop.png',
        type: 'image',
        lazy: false,
        aspectRatio: 2
      }
    })
    await flushAsync()
    expect(String((wrapper.element as HTMLElement).getAttribute('style') || '')).toContain('aspect-ratio: 2')
  })

  it('shows backdrop for contain-fit images and hides it once hasError is set', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/contain.png',
        type: 'image',
        fit: 'contain',
        lazy: false,
        showSkeleton: false
      }
    })
    await flushAsync()

    const backdrop = wrapper.find('img[aria-hidden=\"true\"]')
    expect(backdrop.exists()).toBe(true)
    expect(backdrop.attributes('src')).toBe('http://x/contain.png')

    // Trigger error -> switches to error state, backdrop no longer rendered.
    await wrapper.find('img:not([aria-hidden=\"true\"])').trigger('error')
    await flushAsync()
    expect(wrapper.find('img[aria-hidden=\"true\"]').exists()).toBe(false)
  })

  it('covers fit=fill, fill=false, indicator sizes, and handleLoaded video-dimension branch', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/v.mp4',
        type: 'video',
        fit: 'fill',
        fill: false,
        indicatorSize: 'sm',
        lazy: false,
        controls: false,
        showVideoIndicator: true,
        showSkeleton: false
      }
    })
    await flushAsync()

    const video = wrapper.find('video')
    expect(video.exists()).toBe(true)
    expect(video.classes().join(' ')).toContain('object-fill')
    expect(video.classes().join(' ')).toContain('max-w-full')

    // sm indicator
    expect(wrapper.find('.w-6.h-6').exists()).toBe(true)

    // lg indicator
    await wrapper.setProps({ indicatorSize: 'lg' })
    await flushAsync()
    expect(wrapper.find('.w-10.h-10').exists()).toBe(true)

    // handleLoaded: video branch uses videoWidth/videoHeight for cache.
    const fakeVideo = document.createElement('video') as HTMLVideoElement
    Object.defineProperty(fakeVideo, 'videoWidth', { configurable: true, get: () => 300 })
    Object.defineProperty(fakeVideo, 'videoHeight', { configurable: true, get: () => 150 })
    ;(wrapper.vm as any).handleLoaded({ target: fakeVideo } as any)
    await flushAsync()

    expect(wrapper.emitted('load')).toBeTruthy()
    expect(wrapper.emitted('layout')).toBeTruthy()
  })

  it('error text/icon changes by resolvedType (video/file)', async () => {
    // video error
    const wrapper = mount(MediaTile, {
      props: { src: 'http://x/v.mp4', type: 'video', lazy: false, showSkeleton: false }
    })
    await flushAsync()
    await wrapper.find('video').trigger('error')
    await flushAsync()
    expect(wrapper.text()).toContain('视频加载失败')
    expect(wrapper.find('i.fas.fa-video-slash').exists()).toBe(true)

    // file error branch via manual call (file tiles don't emit error events)
    const wrapper2 = mount(MediaTile, {
      props: { src: 'http://x/a.bin', type: 'file', lazy: false }
    })
    await flushAsync()
    ;(wrapper2.vm as any).handleError(new Event('error'))
    await flushAsync()
    expect(wrapper2.text()).toContain('文件不可用')
    expect(wrapper2.find('i.fas.fa-file-excel').exists()).toBe(true)
  })
})
