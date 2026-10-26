import { createApp } from 'vue'
import { createPinia } from 'pinia';
import router from './router';
import ArcoVue from '@arco-design/web-vue';
import '@arco-design/web-vue/dist/arco.css';
import './style.css'
import App from './App.vue'

document.addEventListener('contextmenu', (event) => {
  const path = typeof event.composedPath === 'function' ? event.composedPath() : [];
  const isImageContext = path.some((node) => node?.tagName === 'IMG') || event.target?.tagName === 'IMG';
  if (isImageContext) {
    event.preventDefault();
    event.stopPropagation();
  }
}, true);

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);
app.use(ArcoVue);
app.mount('#app');
