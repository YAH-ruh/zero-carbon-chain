<!--
  EChart.vue
  通用 ECharts 容器组件：
    - 自动 init / setOption / resize / dispose
    - 根据当前 theme(light/dark) 切换图表配色
    - 支持响应式（ResizeObserver）
-->
<template>
  <div ref="chartEl" class="echart-box" :style="{ height: height }"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, computed } from 'vue'
import * as echarts from 'echarts'
import { useTheme } from '../composables/useTheme.js'

const props = defineProps({
  /** 图表 option */
  option: { type: Object, default: () => ({}) },
  /** 容器高度 */
  height: { type: String, default: '280px' },
  /** 是否深色（会自动根据 theme 判断，也可手动覆盖） */
  dark: { type: Boolean, default: null }
})

const chartEl = ref(null)
let instance = null
let ro = null

const { theme } = useTheme()

/* 是否深色模式（优先用 props.dark，否则跟随全局 theme） */
const isDark = computed(() => {
  if (props.dark !== null) return props.dark
  return theme.value === 'dark'
})

onMounted(() => {
  if (!chartEl.value) return
  instance = echarts.init(chartEl.value)
  instance.setOption(normalizeOption(props.option, isDark.value))

  // 监听容器尺寸变化
  ro = new ResizeObserver(() => instance?.resize())
  ro.observe(chartEl.value)
})

onBeforeUnmount(() => {
  ro?.disconnect()
  instance?.dispose()
  instance = null
})

/* option 变化时重新设置 */
watch(() => props.option, (opt) => {
  instance?.setOption(normalizeOption(opt, isDark.value), true)
}, { deep: true })

/* 主题变化时重新 setOption */
watch(isDark, (d) => {
  instance?.setOption(normalizeOption(props.option, d), true)
})

/* ============================================================
   主题适配：把浅色 option 转成深色
   核心：把 axisLine/axisLabel/splitLine/legend/textStyle 颜色
         替换成 CSS 变量对应的深色值
   ============================================================ */
function normalizeOption(opt, dark) {
  if (!dark) return opt
  // 🌿 绿色优先深色色板
  const darkPalette = ['#10b981', '#0d9488', '#22d3ee', '#a78bfa', '#f59e0b', '#f472b6', '#84cc16', '#ef4444']
  const deepClone = JSON.parse(JSON.stringify(opt))
  deepClone.color = darkPalette

  const darkText = '#86efac'
  const darkLine = 'rgba(16, 185, 129, 0.22)'
  const darkSplit = 'rgba(16, 185, 129, 0.10)'

  // 遍历 series 中的 areaStyle，替换渐变
  if (deepClone.series) {
    deepClone.series.forEach(s => {
      if (s.areaStyle) {
        s.areaStyle = {
          type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(16, 185, 129, 0.35)' },
            { offset: 1, color: 'rgba(16, 185, 129, 0.02)' }
          ]
        }
      }
      if (s.itemStyle?.borderColor === '#ffffff') {
        s.itemStyle.borderColor = '#f0fdf4'
      }
    })
  }

  // 遍历 tooltip / legend / xAxis / yAxis 染色
  if (deepClone.tooltip) {
    deepClone.tooltip.backgroundColor = '#ffffff'
    deepClone.tooltip.borderColor = 'rgba(16,185,129,0.25)'
    if (!deepClone.tooltip.textStyle) deepClone.tooltip.textStyle = {}
    deepClone.tooltip.textStyle.color = '#1e293b'
  }
  if (deepClone.legend?.textStyle) deepClone.legend.textStyle.color = darkText
  if (deepClone.xAxis) {
    [].concat(deepClone.xAxis).forEach(a => {
      if (a.axisLine) a.axisLine.lineStyle = { color: darkLine }
      if (a.axisLabel) a.axisLabel.color = darkText
      if (a.splitLine) a.splitLine.lineStyle = { color: darkSplit }
    })
  }
  if (deepClone.yAxis) {
    [].concat(deepClone.yAxis).forEach(a => {
      if (a.axisLine) a.axisLine.lineStyle = { color: darkLine }
      if (a.axisLabel) a.axisLabel.color = darkText
      if (a.splitLine) a.splitLine.lineStyle = { color: darkSplit }
    })
  }

  return deepClone
}
</script>

<style scoped>
.echart-box {
  width: 100%;
  min-height: 200px;
}
</style>