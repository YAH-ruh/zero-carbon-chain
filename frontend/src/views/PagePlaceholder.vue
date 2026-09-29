<!--
  PagePlaceholder.vue
  通用页面占位：后续逐个替换为真实实现
  props.title: 页面标题
  props.icon: 图标名
-->
<template>
  <div class="placeholder">
    <div class="ph-card">
      <div class="ph-icon">
        <el-icon :size="40"><component :is="icon" /></el-icon>
      </div>
      <h2 class="ph-title">{{ title }}</h2>
      <p class="ph-subtitle">页面开发中 · 后续接入真实业务功能</p>
      <div class="ph-bar">
        <span>Step 1 已完成：布局骨架 + 主题 + 侧边栏</span>
        <el-icon :size="14"><Loading /></el-icon>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  DataBoard, Monitor, EditPen, Cpu, Lock, Wallet, TrendCharts,
  Money, Collection, Document, MagicStick, ScaleToOriginal, Tickets, Medal, Key, PieChart, DataLine
} from '@element-plus/icons-vue'

const props = defineProps({
  title: { type: String, default: '' }
})
const route = useRoute()

const meta = {
  'datav':         { title: '数据中台大屏',       icon: Monitor },
  'energy-report': { title: '能耗上报',           icon: EditPen },
  'iot-devices':   { title: 'IoT能耗设备管理',    icon: Cpu },
  'zkp-proof':     { title: '隐私核算(ZKP证明)',  icon: Lock },
  'credits-center':{ title: '碳积分资产中心',     icon: Wallet },
  'exchange':      { title: '碳交易所',           icon: TrendCharts },
  'pledge':        { title: '碳资产质押融资',     icon: Money },
  'archive':       { title: '碳信用档案',         icon: Collection },
  'footprint':     { title: '产品碳足迹管理',     icon: Document },
  'ai-assistant':  { title: 'AI减排助手',         icon: MagicStick },
  'ai-agent':      { title: 'AI交易Agent管理',    icon: MagicStick },
  'arbitration':   { title: '交易仲裁中心',       icon: ScaleToOriginal },
  'rollup-verify': { title: 'Layer2交易证明校验', icon: Tickets },
  'incentive':     { title: '链上激励池管理',     icon: Medal },
  'permission':    { title: '系统权限管理',       icon: Key },
}

const resolved = computed(() => {
  const key = route.path.replace(/^\//, '')
  return meta[key] || { title: props.title || '页面', icon: DataBoard }
})
const title = computed(() => resolved.value.title)
const icon  = computed(() => resolved.value.icon)
</script>

<style scoped>
.placeholder { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.ph-card {
  text-align: center;
  padding: 80px 40px;
  background: var(--card-bg);
  border: 1px dashed var(--line-strong);
  border-radius: var(--radius-lg);
}
.ph-icon {
  width: 80px; height: 80px;
  margin: 0 auto 20px;
  display: flex; align-items: center; justify-content: center;
  background: rgba(22, 163, 74, 0.08);
  color: var(--primary-green);
  border-radius: 50%;
}
.ph-title { font-size: 20px; margin-bottom: 8px; }
.ph-subtitle { font-size: 13px; color: var(--text-tertiary); margin-bottom: 24px; }
.ph-bar {
  display: inline-flex; align-items: center; gap: 8px;
  padding: 8px 18px;
  background: var(--bg-secondary);
  border-radius: 999px;
  font-size: 12px;
  color: var(--text-secondary);
}
</style>