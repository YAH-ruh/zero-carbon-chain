<!--
  pledge.vue - 碳资产质押融资（仅 enterprise 角色）
  功能：
    1. 账户总览：可用积分 / 已质押 / 融资金额 / 还款进度
    2. 发起质押：选积分 → 评估额度 → 确认生成质押单并上链
    3. 还款记录：未还/已还两种状态
    4. 链上质押证明
  后端接口：
    POST /api/pledge/create → handlers.CreatePledge
    GET  /api/pledge/list   → handlers.ListPledges
-->
<template>
  <div class="pledge-page page">

    <!-- ===== 头部：账户总览 ===== -->
    <div class="acct-card">
      <div class="acct-head">
        <h2>
          <el-icon :size="18"><Money /></el-icon>
          碳资产质押融资
        </h2>
        <p>将持有碳积分质押，获取现金流。质押期间积分被锁定，还款完成后自动解锁。</p>
      </div>
      <div class="acct-metrics">
        <div class="am-item">
          <div class="am-icon free"><el-icon :size="20"><Coin /></el-icon></div>
          <div class="am-val">{{ availableCredits }}</div>
          <div class="am-lab">可用碳积分</div>
        </div>
        <div class="am-item">
          <div class="am-icon locked"><el-icon :size="20"><Lock /></el-icon></div>
          <div class="am-val">{{ pledgedCredits }}</div>
          <div class="am-lab">已质押积分</div>
        </div>
        <div class="am-item">
          <div class="am-icon loan"><el-icon :size="20"><DataAnalysis /></el-icon></div>
          <div class="am-val">¥{{ totalLoan.toLocaleString() }}</div>
          <div class="am-lab">累计融资金额</div>
        </div>
        <div class="am-item">
          <div class="am-icon repay"><el-icon :size="20"><CircleCheck /></el-icon></div>
          <div class="am-val">¥{{ repaidAmount.toLocaleString() }}</div>
          <div class="am-lab">已偿还金额</div>
        </div>
        <div class="am-item">
          <div class="am-icon pending"><el-icon :size="20"><Clock /></el-icon></div>
          <div class="am-val">¥{{ unpaidAmount.toLocaleString() }}</div>
          <div class="am-lab">待偿还</div>
        </div>
      </div>

      <!-- 还款进度条 -->
      <div v-if="totalLoan > 0" class="repay-progress">
        <div class="rp-label">
          还款进度
          <span class="rp-val">{{ repayProgress }}%</span>
        </div>
        <el-progress :percentage="repayProgress" :color="repayProgress >= 80 ? '#10b981' : repayProgress >= 50 ? '#f59e0b' : '#ef4444'" :stroke-width="8" />
      </div>
    </div>

    <!-- ===== 发起新质押 ===== -->
    <div class="pledge-form-card">
      <div class="pf-head">
        <h3>发起新质押融资</h3>
        <el-tag effect="dark" size="small" type="primary">
          质押率 {{ pledgeRate }}% · 年利率 {{ (annualRate * 100).toFixed(0) }}%
        </el-tag>
      </div>
      <el-form :model="pledgeForm" label-width="120px" class="pf-form">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="质押积分数量" required>
              <el-input-number
                v-model="pledgeForm.amount"
                :min="1"
                :max="maxPledgeable"
                :step="10"
                style="width:100%"
              />
              <div class="form-hint">最多可质押 {{ availableCredits }} 积分</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="质押期限">
              <el-select v-model="pledgeForm.term" style="width:100%">
                <el-option :value="3"  label="3 个月（年化 6%）" />
                <el-option :value="6"  label="6 个月（年化 7%）" />
                <el-option :value="12" label="12 个月（年化 8%）" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="预估融资额">
              <el-input :model-value="estimatedLoan" disabled>
                <template #append>元</template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="预估利息">
              <el-input :model-value="estimatedInterest" disabled>
                <template #append>元</template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 质押条款 -->
        <el-alert type="warning" :closable="false" show-icon style="margin-bottom:14px">
          <template #title>
            <div style="display:flex;justify-content:space-between;font-size:12px">
              <span>质押期间积分将被锁定，无法挂单交易。还款完成后自动解锁。</span>
              <el-checkbox v-model="pledgeForm.agreed">我已阅读并同意质押条款</el-checkbox>
            </div>
          </template>
        </el-alert>

        <el-button
          type="primary"
          :icon="Money"
          :disabled="!pledgeForm.amount || !pledgeForm.agreed"
          :loading="loadingCreate"
          @click="confirmPledge"
        >
          确认质押并上链存证
        </el-button>
      </el-form>
    </div>

    <!-- ===== 质押订单列表 ===== -->
    <div class="orders-card">
      <div class="oc-head">
        <h3>质押融资订单</h3>
        <div class="oc-filters">
          <el-radio-group v-model="statusFilter" size="small">
            <el-radio-button label="all">全部</el-radio-button>
            <el-radio-button label="active">质押中</el-radio-button>
            <el-radio-button label="cleared">已还清</el-radio-button>
          </el-radio-group>
          <el-button :icon="Refresh" @click="refreshAll" :loading="loadingAll">刷新</el-button>
        </div>
      </div>

      <el-table :data="filteredOrders" stripe style="width:100%" v-loading="loadingOrders" empty-text="暂无质押订单">
        <el-table-column prop="pledge_no" label="质押编号" width="180">
          <template #default="{ row }"><code class="mono">{{ row.pledge_no }}</code></template>
        </el-table-column>
        <el-table-column label="质押积分" width="110" align="right">
          <template #default="{ row }"><b class="green">{{ row.pledge_amount }}</b></template>
        </el-table-column>
        <el-table-column label="融资额(元)" width="120" align="right">
          <template #default="{ row }">{{ row.loan_amount.toLocaleString() }}</template>
        </el-table-column>
        <el-table-column label="期限" width="90" align="center">
          <template #default="{ row }">{{ row.term_months ? row.term_months + ' 个月' : '--' }}</template>
        </el-table-column>
        <el-table-column label="年利率" width="90" align="center">
          <template #default="{ row }">{{ row.annual_rate ? (row.annual_rate * 100).toFixed(1) + '%' : '--' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="120" align="center">
          <template #default="{ row }">
            <el-tag :class="'pledge-status ' + row.status" size="small" effect="plain">
              {{ pledgeStatusMap[row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="180">
          <template #default="{ row }">
            <el-progress
              v-if="row.status === 'active'"
              :percentage="Math.min(100, Math.round((row.repaid_amount || 0) / (row.loan_amount || 1) * 100))"
              :stroke-width="6"
              :show-text="true"
            />
            <span v-else style="color:#10b981;font-size:12px">已完成</span>
          </template>
        </el-table-column>
        <el-table-column label="上链" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" type="success" size="small">已上链</el-tag>
            <el-tag v-else type="info" size="small">--</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="质押时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Money, Coin, Lock, DataAnalysis, CircleCheck, Clock, Refresh
} from '@element-plus/icons-vue'
import { rwaAPI, carbonAPI } from '../api/index.js'

/* ===== 常量 ===== */
const pledgeRate = 80        // 质押率 80%
const pricePerCredit = 50    // 碳积分单价 50 元
const annualRate = 0.07      // 年化 7%（展示用）
const termRates = { 3: 0.06, 6: 0.07, 12: 0.08 }

const pledgeStatusMap = {
  active: '质押中',
  cleared: '已还清',
  repaid: '已还款',
  overdue: '逾期',
  cancelled: '已取消'
}

/* ===== 状态 ===== */
const credits = ref([])
const orders = ref([])
const loadingOrders = ref(false)
const loadingAll = ref(false)
const loadingCreate = ref(false)
const statusFilter = ref('all')

/* ===== 表单 ===== */
const pledgeForm = reactive({
  amount: 100,
  term: 6,
  agreed: false,
})

const availableCredits = computed(() => {
  const avail = credits.value.reduce((s, c) => s + (c.carbon_credits || 0), 0) - pledgedCredits.value
  return Math.max(0, Math.round(avail))
})
// InputNumber 要求 max >= min，积分未加载完/不足时兜底为 1，防止 min>max 崩溃
const maxPledgeable = computed(() => Math.max(1, availableCredits.value))
const pledgedCredits = computed(() => orders.value.filter(o => o.status === 'active').reduce((s, o) => s + o.pledge_amount, 0))
const totalLoan = computed(() => orders.value.reduce((s, o) => s + o.loan_amount, 0))
// 应付本息：质押中订单按 本金×(1+年利率×期限/12) 预估，已还清订单取实际偿还金额
const payableAmount = computed(() => orders.value.reduce((s, o) => {
  const payable = o.loan_amount * (1 + (o.annual_rate || 0.07) * (o.term_months || 6) / 12)
  return s + (o.status === 'active' ? payable : (o.repaid_amount || payable))
}, 0))
const repaidAmount = computed(() => orders.value
  .filter(o => o.status === 'cleared' || o.status === 'repaid')
  .reduce((s, o) => s + (o.repaid_amount || 0), 0))
const unpaidAmount = computed(() => Math.max(0, Math.round(payableAmount.value - repaidAmount.value)))
const repayProgress = computed(() => payableAmount.value ? Math.min(100, Math.round(repaidAmount.value / payableAmount.value * 100)) : 0)

const estimatedLoan = computed(() => {
  if (!pledgeForm.amount) return 0
  return Math.round(pledgeForm.amount * pricePerCredit * pledgeRate / 100)
})
const estimatedInterest = computed(() => {
  const rate = termRates[pledgeForm.term] || 0.07
  const months = pledgeForm.term / 12
  return Math.round(estimatedLoan.value * rate * months)
})

const filteredOrders = computed(() => {
  if (statusFilter.value === 'active') return orders.value.filter(o => o.status === 'active')
  if (statusFilter.value === 'cleared') return orders.value.filter(o => o.status === 'cleared' || o.status === 'repaid')
  return orders.value
})

/* ===== 加载 ===== */
async function loadCredits() {
  try {
    const res = await carbonAPI.myCredits({ page: 1, page_size: 50 })
    credits.value = res.data?.list || res.data || []
  } catch {
    // 数据真实性原则：加载失败展示空列表并提示，不使用模拟数据兜底
    credits.value = []
    ElMessage.error('碳积分加载失败，请检查后端服务')
  }
}

async function loadOrders() {
  loadingOrders.value = true
  try {
    const res = await rwaAPI.pledges({ page: 1, page_size: 20 })
    orders.value = res.data?.list || res.data || []
  } catch {
    // 数据真实性原则：加载失败展示空列表并提示，不使用模拟数据兜底
    orders.value = []
    ElMessage.error('质押订单加载失败，请检查后端服务')
  } finally {
    loadingOrders.value = false
  }
}

async function refreshAll() {
  loadingAll.value = true
  await Promise.all([loadCredits(), loadOrders()])
  loadingAll.value = false
  ElMessage.success('数据已刷新')
}

/* ===== 发起质押 ===== */
async function confirmPledge() {
  if (availableCredits.value <= 0) {
    ElMessage.warning('当前无可质押的碳积分')
    return
  }
  if (pledgeForm.amount > availableCredits.value) {
    ElMessage.warning(`质押数量不能超过可用积分 ${availableCredits.value}`)
    return
  }
  try {
    await ElMessageBox.confirm(
      `确认将 ${pledgeForm.amount} 碳积分质押，融资 ¥${estimatedLoan.value}（${pledgeForm.term} 个月，年利率 ${((termRates[pledgeForm.term] || 0.07) * 100).toFixed(1)}%）`,
      '质押确认',
      { confirmButtonText: '确认质押', cancelButtonText: '再想想', type: 'warning' }
    )
  } catch { return }

  loadingCreate.value = true
  try {
    await rwaAPI.createPledge({
      pledge_amount: pledgeForm.amount,
      term_months: pledgeForm.term,
    })
    ElMessage.success(`质押成功！融资 ¥${estimatedLoan.value} 已划入账户，链上存证已生成`)
    pledgeForm.amount = 100
    pledgeForm.agreed = false
    refreshAll()
  } catch {
    ElMessage.error('质押创建失败，请稍后重试')
  } finally {
    loadingCreate.value = false
  }
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

onMounted(() => refreshAll())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 账户总览卡 ===== */
.acct-card {
  padding: 22px;
  margin-bottom: 16px;
  background: linear-gradient(135deg, #059669, #0891b2);
  border-radius: var(--radius-lg);
  color: #fff;
}
.acct-head h2 { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.acct-head p { font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }

.acct-metrics {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px;
  margin: 16px 0;
}
.am-item {
  display: flex; flex-direction: column; gap: 4px;
  padding: 14px;
  background: rgba(255,255,255,0.12);
  border-radius: 10px;
}
.am-icon { display: inline-flex; align-items: center; justify-content: center; width: 36px; height: 36px; border-radius: 8px; }
.am-icon.free   { background: rgba(255,255,255,0.25); }
.am-icon.locked { background: rgba(251,191,36,0.3); }
.am-icon.loan   { background: rgba(34,211,238,0.3); }
.am-icon.repay  { background: rgba(16,185,129,0.3); }
.am-icon.pending{ background: rgba(239,68,68,0.3); }
.am-val { font-size: 22px; font-weight: 700; font-variant-numeric: tabular-nums; margin-top: 4px; }
.am-lab { font-size: 11px; color: rgba(255,255,255,0.85); }

.repay-progress {
  background: rgba(255,255,255,0.1);
  padding: 12px 16px;
  border-radius: 8px;
}
.rp-label { display: flex; justify-content: space-between; font-size: 12px; margin-bottom: 6px; }
.rp-val { font-weight: 700; }

/* ===== 表单 ===== */
.pledge-form-card, .orders-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.pf-head, .oc-head {
  display: flex; justify-content: space-between; align-items: center;
  padding-bottom: 12px; margin-bottom: 14px;
  border-bottom: 1px solid var(--line-color);
}
.pf-head h3, .oc-head h3 { font-size: 14px; font-weight: 600; margin: 0; }

.form-hint { font-size: 11px; color: var(--text-tertiary); margin-top: 4px; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }
.green { color: var(--primary-green); }

.oc-filters { display: flex; gap: 10px; align-items: center; }

.pledge-status.active   { background: rgba(245,158,11,0.08); color: #d97706; }
.pledge-status.repaid   { background: rgba(16,185,129,0.08); color: #10b981; }
.pledge-status.overdue  { background: rgba(239,68,68,0.08); color: #ef4444; }
.pledge-status.cancelled{ background: rgba(148,163,184,0.08); color: #64748b; }

@media (max-width: 900px) {
  .acct-metrics { grid-template-columns: repeat(2, 1fr); }
}
</style>