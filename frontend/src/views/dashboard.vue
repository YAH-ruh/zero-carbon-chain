<!--
  dashboard.vue - 仪表盘
  根据当前登录角色渲染不同内容：
    enterprise（小微企业）：浅色工作台，能耗趋势 + 碳积分分布 + 快捷入口
    park_admin（园区管理员）：深色大屏风格，园区分布 + 企业排行 + 风险告警
    exchange（碳交易所）：交易趋势 + 类型构成 + 成交榜
    regulator（监管核查）：审计大屏 + 风险告警 + 链上校验
-->
<template>
  <div class="dashboard" :class="['role-' + role, isDark ? 'theme-dark' : 'theme-light']">

    <!-- ============ 欢迎横幅 ============ -->
    <div class="welcome-banner">
      <div class="wb-left">
        <h1>{{ welcomeTitle }}</h1>
        <p>{{ welcomeSub }}</p>
      </div>
      <div class="wb-right">
        <div class="wb-time">
          <el-icon :size="14"><Clock /></el-icon>
          <span>{{ now }}</span>
        </div>
        <div class="wb-role-badge" :style="{ background: roleColor }">
          {{ roleLabel }}
        </div>
      </div>
    </div>

    <!-- ============ 指标卡片（所有角色通用结构，内容按角色定制） ============ -->
    <div class="stats-grid">
      <div
        v-for="(s, idx) in statsCards"
        :key="s.label"
        class="stat-card"
        :style="{ animationDelay: (idx * 0.05) + 's' }"
      >
        <div class="sc-bg" :style="{ background: s.colorBg }"></div>
        <div class="sc-body">
          <div class="sc-icon" :style="{ background: s.colorBg, color: s.color }">
            <el-icon :size="18"><component :is="s.icon" /></el-icon>
          </div>
          <div class="sc-info">
            <div class="sc-num" :style="{ color: s.color }">{{ s.value }}</div>
            <div class="sc-label">{{ s.label }}</div>
          </div>
          <div class="sc-trend" :class="s.trend >= 0 ? 'up' : 'down'">
            {{ s.trend >= 0 ? '+' : '' }}{{ s.trend }}%
            <span class="sc-trend-label">较上月</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ============ 图表区 ============ -->
    <!-- 角色：小微企业（浅色工作台） -->
    <template v-if="role === 'enterprise'">
      <!-- 企业名录(仅查看)：展示平台全部注册企业，与仪表盘卡片风格一致 -->
      <div class="chart-card directory-card">
        <div class="cc-header">
          <h3>企业名录</h3>
          <span class="cc-sub">仅查看</span>
        </div>
        <div class="directory-body">
          <div class="directory-info">
            <el-icon :size="22" color="#0d9488"><OfficeBuilding /></el-icon>
            <div>
              <div class="dir-title">平台注册企业清单</div>
              <div class="dir-sub">查看全部 {{ enterpriseTotal }} 家企业的名称、类型、注册时间与状态</div>
            </div>
          </div>
          <el-button type="primary" :icon="View" :loading="directoryLoading" @click="openDirectory">查看名录</el-button>
        </div>
      </div>

      <div class="charts-row">
        <div class="chart-card chart-main">
          <div class="cc-header">
            <h3>能耗 & 碳积分趋势</h3>
            <span class="cc-sub">近 12 个月</span>
          </div>
          <EChart :option="lineOption" height="300px" />
        </div>
        <div class="chart-card chart-side">
          <div class="cc-header">
            <h3>碳积分分布</h3>
            <span class="cc-sub">实时</span>
          </div>
          <EChart :option="pieOption" height="300px" />
        </div>
      </div>

      <div class="charts-row">
        <div class="chart-card chart-wide">
          <div class="cc-header">
            <h3>企业健康度雷达</h3>
            <span class="cc-sub">6 维评估</span>
          </div>
          <EChart :option="radarOption" height="300px" />
        </div>
        <div class="chart-card chart-list">
          <div class="cc-header">
            <h3>风险提示</h3>
            <span class="cc-tag danger">{{ riskList.length }}</span>
          </div>
          <div class="risk-list">
            <div v-for="(r, i) in riskList" :key="i" class="risk-item" :class="r.level">
              <el-icon :size="14"><Warning /></el-icon>
              <span>{{ r.text }}</span>
              <button class="risk-btn" @click="router.push('/' + r.action)">处理 →</button>
            </div>
            <div v-if="!riskList.length" class="risk-empty">
              <el-icon :size="24" color="#22c55e"><CircleCheck /></el-icon>
              <span>当前无风险告警</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 角色：园区管理员 -->
    <template v-else-if="role === 'park_admin'">
      <div class="charts-row">
        <div class="chart-card chart-main">
          <div class="cc-header">
            <h3>园区企业能耗排行 TOP 8</h3>
            <span class="cc-sub">单位 kgCO₂</span>
          </div>
          <EChart :option="barOption" height="320px" />
        </div>
        <div class="chart-card chart-side">
          <div class="cc-header">
            <h3>园区月度能耗趋势</h3>
            <span class="cc-sub">近 12 个月</span>
          </div>
          <EChart :option="lineOption" height="320px" />
        </div>
      </div>
      <div class="charts-row">
        <div class="chart-card chart-wide">
          <div class="cc-header">
            <h3>园区分布统计</h3>
          </div>
          <div class="park-table">
            <div class="pt-head">
              <span>园区名称</span><span>企业数</span><span>总排放(kg)</span><span>碳积分</span><span>环比</span>
            </div>
            <div v-for="p in parkList" :key="p.name" class="pt-row">
              <span>{{ p.name }}</span>
              <span>{{ p.enterprises }}</span>
              <span>{{ p.emission.toLocaleString() }}</span>
              <span class="highlight">{{ p.credits }}</span>
              <span :class="p.trend >= 0 ? 'up' : 'down'">{{ p.trend >= 0 ? '+' : '' }}{{ p.trend }}%</span>
            </div>
          </div>
        </div>
        <div class="chart-card chart-list">
          <div class="cc-header">
            <h3>风险告警</h3>
            <span class="cc-tag danger">{{ riskList.length }}</span>
          </div>
          <div class="risk-list">
            <div v-for="(r, i) in riskList" :key="i" class="risk-item" :class="r.level">
              <el-icon :size="14"><Warning /></el-icon>
              <span>{{ r.text }}</span>
            </div>
            <div v-if="!riskList.length" class="risk-empty">
              <el-icon :size="24" color="#22c55e"><CircleCheck /></el-icon>
              <span>当前无风险告警</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 角色：碳交易所 -->
    <template v-else-if="role === 'exchange'">
      <div class="charts-row">
        <div class="chart-card chart-main">
          <div class="cc-header">
            <h3>碳积分交易趋势</h3>
            <span class="cc-sub">近 12 个月 · 成交量</span>
          </div>
          <EChart :option="lineOption" height="320px" />
        </div>
        <div class="chart-card chart-side">
          <div class="cc-header">
            <h3>交易类型构成</h3>
            <span class="cc-sub">现货/租赁/远期/质押</span>
          </div>
          <EChart :option="pieOption" height="320px" />
        </div>
      </div>
      <div class="charts-row">
        <div class="chart-card chart-wide">
          <div class="cc-header">
            <h3>成交榜 TOP 企业</h3>
          </div>
          <EChart :option="barOption" height="280px" />
        </div>
        <div class="chart-card chart-list">
          <div class="cc-header">
            <h3>交易告警</h3>
            <span class="cc-tag danger">{{ riskList.length }}</span>
          </div>
          <div class="risk-list">
            <div v-for="(r, i) in riskList" :key="i" class="risk-item" :class="r.level">
              <el-icon :size="14"><Warning /></el-icon>
              <span>{{ r.text }}</span>
            </div>
            <div v-if="!riskList.length" class="risk-empty">
              <el-icon :size="24" color="#22c55e"><CircleCheck /></el-icon>
              <span>当前无告警</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 角色：监管核查 -->
    <template v-else-if="role === 'regulator'">
      <div class="charts-row">
        <div class="chart-card chart-main">
          <div class="cc-header">
            <h3>链上审计 · 月度核验量</h3>
            <span class="cc-sub">近 12 个月</span>
          </div>
          <EChart :option="lineOption" height="320px" />
        </div>
        <div class="chart-card chart-side">
          <div class="cc-header">
            <h3>企业合规分布</h3>
            <span class="cc-sub">实时</span>
          </div>
          <EChart :option="pieOption" height="320px" />
        </div>
      </div>
      <div class="charts-row">
        <div class="chart-card chart-wide">
          <div class="cc-header">
            <h3>企业合规评分 · TOP 排行</h3>
          </div>
          <EChart :option="barOption" height="280px" />
        </div>
        <div class="chart-card chart-list">
          <div class="cc-header">
            <h3>风险告警</h3>
            <span class="cc-tag danger">{{ riskList.length }}</span>
          </div>
          <div class="risk-list">
            <div v-for="(r, i) in riskList" :key="i" class="risk-item" :class="r.level">
              <el-icon :size="14"><Warning /></el-icon>
              <span>{{ r.text }}</span>
              <button class="risk-btn" @click="router.push('/rollup-verify')">核查 →</button>
            </div>
            <div v-if="!riskList.length" class="risk-empty">
              <el-icon :size="24" color="#22c55e"><CircleCheck /></el-icon>
              <span>当前无风险告警</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 企业名录弹窗(仅查看，不做编辑/删除) -->
    <el-dialog v-model="directoryVisible" width="720px" title="企业名录 · 平台注册企业">
      <el-table :data="enterpriseList" v-loading="directoryLoading" stripe style="width:100%">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column label="企业名称" min-width="170">
          <template #default="{ row }">{{ brandText(row.company || row.username) }}</template>
        </el-table-column>
        <el-table-column prop="enterprise_type" label="企业类型" width="140" />
        <el-table-column prop="created_at" label="注册时间" width="170" />
        <el-table-column label="企业状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small" effect="plain">
              {{ row.status === 1 ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import EChart from '../components/EChart.vue'
import { useTheme } from '../composables/useTheme.js'
import { mkLineOption, mkBarOption, mkPieOption } from '../utils/echarts.js'
import {
  genMonthlyTrend, genCreditDistribution, genTradeComposition,
  genEnterpriseRanking, genParkDistribution, genEnterpriseRadar
} from '../utils/mockChartData.js'

import { ElMessage } from 'element-plus'
import {
  Clock, Warning, CircleCheck, DataBoard, Cpu, EditPen, Lock, Wallet,
  TrendCharts, Money, Collection, Document, MagicStick, Monitor, ScaleToOriginal,
  Tickets, Medal, Key, PieChart, DataLine, OfficeBuilding, User, View
} from '@element-plus/icons-vue'
import { brandText } from '../utils/brand.js'
import { directoryAPI, iotAPI } from '../api/index.js'

const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{}')
const role = user.role || 'enterprise'
const { isDark } = useTheme()

/* 角色元 */
const roleLabelMap = {
  enterprise: '小微企业工作台',
  park_admin: '园区管理员',
  exchange: '碳交易所',
  regulator: '监管核查'
}
const roleColorMap = {
  enterprise: 'linear-gradient(135deg,#16a34a,#22c55e)',
  park_admin: 'linear-gradient(135deg,#0d9488,#06b6d4)',
  exchange: 'linear-gradient(135deg,#7c3aed,#a78bfa)',
  regulator: 'linear-gradient(135deg,#dc2626,#f59e0b)',
}
const roleLabel = roleLabelMap[role] || ''
const roleColor = roleColorMap[role] || roleColorMap.enterprise

/* ===== 企业名录(仅查看) ===== */
const directoryVisible = ref(false)
const directoryLoading = ref(false)
const enterpriseList = ref([])
let enterpriseLoadingOnce = false
const enterpriseTotal = computed(() => enterpriseList.value.length)

/** 加载企业清单(名录卡片数量展示 + 弹窗数据共用) */
async function loadEnterpriseList() {
  if (enterpriseLoadingOnce) return
  enterpriseLoadingOnce = true
  directoryLoading.value = true
  try {
    const res = await directoryAPI.list()
    enterpriseList.value = res.data?.list || []
  } catch (e) {
    ElMessage.error(e?.msg || '查询企业名录失败')
  } finally {
    directoryLoading.value = false
    enterpriseLoadingOnce = false
  }
}

/** 打开企业名录弹窗(列表已在进入页面时预加载，避免重复请求) */
async function openDirectory() {
  directoryVisible.value = true
  if (!enterpriseList.value.length) {
    await loadEnterpriseList()
  }
}

const welcomeTitleMap = {
  enterprise: `欢迎回来，${brandText(user.company || user.username)}`,
  park_admin: '园区碳数据统筹面板',
  exchange: '碳交易所 · 撮合大屏',
  regulator: '监管核查 · 审计大屏'
}
const welcomeSubMap = {
  enterprise: '你的企业碳数据概览，完成能耗上报后数据自动更新',
  park_admin: '统筹管理园区内企业碳排放与碳积分流转',
  exchange: '全量挂单 · 撮合 · 成交统计 · 链上留痕',
  regulator: '链上审计 · 数据校验 · 风险告警 · ZK证明核查'
}
const welcomeTitle = welcomeTitleMap[role] || '仪表盘'
const welcomeSub = welcomeSubMap[role] || ''

/* ===== IoT设备在线数：与 IoT能耗设备管理页 同源真实数据 ===== */
const iotOnlineCount = ref(0)
async function loadIotOnline() {
  if (role !== 'enterprise') return
  try {
    const res = await iotAPI.devices({})
    const list = res.data?.list || res.data || []
    // 判定口径与 iot-devices.vue 保持一致：优先 online 字段，回退 status === 'online'
    iotOnlineCount.value = list.filter(d =>
      d.online !== undefined ? d.online : d.status === 'online'
    ).length
  } catch {
    /* IoT 统计失败不阻塞仪表盘其余数据 */
  }
}

/* 时间 */
const now = ref('')
onMounted(() => {
  const tick = () => {
    const d = new Date()
    now.value = `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}`
  }
  tick()
  setInterval(tick, 1000)
  loadIotOnline()
  loadEnterpriseList()
})

/* ========= 指标卡片（按角色定制） ========= */
const statsCards = computed(() => {
  const green = '#16a34a',  greenBg = 'rgba(22,163,74,0.10)'
  const blue  = '#0d9488',  blueBg  = 'rgba(16,185,129,0.10)'
  const amber = '#f59e0b',  amberBg = 'rgba(245,158,11,0.10)'
  const red   = '#ef4444',  redBg   = 'rgba(239,68,68,0.10)'
  const teal  = '#0d9488',  tealBg  = 'rgba(13,148,136,0.10)'

  if (role === 'enterprise') {
    return [
      { label: '累计能耗(kg)', value: 12840, icon: EditPen, color: green, colorBg: greenBg, trend: 8 },
      { label: '碳积分余额',    value:  1562, icon: Wallet, color: teal,  colorBg: tealBg,  trend: 15 },
      { label: '已挂单积分',    value:   420, icon: TrendCharts, color: blue, colorBg: blueBg, trend: -5 },
      { label: '链上交易笔数',  value:    37, icon: DataBoard, color: green, colorBg: greenBg, trend: 22 },
      { label: '风险告警',      value:     1, icon: Warning, color: red,   colorBg: redBg,   trend: -50 },
      { label: 'IoT设备在线',   value:     iotOnlineCount.value, icon: Cpu,   color: amber, colorBg: amberBg, trend: 0 },
    ]
  }
  if (role === 'park_admin') {
    return [
      { label: '园区企业总数',  value:   128, icon: OfficeBuilding, color: teal,  colorBg: tealBg,  trend: 4 },
      { label: '总碳排放量(t)', value:  1842, icon: EditPen,        color: green, colorBg: greenBg, trend: -3 },
      { label: '碳积分流通量',  value: 24680, icon: Wallet,        color: blue,  colorBg: blueBg,  trend: 12 },
      { label: '挂单总量',      value:   342, icon: TrendCharts,   color: amber, colorBg: amberBg, trend: 7 },
      { label: '风险企业',      value:    12, icon: Warning,       color: red,   colorBg: redBg,   trend: -10 },
      { label: 'AI 减排建议',   value:    28, icon: MagicStick,    color: green, colorBg: greenBg, trend: 30 },
    ]
  }
  if (role === 'exchange') {
    return [
      { label: '今日成交量',    value:   124, icon: TrendCharts,   color: green, colorBg: greenBg, trend: 18 },
      { label: '挂单总量',      value:  2840, icon: Collection,    color: teal,  colorBg: tealBg,  trend: 5 },
      { label: '成交总额(万)',  value:  3820, icon: Money,         color: blue,  colorBg: blueBg,  trend: 22 },
      { label: '现货/租赁/远期',value: '12/3/1', icon: Document,  color: amber, colorBg: amberBg, trend: 0 },
      { label: '纠纷仲裁中',    value:     5, icon: ScaleToOriginal, color: red, colorBg: redBg, trend: 15 },
      { label: '链上区块',      value:  2341, icon: DataBoard,     color: green, colorBg: greenBg, trend: 9 },
    ]
  }
  // regulator
  return [
    { label: '监管企业数',    value:   356, icon: OfficeBuilding, color: teal,  colorBg: tealBg,  trend: 2 },
    { label: '待核验能耗记录',value:    18, icon: EditPen,        color: amber, colorBg: amberBg, trend: -20 },
    { label: '链上总交易',    value: 12840, icon: TrendCharts,   color: blue,  colorBg: blueBg,  trend: 15 },
    { label: 'ZKP 校验通过率',value:  '98.6%', icon: Lock,        color: green, colorBg: greenBg, trend: 0 },
    { label: '风险告警',      value:    23, icon: Warning,       color: red,   colorBg: redBg,   trend: -15 },
    { label: '合规评分均值',  value:  '84.2', icon: Monitor,      color: teal,  colorBg: tealBg,  trend: 3 },
  ]
})

/* ========= 图表 option ========= */
const trendData = computed(() => {
  const r = role === 'enterprise' ? 600 : role === 'park_admin' ? 3500 : role === 'exchange' ? 2000 : 800
  return genMonthlyTrend(r, 0.3)
})

// 双序列折线图（能耗 + 积分）
const lineOption = computed(() => ({
  color: ['#16a34a', '#0d9488'],
  tooltip: { trigger: 'axis' },
  legend: { data: ['能耗(kg)', '碳积分'], top: 0, right: 10, icon: 'circle' },
  grid: { left: 10, right: 10, top: 40, bottom: 0, containLabel: true },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: trendData.value.labels,
    axisLine: { lineStyle: { color: '#e2e8f0' } },
    axisLabel: { color: '#475569', fontSize: 11 },  /* 深灰，白底上清晰可读 */
    axisTick: { show: false },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: '#e2e8f0' } },  /* 网格线加深 */
    axisLabel: { color: '#475569', fontSize: 11 }      /* 深灰，白底上清晰可读 */
  },
  series: [
    {
      name: '能耗(kg)',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2.5 },
      areaStyle: {
        type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: 'rgba(22,163,74,0.30)' },
          { offset: 1, color: 'rgba(22,163,74,0.02)' }
        ]
      },
      data: trendData.value.emission
    },
    {
      name: '碳积分',
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2 },
      data: trendData.value.credits
    }
  ]
}))

// 环形图（按角色切不同数据）
const pieOption = computed(() => {
  const data = role === 'exchange'
    ? genTradeComposition()
    : role === 'regulator'
    ? [{name:'合规', value:310},{name:'待核验', value:23},{name:'违规', value:12},{name:'注销', value:11}]
    : genCreditDistribution()
  return mkPieOption({ data })
})

// 柱状图（排行）—— 真实企业全称较长，每 5 字换行防重叠
const barOption = computed(() => {
  const { labels, values } = role === 'park_admin'
    ? genEnterpriseRanking(8)
    : genEnterpriseRanking(6)
  const opt = mkBarOption({ labels, values, colors: ['#16a34a', '#0d9488', '#0d9488', '#f59e0b', '#7c3aed', '#22c55e', '#06b6d4', '#ef4444'] })
  /* 浅色主题：坐标轴/网格线颜色加深，白底上清晰可读（不动 mkBarOption 工具函数） */
  opt.xAxis.axisLabel.color = '#475569'
  opt.xAxis.axisLabel.formatter = (name) => String(name).replace(/(.{5})/g, '$1\n')
  opt.yAxis.axisLabel.color = '#475569'
  opt.yAxis.splitLine.lineStyle.color = '#e2e8f0'
  return opt
})

// 雷达图（仅小微企业）
const radarOption = computed(() => {
  const r = genEnterpriseRadar()
  return {
    tooltip: {},
    legend: { data: r.series.map(s => s.name), bottom: 0, icon: 'circle' },
    radar: {
      indicator: r.indicator,
      center: ['50%', '50%'],
      radius: '65%',
      axisName: { color: '#64748b', fontSize: 11 },
      splitArea: { areaStyle: { color: ['rgba(22,163,74,0.03)', 'rgba(22,163,74,0.06)'] } },
      axisLine: { lineStyle: { color: '#e2e8f0' } },
      splitLine: { lineStyle: { color: '#e2e8f0' } },
    },
    series: [{
      type: 'radar',
      data: r.series.map((s, i) => ({
        name: s.name,
        value: s.value,
        areaStyle: { opacity: 0.25 },
        lineStyle: { width: 2 },
        itemStyle: { color: i === 0 ? '#16a34a' : '#0d9488' }
      }))
    }]
  }
})

// 园区列表（park_admin 用）
const parkList = computed(() => {
  return genParkDistribution().map((p, i) => ({
    ...p,
    trend: [4, -3, 8, -1][i % 4]
  }))
})

/* ========= 风险告警列表（按角色定制） ========= */
const riskList = computed(() => {
  const common = [
    { level: 'warn',  text: '近 7 天能耗超基准线 +18%',      action: 'energy-report' },
    { level: 'info',  text: '有 1 条挂单待成交超过 48 小时', action: 'exchange' },
  ]
  if (role === 'enterprise') return common
  if (role === 'park_admin') return [
    { level: 'danger', text: '绿能科技能耗异常波动 +35%', action: 'energy-report' },
    { level: 'warn',   text: '3 家企业连续 2 个月未上报', action: 'energy-report' },
    { level: 'info',   text: '链上区块高度落后 12 个',    action: 'datav' },
  ]
  if (role === 'exchange') return [
    { level: 'danger', text: '2 笔订单价格偏离大盘 >30%', action: 'arbitration' },
    { level: 'warn',   text: '现货/远期价差持续扩大',     action: 'exchange' },
  ]
  // regulator
  return [
    { level: 'danger', text: '绿能科技能耗数据 ZKP 校验失败', action: 'rollup-verify' },
    { level: 'danger', text: '有企业疑似双重提交能耗记录',   action: 'zkp-proof' },
    { level: 'warn',   text: '3 条交易上链超时未确认',       action: 'archive' },
  ]
})

</script>

<style scoped>
/* ============ 仪表盘根容器：白色底 ============ */
.dashboard {
  animation: fadeUp 0.4s ease;
  background: #ffffff;             /* 白色底色，与数据中台大屏卡片统一 */
  border-radius: var(--radius-lg);
  padding: 20px 22px;
  box-shadow: 0 2px 12px rgba(5,150,105,0.06);  /* 浅阴影浮起 */
}
@keyframes fadeUp { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }

/* ============ 企业名录(仅查看)：绿色主题，与仪表盘卡片风格一致 ============ */
.directory-card { padding-bottom: 14px; }
.directory-body { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-top: 8px; }
.directory-info { display: flex; align-items: center; gap: 12px; }
.dir-title { font-size: 14px; font-weight: 600; color: var(--text-primary, #1f2937); }
.dir-sub { font-size: 12px; color: var(--text-secondary, #6b7280); margin-top: 2px; }

/* ============ 欢迎横幅：青绿色渐变（用户要求保留不变） ============ */
.welcome-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  margin-bottom: 20px;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--primary-green), #16a34acc);
  color: #fff;
  box-shadow: 0 4px 16px rgba(22, 163, 74, 0.25);
}
.theme-dark .welcome-banner {
  background: linear-gradient(135deg, #0d9488, #06b6d4aa);
  box-shadow: 0 4px 24px rgba(13, 148, 136, 0.35);
}
.wb-left h1 { font-size: 18px; font-weight: 600; color: #fff; margin-bottom: 6px; }
.wb-left p { font-size: 13px; color: rgba(255,255,255,0.88); margin: 0; }
.wb-right { display: flex; align-items: center; gap: 12px; }
.wb-time {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-family: monospace;
  background: rgba(255,255,255,0.18);
  padding: 6px 12px; border-radius: 999px;
}
.wb-role-badge {
  padding: 6px 14px; border-radius: 999px;
  font-size: 12px; font-weight: 600; color: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
}

/* ============ 指标卡片：白色底 + 浅阴影 ============ */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
  margin-bottom: 20px;
}
.stat-card {
  position: relative;
  overflow: hidden;
  background: #ffffff;              /* 白色卡片 */
  border: 1px solid #e8f7ef;        /* 淡绿细线 */
  border-radius: var(--radius);
  padding: 0;
  transition: all 0.25s ease;
  animation: statFloat 0.5s ease both;
  box-shadow: 0 2px 8px rgba(5,150,105,0.05);  /* 浅阴影 */
}
@keyframes statFloat { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
.stat-card:hover {
  transform: translateY(-3px);
  border-color: var(--primary-green);
  box-shadow: 0 8px 24px rgba(5,150,105,0.12);  /* hover 阴影加深 */
}
.sc-bg {
  position: absolute; inset: 0;
  pointer-events: none;
  opacity: 0.35;  /* 降低透明度避免遮盖白底 */
}
.sc-body {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
}
.sc-icon {
  width: 42px; height: 42px;
  border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.sc-info { flex: 1; min-width: 0; }
.sc-num {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
  color: #1e293b;  /* 深色数字，白底上清晰 */
}
.sc-label { font-size: 12px; color: #64748b; margin-top: 2px; }
.sc-trend {
  display: flex; flex-direction: column; align-items: flex-end;
  font-size: 13px; font-weight: 600;
}
.sc-trend.up { color: #22c55e; }
.sc-trend.down { color: #ef4444; }
.sc-trend-label { font-size: 10px; font-weight: 400; opacity: 0.7; margin-top: 2px; }

/* ============ 图表卡片：白色底 + 浅阴影 ============ */
.charts-row {
  display: grid;
  gap: 16px;
  margin-bottom: 16px;
}
.charts-row:first-of-type { grid-template-columns: 2fr 1fr; }
.charts-row:nth-of-type(2) { grid-template-columns: 2fr 1fr; }
.chart-card {
  background: #ffffff;              /* 白色卡片，图表区域也是白色 */
  border: 1px solid #e8f7ef;
  border-radius: var(--radius);
  padding: 18px 20px;
  transition: border-color 0.2s;
  box-shadow: 0 2px 8px rgba(5,150,105,0.05);  /* 浅阴影 */
}
.chart-card:hover { border-color: var(--primary-green); box-shadow: 0 4px 16px rgba(5,150,105,0.10); }
.cc-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.cc-header h3 {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;  /* 深色标题 */
}
.cc-sub {
  font-size: 11px;
  color: #64748b;
  padding: 2px 8px;
  background: #f0fdf4;  /* 淡绿色 tag 背景 */
  border-radius: 999px;
}
.cc-tag {
  font-size: 11px;
  padding: 2px 10px;
  border-radius: 999px;
  font-weight: 600;
}
.cc-tag.danger {
  background: rgba(239,68,68,0.12);
  color: #ef4444;
}

/* ============ 风险列表 ============ */
.risk-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 280px;
  overflow-y: auto;
}
.risk-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 12.5px;
  border-left: 3px solid transparent;
  background: #f8fffb;   /* 极浅绿底（白底上的微差） */
}
.risk-item.danger { border-left-color: #ef4444; background: rgba(239,68,68,0.06); color: #dc2626; }
.risk-item.warn   { border-left-color: #f59e0b; background: rgba(245,158,11,0.08); color: #d97706; }
.risk-item.info   { border-left-color: #0d9488; background: rgba(16,185,129,0.08); color: #2563eb; }
.risk-item span { flex: 1; color: #1e293b; }
.risk-btn {
  font-size: 11px;
  padding: 3px 10px;
  border: 1px solid currentColor;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  transition: all 0.2s;
}
.risk-btn:hover { background: currentColor; color: #fff; }
.risk-empty {
  display: flex; flex-direction: column; align-items: center; gap: 8px;
  padding: 24px; color: #64748b; font-size: 12px;
}

/* ============ 园区表（park_admin） ============ */
.park-table {
  font-size: 12.5px;
  border-top: none;
}
.park-table .pt-head {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  padding: 10px 14px;
  color: #ffffff;
  background: #059669;
  font-weight: 600;
  border-radius: 6px 6px 0 0;
}
.park-table .pt-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  padding: 12px 14px;
  border-bottom: 1px solid #e8f7ef;
  align-items: center;
  color: #334155;
  background: #ffffff;
  transition: background 0.15s;
}
.park-table .pt-row:hover { background: #f0fdf4; }
.park-table .pt-row:last-child { border-bottom: none; }
.park-table .highlight { color: var(--primary-green); font-weight: 600; }
.park-table .up { color: #22c55e; font-weight: 500; }
.park-table .down { color: #ef4444; font-weight: 500; }

/* ============ 深色主题额外样式（保留不动） ============ */
.theme-dark .stat-card .sc-bg { opacity: 0.35; }
.theme-dark .risk-item.danger { background: rgba(239,68,68,0.12); }
.theme-dark .risk-item.warn   { background: rgba(245,158,11,0.12); }
.theme-dark .risk-item.info   { background: rgba(16,185,129,0.12); }

@media (max-width: 900px) {
  .charts-row:first-of-type,
  .charts-row:nth-of-type(2) { grid-template-columns: 1fr; }
}
</style>