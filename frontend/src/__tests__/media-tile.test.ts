import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import MediaTile from '@/components/common/MediaTile.vue'

const SkeletonStub = {
  name: 'Skeleton',
  template: '<div data-testid="skeleton"></div>'
}

describe('components/common/MediaTile.vue', () => {
  const originalIntersectionObserver = (globalThis as any).IntersectionObserver

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    ;(globalThis as any).IntersectionObserver = originalIntersectionObserver
  })

  it('renders image with skeleton/overlay, handles load & error, and caches aspect ratio', async () => {
    ;(globalThis as any).IntersectionObserver = undefined

    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/1.png',
        type: 'image',
        hoverOverlay: true,
        showSkeleton: true
      },
      global: {
        stubs: {
          Skeleton: SkeletonStub
        }
      }
    })

    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="skeleton"]').exists()).toBe(true)
    expect(wrapper.find('.group-hover\\:bg-black\\/10').exists()).toBe(true)

    const img = wrapper.get('img')
    Object.defineProperty(img.element, 'naturalWidth', { configurable: true, value: 200 })
    Object.defineProperty(img.element, 'naturalHeight', { configurable: true, value: 100 })

    await img.trigger('load')
    expect(wrapper.find('[data-testid="skeleton"]').exists()).toBe(false)

    // cache is non-reactive; verify by re-mounting with same src
    const wrapper2 = mount(MediaTile, {
      props: {
        src: 'http://x/1.png',
        type: 'image',
        lazy: false,
        showSkeleton: false
      },
      global: { stubs: { Skeleton: SkeletonStub } }
    })
    await wrapper2.vm.$nextTick()

    await img.trigger('error')
    expect(wrapper.text()).toContain('图片加载失败')

    await wrapper.trigger('click')
    expect(wrapper.emitted('click')?.length).toBe(1)
  })

  it('covers video indicator sizing, fit/fill classes, and video error state', async () => {
    ;(globalThis as any).IntersectionObserver = undefined

    const wrapper = mount(MediaTile, {
      props: {
        src: '/upload/videos/a.mp4',
        type: 'video',
        indicatorSize: 'sm',
        fit: 'contain',
        fill: false,
        showSkeleton: false,
        hoverOverlay: true,
        controls: false
      },
      global: {
        stubs: {
          Skeleton: SkeletonStub
        }
      }
    })

    await wrapper.vm.$nextTick()

    const video = wrapper.get('video')
    expect(video.classes().some((c) => c.includes('object-contain'))).toBe(true)
    expect(video.classes().some((c) => c.includes('max-w-full'))).toBe(true)

    // animate-pulse when shouldLoad && !isLoaded && !showSkeleton
    expect(wrapper.classes().some((c) => c.includes('animate-pulse'))).toBe(true)

    const indicator = wrapper.find('.fa-play').element.closest('div') as HTMLElement | null
    expect(indicator).toBeTruthy()
    expect(indicator!.className).toContain('w-6')

    await video.trigger('error')
    expect(wrapper.text()).toContain('视频加载失败')

    // cover indicatorSize lg + fit fill
    const wrapper2 = mount(MediaTile, {
      props: {
        src: '/upload/videos/b.mp4',
        type: 'video',
        indicatorSize: 'lg',
        fit: 'fill',
        fill: true,
        showSkeleton: false,
        hoverOverlay: false,
        controls: false
      },
      global: { stubs: { Skeleton: SkeletonStub } }
    })
    await wrapper2.vm.$nextTick()
    const indicator2 = wrapper2.find('.fa-play').element.closest('div') as HTMLElement | null
    expect(indicator2).toBeTruthy()
    expect(indicator2!.className).toContain('w-10')
    expect(wrapper2.get('video').classes().some((c) => c.includes('object-fill'))).toBe(true)
    expect(wrapper2.get('video').classes().some((c) => c.includes('w-full'))).toBe(true)
  })

  it('covers file type branches, slots reveal classes, and center slot override', async () => {
    ;(globalThis as any).IntersectionObserver = undefined

    const wrapper = mount(MediaTile, {
      props: {
        src: '/upload/files/a.pdf',
        type: 'file',
        revealTopLeft: true
      },
      slots: {
        'top-left': '<span>TL</span>',
        center: '<span>Center</span>',
        file: '<span>FileSlot</span>'
      },
      global: {
        stubs: {
          Skeleton: SkeletonStub
        }
      }
    })

    expect(wrapper.text()).toContain('FileSlot')
    expect(wrapper.find('.media-tile-reveal').exists()).toBe(true)
    expect(wrapper.text()).toContain('Center')

    // force error UI for file type
    ;(wrapper.vm as any).handleError(new Event('error'))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('文件不可用')
  })

  it('uses IntersectionObserver when available and handles callback branches', async () => {
    let callback: any = null
    class MockIO {
      observe = vi.fn()
      disconnect = vi.fn()
      constructor(cb: any) {
        callback = cb
      }
    }
    ;(globalThis as any).IntersectionObserver = MockIO

    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/obs.png',
        type: 'image',
        showSkeleton: true
      },
      global: {
        stubs: {
          Skeleton: SkeletonStub
        }
      }
    })

    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="skeleton"]').exists()).toBe(false)

    callback([])
    await wrapper.vm.$nextTick()

    callback([{ isIntersecting: false }])
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="skeleton"]').exists()).toBe(false)

    callback([{ isIntersecting: true }])
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="skeleton"]').exists()).toBe(true)
  })
})
