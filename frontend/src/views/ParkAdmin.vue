<template>
  <Layout>
    <div class="page">
      <!-- 子功能页工具栏：返回本角色工作台；跨角色身份切换统一使用顶部身份切换胶囊 -->
      <div class="feature-toolbar">
        <button class="ft-back" @click="goHome"><ArrowLeft /> 返回工作台</button>
        <div class="ft-crumb"><span>园区管理员</span><Right /><b>{{ featureTitle }}</b></div>
      </div>

      <!-- 企业入驻管理 -->
      <div v-if="feature === 'enterprises'">
        <h2>企业入驻管理</h2>
        <div class="info-panel">管理园区内企业账号的启用/禁用状态。被禁用的企业账号无法登录系统进行操作。园区管理员可查看企业详情，包括碳排放统计和碳积分数据。</div>
        <!-- 企业管理流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">🏢</div>
            <span class="step-label">企业账号列表</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔍</div>
            <span class="step-label">查看企业详情</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">⚙️</div>
            <span class="step-label">启用/禁用管理</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📊</div>
            <span class="step-label">碳排放监控</span>
          </div>
        </div>
        <table class="table">
          <thead><tr><th>ID</th><th>用户名</th><th>企业名称</th><th>园区ID</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="e in enterprises" :key="e.id">
              <td>{{ e.id }}</td>
              <td>{{ e.username }}</td>
              <td>{{ e.company }}</td>
              <td>{{ e.park_id }}</td>
              <td><span :class="'tag tag-' + (e.status ? 'available' : 'locked')">{{ e.status ? '已启用' : '已禁用' }}</span></td>
              <td>
                <button class="btn-sm" :class="e.status ? 'btn-danger' : 'btn-success'"
                  @click="toggleStatus(e)" :disabled="loadingToggle === e.id">
                  <span v-if="loadingToggle === e.id" class="loading"><span class="spinner"></span></span>
                  <span v-else>{{ e.status ? '禁用' : '启用' }}</span>
                </button>
              </td>
            </tr>
            <tr v-if="!enterprises.length"><td colspan="6" class="empty">暂无企业</td></tr>
          </tbody>
        </table>
        <div class="form-row" style="margin-top:12px">
          <input v-model.number="detailEid" type="number" placeholder="输入企业ID查看详情" />
          <button class="btn" @click="handleDetail" :disabled="loadingDetail">
            <span v-if="loadingDetail" class="loading"><span class="spinner"></span>查询中...</span>
            <span v-else>查看详情</span>
          </button>
        </div>
        <div v-if="detailData" class="result-box">
          <p>企业: {{ detailData.enterprise?.company }} ({{ detailData.enterprise?.username }})</p>
          <p>总排放: {{ (detailData.stats?.total_emission || 0).toFixed(2) }} kgCO2</p>
          <p>总积分: {{ (detailData.stats?.total_credits || 0).toFixed(2) }}</p>
        </div>
      </div>

      <!-- 低碳报告生成 -->
      <div v-if="feature === 'report'">
        <h2>园区低碳报告生成</h2>
        <div class="info-panel">基于园区碳排放统计数据，调用DeepSeek AI大模型生成园区低碳报告和补贴申报材料。报告内容涵盖园区碳排放总量、各企业排放分布、减排建议和低碳改造方案。</div>
        <!-- 报告生成流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📊</div>
            <span class="step-label">园区碳排放数据</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🤖</div>
            <span class="step-label">DeepSeek AI分析</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📝</div>
            <span class="step-label">生成低碳报告</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">✅</div>
            <span class="step-label">补贴申报材料</span>
          </div>
        </div>
        <div class="form-card">
          <h3>报告参数</h3>
          <div class="form-row">
            <div class="input-group">
              <label class="input-label">园区名称</label>
              <input v-model="reportForm.park_name" placeholder="请输入园区名称" />
            </div>
            <button class="btn" @click="handleReport" :disabled="loadingReport">
              <span v-if="loadingReport" class="loading"><span class="spinner"></span>生成中...</span>
              <span v-else>生成报告</span>
            </button>
          </div>
        </div>
        <div v-if="reportResult" class="advice-box">
          <pre>{{ reportResult }}</pre>
        </div>
      </div>
    </div>
  </Layout>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import { adminAPI } from '../api/index.js'

const route = useRoute()
const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{}')
// 子功能标识来自真实 vue-router 路径参数(如 /park-admin/enterprises)
const feature = computed(() => route.params.feature || '')
const featureTitles = { enterprises: '企业入驻管理', report: '园区低碳报告' }
const featureTitle = computed(() => featureTitles[feature.value] || feature.value)
// 页面内导航：返回本角色工作台
function goHome() { router.push('/park-admin') }

const overview = ref({})
const enterprises = ref([])
// 各企业碳统计(用于园区可视化汇总；个别企业查询失败以空统计兜底)
const enterpriseStats = ref([])
const detailEid = ref('')
const detailData = ref(null)
const reportForm = reactive({ park_name: '绿色科技示范园区' })
const reportResult = ref('')

// ===== 园区可视化看板数据(仅展示，不改变业务接口) =====
const entShortName = (es) => es.enterprise?.company || es.enterprise?.username || ('企业#' + es.enterprise?.id)
const entEmissionItems = computed(() =>
  enterpriseStats.value.map(es => ({
    label: entShortName(es),
    value: Number((es.stats?.total_emission || 0).toFixed(1))
  }))
)
const parkPalette = ['#0d9488', '#2563eb', '#d97706', '#7c3aed', '#059669', '#06b6d4']
const entCreditItems = computed(() =>
  enterpriseStats.value.map((es, i) => ({
    label: entShortName(es),
    value: Number((es.stats?.total_credits || 0).toFixed(1)),
    color: parkPalette[i % parkPalette.length]
  }))
)

// Loading states
const loadingOverview = ref(false)
const loadingToggle = ref(null)
const loadingDetail = ref(false)
const loadingReport = ref(false)

onMounted(() => { refreshAll() })

async function refreshAll() {
  try {
    const [ov, en] = await Promise.all([
      adminAPI.parkOverview({ park_id: 1 }),
      adminAPI.parkEnterprises({ park_id: 1, page_size: 50 })
    ])
    overview.value = ov.data
    enterprises.value = en.data.list
    // 逐企业加载碳统计供园区图表使用(单项失败不阻塞)
    const rows = await Promise.all(enterprises.value.map(async e => {
      try {
        const d = await adminAPI.enterpriseDetail({ enterprise_id: e.id })
        return { enterprise: e, stats: d.data.stats || {} }
      } catch (err) {
        return { enterprise: e, stats: {} }
      }
    }))
    enterpriseStats.value = rows
  } catch (e) { console.error(e) }
}
async function refreshOverview() {
  loadingOverview.value = true
  try {
    const r = await adminAPI.parkOverview({ park_id: 1 })
    overview.value = r.data
  } catch (e) {} finally { loadingOverview.value = false }
}
async function toggleStatus(e) {
  loadingToggle.value = e.id
  try {
    await adminAPI.updateStatus({ enterprise_id: e.id, status: e.status ? 0 : 1 })
    e.status = e.status ? 0 : 1
  } catch (err) { alert(err?.msg || '操作失败') } finally { loadingToggle.value = null }
}
async function handleDetail() {
  if (!detailEid.value) return
  loadingDetail.value = true
  try {
    const r = await adminAPI.enterpriseDetail({ enterprise_id: detailEid.value })
    detailData.value = r.data
  } catch (e) {
    detailData.value = null; alert(e?.msg || '查询失败')
  } finally { loadingDetail.value = false }
}
async function handleReport() {
  loadingReport.value = true
  try {
    const r = await adminAPI.parkReport({ park_id: 1, park_name: reportForm.park_name })
    reportResult.value = r.data.report
  } catch (e) { alert(e?.msg || '生成失败') } finally { loadingReport.value = false }
}
</script>

<style scoped>
.dashboard-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
  animation: fadeIn 0.4s ease forwards;
}
.dash-card {
  background: #fff;
  border-radius: 12px;
  padding: 18px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
  transition: all 0.2s ease;
}
.dash-card:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.08); }
.dash-icon { width: 36px; height: 36px; border-radius: 10px; display: flex; align-items: center; justify-content: center; font-size: 18px; margin-bottom: 10px; }
.dash-info { margin-bottom: 8px; }
.dash-val { display: block; font-size: 24px; font-weight: 700; color: #14532d; }
.dash-label { display: block; font-size: 12px; color: #94a3b8; margin-top: 2px; }
.dash-bar { height: 6px; background: #f1f5f9; border-radius: 3px; overflow: hidden; }
.dash-fill { height: 100%; border-radius: 3px; transition: width 0.8s ease; }
.input-group { display: flex; flex-direction: column; gap: 4px; }
.input-label { font-size: 13px; font-weight: 500; color: #475569; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* 视觉收敛：数据概览改为简洁数值卡 */
.dashboard-overview {
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
  margin-bottom: 24px;
}
.dash-card {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  padding: 18px;
  box-shadow: var(--shadow-sm);
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.dash-card:hover { border-color: #99f6e4; box-shadow: var(--shadow-md); }
.dash-icon, .dash-bar, .dash-fill { display: none; }
.dash-info { margin-bottom: 0; }
.dash-val {
  display: block;
  font-size: 24px;
  font-weight: 650;
  line-height: 1.3;
  letter-spacing: -0.01em;
  color: var(--text-1);
  margin-bottom: 4px;
}
.dash-label { display: block; font-size: 12px; color: var(--text-3); }
</style>