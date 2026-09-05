<template>
  <!-- 碳交易所 · 独立首页数据面板(指标/图表/业务入口与其他角色完全独立，样式 scoped) -->
  <Layout>
    <div class="page">
      <!-- 角色身份标题 -->
      <div class="hero">
        <div class="hero-left">
          <span class="hero-icon"><TrendCharts /></span>
          <div>
            <h1>碳交易所撮合面板</h1>
            <p>园区内所有入驻企业均可作买方(园区内部交易)，亦支持跨园区购买；撮合统一由本角色完成，园区管理员不参与</p>
          </div>
        </div>
        <span class="hero-tag">碳交易所</span>
      </div>

      <!-- 四个数字统计指标(始终取自真实接口，不受图表预览影响) -->
      <div class="stat-grid">
        <div class="stat-card">
          <div class="sc-icon" style="background:#fff3dc"><Files /></div>
          <div>
            <div class="sc-value">{{ orders.length }}</div>
            <div class="sc-label">挂单中(待成交)</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#ddf6ee"><CircleCheck /></div>
          <div>
            <div class="sc-value">{{ transactions.length }}</div>
            <div class="sc-label">已成交笔数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#e2edff"><Coin /></div>
          <div>
            <div class="sc-value">{{ fmt(totalQty) }}</div>
            <div class="sc-label">累计成交碳积分</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#eee9ff"><Wallet /></div>
          <div>
            <div class="sc-value">¥{{ fmt(totalAmount) }}</div>
            <div class="sc-label">累计交易金额</div>
          </div>
        </div>
      </div>

      <!-- 图表模拟预览开关：仅填充图表 series，指标卡数字与业务表格仍来自真实接口 -->
      <MockPreviewBar v-model="previewMock" />

      <!-- 三个可视化图表 -->
      <div class="chart-grid">
        <div class="chart-card">
          <div class="cc-head">
            <h3><TrendCharts class="cc-ico" /> 碳积分成交趋势</h3>
            <span class="cc-sub">按成交日期汇总成交量</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('trend', el)"></div>
        </div>

        <div class="chart-card">
          <div class="cc-head">
            <h3><Files class="cc-ico" /> 挂单数据概览</h3>
            <span class="cc-sub">待成交挂单量排行</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('orders', el)"></div>
        </div>

        <div class="chart-card">
          <div class="cc-head">
            <h3><PieChart class="cc-ico" /> 成交方成交分布</h3>
            <span class="cc-sub">按卖方企业累计成交量占比</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('seller', el)"></div>
        </div>
      </div>

      <!-- 卖方挂单浏览表格(真实业务数据) -->
      <div class="table-block">
        <div class="block-title no-mb">
          <h2>卖方挂单浏览</h2>
          <span v-if="orders.length" class="muted">共 {{ orders.length }} 笔待成交挂单</span>
        </div>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>挂单编号</th><th>卖方企业</th><th>卖方园区</th><th>单价(元/积分)</th><th>挂单数量</th><th>挂单时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="o in orders" :key="o.id">
                <td class="mono">{{ o.order_no }}</td>
                <td>{{ o.seller?.company || ('卖方#' + o.seller_id) }}</td>
                <td>{{ o.seller?.park_id ? ('园区' + o.seller.park_id) : '—' }}</td>
                <td>{{ fmt(o.unit_price) }}</td>
                <td>{{ fmt(o.quantity) }}</td>
                <td>{{ fmtTime(o.created_at) }}</td>
              </tr>
              <tr v-if="!orders.length">
                <td colspan="6" class="empty-cell">暂无挂单数据，等待企业提交碳积分挂单</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 交易所业务功能入口 -->
      <div class="block-title">
        <h2>交易所业务模块</h2>
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
// 碳交易所首页数据面板：
//  - 顶部统计卡片数字与"卖方挂单浏览"业务表格始终来自真实接口；
//  - 图表 series 与统计卡数据源分开处理：打开"图表模拟预览"后图表填充内置演示数据集，
//    关闭预览且有真实数据时绘制真实图表，真实数据为空时图表区居中渲染统一"暂无数据"空态。
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import MockPreviewBar from '../components/MockPreviewBar.vue'
import { exchangeAPI } from '../api/index.js'
import { initChart, mkLineOption, mkBarOption, mkPieOption, emptyOption, PALETTE } from '../utils/echarts.js'
import { MOCK } from '../utils/mockChartData.js'

const router = useRouter()
const orders = ref([])
const transactions = ref([])
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

// ========== 指标(真实接口) ==========
const totalQty = computed(() => transactions.value.reduce((s, t) => s + (Number(t.quantity) || 0), 0))
const totalAmount = computed(() => transactions.value.reduce((s, t) => s + (Number(t.total_amount) || 0), 0))
function fmt(v) {
  const n = Number(v) || 0
  if (Number.isInteger(n)) return n.toLocaleString()
  return n.toFixed(2)
}

// ========== 图表数据(与统计卡分开处理) ==========
function shortDate(t) {
  if (!t) return '-'
  const d = new Date(t)
  return isNaN(d.getTime()) ? '-' : `${d.getMonth() + 1}/${d.getDate()}`
}
function fmtTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return String(t)
  const p = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}
const trendMap = computed(() => {
  const sorted = [...transactions.value].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  const m = new Map()
  sorted.forEach(t => {
    const k = shortDate(t.created_at)
    m.set(k, (m.get(k) || 0) + (Number(t.quantity) || 0))
  })
  return [...m.entries()].map(([label, value]) => ({ label, value: Number(value.toFixed(1)) }))
})
const trendLabels = computed(() => trendMap.value.map(x => x.label))
const trendValues = computed(() => trendMap.value.map(x => x.value))
const orderTop = computed(() =>
  [...orders.value].sort((a, b) => (Number(b.quantity) || 0) - (Number(a.quantity) || 0)).slice(0, 10)
)
const orderNames = computed(() => orderTop.value.map(o => String(o.order_no || o.id)))
const orderValues = computed(() => orderTop.value.map(o => Number(o.quantity) || 0))
const sellerData = computed(() => {
  const m = new Map()
  transactions.value.forEach(t => {
    const n = t.seller?.company || ('卖方#' + t.seller_id)
    m.set(n, (m.get(n) || 0) + (Number(t.quantity) || 0))
  })
  return [...m.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 6)
    .map(([name, value], i) => ({
      name: name.length > 10 ? name.slice(0, 10) + '…' : name,
      value: Number(value.toFixed(1)),
      itemStyle: { color: PALETTE[i % PALETTE.length] }
    }))
})

// ========== 图表绘制(空态规则统一) ==========
function paint() {
  if (!loaded.value) return
  if (previewMock.value) {
    // 预览模式：series 使用内置模拟数据集(指标卡数字与业务表格不受影响)
    setChart('trend', mkLineOption(MOCK.exchange.trend))
    setChart('orders', mkBarOption({ labels: MOCK.exchange.orderRank.labels, values: MOCK.exchange.orderRank.values, colors: PALETTE }))
    setChart('seller', mkPieOption({ data: MOCK.exchange.seller.data }))
    return
  }
  // 业务模式：有真实数据画真实图表，无数据时居中渲染统一空态文案(不留白)
  setChart('trend', transactions.value.length
    ? mkLineOption({ labels: trendLabels.value, values: trendValues.value })
    : emptyOption())
  setChart('orders', orders.value.length
    ? mkBarOption({ labels: orderNames.value, values: orderValues.value, colors: PALETTE })
    : emptyOption())
  setChart('seller', transactions.value.length
    ? mkPieOption({ data: sellerData.value })
    : emptyOption())
}
watch(previewMock, () => paint())

// ========== 业务入口 ==========
const modules = [
  { path: '/exchange/match', name: '交易撮合', icon: 'Connection', desc: '买卖双方成交撮合' },
  { path: '/exchange/verify', name: '积分核验', icon: 'CircleCheck', desc: '挂单碳积分合规核验' },
  { path: '/exchange/history', name: '成交记录', icon: 'List', desc: '历史成交与存证明细' }
]

// ========== 加载 ==========
async function refreshData() {
  loaded.value = true
  await ensureCharts()
  if (!alive) return
  paint() // 先绘制(真实数据未返回时即为空态文案)，保证图表区不留白
  try {
    const [od, tx] = await Promise.all([
      exchangeAPI.orders({ page_size: 100 }),
      exchangeAPI.transactions({ page_size: 200 })
    ])
    if (!alive) return
    orders.value = od.data.list || []
    transactions.value = tx.data.list || []
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
.hero-icon { width: 46px; height: 46px; border-radius: 12px; background: linear-gradient(135deg, #d97706, #f59e0b); color: #fff; display: flex; align-items: center; justify-content: center; box-shadow: 0 6px 16px rgba(217, 119, 6, 0.3); }
.hero-icon svg { width: 24px; height: 24px; }
.hero h1 { font-size: 22px; font-weight: 700; color: var(--text-1); }
.hero p { font-size: 12px; color: var(--text-4); margin-top: 2px; }
.hero-tag { font-size: 12px; font-weight: 600; color: #b45309; background: #fffbeb; border: 1px solid #fef3c7; border-radius: 999px; padding: 4px 14px; }

.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 22px; }
.stat-card { display: flex; align-items: center; gap: 14px; background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 18px; box-shadow: var(--shadow-sm); }
.sc-icon { width: 44px; height: 44px; border-radius: 11px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.sc-icon svg { width: 22px; height: 22px; color: #b45309; }
.sc-value { font-size: 22px; font-weight: 700; color: var(--text-1); line-height: 1.2; }
.sc-label { font-size: 12px; color: var(--text-3); margin-top: 2px; }

.chart-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px; margin-bottom: 26px; }
.chart-card { background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); box-shadow: var(--shadow-sm); padding: 16px 18px; min-width: 0; }
.cc-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.cc-head h3 { display: flex; align-items: center; gap: 6px; font-size: 14px; font-weight: 600; color: var(--text-1); margin: 0; }
.cc-ico { width: 16px; height: 16px; color: var(--primary); }
.cc-sub { font-size: 11px; color: var(--text-4); }
/* 图表容器固定最小高度：防止 ECharts 因 DOM 尺寸未加载导致空白 */
.cc-body { position: relative; min-height: 240px; height: 240px; }

.table-block { margin-bottom: 8px; }
.block-title { display: flex; align-items: baseline; gap: 10px; margin: 6px 0 14px; }
.block-title.no-mb { margin-top: 2px; margin-bottom: 10px; }
.block-title h2 { margin: 0; font-size: 16px; font-weight: 600; color: var(--text-1); }
.block-title span { font-size: 12px; color: var(--text-4); }
.muted { color: var(--text-4); }
.table-wrap { background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); overflow: hidden; margin-bottom: 8px; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th, .data-table td { padding: 10px 14px; font-size: 13px; text-align: left; border-bottom: 1px solid #f1f5f4; }
.data-table th { background: #f8fbfa; color: var(--text-3); font-weight: 500; font-size: 12px; white-space: nowrap; }
.data-table tbody tr:hover { background: #f9fdfc; }
.mono { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 12px; color: var(--text-2); }
.empty-cell { text-align: center; color: #9ca3af; padding: 26px 0 !important; }

.module-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 14px; margin-bottom: 26px; }
.module-card { display: flex; align-items: center; gap: 10px; text-align: left; background: #fff; border: 1px solid var(--line); border-radius: var(--radius); padding: 14px; cursor: pointer; box-shadow: var(--shadow-sm); transition: all 0.15s ease; }
.module-card:hover { border-color: #fde68a; transform: translateY(-2px); box-shadow: 0 8px 18px rgba(217, 119, 6, 0.12); }
.mc-icon { width: 38px; height: 38px; border-radius: 10px; background: #fffbeb; color: #b45309; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.mc-icon svg { width: 19px; height: 19px; }
.mc-info { flex: 1; min-width: 0; display: flex; flex-direction: column; text-align: left; }
.mc-info b { font-size: 13px; font-weight: 600; color: var(--text-1); }
.mc-info i { font-style: normal; font-size: 11px; color: var(--text-4); }
.mc-arrow { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }

@media (max-width: 768px) {
  .cc-body { min-height: 210px; height: 210px; }
}
</style>
