<template>
  <!-- 监管核查 · 独立首页数据面板(指标/图表/业务入口与其他角色完全独立，样式 scoped) -->
  <Layout>
    <div class="page">
      <!-- 角色身份标题 -->
      <div class="hero">
        <div class="hero-left">
          <span class="hero-icon"><Monitor /></span>
          <div>
            <h1>监管核查审计面板</h1>
            <p>全链路链上存证核查 · 哈希审计 · 防篡改校验，保障碳数据真实可信</p>
          </div>
        </div>
        <span class="hero-tag">监管核查</span>
      </div>

      <!-- 四个数字统计指标(始终取自真实接口，不受图表预览影响) -->
      <div class="stat-grid">
        <div class="stat-card">
          <div class="sc-icon" style="background:#ddf6ee"><Link /></div>
          <div>
            <div class="sc-value">{{ blockTotal }}</div>
            <div class="sc-label">链上区块总数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#fef3c7"><Coin /></div>
          <div>
            <div class="sc-value">{{ fmt(stats.total_issued_credits) }}</div>
            <div class="sc-label">总发行积分</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#e2edff"><OfficeBuilding /></div>
          <div>
            <div class="sc-value">{{ stats.enterprise_count || 0 }}</div>
            <div class="sc-label">企业总数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#eee9ff"><Files /></div>
          <div>
            <div class="sc-value">{{ stats.transaction_count || 0 }}</div>
            <div class="sc-label">交易笔数</div>
          </div>
        </div>
      </div>

      <!-- 图表模拟预览开关：仅填充图表 series，指标卡数字仍来自真实接口 -->
      <MockPreviewBar v-model="previewMock" />

      <!-- 三个可视化图表 -->
      <div class="chart-grid">
        <div class="chart-card">
          <div class="cc-head">
            <h3><TrendCharts class="cc-ico" /> 上链存证趋势</h3>
            <span class="cc-sub">按日期新增区块数量</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('trend', el)"></div>
        </div>

        <div class="chart-card">
          <div class="cc-head">
            <h3><DataAnalysis class="cc-ico" /> 上链存证数量统计</h3>
            <span class="cc-sub">按业务类型统计存证区块</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('types', el)"></div>
        </div>

        <div class="chart-card">
          <div class="cc-head">
            <h3><PieChart class="cc-ico" /> 平台碳积分流通分布</h3>
            <span class="cc-sub">可用 / 锁定 / 已交易</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('flow', el)"></div>
        </div>
      </div>

      <!-- 监管业务功能入口 -->
      <div class="block-title">
        <h2>监管业务模块</h2>
        <span>点击进入对应功能子页面</span>
      </div>
      <div class="module-grid">
        <button v-for="m in modules" :key="m.path" class="module-card" @click="router.push(m.path)">
          <span class="mc-icon"><component :is="m.icon" /></span>
          <span class="mc-info"><b>{{ m.name }}</b><i>{{ m.desc }}</i></span>
          <Right class="mc-arrow" />
        </button>
      </div>
    </div>
  </Layout>
</template>

<script setup>
// 监管核查首页数据面板：
//  - 顶部统计卡片数字始终来自链上/平台真实接口；
//  - 图表 series 与统计卡数据源分开处理：打开"图表模拟预览"后图表填充内置演示数据集，
//    关闭预览且有真实数据时绘制真实图表，真实数据为空时图表区居中渲染统一"暂无数据"空态。
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import MockPreviewBar from '../components/MockPreviewBar.vue'
import { chainAPI, regulatorAPI } from '../api/index.js'
import { initChart, mkLineOption, mkBarOption, mkPieOption, emptyOption, PALETTE } from '../utils/echarts.js'
import { MOCK } from '../utils/mockChartData.js'

const router = useRouter()
const blocks = ref([])
const blockTotal = ref(0)
const stats = ref({})
const loaded = ref(false)

// ========== 图表模拟预览开关 ==========
const previewMock = ref(false)

// ========== 图表实例管理 ==========
// ECharts 初始化规则：ref 只登记容器，真正 init 在 nextTick(DOM 挂载)后统一执行；
// 容器已由 CSS 固定 min-height，避免因 DOM 尺寸未加载导致空白。
const chartEls = {}
const charts = {}
let alive = true
function bindChart(key, el) { if (el) chartEls[key] = el }
async function ensureCharts() {
  await nextTick()
  if (!alive) return
  Object.keys(chartEls).forEach(k => {
    const el = chartEls[k]
    if (el && !charts[k]) charts[k] = initChart(el)
  })
}
function setChart(key, option) { if (charts[key]) charts[key].setOption(option, true) }
function onResize() { Object.values(charts).forEach(c => c && c.resize()) }

// ========== 图表数据(与统计卡分开处理) ==========
function toTime(t) {
  if (t == null) return 0
  if (typeof t === 'number') return t > 1e12 ? t : t * 1000
  return new Date(t).getTime()
}
function dayKey(t) {
  const d = new Date(toTime(t))
  if (isNaN(d.getTime())) return '-'
  return `${d.getMonth() + 1}/${d.getDate()}`
}
const trendData = computed(() => {
  const sorted = [...blocks.value].sort((a, b) => toTime(a.timestamp) - toTime(b.timestamp))
  const m = new Map()
  sorted.forEach(b => {
    const k = dayKey(b.timestamp)
    m.set(k, (m.get(k) || 0) + 1)
  })
  return [...m.entries()].map(([label, value]) => ({ label, value }))
})
const trendLabels = computed(() => trendData.value.map(x => x.label))
const trendValues = computed(() => trendData.value.map(x => x.value))

const TYPE_NAMES = { energy: '能耗数据', credit: '碳积分', transaction: '交易', report: 'AI报告', genesis: '创世块', operation: '操作日志' }
const TYPE_ORDER = ['energy', 'credit', 'transaction', 'report', 'genesis', 'operation']
const typeStat = computed(() => {
  const m = new Map()
  blocks.value.forEach(b => m.set(b.data_type || 'unknown', (m.get(b.data_type || 'unknown') || 0) + 1))
  const extra = [...m.keys()].filter(k => !TYPE_ORDER.includes(k))
  return [...TYPE_ORDER, ...extra]
    .filter(k => m.has(k))
    .map(k => ({ name: TYPE_NAMES[k] || k, count: m.get(k) }))
})
const typeLabels = computed(() => typeStat.value.map(x => x.name))
const typeValues = computed(() => typeStat.value.map(x => x.count))

const flowData = computed(() => [
  { name: '可用', value: Number((stats.value.total_available_credits || 0).toFixed(1)) },
  { name: '锁定(挂单中)', value: Number((stats.value.total_locked_credits || 0).toFixed(1)) },
  { name: '已交易', value: Number((stats.value.total_traded_credits || 0).toFixed(1)) }
])
const hasFlow = computed(() =>
  ['total_available_credits', 'total_locked_credits', 'total_traded_credits'].some(k => Number(stats.value[k]) > 0)
)
function fmt(v) {
  const n = Number(v) || 0
  return Number.isInteger(n) ? n.toLocaleString() : n.toFixed(1)
}

// ========== 图表绘制(空态规则统一) ==========
function paint() {
  if (!loaded.value) return
  if (previewMock.value) {
    // 预览模式：series 使用内置模拟数据集(指标卡数字不受影响)
    setChart('trend', mkLineOption(MOCK.regulator.trend))
    setChart('types', mkBarOption({ labels: MOCK.regulator.types.labels, values: MOCK.regulator.types.values, colors: PALETTE }))
    setChart('flow', mkPieOption({ data: MOCK.regulator.flow.data }))
    return
  }
  // 业务模式：有真实数据画真实图表，无数据时居中渲染统一空态文案(不留白)
  setChart('trend', blocks.value.length
    ? mkLineOption({ labels: trendLabels.value, values: trendValues.value })
    : emptyOption())
  setChart('types', blocks.value.length
    ? mkBarOption({ labels: typeLabels.value, values: typeValues.value, colors: PALETTE })
    : emptyOption())
  setChart('flow', hasFlow.value
    ? mkPieOption({ data: flowData.value })
    : emptyOption())
}
watch(previewMock, () => paint())

// ========== 业务入口 ==========
const modules = [
  { path: '/regulator/chain', name: '链上溯源', icon: 'Connection', desc: '全部区块存证记录' },
  { path: '/regulator/verify', name: '数据校验', icon: 'CircleCheck', desc: 'SHA-256 数据完整性校验' },
  { path: '/regulator/alert', name: '风险告警', icon: 'Warning', desc: '企业减排风险识别' },
  { path: '/regulator/users', name: '用户管理', icon: 'User', desc: '平台账号与角色查看' }
]

// ========== 加载 ==========
async function refreshData() {
  loaded.value = true
  await ensureCharts()
  if (!alive) return
  paint() // 先绘制(真实数据未返回时即为空态文案)，保证图表区不留白
  try {
    const [b, s] = await Promise.all([
      chainAPI.blocks({ page_size: 100 }),
      regulatorAPI.platformStats()
    ])
    if (!alive) return
    blocks.value = b.data.list || []
    blockTotal.value = b.data.total
    stats.value = s.data || {}
  } catch (e) { console.error(e) }
  if (!alive) return
  await nextTick()
  paint()
}
onMounted(() => { refreshData(); window.addEventListener('resize', onResize) })
onBeforeUnmount(() => {
  alive = false
  window.removeEventListener('resize', onResize)
  Object.values(charts).forEach(c => c && c.dispose())
})
</script>

<style scoped>
.page { max-width: 1160px; margin: 0 auto; }
.hero { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 20px; flex-wrap: wrap; }
.hero-left { display: flex; align-items: center; gap: 14px; }
.hero-icon { width: 46px; height: 46px; border-radius: 12px; background: linear-gradient(135deg, #6d28d9, #8b5cf6); color: #fff; display: flex; align-items: center; justify-content: center; box-shadow: 0 6px 16px rgba(124, 58, 237, 0.3); }
.hero-icon svg { width: 24px; height: 24px; }
.hero h1 { font-size: 22px; font-weight: 700; color: var(--text-1); }
.hero p { font-size: 12px; color: var(--text-4); margin-top: 2px; }
.hero-tag { font-size: 12px; font-weight: 600; color: #6d28d9; background: #f5f3ff; border: 1px solid #ede9fe; border-radius: 999px; padding: 4px 14px; }

.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 22px; }
.stat-card { display: flex; align-items: center; gap: 14px; background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 18px; box-shadow: var(--shadow-sm); }
.sc-icon { width: 44px; height: 44px; border-radius: 11px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.sc-icon svg { width: 22px; height: 22px; color: #7c3aed; }
.sc-value { font-size: 24px; font-weight: 700; color: var(--text-1); line-height: 1.2; }
.sc-label { font-size: 12px; color: var(--text-3); margin-top: 2px; }

.chart-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px; margin-bottom: 26px; }
.chart-card { background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); box-shadow: var(--shadow-sm); padding: 16px 18px; min-width: 0; }
.cc-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.cc-head h3 { display: flex; align-items: center; gap: 6px; font-size: 14px; font-weight: 600; color: var(--text-1); margin: 0; }
.cc-ico { width: 16px; height: 16px; color: var(--primary); }
.cc-sub { font-size: 11px; color: var(--text-4); }
/* 图表容器固定最小高度：防止 ECharts 因 DOM 尺寸未加载导致空白 */
.cc-body { position: relative; min-height: 240px; height: 240px; }

.block-title { display: flex; align-items: baseline; gap: 10px; margin: 6px 0 14px; }
.block-title h2 { margin: 0; font-size: 16px; font-weight: 600; color: var(--text-1); }
.block-title span { font-size: 12px; color: var(--text-4); }

.module-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 14px; margin-bottom: 26px; }
.module-card { display: flex; align-items: center; gap: 10px; text-align: left; background: #fff; border: 1px solid var(--line); border-radius: var(--radius); padding: 14px; cursor: pointer; box-shadow: var(--shadow-sm); transition: all 0.15s ease; }
.module-card:hover { border-color: #ddd6fe; transform: translateY(-2px); box-shadow: 0 8px 18px rgba(124, 58, 237, 0.12); }
.mc-icon { width: 38px; height: 38px; border-radius: 10px; background: #f5f3ff; color: #7c3aed; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.mc-icon svg { width: 19px; height: 19px; }
.mc-info { flex: 1; min-width: 0; display: flex; flex-direction: column; text-align: left; }
.mc-info b { font-size: 13px; font-weight: 600; color: var(--text-1); }
.mc-info i { font-style: normal; font-size: 11px; color: var(--text-4); }
.mc-arrow { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }

@media (max-width: 768px) {
  .cc-body { min-height: 210px; height: 210px; }
}
</style>
