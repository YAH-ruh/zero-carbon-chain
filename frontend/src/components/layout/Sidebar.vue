<!--
  Sidebar.vue
  左侧固定侧边栏 —— 根据当前角色过滤菜单项
  菜单项扁平化无嵌套，点击直接跳转路由
-->
<template>
  <aside class="sidebar">
    <!-- Logo 区 -->
    <div class="sidebar-logo">
      <div class="logo-icon">
        <img class="logo-mark" src="/favicon.svg" alt="零碳微证标识" />
      </div>
      <div class="logo-text">
        <span class="logo-title">小微企业</span>
        <span class="logo-sub">可信交易平台</span>
      </div>
    </div>

    <!-- 分隔线 -->
    <div class="sidebar-divider"></div>

    <!-- 菜单列表 -->
    <nav class="sidebar-menu">
      <div
        v-for="item in visibleMenu"
        :key="item.key"
        class="menu-item"
        :class="{ active: isActive(item.key) }"
        @click="go(item)"
      >
        <el-icon class="menu-icon"><component :is="item.icon" /></el-icon>
        <span class="menu-label">{{ item.name }}</span>
        <span v-if="item.badge" class="menu-badge">{{ item.badge }}</span>
      </div>

      <!-- 空状态：当前角色没有可见菜单 -->
      <div v-if="!visibleMenu.length" class="menu-empty">
        <el-icon :size="28" color="#64748b"><Warning /></el-icon>
        <p>当前角色无可用菜单</p>
      </div>
    </nav>

    <!-- 底部退出按钮 -->
    <div class="sidebar-footer">
      <button class="sidebar-logout" @click="emit('logout')">
        <el-icon :size="14"><SwitchButton /></el-icon>
        <span>退出登录</span>
      </button>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAiAssistantStore } from '../../stores/aiAssistant.js'
import {
  DataBoard, Cpu, EditPen, Lock, Wallet, TrendCharts,
  Money, Collection, Document, MagicStick, Monitor, ScaleToOriginal,
  Tickets, Medal, Key, Warning, SwitchButton
} from '@element-plus/icons-vue'

const props = defineProps({
  /** 当前角色 */
  role: { type: String, required: true }
})
const emit = defineEmits(['logout'])

const router = useRouter()
const route = useRoute()
const aiStore = useAiAssistantStore()

/* ============================================================
   17 个菜单项权限矩阵（图标用 Element Plus Icons）
   roles 数组表示哪些角色可见
   ============================================================ */
const ALL_MENU = [
  { key: 'dashboard',     name: '首页仪表盘',         icon: DataBoard,    roles: ['enterprise','park_admin','exchange','regulator'] },
  { key: 'iot-devices',   name: 'IoT能耗设备管理',    icon: Cpu,          roles: ['enterprise'] },
  { key: 'energy-report', name: '能耗上报',           icon: EditPen,      roles: ['enterprise'] },
  { key: 'zkp-proof',     name: '隐私核算(ZKP证明)',  icon: Lock,         roles: ['enterprise','regulator'] },
  { key: 'credits-center',name: '碳积分资产中心',     icon: Wallet,       roles: ['enterprise','park_admin','exchange','regulator'] },
  { key: 'exchange',      name: '碳交易所',           icon: TrendCharts,  roles: ['exchange'] },
  { key: 'pledge',        name: '碳资产质押融资',     icon: Money,        roles: ['enterprise'] },
  { key: 'archive',       name: '碳信用档案',         icon: Collection,   roles: ['exchange','regulator'] },
  { key: 'footprint',     name: '产品碳足迹管理',     icon: Document,     roles: ['enterprise'] },
  { key: 'ai-assistant',  name: 'AI减排助手',         icon: MagicStick,   roles: ['enterprise','park_admin'] },
  { key: 'ai-agent',      name: 'AI交易Agent管理',    icon: Monitor,      roles: ['exchange','regulator'] },
  { key: 'arbitration',   name: '交易仲裁中心',       icon: ScaleToOriginal, roles: ['enterprise','exchange','regulator'] },
  { key: 'rollup-verify', name: 'Layer2交易证明校验', icon: Tickets,      roles: ['regulator'] },
  { key: 'incentive',     name: '链上激励池管理',     icon: Medal,        roles: ['exchange','park_admin'] },
  { key: 'datav',         name: '数据中台大屏',       icon: Monitor,      roles: ['park_admin','regulator'] },
  { key: 'permission',    name: '系统权限管理',       icon: Key,          roles: ['regulator'] },
]

/* 根据当前角色过滤菜单 */
const visibleMenu = computed(() => {
  return ALL_MENU.filter(m => m.roles.includes(props.role))
})

/* 判断当前路由是否高亮 */
function isActive(key) {
  return route.path.startsWith('/' + key)
}

/* 点击跳转 */
function go(item) {
  // "AI减排助手" → 不打开独立页面，直接展开右下角悬浮对话组件
  if (item.key === 'ai-assistant') {
    aiStore.openAssistant()
    return
  }
  router.push('/' + item.key)
}
</script>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  flex-shrink: 0;
  height: 100%;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--sidebar-border);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: var(--sidebar-scrollbar) transparent;
}
.sidebar::-webkit-scrollbar { width: 4px; }
.sidebar::-webkit-scrollbar-thumb { background: var(--sidebar-scrollbar); border-radius: 2px; }

/* Logo 区 */
.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 16px 16px;
  cursor: default;
}
.logo-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--primary-green), var(--primary-green-light));
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  box-shadow: var(--sidebar-logo-shadow);
  flex-shrink: 0;
  overflow: hidden;
}
.logo-mark { width: 100%; height: 100%; border-radius: inherit; display: block; }
.logo-text { display: flex; flex-direction: column; min-width: 0; }
.logo-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--sidebar-title);
  letter-spacing: 0.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.logo-sub {
  font-size: 11px;
  color: var(--sidebar-sub);
  letter-spacing: 0.5px;
}

.sidebar-divider {
  height: 1px;
  margin: 4px 12px 8px;
  background: var(--sidebar-border);
}

/* 菜单 */
.sidebar-menu {
  flex: 1;
  padding: 4px 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  color: var(--sidebar-text);
  font-size: 13px;
  transition: all 0.2s ease;
  position: relative;
  white-space: nowrap;
}
.menu-item:hover {
  background: var(--sidebar-hover);
  color: var(--sidebar-hover-text);
}
.menu-item.active {
  background: var(--sidebar-bg-active);
  color: var(--sidebar-text-active);
  box-shadow: inset 3px 0 0 var(--primary-green);
}
.menu-item.active::before {
  content: '';
  position: absolute;
  left: -1px;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  border-radius: 2px;
  background: var(--primary-green);
}

.menu-icon {
  font-size: 16px;
  width: 20px;
  flex-shrink: 0;
}
.menu-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.menu-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 999px;
  background: rgba(239, 68, 68, 0.85);
  color: #fff;
  font-weight: 500;
}

.menu-empty {
  text-align: center;
  padding: 40px 16px;
  color: var(--text-tertiary);
}
.menu-empty p { margin-top: 8px; font-size: 12px; }

/* 底部退出 */
.sidebar-footer {
  padding: 12px;
  border-top: 1px solid var(--sidebar-border);
}
.sidebar-logout {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  padding: 10px;
  border: 1px solid var(--sidebar-logout-border);
  border-radius: 8px;
  background: transparent;
  color: var(--sidebar-text);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}
.sidebar-logout:hover {
  background: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.4);
  color: var(--danger);
}
</style>