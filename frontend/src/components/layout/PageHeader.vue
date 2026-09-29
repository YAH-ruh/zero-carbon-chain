<!--
  PageHeader.vue
  页面顶部窄条：面包屑
-->
<template>
  <div class="page-header">
    <div class="breadcrumb">
      <template v-for="(item, idx) in breadcrumb" :key="idx">
        <span
          class="crumb-item"
          :class="{ active: idx === breadcrumb.length - 1 }"
          @click="idx !== breadcrumb.length - 1 && goHome()"
        >{{ item }}</span>
        <el-icon v-if="idx < breadcrumb.length - 1" class="crumb-sep" :size="12">
          <ArrowRight />
        </el-icon>
      </template>
    </div>
    <div class="page-actions">
      <button class="header-icon-btn" title="刷新" @click="refreshPage">
        <el-icon :size="16"><Refresh /></el-icon>
      </button>
      <div class="user-avatar" :title="user?.company || user?.username">
        <el-icon :size="14"><User /></el-icon>
      </div>
    </div>
  </div>
</template>

<script setup>
import { inject } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, Refresh, User } from '@element-plus/icons-vue'

defineProps({
  breadcrumb: { type: Array, default: () => [] }
})

const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{}')

function goHome() {
  const role = user.role
  const map = {
    enterprise: '/dashboard',
    park_admin: '/dashboard',
    exchange: '/dashboard',
    regulator: '/dashboard'
  }
  router.push(map[role] || '/dashboard')
}
function refreshPage() { window.location.reload() }
</script>

<style scoped>
.page-header {
  flex-shrink: 0;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--line-color);
}
.breadcrumb { display: flex; align-items: center; gap: 4px; font-size: 13px; }
.crumb-item { color: var(--text-tertiary); cursor: pointer; transition: color 0.15s; }
.crumb-item:hover { color: var(--primary-green); }
.crumb-item.active { color: var(--text-primary); font-weight: 500; cursor: default; }
.crumb-item.active:hover { color: var(--text-primary); }
.crumb-sep { color: var(--text-tertiary); }

.page-actions { display: flex; align-items: center; gap: 8px; }
.header-icon-btn {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--line-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}
.header-icon-btn:hover { border-color: var(--primary-green); color: var(--primary-green); }

.user-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary-green), #22c55e);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: default;
}
</style>