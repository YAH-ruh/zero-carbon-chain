<template>
  <Layout>
    <div class="page">
      <!-- 子功能页工具栏：返回本角色工作台；跨角色身份切换统一使用顶部身份切换胶囊 -->
      <div class="feature-toolbar">
        <button class="ft-back" @click="goHome"><ArrowLeft /> 返回工作台</button>
        <div class="ft-crumb"><span>碳交易所</span><Right /><b>{{ featureTitle }}</b></div>
      </div>

      <!-- 卖方挂单浏览 -->
      <div v-if="feature === 'orders'">
        <h2>卖方挂单浏览</h2>
        <div class="info-panel">浏览园区内所有卖方挂单货源。交易机制：园区内所有入驻企业均可作为买方(园区内部交易)，也支持跨园区入驻企业购买(跨园区交易)；买方由碳交易所统一撮合匹配，园区管理员仅负责企业入驻与园区统计，不参与撮合。</div>
        <!-- 挂单交易流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📤</div>
            <span class="step-label">企业挂单</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📋</div>
            <span class="step-label">交易所浏览</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🤝</div>
            <span class="step-label">撮合匹配</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">上链存证</span>
          </div>
        </div>
        <table class="table">
          <thead><tr><th>挂单编号</th><th>卖方</th><th>数量</th><th>单价(元)</th><th>总金额</th><th>状态</th></tr></thead>
          <tbody>
            <tr v-for="o in orders" :key="o.id">
              <td>{{ o.order_no }}</td>
              <td>{{ o.enterprise?.company || o.enterprise_id }}<span v-if="o.enterprise?.park_id" class="park-tag">园区{{ o.enterprise.park_id }}</span></td>
              <td>{{ o.quantity }}</td>
              <td>{{ o.unit_price }}</td>
              <td>{{ o.total_amount }}</td>
              <td><span class="tag tag-pending">挂单中</span></td>
            </tr>
            <tr v-if="!orders.length"><td colspan="6" class="empty">暂无挂单</td></tr>
          </tbody>
        </table>
        <button class="btn" @click="refreshOrders" :disabled="loadingOrders">
          <span v-if="loadingOrders" class="loading"><span class="spinner"></span>刷新中...</span>
          <span v-else>刷新</span>
        </button>
      </div>

      <!-- 交易撮合 -->
      <div v-if="feature === 'match'">
        <h2>交易撮合</h2>
        <div class="info-panel">撮合机制：园区内所有入驻企业均可作买方(园区内部交易)，也支持跨园区入驻企业作买方(跨园区交易)；系统自动完成挂单成交、积分权属转移与交易上链存证。园区管理员不参与撮合。</div>
        <!-- 撮合流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📋</div>
            <span class="step-label">选择待成交挂单</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">👤</div>
            <span class="step-label">指定买方</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🤝</div>
            <span class="step-label">撮合成交</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">上链存证</span>
          </div>
        </div>
        <div class="form-card">
          <h3>撮合参数</h3>
          <div class="form-row">
            <div class="input-group">
              <label class="input-label">选择待成交挂单</label>
              <select v-model="matchOrderId">
                <option value="0">请选择挂单</option>
                <option v-for="o in orders" :key="o.id" :value="o.id">{{ o.order_no }} ({{ o.quantity }}积分/{{ o.total_amount }}元)</option>
              </select>
            </div>
            <div class="input-group">
              <label class="input-label">选择买方(园区入驻企业，支持跨园区)</label>
              <select v-model="matchBuyerId">
                <option :value="0">请选择买方企业</option>
                <option v-for="b in buyers" :key="b.id" :value="b.id">{{ b.company || b.username }}（园区{{ b.park_id || '—' }}）</option>
              </select>
              <span class="buyer-hint">候选 = 全部园区已入驻且启用的小微企业，跨园区企业同样可成交</span>
            </div>
            <button class="btn" @click="handleMatch" :disabled="loadingMatch">
              <span v-if="loadingMatch" class="loading"><span class="spinner"></span>撮合中...</span>
              <span v-else>撮合成交</span>
            </button>
          </div>
          <p v-if="matchMsg" :class="matchMsgType">{{ matchMsg }}</p>
        </div>

        <div v-if="matchResult" class="result-box">
          <h3>成交凭证</h3>
          <p>交易编号: {{ matchResult.tx_no }}</p>
          <p>卖方: {{ matchResult.seller?.company }}（园区{{ matchResult.seller?.park_id }}）(ID: {{ matchResult.seller_id }})</p>
          <p>买方: {{ matchResult.buyer?.company }}（园区{{ matchResult.buyer?.park_id }}）(ID: {{ matchResult.buyer_id }})</p>
          <p>撮合范围: {{ matchResult.seller?.park_id === matchResult.buyer?.park_id ? '园区内部交易' : '跨园区交易' }}</p>
          <p>数量: {{ matchResult.quantity }} 积分</p>
          <p>单价: {{ matchResult.unit_price }} 元</p>
          <p>总金额: {{ matchResult.total_amount }} 元</p>
          <p>区块哈希: <code>{{ matchResult.block_hash }}</code></p>
        </div>
      </div>

      <!-- 碳积分核验 -->
      <div v-if="feature === 'verify'">
        <h2>碳积分来源核验</h2>
        <div class="info-panel">核验碳积分来源的真实性，通过区块链哈希校验确认数据是否被篡改，保证碳积分来源可信。输入碳积分编号，系统将从区块链中查询原始哈希并比对验证。</div>
        <!-- 核验流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">🔍</div>
            <span class="step-label">输入积分编号</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">查询区块链记录</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">✅</div>
            <span class="step-label">哈希比对验证</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📋</div>
            <span class="step-label">输出核验结果</span>
          </div>
        </div>
        <div class="form-row">
          <input v-model="verifyCreditNo" placeholder="输入碳积分编号" />
          <button class="btn" @click="handleVerify" :disabled="loadingVerify">
            <span v-if="loadingVerify" class="loading"><span class="spinner"></span>核验中...</span>
            <span v-else>核验</span>
          </button>
        </div>
        <div v-if="verifyResult" class="result-box">
          <p>区块索引: {{ verifyResult.block_index }}</p>
          <p>区块哈希: <code>{{ verifyResult.block_hash }}</code></p>
          <p>原始哈希: <code>{{ verifyResult.original_hash }}</code></p>
          <p :class="verifyResult.match ? 'success' : 'error'">
            {{ verifyResult.match ? '来源可信，数据完整' : '数据已被篡改，来源不可信' }}
          </p>
        </div>
      </div>

      <!-- 交易记录 -->
      <div v-if="feature === 'history'">
        <h2>交易记录</h2>
        <div class="info-panel">查看平台上所有已完成的碳积分交易记录，所有交易都已上链存证，可追溯核验。每笔交易包含卖方、买方、数量、金额和区块链哈希信息。</div>
        <!-- 交易记录流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">🤝</div>
            <span class="step-label">交易撮合完成</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">自动上链存证</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📜</div>
            <span class="step-label">生成交易记录</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔍</div>
            <span class="step-label">可追溯核验</span>
          </div>
        </div>
        <table class="table">
          <thead><tr><th>交易编号</th><th>卖方</th><th>买方</th><th>数量</th><th>总金额</th><th>上链</th><th>时间</th></tr></thead>
          <tbody>
            <tr v-for="t in transactions" :key="t.id">
              <td>{{ t.tx_no }}</td>
              <td>{{ t.seller?.company || ('卖方#' + t.seller_id) }}<span v-if="t.seller?.park_id" class="park-tag">园区{{ t.seller.park_id }}</span></td>
              <td>{{ t.buyer?.company || ('买方#' + t.buyer_id) }}<span v-if="t.buyer?.park_id" class="park-tag">园区{{ t.buyer.park_id }}</span></td>
              <td>{{ t.quantity }}</td>
              <td>{{ t.total_amount }}</td>
              <td>{{ t.on_chain ? '已上链' : '未上链' }}</td>
              <td>{{ formatTime(t.created_at) }}</td>
            </tr>
            <tr v-if="!transactions.length"><td colspan="7" class="empty">暂无交易记录</td></tr>
          </tbody>
        </table>
        <button class="btn" @click="refreshTx" :disabled="loadingTx">
          <span v-if="loadingTx" class="loading"><span class="spinner"></span>刷新中...</span>
          <span v-else>刷新</span>
        </button>
      </div>
    </div>
  </Layout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import { exchangeAPI } from '../api/index.js'

const route = useRoute()
const router = useRouter()
// 子功能标识来自真实 vue-router 路径参数(如 /exchange/match)
const feature = computed(() => route.params.feature || '')
const featureTitles = { match: '交易撮合', verify: '积分核验', history: '成交记录', orders: '卖方挂单浏览' }
const featureTitle = computed(() => featureTitles[feature.value] || feature.value)
// 页面内导航：返回本角色工作台
function goHome() { router.push('/exchange') }

const orders = ref([])
const buyers = ref([]) // 可作买方的园区入驻企业候选(含跨园区企业)
const matchOrderId = ref(0); const matchBuyerId = ref(0)
const matchMsg = ref(''); const matchMsgType = ref('')
const matchResult = ref(null)
const verifyCreditNo = ref(''); const verifyResult = ref(null)
const transactions = ref([])

// Dashboard computed
const totalTradedQty = computed(() => {
  return transactions.value.reduce((s, t) => s + (Number(t.quantity) || 0), 0).toFixed(1)
})
const totalTxAmount = computed(() => {
  return transactions.value.reduce((s, t) => s + (t.total_amount || 0), 0).toFixed(1)
})

// ===== 交易可视化看板数据(仅展示，由已加载业务记录计算生成) =====
function shortDate(t) {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return '-'
  return (d.getMonth() + 1) + '/' + d.getDate()
}
// 成交量趋势：按成交日期汇总
const txTrendItems = computed(() => {
  const sorted = [...transactions.value].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  const map = new Map()
  sorted.forEach(t => {
    const k = shortDate(t.created_at)
    map.set(k, (map.get(k) || 0) + (Number(t.quantity) || 0))
  })
  return [...map.entries()].map(([label, value]) => ({ label, value: Number(value.toFixed(1)) }))
})
// 挂单概览：按挂单量降序展示待成交挂单
const orderRankItems = computed(() =>
  [...orders.value]
    .sort((a, b) => (Number(b.quantity) || 0) - (Number(a.quantity) || 0))
    .slice(0, 12)
    .map(o => ({ label: String(o.order_no || o.id), value: Number(o.quantity) || 0 }))
)
// 按卖方成交分布
const txPalette = ['#0d9488', '#2563eb', '#d97706', '#7c3aed', '#059669', '#06b6d4']
const sellerSplitItems = computed(() => {
  const map = new Map()
  transactions.value.forEach(t => {
    const name = t.seller?.company || ('卖方#' + t.seller_id)
    map.set(name, (map.get(name) || 0) + (Number(t.quantity) || 0))
  })
  return [...map.entries()]
    .map(([label, value], i) => ({
      label,
      value: Number(value.toFixed(1)),
      color: txPalette[i % txPalette.length]
    }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 6)
})

// Loading states
const loadingOrders = ref(false)
const loadingMatch = ref(false)
const loadingVerify = ref(false)
const loadingTx = ref(false)

onMounted(() => { refreshAll() })

async function refreshAll() {
  try {
    const [o, t, b] = await Promise.all([
      exchangeAPI.orders({ page_size: 50 }),
      exchangeAPI.transactions({ page_size: 50 }),
      exchangeAPI.buyers()
    ])
    orders.value = o.data.list
    transactions.value = t.data.list
    buyers.value = b.data.list
  } catch (e) { console.error(e) }
}
async function refreshOrders() {
  loadingOrders.value = true
  try {
    const r = await exchangeAPI.orders({ page_size: 50 })
    orders.value = r.data.list
  } catch (e) {} finally { loadingOrders.value = false }
}
async function refreshTx() {
  loadingTx.value = true
  try {
    const r = await exchangeAPI.transactions({ page_size: 50 })
    transactions.value = r.data.list
  } catch (e) {} finally { loadingTx.value = false }
}
async function handleMatch() {
  if (!matchOrderId.value || matchOrderId.value === 0) {
    matchMsg.value = '请选择挂单'; matchMsgType.value = 'error'; return
  }
  if (!matchBuyerId.value) {
    matchMsg.value = '请选择买方企业(园区入驻企业，支持跨园区)'; matchMsgType.value = 'error'; return
  }
  loadingMatch.value = true
  try {
    const r = await exchangeAPI.match({ order_id: matchOrderId.value, buyer_id: matchBuyerId.value })
    matchResult.value = r.data.transaction
    matchMsg.value = '撮合成功'; matchMsgType.value = 'success'
    refreshAll()
  } catch (e) {
    matchMsg.value = e?.msg || '撮合失败'; matchMsgType.value = 'error'
  } finally { loadingMatch.value = false }
}
async function handleVerify() {
  if (!verifyCreditNo.value) return
  loadingVerify.value = true
  try {
    const r = await exchangeAPI.verifyCredit({ credit_no: verifyCreditNo.value })
    verifyResult.value = r.data
  } catch (e) {
    verifyResult.value = null; alert(e?.msg || '核验失败')
  } finally { loadingVerify.value = false }
}
function formatTime(t) { return t ? new Date(t).toLocaleString() : '-' }
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
.dash-val { display: block; font-size: 24px; font-weight: 700; color: #1e293b; }
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
.park-tag {
  display: inline-block;
  margin-left: 6px;
  font-size: 11px;
  color: #0f766e;
  background: #f0fdfa;
  border: 1px solid #ccfbf1;
  border-radius: 4px;
  padding: 0 6px;
  vertical-align: middle;
}
.buyer-hint { font-size: 11px; color: var(--text-4); line-height: 1.6; }
</style>