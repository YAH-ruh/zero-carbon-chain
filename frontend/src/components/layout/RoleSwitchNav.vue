<!--
  RoleSwitchNav.vue
  顶部角色切换导航栏 —— 全局只挂一次，DOM 永不重绘
  角色切换通过 emit('switch', newRole) 通知父组件更新侧边菜单
-->
<template>
  <header class="role-switch-nav">
    <!-- 左侧 Logo -->
    <div class="rsn-left">
      <span class="rsn-logo">
        <img class="rsn-logo-mark" src="/favicon.svg" alt="零碳微证标识" />
        <span class="rsn-logo-em">零</span>碳微证
      </span>
      <span class="rsn-divider"></span>
      <span class="rsn-subtitle">碳积分可信交易平台</span>
    </div>

    <!-- 中间角色切换胶囊（4 个角色固定写死，DOM 永不变化） -->
    <nav class="rsn-chips">
      <button
        v-for="r in ROLE_META"
        :key="r.key"
        class="rsn-chip"
        :class="{ active: r.key === currentRole, disabled: busyKey === r.key }"
        :disabled="busyKey === r.key"
        @click="handleSwitch(r.key)"
      >
        <el-icon :size="14"><component :is="r.icon" /></el-icon>
        <span>{{ r.name }}</span>
      </button>
    </nav>

    <!-- 右侧用户区 -->
    <div class="rsn-right">
      <span class="rsn-who">{{ user?.company || user?.username }}</span>
      <button class="rsn-logout" @click="handleLogout">
        <el-icon :size="14"><SwitchButton /></el-icon>
        <span>退出</span>
      </button>
    </div>
  </header>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { User, OfficeBuilding, TrendCharts, Monitor, SwitchButton } from '@element-plus/icons-vue'

const props = defineProps({
  /** 当前登录用户信息 { role, username, company } */
  user: { type: Object, required: true }
})
const emit = defineEmits(['switch'])

const router = useRouter()

// 角色元信息（固定 4 项，模板写死，保证 DOM 稳定）
const ROLE_META = [
  { key: 'enterprise',  name: '小微企业',   icon: User },
  { key: 'park_admin',  name: '园区管理员', icon: OfficeBuilding },
  { key: 'exchange',    name: '碳交易所',   icon: TrendCharts },
  { key: 'regulator',   name: '监管核查',   icon: Monitor },
]

const currentRole = computed(() => props.user.role || '')
const busyKey = ref('')

// 点击角色胶囊：emit 给父组件处理（静默登录 + 跳转）
function handleSwitch(key) {
  if (busyKey.value) return
  if (key === currentRole.value) return
  busyKey.value = key
  emit('switch', key, () => { busyKey.value = '' })
}

function handleLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  router.push('/login')
}
</script>

<style scoped>
.role-switch-nav {
  flex-shrink: 0;
  height: var(--topbar-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 20px;
  background: var(--topbar-bg);
  border-bottom: 1px solid var(--topbar-border);
  backdrop-filter: blur(8px);
  z-index: 100;
  /* 关键：过渡动画让主题切换更平滑 */
  transition: background 0.3s ease, border-color 0.3s ease;
}

/* 左侧 Logo */
.rsn-left { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.rsn-logo {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--text-primary);
  cursor: pointer;
  white-space: nowrap;
}
.rsn-logo-mark { width: 24px; height: 24px; border-radius: 6px; box-shadow: 0 2px 6px rgba(5, 150, 105, 0.25); }
.rsn-logo-em { color: var(--primary-green); }
.rsn-divider {
  width: 1px;
  height: 16px;
  background: var(--line-strong);
}
.rsn-subtitle {
  font-size: 12px;
  color: var(--text-tertiary);
  white-space: nowrap;
}

/* 角色切换胶囊 */
.rsn-chips {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 auto;
}
.rsn-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 14px;
  border: 1px solid var(--line-strong);
  border-radius: 999px;
  background: var(--card-bg);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}
.rsn-chip:hover:not(.disabled) {
  border-color: var(--primary-green);
  color: var(--primary-green);
  background: var(--card-bg-hover);
}
.rsn-chip.active {
  border-color: var(--primary-green);
  background: var(--primary-green);
  color: #fff;
  box-shadow: 0 2px 8px rgba(22, 163, 74, 0.35);
}
.rsn-chip.disabled { opacity: 0.5; cursor: not-allowed; }

/* 右侧用户 */
.rsn-right { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.rsn-who {
  font-size: 13px;
  color: var(--text-secondary);
  white-space: nowrap;
}
.rsn-logout {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
  border: 1px solid var(--line-strong);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}
.rsn-logout:hover {
  border-color: var(--danger);
  color: var(--danger);
  background: rgba(220, 38, 38, 0.06);
}

/* 响应式 */
@media (max-width: 900px) {
  .rsn-subtitle { display: none; }
  .rsn-chips { flex-wrap: wrap; }
}
</style>