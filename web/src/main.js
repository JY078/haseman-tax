// Polyfill for crypto.randomUUID (for older browsers / non-secure contexts)
if (typeof crypto.randomUUID !== 'function') {
  crypto.randomUUID = () => {
    return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, c =>
      (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16)
    )
  }
}

import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { RecycleScroller, DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import App from './App.vue'
import router, { startMaintenancePolling } from './router'
import { useAuthStore } from './stores/auth'

document.addEventListener('contextmenu', (event) => {
  const path = typeof event.composedPath === 'function' ? event.composedPath() : []
  const isImageContext = path.some((node) => node?.tagName === 'IMG') || event.target?.tagName === 'IMG'
  if (isImageContext) {
    event.preventDefault()
    event.stopPropagation()
  }
}, true)

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)

const authStore = useAuthStore()
authStore.init()

app.use(router)
app.component('RecycleScroller', RecycleScroller)
app.component('DynamicScroller', DynamicScroller)
app.component('DynamicScrollerItem', DynamicScrollerItem)

app.mount('#app')
startMaintenancePolling()
