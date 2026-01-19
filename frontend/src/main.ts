import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import './index.css'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import App from './App.vue'

const initAppHeightCssVar = () => {
  if (typeof window === 'undefined' || typeof document === 'undefined') return

  let rafId: number | null = null

  const update = () => {
    const height = window.visualViewport?.height ?? window.innerHeight
    document.documentElement.style.setProperty('--app-height', `${Math.round(height)}px`)
  }

  const schedule = () => {
    if (rafId !== null) return
    if (typeof window.requestAnimationFrame !== 'function') {
      update()
      return
    }

    rafId = window.requestAnimationFrame(() => {
      rafId = null
      update()
    })
  }

  schedule()
  window.addEventListener('resize', schedule, { passive: true })
  window.addEventListener('orientationchange', schedule, { passive: true })
  window.visualViewport?.addEventListener('resize', schedule)
  window.visualViewport?.addEventListener('scroll', schedule)
}

initAppHeightCssVar()

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.mount('#app')
