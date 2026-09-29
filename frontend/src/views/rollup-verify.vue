<!--
  rollup-verify.vue - Layer2 交易证明校验（regulator 角色）
  功能：
    1. ZK-Rollup 批次列表（批次号/交易数/状态/聚合哈希/DA 层）
    2. 单批次故障证明校验（VerifyRollupBatch）
    3. 校验结果详情（交易根哈希 / ZK 证明 / 链上状态）
  后端接口：scalingAPI.rollupBatches / verifyRollup
-->
<template>
  <div class="rv-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>
          <el-icon :size="18"><Tickets /></el-icon>
          Layer2 · ZK-Rollup 交易证明校验
        </h2>
        <p>监管专用：ZK-Rollup 批次聚合交易故障证明校验 · 可插拔 DA 层验证</p>
      </div>
      <el-tag effect="dark" size="small" type="success">
        <el-icon :size="12"><Connection /></el-icon>
        连接 Layer2 节点：l2.rollup.chain:9545
      </el-tag>
    </div>

    <!-- ===== 链上钱包（手动连接 → 读取 Ganache 链上交易数据 + 合约校验） ===== -->
    <div class="wallet-card">
      <WalletConnect hint="Layer2 校验 · 链上钱包（读取链上数据 / 合约校验证明）" />
      <div class="wallet-assets">
        <div class="wa-item">
          <span class="wa-label">链上最新区块高度</span>
          <span class="wa-val">{{ chainData.blockNumber }}</span>
        </div>
        <div class="wa-item wa-wide">
          <span class="wa-label">最新区块哈希</span>
          <span class="wa-val mono-val" :title="chainData.blockHashFull">{{ chainData.blockHash }}</span>
        </div>
        <div class="wa-item">
          <span class="wa-label">区块交易数</span>
          <span class="wa-val">{{ chainData.txCount }}</span>
        </div>
      </div>
    </div>

    <!-- ===== 统计卡 ===== -->
    <div class="stat-row">
      <div class="stat-card total">
        <div class="sc-icon"><el-icon :size="18"><Tickets /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.total }}</div>
          <div class="sc-lab">Rollup 批次总数</div>
        </div>
      </div>
      <div class="stat-card verified">
        <div class="sc-icon"><el-icon :size="18"><CircleCheck /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.verified }}</div>
          <div class="sc-lab">校验通过</div>
        </div>
      </div>
      <div class="stat-card pending">
        <div class="sc-icon"><el-icon :size="18"><Warning /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.pending }}</div>
          <div class="sc-lab">待校验</div>
        </div>
      </div>
      <div class="stat-card batch">
        <div class="sc-icon"><el-icon :size="18"><Collection /></el-icon></div>
        <div>
          <div class="sc-num">{{ stats.totalTxs.toLocaleString() }}</div>
          <div class="sc-lab">聚合交易总数</div>
        </div>
      </div>
    </div>

    <!-- ===== 批次列表 ===== -->
    <div class="list-card">
      <div class="lc-head">
        <h3>ZK-Rollup 批次列表</h3>
        <div class="lc-filters">
          <el-select v-model="statusFilter" placeholder="全部状态" size="small" style="width:130px" clearable>
            <el-option label="校验通过" value="verified" />
            <el-option label="待校验" value="pending" />
            <el-option label="校验失败（存在故障）" value="failed" />
          </el-select>
          <el-button :icon="Refresh" size="small" @click="loadBatches" :loading="loadingList">刷新</el-button>
          <el-button type="primary" :icon="Plus" size="small" @click="openCreate" :loading="creating">新增批次</el-button>
        </div>
      </div>

      <div v-if="loadingList" class="list-loading">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
      </div>
      <div v-else-if="!filteredBatches.length" class="list-empty">
        <el-icon :size="40"><Tickets /></el-icon>
        <p>暂无 Rollup 批次数据</p>
      </div>

      <el-table v-else :data="filteredBatches" stripe size="small" style="width:100%">
        <el-table-column label="批次号" width="170">
          <template #default="{ row }"><code class="mono">{{ row.batch_no }}</code></template>
        </el-table-column>
        <el-table-column label="区块高度" width="120" align="right">
          <template #default="{ row }">#{{ row.block_height?.toLocaleString() || '--' }}</template>
        </el-table-column>
        <el-table-column label="聚合交易数" width="110" align="right">
          <template #default="{ row }">{{ row.tx_count?.toLocaleString() || 0 }}</template>
        </el-table-column>
        <el-table-column label="DA 层" width="130">
          <template #default="{ row }">
            <el-tag size="small" :class="'da-tag ' + (row.da_layer || 'shared')">
              {{ row.da_layer === 'private' ? '私有 DA' : '共享 DA' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="聚合根哈希" min-width="220">
          <template #default="{ row }">
            <code class="mono-sm">{{ (row.aggregated_root || '').slice(0, 24) }}...</code>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag :class="'rv-status ' + row.verify_status" size="small" effect="dark">
              {{ verifyStatusMap[row.verify_status] || row.verify_status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="viewBatch(row)">详情</el-button>
            <el-button
              size="small" link
              :type="row.verify_status === 'verified' ? 'primary' : 'warning'"
              :loading="verifyingNo === row.batch_no"
              :disabled="!!verifyingNo && verifyingNo !== row.batch_no"
              @click="verifyBatch(row)"
            >
              {{ row.verify_status === 'verified' ? '重新校验' : '故障证明校验' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== 批次详情 + 校验弹窗 ===== -->
    <el-dialog v-model="detailVisible" width="680px" :title="'批次详情 · ' + (currentBatch?.batch_no || '')">
      <div v-if="currentBatch">
        <!-- 基础信息 -->
        <div class="detail-grid">
          <div class="dg-item"><span>批次号</span><code class="mono">{{ currentBatch.batch_no }}</code></div>
          <div class="dg-item"><span>区块高度</span>{{ currentBatch.block_height?.toLocaleString() }}</div>
          <div class="dg-item"><span>聚合交易数</span><b>{{ currentBatch.tx_count?.toLocaleString() }}</b></div>
          <div class="dg-item"><span>DA 层</span>
            <el-tag size="small" :class="'da-tag ' + (currentBatch.da_layer || 'shared')">
              {{ currentBatch.da_layer === 'private' ? '私有 DA' : '共享 DA' }}
            </el-tag>
          </div>
          <div class="dg-item"><span>验证时长</span>{{ currentBatch.proving_time_ms || '--' }} ms</div>
          <div class="dg-item"><span>状态</span>
            <el-tag :class="'rv-status ' + currentBatch.verify_status" size="small">
              {{ verifyStatusMap[currentBatch.verify_status] }}
            </el-tag>
          </div>
        </div>

        <!-- 哈希信息 -->
        <div class="hash-block">
          <h4>🔗 链上哈希</h4>
          <div class="hb-row">
            <span>聚合根哈希</span><code>{{ currentBatch.aggregated_root || '--' }}</code>
          </div>
          <div class="hb-row">
            <span>批次哈希</span><code>{{ currentBatch.batch_hash || '--' }}</code>
          </div>
          <div class="hb-row">
            <span>L1 锚定哈希</span><code>{{ currentBatch.l1_anchor || '--' }}</code>
          </div>
        </div>

        <!-- 批次内聚合交易明细（真实已上链交易） -->
        <div class="tx-block">
          <h4>📋 批次内聚合交易明细 <el-tag v-if="batchTxTotal" size="small" effect="plain" class="tx-total-tag">共 {{ batchTxTotal }} 笔</el-tag></h4>
          <div v-if="txLoading" class="tx-loading">
            <el-icon class="is-loading" :size="16"><Loading /></el-icon> 正在从 DA 层读取交易明细...
          </div>
          <el-table v-else-if="batchTxs.length" :data="batchTxs" size="small" max-height="240" stripe>
            <el-table-column label="交易编号" min-width="150">
              <template #default="{ row }"><code class="mono">{{ row.tx_no }}</code></template>
            </el-table-column>
            <el-table-column label="卖方（转让方）" min-width="150" show-overflow-tooltip>
              <template #default="{ row }">{{ row.seller }}</template>
            </el-table-column>
            <el-table-column label="买方（受让方）" min-width="150" show-overflow-tooltip>
              <template #default="{ row }">{{ row.buyer }}</template>
            </el-table-column>
            <el-table-column label="成交金额（元）" width="120" align="right">
              <template #default="{ row }">{{ Number(row.total_amount || 0).toLocaleString() }}</template>
            </el-table-column>
            <el-table-column label="交易时间" width="160">
              <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
          </el-table>
          <div v-else class="tx-empty">该批次暂无聚合交易明细</div>
        </div>

        <!-- 合约接口校验提示（预留 ABI 入口，钱包连接后展示） -->
        <div v-if="currentBatch.contractCallResult" class="contract-call-tip">
          <el-icon :size="12"><Connection /></el-icon>
          合约接口校验（预留 ABI 入口）：{{ currentBatch.contractCallResult }}
        </div>

        <!-- 校验结果 -->
        <div v-if="verifyResult" class="verify-box" :class="verifyResult.ok ? 'ok' : 'fail'">
          <div class="vb-head">
            <el-icon :size="18">
              <CircleCheck v-if="verifyResult.ok" />
              <Warning v-else />
            </el-icon>
            <span>{{ verifyResult.ok ? 'ZK-Rollup 批次校验通过' : 'ZK-Rollup 批次校验失败' }}</span>
          </div>
          <div class="vb-body">
            <div v-if="verifyResult.ok">
              <p>✓ 聚合根哈希与链上记录一致</p>
              <p>✓ ZK 证明 (Groth16) 验证通过</p>
              <p>✓ DA 层数据可用性确认（{{ verifyResult.da_mode || (currentBatch.da_layer === 'private' ? '私有 DA' : '共享 DA') }}）</p>
              <p>✓ L1 锚定哈希匹配</p>
              <p>✓ 聚合交易数 {{ currentBatch.tx_count }} 笔已全部包含</p>
            </div>
            <div v-else>
              <p class="vb-err">✗ {{ verifyResult.message }}</p>
              <p v-if="verifyResult.expected">期望哈希: <code>{{ verifyResult.expected }}</code></p>
              <p v-if="verifyResult.actual">实际哈希: <code>{{ verifyResult.actual }}</code></p>
            </div>
            <div class="vb-hash" v-if="verifyResult.proof_hash">
              ZK 证明哈希: <code>{{ verifyResult.proof_hash }}</code>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button
          type="primary" :icon="MagicStick"
          :loading="verifyingNo === currentBatch?.batch_no"
          @click="verifyBatch(currentBatch)"
        >
          执行故障证明校验
        </el-button>
      </template>
    </el-dialog>

    <!-- ===== 新增批次（打包成交记录）弹窗 ===== -->
    <el-dialog v-model="createVisible" width="760px" title="新增 ZK-Rollup 批次">
      <el-alert
        type="info" :closable="false" show-icon class="create-tip"
        title="系统将选取平台内未被打包的历史成交记录，自动打包为一个ZK-Rollup批次。"
      />
      <div class="create-row">
        <span class="cr-label">打包方式</span>
        <el-radio-group v-model="createMode">
          <el-radio value="auto">自动打包全部未打包交易（{{ unpackedTotal }} 笔）</el-radio>
          <el-radio value="manual">手动选择交易进行打包</el-radio>
        </el-radio-group>
      </div>
      <div class="create-row">
        <span class="cr-label">DA 层</span>
        <el-select v-model="createDa" style="width:170px">
          <el-option label="共享 DA" value="shared" />
          <el-option label="私有 DA" value="private" />
        </el-select>
      </div>

      <template v-if="createMode === 'manual'">
        <div class="create-row">
          <span class="cr-label">选择成交记录</span>
          <el-button size="small" link type="primary" :icon="Refresh" @click="loadUnpacked">刷新列表</el-button>
        </div>
        <el-table
          :data="unpackedTxs" v-loading="unpackedLoading" size="small" max-height="240" stripe
          @selection-change="s => (selectedTxs = s)"
        >
          <el-table-column type="selection" width="42" />
          <el-table-column label="交易编号" min-width="140">
            <template #default="{ row }"><code class="mono">{{ row.tx_no }}</code></template>
          </el-table-column>
          <el-table-column label="卖方" min-width="140" show-overflow-tooltip>
            <template #default="{ row }">{{ row.seller }}</template>
          </el-table-column>
          <el-table-column label="买方" min-width="140" show-overflow-tooltip>
            <template #default="{ row }">{{ row.buyer }}</template>
          </el-table-column>
          <el-table-column label="成交金额（元）" width="115" align="right">
            <template #default="{ row }">{{ Number(row.total_amount || 0).toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="交易时间" width="150">
            <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
        <div class="create-selected">
          已选 <b>{{ selectedTxs.length }}</b> 笔交易 · 批次号/区块高度/聚合根哈希由系统自动生成
        </div>
      </template>

      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button
          type="primary" :loading="creating"
          :disabled="createMode === 'manual' && !selectedTxs.length"
          @click="confirmCreate"
        >
          确认打包
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Tickets, Connection, CircleCheck, Warning, Collection, Refresh, Loading, MagicStick, Plus
} from '@element-plus/icons-vue'
import { scalingAPI } from '../api/index.js'
import WalletConnect from '../components/WalletConnect.vue'
import { useWalletStore } from '../stores/wallet.js'
import {
  getLatestBlock, callContractRead, formatWalletError,
  CARBON_CONTRACTS, abiEncodeCall, abiEncodeWord,
} from '../utils/web3.js'

const verifyStatusMap = { verified: '校验通过', pending: '待校验', failed: '校验失败（存在故障）' }

/* ===== 链上钱包：读取 Sepolia 链上交易数据 + 合约校验入口（MetaMask，仅用户手动连接） ===== */
const wallet = useWalletStore()
const chainData = reactive({ blockNumber: '--', blockHash: '--', blockHashFull: '', txCount: '--' })

/* 钱包就绪状态变化：就绪 → 读取链上区块数据；断开 / 切离 Sepolia → 清空展示 */
watch(() => wallet.isReady, ready => ready ? loadChainData() : resetChainData())

/** 连接钱包后读取链上最新区块数据（区块高度 / 哈希 / 交易数） */
async function loadChainData() {
  if (!wallet.isReady) return
  try {
    const latest = await getLatestBlock()
    chainData.blockNumber = latest.number
    chainData.blockHashFull = latest.hash
    chainData.blockHash = latest.hash.slice(0, 14) + '...' + latest.hash.slice(-8)
    chainData.txCount = latest.txCount
    ElMessage.success('已读取 Sepolia 链上最新区块数据')
  } catch (e) {
    ElMessage.error(formatWalletError(e))
  }
}

function resetChainData() {
  chainData.blockNumber = '--'
  chainData.blockHash = '--'
  chainData.blockHashFull = ''
  chainData.txCount = '--'
}

/* ===== 状态 ===== */
const batches = ref([])
const loadingList = ref(false)
const statusFilter = ref('')
const detailVisible = ref(false)
const currentBatch = ref(null)
const verifyResult = ref(null)
const verifyingNo = ref('') // 正在校验的批次号（按钮 loading / 禁用）
const batchTxs = ref([])   // 批次内聚合交易明细
const batchTxTotal = ref(0)
const txLoading = ref(false)

const stats = computed(() => ({
  total: batches.value.length,
  verified: batches.value.filter(b => b.verify_status === 'verified').length,
  pending: batches.value.filter(b => b.verify_status === 'pending').length,
  totalTxs: batches.value.reduce((s, b) => s + (b.tx_count || 0), 0),
}))

const filteredBatches = computed(() => {
  if (!statusFilter.value) return batches.value
  return batches.value.filter(b => b.verify_status === statusFilter.value)
})

/* ===== 加载 ===== */
async function loadBatches() {
  loadingList.value = true
  try {
    // 加载全部批次记录（统计口径: 批次表全部行数）
    const res = await scalingAPI.rollupBatches()
    batches.value = res.data?.list || []
  } catch (e) {
    batches.value = []
    ElMessage.error(e?.msg || '加载 Rollup 批次失败')
  }
  loadingList.value = false
}

/* ===== 新增批次（打包成交记录） ===== */
const createVisible = ref(false)
const createMode = ref('auto')     // auto=自动打包全部 / manual=手动选择
const createDa = ref('shared')     // DA层: shared(默认) / private
const unpackedTxs = ref([])        // 未打包成交记录
const unpackedTotal = ref(0)
const unpackedLoading = ref(false)
const selectedTxs = ref([])        // 手动模式勾选的交易
const creating = ref(false)

// 切回自动模式时清空勾选
watch(createMode, m => { if (m === 'auto') selectedTxs.value = [] })

function openCreate() {
  createVisible.value = true
  createMode.value = 'auto'
  createDa.value = 'shared'
  selectedTxs.value = []
  loadUnpacked()
}

/** 加载未打包成交记录 */
async function loadUnpacked() {
  unpackedLoading.value = true
  try {
    const res = await scalingAPI.unpackedTransactions()
    unpackedTxs.value = res.data?.list || []
    unpackedTotal.value = res.data?.total || unpackedTxs.value.length
  } catch {
    unpackedTxs.value = []
    unpackedTotal.value = 0
  }
  unpackedLoading.value = false
}

/** 确认打包生成批次 */
async function confirmCreate() {
  creating.value = true
  try {
    const payload = { mode: createMode.value, da_source: createDa.value }
    if (createMode.value === 'manual') {
      payload.tx_nos = selectedTxs.value.map(t => t.tx_no)
    }
    // 钱包已连接时优先采用 Ganache 链上区块高度
    const n = Number(chainData.blockNumber)
    if (wallet.isReady && Number.isFinite(n) && n > 0) payload.chain_height = n
    const res = await scalingAPI.createRollupBatch(payload)
    ElMessage.success(res?.msg || 'ZK-Rollup 批次打包成功')
    createVisible.value = false
    await loadBatches() // 刷新列表 → 顶部统计卡片实时联动
  } catch (e) {
    ElMessage.error(e?.msg || '新增批次失败')
  }
  creating.value = false
}

/* ===== 详情 ===== */
function viewBatch(b) {
  currentBatch.value = b
  verifyResult.value = null
  detailVisible.value = true
  loadBatchTxs(b)
}

/** 加载批次内聚合交易明细（真实已上链交易） */
async function loadBatchTxs(b) {
  txLoading.value = true
  batchTxs.value = []
  batchTxTotal.value = 0
  try {
    const res = await scalingAPI.batchTransactions({ batch_no: b.batch_no })
    batchTxs.value = res.data?.list || []
    batchTxTotal.value = res.data?.total || batchTxs.value.length
  } catch {
    batchTxs.value = []
  }
  txLoading.value = false
}

/** 时间格式化 */
function fmtTime(t) {
  if (!t) return '--'
  return String(t).replace('T', ' ').slice(0, 19)
}

/* ===== 校验 ===== */
async function verifyBatch(b) {
  if (!b) return
  currentBatch.value = b
  verifyResult.value = null
  if (!detailVisible.value) detailVisible.value = true
  verifyingNo.value = b.batch_no

  // 钱包就绪时：先调用 Layer2 校验合约接口（预留 ABI 入口）尝试链上校验
  if (wallet.isReady) {
    try {
      const { address, methods } = CARBON_CONTRACTS.rollupVerifier
      const data = abiEncodeCall(methods.verifyBatch, [abiEncodeWord(b.aggregated_root || '')])
      const res = await callContractRead(address, data)
      b.contractCallResult = (res && res !== '0x')
        ? '合约返回 ' + res.slice(0, 12) + '...（链上校验完成）'
        : '合约未部署，返回空值（0x），回退后端聚合根比对校验'
    } catch (e) {
      b.contractCallResult = '合约接口调用失败：' + formatWalletError(e)
    }
  }

  try {
    const res = await scalingAPI.verifyRollup({ batch_no: b.batch_no })
    const d = res.data || {}
    const ok = d.ok ?? d.verified
    verifyResult.value = { ok, proof_hash: d.proof_hash, message: d.proof }
    // 同步批次状态 → 顶部统计卡片经 computed 自动联动更新
    b.verify_status = d.verify_status || (ok ? 'verified' : 'failed')
    b.fault_proof = d.proof || b.fault_proof
    if (ok) {
      ElMessageBox.alert('故障证明校验完成，该批次全部碳积分交易合法有效', '校验通过', {
        type: 'success', confirmButtonText: '知道了',
      })
    } else {
      ElMessageBox.alert('该批次ZK证明校验失败，交易数据存在异常故障，请核查', '校验失败', {
        type: 'error', confirmButtonText: '知道了',
      })
    }
  } catch (e) {
    ElMessage.error(e?.msg || '校验请求失败，请稍后重试')
  }

  verifyingNo.value = ''
}

onMounted(() => loadBatches())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #064e3b, #064e3b);
  border-radius: var(--radius-lg); color: #fff;
}
.page-head h2 { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.page-head p { font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }

/* ===== 新增批次弹窗 ===== */
.create-tip { margin-bottom: 14px; }
.create-row {
  display: flex; align-items: center; gap: 12px;
  margin-bottom: 12px;
}
.cr-label { font-size: 12.5px; color: var(--text-tertiary); min-width: 84px; flex-shrink: 0; }
.create-selected { margin-top: 10px; font-size: 12px; color: var(--text-tertiary); }
.create-selected b { color: var(--primary-green); }

/* ===== 批次内交易明细 ===== */
.tx-block { margin-bottom: 14px; }
.tx-block h4 {
  display: flex; align-items: center; gap: 8px;
  font-size: 12px; font-weight: 600; color: var(--text-primary); margin: 0 0 10px;
}
.tx-total-tag { color: var(--primary-green); }
.tx-loading {
  display: flex; align-items: center; justify-content: center; gap: 8px;
  padding: 24px; color: var(--text-tertiary); font-size: 12.5px;
}
.tx-empty { padding: 20px; text-align: center; color: var(--text-tertiary); font-size: 12.5px; }

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
.stat-card.total    { border-left-color: #6366f1; }
.stat-card.verified { border-left-color: #10b981; }
.stat-card.pending  { border-left-color: #f59e0b; }
.stat-card.batch    { border-left-color: #0891b2; }
.sc-icon {
  width: 40px; height: 40px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-card.total    .sc-icon { background: rgba(99,102,241,0.12); color: #6366f1; }
.stat-card.verified .sc-icon { background: rgba(16,185,129,0.12); color: #10b981; }
.stat-card.pending  .sc-icon { background: rgba(245,158,11,0.12); color: #d97706; }
.stat-card.batch    .sc-icon { background: rgba(8,145,178,0.12); color: #0891b2; }
.sc-num { font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary); }
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

.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }
.mono-sm { font-family: ui-monospace, Consolas, monospace; font-size: 10.5px; }

.da-tag.shared  { background: rgba(16,185,129,0.1); color: #2563eb; }
.da-tag.private { background: rgba(124,58,237,0.1); color: #7c3aed; }

.rv-status.verified  { background: rgba(16,185,129,0.15); color: #10b981; border: none; }
.rv-status.pending   { background: rgba(245,158,11,0.15); color: #d97706; border: none; }
.rv-status.failed    { background: rgba(239,68,68,0.15); color: #ef4444; border: none; }
.rv-status.verifying { background: rgba(99,102,241,0.15); color: #6366f1; border: none; }

/* ===== 详情 ===== */
.detail-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 12px; margin-bottom: 14px;
}
.dg-item {
  display: flex; flex-direction: column; gap: 4px;
  padding: 10px 14px; background: var(--bg-secondary); border-radius: 8px;
  font-size: 12px;
}
.dg-item span { color: var(--text-tertiary); font-size: 11px; }
.dg-item b { color: var(--primary-green); }

.hash-block {
  padding: 14px; background: #0a1628; border-radius: 8px; margin-bottom: 14px;
}
.hash-block h4 { font-size: 12px; color: #5eead4; margin: 0 0 10px; font-weight: 600; }
.hb-row {
  display: flex; gap: 10px; font-size: 11.5px; margin: 5px 0;
  word-break: break-all;
}
.hb-row span { color: #64748b; min-width: 90px; flex-shrink: 0; }
.hb-row code { font-family: ui-monospace, Consolas, monospace; color: #cbd5e1; }

.verify-box {
  border-radius: 10px; padding: 14px 18px;
}
.verify-box.ok {
  background: rgba(16,185,129,0.08);
  border: 1px solid rgba(16,185,129,0.3);
  color: #10b981;
}
.verify-box.fail {
  background: rgba(239,68,68,0.08);
  border: 1px solid rgba(239,68,68,0.3);
  color: #ef4444;
}
.vb-head {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 700; margin-bottom: 10px;
}
.vb-body p { font-size: 12.5px; margin: 4px 0; }
.vb-body code { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }
.vb-err { font-weight: 600; }
.vb-hash {
  margin-top: 12px; padding-top: 10px;
  border-top: 1px dashed rgba(255,255,255,0.2);
  font-size: 11px;
}

/* ===== 链上钱包卡 + 合约调用提示 ===== */
.wallet-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 14px 18px; margin-bottom: 16px;
  border-left: 4px solid var(--primary-green);
}
.wallet-assets { display: flex; align-items: center; gap: 28px; flex-wrap: wrap; }
.wa-item { display: flex; flex-direction: column; gap: 2px; min-width: 120px; }
.wa-item.wa-wide { min-width: 220px; }
.wa-label { font-size: 11px; color: var(--text-tertiary); }
.wa-val { font-size: 18px; font-weight: 700; color: var(--text-primary); font-variant-numeric: tabular-nums; }
.mono-val { font-family: ui-monospace, Consolas, monospace; font-size: 13px; font-weight: 500; }
.contract-call-tip {
  display: flex; align-items: center; gap: 6px;
  margin-bottom: 12px; padding: 8px 12px;
  background: rgba(8,145,178,0.08); border: 1px dashed rgba(8,145,178,0.4);
  border-radius: 8px; font-size: 11.5px; color: #0891b2;
  word-break: break-all;
}

@media (max-width: 900px) {
  .stat-row { grid-template-columns: repeat(2, 1fr); }
  .detail-grid { grid-template-columns: 1fr; }
}
</style>