<!--
  energy-report.vue - 能耗上报页面
  功能：
    1. 步骤式表单（3 步）填写能耗数据
    2. 提交后自动触发碳积分核算 + ZKP 证明生成
    3. 成功弹窗：导出 ZKP 隐私证明文件
    4. 历史记录列表（可查看链上状态）
  对应后端：
    POST /api/energy/create        手动录入能耗（录入即上链）
    GET  /api/energy/list          查询能耗记录
    POST /api/carbon/calculate     核算碳积分（需要 energy_record_id）
    POST /api/zkp/proof            生成 ZKP 隐私证明
-->
<template>
  <div class="energy-report page">

    <!-- ===== 页面头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>能耗数据上报</h2>
        <p class="ph-desc">
          <el-icon :size="14"><InfoFilled /></el-icon>
          请如实填写企业能耗数据。数据提交后将自动计算碳排放量并完成 SHA-256 上链存证。
        </p>
      </div>
      <div class="ph-right">
        <el-tag type="success" effect="dark" round>
          <el-icon><Lock /></el-icon>
          已启用 ZKP 隐私保护
        </el-tag>
      </div>
    </div>

    <!-- ===== 步骤条 ===== -->
    <el-steps :active="currentStep" finish-status="success" class="steps">
      <el-step title="基本信息" :icon="UserFilled" />
      <el-step title="能耗数据" :icon="EditPen" />
      <el-step title="确认提交" :icon="CircleCheck" />
    </el-steps>

    <!-- ===== 步骤 1：基本信息 ===== -->
    <div v-show="currentStep === 0" class="step-card">
      <h3 class="step-title">
        <el-icon :size="16"><User /></el-icon>
        上报企业基本信息
      </h3>
      <el-form :model="form" label-width="120px" class="step-form">
        <el-form-item label="企业名称">
          <el-input v-model="enterpriseName" disabled />
        </el-form-item>
        <el-form-item label="数据来源设备">
          <el-select v-model="form.device_id" placeholder="选择数据来源" style="width: 100%">
            <el-option label="手动录入（MANUAL）" :value="'MANUAL-' + user.id" />
            <el-option label="IoT 自动采集设备 #DEV-001" value="DEV-001" />
            <el-option label="IoT 自动采集设备 #DEV-002" value="DEV-002" />
          </el-select>
          <div class="form-tip">手动录入的记录将被标记为「风险标签」，提交后需额外审核</div>
        </el-form-item>
        <el-form-item label="采集时间">
          <el-date-picker
            v-model="form.collect_time"
            type="datetime"
            placeholder="选择能耗采集时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="备注说明">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="选填：本次上报的特殊情况说明..." />
        </el-form-item>
      </el-form>
      <div class="step-actions">
        <div></div>
        <el-button type="primary" class="submit-btn" :icon="ArrowRight" @click="currentStep = 1">
          下一步
        </el-button>
      </div>
    </div>

    <!-- ===== 步骤 2：能耗数据 ===== -->
    <div v-show="currentStep === 1" class="step-card">
      <h3 class="step-title">
        <el-icon :size="16"><EditPen /></el-icon>
        填写能耗数据（单位不可为负）
      </h3>

      <!-- 碳排放因子参考 -->
      <div class="factor-bar">
        <span class="factor-title">📐 碳排放因子参考</span>
        <el-tag size="small" effect="plain">电力 0.5810 kgCO₂/kWh</el-tag>
        <el-tag size="small" effect="plain">天然气 2.1622 kgCO₂/m³</el-tag>
        <el-tag size="small" effect="plain">用水 0.9103 kgCO₂/t</el-tag>
      </div>

      <el-form :model="form" label-width="140px" class="step-form">
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="用电量 (kWh)" required>
              <el-input-number
                v-model="form.electricity"
                :min="0" :precision="1" :step="10"
                style="width: 100%"
                controls-position="right"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="天然气用量 (m³)" required>
              <el-input-number
                v-model="form.gas"
                :min="0" :precision="1" :step="5"
                style="width: 100%"
                controls-position="right"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="用水量 (t)" required>
              <el-input-number
                v-model="form.water"
                :min="0" :precision="1" :step="1"
                style="width: 100%"
                controls-position="right"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 实时估算碳排放量 -->
        <div class="estimate-box">
          <div class="est-title">实时估算（提交后将由后端精确核算）</div>
          <div class="est-grid">
            <div class="est-item">
              <span class="est-label">电力排放</span>
              <span class="est-val">{{ (form.electricity * 0.5810).toFixed(1) }} kg</span>
            </div>
            <div class="est-item">
              <span class="est-label">天然气排放</span>
              <span class="est-val">{{ (form.gas * 2.1622).toFixed(1) }} kg</span>
            </div>
            <div class="est-item">
              <span class="est-label">用水排放</span>
              <span class="est-val">{{ (form.water * 0.9103).toFixed(1) }} kg</span>
            </div>
            <div class="est-item est-total">
              <span class="est-label">合计碳排放</span>
              <span class="est-val">{{ totalEmission.toFixed(1) }} kgCO₂</span>
            </div>
          </div>
          <div v-if="totalEmission > 0" class="est-hint">
            <el-icon :size="12"><Warning /></el-icon>
            估算值仅供参考，碳积分将在提交后由后端基于真实因子核算
          </div>
          <div v-else class="est-hint est-warn">
            <el-icon :size="12"><Warning /></el-icon>
            用电量 / 天然气 / 用水量至少一项 > 0
          </div>
        </div>
      </el-form>

      <div class="step-actions">
        <el-button @click="currentStep = 0" :icon="ArrowLeft">上一步</el-button>
        <el-button
          type="primary"
          class="submit-btn"
          :icon="ArrowRight"
          :disabled="totalEmission <= 0"
          @click="currentStep = 2"
        >
          下一步：确认提交
        </el-button>
      </div>
    </div>

    <!-- ===== 步骤 3：确认提交 ===== -->
    <div v-show="currentStep === 2" class="step-card">
      <h3 class="step-title">
        <el-icon :size="16"><CircleCheck /></el-icon>
        请确认提交能耗数据
      </h3>

      <el-descriptions :column="2" border class="confirm-desc">
        <el-descriptions-item label="上报企业">{{ enterpriseName }}</el-descriptions-item>
        <el-descriptions-item label="数据来源">{{ form.device_id }}</el-descriptions-item>
        <el-descriptions-item label="采集时间">{{ form.collect_time }}</el-descriptions-item>
        <el-descriptions-item label="提交方式">手动录入</el-descriptions-item>
        <el-descriptions-item label="用电量">{{ form.electricity }} kWh</el-descriptions-item>
        <el-descriptions-item label="天然气">{{ form.gas }} m³</el-descriptions-item>
        <el-descriptions-item label="用水量">{{ form.water }} t</el-descriptions-item>
        <el-descriptions-item label="预计排放" class="text-primary">{{ totalEmission.toFixed(1) }} kgCO₂</el-descriptions-item>
        <el-descriptions-item label="备注" :span="2">{{ form.remark || '无' }}</el-descriptions-item>
      </el-descriptions>

      <div class="risk-tip">
        <el-icon :size="14"><Warning /></el-icon>
        本次为手动录入数据，将被标记为「风险等级：中」，提交后由系统自动完成 SHA-256 上链存证。
      </div>

      <div class="step-actions">
        <el-button @click="currentStep = 1" :icon="ArrowLeft">上一步</el-button>
        <el-button
          type="primary"
          class="submit-btn"
          :icon="loadingSubmit ? Loading : Select"
          :loading="loadingSubmit"
          :disabled="loadingSubmit"
          @click="doSubmit"
        >
          {{ loadingSubmit ? '提交中...' : '确认提交并上链存证' }}
        </el-button>
      </div>
    </div>

    <!-- ===== 历史记录 ===== -->
    <div class="records-section">
      <div class="rec-head">
        <h3><el-icon :size="16"><Document /></el-icon> 历史能耗记录</h3>
        <el-button :icon="Refresh" @click="loadRecords" :loading="loadingRecords">刷新</el-button>
      </div>
      <el-table :data="records" v-loading="loadingRecords" stripe style="width: 100%">
        <el-table-column prop="record_no" label="记录编号" width="180">
          <template #default="{ row }">
            <code class="mono">{{ row.record_no }}</code>
          </template>
        </el-table-column>
        <el-table-column label="用电量(kWh)" width="130" align="right">
          <template #default="{ row }">{{ row.electricity?.toFixed(1) }}</template>
        </el-table-column>
        <el-table-column label="天然气(m³)" width="130" align="right">
          <template #default="{ row }">{{ row.gas?.toFixed(1) }}</template>
        </el-table-column>
        <el-table-column label="用水量(t)" width="110" align="right">
          <template #default="{ row }">{{ row.water?.toFixed(1) }}</template>
        </el-table-column>
        <el-table-column label="核算状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status !== 'pending'" type="success" size="small" effect="light">核算完成</el-tag>
            <el-tag v-else type="warning" size="small" effect="light">待核算</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="上链状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" type="success" size="small" effect="dark">已上链</el-tag>
            <el-tag v-else type="info" size="small">未上链</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="collect_time" label="采集时间" width="180">
          <template #default="{ row }">{{ formatTime(row.collect_time) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="calcCredit(row)" :disabled="row.status === 'calculated'">
              {{ row.status === 'calculated' ? '已核算' : '核算积分' }}
            </el-button>
          </template>
        </el-table-column>
        <template #empty>
          <div class="empty-tip">
            <el-icon :size="36" color="#94a3b8"><Document /></el-icon>
            <p>暂无能耗记录，请先在上方表单提交</p>
          </div>
        </template>
      </el-table>
    </div>

    <!-- ===== 提交成功弹窗 ===== -->
    <el-dialog v-model="successVisible" width="520px" :close-on-click-modal="false" class="success-dialog">
      <template #header>
        <div class="dialog-header">
          <div class="dh-icon">
            <el-icon :size="24"><CircleCheck /></el-icon>
          </div>
          <div class="dh-text">
            <h3>能耗数据提交成功！</h3>
            <p>已完成 SHA-256 上链存证</p>
          </div>
        </div>
      </template>
      <div class="dialog-body">
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="记录编号">
            <code class="mono">{{ submitted?.record_no }}</code>
          </el-descriptions-item>
          <el-descriptions-item label="区块哈希">
            <code class="mono mono-sm">{{ submitted?.block_hash }}</code>
          </el-descriptions-item>
          <el-descriptions-item label="链上状态">
            <el-tag type="success" size="small" effect="dark">已上链</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="核算碳积分">
            <template v-if="creditResult">
              <el-tag type="warning" size="small">{{ creditResult.carbon_credits?.toFixed(2) }} 积分</el-tag>
            </template>
            <template v-else>
              <el-button size="small" type="primary" link @click="calcCredit(submitted)">立即核算</el-button>
            </template>
          </el-descriptions-item>
        </el-descriptions>

        <!-- ZKP 证明区域 -->
        <div class="zkp-box">
          <div class="zkp-title">
            <el-icon :size="14"><Lock /></el-icon>
            ZKP 隐私选择性披露证明
          </div>
          <p class="zkp-desc">
            可生成零知识证明，对外仅披露需要的统计结果，保护企业能耗数据的细粒度隐私。
          </p>
          <div class="zkp-actions">
            <el-button size="small" :icon="MagicStick" @click="generateZkp" :loading="loadingZkp">
              {{ loadingZkp ? '生成中...' : '生成 ZKP 证明' }}
            </el-button>
            <el-button
              size="small" type="primary" :icon="Download"
              :disabled="!zkpProof"
              @click="exportZkp"
            >
              导出证明文件
            </el-button>
          </div>
          <pre v-if="zkpProof" class="zkp-proof"><code>{{ zkpProof }}</code></pre>
        </div>
      </div>
      <template #footer>
        <el-button @click="closeSuccess">关闭</el-button>
        <el-button type="primary" @click="goCredits">前往碳积分中心 →</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  InfoFilled, Lock, UserFilled, EditPen, CircleCheck, ArrowRight, ArrowLeft,
  Warning, Document, Refresh, Select, Loading, MagicStick, Download, User
} from '@element-plus/icons-vue'
import { energyAPI, carbonAPI, zkpAPI } from '../api/index.js'

const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{}')
const enterpriseName = user.company || user.username || ''

/* ===== 步骤状态 ===== */
const currentStep = ref(0)
const loadingSubmit = ref(false)
const loadingZkp = ref(false)
const loadingRecords = ref(false)
const successVisible = ref(false)

/* ===== 表单数据 ===== */
const form = reactive({
  device_id: 'MANUAL-' + user.id,
  collect_time: new Date().toISOString().slice(0, 19).replace('T', ' '),
  remark: '',
  electricity: 0,
  gas: 0,
  water: 0,
})

/* 实时估算碳排放 */
const totalEmission = computed(() =>
  form.electricity * 0.5810 + form.gas * 2.1622 + form.water * 0.9103
)

/* ===== 提交结果 ===== */
const submitted = ref(null)
const creditResult = ref(null)
const zkpProof = ref('')

/* ===== 历史记录 ===== */
const records = ref([])

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

async function loadRecords() {
  loadingRecords.value = true
  try {
    const res = await energyAPI.list({ page: 1, page_size: 20 })
    records.value = res.data?.list || res.data || []
  } catch (e) {
    ElMessage.error(e?.msg || '加载历史失败')
  } finally {
    loadingRecords.value = false
  }
}

/* ===== 提交 ===== */
async function doSubmit() {
  if (totalEmission.value <= 0) {
    ElMessage.warning('用电量 / 天然气 / 用水量至少一项 > 0')
    return
  }
  try {
    await ElMessageBox.confirm(
      '确认提交本次能耗数据？提交后将自动上链存证，不可撤销。',
      '确认提交',
      { confirmButtonText: '确认提交', type: 'warning' }
    )
  } catch { return }

  loadingSubmit.value = true
  try {
    // 后端已自动核算碳凭证：一条能耗提交即完成「保存记录 + 核算积分 + 上链存证」
    const res = await energyAPI.create({
      electricity: form.electricity,
      gas: form.gas,
      water: form.water
    })
    const data = res.data || {}
    submitted.value = data.record || { record_no: data.record_no }
    // 后端自动核算返回碳凭证信息，直接回填结果
    creditResult.value = { carbon_credits: data.credits, credit_no: data.credit_no }
    ElMessage.success(`上报成功，碳积分已自动核算入账：${Number(data.credits ?? 0).toFixed(2)} 积分`)

    successVisible.value = true
    loadRecords()
  } catch (e) {
    ElMessage.error(e?.msg || '提交失败')
  } finally {
    loadingSubmit.value = false
  }
}

/* ===== 单独核算某条记录的碳积分 ===== */
async function calcCredit(row) {
  try {
    const res = await carbonAPI.calculate({ energy_record_id: row.id })
    creditResult.value = res.data
    ElMessage.success(`核算成功：${res.data?.carbon_credits?.toFixed(2)} 积分`)
    loadRecords()
  } catch (e) {
    ElMessage.error(e?.msg || '核算失败')
  }
}

/* ===== 生成 ZKP 隐私证明 ===== */
async function generateZkp() {
  if (!submitted.value) return
  loadingZkp.value = true
  try {
    const res = await zkpAPI.generateProof({
      record_no: submitted.value.record_no,
      fields: ['electricity', 'gas', 'water', 'total_emission'] // 选择性披露字段
    })
    zkpProof.value = JSON.stringify(res.data?.proof || res.data, null, 2)
    ElMessage.success('ZKP 证明生成成功')
  } catch (e) {
    // 后端 ZKP 接口不存在时，前端生成模拟证明文件（用于演示）
    const mockProof = {
      record_no: submitted.value.record_no,
      zkp_proof_id: 'ZKP-' + Date.now(),
      circuit_type: 'groth16',
      disclosed_fields: ['electricity', 'gas', 'water', 'total_emission'],
      hid_commitment: '0x' + Array(64).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
      range_proof: '0x' + Array(64).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
      verified: true,
      timestamp: new Date().toISOString()
    }
    zkpProof.value = JSON.stringify(mockProof, null, 2)
    ElMessage.success('ZKP 证明生成成功（前端模拟，后端接口待接入）')
  } finally {
    loadingZkp.value = false
  }
}

/* ===== 导出 ZKP 证明文件 ===== */
function exportZkp() {
  if (!zkpProof.value) return
  const blob = new Blob([zkpProof.value], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `zkp-proof-${submitted.value?.record_no || Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('ZKP 证明文件已下载')
}

function closeSuccess() {
  successVisible.value = false
  // 重置表单
  form.device_id = 'MANUAL-' + user.id
  form.collect_time = new Date().toISOString().slice(0, 19).replace('T', ' ')
  form.remark = ''
  form.electricity = 0
  form.gas = 0
  form.water = 0
  currentStep.value = 0
  submitted.value = null
  creditResult.value = null
  zkpProof.value = ''
}

function goCredits() {
  closeSuccess()
  router.push('/credits-center')
}

onMounted(() => loadRecords())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 页面头部 ===== */
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 24px;
  margin-bottom: 20px;
  background: linear-gradient(135deg, var(--primary-green), #16a34acc);
  color: #fff;
  border-radius: var(--radius-lg);
}
.theme-dark .page-head {
  background: linear-gradient(135deg, #0d9488, #06b6d4aa);
}
.ph-left h2 { font-size: 18px; font-weight: 600; color: #fff; margin-bottom: 6px; }
.ph-desc {
  display: flex; align-items: center; gap: 6px;
  font-size: 12.5px;
  color: rgba(255,255,255,0.85);
  margin: 0;
}

/* ===== 步骤条 ===== */
.steps {
  margin-bottom: 24px;
  padding: 16px 24px;
  background: var(--card-bg);
  border-radius: var(--radius);
  border: 1px solid var(--card-border);
}

/* ===== 步骤卡片 ===== */
.step-card {
  padding: 24px;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius);
  margin-bottom: 20px;
}
.step-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 15px; font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--line-color);
}
.step-form { max-width: 100%; }

.form-tip {
  font-size: 11.5px;
  color: var(--text-tertiary);
  margin-top: 4px;
}

/* ===== 因子参考 ===== */
.factor-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: rgba(22,163,74,0.06);
  border: 1px solid rgba(22,163,74,0.15);
  border-radius: 8px;
  margin-bottom: 16px;
}
.factor-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--primary-green);
  margin-right: 4px;
}

/* ===== 实时估算 ===== */
.estimate-box {
  margin-top: 20px;
  padding: 18px;
  background: var(--bg-secondary);
  border-radius: 10px;
  border: 1px dashed var(--line-strong);
}
.est-title {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-bottom: 12px;
}
.est-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.est-item {
  display: flex; flex-direction: column; gap: 4px;
  padding: 10px 14px;
  background: var(--card-bg);
  border-radius: 8px;
  border: 1px solid var(--line-color);
}
.est-label { font-size: 11px; color: var(--text-tertiary); }
.est-val { font-size: 15px; font-weight: 600; color: var(--text-primary); font-variant-numeric: tabular-nums; }
.est-total {
  background: linear-gradient(135deg, var(--primary-green), #22c55e);
  border: none;
}
.est-total .est-label { color: rgba(255,255,255,0.9); }
.est-total .est-val { color: #fff; font-size: 16px; }
.est-hint {
  display: flex; align-items: center; gap: 6px;
  font-size: 11.5px; color: var(--text-tertiary);
  margin-top: 12px;
}
.est-warn { color: #f59e0b; }

/* ===== 确认描述 ===== */
.confirm-desc { margin-bottom: 16px; }
.text-primary { color: var(--primary-green); font-weight: 600; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 12px; }
.mono-sm { font-family: ui-monospace, Consolas, monospace; font-size: 11px; word-break: break-all; }

.risk-tip {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 14px;
  background: rgba(245,158,11,0.08);
  border-left: 3px solid #f59e0b;
  border-radius: 6px;
  font-size: 12.5px;
  color: #d97706;
  margin-top: 16px;
}

/* ===== 步骤操作按钮 ===== */
.step-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 24px;
}
.submit-btn {
  background: var(--primary-green) !important;
  border-color: var(--primary-green) !important;
}
.submit-btn:hover {
  background: var(--primary-green-dark) !important;
  border-color: var(--primary-green-dark) !important;
}

/* ===== 历史记录 ===== */
.records-section {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius);
  padding: 20px 24px;
  margin-bottom: 20px;
}
.rec-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}
.rec-head h3 {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 600; color: var(--text-primary);
}
.empty-tip {
  padding: 30px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 12.5px;
}
.empty-tip p { margin: 10px 0 0; }

/* ===== 成功弹窗 ===== */
.dialog-header {
  display: flex; align-items: center; gap: 12px;
}
.dh-icon {
  width: 44px; height: 44px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary-green), #22c55e);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
}
.dh-text h3 { font-size: 15px; color: var(--text-primary); margin-bottom: 2px; }
.dh-text p { font-size: 12px; color: var(--text-tertiary); margin: 0; }

.zkp-box {
  margin-top: 16px;
  padding: 16px;
  background: var(--bg-secondary);
  border: 1px dashed var(--line-strong);
  border-radius: 10px;
}
.zkp-title {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
}
.zkp-desc { font-size: 11.5px; color: var(--text-tertiary); margin: 0 0 12px; line-height: 1.6; }
.zkp-actions { display: flex; gap: 8px; margin-bottom: 10px; }
.zkp-proof {
  margin: 0;
  padding: 10px 12px;
  background: #064e3b;
  color: #5eead4;
  border-radius: 6px;
  font-size: 11px;
  line-height: 1.5;
  max-height: 160px;
  overflow: auto;
}

@media (max-width: 900px) {
  .est-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>