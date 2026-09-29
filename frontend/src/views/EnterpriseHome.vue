<template>
  <!-- 小微企业 · 独立首页数据面板(指标/图表/业务入口与其他角色完全独立，样式 scoped) -->
  <Layout>
    <div class="page">
      <!-- 角色身份标题 -->
      <div class="hero">
        <div class="hero-left">
          <span class="hero-icon"><OfficeBuilding /></span>
          <div>
            <h1>小微企业工作台</h1>
            <p>手动能耗上报 → SHA-256 碳积分核算 → 挂单交易流转，企业侧碳资产管理闭环</p>
          </div>
        </div>
        <span class="hero-tag">小微企业</span>
      </div>

      <!-- 四个数字统计指标(始终取自真实接口，不受图表预览影响) -->
      <div class="stat-grid">
        <div class="stat-card">
          <div class="sc-icon" style="background:#fff3dc"><Lightning /></div>
          <div>
            <div class="sc-value">{{ fmt(totalElectricity) }}</div>
            <div class="sc-label">总用电量(kWh)</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#ddf6ee"><Coin /></div>
          <div>
            <div class="sc-value">{{ fmt(stats.total_credits) }}</div>
            <div class="sc-label">累计碳积分</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#e2edff"><Link /></div>
          <div>
            <div class="sc-value">{{ onChainCount }}</div>
            <div class="sc-label">已上链记录数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#eee9ff"><TrendCharts /></div>
          <div>
            <div class="sc-value">{{ fmt(stats.total_emission) }}</div>
            <div class="sc-label">总碳排放(kgCO₂)</div>
          </div>
        </div>
      </div>

      <!-- 图表模拟预览开关：仅填充图表 series，指标卡数字仍来自真实接口 -->
      <MockPreviewBar v-model="previewMock" />

      <!-- 三个可视化图表 -->
      <div class="chart-grid">
        <div class="chart-card">
          <div class="cc-head">
            <h3><TrendCharts class="cc-ico" /> 企业碳积分趋势</h3>
            <span class="cc-sub">按积分生成时间累计</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('trend', el)"></div>
        </div>

        <div class="chart-card">
          <div class="cc-head">
            <h3><DataLine class="cc-ico" /> 历史能耗构成统计</h3>
            <span class="cc-sub">电 / 天然气 / 水 累计用量</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('usage', el)"></div>
        </div>

        <div class="chart-card">
          <div class="cc-head">
            <h3><PieChart class="cc-ico" /> 碳积分状态分布</h3>
            <span class="cc-sub">可用 / 锁定 / 已售出</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('status', el)"></div>
        </div>
      </div>

      <!-- 我的业务功能入口(子路由跳转) -->
      <div class="block-title">
        <h2>我的业务模块</h2>
        <span>点击进入对应功能子页面</span>
      </div>
      <div class="module-grid">
        <button v-for="m in modules" :key="m.path" class="module-card" @click="router.push(m.path)">
          <span class="mc-icon"><component :is="m.icon" /></span>
          <span class="mc-info">
            <b>{{ m.name }}</b>
            <i>{{ m.desc }}</i>
          </span>
          <Right class="mc-arrow" />
        </button>
      </div>
    </div>
  </Layout>
</template>

<script setup>
// 小微企业首页数据面板：
//  - 顶部统计卡片数字始终来自真实业务接口；
//  - 图表 series 与统计卡数据源分开处理：打开"图表模拟预览"后图表填充内置演示数据集，
//    关闭预览且有真实数据时绘制真实图表，真实数据为空时图表区居中渲染统一"暂无数据"空态。
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import MockPreviewBar from '../components/MockPreviewBar.vue'
import { energyAPI, carbonAPI } from '../api/index.js'
import { initChart, mkLineOption, mkBarOption, mkPieOption, emptyOption, PALETTE } from '../utils/echarts.js'
import { MOCK } from '../utils/mockChartData.js'

const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{}')

// ========== 真实业务数据 ==========
const energyRecords = ref([])
const credits = ref([])
const stats = ref({})
const loaded = ref(false)

// ========== 图表模拟预览开关 ==========
const previewMock = ref(false)

// ========== 图表实例管理 ==========
// ECharts 初始化规则：ref 只登记容器，真正 init 在 nextTick(DOM 挂载)后统一执行，
// 避免容器尺寸未加载导致初始化空白；图表容器已由 CSS 固定 min-height 兜底。
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

// ========== 四个统计指标(真实接口) ==========
const totalElectricity = computed(() => energyRecords.value.reduce((s, r) => s + (r.electricity || 0), 0))
const onChainCount = computed(() =>
  energyRecords.value.filter(r => r.on_chain).length + credits.value.filter(c => c.on_chain).length
)
function fmt(v) {
  const n = Number(v) || 0
  return Number.isInteger(n) ? n.toLocaleString() : n.toFixed(1)
}

// ========== 图表数据(与统计卡分开处理) ==========
function shortDate(t) {
  if (!t) return '-'
  const d = new Date(t)
  return isNaN(d.getTime()) ? '-' : `${d.getMonth() + 1}/${d.getDate()}`
}
const creditTrend = computed(() => {
  const sorted = [...credits.value].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  let acc = 0
  return sorted.map(c => {
    acc += Number(c.carbon_credits) || 0
    return { label: shortDate(c.created_at), value: Number(acc.toFixed(1)) }
  })
})
const trendLabels = computed(() => creditTrend.value.map(x => x.label))
const trendValues = computed(() => creditTrend.value.map(x => x.value))
const usageValues = computed(() => {
  const er = energyRecords.value
  return [
    Number(er.reduce((s, r) => s + (r.electricity || 0), 0).toFixed(1)),
    Number(er.reduce((s, r) => s + (r.gas || 0), 0).toFixed(1)),
    Number(er.reduce((s, r) => s + (r.water || 0), 0).toFixed(1))
  ]
})
const statusData = computed(() => {
  const q = { available: 0, locked: 0, sold: 0 }
  credits.value.forEach(c => { q[c.status] = (q[c.status] || 0) + (Number(c.carbon_credits) || 0) })
  return [
    { name: '可用', value: Number(q.available.toFixed(1)) },
    { name: '锁定(挂单中)', value: Number(q.locked.toFixed(1)) },
    { name: '已售出', value: Number(q.sold.toFixed(1)) }
  ]
})
const hasCreditData = computed(() => credits.value.some(c => Number(c.carbon_credits) > 0))

// ========== 图表绘制(空态规则统一) ==========
function paint() {
  if (!loaded.value) return
  if (previewMock.value) {
    // 预览模式：series 使用内置模拟数据集(指标卡数字不受影响)
    setChart('trend', mkLineOption(MOCK.enterprise.trend))
    setChart('usage', mkBarOption({ labels: MOCK.enterprise.usage.labels, values: MOCK.enterprise.usage.values, colors: PALETTE }))
    setChart('status', mkPieOption({ data: MOCK.enterprise.status.data }))
    return
  }
  // 业务模式：有真实数据画真实图表，无数据时居中渲染统一空态文案(不留白)
  setChart('trend', credits.value.length
    ? mkLineOption({ labels: trendLabels.value, values: trendValues.value })
    : emptyOption())
  setChart('usage', energyRecords.value.length
    ? mkBarOption({ labels: ['用电', '天然气', '用水'], values: usageValues.value, colors: PALETTE })
    : emptyOption())
  setChart('status', hasCreditData.value
    ? mkPieOption({ data: statusData.value })
    : emptyOption())
}
watch(previewMock, () => paint())

// ========== 业务入口 ==========
const modules = [
  { path: '/enterprise/energy', name: '能耗数据管理', icon: 'Histogram', desc: '手动录入能耗并上报' },
  { path: '/enterprise/credits', name: '碳积分管理', icon: 'Coin', desc: '核算记录与积分余额' },
  { path: '/enterprise/sell', name: '挂单交易', icon: 'TrendCharts', desc: '可用积分挂单出售' },
  { path: '/enterprise/chain', name: '区块链溯源', icon: 'Link', desc: '哈希存证与溯源校验' },
  { path: '/enterprise/advice', name: 'AI减排建议', icon: 'MagicStick', desc: '大模型生成降碳方案' }
]

// ========== 加载 ==========
async function refreshData() {
  loaded.value = true
  await ensureCharts()
  if (!alive) return
  paint() // 先绘制(真实数据未返回时即为空态文案)，保证图表区不留白
  try {
    const [er, cr, st] = await Promise.all([
      energyAPI.list({ enterprise_id: user.id, page_size: 500 }),
      carbonAPI.myCredits({ enterprise_id: user.id, page_size: 200 }),
      carbonAPI.stats({ enterprise_id: user.id })
    ])
    if (!alive) return
    energyRecords.value = er.data.list || []
    credits.value = cr.data.list || []
    stats.value = st.data || {}
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
.hero-icon { width: 46px; height: 46px; border-radius: 12px; background: linear-gradient(135deg, #0f766e, #14b8a6); color: #fff; display: flex; align-items: center; justify-content: center; box-shadow: 0 6px 16px rgba(13, 148, 136, 0.35); }
.hero-icon svg { width: 24px; height: 24px; }
.hero h1 { font-size: 22px; font-weight: 700; color: var(--text-1); letter-spacing: -0.01em; }
.hero p { font-size: 12px; color: var(--text-4); margin-top: 2px; }
.hero-tag { font-size: 12px; font-weight: 600; color: #0f766e; background: #ecfdf5; border: 1px solid #d1fae5; border-radius: 999px; padding: 4px 14px; }

.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 22px; }
.stat-card { display: flex; align-items: center; gap: 14px; background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 18px; box-shadow: var(--shadow-sm); }
.sc-icon { width: 44px; height: 44px; border-radius: 11px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.sc-icon svg { width: 22px; height: 22px; color: #0f766e; }
.sc-value { font-size: 24px; font-weight: 700; color: var(--text-1); line-height: 1.2; letter-spacing: -0.01em; }
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
.module-card:hover { border-color: #99f6e4; transform: translateY(-2px); box-shadow: 0 8px 18px rgba(13, 148, 136, 0.12); }
.mc-icon { width: 38px; height: 38px; border-radius: 10px; background: #ecfdf5; color: #0f766e; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.mc-icon svg { width: 19px; height: 19px; }
.mc-info { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.mc-info b { font-size: 13px; font-weight: 600; color: var(--text-1); }
.mc-info i { font-style: normal; font-size: 11px; color: var(--text-4); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mc-arrow { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }

@media (max-width: 768px) {
  .chart-card .cc-body { min-height: 210px; height: 210px; }
}
</style>
