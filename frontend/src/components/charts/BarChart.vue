<template>
  <div class="bar-chart" :style="{ height: height + 'px' }">
    <!-- 空态：无数据时展示占位提示，不破坏图表区域 -->
    <div v-if="!chartItems.length" class="chart-empty">
      <svg viewBox="0 0 48 48" width="26" height="26"><path d="M10 38V22m14 16V10m14 28V16" stroke="currentColor" stroke-width="3" stroke-linecap="round" fill="none"/></svg>
      <span>暂无数据，完成业务后自动生成图表</span>
    </div>
    <template v-else>
      <div class="chart-grid" ref="box">
        <svg :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" class="chart-svg">
          <!-- 横向网格线 -->
          <line v-for="g in grid" :key="g.y" :x1="0" :x2="W" :y1="g.y" :y2="g.y" stroke="#eef2f2" stroke-width="1" />
        </svg>
        <div class="bar-area">
          <div v-for="(item, i) in chartItems" :key="i" class="bar-col" :title="`${item.label}: ${fmt(item.value)}`">
            <div class="bar-value" v-if="showValue">{{ fmt(item.value) }}</div>
            <div class="bar-track">
              <div class="bar-fill" :style="{ height: barHeight(item.value) + '%', background: colorOf(i) }"></div>
            </div>
            <div class="bar-label">{{ item.label }}</div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
// BarChart 轻量柱状图(SVG/CSS 实现，无第三方依赖)
// items: [{ label, value }]；自动按最大值归一化，超长 label 自动截断，保持横向自适应。
import { computed } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
  height: { type: Number, default: 200 },
  showValue: { type: Boolean, default: false },
  colors: { type: Array, default: () => [] } // 可覆盖默认主题渐变色
})

const W = 600
const H = 220
const defaultColors = [
  'linear-gradient(180deg,#2dd4bf,#0d9488)',
  'linear-gradient(180deg,#93c5fd,#2563eb)',
  'linear-gradient(180deg,#fcd34d,#d97706)',
  'linear-gradient(180deg,#c4b5fd,#7c3aed)',
  'linear-gradient(180deg,#a7f3d0,#059669)'
]

const chartItems = computed(() =>
  (props.items || []).filter(i => i && i.value != null).slice(0, 14)
)
const max = computed(() => Math.max(1, ...chartItems.value.map(i => Math.abs(Number(i.value) || 0))))
const grid = computed(() => [0.25, 0.5, 0.75, 1].map(f => ({ y: H * (1 - f) })))

function barHeight(v) {
  const val = Math.abs(Number(v) || 0)
  return val === 0 ? 2 : Math.max(2, (val / max.value) * 100)
}
function colorOf(i) {
  return props.colors[i % props.colors.length] || defaultColors[i % defaultColors.length]
}
function fmt(v) {
  const n = Number(v) || 0
  if (Number.isInteger(n)) return n.toLocaleString()
  return n.toFixed(1)
}
</script>

<style scoped>
.bar-chart { width: 100%; position: relative; }
.chart-grid { position: relative; height: 100%; }
.chart-svg { position: absolute; inset: 0; width: 100%; height: calc(100% - 22px); }
.bar-area {
  position: absolute;
  inset: 0;
  height: calc(100% - 22px);
  display: flex;
  align-items: flex-end;
  gap: 6px;
  padding: 0 4px;
}
.bar-col {
  flex: 1;
  min-width: 14px;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  align-items: center;
  height: 100%;
  gap: 3px;
}
.bar-track {
  width: 60%;
  max-width: 30px;
  min-width: 6px;
  flex: 1;
  display: flex;
  align-items: flex-end;
  background: #f1f5f9;
  border-radius: 4px 4px 2px 2px;
  overflow: hidden;
}
.bar-fill {
  width: 100%;
  border-radius: 4px 4px 2px 2px;
  transition: height 0.6s ease;
  min-height: 2px;
}
.bar-label {
  height: 22px;
  font-size: 11px;
  color: var(--text-4);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  text-align: center;
}
.bar-value {
  font-size: 11px;
  color: var(--text-3);
  font-weight: 500;
}
.chart-empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-4);
  font-size: 12px;
}
.chart-empty svg { color: #cbd5d1; }
</style>
