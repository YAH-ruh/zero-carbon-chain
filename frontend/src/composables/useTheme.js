/**
 * useTheme.js
 * 全局主题 composable —— 监听 document.documentElement.setAttribute('theme', ...)
 * 提供响应式的 theme ref
 */
import { ref, onMounted, onUnmounted } from 'vue'

const theme = ref('light')
let initialized = false

export function useTheme() {
  if (!initialized) {
    initialized = true
    theme.value = document.documentElement.getAttribute('theme') || 'light'

    // 监听 attribute 变化
    const observer = new MutationObserver(() => {
      theme.value = document.documentElement.getAttribute('theme') || 'light'
    })
    onMounted(() => {
      observer.observe(document.documentElement, { attributes: true, attributeFilter: ['theme'] })
    })
    onUnmounted(() => observer.disconnect())
  }

  function setTheme(t) {
    document.documentElement.setAttribute('theme', t)
    theme.value = t
  }

  return { theme, setTheme, isDark: () => theme.value === 'dark' }
}
