<template>
  <!-- 园区管理员 · 独立首页数据面板(指标/图表/业务入口与其他角色完全独立，样式 scoped) -->
  <Layout>
    <div class="page">
      <!-- 角色身份标题 -->
      <div class="hero">
        <div class="hero-left">
          <span class="hero-icon"><Histogram /></span>
          <div>
            <h1>园区管理员碳统筹面板</h1>
            <p>园区管理员仅负责企业入驻与园区数据统计，不参与交易撮合，撮合由碳交易所统一完成</p>
          </div>
        </div>
        <span class="hero-tag">园区管理员</span>
      </div>

      <!-- 四个数字统计指标(始终取自真实接口，不受图表预览影响) -->
      <div class="stat-grid">
        <div class="stat-card">
          <div class="sc-icon" style="background:#e2edff"><OfficeBuilding /></div>
          <div>
            <div class="sc-value">{{ overview.enterprise_count ?? 0 }}</div>
            <div class="sc-label">园区企业数量</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#ffe8d6"><Aim /></div>
          <div>
            <div class="sc-value">{{ fmt(overview.total_emission) }}</div>
            <div class="sc-label">园区总排放(kgCO₂)</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#ddf6ee"><Coin /></div>
          <div>
            <div class="sc-value">{{ fmt(overview.total_credits) }}</div>
            <div class="sc-label">园区总积分</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="sc-icon" style="background:#eef2ff"><Connection /></div>
          <div>
            <div class="sc-value">{{ enterprises.length }}</div>
            <div class="sc-label">已入驻企业</div>
          </div>
        </div>
      </div>

      <!-- 图表模拟预览开关：仅填充图表 series，指标卡数字仍来自真实接口 -->
      <MockPreviewBar v-model="previewMock" />

      <!-- 两个可视化图表 -->
      <div class="chart-grid">
        <div class="chart-card">
          <div class="cc-head">
            <h3><DataLine class="cc-ico" /> 辖区企业能耗排放汇总</h3>
            <span class="cc-sub">各入驻企业累计碳排放量对比</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('emission', el)"></div>
        </div>

        <div class="chart-card">
          <div class="cc-head">
            <h3><PieChart class="cc-ico" /> 企业碳积分分布统计</h3>
            <span class="cc-sub">园区积分按企业占比</span>
          </div>
          <div class="cc-body" :ref="el => bindChart('credits', el)"></div>
        </div>
      </div>

      <!-- 园区业务功能入口(子路由跳转) -->
      <div class="block-title">
        <h2>园区业务模块</h2>
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
// 园区管理员首页数据面板：
//  - 顶部统计卡片数字始终来自园区真实接口；
//  - 图表 series 与统计卡数据源分开处理：打开"图表模拟预览"后图表填充内置演示数据集，
//    关闭预览且有真实数据时绘制真实图表，真实数据为空时图表区居中渲染统一"暂无数据"空态。
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import MockPreviewBar from '../components/MockPreviewBar.vue'
import { adminAPI } from '../api/index.js'
import { initChart, mkBarOption, mkPieOption, emptyOption, PALETTE } from '../utils/echarts.js'
import { MOCK } from '../utils/mockChartData.js'

const router = useRouter()
const overview = ref({})
const enterprises = ref([])
const entStats = ref([])          // [{ enterprise, stats }]
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
const entNames = computed(() => entStats.value.map(e => shortName(e.enterprise)))
const entEmission = computed(() => entStats.value.map(e => Number((e.stats?.total_emission || 0).toFixed(1))))
const entCreditData = computed(() =>
  entStats.value.map((e, i) => ({
    name: shortName(e.enterprise),
    value: Number((e.stats?.total_credits || 0).toFixed(1)),
    itemStyle: { color: PALETTE[i % PALETTE.length] }
  }))
)
const hasCreditData = computed(() => entStats.value.some(e => Number(e.stats?.total_credits) > 0))
function shortName(en) {
  const n = en?.company || en?.username || ('企业#' + en?.id)
  return n.length > 8 ? n.slice(0, 8) + '…' : n
}
function fmt(v) {
  const n = Number(v) || 0
  return Number.isInteger(n) ? n.toLocaleString() : n.toFixed(1)
}

// ========== 图表绘制(空态规则统一) ==========
function paint() {
  if (!loaded.value) return
  if (previewMock.value) {
    // 预览模式：series 使用内置模拟数据集(指标卡数字不受影响)
    setChart('emission', mkBarOption({ labels: MOCK.park.emission.labels, values: MOCK.park.emission.values, colors: PALETTE }))
    setChart('credits', mkPieOption({ data: MOCK.park.credits.data }))
    return
  }
  // 业务模式：有真实数据画真实图表，无数据时居中渲染统一空态文案(不留白)
  setChart('emission', entStats.value.length
    ? mkBarOption({ labels: entNames.value, values: entEmission.value, colors: PALETTE })
    : emptyOption())
  setChart('credits', hasCreditData.value
    ? mkPieOption({ data: entCreditData.value })
    : emptyOption())
}
watch(previewMock, () => paint())

// ========== 业务入口 ==========
const modules = [
  { path: '/park-admin/enterprises', name: '企业入驻管理', icon: 'OfficeBuilding', desc: '入驻审核与状态管理' },
  { path: '/park-admin/report', name: '园区AI报告', icon: 'MagicStick', desc: '生成园区低碳运营报告' }
]

// ========== 加载 ==========
async function refreshData() {
  loaded.value = true
  await ensureCharts()
  if (!alive) return
  paint() // 先绘制(真实数据未返回时即为空态文案)，保证图表区不留白
  try {
    const [ov, en] = await Promise.all([
      adminAPI.parkOverview({ park_id: 1 }),
      adminAPI.parkEnterprises({ park_id: 1, page_size: 50 })
    ])
    if (!alive) return
    overview.value = ov.data
    enterprises.value = en.data.list
    const rows = await Promise.all(enterprises.value.map(async ent => {
      try {
        const d = await adminAPI.enterpriseDetail({ enterprise_id: ent.id })
        return { enterprise: ent, stats: d.data.stats || {} }
      } catch (err) { return { enterprise: ent, stats: {} } }
    }))
    if (!alive) return
    entStats.value = rows
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
.hero-icon { width: 46px; height: 46px; border-radius: 12px; background: linear-gradient(135deg, #1d4ed8, #0d9488); color: #fff; display: flex; align-items: center; justify-content: center; box-shadow: 0 6px 16px rgba(59, 130, 246, 0.3); }
.hero-icon svg { width: 24px; height: 24px; }
.hero h1 { font-size: 22px; font-weight: 700; color: var(--text-1); }
.hero p { font-size: 12px; color: var(--text-4); margin-top: 2px; }
.hero-tag { font-size: 12px; font-weight: 600; color: #1d4ed8; background: #eff6ff; border: 1px solid #dbeafe; border-radius: 999px; padding: 4px 14px; }

.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 22px; }
.stat-card { display: flex; align-items: center; gap: 14px; background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 18px; box-shadow: var(--shadow-sm); }
.sc-icon { width: 44px; height: 44px; border-radius: 11px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.sc-icon svg { width: 22px; height: 22px; color: #1d4ed8; }
.sc-value { font-size: 24px; font-weight: 700; color: var(--text-1); line-height: 1.2; }
.sc-label { font-size: 12px; color: var(--text-3); margin-top: 2px; }

.chart-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 16px; margin-bottom: 26px; }
.chart-card { background: #fff; border: 1px solid var(--line); border-radius: var(--radius-lg); box-shadow: var(--shadow-sm); padding: 16px 18px; min-width: 0; }
.cc-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.cc-head h3 { display: flex; align-items: center; gap: 6px; font-size: 14px; font-weight: 600; color: var(--text-1); margin: 0; }
.cc-ico { width: 16px; height: 16px; color: var(--primary); }
.cc-sub { font-size: 11px; color: var(--text-4); }
/* 图表容器固定最小高度：防止 ECharts 因 DOM 尺寸未加载导致空白 */
.cc-body { position: relative; min-height: 250px; height: 250px; }

.block-title { display: flex; align-items: baseline; gap: 10px; margin: 6px 0 14px; }
.block-title h2 { margin: 0; font-size: 16px; font-weight: 600; color: var(--text-1); }
.block-title span { font-size: 12px; color: var(--text-4); }

.module-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 14px; margin-bottom: 26px; }
.module-card { display: flex; align-items: center; gap: 10px; text-align: left; background: #fff; border: 1px solid var(--line); border-radius: var(--radius); padding: 14px; cursor: pointer; box-shadow: var(--shadow-sm); transition: all 0.15s ease; }
.module-card:hover { border-color: #bfdbfe; transform: translateY(-2px); box-shadow: 0 8px 18px rgba(37, 99, 235, 0.12); }
.mc-icon { width: 38px; height: 38px; border-radius: 10px; background: #eff6ff; color: #1d4ed8; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.mc-icon svg { width: 19px; height: 19px; }
.mc-info { flex: 1; min-width: 0; display: flex; flex-direction: column; text-align: left; }
.mc-info b { font-size: 13px; font-weight: 600; color: var(--text-1); }
.mc-info i { font-style: normal; font-size: 11px; color: var(--text-4); }
.mc-arrow { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }

@media (max-width: 768px) {
  .cc-body { min-height: 210px; height: 210px; }
}
</style>
