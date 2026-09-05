// utils/echarts.js
// 统一 ECharts 初始化与主题化图表配置工厂(青-绿低碳主题)。
// 说明：仅提供"图表实例/option"基础能力，不构成角色业务面板组件，
// 每个角色首页各自组合其专属指标与图表，样式仍由各页面 scoped 隔离。
import * as echarts from 'echarts'

export { echarts }

// 主题色板：主色 teal，辅以蓝/琥珀/紫罗兰等用于多分类占比
export const PALETTE = ['#0d9488', '#2563eb', '#f59e0b', '#7c3aed', '#059669', '#06b6d4', '#84cc16', '#ec4899']
// 品牌渐变(面积/柱状填充用)
export const TEAL_GRAD = {
  type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
  colorStops: [
    { offset: 0, color: 'rgba(13, 148, 136, 0.28)' },
    { offset: 1, color: 'rgba(13, 148, 136, 0.02)' }
  ]
}

const TOOLTIP = {
  backgroundColor: '#ffffff',
  borderColor: '#e5e7eb',
  textStyle: { color: '#374151', fontSize: 12 },
  extraCssText: 'box-shadow:0 6px 18px rgba(15,23,42,.10); border-radius:8px;'
}
const GRID = { left: 4, right: 14, top: 30, bottom: 0, containLabel: true }

/**
 * 在 DOM 元素上初始化 ECharts 实例
 * @param {HTMLElement} el
 */
export function initChart(el) {
  if (!el) return null
  return echarts.init(el)
}

/**
 * 折线图(面积渐变)，用于各类"趋势"展示
 */
export function mkLineOption({ labels = [], values = [], color = '#0d9488', yName = '' }) {
  return {
    color: [color],
    tooltip: { ...TOOLTIP, trigger: 'axis' },
    grid: GRID,
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: labels,
      axisLine: { lineStyle: { color: '#e5e7eb' } },
      axisTick: { show: false },
      axisLabel: { color: '#94a3b8', fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      name: yName,
      nameTextStyle: { color: '#cbd5d1', fontSize: 10, padding: [0, 0, 0, -14] },
      splitLine: { lineStyle: { color: '#f0f4f7' } },
      axisLabel: { color: '#94a3b8', fontSize: 11 }
    },
    series: [{
      type: 'line',
      smooth: true,
      showSymbol: values.length <= 20,
      symbol: 'circle',
      symbolSize: 6,
      data: values,
      lineStyle: { width: 2.5, color },
      itemStyle: { color },
      areaStyle: { color: TEAL_GRAD }
    }]
  }
}

/**
 * 柱状图，用于"构成/排行/统计"
 */
export function mkBarOption({ labels = [], values = [], colors = null }) {
  const c = colors || Array(labels.length).fill('#0d9488')
  return {
    color: [c[0]],
    tooltip: { ...TOOLTIP, trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: GRID,
    xAxis: {
      type: 'category',
      data: labels,
      axisLine: { lineStyle: { color: '#e5e7eb' } },
      axisTick: { show: false },
      axisLabel: { color: '#94a3b8', fontSize: 11, interval: 0, rotate: labels.length > 6 ? 18 : 0 }
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: '#f0f4f7' } },
      axisLabel: { color: '#94a3b8', fontSize: 11 }
    },
    series: [{
      type: 'bar',
      barMaxWidth: 30,
      itemStyle: {
        borderRadius: [5, 5, 0, 0]
      },
      data: values.map((v, i) => ({
        value: v,
        itemStyle: {
          color: {
            type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: c[i % c.length] },
              { offset: 1, color: c[i % c.length] }
            ]
          }
        }
      }))
    }]
  }
}

/**
 * 环形饼图，用于"分布/占比"
 */
export function mkPieOption({ data = [], colors = PALETTE }) {
  return {
    color: colors,
    tooltip: { ...TOOLTIP, trigger: 'item', formatter: '{b}：{c} ({d}%)' },
    legend: {
      bottom: 0,
      left: 'center',
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 12,
      textStyle: { color: '#64748b', fontSize: 11 }
    },
    series: [{
      type: 'pie',
      radius: ['44%', '68%'],
      center: ['50%', '42%'],
      itemStyle: { borderRadius: 4, borderColor: '#ffffff', borderWidth: 2 },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 12, fontWeight: 600, color: '#111827' },
        scaleSize: 6
      },
      data: data.map((d, i) => ({ ...d, itemStyle: { color: colors[i % colors.length] } }))
    }]
  }
}

/**
 * 图表无业务数据时的统一居中文案(规则："暂无数据，完成业务后自动生成图表")
 */
export const CHART_EMPTY = '暂无数据，完成业务后自动生成图表'

/**
 * 空数据提示 option(图表容器存在但业务数据为空时的居中占位文案)
 */
export function emptyOption(text = CHART_EMPTY) {
  return {
    graphic: {
      type: 'text',
      left: 'center',
      top: 'middle',
      style: { text, fill: '#9ca3af', fontSize: 12 }
    }
  }
}
