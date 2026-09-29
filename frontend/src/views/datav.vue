<!--
  datav.vue - 数据中台大屏
  全屏深色大屏：顶部 KPI + 左中右三列图表 + 底部滚动企业状态
-->
<template>
  <div class="datav">

    <!-- ===== 顶部退出 + 时间 ===== -->
    <div class="dv-top-bar">
      <div class="dv-title">
        <span class="dt-logo">🌿</span>
        <span class="dt-text">零碳微证 · 数据中台大屏</span>
        <span class="dt-sub">CARBON INTELLIGENCE · PARK INTEGRATED CONTROL CENTER</span>
      </div>
      <div class="dv-time">
        <span class="dt-date">{{ nowDate }}</span>
        <span class="dt-clock">{{ nowClock }}</span>
      </div>
      <div class="dv-exit" @click="goDashboard">
        <el-icon><Close /></el-icon>
        退出大屏
      </div>
    </div>

    <!-- ===== 顶部 KPI 条 ===== -->
    <div class="dv-kpi-row">
      <div v-for="(k, idx) in kpiData" :key="k.label" class="kpi-item" :style="{ animationDelay: idx * 0.08 + 's' }">
        <div class="kpi-num" :style="{ color: k.color }">{{ k.value }}</div>
        <div class="kpi-label">{{ k.label }}</div>
        <div class="kpi-trend" :class="k.trend >= 0 ? 'up' : 'down'">
          <el-icon v-if="k.trend >= 0"><ArrowUp /></el-icon>
          <el-icon v-else><ArrowDown /></el-icon>
          {{ Math.abs(k.trend) }}%
        </div>
        <div class="kpi-bar" :style="{ background: k.color + '40' }"></div>
      </div>
    </div>

    <!-- ===== 筛选条 ===== -->
    <div class="dv-filter-row">
      <div class="filter-group">
        <span class="fg-label">园区</span>
        <el-select v-model="filterPark" size="small" class="fg-select">
          <el-option label="全部园区" value="all" />
          <el-option v-for="p in parkList" :key="p.name" :label="p.name" :value="p.name" />
        </el-select>
      </div>
      <div class="filter-group">
        <span class="fg-label">时间范围</span>
        <el-radio-group v-model="filterRange" size="small">
          <el-radio-button label="7">近7天</el-radio-button>
          <el-radio-button label="30">近30天</el-radio-button>
          <el-radio-button label="365">近一年</el-radio-button>
        </el-radio-group>
      </div>
      <el-button size="small" type="primary" class="fg-refresh" @click="refreshAll" :loading="loadingAll">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <!-- ===== 主体三列 ===== -->
    <div class="dv-main">

      <!-- ========== 左列 ========== -->
      <div class="dv-col dv-col-left">

        <div class="dv-panel">
          <div class="dp-head">
            <span class="dp-title"><span class="dp-dot"></span>碳积分构成</span>
          </div>
          <div class="dp-body">
            <EChart :option="pieOption" height="240px" dark />
          </div>
        </div>

        <div class="dv-panel">
          <div class="dp-head">
            <span class="dp-title"><span class="dp-dot"></span>企业健康度雷达</span>
          </div>
          <div class="dp-body">
            <EChart :option="radarOption" height="240px" />
          </div>
        </div>

      </div>

      <!-- ========== 中央：园区分布 ========== -->
      <div class="dv-col dv-col-center">

        <div class="dv-panel dv-panel-center">
          <div class="dp-head">
            <span class="dp-title"><span class="dp-dot"></span>园区分布 · 实时数据</span>
            <span class="dp-sub">共 {{ parkList.length }} 个园区 · {{ totalEnterprises }} 家企业</span>
          </div>
          <div class="dp-body park-grid">
            <div
              v-for="(p, idx) in parkList"
              :key="p.name"
              class="park-card"
              :style="{ animationDelay: idx * 0.1 + 's' }"
            >
              <div class="park-header">
                <div class="park-icon">{{ parkIcons[idx % parkIcons.length] }}</div>
                <div class="park-name">{{ p.name }}</div>
                <div class="park-status" :class="p.alert ? 'alert' : 'ok'">
                  <span class="pulse"></span>
                  {{ p.alert ? '告警' : '正常' }}
                </div>
              </div>
              <div class="park-metrics">
                <div class="pm">
                  <span class="pm-val">{{ p.enterprises }}</span>
                  <span class="pm-lab">企业</span>
                </div>
                <div class="pm">
                  <span class="pm-val green">{{ p.emission.toLocaleString() }}</span>
                  <span class="pm-lab">排放(kg)</span>
                </div>
                <div class="pm">
                  <span class="pm-val blue">{{ p.credits }}</span>
                  <span class="pm-lab">积分</span>
                </div>
              </div>
              <!-- 迷你柱状：近 7 天排放 -->
              <div class="park-spark">
                <div
                  v-for="(h, i) in p.spark"
                  :key="i"
                  class="spark-bar"
                  :style="{ height: h + '%' }"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <!-- 中央下方：链上交易趋势 -->
        <div class="dv-panel">
          <div class="dp-head">
            <span class="dp-title"><span class="dp-dot"></span>链上交易趋势 · 近 {{ filterRange }} 天</span>
            <span class="dp-sub">区块链存证实时同步</span>
          </div>
          <div class="dp-body">
            <EChart :option="trendOption" height="200px" dark />
          </div>
        </div>
      </div>

      <!-- ========== 右列 ========== -->
      <div class="dv-col dv-col-right">

        <div class="dv-panel">
          <div class="dp-head">
            <span class="dp-title"><span class="dp-dot"></span>园区碳排放对比 TOP</span>
          </div>
          <div class="dp-body">
            <EChart :option="barOption" height="240px" dark />
          </div>
        </div>

        <!-- 风险告警 -->
        <div class="dv-panel dv-panel-risk">
          <div class="dp-head">
            <span class="dp-title risk"><span class="dp-dot pulse-dot"></span>实时风险告警</span>
            <span class="dp-sub danger">{{ riskList.length }} 条</span>
          </div>
          <div class="dp-body risk-body">
            <div
              v-for="(r, i) in riskList"
              :key="i"
              class="risk-item"
              :class="r.level"
              :style="{ animationDelay: i * 0.05 + 's' }"
            >
              <div class="ri-bar"></div>
              <div class="ri-content">
                <div class="ri-top">
                  <el-icon :size="12"><Warning /></el-icon>
                  <span class="ri-title">{{ r.title }}</span>
                </div>
                <div class="ri-desc">{{ r.desc }}</div>
                <div class="ri-time">{{ r.time }}</div>
              </div>
              <span class="ri-badge">{{ r.level === 'danger' ? '严重' : '警告' }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 底部：滚动企业状态 ===== -->
    <div class="dv-bottom">
      <div class="dv-panel">
        <div class="dp-head">
          <span class="dp-title"><span class="dp-dot"></span>企业链上状态监控</span>
          <span class="dp-sub">实时刷新 · 共 {{ enterpriseStatus.length }} 家</span>
        </div>
        <div class="dp-body status-body">
          <div class="status-scroll">
            <div class="status-row" v-for="e in enterpriseStatus" :key="e.id">
              <span class="sr-id">{{ e.id }}</span>
              <span class="sr-name">{{ e.name }}</span>
              <span class="sr-park">{{ e.park }}</span>
              <span class="sr-emission">{{ e.emission.toFixed(0) }} kg</span>
              <span class="sr-credit">{{ e.credits }} 积分</span>
              <span class="sr-chain" :class="{ on: e.on_chain }">
                <el-icon v-if="e.on_chain"><Connection /></el-icon>
                <el-icon v-else><Warning /></el-icon>
                {{ e.on_chain ? '已上链' : '待上链' }}
              </span>
              <span class="sr-compliance" :class="e.compliance >= 80 ? 'ok' : e.compliance >= 60 ? 'warn' : 'danger'">
                {{ e.compliance }} 分
              </span>
              <span class="sr-time">{{ e.last_time }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import {
  Close, ArrowUp, ArrowDown, Refresh, Warning, Connection
} from '@element-plus/icons-vue'
import EChart from '../components/EChart.vue'
import {
  genMonthlyTrend, genCreditDistribution, genEnterpriseRanking,
  genParkDistribution, genEnterpriseRadar
} from '../utils/mockChartData.js'

const router = useRouter()

/* ===== 时间 ===== */
const nowDate = ref('')
const nowClock = ref('')
let timer = null
function tick() {
  const d = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  nowDate.value = `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${['周日','周一','周二','周三','周四','周五','周六'][d.getDay()]}`
  nowClock.value = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
onMounted(() => { tick(); timer = setInterval(tick, 1000) })
onBeforeUnmount(() => clearInterval(timer))

/* ===== 筛选 ===== */
const filterPark = ref('all')
const filterRange = ref('30')

/* ===== KPI ===== */
const kpiData = ref([
  { label: '监管企业数',    value: 356,    color: '#059669',  trend: 4 },
  { label: '月碳排放(t)',  value: 1842,   color: '#10b981',  trend: -3 },
  { label: '碳积分流通量',  value: 24680,  color: '#a78bfa',  trend: 12 },
  { label: '挂单总量',      value: 2840,   color: '#f59e0b',  trend: 7 },
  { label: '今日成交(万)', value: 3820,   color: '#22c55e',  trend: 22 },
  { label: '风险告警',      value: 23,     color: '#ef4444',  trend: -15 },
  { label: '链上区块',      value: '2.3k', color: '#059669',  trend: 9 },
])

/* ===== 园区列表 ===== */
const parkIcons = ['🏭', '🌿', '⚡', '♻️', '🏗️', '🔬', '🚗', '📦']
const parkList = ref(genParkDistribution().map((p, i) => ({
  ...p,
  spark: Array.from({ length: 7 }, () => Math.round(30 + Math.random() * 70)),
  alert: Math.random() > 0.7,
})))
const totalEnterprises = computed(() => parkList.value.reduce((s, p) => s + p.enterprises, 0))

/* ===== 风险告警 ===== */
const riskList = ref([
  { level: 'danger', title: '绿能科技能耗异常', desc: '近 7 天能耗超基准线 +35%', time: '2 分钟前' },
  { level: 'danger', title: '疑似双重提交',     desc: '企业#218 同一时段两次能耗上报', time: '8 分钟前' },
  { level: 'warn',   title: '链上区块落后',     desc: '能源工业园落后 12 个区块', time: '14 分钟前' },
  { level: 'warn',   title: '连续未上报',       desc: '3 家企业 2 个月未提交能耗数据', time: '32 分钟前' },
  { level: 'warn',   title: 'ZKP 校验失败',     desc: '2 条记录范围证明不匹配', time: '45 分钟前' },
])

/* ===== 企业链上状态 ===== */
const enterpriseStatus = ref(
  Array.from({ length: 12 }, (_, i) => ({
    id: i + 1,
    name: ['绿能科技','低碳智造','星辉电子','清源化工','蓝天包装','金石制造','华兴能源','创新材料','节能建材','环保食品','智绿集团','循环科技'][i],
    park: parkList.value[i % parkList.value.length]?.name?.slice(0, 6) + '...',
    emission: Math.round(500 + Math.random() * 2000),
    credits: Math.round(50 + Math.random() * 800),
    on_chain: Math.random() > 0.15,
    compliance: Math.round(55 + Math.random() * 45),
    last_time: `${String(14 + Math.floor(Math.random()*4)).padStart(2,'0')}:${String(Math.floor(Math.random()*60)).padStart(2,'0')}`,
  }))
)

const loadingAll = ref(false)
function refreshAll() {
  loadingAll.value = true
  setTimeout(() => {
    kpiData.value = kpiData.value.map(k => ({ ...k, trend: Math.round((Math.random() - 0.5) * 30) }))
    parkList.value = genParkDistribution().map((p, i) => ({
      ...p,
      spark: Array.from({ length: 7 }, () => Math.round(30 + Math.random() * 70)),
      alert: Math.random() > 0.7,
    }))
    enterpriseStatus.value = enterpriseStatus.value.map(e => ({ ...e, on_chain: Math.random() > 0.15, compliance: Math.round(55 + Math.random() * 45) }))
    loadingAll.value = false
  }, 600)
}

/* ===== 图表 ===== */
const trendData = computed(() => genMonthlyTrend(1500, 0.3))

const trendOption = computed(() => ({
  color: ['#059669', '#10b981'],
  tooltip: { trigger: 'axis' },
  legend: { data: ['总排放(kg)', '碳积分'], top: 0, right: 10, textStyle: { color: '#64748b', fontSize: 11 }, icon: 'circle' },
  grid: { left: 10, right: 10, top: 36, bottom: 0, containLabel: true },
  xAxis: {
    type: 'category', boundaryGap: false, data: trendData.value.labels,
    axisLine: { lineStyle: { color: '#cbd5e1' } },  /* 浅灰轴线条，白底清晰 */
    axisLabel: { color: '#475569', fontSize: 10 },     /* 深灰文字 */
    axisTick: { show: false },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: '#e2e8f0' } },  /* 浅灰网格线 */
    axisLabel: { color: '#475569', fontSize: 10 }     /* 深灰文字 */
  },
  series: [
    {
      name: '总排放(kg)', type: 'line', smooth: true, showSymbol: false,
      lineStyle: { width: 2.5 },
      areaStyle: { type: 'linear', x:0, y:0, x2:0, y2:1,
        colorStops: [
          { offset: 0, color: 'rgba(5,150,105,0.3)' },
          { offset: 1, color: 'rgba(5,150,105,0.02)' }
        ]
      },
      data: trendData.value.emission
    },
    {
      name: '碳积分', type: 'line', smooth: true, showSymbol: false,
      lineStyle: { width: 2 }, data: trendData.value.credits
    }
  ]
}))

const pieOption = computed(() => ({
  tooltip: { trigger: 'item' },
  legend: { orient: 'vertical', right: 0, top: 'middle', textStyle: { color: '#64748b', fontSize: 11 } },
  series: [{
    type: 'pie', radius: ['42%', '68%'], center: ['38%', '50%'],
    avoidLabelOverlap: false,
    itemStyle: { borderRadius: 4, borderColor: '#f0fdf4', borderWidth: 2 },
    label: { show: false }, labelLine: { show: false },
    data: [
      { value: 500, name: '已挂单' },
      { value: 800, name: '已成交' },
      { value: 400, name: '可用余额' },
      { value: 200, name: '质押中' },
      { value: 50,  name: '已过期' },
    ]
  }]
}))

const radarOption = computed(() => {
  const r = genEnterpriseRadar()
  return {
    tooltip: {},
    legend: { data: r.series.map(s => s.name), bottom: 0, textStyle: { color: '#475569', fontSize: 11 }, icon: 'circle' },
    radar: {
      indicator: r.indicator,
      center: ['50%', '48%'], radius: '60%',
      axisName: { color: '#475569', fontSize: 10 },           /* 深灰指标名 */
      splitArea: { areaStyle: { color: ['rgba(5,150,105,0.04)', 'rgba(5,150,105,0.08)'] } },
      axisLine: { lineStyle: { color: '#cbd5e1' } },          /* 浅灰轴线 */
      splitLine: { lineStyle: { color: '#e2e8f0' } },         /* 浅灰网格 */
    },
    series: [{
      type: 'radar',
      data: r.series.map((s, i) => ({
        name: s.name, value: s.value,
        areaStyle: { opacity: 0.2 },
        lineStyle: { width: 2 },
        itemStyle: { color: i === 0 ? '#059669' : '#10b981' }
      }))
    }]
  }
})

const barOption = computed(() => {
  const { labels, values } = genEnterpriseRanking(6)
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 10, right: 10, top: 10, bottom: 20, containLabel: true },
    xAxis: {
      type: 'category', data: labels,
      axisLine: { lineStyle: { color: '#cbd5e1' } },     /* 浅灰轴线 */
      axisLabel: { color: '#475569', fontSize: 10, interval: 0, rotate: 0, formatter: (n) => String(n).replace(/(.{5})/g, '$1\n') },  /* 深灰文字 · 每5字换行防重叠 */
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: '#e2e8f0' } },    /* 浅灰网格 */
      axisLabel: { color: '#475569', fontSize: 10 }      /* 深灰文字 */
    },
    series: [{
      type: 'bar', data: values,
      barWidth: '45%',
      itemStyle: {
        borderRadius: [4, 4, 0, 0],
        color: {
          type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: '#059669' },
            { offset: 1, color: '#10b981' }
          ]
        }
      }
    }]
  }
})

function goDashboard() { router.push('/dashboard') }
</script>

<style scoped>
/* ===== 全屏容器 ===== */
.datav {
  height: 100vh;
  width: 100vw;
  padding: 12px 16px;
  background:
    radial-gradient(ellipse at 20% 0%, rgba(16, 185, 129, 0.12) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 100%, rgba(16, 185, 129, 0.08) 0%, transparent 50%),
    linear-gradient(180deg, #f8fffb 0%, #ecfdf5 100%);
  color: #14532d;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
  animation: fadeIn 0.4s ease;
}
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

/* ===== 顶部栏 ===== */
.dv-top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px;
  height: 48px;
}
.dv-title {
  display: flex; align-items: center; gap: 12px;
  flex: 1;
}
.dt-logo {
  font-size: 22px;
  filter: drop-shadow(0 2px 4px rgba(5, 150, 105, 0.15));
}
.dt-text {
  font-size: 20px;
  font-weight: 700;
  background: linear-gradient(90deg, #059669, #10b981);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  letter-spacing: 2px;
}
.dt-sub {
  font-size: 11px;
  color: #475569;
  letter-spacing: 1px;
}
.dv-time { text-align: right; }
.dt-date { font-size: 11px; color: #64748b; }
.dt-clock {
  font-size: 20px;
  font-weight: 700;
  font-family: ui-monospace, Consolas, monospace;
  color: #059669;
  
}
.dv-exit {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 14px;
  font-size: 12px;
  color: #64748b;
  cursor: pointer;
  border: 1px solid #bbf7d0;
  border-radius: 6px;
  transition: all 0.2s;
}
.dv-exit:hover {
  background: rgba(239,68,68,0.15);
  border-color: #ef4444;
  color: #ef4444;
}

/* ===== KPI 条 ===== */
.dv-kpi-row {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 8px;
}
.kpi-item {
  position: relative;
  padding: 12px 14px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.85), rgba(240, 253, 244, 0.9));
  border: 1px solid #bbf7d0;
  border-radius: 6px;
  overflow: hidden;
  animation: fadeUp 0.5s ease both;
}
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }
.kpi-bar {
  position: absolute;
  bottom: 0; left: 0;
  height: 2px;
  width: 100%;
  transform: scaleX(0.6);
  transform-origin: left;
}
.kpi-num {
  font-size: 22px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
  
}
.kpi-label {
  font-size: 11px;
  color: #64748b;
  margin-top: 2px;
}
.kpi-trend {
  position: absolute;
  top: 10px; right: 12px;
  font-size: 11px;
  display: flex; align-items: center; gap: 2px;
}
.kpi-trend.up { color: #10b981; }
.kpi-trend.down { color: #ef4444; }

/* ===== 筛选条 ===== */
.dv-filter-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 8px 14px;
  background: rgba(240, 253, 244, 0.5);
  border: 1px solid #bbf7d0;
  border-radius: 6px;
}
.filter-group { display: flex; align-items: center; gap: 8px; }
.fg-label { font-size: 11px; color: #64748b; }
.fg-select :deep(.el-select__wrapper) {
  background: #f0fdf4;
  box-shadow: 0 0 0 1px #bbf7d0 inset;
}
.fg-select :deep(.el-select__placeholder),
.fg-select :deep(.el-select__selected-item) {
  color: #14532d;
}
.fg-refresh { background: #10b981 !important; border-color: #10b981 !important; }

/* ===== 主体三列 ===== */
.dv-main {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1.6fr 1fr;
  gap: 10px;
  min-height: 0;
}
.dv-col {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
}

/* ===== 面板通用 ===== */
.dv-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: linear-gradient(180deg, rgba(255,255,255,0.85) 0%, rgba(240,253,244,0.9) 100%);
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  overflow: hidden;
}
.dv-panel-center { flex: 1.3; }

.dp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid #bbf7d0;
  background: linear-gradient(90deg, rgba(5,150,105,0.08) 0%, transparent 100%);
}
.dp-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 600; color: #14532d;
}
.dp-title.risk { color: #ef4444; }
.dp-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  background: #059669;
  box-shadow: 0 2px 8px rgba(5, 150, 105, 0.3);
}
.dp-dot.pulse-dot {
  background: #ef4444;
  box-shadow: 0 2px 8px rgba(239, 68, 68, 0.35);
  animation: pulse 1.5s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.65; transform: scale(1.05); }
}
.dp-sub { font-size: 11px; color: #64748b; }
.dp-sub.danger { color: #ef4444; }

.dp-body {
  flex: 1;
  padding: 12px 14px;
  min-height: 0;
  overflow: hidden;
}

/* ===== 园区卡片网格 ===== */
.park-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  padding: 12px;
}
.park-card {
  padding: 12px 14px;
  background: rgba(255, 255, 255, 0.75);
  border: 1px solid #bbf7d0;
  border-radius: 8px;
  animation: parkIn 0.5s ease both;
  transition: border-color 0.2s;
}
.park-card:hover { border-color: #059669; }
@keyframes parkIn { from { opacity: 0; transform: scale(0.95); } to { opacity: 1; transform: scale(1); } }

.park-header {
  display: flex; align-items: center; gap: 8px;
  margin-bottom: 10px;
}
.park-icon { font-size: 18px; }
.park-name { font-size: 12px; font-weight: 600; color: #14532d; flex: 1; }
.park-status {
  display: flex; align-items: center; gap: 4px;
  font-size: 10px; padding: 2px 8px; border-radius: 999px;
}
.park-status.ok { background: rgba(16,185,129,0.15); color: #10b981; }
.park-status.alert { background: rgba(239,68,68,0.15); color: #ef4444; }
.pulse {
  width: 6px; height: 6px; border-radius: 50%;
  background: currentColor;
  animation: pulse 1.5s infinite;
}

.park-metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
  margin-bottom: 10px;
}
.pm { display: flex; flex-direction: column; }
.pm-val {
  font-size: 14px; font-weight: 700; color: #14532d;
  font-variant-numeric: tabular-nums;
}
.pm-val.green { color: #10b981; }
.pm-val.blue { color: #059669; }
.pm-lab { font-size: 10px; color: #64748b; }

.park-spark {
  display: flex; align-items: flex-end; gap: 3px;
  height: 26px;
}
.spark-bar {
  flex: 1;
  background: linear-gradient(180deg, #059669, #10b981);
  border-radius: 2px 2px 0 0;
  min-height: 2px;
  opacity: 0.7;
}

/* ===== 风险告警 ===== */
.risk-body {
  display: flex; flex-direction: column; gap: 8px;
  overflow-y: auto;
}
.risk-item {
  display: flex; gap: 10px;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.75);
  border-radius: 6px;
  animation: slideIn 0.4s ease both;
}
@keyframes slideIn { from { opacity: 0; transform: translateX(10px); } to { opacity: 1; transform: translateX(0); } }
.ri-bar { width: 3px; border-radius: 2px; flex-shrink: 0; }
.risk-item.danger .ri-bar { background: #ef4444; }
.risk-item.warn .ri-bar { background: #f59e0b; }

.ri-content { flex: 1; min-width: 0; }
.ri-top {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 600; color: #14532d;
}
.risk-item.danger .ri-top { color: #ef4444; }
.risk-item.warn .ri-top { color: #f59e0b; }

.ri-desc {
  font-size: 11px; color: #64748b; margin-top: 3px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.ri-time { font-size: 10px; color: #475569; margin-top: 3px; }

.ri-badge {
  font-size: 10px; padding: 2px 8px; border-radius: 999px;
  flex-shrink: 0; align-self: center;
}
.risk-item.danger .ri-badge { background: rgba(239,68,68,0.15); color: #ef4444; }
.risk-item.warn .ri-badge { background: rgba(245,158,11,0.15); color: #f59e0b; }

/* ===== 底部状态条 ===== */
.dv-bottom { height: 160px; flex-shrink: 0; }
.status-body { padding: 0 !important; overflow: hidden; }

.status-scroll {
  height: 100%;
  overflow-y: auto;
}
.status-row {
  display: grid;
  grid-template-columns: 50px 120px 120px 100px 100px 90px 90px 80px;
  gap: 10px;
  align-items: center;
  padding: 8px 14px;
  font-size: 11.5px;
  border-bottom: 1px solid #e8f7ef;
  transition: background 0.15s;
}
.status-row:hover { background: rgba(5,150,105,0.05); }
.sr-id { color: #475569; font-family: monospace; }
.sr-name { color: #14532d; font-weight: 500; }
.sr-park { color: #64748b; font-size: 10px; }
.sr-emission { color: #64748b; text-align: right; }
.sr-credit { color: #059669; text-align: right; font-weight: 600; }
.sr-chain {
  display: flex; align-items: center; gap: 4px;
  font-size: 11px; justify-content: center;
  color: #64748b;
}
.sr-chain.on { color: #10b981; }
.sr-compliance {
  text-align: center; font-weight: 600;
}
.sr-compliance.ok { color: #10b981; }
.sr-compliance.warn { color: #f59e0b; }
.sr-compliance.danger { color: #ef4444; }
.sr-time { color: #475569; text-align: right; font-family: monospace; }

.status-scroll::-webkit-scrollbar { width: 5px; }
.status-scroll::-webkit-scrollbar-thumb { background: #bbf7d0; border-radius: 3px; }
</style>