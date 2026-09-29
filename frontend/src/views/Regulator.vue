<template>
  <Layout>
    <div class="page">
      <!-- 子功能页工具栏：返回本角色工作台；跨角色身份切换统一使用顶部身份切换胶囊 -->
      <div class="feature-toolbar">
        <button class="ft-back" @click="goHome"><ArrowLeft /> 返回工作台</button>
        <div class="ft-crumb"><span>监管核查</span><Right /><b>{{ featureTitle }}</b></div>
      </div>

      <!-- 链上溯源 -->
      <div v-if="feature === 'chain'">
        <h2>链上溯源 - 全部区块记录</h2>
        <div class="info-panel">提供监管部门对企业数据、交易记录和存证信息的核查能力。本地模拟联盟链，以SHA-256哈希链将所有业务数据记录在区块中，实现数据不可篡改、可追溯。</div>
        <!-- 溯源流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">🔍</div>
            <span class="step-label">查询区块记录</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">查看区块哈希链</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">✅</div>
            <span class="step-label">校验数据完整性</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📋</div>
            <span class="step-label">输出溯源结果</span>
          </div>
        </div>
        <table class="table">
          <thead><tr><th>区块高度</th><th>区块哈希</th><th>前序哈希</th><th>数据类型</th><th>数据编号</th><th>时间戳</th></tr></thead>
          <tbody>
            <tr v-for="b in blocks" :key="b.index">
              <td>{{ b.index }}</td>
              <td><code>{{ b.hash?.substring(0, 16) }}...</code></td>
              <td><code>{{ b.prev_hash?.substring(0, 16) || '-' }}...</code></td>
              <td><span class="tag">{{ b.data_type }}</span></td>
              <td>{{ b.data_id }}</td>
              <td>{{ formatTime(b.timestamp) }}</td>
            </tr>
            <tr v-if="!blocks.length"><td colspan="6" class="empty">暂无区块</td></tr>
          </tbody>
        </table>
        <p>总区块数: {{ blockTotal }}</p>
      </div>

      <!-- 平台统计 -->
      <div v-if="feature === 'stats'">
        <h2>平台碳积分发行量统计</h2>
        <div class="info-panel">监管角色可查看平台整体碳积分发行、交易和可用情况，以及企业数量和区块数量，全面掌握平台运行状态。</div>
        <!-- 数据统计流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📊</div>
            <span class="step-label">数据采集汇总</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📈</div>
            <span class="step-label">统计分析</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📋</div>
            <span class="step-label">生成统计报表</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🛡️</div>
            <span class="step-label">监管决策支持</span>
          </div>
        </div>
        <div class="stats-row">
          <div class="stat-card"><span class="num">{{ (stats.total_issued_credits || 0).toFixed(1) }}</span><span>总发行积分</span></div>
          <div class="stat-card"><span class="num">{{ (stats.total_traded_credits || 0).toFixed(1) }}</span><span>已交易积分</span></div>
          <div class="stat-card"><span class="num">{{ (stats.total_available_credits || 0).toFixed(1) }}</span><span>可用积分</span></div>
          <div class="stat-card"><span class="num">{{ stats.enterprise_count || 0 }}</span><span>企业数量</span></div>
          <div class="stat-card"><span class="num">{{ stats.transaction_count || 0 }}</span><span>交易笔数</span></div>
          <div class="stat-card"><span class="num">{{ stats.chain_block_count || 0 }}</span><span>区块数</span></div>
        </div>
        <button class="btn" @click="refreshStats" :disabled="loadingStats">
          <span v-if="loadingStats" class="loading"><span class="spinner"></span>刷新中...</span>
          <span v-else>刷新</span>
        </button>
      </div>

      <!-- 数据篡改校验 -->
      <div v-if="feature === 'verify'">
        <h2>数据篡改校验</h2>
        <div class="info-panel">通过区块链哈希校验，验证业务数据是否被篡改。重新计算当前数据的哈希值，与链上存储的原始哈希比对，判断数据是否完整可信。</div>
        <!-- 校验流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📊</div>
            <span class="step-label">选择数据类型</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔐</div>
            <span class="step-label">重新计算哈希</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">比对链上哈希</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">✅</div>
            <span class="step-label">输出校验结果</span>
          </div>
        </div>
        <div class="form-row">
          <select v-model="verifyType">
            <option value="energy">能耗数据</option>
            <option value="credit">碳积分</option>
            <option value="transaction">交易</option>
          </select>
          <input v-model="verifyId" placeholder="输入业务编号" />
          <button class="btn" @click="handleVerify" :disabled="loadingVerify">
            <span v-if="loadingVerify" class="loading"><span class="spinner"></span>校验中...</span>
            <span v-else>校验</span>
          </button>
        </div>
        <div v-if="verifyResult" class="result-box">
          <p :class="verifyResult.match ? 'success' : 'error'">
            {{ verifyResult.match ? '数据完整，未被篡改' : '数据已被篡改或未上链' }}
          </p>
          <p>区块索引: {{ verifyResult.block_index }}</p>
          <p>区块哈希: <code>{{ verifyResult.block_hash }}</code></p>
          <p>原始哈希: <code>{{ verifyResult.original_hash }}</code></p>
          <p>当前哈希: <code>{{ verifyResult.current_hash }}</code></p>
        </div>
      </div>

      <!-- 风险告警 -->
      <div v-if="feature === 'alert'">
        <h2>风险告警</h2>
        <div class="info-panel">系统自动检测平台运行风险，包括数据异常、篡改可能等，帮助监管角色及时发现并处理问题。告警级别分为正常、信息、警告和危险，便于快速定位问题严重程度。</div>
        <!-- 告警流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">🔍</div>
            <span class="step-label">自动检测</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">⚠️</div>
            <span class="step-label">风险识别</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📋</div>
            <span class="step-label">告警分级</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🛡️</div>
            <span class="step-label">监管处置</span>
          </div>
        </div>
        <div v-for="a in alerts" :key="a.type" :class="'alert-card alert-' + a.level">
          <div class="alert-header">
            <span class="alert-level">{{ a.level === 'info' ? '信息' : a.level === 'warning' ? '警告' : a.level === 'success' ? '正常' : '危险' }}</span>
            <span>{{ a.type }}</span>
          </div>
          <p>{{ a.message }}</p>
        </div>
        <p v-if="!alerts.length" class="empty">暂无告警</p>
        <button class="btn" @click="refreshAlerts" :disabled="loadingAlerts">
          <span v-if="loadingAlerts" class="loading"><span class="spinner"></span>刷新中...</span>
          <span v-else>刷新</span>
        </button>
      </div>

      <!-- 用户管理 -->
      <div v-if="feature === 'users'">
        <h2>用户管理</h2>
        <div class="info-panel">查看系统所有用户信息，包括账号状态、角色类型和注册时间。监管角色可全面掌握平台用户分布情况，确保系统运行安全合规。</div>
        <!-- 用户管理流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">👥</div>
            <span class="step-label">用户列表</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔍</div>
            <span class="step-label">查看用户信息</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">⚙️</div>
            <span class="step-label">账号状态管理</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🛡️</div>
            <span class="step-label">安全合规监控</span>
          </div>
        </div>
        <table class="table">
          <thead><tr><th>ID</th><th>用户名</th><th>企业名称</th><th>角色</th><th>状态</th><th>创建时间</th></tr></thead>
          <tbody>
            <tr v-for="u in users" :key="u.id">
              <td>{{ u.id }}</td>
              <td>{{ u.username }}</td>
              <td>{{ u.company }}</td>
              <td><span class="tag">{{ roleLabels[u.role] || u.role }}</span></td>
              <td><span :class="'tag tag-' + (u.status ? 'available' : 'locked')">{{ u.status ? '正常' : '禁用' }}</span></td>
              <td>{{ formatTime(u.created_at) }}</td>
            </tr>
            <tr v-if="!users.length"><td colspan="6" class="empty">暂无用户</td></tr>
          </tbody>
        </table>
        <p>用户总数: {{ userTotal }}</p>
      </div>
    </div>
  </Layout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import { regulatorAPI, chainAPI } from '../api/index.js'

const route = useRoute()
const router = useRouter()
// 子功能标识来自真实 vue-router 路径参数(如 /regulator/chain)
const feature = computed(() => route.params.feature || '')
const featureTitles = { chain: '链上溯源', stats: '平台统计', verify: '数据篡改校验', alert: '风险告警', users: '用户管理' }
const featureTitle = computed(() => featureTitles[feature.value] || feature.value)
// 页面内导航：返回本角色工作台
function goHome() { router.push('/regulator') }

// 用户管理列表中的角色展示名(正式名称，仅界面展示)
const roleLabels = { enterprise: '小微企业', park_admin: '园区管理员', exchange: '碳交易所', regulator: '监管核查' }

// 链上溯源
const blocks = ref([]); const blockTotal = ref(0)

// 平台统计
const stats = ref({})

// 篡改校验
const verifyType = ref('credit'); const verifyId = ref(''); const verifyResult = ref(null)

// 风险告警
const alerts = ref([])

// 用户管理
const users = ref([]); const userTotal = ref(0)

// Loading states
const loadingStats = ref(false)
const loadingVerify = ref(false)
const loadingAlerts = ref(false)

onMounted(() => { refreshAll() })

// ===== 监管可视化看板数据(仅展示，由已加载区块/统计/告警数据计算生成) =====
function toTime(t) {
  if (t == null) return 0
  if (typeof t === 'number') return t > 1e12 ? t : t * 1000 // 秒级时间戳 → 毫秒
  return new Date(t).getTime()
}
function dayKey(t) {
  const d = new Date(toTime(t))
  if (isNaN(d.getTime())) return '-'
  return (d.getMonth() + 1) + '/' + d.getDate()
}
// 上链存证趋势：按日期统计新增区块数
const chainTrendItems = computed(() => {
  const sorted = [...blocks.value].sort((a, b) => toTime(a.timestamp) - toTime(b.timestamp))
  const map = new Map()
  sorted.forEach(b => {
    const k = dayKey(b.timestamp)
    map.set(k, (map.get(k) || 0) + 1)
  })
  return [...map.entries()].map(([label, value]) => ({ label, value }))
})
// 上链存证数量统计：按业务类型
const dataTypeLabels = { energy: '能耗数据', credit: '碳积分', transaction: '交易', report: 'AI报告', genesis: '创世块', operation: '操作日志' }
const typeOrder = ['energy', 'credit', 'transaction', 'report', 'genesis', 'operation']
const blockTypeItems = computed(() => {
  const map = new Map()
  blocks.value.forEach(b => {
    const key = b.data_type || 'unknown'
    map.set(key, (map.get(key) || 0) + 1)
  })
  const names = [...typeOrder, ...[...map.keys()].filter(k => !typeOrder.includes(k))]
  return names
    .filter(k => map.has(k))
    .map(k => ({ label: dataTypeLabels[k] || k, value: map.get(k) }))
})
// 平台碳积分流通分布(可用/锁定/已交易 三态之和 = 已发行)
const creditFlowItems = computed(() => [
  { label: '可用', value: Number((stats.value.total_available_credits || 0).toFixed(1)), color: '#0d9488' },
  { label: '锁定(挂单中)', value: Number((stats.value.total_locked_credits || 0).toFixed(1)), color: '#0d9488' },
  { label: '已交易', value: Number((stats.value.total_traded_credits || 0).toFixed(1)), color: '#f59e0b' }
])
// 数据校验/风险告警结果汇总：按级别统计
const levelLabels = { success: '正常', info: '提示', warning: '警告', error: '异常' }
const levelColors = { success: '#10b981', info: '#0d9488', warning: '#f59e0b', error: '#ef4444' }
const levelOrder = ['success', 'info', 'warning', 'error']
const alertLevelItems = computed(() => {
  const map = new Map()
  alerts.value.forEach(a => map.set(a.level || 'info', (map.get(a.level || 'info') || 0) + 1))
  return levelOrder
    .filter(k => map.has(k))
    .map(k => ({ label: levelLabels[k] || k, value: map.get(k), color: levelColors[k] }))
})

async function refreshAll() {
  try {
    const [b, s, a, u] = await Promise.all([
      chainAPI.blocks({ page_size: 100 }),
      regulatorAPI.platformStats(),
      regulatorAPI.riskAlert(),
      regulatorAPI.users({ page_size: 100 })
    ])
    blocks.value = b.data.list; blockTotal.value = b.data.total
    stats.value = s.data
    alerts.value = a.data.alerts
    users.value = u.data.list; userTotal.value = u.data.total
  } catch (e) { console.error(e) }
}
async function refreshStats() {
  loadingStats.value = true
  try {
    const r = await regulatorAPI.platformStats()
    stats.value = r.data
  } catch (e) {} finally { loadingStats.value = false }
}
async function refreshAlerts() {
  loadingAlerts.value = true
  try {
    const r = await regulatorAPI.riskAlert()
    alerts.value = r.data.alerts
  } catch (e) {} finally { loadingAlerts.value = false }
}
async function handleVerify() {
  if (!verifyId.value) return
  loadingVerify.value = true
  try {
    const r = await regulatorAPI.verify({ data_type: verifyType.value, data_id: verifyId.value })
    verifyResult.value = r.data
  } catch (e) {
    verifyResult.value = null; alert(e?.msg || '校验失败')
  } finally { loadingVerify.value = false }
}
function formatTime(t) {
  if (!t) return '-'
  if (typeof t === 'number') return new Date(t * 1000).toLocaleString()
  return new Date(t).toLocaleString()
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
.dash-card { background: #fff; border-radius: 12px; padding: 18px; box-shadow: 0 1px 3px rgba(0,0,0,0.06); transition: all 0.2s ease; }
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