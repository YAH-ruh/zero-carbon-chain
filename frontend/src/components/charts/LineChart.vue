<template>
  <div class="line-chart" :style="{ height: height + 'px' }">
    <div v-if="!chartItems.length" class="chart-empty">
      <svg viewBox="0 0 48 48" width="26" height="26"><path d="M6 34 17 21l8 6 12-13" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" fill="none"/></svg>
      <span>暂无数据，完成业务后自动生成趋势图</span>
    </div>
    <svg v-else :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" class="line-svg">
      <defs>
        <linearGradient :id="gradId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#0d9488" stop-opacity="0.28" />
          <stop offset="100%" stop-color="#0d9488" stop-opacity="0.02" />
        </linearGradient>
      </defs>
      <!-- 网格 -->
      <line v-for="g in grid" :key="g.y" :x1="PL" :x2="W - PR" :y1="g.y" :y2="g.y" stroke="#eef2f2" stroke-width="1" stroke-dasharray="3 4" />
      <!-- 面积填充 -->
      <polygon :points="areaPoints" :fill="`url(#${gradId})`" />
      <!-- 折线 -->
      <polyline :points="linePoints" fill="none" stroke="#0d9488" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" />
      <!-- 数据点 -->
      <circle v-for="(pt, i) in points" :key="i" :cx="pt.x" :cy="pt.y" r="3" fill="#fff" stroke="#0d9488" stroke-width="2" />
      <!-- X 轴标签 -->
      <template v-for="(item, i) in chartItems" :key="'l' + i">
        <text v-if="i % labelStep === 0 || i === chartItems.length - 1" :x="points[i].x" :y="H - 2" text-anchor="middle" class="axis-label">{{ item.label }}</text>
      </template>
    </svg>
  </div>
</template>

<script setup>
// LineChart 轻量折线趋势图(SVG，无第三方依赖)
// items: [{ label, value }]，自动归一化；渐变面积填充贴合低碳主题。
import { computed } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
  height: { type: Number, default: 200 }
})

const W = 640
const H = 230
const PL = 8  // 左侧留白
const PR = 8  // 右侧留白
const PT = 12 // 顶部留白

const gradId = 'lg' + Math.random().toString(36).slice(2, 8)

const chartItems = computed(() =>
  (props.items || []).filter(i => i && i.value != null).slice(0, 24)
)
const max = computed(() => Math.max(1, ...chartItems.value.map(i => Math.abs(Number(i.value) || 0))))
const points = computed(() => {
  const n = chartItems.value.length
  if (!n) return []
  const stepX = n === 1 ? (W - PL - PR) / 2 : (W - PL - PR) / (n - 1)
  return chartItems.value.map((item, i) => ({
    x: PL + i * stepX,
    y: PT + (1 - (Math.abs(Number(item.value) || 0) / max.value)) * (H - PT - 26)
  }))
})
const linePoints = computed(() => points.value.map(p => `${p.x},${p.y}`).join(' '))
const areaPoints = computed(() => {
  if (!points.value.length) return ''
  const first = points.value[0]
  const last = points.value[points.value.length - 1]
  return `${first.x},${H - 24} ` + linePoints.value + ` ${last.x},${H - 24}`
})
const grid = computed(() => [0.25, 0.5, 0.75, 1].map(f => ({ y: PT + (1 - f) * (H - PT - 26) })))
const labelStep = computed(() => {
  const n = chartItems.value.length
  if (n <= 6) return 1
  return Math.ceil(n / 6)
})
</script>

<style scoped>
.line-chart { width: 100%; position: relative; }
.line-svg { width: 100%; height: 100%; display: block; }
.axis-label {
  font-size: 10px;
  fill: #9ca3af;
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
