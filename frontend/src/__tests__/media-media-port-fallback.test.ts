import { beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import MediaTile from '@/components/common/MediaTile.vue'
import { MEDIA_PORT_CANDIDATES, learnedMediaPort, resetLearnedMediaPort } from '@/utils/mediaPort'

const flushAsync = async () => {
  await Promise.resolve()
  await nextTick()
}

describe('MediaTile 上游端口回退', () => {
  beforeEach(() => {
    resetLearnedMediaPort()
  })

  it('加载失败时自动切到下一个候选端口，而不是直接显示错误', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://1.2.3.4:9006/img/Upload/a.png',
        type: 'image',
        lazy: false,
        showSkeleton: false
      }
    })
    await flushAsync()

    const img = wrapper.get('img')
    expect(img.attributes('src')).toBe('http://1.2.3.4:9006/img/Upload/a.png')

    await img.trigger('error')
    await flushAsync()

    // 换到候选表里第一个端口，且不进入错误态
    const nextSrc = wrapper.get('img').attributes('src')
    expect(nextSrc).toBe(`http://1.2.3.4:${MEDIA_PORT_CANDIDATES[0]}/img/Upload/a.png`)
    expect(wrapper.text()).not.toContain('图片加载失败')
    expect(wrapper.emitted('error')).toBeFalsy()
  })

  it('候选端口全部失败后才显示错误状态并抛出 error 事件', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://1.2.3.4:9006/img/Upload/a.png',
        type: 'image',
        lazy: false,
        showSkeleton: false
      }
    })
    await flushAsync()

    // 反复触发 error，耗尽所有候选端口
    for (let i = 0; i < MEDIA_PORT_CANDIDATES.length + 2; i += 1) {
      const img = wrapper.find('img')
      if (!img.exists()) break
      await img.trigger('error')
      await flushAsync()
    }

    expect(wrapper.text()).toContain('图片加载失败')
    expect(wrapper.emitted('error')?.length).toBe(1)
  })

  it('回退成功后记住可用端口，供其他媒体复用', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://1.2.3.4:9006/img/Upload/a.png',
        type: 'image',
        lazy: false,
        showSkeleton: false
      }
    })
    await flushAsync()

    await wrapper.get('img').trigger('error')
    await flushAsync()

    // 切到 7001 后“加载成功”
    await wrapper.get('img').trigger('load')
    await flushAsync()

    expect(learnedMediaPort.value).toBe(MEDIA_PORT_CANDIDATES[0])
  })

  it('非上游地址（本地 /upload、同源代理）失败时不做端口替换', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: '/upload/images/a.png',
        type: 'image',
        lazy: false,
        showSkeleton: false
      }
    })
    await flushAsync()

    await wrapper.get('img').trigger('error')
    await flushAsync()

    expect(wrapper.text()).toContain('图片加载失败')
    expect(wrapper.emitted('error')?.length).toBe(1)
  })

  it('换图后重置回退状态，重新从配置端口开始', async () => {
    const wrapper = mount(MediaTile, {
      props: {
        src: 'http://1.2.3.4:9006/img/Upload/a.png',
        type: 'image',
        lazy: false,
        showSkeleton: false
      }
    })
    await flushAsync()

    await wrapper.get('img').trigger('error')
    await flushAsync()
    expect(wrapper.get('img').attributes('src')).toBe(
      `http://1.2.3.4:${MEDIA_PORT_CANDIDATES[0]}/img/Upload/a.png`
    )

    // 换一张图：回退状态必须清空
    await wrapper.setProps({ src: 'http://1.2.3.4:9006/img/Upload/b.png' })
    await flushAsync()
    expect(wrapper.get('img').attributes('src')).toBe('http://1.2.3.4:9006/img/Upload/b.png')
  })
})
