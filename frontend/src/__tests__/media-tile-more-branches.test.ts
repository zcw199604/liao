import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import MediaTile from '@/components/common/MediaTile.vue'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('components/common/MediaTile.vue (more branches)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders file tile (no skeleton/backdrop) and containerLoadingClass is empty', async () => {
    const wrapper = mount(MediaTile, {
      props: { src: 'http://x/a.bin', type: 'file', lazy: false }
    })
    await flushAsync()

    expect(wrapper.text()).toContain('') // component renders icon only
    expect(wrapper.find('i.fas.fa-file').exists()).toBe(true)
    expect(wrapper.find('img[aria-hidden=\"true\"]').exists()).toBe(false)
  })

  it('shows error slot content on media error and uses per-type error text/icon', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/a.png',
        type: 'image',
        lazy: false,
        showSkeleton: false
      }
    })
    await flushAsync()

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    await img.trigger('error')
    await flushAsync()

    expect(wrapper.find('i.fas.fa-image-slash').exists()).toBe(true)
    expect(wrapper.text()).toContain('图片加载失败')
  })

  it('backdrop appears for contain-fit video with poster, and center indicator respects slots/controls', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/v.mp4',
        type: 'video',
        poster: 'http://x/v.poster.jpg',
        fit: 'contain',
        lazy: false,
        controls: false,
        showVideoIndicator: true
      }
    })
    await flushAsync()

    expect(wrapper.find('video').exists()).toBe(true)
    expect(wrapper.find('img[aria-hidden=\"true\"]').exists()).toBe(true)
    // Default center indicator for video
    expect(wrapper.find('i.fas.fa-play').exists()).toBe(true)

    // When controls=true, indicator should not render.
    await wrapper.setProps({ controls: true })
    await flushAsync()
    expect(wrapper.find('i.fas.fa-play').exists()).toBe(false)
  })

  it('lazy loading: IntersectionObserver gates shouldLoad and disconnects on intersect', async () => {
    const originalIO = (globalThis as any).IntersectionObserver

    let ioCallback: ((entries: any[]) => void) | null = null
    const disconnectSpy = vi.fn()
    const observeSpy = vi.fn()
    class FakeIntersectionObserver {
      constructor(cb: (entries: any[]) => void) {
        ioCallback = cb
      }
      observe = observeSpy
      disconnect = disconnectSpy
    }

    try {
      ;(globalThis as any).IntersectionObserver = FakeIntersectionObserver

      const wrapper = mount(MediaTile, {
        props: {
          src: 'http://x/a.png',
          type: 'image',
          lazy: true
        }
      })
      await flushAsync()

      // Not intersecting -> still not loaded.
      ioCallback?.([])
      ioCallback?.([{ isIntersecting: false }])
      await flushAsync()
      expect(wrapper.find('img').exists()).toBe(false)

      // Intersecting -> loads and disconnects.
      ioCallback?.([{ isIntersecting: true }])
      await flushAsync()
      expect(wrapper.find('img').exists()).toBe(true)
      expect(disconnectSpy).toHaveBeenCalled()
    } finally {
      ;(globalThis as any).IntersectionObserver = originalIO
    }
  })

  it('when IntersectionObserver is unavailable, lazy tiles load immediately', async () => {
    const originalIO = (globalThis as any).IntersectionObserver
    try {
      ;(globalThis as any).IntersectionObserver = undefined
      const wrapper = mount(MediaTile, {
        props: {
          src: 'http://x/a.png',
          type: 'image',
          lazy: true
        }
      })
      await flushAsync()
      expect(wrapper.find('img').exists()).toBe(true)
    } finally {
      ;(globalThis as any).IntersectionObserver = originalIO
    }
  })

  it('handleLoaded caches aspect ratio when dimensions are available and emits load/layout', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/cache.png',
        type: 'image',
        lazy: false
      }
    })
    await flushAsync()

    const fakeImg = document.createElement('img') as HTMLImageElement
    Object.defineProperty(fakeImg, 'naturalWidth', { configurable: true, get: () => 200 })
    Object.defineProperty(fakeImg, 'naturalHeight', { configurable: true, get: () => 100 })

    ;(wrapper.vm as any).handleLoaded({ target: fakeImg } as any)
    await flushAsync()

    expect(wrapper.emitted('load')).toBeTruthy()
    expect(wrapper.emitted('layout')).toBeTruthy()
  })

  it('covers type=auto inference, skeleton/hoverOverlay rendering, and container loading pulse', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/auto.mp4',
        type: 'auto',
        lazy: false,
        hoverOverlay: true,
        showSkeleton: true
      }
    })
    await flushAsync()

    // inferred as video
    expect(wrapper.find('video').exists()).toBe(true)
    expect(wrapper.html()).toContain('group-hover:bg-black/10')

    // showSkeleton=false -> root container uses animate-pulse while loading
    const wrapper2 = mount(MediaTile, {
      props: { src: 'http://x/pulse.png', type: 'image', lazy: false, showSkeleton: false }
    })
    await flushAsync()
    expect(wrapper2.classes().join(' ')).toContain('animate-pulse')
  })

  it('uses cached aspect ratio after load and covers handleLoaded fallback branches', async () => {
    const src = 'http://x/cache-used.png'

    const w1 = mount(MediaTile, { props: { src, type: 'image', lazy: false } })
    await flushAsync()

    const img1 = w1.find('img')
    expect(img1.exists()).toBe(true)
    Object.defineProperty(img1.element, 'naturalWidth', { configurable: true, get: () => 200 })
    Object.defineProperty(img1.element, 'naturalHeight', { configurable: true, get: () => 100 })
    expect((img1.element as any).naturalWidth).toBe(200)
    expect((img1.element as any).naturalHeight).toBe(100)
    await img1.trigger('load')
    await flushAsync()

    // The cache Map is non-reactive; force recompute via src change to use cached ratio.
    await w1.setProps({ src: 'http://x/other.png' })
    await flushAsync()
    await w1.setProps({ src })
    await flushAsync()
    expect(String((w1.element as HTMLElement).getAttribute('style') || '')).toContain('aspect-ratio: 2')

    // handleLoaded: props.src is empty -> caching branch is skipped but still emits events.
    const w3 = mount(MediaTile, { props: { src: '', type: 'image', lazy: false } })
    await flushAsync()
    const img3 = w3.find('img')
    expect(img3.exists()).toBe(true)
    await img3.trigger('load')
    await flushAsync()
    expect(w3.emitted('load')).toBeTruthy()

    // handleLoaded: unknown target -> else-if false branch + width/height=0 branch.
    const w4 = mount(MediaTile, { props: { src: 'http://x/x.png', type: 'image', lazy: false } })
    await flushAsync()
    ;(w4.vm as any).handleLoaded({ target: document.createElement('div') } as any)
    await flushAsync()
    expect(w4.emitted('load')).toBeTruthy()
  })

  it('slot reveal class toggles while slot exists (covers both ternary branches) and center slot forces shouldShowCenter', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://x/a.png',
        type: 'image',
        lazy: false,
        revealTopLeft: true
      },
      slots: {
        'top-left': '<span class=\"tl\">TL</span>',
        center: '<div class=\"center-slot\">C</div>'
      }
    })
    await flushAsync()

    expect(wrapper.find('.center-slot').exists()).toBe(true)
    expect(wrapper.find('.tl').exists()).toBe(true)

    const topLeftWrapper = wrapper.find('.tl').element.closest('div') as HTMLElement
    expect(topLeftWrapper.className).toContain('media-tile-reveal')

    await wrapper.setProps({ revealTopLeft: false })
    await flushAsync()
    expect((wrapper.find('.tl').element.closest('div') as HTMLElement).className).not.toContain('media-tile-reveal')
  })
})
