<template>
  <div class="donut-wrap">
    <div v-if="!hasData" class="donut-empty">
      <svg viewBox="0 0 48 48" width="26" height="26"><circle cx="24" cy="24" r="15" stroke="currentColor" stroke-width="4" fill="none" stroke-dasharray="50 44" stroke-linecap="round"/></svg>
      <span>暂无数据</span>
    </div>
    <template v-else>
      <div class="donut-left">
        <div class="donut" :style="{ background: conic }">
          <div class="donut-center">
            <span class="donut-value">{{ displayTotal }}</span>
            <span class="donut-title">{{ centerTitle }}</span>
          </div>
        </div>
      </div>
      <div class="donut-legend">
        <div v-for="(s, i) in segments" :key="i" class="legend-row" :title="`${s.label}: ${fmt(s.value)}`">
          <span class="legend-dot" :style="{ background: s.color || palette[i % palette.length] }"></span>
          <span class="legend-label">{{ s.label }}</span>
          <span class="legend-val">{{ fmt(s.value) }}</span>
        </div>
        <p v-if="!segments.length" class="donut-none">暂无数据</p>
      </div>
    </template>
  </div>
</template>

<script setup>
// DonutChart 环形占比图(CSS conic-gradient 实现，无第三方依赖)
// segments: [{ label, value, color? }]；中心展示合计与标题。
import { computed } from 'vue'

const props = defineProps({
  segments: { type: Array, default: () => [] },
  centerTitle: { type: String, default: '合计' },
  unit: { type: String, default: '' }
})

const palette = ['#0d9488', '#0d9488', '#f59e0b', '#8b5cf6', '#10b981', '#06b6d4', '#ec4899']

const data = computed(() => (props.segments || []).filter(s => s && Number(s.value) > 0))
const hasData = computed(() => data.value.length > 0)
const total = computed(() => data.value.reduce((sum, s) => sum + (Number(s.value) || 0), 0))

const conic = computed(() => {
  if (!hasData.value) return '#eef2f2'
  let acc = 0
  const stops = data.value.map((s, i) => {
    const deg = (Number(s.value) / total.value) * 360
    const from = acc
    acc += deg
    const c = s.color || palette[i % palette.length]
    return `${c} ${from.toFixed(2)}deg ${acc.toFixed(2)}deg`
  })
  return `conic-gradient(${stops.join(', ')})`
})

const displayTotal = computed(() => {
  const v = total.value
  const n = Number(v) || 0
  return (Number.isInteger(n) ? n.toLocaleString() : n.toFixed(1)) + (props.unit ? ' ' + props.unit : '')
})

function fmt(v) {
  const n = Number(v) || 0
  return (Number.isInteger(n) ? n.toLocaleString() : n.toFixed(1))
}
</script>

<style scoped>
.donut-wrap {
  display: flex;
  align-items: center;
  gap: 26px;
  flex-wrap: wrap;
}
.donut-left { flex-shrink: 0; }
.donut {
  position: relative;
  width: 148px;
  height: 148px;
  border-radius: 50%;
  -webkit-mask: radial-gradient(circle closest-side, transparent 64%, #000 65%);
  mask: radial-gradient(circle closest-side, transparent 64%, #000 65%);
  background: #eef2f2;
}
.donut-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.donut-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-1);
  letter-spacing: -0.01em;
}
.donut-title {
  margin-top: 2px;
  font-size: 11px;
  color: var(--text-4);
}
.donut-legend {
  flex: 1;
  min-width: 150px;
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.legend-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-3);
}
.legend-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  flex-shrink: 0;
}
.legend-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.legend-val {
  font-weight: 600;
  color: var(--text-2);
}
.donut-empty {
  height: 148px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-4);
  font-size: 12px;
  width: 100%;
}
.donut-empty svg { color: #cbd5d1; }
.donut-none { margin: 0; color: var(--text-4); font-size: 12px; }
</style>
