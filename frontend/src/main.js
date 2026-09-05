import { createApp } from 'vue'
import App from './App.vue'
import router from './router/index.js'
// 全局注册 ElementPlus 矢量图标(轻量 SVG，不含组件库样式)，满足"图标统一使用 ElementPlus"规范
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

const app = createApp(App)

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(router)
app.mount('#app')
