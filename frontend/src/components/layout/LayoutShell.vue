<!--
  LayoutShell.vue
  整体布局壳 —— 保证 RoleSwitchNav 只挂载一次
  布局：
    RoleSwitchNav (固定顶部，DOM 永不重绘)
    ├── Sidebar (根据 role 过滤菜单)
    └── MainContent (<router-view />)
-->
<template>
  <div class="layout-shell" :class="{ fullscreen: isFullscreen }">
    <!-- ⭐ 顶部角色切换导航：只在 LayoutShell 顶层渲染一次，切换角色不触发 remount -->
    <RoleSwitchNav
      v-if="!isFullscreen"
      :user="user"
      @switch="handleSwitchRole"
      @logout="handleLogout"
    />

    <!-- 下方区域：Sidebar + MainContent -->
    <div class="body-area">
      <!-- 侧边栏：用 role 作为 key 强制角色切换时刷新菜单 -->
      <Sidebar
        v-if="!isFullscreen"
        :key="user.role"
        :role="user.role"
        @logout="handleLogout"
      />

      <!-- 主内容区：面包屑 + router-view -->
      <main class="main-content">
        <PageHeader v-if="!isFullscreen" :breadcrumb="breadcrumb" />
        <div class="page-scroll" :class="{ 'page-scroll--full': isFullscreen }">
          <router-view v-slot="{ Component }">
            <transition name="fade" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </div>
      </main>
    </div>

    <!-- AI 助手悬浮组件（全局，只在非登录/注册/门户页显示） -->
    <AiAssistant v-if="!isFullscreen && showAiAssistant" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import RoleSwitchNav from './RoleSwitchNav.vue'
import Sidebar from './Sidebar.vue'
import PageHeader from './PageHeader.vue'
import AiAssistant from '../AiAssistant.vue'
import { loginPreset, ROLE_ROUTES } from '../../utils/roleSwitch.js'

const router = useRouter()
const route = useRoute()

/* 当前登录用户（从 localStorage 读取） */
const user = ref(JSON.parse(localStorage.getItem('user') || '{}'))

/* 🌿 统一使用小微企业清新浅色主题，所有角色一致 */
watch(
  () => user.value.role,
  () => {
    document.documentElement.setAttribute('theme', 'light')
  },
  { immediate: true }
)

/* ====== 全屏大屏模式：route.meta.fullscreen 时隐藏所有侧边/头部 ====== */
const isFullscreen = computed(() => !!route.meta?.fullscreen)

/* 面包屑：从路由 meta 读取 */
const breadcrumb = computed(() => {
  const path = route.path.replace(/^\/+/, '').split('/').filter(Boolean)
  if (!path.length) return ['首页']
  const last = path[path.length - 1]
  return ['首页', formatKey(last)]
})
function formatKey(k) {
  const map = {
    'dashboard': '仪表盘',
    'datav': '数据中台大屏',
    'iot-devices': 'IoT设备',
    'energy-report': '能耗上报',
    'zkp-proof': '隐私核算',
    'credits-center': '碳积分中心',
    'exchange': '碳交易所',
    'pledge': '质押融资',
    'archive': '碳信用档案',
    'footprint': '碳足迹',
    'ai-assistant': 'AI减排助手',
    'ai-agent': 'AI交易Agent',
    'arbitration': '仲裁中心',
    'rollup-verify': 'Layer2校验',
    'incentive': '激励池',
    'permission': '权限管理',
  }
  return map[k] || k
}

/* AI 助手只在 LayoutShell 内的业务页面显示 */
const showAiAssistant = computed(() => true)

/* 切换角色：静默登录预置账号 + 更新本地状态 + 跳转 */
const busy = ref(false)
async function handleSwitchRole(role, done) {
  if (busy.value) { done && done(); return }
  busy.value = true
  try {
    const u = await loginPreset(role)
    user.value = u
    localStorage.setItem('user', JSON.stringify(u))
    router.push(ROLE_ROUTES[role])
  } catch (e) {
    alert('切换失败：' + (e?.msg || '网络错误'))
  } finally {
    busy.value = false
    done && done()
  }
}

function handleLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  document.documentElement.setAttribute('theme', 'light')
  router.push('/login')
}

onMounted(() => {
  // 🌿 强制全系统使用清新浅色主题
  document.documentElement.setAttribute('theme', 'light')
})
</script>

<style scoped>
.layout-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: var(--bg-primary);
}

.body-area {
  flex: 1;
  display: flex;
  min-height: 0; /* 关键：允许子元素正确滚动 */
  overflow: hidden;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.page-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px 40px;
  scrollbar-width: thin;
  scrollbar-color: var(--line-strong) transparent;
}
.page-scroll::-webkit-scrollbar { width: 6px; }
.page-scroll::-webkit-scrollbar-thumb {
  background: var(--line-strong);
  border-radius: 3px;
}

/* 页面切换过渡 */
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* 全屏大屏模式：清新浅绿 + 深绿点缀 */
.layout-shell.fullscreen {
  background:
    radial-gradient(ellipse 1200px 600px at 20% 0%, rgba(16, 185, 129, 0.08), transparent 55%),
    radial-gradient(ellipse 1200px 600px at 80% 100%, rgba(5, 150, 105, 0.06), transparent 55%),
    #f8fffb;
}
.layout-shell.fullscreen .body-area {
  width: 100vw;
}
.page-scroll--full {
  padding: 0 !important;
  background:
    radial-gradient(ellipse 1200px 600px at 20% 0%, rgba(16, 185, 129, 0.08), transparent 55%),
    radial-gradient(ellipse 1200px 600px at 80% 100%, rgba(5, 150, 105, 0.06), transparent 55%),
    #f8fffb;
}
</style>