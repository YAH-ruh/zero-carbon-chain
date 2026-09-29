import { createApp } from 'vue'
import App from './App.vue'
import router from './router/index.js'
import { createPinia } from 'pinia'

/* Element Plus 全量引入（方便快速开发生产时按需优化） */
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

/* 全局注册 Element Plus 图标 */
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

/* 主题变量 + 基础样式 */
import './assets/styles/variables.css'

const app = createApp(App)
const pinia = createPinia()

// 注册所有 Element Plus 图标组件
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus)
app.use(pinia)
app.use(router)

app.mount('#app')
