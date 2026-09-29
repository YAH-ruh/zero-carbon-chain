<!--
  incentive.vue - 链上激励池管理（exchange / park_admin 角色）
  功能：
    1. 激励池资金总览（池余额 / 已分发 / 剩余 / 分配机制）
    2. 发放激励（按企业 / 按减排量 / 按积分持有量）
    3. 激励发放记录表格
    4. 激励池贡献来源（政府补贴 / 企业基金 / 碳交易手续费）
    5. 链上证明
-->
<template>
  <div class="ip-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>
          <el-icon :size="18"><Medal /></el-icon>
          链上激励池管理
        </h2>
        <p>绿色低碳激励资金池 · 链上透明分配 · 企业减排行为正向激励</p>
      </div>
      <el-tag effect="dark" size="small" type="success">DAO 治理地址: 0x...8899</el-tag>
    </div>

    <!-- ===== 激励池资金卡 ===== -->
    <div class="pool-card">
      <div class="pc-main">
        <div class="pc-label">激励池总余额</div>
        <div class="pc-amount">¥{{ pool.total.toLocaleString() }}</div>
        <div class="pc-sub">
          <span>已分发 <b class="green">¥{{ pool.distributed.toLocaleString() }}</b></span>
          <span>剩余 <b class="warn">¥{{ pool.remaining.toLocaleString() }}</b></span>
        </div>
        <div class="pc-progress">
          <el-progress
            :percentage="poolProgress"
            :stroke-width="10"
            :show-text="false"
            :color="poolProgress > 80 ? '#ef4444' : '#10b981'"
          />
          <span class="pp-text">已分发 {{ poolProgress }}%</span>
        </div>
      </div>
      <div class="pc-sources">
        <div class="ps-title">📊 资金来源构成</div>
        <div class="ps-item" v-for="s in sources" :key="s.name">
          <div class="ps-bar" :style="{ background: s.color, width: (s.value / pool.total * 100) + '%' }"></div>
          <div class="ps-info">
            <div class="ps-name">{{ s.name }}</div>
            <div class="ps-val">¥{{ s.value.toLocaleString() }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 发放激励 ===== -->
    <div class="action-card">
      <div class="ac-head">
        <h3>🎯 发放激励</h3>
        <el-tag effect="dark" size="small">按减排贡献自动分配</el-tag>
      </div>
      <el-form :inline="true" @submit.prevent="doIssue">
        <el-form-item label="激励类型">
          <el-select v-model="issueForm.type" style="width:140px">
            <el-option label="按减排量" value="emission" />
            <el-option label="按积分持有量" value="credits" />
            <el-option label="手动指定" value="manual" />
          </el-select>
        </el-form-item>
        <el-form-item label="发放企业">
          <el-select v-model="issueForm.enterprise_id" style="width:180px" filterable placeholder="选择企业">
            <el-option v-for="e in enterprises" :key="e.id" :label="e.company" :value="e.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="金额(元)">
          <el-input-number v-model="issueForm.amount" :min="100" :max="pool.remaining" style="width:150px" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="issueForm.reason" style="width:200px" placeholder="如: Q2 减排明星企业" />
        </el-form-item>
        <el-button type="primary" :icon="Medal" :loading="issueLoading" @click="doIssue">
          发放并上链存证
        </el-button>
      </el-form>
    </div>

    <!-- ===== 发放记录 ===== -->
    <div class="list-card">
      <div class="lc-head">
        <h3>激励发放记录</h3>
        <el-button :icon="Refresh" size="small" @click="loadRecords" :loading="loadingList">刷新</el-button>
      </div>

      <el-table :data="records" stripe size="small" style="width:100%" empty-text="暂无发放记录">
        <el-table-column label="编号" width="170">
          <template #default="{ row }"><code class="mono">{{ row.incentive_no }}</code></template>
        </el-table-column>
        <el-table-column label="企业" width="160">
          <template #default="{ row }">{{ row.company }}</template>
        </el-table-column>
        <el-table-column label="类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" :class="'it-' + row.type">{{ issueTypeMap[row.type] || row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="金额(元)" width="120" align="right">
          <template #default="{ row }"><b class="green">¥{{ row.amount.toLocaleString() }}</b></template>
        </el-table-column>
        <el-table-column prop="reason" label="说明" show-overflow-tooltip />
        <el-table-column label="上链" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" type="success" size="small">已上链</el-tag>
            <el-tag v-else type="info" size="small">--</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Medal, Refresh } from '@element-plus/icons-vue'

const issueTypeMap = { emission: '按减排量', credits: '按积分量', manual: '手动指定' }

/* ===== 状态 ===== */
const pool = reactive({ total: 5_000_000, distributed: 1_850_000, remaining: 3_150_000 })
const sources = ref([
  { name: '政府补贴（财政）', value: 2_000_000, color: '#0891b2' },
  { name: '碳交易所手续费',  value: 1_800_000, color: '#10b981' },
  { name: '企业自愿基金',    value: 1_000_000, color: '#f59e0b' },
  { name: 'DAO 社区捐赠',    value: 200_000,  color: '#7c3aed' },
])
const enterprises = ref([
  { id: 1, company: '绿恒节能科技有限公司' },
  { id: 2, company: '低碳智造有限公司' },
  { id: 3, company: '星辉电子科技' },
  { id: 4, company: '清源化工集团' },
  { id: 5, company: '蓝天包装股份' },
])
const records = ref([])
const loadingList = ref(false)
const issueLoading = ref(false)

const issueForm = reactive({ type: 'emission', enterprise_id: 1, amount: 50000, reason: '' })

const poolProgress = computed(() => Math.round(pool.distributed / pool.total * 100))

/* ===== 加载 ===== */
async function loadRecords() {
  loadingList.value = true
  // 模拟后端 /api/incentive/list
  records.value = Array.from({ length: 6 }, (_, i) => ({
    id: i + 1,
    incentive_no: 'INC-202609-' + String(1000 + i),
    company: enterprises.value[i % enterprises.value.length].company,
    type: ['emission','credits','manual','emission','credits','manual'][i],
    amount: [50000, 35000, 20000, 80000, 42000, 15000][i],
    reason: ['Q2 减排明星','积分持有贡献','重点扶持企业','碳中和达标奖励','交易活跃贡献','临时激励'][i],
    on_chain: true,
    created_at: new Date(Date.now() - i * 86400000 * 3).toISOString(),
  }))
  loadingList.value = false
}

/* ===== 发放 ===== */
async function doIssue() {
  if (!issueForm.amount || issueForm.amount < 100) { ElMessage.warning('请填写有效金额'); return }
  if (issueForm.amount > pool.remaining) { ElMessage.warning('金额超过激励池剩余余额'); return }
  issueLoading.value = true
  await new Promise(r => setTimeout(r, 800))

  const ent = enterprises.value.find(e => e.id === issueForm.enterprise_id)
  pool.distributed += issueForm.amount
  pool.remaining -= issueForm.amount

  records.value.unshift({
    id: records.value.length + 1,
    incentive_no: 'INC-' + Date.now().toString().slice(-8),
    company: ent?.company || '未知企业',
    type: issueForm.type,
    amount: issueForm.amount,
    reason: issueForm.reason || '手动激励',
    on_chain: true,
    created_at: new Date().toISOString(),
  })

  issueLoading.value = false
  ElMessage.success(`激励发放成功！¥${issueForm.amount.toLocaleString()} 已上链存证`)
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

onMounted(() => loadRecords())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #d97706, #16a34a);
  border-radius: var(--radius-lg); color: #fff;
}
.page-head h2 { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.page-head p { font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }

/* ===== 激励池卡 ===== */
.pool-card {
  display: grid; grid-template-columns: 1.5fr 1fr;
  gap: 24px; padding: 28px 32px;
  background: linear-gradient(135deg, #fefce8, #f0fdf4);
  border-radius: 14px;
  border: 1px solid #fde68a;
  margin-bottom: 16px;
}
.pc-main { display: flex; flex-direction: column; justify-content: center; }
.pc-label { font-size: 13px; color: #a16207; margin-bottom: 6px; }
.pc-amount {
  font-size: 38px; font-weight: 800; font-variant-numeric: tabular-nums;
  background: linear-gradient(135deg, #d97706, #16a34a);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 8px;
}
.pc-sub { display: flex; gap: 16px; font-size: 12px; color: #64748b; margin-bottom: 16px; }
.green { color: #10b981; }
.warn { color: #f59e0b; }
.pc-progress { display: flex; align-items: center; gap: 12px; }
.pc-progress :deep(.el-progress-bar__outer) { background: rgba(0,0,0,0.08); }
.pp-text { font-size: 11px; color: #64748b; white-space: nowrap; }

.pc-sources {
  background: rgba(255,255,255,0.7);
  padding: 14px 18px; border-radius: 10px;
}
.ps-title { font-size: 12px; font-weight: 600; color: #78350f; margin-bottom: 12px; }
.ps-item {
  display: grid; grid-template-columns: 120px 1fr 100px;
  gap: 8px; align-items: center; margin-bottom: 8px;
}
.ps-bar { height: 10px; border-radius: 3px; min-width: 4px; }
.ps-name { font-size: 11px; color: #374151; }
.ps-val { font-size: 11px; color: #78350f; font-weight: 600; text-align: right; font-variant-numeric: tabular-nums; }

/* ===== 发放 ===== */
.action-card, .list-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.ac-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.ac-head h3 { font-size: 14px; font-weight: 600; margin: 0; }

/* ===== 列表 ===== */
.lc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.lc-head h3 { font-size: 14px; font-weight: 600; margin: 0; }

.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }

.it-emission { background: rgba(16,185,129,0.1); color: #10b981; }
.it-credits  { background: rgba(8,145,178,0.1); color: #0891b2; }
.it-manual   { background: rgba(245,158,11,0.1); color: #d97706; }

@media (max-width: 900px) {
  .pool-card { grid-template-columns: 1fr; }
}
</style>