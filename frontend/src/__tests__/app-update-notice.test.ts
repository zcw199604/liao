import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { expect, it, vi } from 'vitest'
import AppUpdateNotice from '@/components/common/AppUpdateNotice.vue'

const watcher = vi.hoisted(() => ({ callback: undefined as ((reason: 'version' | 'load-error') => void) | undefined, stop: vi.fn() }))
vi.mock('@/utils/appUpdate', () => ({
  watchAppUpdates: (callback: (reason: 'version' | 'load-error') => void) => {
    watcher.callback = callback
    return watcher.stop
  }
}))

it('shows a refresh action only after an update and stops monitoring on unmount', async () => {
  const wrapper = mount(AppUpdateNotice)
  expect(wrapper.find('[role="status"]').exists()).toBe(false)
  watcher.callback?.('version')
  await nextTick()
  expect(wrapper.get('[role="status"]').text()).toContain('页面已更新')
  expect(wrapper.get('button').text()).toBe('刷新')
  wrapper.unmount()
  expect(watcher.stop).toHaveBeenCalledOnce()
})
