import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

const toastShow = vi.fn()

const uploadMocks = {
  uploadFile: vi.fn()
}

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ show: toastShow })
}))

vi.mock('@/composables/useUpload', () => ({
  useUpload: () => uploadMocks
}))

vi.mock('@/stores/videoExtract', () => ({
  useVideoExtractStore: () => ({ openCreateFromMedia: vi.fn() })
}))

vi.mock('plyr', () => {
  class MockPlyr {
    media: any
    speed = 1
    volume = 1
    currentTime = 0
    fullscreen = { active: false, toggle: () => {}, exit: () => {} }
    constructor(media: any) {
      this.media = media
    }
    play() {
      return Promise.resolve()
    }
    pause() {}
    destroy() {}
  }
  return { default: MockPlyr }
})

vi.mock('@/api/mtphoto', () => ({
  getSameMedia: vi.fn(),
  openFolder: vi.fn()
}))

import MediaPreview from '@/components/media/MediaPreview.vue'
import { MEDIA_PORT_CANDIDATES, resetLearnedMediaPort } from '@/utils/mediaPort'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  setActivePinia(createPinia())
  resetLearnedMediaPort()
})

const mountPreview = (url: string) =>
  mount(MediaPreview, {
    props: {
      visible: true,
      url,
      type: 'image',
      mediaList: [{ url, type: 'image' }]
    },
    global: {
      stubs: {
        teleport: true,
        MediaDetailPanel: { template: '<div />' }
      }
    }
  })

describe('MediaPreview 上游端口回退', () => {
  it('预览大图加载失败时先换端口，而不是先走 cache-buster 重试', async () => {
    const wrapper = mountPreview('http://1.2.3.4:9006/img/Upload/a.png')
    await flushAsync()

    const img = wrapper.get('img[alt="预览"]')
    expect(img.attributes('src')).toBe('http://1.2.3.4:9006/img/Upload/a.png')

    await img.trigger('error')
    await flushAsync()

    expect(wrapper.get('img[alt="预览"]').attributes('src')).toBe(
      `http://1.2.3.4:${MEDIA_PORT_CANDIDATES[0]}/img/Upload/a.png`
    )
  })

  it('候选端口耗尽后才回落到 cache-buster 重试', async () => {
    vi.useFakeTimers()
    const wrapper = mountPreview('http://1.2.3.4:9006/img/Upload/a.png')
    await flushAsync()

    // 起始端口 9006 也在候选表内：候选耗尽恰好需要 12 次失败，第 13 次才回落到重试
    for (let i = 0; i < MEDIA_PORT_CANDIDATES.length; i += 1) {
      const img = wrapper.find('img[alt="预览"]')
      if (!img.exists()) break
      await img.trigger('error')
      await flushAsync()
    }

    const vm = wrapper.vm as any
    expect(vm.mediaRetryCount).toBe(1)

    // 重试由定时器触发：推进时间后应带上 cache-buster
    await vi.advanceTimersByTimeAsync(600)
    await flushAsync()
    expect(String(vm.currentMediaDisplayUrl)).toMatch(/\/img\/Upload\/a\.png\?_=\d+$/)

    vi.useRealTimers()
  })

  it('非上游地址不做端口替换', async () => {
    const wrapper = mountPreview('http://x/1.png')
    await flushAsync()

    await wrapper.get('img[alt="预览"]').trigger('error')
    await flushAsync()

    expect(wrapper.get('img[alt="预览"]').attributes('src')).toBe('http://x/1.png')
  })
})
