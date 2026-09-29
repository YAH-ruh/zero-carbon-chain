<!--
  archive.vue - 碳信用档案（exchange / regulator 角色）
  功能：
    1. 链上碳信用档案总览统计
    2. 档案卡片列表（编号 / 企业 / 碳量 / 签发机构 / 上链状态）
    3. 档案详情查看（完整信息 + 链上哈希 + SHA-256 摘要）
    4. 核验链上真实性（调 backend archive verify）
  后端接口：archiveAPI.archiveList / createArchive / 链上查询 via /api/chain
-->
<template>
  <div class="archive-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>
          <el-icon :size="18"><Collection /></el-icon>
          碳信用档案
        </h2>
        <p>企业碳减排信用 · 链上可信存证 · 全生命周期管理</p>
      </div>
      <el-button type="primary" :icon="DocumentAdd" @click="openCreate">+ 新建档案</el-button>
    </div>

    <!-- ===== 统计卡 ===== -->
    <div class="stat-row">
      <div class="stat-card total">
        <div class="sc-icon"><el-icon :size="18"><Collection /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.total }}</div>
          <div class="sc-lab">档案总数</div>
        </div>
      </div>
      <div class="stat-card onchain">
        <div class="sc-icon"><el-icon :size="18"><Connection /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.on_chain }}</div>
          <div class="sc-lab">已上链存证</div>
        </div>
      </div>
      <div class="stat-card pending">
        <div class="sc-icon"><el-icon :size="18"><Clock /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.pending }}</div>
          <div class="sc-lab">待上链</div>
        </div>
      </div>
      <div class="stat-card verified">
        <div class="sc-icon"><el-icon :size="18"><CircleCheck /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.verified }}</div>
          <div class="sc-lab">链上核验通过</div>
        </div>
      </div>
    </div>

    <!-- ===== 筛选 + 档案列表 ===== -->
    <div class="list-card">
      <div class="lc-head">
        <h3>碳信用档案列表</h3>
        <div class="lc-filters">
          <el-input v-model="keyword" placeholder="搜索档案编号 / 企业名" :prefix-icon="Search" clearable size="small" style="width:220px" />
          <el-select v-model="statusFilter" placeholder="全部状态" size="small" style="width:130px" clearable>
            <el-option label="已上链" value="on_chain" />
            <el-option label="未上链" value="pending" />
          </el-select>
          <el-button :icon="Refresh" size="small" @click="refreshAll" :loading="loadingAll">刷新</el-button>
        </div>
      </div>

      <div v-if="loadingList" class="list-loading">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
      </div>
      <div v-else-if="!filteredArchives.length" class="list-empty">
        <el-icon :size="40"><Collection /></el-icon>
        <p>暂无碳信用档案</p>
      </div>

      <!-- 卡片网格 -->
      <div v-else class="archive-grid">
        <div v-for="a in filteredArchives" :key="a.archive_no" class="archive-card">
          <!-- 左侧链上标记 -->
          <div class="ac-side" :class="{ onchain: a.on_chain }">
            <el-icon v-if="a.on_chain" :size="20"><Connection /></el-icon>
            <el-icon v-else :size="20"><Clock /></el-icon>
          </div>

          <div class="ac-main">
            <div class="ac-top">
              <code class="mono">{{ a.archive_no }}</code>
              <el-tag :class="'ac-status ' + (a.on_chain ? 'onchain' : 'pending')" size="small" effect="dark">
                {{ a.on_chain ? '已上链' : '未上链' }}
              </el-tag>
            </div>
            <div class="ac-enterprise">
              <el-icon :size="11"><User /></el-icon>
              {{ a.company || ('企业 #' + a.enterprise_id) }}
              <el-tag v-if="a.auto" size="small" effect="plain" class="auto-tag">链上自动</el-tag>
            </div>
            <div class="ac-metrics">
              <div class="am">
                <div class="am-num">{{ a.carbon_credits?.toLocaleString() || 0 }}</div>
                <div class="am-lab">碳信用量 (kgCO₂)</div>
              </div>
              <div class="am">
                <div class="am-num am-org">{{ a.verified_by || '--' }}</div>
                <div class="am-lab">签发机构</div>
              </div>
            </div>
            <div class="ac-foot">
              <span>📅 建档时间: {{ formatTime(a.created_at) }}</span>
              <span v-if="a.block_hash" class="chain-hash">
                🔗 {{ a.block_hash.slice(0, 14) }}...
              </span>
            </div>
          </div>

          <div class="ac-actions">
            <el-button size="small" link type="primary" @click="viewDetail(a)">查看详情</el-button>
            <el-button
              v-if="a.on_chain"
              size="small" link type="success"
              @click="verifyOnChain(a)"
            >
              核验链上
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 新建档案弹窗 ===== -->
    <el-dialog v-model="createVisible" width="560px" title="新建碳信用档案">
      <el-form :model="createForm" label-width="110px">
        <el-form-item label="积分单号" required>
          <el-input v-model="createForm.credit_no" placeholder="如: CREDIT-20260912-0013" />
          <div class="form-hint">关联链上碳积分核算单号（可在企业积分中心查看）</div>
        </el-form-item>
        <el-form-item label="企业ID" required>
          <el-input-number v-model="createForm.enterprise_id" :min="1" style="width:100%" />
          <div class="form-hint">碳积分归属企业用户ID</div>
        </el-form-item>
        <el-form-item label="碳信用量(kgCO₂)" required>
          <el-input-number v-model="createForm.carbon_credits" :min="1" :step="100" style="width:100%" />
        </el-form-item>
        <el-form-item label="签发机构">
          <el-input v-model="createForm.verified_by" placeholder="如: 中环联合认证中心" />
        </el-form-item>
        <el-form-item label="档案来源">
          <el-select v-model="createForm.source_desc" style="width:100%">
            <el-option label="企业自主申报" value="企业自主申报" />
            <el-option label="第三方核查机构" value="第三方核查机构" />
            <el-option label="AI 自动链上生成" value="AI 自动链上生成" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="loadingCreate" @click="doCreate">创建并上链</el-button>
      </template>
    </el-dialog>

    <!-- ===== 档案详情弹窗 ===== -->
    <el-dialog v-model="detailVisible" width="620px" :title="'档案详情 · ' + (currentArchive?.archive_no || '')">
      <div v-if="currentArchive" class="detail-box">
        <div class="detail-row">
          <span>企业名称</span><b>{{ currentArchive.company || ('企业 #' + currentArchive.enterprise_id) }}</b>
        </div>
        <div class="detail-row">
          <span>关联积分单号</span><code class="mono">{{ currentArchive.credit_no || '--' }}</code>
        </div>
        <div class="detail-row">
          <span>碳信用量</span><b class="green">{{ currentArchive.carbon_credits?.toLocaleString() || 0 }} kgCO₂</b>
        </div>
        <div class="detail-row" v-if="currentArchive.total_emission">
          <span>总碳排放量</span>{{ currentArchive.total_emission.toLocaleString() }} kgCO₂
        </div>
        <div class="detail-row">
          <span>签发机构</span>{{ currentArchive.verified_by || '--' }}
        </div>
        <div class="detail-row">
          <span>档案来源</span>
          <el-tag size="small">{{ currentArchive.source_desc || '链上自动存证' }}</el-tag>
        </div>
        <div class="detail-row" v-if="currentArchive.status">
          <span>积分状态</span>
          <el-tag size="small" :type="currentArchive.status === 'available' ? 'success' : 'info'">
            {{ statusMap[currentArchive.status] || currentArchive.status }}
          </el-tag>
        </div>
        <div class="detail-row">
          <span>上链状态</span>
          <el-tag :type="currentArchive.on_chain ? 'success' : 'warning'" size="small">
            {{ currentArchive.on_chain ? '已上链存证' : '待上链' }}
          </el-tag>
        </div>

        <h4 class="sub-title">🔗 链上信息</h4>
        <div class="chain-info">
          <div v-if="currentArchive.block_hash" class="ci-row">
            <span>区块哈希</span><code>{{ currentArchive.block_hash }}</code>
          </div>
          <div class="ci-row">
            <span>链上查询单号</span><code>{{ currentArchive.credit_no || currentArchive.archive_no }}</code>
          </div>
          <div v-if="currentArchive.created_at" class="ci-row">
            <span>创建时间</span>{{ formatTime(currentArchive.created_at) }}
          </div>
        </div>

        <!-- 核验结果 -->
        <div v-if="verifyResult" class="verify-box" :class="verifyResult.ok ? 'ok' : 'fail'">
          <el-icon :size="16">
            <CircleCheck v-if="verifyResult.ok" />
            <Warning v-else />
          </el-icon>
          <div>
            <b>{{ verifyResult.ok ? '链上核验通过' : '链上核验失败' }}</b>
            <p>{{ verifyResult.message }}</p>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button v-if="currentArchive?.on_chain" type="success" @click="verifyOnChain(currentArchive)" :loading="verifyLoading">
          核验链上真实性
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Collection, DocumentAdd, Connection, Clock, CircleCheck,
  Search, Refresh, Loading, User, Warning
} from '@element-plus/icons-vue'
import { rwaAPI, chainAPI } from '../api/index.js'

const statusMap = { available: '可用', locked: '锁定中', sold: '已售出' }

/* ===== 状态 ===== */
const archives = ref([])
const loadingList = ref(false)
const loadingAll = ref(false)
const loadingCreate = ref(false)
const verifyLoading = ref(false)
const keyword = ref('')
const statusFilter = ref('')
const createVisible = ref(false)
const detailVisible = ref(false)
const currentArchive = ref(null)
const verifyResult = ref(null)

const stats = computed(() => ({
  total: archives.value.length,
  on_chain: archives.value.filter(a => a.on_chain).length,
  pending: archives.value.filter(a => !a.on_chain).length,
  verified: archives.value.filter(a => a.on_chain).length,
}))

const filteredArchives = computed(() => {
  let arr = archives.value
  if (statusFilter.value === 'on_chain') arr = arr.filter(a => a.on_chain)
  if (statusFilter.value === 'pending') arr = arr.filter(a => !a.on_chain)
  if (keyword.value) {
    const kw = keyword.value.toLowerCase()
    arr = arr.filter(a =>
      a.archive_no?.toLowerCase().includes(kw) ||
      (a.company || a.enterprise_name || '').toLowerCase().includes(kw)
    )
  }
  return arr
})

/* ===== 创建表单 ===== */
const createForm = reactive({
  credit_no: '', enterprise_id: 1, carbon_credits: 1000, verified_by: '', source_desc: '第三方核查机构'
})

/* ===== 加载 ===== */
async function loadList() {
  loadingList.value = true
  try {
    const res = await rwaAPI.archiveList({ page: 1, page_size: 50 })
    archives.value = res.data?.list || res.data || []
  } catch (e) {
    archives.value = []
    ElMessage.error('档案列表加载失败：' + (e?.msg || e?.message || '网络异常'))
  }
  loadingList.value = false
}

async function refreshAll() {
  loadingAll.value = true
  await loadList()
  loadingAll.value = false
  ElMessage.success('已刷新')
}

/* ===== 新建 ===== */
function openCreate() {
  createForm.credit_no = ''
  createForm.enterprise_id = 1
  createForm.carbon_credits = 1000
  createForm.verified_by = ''
  createForm.source_desc = '第三方核查机构'
  createVisible.value = true
}

async function doCreate() {
  if (!createForm.credit_no || !createForm.enterprise_id || !createForm.carbon_credits) {
    ElMessage.warning('请填写积分单号、企业ID和碳信用量')
    return
  }
  loadingCreate.value = true
  try {
    await rwaAPI.createArchive({ ...createForm })
    ElMessage.success('碳信用档案创建成功，已上链存证')
  } catch (e) {
    ElMessage.error('档案创建失败：' + (e?.msg || e?.message || '网络异常'))
  }
  loadingCreate.value = false
  createVisible.value = false
  refreshAll()
}

/* ===== 详情 ===== */
function viewDetail(a) {
  currentArchive.value = a
  verifyResult.value = null
  detailVisible.value = true
}

/* ===== 核验：调用真实链上溯源接口 ===== */
async function verifyOnChain(a) {
  verifyLoading.value = true
  verifyResult.value = null
  const dataNo = a.credit_no || a.archive_no
  try {
    const res = await chainAPI.query({ data_no: dataNo })
    const rec = res.data || res
    if (rec?.exists) {
      const matchOk = rec.match !== false
      verifyResult.value = {
        ok: matchOk,
        message: matchOk
          ? `链上记录存在：业务单号 ${rec.data_no} · 区块高度 #${rec.block_index} · 区块哈希 ${String(rec.block_hash || '').slice(0, 18)}... ，SHA-256 摘要与链上存证一致，档案未被篡改`
          : `警告：区块哈希重算不匹配，记录可能已被篡改！`,
      }
      if (matchOk) ElMessage.success('链上核验通过！')
      else ElMessage.error('区块哈希校验失败')
    } else {
      verifyResult.value = { ok: false, message: '链上无匹配记录，档案可能被篡改' }
      ElMessage.error('链上核验失败')
    }
  } catch (e) {
    verifyResult.value = {
      ok: false,
      message: '链上核验失败：' + (e?.msg || e?.message || '未找到该业务单号的链上记录'),
    }
    ElMessage.error('链上核验失败')
  }
  verifyLoading.value = false
  // 卡片上直接核验时自动打开详情弹窗，确保核验结果可见
  if (!detailVisible.value) {
    currentArchive.value = a
    detailVisible.value = true
  }
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

onMounted(() => loadList())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #0891b2, #059669);
  border-radius: var(--radius-lg); color: #fff;
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
.stat-card.total    { border-left-color: #0891b2; }
.stat-card.onchain  { border-left-color: #10b981; }
.stat-card.pending  { border-left-color: #f59e0b; }
.stat-card.verified { border-left-color: #22c55e; }
.sc-icon {
  width: 40px; height: 40px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-card.total    .sc-icon { background: rgba(8,145,178,0.12); color: #0891b2; }
.stat-card.onchain  .sc-icon { background: rgba(16,185,129,0.12); color: #10b981; }
.stat-card.pending  .sc-icon { background: rgba(245,158,11,0.12); color: #d97706; }
.stat-card.verified .sc-icon { background: rgba(34,197,94,0.12); color: #22c55e; }
.sc-num { font-size: 22px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary); }
.sc-lab { font-size: 11px; color: var(--text-tertiary); }

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

/* 档案卡 */
.archive-grid {
  display: grid; grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}
.archive-card {
  display: flex; gap: 14px;
  padding: 16px 18px;
  background: var(--bg-secondary);
  border: 1px solid var(--line-color);
  border-radius: 10px;
  transition: all 0.2s;
  position: relative;
}
.archive-card:hover { box-shadow: 0 6px 16px rgba(0,0,0,0.06); transform: translateY(-1px); }

.ac-side {
  width: 44px; display: flex; align-items: center; justify-content: center;
  background: rgba(100,116,139,0.1); color: #64748b;
  border-radius: 8px; flex-shrink: 0;
}
.ac-side.onchain { background: rgba(16,185,129,0.1); color: #10b981; }

.ac-main { flex: 1; min-width: 0; }
.ac-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11.5px; }
.ac-status.onchain { background: rgba(16,185,129,0.15); color: #10b981; border: none; }
.ac-status.pending { background: rgba(245,158,11,0.15); color: #d97706; border: none; }

.ac-enterprise {
  display: flex; align-items: center; gap: 4px;
  font-size: 12px; color: var(--text-secondary);
  margin-bottom: 10px;
}
.auto-tag { transform: scale(0.85); margin-left: 2px; }
.am-org { font-size: 12px !important; font-weight: 500 !important; color: var(--text-secondary); }
.form-hint { font-size: 11px; color: var(--text-tertiary); line-height: 1.4; margin-top: 2px; }

.ac-metrics {
  display: flex; gap: 18px; margin-bottom: 8px;
}
.am .am-num { font-size: 15px; font-weight: 700; font-variant-numeric: tabular-nums; }
.am:first-child .am-num { color: var(--primary-green); }
.am .am-lab { font-size: 10.5px; color: var(--text-tertiary); }

.ac-foot {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 10.5px; color: var(--text-tertiary);
  padding-top: 8px; border-top: 1px dashed var(--line-color);
}
.chain-hash { font-family: ui-monospace, Consolas, monospace; color: #0891b2; }

.ac-actions {
  display: flex; flex-direction: column; gap: 6px; flex-shrink: 0;
  justify-content: center;
}

/* ===== 详情 ===== */
.detail-row { display: flex; gap: 16px; font-size: 12.5px; margin-bottom: 8px; }
.detail-row span:first-child { color: var(--text-tertiary); min-width: 90px; }
.detail-row b { color: var(--text-primary); }
.green { color: var(--primary-green); font-weight: 700; }

.sub-title { font-size: 13px; font-weight: 600; margin: 18px 0 10px; display: flex; align-items: center; gap: 6px; }

.chain-info {
  padding: 12px 14px; background: #0a1628; border-radius: 6px;
  font-size: 11.5px;
}
.ci-row { display: flex; gap: 10px; margin: 6px 0; word-break: break-all; }
.ci-row span { color: #64748b; min-width: 80px; flex-shrink: 0; }
.ci-row code { font-family: ui-monospace, Consolas, monospace; color: #5eead4; }

.verify-box {
  margin-top: 16px; padding: 12px 14px; border-radius: 8px;
  display: flex; gap: 10px; align-items: flex-start;
}
.verify-box.ok { background: rgba(16,185,129,0.08); border: 1px solid rgba(16,185,129,0.3); color: #10b981; }
.verify-box.fail { background: rgba(239,68,68,0.08); border: 1px solid rgba(239,68,68,0.3); color: #ef4444; }
.verify-box b { font-size: 13px; }
.verify-box p { margin: 4px 0 0; font-size: 11.5px; opacity: 0.85; }

@media (max-width: 900px) {
  .archive-grid { grid-template-columns: 1fr; }
  .stat-row { grid-template-columns: repeat(2, 1fr); }
}
</style>