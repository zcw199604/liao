import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import MediaDetailPanel from '@/components/media/MediaDetailPanel.vue'

describe('components/media/MediaDetailPanel.vue', () => {
  it('renders all fields when provided and closes via close button/backdrop', async () => {
    const wrapper = mount(MediaDetailPanel, {
      props: {
        visible: true,
        media: {
          url: 'http://x/a.png',
          type: 'image',
          originalFilename: '中文.png',
          localFilename: 'local.png',
          fileSize: 1024,
          fileExtension: 'png',
          fileType: 'image/png',
          width: 100,
          height: 200,
          duration: 12,
          day: '2026-01-01',
          uploadTime: '2026-01-01 00:00:00',
          updateTime: '2026-01-01 00:00:01',
          md5: 'm1',
          pHash: 'p1',
          similarity: 0.1234
        } as any
      },
      global: { stubs: { teleport: true } }
    })

    expect(wrapper.text()).toContain('文件详细信息')
    expect(wrapper.text()).toContain('原始文件名')
    expect(wrapper.text()).toContain('中文.png')
    expect(wrapper.text()).toContain('本地存储名')
    expect(wrapper.text()).toContain('local.png')
    expect(wrapper.text()).toContain('文件大小')
    expect(wrapper.text()).toContain('文件格式')
    expect(wrapper.text()).toContain('PNG')
    expect(wrapper.text()).toContain('(image/png)')
    expect(wrapper.text()).toContain('分辨率')
    expect(wrapper.text()).toContain('100 × 200')
    expect(wrapper.text()).toContain('时长')
    expect(wrapper.text()).toContain('12s')
    expect(wrapper.text()).toContain('日期')
    expect(wrapper.text()).toContain('2026-01-01')
    expect(wrapper.text()).toContain('MD5')
    expect(wrapper.text()).toContain('m1')
    expect(wrapper.text()).toContain('pHash')
    expect(wrapper.text()).toContain('p1')
    expect(wrapper.text()).toContain('相似度')
    expect(wrapper.text()).toContain('12.34%')

    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('update:visible')?.some((e) => e[0] === false)).toBe(true)

    await wrapper.get('div.absolute').trigger('click')
    expect(wrapper.emitted('update:visible')?.length).toBeGreaterThanOrEqual(2)
  })

  it('hides optional sections when fields are missing', () => {
    const wrapper = mount(MediaDetailPanel, {
      props: {
        visible: true,
        media: { url: '/x', type: 'file' } as any
      },
      global: { stubs: { teleport: true } }
    })

    expect(wrapper.text()).toContain('文件详细信息')
    expect(wrapper.text()).not.toContain('原始文件名')
    expect(wrapper.text()).not.toContain('文件大小')
    expect(wrapper.text()).not.toContain('分辨率')
    expect(wrapper.text()).not.toContain('MD5')
  })

  it('does not render when not visible', () => {
    const wrapper = mount(MediaDetailPanel, {
      props: {
        visible: false,
        media: { url: '/x', type: 'file' } as any
      },
      global: { stubs: { teleport: true } }
    })

    expect(wrapper.html()).not.toContain('文件详细信息')
  })
})
