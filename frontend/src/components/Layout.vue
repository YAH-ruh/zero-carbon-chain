<template>
  <!-- Layout：顶部轻量栏(身份切换胶囊按钮全页面通用) + 内容承载。
       页面间业务跳转由各页面内部按钮完成(vue-router 真实路由跳转)；
       顶部栏提供 4 个角色胶囊，点击即静默切换预置账号并跳转到对应角色首页。 -->
  <div class="layout">
    <!-- 粒子光影氛围背景(纯装饰) -->
    <ParticleBg :density="26" />

    <header class="topbar">
      <div class="top-left">
        <span class="logo" @click="goRoleHome">微碳链</span>
        <span class="divider"></span>
        <span class="subtitle">碳积分可信交易平台 · {{ roleLabel }}</span>
      </div>

      <!-- 身份切换胶囊：4 角色快捷入口(当前角色高亮；点击其他角色静默登录后跳转) -->
      <nav class="role-switch">
        <button
          v-for="r in ROLE_META"
          :key="r.key"
          class="rs-chip"
          :class="{ active: r.key === currentRole, disabled: busyKey === r.key }"
          :disabled="busyKey === r.key"
          @click="switchRole(r.key)"
        >
          <component :is="r.icon" />
          <span>{{ r.name }}</span>
        </button>
      </nav>

      <div class="top-right">
        <span class="who">{{ user?.company || user?.username }}</span>
        <button class="btn-logout" @click="handleLogout"><SwitchButton /> 退出</button>
      </div>
    </header>

    <main class="content">
      <slot />
    </main>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import ParticleBg from './ParticleBg.vue'
import { ROLE_META, ROLE_ROUTES, loginPreset } from '../utils/roleSwitch.js'

const router = useRouter()
const route = useRoute()
const user = JSON.parse(localStorage.getItem('user') || '{}')

const currentRole = computed(() => user.role || '')

// 角色正式展示名(仅展示，不改后端角色标识)
const roleMap = {
  enterprise: '小微企业',
  park_admin: '园区管理员',
  exchange: '碳交易所',
  regulator: '监管核查'
}
const roleLabel = computed(() => roleMap[user.role] || '未知角色')

// 顶部品牌点击：回到当前角色首页(仅内部导航)
function goRoleHome() {
  const base = ROLE_ROUTES[currentRole.value]
  if (base && route.path !== base) router.push(base)
}

// 顶部身份胶囊：点击其他角色 -> 静默登录预置账号 -> 跳转对应角色首页；当前角色 -> 回到本角色首页
const busyKey = ref('')
async function switchRole(key) {
  if (busyKey.value) return
  if (key === currentRole.value) return goRoleHome()
  busyKey.value = key
  try {
    const u = await loginPreset(key)
    user.role = u.role
    router.push(ROLE_ROUTES[key])
  } catch (e) {
    alert('切换失败：' + (e?.msg || '网络错误'))
  } finally {
    busyKey.value = ''
  }
}

function handleLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  router.push('/login')
}
</script>

<style scoped>
.layout { display: flex; flex-direction: column; height: 100vh; position: relative; }
.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 0 18px;
  height: 54px;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(6px);
  border-bottom: 1px solid var(--line);
  flex-shrink: 0;
  z-index: 10;
}
.top-left { display: flex; align-items: center; gap: 12px; min-width: 0; flex-shrink: 0; }
.logo { font-size: 17px; font-weight: 700; letter-spacing: 0.03em; color: var(--text-1); cursor: pointer; white-space: nowrap; }
.logo::first-letter { color: var(--primary); }
.divider { width: 1px; height: 16px; background: var(--line-strong); }
.subtitle { font-size: 12px; color: var(--text-4); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

/* 身份切换胶囊(全页面通用) */
.role-switch { display: flex; align-items: center; gap: 6px; margin: 0 auto; }
.rs-chip {
  display: inline-flex; align-items: center; gap: 4px;
  border: 1px solid var(--line-strong); background: #fff; color: var(--text-3);
  border-radius: 999px; padding: 4px 11px;
  cursor: pointer; font-size: 12px; white-space: nowrap;
  transition: all 0.15s ease;
}
.rs-chip svg { width: 13px; height: 13px; }
.rs-chip:hover { border-color: #99f6e4; color: #0f766e; background: #f0fdfa; }
.rs-chip.active {
  border-color: #0f766e; color: #fff; background: #0f766e; font-weight: 500;
  box-shadow: 0 2px 8px rgba(13, 148, 136, 0.3);
}
.rs-chip.disabled { opacity: 0.55; cursor: not-allowed; }

.top-right { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.who { font-size: 13px; color: var(--text-2); white-space: nowrap; }
.btn-logout {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 6px 13px;
  border: 1px solid var(--line-strong);
  border-radius: var(--radius-sm);
  background: #fff; color: var(--text-3);
  cursor: pointer; font-size: 13px;
  transition: all 0.15s ease; white-space: nowrap;
}
.btn-logout svg { width: 14px; height: 14px; }
.btn-logout:hover { border-color: #fca5a5; color: var(--danger); background: #fef2f2; }
.content { flex: 1; overflow-y: auto; padding: 22px 24px 40px; background: transparent; position: relative; z-index: 1; }

@media (max-width: 1080px) {
  .role-switch { order: 3; width: 100%; justify-content: center; padding: 6px 0 2px; flex-wrap: wrap; }
  .topbar { flex-wrap: wrap; height: auto; padding: 6px 14px; }
}
@media (max-width: 560px) {
  .subtitle { display: none; }
  .who { display: none; }
}
</style>
