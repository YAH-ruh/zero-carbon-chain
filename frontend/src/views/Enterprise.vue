<template>
  <Layout>
    <div class="page">
      <!-- 子功能页工具栏：返回本角色工作台；跨角色身份切换统一使用顶部身份切换胶囊 -->
      <div class="feature-toolbar">
        <button class="ft-back" @click="goHome"><ArrowLeft /> 返回工作台</button>
        <div class="ft-crumb"><span>小微企业</span><Right /><b>{{ featureTitle }}</b></div>
      </div>

      <!-- 能耗数据管理（手动录入） -->
      <div v-if="feature === 'energy'">
        <h2>能耗数据管理</h2>
        <div class="info-panel">用于上报企业能耗数据，系统将自动核算碳排放量并完成存证。能耗数据需手动录入提交，禁止自动模拟生成。</div>
        <!-- 业务流程示意图 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📝</div>
            <span class="step-label">手动录入能耗</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🧮</div>
            <span class="step-label">自动核算碳积分</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">SHA-256上链存证</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🌿</div>
            <span class="step-label">碳积分交易流转</span>
          </div>
        </div>
        <div class="form-card">
          <h3>手动录入能耗数据</h3>
          <div class="form-row">
            <div class="input-group">
              <label class="input-label">用电量（kWh）</label>
              <input v-model="manualForm.electricity" type="number" step="0.1" placeholder="请输入用电量" required />
            </div>
            <div class="input-group">
              <label class="input-label">天然气用量（m³）</label>
              <input v-model="manualForm.gas" type="number" step="0.1" placeholder="请输入天然气用量" required />
            </div>
            <div class="input-group">
              <label class="input-label">用水量（吨）</label>
              <input v-model="manualForm.water" type="number" step="0.1" placeholder="请输入用水量" required />
            </div>
            <button class="btn btn-primary" @click="handleManualSubmit" :disabled="loadingManual">
              <span v-if="loadingManual" class="loading"><span class="spinner"></span>提交中...</span>
              <span v-else>提交能耗数据</span>
            </button>
          </div>
          <p v-if="manualMsg" :class="manualMsgType">{{ manualMsg }}</p>
        </div>
        <div class="form-row" style="margin-top:12px" v-if="energyRecords.length > 0">
          <button class="btn" @click="handleCalculate(energyRecords[0]?.id)" :disabled="loadingCalc">
            <span v-if="loadingCalc" class="loading"><span class="spinner"></span>核算中...</span>
            <span v-else>核算最新能耗 → 碳积分并上链</span>
          </button>
        </div>
        <h3>能耗记录</h3>
        <table class="table">
          <thead><tr><th>记录编号</th><th>用电(kWh)</th><th>天然气(m³)</th><th>用水(t)</th><th>上链</th><th>采集时间</th></tr></thead>
          <tbody>
            <tr v-for="r in energyRecords" :key="r.id">
              <td>{{ r.record_no }}</td>
              <td>{{ r.electricity.toFixed(1) }}</td>
              <td>{{ r.gas.toFixed(1) }}</td>
              <td>{{ r.water.toFixed(1) }}</td>
              <td>{{ r.on_chain ? '已上链' : '未上链' }}</td>
              <td>{{ formatTime(r.collect_time) }}</td>
            </tr>
            <tr v-if="!energyRecords.length"><td colspan="6" class="empty">暂无数据，请先录入能耗数据</td></tr>
          </tbody>
        </table>
      </div>

      <!-- Tab: 碳积分管理 -->
      <div v-if="feature === 'credits'">
        <h2>碳积分管理</h2>
        <div class="info-panel">展示企业当前碳积分余额、来源记录和使用记录。基于能耗数据核算碳排放量，自动生成对应碳积分，积分可用于挂单交易。核算结果自动上链存证，保证数据可信不可篡改。</div>
        <!-- 碳积分流转示意 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon"></div>
            <span class="step-label">能耗数据</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🧮</div>
            <span class="step-label">碳排放核算</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🌿</div>
            <span class="step-label">生成碳积分</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">上链存证</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📤</div>
            <span class="step-label">挂单交易</span>
          </div>
        </div>
        <div class="stats-row">
          <div class="stat-card"><span class="num">{{ stats.total_credits || 0 }}</span><span>总积分</span></div>
          <div class="stat-card"><span class="num">{{ stats.available_credits || 0 }}</span><span>可用积分</span></div>
          <div class="stat-card"><span class="num">{{ stats.sold_credits || 0 }}</span><span>已售出</span></div>
          <div class="stat-card"><span class="num">{{ (stats.total_emission || 0).toFixed(1) }}</span><span>总排放(kg)</span></div>
        </div>
        <button class="btn" @click="refreshStats" :disabled="loadingStats">
          <span v-if="loadingStats" class="loading"><span class="spinner"></span>刷新中...</span>
          <span v-else>刷新数据</span>
        </button>
        <h3>碳积分列表</h3>
        <table class="table">
          <thead><tr><th>编号</th><th>碳排放量</th><th>碳积分</th><th>状态</th><th>上链</th><th>时间</th></tr></thead>
          <tbody>
            <tr v-for="c in credits" :key="c.id">
              <td>{{ c.credit_no }}</td>
              <td>{{ c.total_emission.toFixed(2) }}</td>
              <td>{{ c.carbon_credits.toFixed(2) }}</td>
              <td><span :class="'tag tag-' + c.status">{{ statusMap[c.status] }}</span></td>
              <td>{{ c.on_chain ? '已上链' : '未上链' }}</td>
              <td>{{ formatTime(c.created_at) }}</td>
            </tr>
            <tr v-if="!credits.length"><td colspan="6" class="empty">暂无数据，请先核算碳积分</td></tr>
          </tbody>
        </table>
      </div>

      <!-- Tab: 挂单交易 -->
      <div v-if="feature === 'sell'">
        <h2>碳积分挂单交易</h2>
        <div class="info-panel">展示碳积分挂单信息，支持企业间积分流转交易。企业可将可用碳积分挂单出售，交易所完成撮合交易后交易记录上链存证，实现碳积分权属可信流转。</div>
        <!-- 交易流程示意 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">🌿</div>
            <span class="step-label">选择可用积分</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📤</div>
            <span class="step-label">创建挂单</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🤝</div>
            <span class="step-label">交易所撮合</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">上链存证</span>
          </div>
        </div>
        <div class="form-card">
          <h3>创建挂单</h3>
          <div class="form-row">
            <div class="input-group">
              <label class="input-label">选择可用积分</label>
              <select v-model="sellForm.credit_id">
                <option value="0">请选择可用积分</option>
                <option v-for="c in availableCredits" :key="c.id" :value="c.id">
                  {{ c.credit_no }} ({{ c.carbon_credits.toFixed(2) }} 积分)
                </option>
              </select>
            </div>
            <div class="input-group">
              <label class="input-label">卖出数量</label>
              <input v-model.number="sellForm.quantity" type="number" placeholder="请输入卖出数量" />
            </div>
            <div class="input-group">
              <label class="input-label">单价（元）</label>
              <input v-model.number="sellForm.unit_price" type="number" step="0.1" placeholder="请输入单价" />
            </div>
            <button class="btn" @click="handleSell" :disabled="loadingSell">
              <span v-if="loadingSell" class="loading"><span class="spinner"></span>创建中...</span>
              <span v-else>创建挂单</span>
            </button>
          </div>
          <p v-if="sellMsg" :class="sellMsgType">{{ sellMsg }}</p>
        </div>
        <h3>我的挂单</h3>
        <table class="table">
          <thead><tr><th>挂单编号</th><th>数量</th><th>单价</th><th>总金额</th><th>状态</th><th>上链</th></tr></thead>
          <tbody>
            <tr v-for="o in sellOrders" :key="o.id">
              <td>{{ o.order_no }}</td>
              <td>{{ o.quantity }}</td>
              <td>{{ o.unit_price }}</td>
              <td>{{ o.total_amount }}</td>
              <td><span :class="'tag tag-' + o.status">{{ statusLabel[o.status] }}</span></td>
              <td>{{ o.on_chain ? '已上链' : '未上链' }}</td>
            </tr>
            <tr v-if="!sellOrders.length"><td colspan="6" class="empty">暂无挂单</td></tr>
          </tbody>
        </table>
      </div>

      <!-- Tab: 区块链溯源 -->
      <div v-if="feature === 'chain'">
        <h2>区块链溯源查询</h2>
        <div class="info-panel">对关键业务数据生成哈希存证，模拟联盟链溯源校验。本地模拟联盟链，对所有业务记录计算SHA-256哈希存入区块，通过哈希链保证数据不可篡改。可随时查询业务记录的哈希校验结果。</div>
        <!-- 链上存证流程 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📊</div>
            <span class="step-label">业务数据</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon"></div>
            <span class="step-label">SHA-256哈希</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🔗</div>
            <span class="step-label">生成区块</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon"></div>
            <span class="step-label">链式校验</span>
          </div>
        </div>
        <div class="form-row">
          <input v-model="queryNo" placeholder="输入业务单号(如 CREDIT-xxx)" />
          <button class="btn" @click="handleQuery" :disabled="loadingQuery">
            <span v-if="loadingQuery" class="loading"><span class="spinner"></span>查询中...</span>
            <span v-else>查询</span>
          </button>
        </div>
        <div v-if="chainResult" class="result-box">
          <p>区块索引: {{ chainResult.block_index }}</p>
          <p>区块哈希: <code>{{ chainResult.block_hash }}</code></p>
          <p>数据类型: {{ chainResult.data_type }}</p>
          <p>数据匹配: {{ chainResult.match ? '匹配' : '不匹配' }}</p>
        </div>
        <div class="form-row" style="margin-top:16px">
          <select v-model="uploadType"><option value="energy">能耗数据</option><option value="credit">碳积分</option><option value="transaction">交易</option></select>
          <input v-model="uploadId" placeholder="业务编号" />
          <button class="btn" @click="handleUpload" :disabled="loadingUpload">
            <span v-if="loadingUpload" class="loading"><span class="spinner"></span>上链中...</span>
            <span v-else>上链存证</span>
          </button>
        </div>
        <p v-if="chainMsg" :class="chainMsgType">{{ chainMsg }}</p>
        <h3>区块链信息</h3>
        <div v-if="chainInfo" class="chain-info">
          <p>区块数量: {{ chainInfo.block_count }}</p>
          <p>最新区块高度: {{ chainInfo.latest_block_index }}</p>
          <p>链类型: {{ chainInfo.chain_type }}</p>
        </div>
      </div>

      <!-- Tab: AI减排建议 -->
      <div v-if="feature === 'advice'">
        <h2>AI 智能减排建议</h2>
        <div class="info-panel">基于企业能耗数据，调用DeepSeek AI大模型生成针对性节能减排建议，辅助企业制定低碳改造方案。输入企业名称和行业信息，AI将结合企业历史能耗数据给出定制化建议。</div>
        <!-- AI流程示意 -->
        <div class="process-flow">
          <div class="process-step">
            <div class="step-icon">📊</div>
            <span class="step-label">历史能耗数据</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">🤖</div>
            <span class="step-label">DeepSeek AI分析</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon">📝</div>
            <span class="step-label">生成减排建议</span>
          </div>
          <span class="process-arrow">⟶</span>
          <div class="process-step">
            <div class="step-icon"></div>
            <span class="step-label">执行优化方案</span>
          </div>
        </div>
        <div class="form-card">
          <h3>生成建议参数</h3>
          <div class="form-row">
            <div class="input-group">
              <label class="input-label">企业名称</label>
              <input v-model="adviceForm.company_name" placeholder="请输入企业名称" />
            </div>
            <div class="input-group">
              <label class="input-label">所属行业</label>
              <input v-model="adviceForm.industry" placeholder="请输入所属行业" />
            </div>
            <button class="btn" @click="handleAdvice" :disabled="loadingAdvice">
              <span v-if="loadingAdvice" class="loading"><span class="spinner"></span>生成中...</span>
              <span v-else>生成建议</span>
            </button>
          </div>
        </div>
        <div v-if="adviceResult" class="advice-box">
          <pre>{{ adviceResult }}</pre>
        </div>
      </div>
    </div>
  </Layout>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Layout from '../components/Layout.vue'
import { energyAPI, carbonAPI, chainAPI, adviceAPI } from '../api/index.js'

const route = useRoute()
const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{}')
// 子功能标识来自真实 vue-router 路径参数(如 /enterprise/energy)
const feature = computed(() => route.params.feature || '')
const featureTitles = { energy: '能耗数据管理', credits: '碳积分管理', sell: '挂单交易', chain: '区块链溯源', advice: 'AI减排建议' }
const featureTitle = computed(() => featureTitles[feature.value] || feature.value)
// 页面内导航：返回本角色工作台
function goHome() { router.push('/enterprise') }

const statusMap = { available: '可用', locked: '已锁定', sold: '已售出' }
const statusLabel = { pending: '挂单中', matched: '已成交', cancelled: '已取消' }

// Dashboard computed
const totalElectricity = computed(() => {
  return energyRecords.value.reduce((s, r) => s + r.electricity, 0)
})
const onChainCount = computed(() => {
  return energyRecords.value.filter(r => r.on_chain).length +
    credits.value.filter(c => c.on_chain).length
})
const totalRecords = computed(() => {
  return energyRecords.value.length + credits.value.length
})
const creditPercent = computed(() => {
  const max = 10000
  return Math.min(((stats.value.total_credits || 0) / max) * 100, 100)
})
const chainPercent = computed(() => {
  const max = 50
  return Math.min((onChainCount.value / max) * 100, 100)
})

// ===== 可视化看板数据(仅做展示，由已加载业务记录计算生成) =====
// 碳积分趋势：按生成时间升序累计
const creditTrendItems = computed(() => {
  const sorted = [...credits.value].sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  let acc = 0
  return sorted.map(c => {
    acc += Number(c.carbon_credits) || 0
    return { label: shortDate(c.created_at), value: Number(acc.toFixed(1)) }
  })
})
// 能耗构成：用电/天然气/用水 累计总量
const energyUsageItems = computed(() => {
  const er = energyRecords.value
  return [
    { label: '用电(kWh)', value: Number(er.reduce((s, r) => s + (r.electricity || 0), 0).toFixed(1)) },
    { label: '天然气(m³)', value: Number(er.reduce((s, r) => s + (r.gas || 0), 0).toFixed(1)) },
    { label: '用水(t)', value: Number(er.reduce((s, r) => s + (r.water || 0), 0).toFixed(1)) }
  ]
})
// 碳积分状态分布：可用/锁定/已售出
const creditStatusItems = computed(() => {
  const q = { available: 0, locked: 0, sold: 0 }
  credits.value.forEach(c => { q[c.status] = (q[c.status] || 0) + (Number(c.carbon_credits) || 0) })
  return [
    { label: '可用', value: Number(q.available.toFixed(1)), color: '#0d9488' },
    { label: '锁定(挂单中)', value: Number(q.locked.toFixed(1)), color: '#3b82f6' },
    { label: '已售出', value: Number(q.sold.toFixed(1)), color: '#f59e0b' }
  ]
})
// 挂单交易统计：各状态挂单积分量
const orderStatusItems = computed(() => {
  const q = { pending: 0, matched: 0, cancelled: 0 }
  sellOrders.value.forEach(o => { q[o.status] = (q[o.status] || 0) + (Number(o.quantity) || 0) })
  return [
    { label: '挂单中', value: Number(q.pending.toFixed(1)) },
    { label: '已成交', value: Number(q.matched.toFixed(1)) },
    { label: '已取消', value: Number(q.cancelled.toFixed(1)) }
  ]
})
function shortDate(t) {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return '-'
  return (d.getMonth() + 1) + '/' + d.getDate()
}

// Loading states
const loadingManual = ref(false)
const loadingCalc = ref(false)
const loadingStats = ref(false)
const loadingSell = ref(false)
const loadingQuery = ref(false)
const loadingUpload = ref(false)
const loadingAdvice = ref(false)

// 能耗数据 - 手动录入
const energyRecords = ref([])
const manualForm = reactive({ electricity: 0, gas: 0, water: 0 })
const manualMsg = ref('')
const manualMsgType = ref('success')

// 碳积分
const credits = ref([])
const stats = ref({})
const availableCredits = computed(() => credits.value.filter(c => c.status === 'available'))

// 挂单
const sellForm = reactive({ credit_id: 0, quantity: 0, unit_price: 0 })
const sellOrders = ref([])
const sellMsg = ref(''); const sellMsgType = ref('')

// 区块链
const queryNo = ref(''); const chainResult = ref(null); const chainInfo = ref(null)
const uploadType = ref('energy'); const uploadId = ref('')
const chainMsg = ref(''); const chainMsgType = ref('')

// AI建议
const adviceForm = reactive({ company_name: user.company || '测试企业', industry: '制造业' })
const adviceResult = ref('')

onMounted(() => { refreshData() })

async function refreshData() {
  try {
    const [er, cr, st, so, ci] = await Promise.all([
      energyAPI.list({ enterprise_id: user.id, page_size: 10 }),
      carbonAPI.myCredits({ enterprise_id: user.id, page_size: 50 }),
      carbonAPI.stats({ enterprise_id: user.id }),
      carbonAPI.sellOrders({ enterprise_id: user.id, page_size: 50 }),
      chainAPI.info()
    ])
    energyRecords.value = er.data.list
    credits.value = cr.data.list
    stats.value = st.data
    sellOrders.value = so.data.list
    chainInfo.value = ci.data
  } catch (e) { console.error(e) }
}
async function refreshStats() {
  loadingStats.value = true
  try {
    const r = await carbonAPI.stats({ enterprise_id: user.id })
    stats.value = r.data
  } catch (e) {} finally { loadingStats.value = false }
}
async function handleManualSubmit() {
  if (!manualForm.electricity || !manualForm.gas || !manualForm.water) {
    manualMsg.value = '请填写完整的能耗数据'; manualMsgType.value = 'error'
    return
  }
  loadingManual.value = true
  try {
    await energyAPI.create({
      enterprise_id: user.id,
      electricity: manualForm.electricity,
      gas: manualForm.gas,
      water: manualForm.water
    })
    manualMsg.value = '能耗数据提交成功'; manualMsgType.value = 'success'
    manualForm.electricity = 0; manualForm.gas = 0; manualForm.water = 0
    refreshData()
  } catch (e) {
    manualMsg.value = e?.msg || '提交失败'; manualMsgType.value = 'error'
  } finally { loadingManual.value = false }
}
async function handleCalculate(id) {
  if (!id) return
  loadingCalc.value = true
  try {
    await carbonAPI.calculate({ energy_record_id: id })
    refreshData()
  } catch (e) { alert(e?.msg || '核算失败') } finally { loadingCalc.value = false }
}
async function handleSell() {
  if (!sellForm.credit_id || sellForm.credit_id === 0) {
    sellMsg.value = '请选择积分类别'; sellMsgType.value = 'error'; return
  }
  loadingSell.value = true
  try {
    await carbonAPI.sell(sellForm)
    sellMsg.value = '挂单创建成功'; sellMsgType.value = 'success'
    refreshData()
  } catch (e) {
    sellMsg.value = e?.msg || '挂单失败'; sellMsgType.value = 'error'
  } finally { loadingSell.value = false }
}
async function handleQuery() {
  if (!queryNo.value) return
  loadingQuery.value = true
  try {
    const r = await chainAPI.query({ data_no: queryNo.value })
    chainResult.value = r.data
  } catch (e) {
    chainResult.value = null; alert(e?.msg || '查询失败')
  } finally { loadingQuery.value = false }
}
async function handleUpload() {
  if (!uploadId.value) return
  loadingUpload.value = true
  try {
    await chainAPI.upload({ data_type: uploadType.value, data_id: uploadId.value })
    chainMsg.value = '上链成功'; chainMsgType.value = 'success'
  } catch (e) {
    chainMsg.value = e?.msg || '上链失败'; chainMsgType.value = 'error'
  } finally { loadingUpload.value = false }
}
async function handleAdvice() {
  loadingAdvice.value = true
  try {
    const r = await adviceAPI.generate({ enterprise_id: user.id, ...adviceForm })
    adviceResult.value = r.data.advice
  } catch (e) { alert(e?.msg || '生成失败') } finally { loadingAdvice.value = false }
}
function formatTime(t) { return t ? new Date(t).toLocaleString() : '-' }
</script>

<style scoped>
/* 数据概览可视化面板 */
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
.dash-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.08);
}
.dash-icon {
  width: 36px; height: 36px;
  border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  font-size: 18px;
  margin-bottom: 10px;
}
.dash-info { margin-bottom: 8px; }
.dash-val { display: block; font-size: 24px; font-weight: 700; color: #1e293b; }
.dash-label { display: block; font-size: 12px; color: #94a3b8; margin-top: 2px; }
.dash-bar {
  height: 6px;
  background: #f1f5f9;
  border-radius: 3px;
  overflow: hidden;
}
.dash-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.8s ease;
}

/* 输入组标签样式 */
.input-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.input-label {
  font-size: 13px;
  font-weight: 500;
  color: #475569;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

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