import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import './index.css'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import App from './App.vue'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.mount('#app')
