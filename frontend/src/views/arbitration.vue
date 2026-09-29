<!--
  arbitration.vue - 交易仲裁中心
  功能：
    1. 企业 / 交易所：发起新仲裁（选交易 / 质押 / 其他 + 填纠纷描述）
    2. 仲裁案件列表（按角色过滤）
    3. 监管 / 交易所：执行裁决（resolve）并上链
    4. 查看案件详情（争议描述 / 双方证据 / 裁决结果 / 链上哈希）
  后端接口：
    POST /api/arbitration/create  → handlers.CreateArbitration
    GET  /api/arbitration/list    → handlers.ListArbitrations
    POST /api/arbitration/resolve → handlers.ResolveArbitration
-->
<template>
  <div class="arb-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>
          <el-icon :size="18"><ScaleToOriginal /></el-icon>
          交易仲裁中心
        </h2>
        <p>交易纠纷发起 · 链上证据存证 · 监管裁决 · 执行闭环</p>
      </div>
      <el-button
        type="primary"
        :icon="EditPen"
        @click="openCreate"
      >
        + 发起新仲裁
      </el-button>
    </div>

    <!-- ===== 统计卡 ===== -->
    <div class="stat-row">
      <div class="stat-card pending">
        <div class="sc-icon"><el-icon :size="18"><Warning /></el-icon></div>
        <div>
          <div class="sc-num">{{ counts.pending }}</div>
          <div class="sc-lab">待处理案件</div>
        </div>
      </div>
      <div class="stat-card processing">
        <div class="sc-icon"><el-icon :size="18"><Clock /></el-icon></div>
        <div>
          <div class="sc-num">{{ counts.processing }}</div>
          <div class="sc-lab">处理中</div>
        </div>
      </div>
      <div class="stat-card resolved">
        <div class="sc-icon"><el-icon :size="18"><CircleCheck /></el-icon></div>
        <div>
          <div class="sc-num">{{ counts.resolved }}</div>
          <div class="sc-lab">已裁决</div>
        </div>
      </div>
      <div class="stat-card chain">
        <div class="sc-icon"><el-icon :size="18"><Connection /></el-icon></div>
        <div>
          <div class="sc-num">{{ counts.on_chain }}</div>
          <div class="sc-lab">已上链存证</div>
        </div>
      </div>
    </div>

    <!-- ===== 案件列表 ===== -->
    <div class="list-card">
      <div class="lc-head">
        <h3>仲裁案件列表</h3>
        <div class="lc-filters">
          <el-select v-model="statusFilter" placeholder="全部状态" size="small" style="width:140px" clearable>
            <el-option label="待处理" value="pending" />
            <el-option label="处理中" value="processing" />
            <el-option label="已裁决" value="resolved" />
          </el-select>
          <el-input
            v-model="keyword" placeholder="搜索案件编号 / 交易号"
            :prefix-icon="Search" clearable size="small"
            style="width:220px"
          />
          <el-button :icon="Refresh" size="small" @click="refreshAll" :loading="loadingAll">刷新</el-button>
        </div>
      </div>

      <div v-if="loadingCases" class="list-loading">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
      </div>
      <div v-else-if="!filteredCases.length" class="list-empty">
        <el-icon :size="40"><ScaleToOriginal /></el-icon>
        <p>暂无仲裁案件</p>
      </div>

      <div v-else class="case-cards">
        <div v-for="c in filteredCases" :key="c.case_no" class="case-card" :class="c.status">
          <div class="cc-head">
            <div class="cc-id">
              <code class="mono">{{ c.case_no }}</code>
              <el-tag :class="'cc-type ' + c.case_type" size="small" effect="plain">
                {{ caseTypeMap[c.case_type] || c.case_type }}
              </el-tag>
              <el-tag :class="'cc-status ' + c.status" size="small" effect="dark">
                {{ caseStatusMap[c.status] || c.status }}
              </el-tag>
            </div>
            <div class="cc-amount">
              <span class="cc-label">争议金额</span>
              <span class="cc-num">¥{{ (c.dispute_amount || 0).toLocaleString() }}</span>
            </div>
          </div>

          <div class="cc-meta">
            <span><el-icon :size="12"><User /></el-icon> 申诉方: {{ c.applicant || '匿名' }}</span>
            <span><el-icon :size="12"><User /></el-icon> 被诉方: {{ c.respondent || '匿名' }}</span>
            <span><el-icon :size="12"><Document /></el-icon> 关联交易: <code>{{ c.transaction_no || '--' }}</code></span>
          </div>

          <div class="cc-desc">{{ c.description }}</div>

          <div class="cc-foot">
            <div class="cc-left">
              <el-icon :size="12"><Clock /></el-icon>
              提交于 {{ formatTime(c.created_at) }}
              <span v-if="c.on_chain" class="chain-badge">
                <el-icon :size="11"><Connection /></el-icon> 已上链
              </span>
            </div>
            <div class="cc-actions">
              <el-button size="small" link type="primary" @click="viewDetail(c)">查看详情</el-button>
              <!-- 监管 / 交易所：执行裁决 -->
              <el-button
                v-if="(role === 'regulator' || role === 'exchange') && c.status !== 'resolved'"
                size="small" type="warning"
                @click="openResolve(c)"
              >
                执行裁决
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 发起仲裁弹窗 ===== -->
    <el-dialog v-model="createVisible" width="520px" title="发起新仲裁">
      <el-form :model="createForm" label-width="110px">
        <el-form-item label="纠纷类型" required>
          <el-select v-model="createForm.case_type" style="width:100%">
            <el-option label="积分交易纠纷" value="credit_dispute" />
            <el-option label="质押融资纠纷" value="pledge_dispute" />
            <el-option label="IoT 数据争议" value="iot_dispute" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="关联交易号" required>
          <!-- 下拉选择：选项来自后端已上链交易(/arbitration/onchain-transactions)，禁止手输 -->
          <el-select
            v-model="createForm.transaction_no"
            filterable
            placeholder="请选择已上链的碳积分交易"
            style="width:100%"
            @change="onTradeChange"
          >
            <el-option
              v-for="t in tradeOptions"
              :key="t.tx_no"
              :label="t.tx_no"
              :value="t.tx_no"
            >
              <div class="trade-option">
                <span class="trade-no">{{ t.tx_no }}</span>
                <span class="trade-parties">{{ t.seller_name }} → {{ t.buyer_name }}</span>
                <span class="trade-time">{{ t.created_at }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="被诉方企业" required>
          <!-- 下拉选择：选项为平台注册的园区入驻企业；选交易后自动回填对手方，可手动改选 -->
          <el-select
            v-model="createForm.respondent_id"
            filterable
            placeholder="请选择被诉方企业"
            style="width:100%"
            @change="onRespondentChange"
          >
            <el-option
              v-for="e in enterpriseOptions"
              :key="e.id"
              :label="e.company || e.username"
              :value="e.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="争议金额(元)">
          <el-input-number v-model="createForm.dispute_amount" :min="0" :precision="2" style="width:100%" />
        </el-form-item>
        <el-form-item label="纠纷描述" required>
          <el-input v-model="createForm.description" type="textarea" :rows="4" placeholder="请描述纠纷发生的时间、具体情况、您的诉求..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="loadingCreate" @click="doCreate">提交仲裁申请</el-button>
      </template>
    </el-dialog>

    <!-- ===== 案件详情 / 裁决弹窗 ===== -->
    <el-dialog v-model="detailVisible" width="620px" :title="'案件详情 · ' + (currentCase?.case_no || '')">
      <div v-if="currentCase" class="detail-block">
        <div class="detail-row">
          <span>纠纷类型</span><b>{{ caseTypeMap[currentCase.case_type] || currentCase.case_type }}</b>
          <span style="margin-left:24px">状态</span><b :style="{ color: currentCase.status === 'resolved' ? '#10b981' : '#d97706' }">
            {{ caseStatusMap[currentCase.status] }}
          </b>
        </div>
        <div class="detail-row">
          <span>申诉方</span><b>{{ currentCase.applicant }}</b>
          <span style="margin-left:24px">被诉方</span><b>{{ currentCase.respondent }}</b>
        </div>
        <div class="detail-row">
          <span>争议金额</span><b class="green">¥{{ currentCase.dispute_amount?.toLocaleString() }}</b>
          <span style="margin-left:24px">关联交易</span><code>{{ currentCase.transaction_no || '--' }}</code>
          <el-button v-if="currentCase.transaction_no" size="small" type="primary" link style="margin-left:8px" @click="traceTrade(currentCase)">
            溯源原始上链交易 →
          </el-button>
        </div>
        <div class="detail-row">
          <span>提交时间</span>{{ formatTime(currentCase.created_at) }}
        </div>

        <h4 class="sub-title">纠纷描述</h4>
        <div class="desc-box">{{ currentCase.description }}</div>

        <div v-if="currentCase.resolution" class="resolution-box">
          <h4 class="sub-title">
            <el-icon :size="14"><CircleCheck /></el-icon>
            裁决结果
          </h4>
          <div class="desc-box">{{ currentCase.resolution }}</div>
          <div class="detail-row">
            <span>裁决时间</span>{{ formatTime(currentCase.resolved_at) }}
            <span style="margin-left:24px">裁决人</span>{{ currentCase.resolver || '监管核查机构' }}
          </div>
        </div>

        <div v-if="currentCase.block_hash" class="chain-box">
          <span>🔗 链上哈希</span>
          <code>{{ currentCase.block_hash }}</code>
        </div>
      </div>

      <!-- 裁决表单（监管/交易所 + 未裁决案件） -->
      <template v-if="(role === 'regulator' || role === 'exchange') && currentCase && currentCase.status !== 'resolved'">
        <el-divider />
        <h4 class="sub-title">执行裁决</h4>
        <el-form :model="resolveForm" label-width="110px">
          <el-form-item label="裁决结果">
            <el-select v-model="resolveForm.result" style="width:100%">
              <el-option label="申诉方胜诉（全额赔付）" value="applicant_win" />
              <el-option label="被诉方胜诉（驳回申诉）" value="respondent_win" />
              <el-option label="双方和解（部分赔付）" value="settled" />
              <el-option label="证据不足（退回补充）" value="insufficient" />
            </el-select>
          </el-form-item>
          <el-form-item label="裁决说明" required>
            <el-input v-model="resolveForm.resolution" type="textarea" :rows="3" placeholder="请说明裁决依据与处理方式..." />
          </el-form-item>
        </el-form>
      </template>

      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button
          v-if="(role === 'regulator' || role === 'exchange') && currentCase && currentCase.status !== 'resolved'"
          type="warning"
          :loading="loadingResolve"
          @click="doResolve"
        >
          确认裁决并上链
        </el-button>
      </template>
    </el-dialog>

    <!-- 原始上链交易溯源弹窗：展示案件关联交易的真实区块存证 -->
    <el-dialog v-model="traceVisible" width="560px" :title="'原始上链交易 · ' + (traceData?.data_no || currentCase?.transaction_no || '')">
      <div v-loading="traceLoading" style="min-height:80px">
        <template v-if="traceData">
          <div class="detail-row"><span>业务单号</span><code>{{ traceData.data_no }}</code></div>
          <div class="detail-row"><span>业务类型</span><b>碳积分交易凭证</b></div>
          <div class="detail-row"><span>区块高度</span><b>#{{ traceData.block_index }}</b></div>
          <div class="detail-row"><span>区块哈希</span><code style="word-break:break-all">{{ traceData.block_hash }}</code></div>
          <div class="detail-row"><span>业务哈希(SHA-256)</span><code style="word-break:break-all">{{ traceData.original_hash }}</code></div>
          <div class="detail-row"><span>上链时间</span>{{ formatTime(traceData.timestamp * 1000) }}</div>
          <div class="detail-row"><span>完整性</span><b :style="{ color: traceData.match ? '#10b981' : '#dc2626' }">{{ traceData.match ? '哈希链完整，区块未被篡改' : '区块哈希校验失败' }}</b></div>
        </template>
      </div>
      <template #footer>
        <el-button @click="traceVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ScaleToOriginal, EditPen, Warning, Clock, CircleCheck, Connection,
  Search, Refresh, Loading, User, Document
} from '@element-plus/icons-vue'
import { rwaAPI, chainAPI, directoryAPI } from '../api/index.js'
import { brandText } from '../utils/brand.js'

const user = JSON.parse(localStorage.getItem('user') || '{}')
const role = user.role || 'enterprise'

const caseTypeMap = { credit_dispute: '积分交易纠纷', pledge_dispute: '质押融资纠纷', iot_dispute: 'IoT数据争议', other: '其他' }
const caseStatusMap = { pending: '待处理', processing: '处理中', resolved: '已裁决' }

/* ===== 状态 ===== */
const cases = ref([])
const loadingCases = ref(false)
const loadingAll = ref(false)
const loadingCreate = ref(false)
const loadingResolve = ref(false)
const statusFilter = ref('')
const keyword = ref('')

const createVisible = ref(false)
const detailVisible = ref(false)
const currentCase = ref(null)

/* ===== 表单 ===== */
const createForm = reactive({
  case_type: 'credit_dispute',
  transaction_no: '',
  respondent_id: 0,   // 被诉方企业ID(下拉选择值，后端按ID精确匹配)
  respondent: '',     // 被诉方企业名称(提交给后端展示/兼容字段)
  dispute_amount: 0,
  description: '',
})
const resolveForm = reactive({ result: 'settled', resolution: '' })

/* ===== 发起新仲裁：下拉数据源(真实后端数据，不使用模拟数据) ===== */
const tradeOptions = ref([])       // 已上链交易列表(/arbitration/onchain-transactions)
const enterpriseOptions = ref([])  // 平台注册企业列表(/enterprises)
const optionsLoading = ref(false)

/** 打开发起弹窗时懒加载下拉数据(已加载过则复用缓存) */
async function loadCreateOptions() {
  if (tradeOptions.value.length && enterpriseOptions.value.length) return
  optionsLoading.value = true
  try {
    const [tradeRes, entRes] = await Promise.all([
      rwaAPI.onchainTransactions().catch(() => null),
      directoryAPI.list().catch(() => null),
    ])
    tradeOptions.value = tradeRes?.data?.list || []
    enterpriseOptions.value = entRes?.data?.list || []
  } finally {
    optionsLoading.value = false
  }
}

/** 当前用户的名称集合(用于判断交易的"对手方") */
function myNameSet() {
  return [user.username, user.company, brandText(user.username)]
    .filter(Boolean)
}

/**
 * 字段联动：选中关联交易 → 自动回填争议金额(成交金额)与被诉方(交易对手方)
 * 对手方判断：与当前用户名称不一致的那一方；判断不出时默认取卖方
 */
function onTradeChange(txNo) {
  const t = tradeOptions.value.find(x => x.tx_no === txNo)
  if (!t) return
  createForm.dispute_amount = t.total_amount || 0
  const names = myNameSet()
  const isSellerMe = names.includes(t.seller_name)
  const isBuyerMe = names.includes(t.buyer_name)
  const oppId = isSellerMe ? t.buyer_id : (isBuyerMe ? t.seller_id : t.seller_id)
  const oppName = isSellerMe ? t.buyer_name : (isBuyerMe ? t.seller_name : t.seller_name)
  createForm.respondent_id = oppId
  createForm.respondent = oppName
}

/** 手动改选被诉方企业(允许在自动回填后重新选择) */
function onRespondentChange(id) {
  const e = enterpriseOptions.value.find(x => x.id === id)
  if (e) createForm.respondent = e.company || e.username
}

/* ===== 过滤后的案件 ===== */
const filteredCases = computed(() => {
  let arr = cases.value
  if (statusFilter.value) arr = arr.filter(c => c.status === statusFilter.value)
  if (keyword.value) {
    const kw = keyword.value.toLowerCase()
    arr = arr.filter(c =>
      c.case_no?.toLowerCase().includes(kw) ||
      c.transaction_no?.toLowerCase().includes(kw) ||
      c.description?.toLowerCase().includes(kw)
    )
  }
  // 案件可见性由后端按用户ID过滤(enterprise 仅见自己为原告/被告的案件)，此处仅做本地筛选
  return arr
})

const counts = computed(() => ({
  pending:    cases.value.filter(c => c.status === 'pending').length,
  processing: cases.value.filter(c => c.status === 'processing').length,
  resolved:   cases.value.filter(c => c.status === 'resolved').length,
  on_chain:   cases.value.filter(c => c.on_chain).length,
}))

/* ===== 加载 ===== */
async function loadCases() {
  loadingCases.value = true
  try {
    const res = await rwaAPI.arbitrations({ page: 1, page_size: 30 })
    // 案件全部来自后端真实数据(由已上链交易产生纠纷后生成)，不再使用任何模拟数据
    cases.value = res.data?.list || []
  } catch (e) {
    ElMessage.error(e?.msg || '加载仲裁案件失败')
    cases.value = []
  } finally {
    loadingCases.value = false
  }
}

async function refreshAll() {
  loadingAll.value = true
  await loadCases()
  loadingAll.value = false
  ElMessage.success('已刷新')
}

/* ===== 发起 ===== */
async function openCreate() {
  createForm.case_type = 'credit_dispute'
  createForm.transaction_no = ''
  createForm.respondent_id = 0
  createForm.respondent = ''
  createForm.dispute_amount = 0
  createForm.description = ''
  createVisible.value = true
  // 打开弹窗时拉取下拉数据源(已上链交易 + 注册企业，真实后端数据)
  loadCreateOptions()
}

async function doCreate() {
  if (!createForm.case_type || !createForm.description) {
    ElMessage.warning('请填写纠纷类型和描述'); return
  }
  if (!createForm.transaction_no) {
    ElMessage.warning('请从下拉列表选择关联的已上链交易'); return
  }
  if (!createForm.respondent_id) {
    ElMessage.warning('请选择被诉方企业'); return
  }
  loadingCreate.value = true
  try {
    await rwaAPI.createArbitration({
      case_type: createForm.case_type,
      transaction_no: createForm.transaction_no,
      respondent_id: createForm.respondent_id,
      respondent: createForm.respondent,
      dispute_amount: createForm.dispute_amount,
      description: createForm.description,
    })
    ElMessage.success('仲裁申请已提交：案件已关联上链交易并存证')
    createVisible.value = false
    refreshAll()
  } catch (e) {
    ElMessage.error(e?.msg || '仲裁申请提交失败')
  } finally {
    loadingCreate.value = false
  }
}

/* ===== 查看详情 ===== */
function viewDetail(c) {
  currentCase.value = c
  detailVisible.value = true
  resolveForm.result = 'settled'
  resolveForm.resolution = ''
}
function openResolve(c) { viewDetail(c) }

/* ===== 溯源原始上链交易：读取案件关联交易的真实区块存证 ===== */
const traceVisible = ref(false)
const traceLoading = ref(false)
const traceData = ref(null)

/** 调用 /api/chain/query 按交易凭证号查询原始上链区块记录 */
async function traceTrade(c) {
  if (!c.transaction_no) { ElMessage.warning('该案件未关联上链交易'); return }
  traceVisible.value = true
  traceLoading.value = true
  traceData.value = null
  try {
    const res = await chainAPI.query({ data_no: c.transaction_no, data_type: 'transaction' })
    traceData.value = res.data
  } catch (e) {
    ElMessage.error(e?.msg || '未找到该交易的链上记录')
    traceVisible.value = false
  } finally {
    traceLoading.value = false
  }
}

/* ===== 执行裁决 ===== */
async function doResolve() {
  if (!resolveForm.resolution) { ElMessage.warning('请填写裁决说明'); return }
  try {
    await ElMessageBox.confirm('确认执行该裁决并上链存证？裁决结果将不可撤销', '裁决确认', { type: 'warning' })
  } catch { return }
  loadingResolve.value = true
  try {
    // 裁决结果与说明合并为裁决文书存入链上案件
    const resultMap = {
      applicant_win: '申诉方胜诉（全额赔付）',
      respondent_win: '被诉方胜诉（驳回申诉）',
      settled: '双方和解（部分赔付）',
      insufficient: '证据不足（退回补充）',
    }
    await rwaAPI.resolveArbitration({
      case_no: currentCase.value.case_no,
      verdict: (resultMap[resolveForm.result] || '已裁决') + '：' + resolveForm.resolution,
      status: resolveForm.result === 'insufficient' ? 'dismissed' : 'resolved',
    })
    ElMessage.success('裁决已执行，链上存证成功')
  } catch (e) {
    ElMessage.error(e?.msg || '裁决执行失败')
  } finally {
    loadingResolve.value = false
    detailVisible.value = false
    refreshAll()
  }
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

onMounted(() => loadCases())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #7c3aed, #2563eb);
  border-radius: var(--radius-lg);
  color: #fff;
}
.page-head h2 { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.page-head p { font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }

/* ===== 统计卡 ===== */
.stat-row {
  display: grid; grid-template-columns: repeat(4, 1fr);
  gap: 12px; margin-bottom: 16px;
}
.stat-card {
  display: flex; gap: 12px; align-items: center;
  padding: 16px 18px;
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius);
  border-left: 4px solid transparent;
}
.stat-card.pending    { border-left-color: #f59e0b; }
.stat-card.processing { border-left-color: #0d9488; }
.stat-card.resolved   { border-left-color: #10b981; }
.stat-card.chain      { border-left-color: #06b6d4; }
.sc-icon {
  width: 40px; height: 40px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.stat-card.pending    .sc-icon { background: rgba(245,158,11,0.12); color: #d97706; }
.stat-card.processing .sc-icon { background: rgba(16,185,129,0.12); color: #2563eb; }
.stat-card.resolved   .sc-icon { background: rgba(16,185,129,0.12); color: #10b981; }
.stat-card.chain      .sc-icon { background: rgba(6,182,212,0.12); color: #0891b2; }
.sc-num { font-size: 22px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary); }
.sc-lab { font-size: 11.5px; color: var(--text-tertiary); }

/* ===== 列表 ===== */
.list-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.lc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.lc-head h3 { font-size: 14px; font-weight: 600; margin: 0; }
.lc-filters { display: flex; gap: 10px; align-items: center; }

.list-loading, .list-empty { padding: 60px; text-align: center; color: var(--text-tertiary); }
.list-empty p { margin-top: 10px; font-size: 12.5px; }

/* 案件卡片 */
.case-cards { display: flex; flex-direction: column; gap: 12px; }
.case-card {
  padding: 16px 18px;
  background: var(--bg-secondary);
  border: 1px solid var(--line-color);
  border-left: 4px solid;
  border-radius: 10px;
  transition: all 0.2s;
}
.case-card:hover { box-shadow: 0 6px 16px rgba(0,0,0,0.06); }
.case-card.pending    { border-left-color: #f59e0b; }
.case-card.processing { border-left-color: #0d9488; }
.case-card.resolved   { border-left-color: #10b981; }

.cc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.cc-id { display: flex; align-items: center; gap: 8px; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11.5px; color: var(--text-primary); }
.cc-type, .cc-status { font-size: 10.5px !important; }
.cc-type.credit_dispute  { background: rgba(124,58,237,0.1); color: #7c3aed; }
.cc-type.pledge_dispute  { background: rgba(245,158,11,0.1); color: #d97706; }
.cc-type.iot_dispute     { background: rgba(6,182,212,0.1); color: #0891b2; }
.cc-status.pending       { background: rgba(245,158,11,0.15); color: #d97706; }
.cc-status.processing    { background: rgba(16,185,129,0.15); color: #2563eb; }
.cc-status.resolved      { background: rgba(16,185,129,0.15); color: #10b981; }

.cc-amount { text-align: right; }
.cc-label { font-size: 10.5px; color: var(--text-tertiary); margin-right: 6px; }
.cc-num { font-size: 15px; font-weight: 700; color: var(--text-primary); font-variant-numeric: tabular-nums; }

.cc-meta { display: flex; gap: 14px; flex-wrap: wrap; font-size: 11.5px; color: var(--text-tertiary); margin-bottom: 8px; }
.cc-meta span { display: inline-flex; align-items: center; gap: 4px; }

.cc-desc {
  font-size: 12.5px; color: var(--text-secondary);
  line-height: 1.6;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
  overflow: hidden;
  margin-bottom: 10px;
}

.cc-foot {
  display: flex; justify-content: space-between; align-items: center;
  padding-top: 10px; border-top: 1px dashed var(--line-color);
  font-size: 11px; color: var(--text-tertiary);
}
.cc-left { display: flex; align-items: center; gap: 8px; }
.chain-badge {
  display: inline-flex; align-items: center; gap: 2px;
  padding: 1px 6px; border-radius: 999px;
  background: rgba(6,182,212,0.1); color: #0891b2;
  font-size: 10px;
  margin-left: 6px;
}
.cc-actions { display: flex; gap: 6px; }

/* ===== 详情 ===== */
.detail-row {
  display: flex; gap: 24px;
  font-size: 12px; margin-bottom: 8px;
}
.detail-row span:first-child { color: var(--text-tertiary); }
.detail-row b { color: var(--text-primary); font-weight: 500; }

.sub-title {
  font-size: 13px; font-weight: 600; margin: 14px 0 8px;
  display: flex; align-items: center; gap: 6px;
}

.desc-box {
  padding: 12px 14px;
  background: var(--bg-secondary);
  border-radius: 8px;
  font-size: 12.5px; line-height: 1.7;
  color: var(--text-secondary);
}

.resolution-box .desc-box {
  background: rgba(16,185,129,0.06);
  border-left: 3px solid #10b981;
}

.chain-box {
  margin-top: 14px;
  padding: 10px 14px;
  background: #0a1628;
  border-radius: 6px;
  font-size: 11.5px;
}
.chain-box span { color: #64748b; margin-right: 10px; }
.chain-box code { font-family: ui-monospace, Consolas, monospace; color: #5eead4; word-break: break-all; }
.green { color: var(--primary-green); }
/* ===== 发起新仲裁：已上链交易下拉选项 ===== */
.trade-option {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}
.trade-option .trade-no {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: var(--primary-green, #16a34a);
  flex-shrink: 0;
}
.trade-option .trade-parties {
  flex: 1;
  font-size: 12px;
  color: var(--text-primary, #1f2937);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.trade-option .trade-time {
  font-size: 11px;
  color: var(--text-tertiary, #9ca3af);
  flex-shrink: 0;
}
</style>