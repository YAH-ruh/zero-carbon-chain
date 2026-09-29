<!--
  credits-center.vue - 碳积分资产中心
  跨 4 角色共享页面，按 role 切换视图：
    enterprise  → 我的资产：余额统计 + 积分列表 + 挂单出售
    park_admin  → 园区汇总：各企业积分分布
    exchange   → 可交易库存
    regulator  → 审计视角：合规状态 + 链上核验
-->
<template>
  <div class="credits-center page" :class="themeDark ? 'dark' : 'light'">

    <!-- ===== 头部：企业专属挂出售按钮 ===== -->
    <div class="cc-head">
      <div class="cc-title">
        <h2>碳积分资产中心</h2>
        <p>{{ roleLabel }} · 余额统计 · 链上留痕</p>
      </div>
      <div class="cc-action">
        <el-tooltip content="可用碳积分不足，请撤销挂单或赎回质押积分" placement="bottom" :disabled="canSell">
          <el-button
            v-if="role === 'enterprise'"
            type="primary"
            :icon="TrendCharts"
            @click="onSellEntry"
            :disabled="!canSell"
          >
            挂单出售积分
          </el-button>
        </el-tooltip>
        <el-button :icon="Refresh" @click="refreshAll(false)" :loading="loadingAll">刷新</el-button>
      </div>
    </div>

    <!-- ===== 链上钱包卡（企业专属：手动连接钱包 → 链上余额读取 / 质押 / 转出） ===== -->
    <div v-if="role === 'enterprise'" class="wallet-card">
      <WalletConnect hint="碳积分资产中心 · 链上钱包（余额读取 / 质押 / 转出）" />
      <div class="wallet-assets">
        <div class="wa-item">
          <span class="wa-label">链上映射碳积分</span>
          <span class="wa-val">{{ walletAssets.credits }}</span>
          <span class="wa-sub">{{ walletAssets.creditsSource }}</span>
        </div>
        <div class="wa-item">
          <span class="wa-label">质押中积分</span>
          <span class="wa-val">{{ walletAssets.pledged.toFixed(2) }}</span>
        </div>
        <div class="wa-item">
          <span class="wa-label">Sepolia ETH 余额</span>
          <span class="wa-val">{{ walletAssets.eth }}</span>
        </div>
        <div class="wa-actions">
          <el-button type="primary" size="small" :icon="Lock" :disabled="!wallet.isReady" @click="openPledgeDialog">
            积分质押
          </el-button>
          <el-button type="warning" size="small" plain :icon="Coin" :disabled="!hasActivePledge" @click="openRedeemDialog">
            质押赎回
          </el-button>
          <el-button type="success" size="small" plain :icon="Position" :disabled="!wallet.isReady" @click="openTransferDialog">
            积分转出
          </el-button>
        </div>
      </div>
      <div v-if="walletTxs.length" class="wallet-tx-log">
        <span class="wt-title">最近链上操作：</span>
        <span v-for="(t, i) in walletTxs.slice(0, 3)" :key="i" class="wt-item" :title="t.hash">
          {{ t.label }} · {{ t.hash.slice(0, 10) }}...{{ t.hash.slice(-6) }}
        </span>
      </div>
    </div>

    <!-- ===== 无碳资产角色提示卡（园区管理员/监管核查；交易所可见碳资产，不调用统计接口） ===== -->
    <div v-if="false" class="no-perm-card">
      <el-icon :size="42" class="np-icon"><Wallet /></el-icon>
      <h3>当前角色不持有碳资产</h3>
      <p>请切换至小微企业角色查看资产中心</p>
    </div>

    <!-- ===== 余额统计卡（小微企业=本企业口径；其余角色=全平台碳资产口径） ===== -->
    <div class="balance-row">
      <div class="bal-card">
        <div class="bal-icon" style="background:rgba(16,185,129,0.1);color:#10b981">
          <el-icon :size="18"><Wallet /></el-icon>
        </div>
        <div class="bal-body">
          <div class="bal-num">{{ stats.total_credits?.toFixed(2) || 0 }}</div>
          <div class="bal-label">累计核算积分</div>
        </div>
      </div>
      <div class="bal-card">
        <div class="bal-icon" style="background:rgba(34,197,94,0.1);color:#22c55e">
          <el-icon :size="18"><Coin /></el-icon>
        </div>
        <div class="bal-body">
          <div class="bal-num">{{ stats.available_credits?.toFixed(2) || 0 }}</div>
          <div class="bal-label">可用余额</div>
        </div>
      </div>
      <div class="bal-card">
        <div class="bal-icon" style="background:rgba(245,158,11,0.1);color:#f59e0b">
          <el-icon :size="18"><TrendCharts /></el-icon>
        </div>
        <div class="bal-body">
          <div class="bal-num">{{ stats.sold_credits?.toFixed(2) || 0 }}</div>
          <div class="bal-label">已售出</div>
        </div>
      </div>
      <div class="bal-card">
        <div class="bal-icon" style="background:rgba(16,185,129,0.1);color:#0d9488">
          <el-icon :size="18"><DataBoard /></el-icon>
        </div>
        <div class="bal-body">
          <div class="bal-num">{{ stats.total_emission?.toFixed(0) || 0 }}</div>
          <div class="bal-label">累计排放(kgCO₂)</div>
        </div>
      </div>
      <div class="bal-card">
        <div class="bal-icon" style="background:rgba(124,58,237,0.1);color:#7c3aed">
          <el-icon :size="18"><Lock /></el-icon>
        </div>
        <div class="bal-body">
          <div class="bal-num">{{ stats.on_chain_credits?.toFixed(2) || 0 }}</div>
          <div class="bal-label">已上链存证(积分)</div>
        </div>
      </div>
      <div class="bal-card">
        <div class="bal-icon" style="background:rgba(245,158,11,0.1);color:#d97706">
          <el-icon :size="18"><Coin /></el-icon>
        </div>
        <div class="bal-body">
          <div class="bal-num">{{ stats.pledged_credits?.toFixed(2) || 0 }}</div>
          <div class="bal-label">质押中的积分</div>
        </div>
      </div>
    </div>

    <!-- ===== 实时交易折线（小微企业=本企业；其余角色=全平台口径） ===== -->
    <div class="charts-row">
        <div class="chart-card chart-main">
          <div class="cc-chart-head">
            <h3>近 12 个月碳积分余额变化</h3>
          </div>
          <EChart :option="balanceTrendOption" height="280px" />
        </div>
        <div class="chart-card chart-side">
          <div class="cc-chart-head">
            <h3>积分来源构成</h3>
          </div>
          <EChart :option="sourcePieOption" height="280px" />
        </div>
      </div>

    <!-- ===== 空资产友好提示（接口成功但名下无碳积分） ===== -->
    <el-alert
      v-if="role === 'enterprise' && statsLoaded && !stats.credit_count"
      class="empty-asset-alert"
      type="info" :closable="false" show-icon
      title="暂无碳积分资产"
      description="完成能耗上报并通过碳积分核算后，名下碳积分资产将在此展示"
    />

    <!-- ===== 碳积分记录表格（小微企业=本企业；其余角色=全平台碳资产明细） ===== -->
    <div class="list-card">
      <div class="list-head">
        <h3>碳积分记录</h3>
        <div class="list-filters">
          <el-input
            v-model="keyword"
            placeholder="搜索编号 / 企业"
            :prefix-icon="Search"
            clearable
            size="small"
            style="width: 200px"
          />
          <el-select v-model="statusFilter" placeholder="全部状态" size="small" style="width: 130px" clearable>
            <el-option label="已上链" value="on" />
            <el-option label="未上链" value="off" />
            <el-option label="已挂单" value="listed" />
            <el-option label="已出售" value="sold" />
          </el-select>
        </div>
      </div>
      <el-table
        :data="filteredCredits"
        v-loading="loadingCredits"
        stripe
        style="width: 100%"
        empty-text="暂无碳积分记录"
      >
        <el-table-column prop="credit_no" label="核算编号" width="200">
          <template #default="{ row }"><code class="mono">{{ row.credit_no }}</code></template>
        </el-table-column>

        <!-- enterprise 看自己 -->
        <el-table-column v-if="role !== 'enterprise'" label="企业" width="160">
          <template #default="{ row }">{{ row.enterprise?.company || row.enterprise?.username || '匿名企业' }}</template>
        </el-table-column>

        <el-table-column label="碳排放(kg)" width="120" align="right">
          <template #default="{ row }">{{ row.total_emission?.toFixed(2) || '--' }}</template>
        </el-table-column>
        <el-table-column label="碳积分" width="120" align="right">
          <template #default="{ row }">
            <span class="credit-val">{{ row.carbon_credits?.toFixed(2) || '--' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="积分状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag :class="'status-tag ' + (row.status || 'pending')" effect="plain" size="small">
              {{ creditStatusMap[row.status] || '待处理' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="链上状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" type="success" size="small" effect="dark">已上链</el-tag>
            <el-tag v-else type="info" size="small">未上链</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="区块哈希" width="180">
          <template #default="{ row }"><code class="mono mono-sm">{{ row.block_hash || '--' }}</code></template>
        </el-table-column>
        <el-table-column label="核算时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>

        <!-- enterprise 专属操作列 -->
        <el-table-column v-if="role === 'enterprise'" label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'available' && row.carbon_credits > 0"
              type="primary" link size="small"
              @click="openSellDialog(row)"
            >挂单出售</el-button>
            <el-button
              v-else-if="row.status === 'locked'"
              type="warning" link size="small"
              @click="doCancelOrder(row)"
            >撤销挂单</el-button>
            <span v-else class="op-empty">--</span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== 链上挂单出售弹窗（仅 enterprise，复用交易所挂单口径） ===== -->
    <el-dialog v-model="sellDialogVisible" width="460px" title="链上挂单出售碳积分">
      <el-form label-width="100px">
        <el-form-item label="可交易积分">
          <span class="credit-val">{{ Number(stats.available_credits || 0).toFixed(2) }}</span>
          <span class="sell-hint">（自动读取当前可用余额）</span>
        </el-form-item>
        <el-form-item label="碳积分凭证" required>
          <el-select
            v-model="selectedCreditId"
            placeholder="请选择本人持有的碳积分凭证"
            style="width:100%" :loading="loadingMyCredits"
            @change="onSellCreditPicked"
          >
            <el-option
              v-for="c in myAvailableCredits" :key="c.id"
              :value="c.id"
              :label="`${c.credit_no}（${c.carbon_credits} 积分）`"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="selectedCredit" label="凭证编号">
          <code class="mono mono-sm">{{ selectedCredit.credit_no }}</code>
        </el-form-item>
        <el-form-item label="可用积分">
          <span class="credit-val">{{ selectedCredit?.carbon_credits?.toFixed(2) || '0.00' }}</span>
        </el-form-item>
        <el-form-item label="挂单数量">
          <el-input-number
            v-model="sellForm.quantity"
            :min="0.01"
            :max="selectedCredit ? selectedCredit.carbon_credits : Math.max(Number(stats.available_credits || 0), 0.02)"
            :precision="2"
            :disabled="!selectedCredit"
            placeholder="请先选择碳积分凭证"
            style="width:100%"
          />
        </el-form-item>
        <el-form-item label="单价 (元)">
          <el-input-number
            v-model="sellForm.unit_price"
            :min="1" :max="200" :precision="2"
            style="width:100%"
          />
        </el-form-item>
        <el-alert
          type="info" :closable="false" show-icon
          :title="`预计成交总额：¥${(sellForm.quantity * sellForm.unit_price).toFixed(2)}`"
        />
      </el-form>
      <template #footer>
        <el-button @click="sellDialogVisible = false">取消</el-button>
        <el-button
          type="primary" :loading="loadingSell"
          :disabled="!selectedCredit || !sellForm.quantity"
          @click="doSell"
        >确认挂单</el-button>
      </template>
    </el-dialog>

    <!-- ===== 积分质押弹窗（唤起 MetaMask 签名） ===== -->
    <el-dialog v-model="pledgeVisible" width="420px" title="碳积分质押（链上签名）">
      <el-form label-width="100px">
        <el-form-item label="钱包地址">
          <span class="mono">{{ wallet.shortAddress }}</span>
        </el-form-item>
        <el-form-item label="质押数量">
          <el-input-number v-model="pledgeForm.amount" :min="1" :max="maxPledgeable" :precision="2" style="width:100%" />
        </el-form-item>
        <el-form-item label="质押期限">
          <el-select v-model="pledgeForm.term" style="width:100%">
            <el-option label="3 个月（年化 4%）" :value="3" />
            <el-option label="6 个月（年化 5%）" :value="6" />
            <el-option label="12 个月（年化 6%）" :value="12" />
          </el-select>
        </el-form-item>
        <el-alert type="info" :closable="false" show-icon title="确认后将唤起 MetaMask 签名，调用碳积分合约 pledge 方法锁定积分" />
      </el-form>
      <template #footer>
        <el-button @click="pledgeVisible = false">取消</el-button>
        <el-button type="primary" :loading="loadingPledge" :disabled="!pledgeForm.amount" @click="doPledge">签名并质押</el-button>
      </template>
    </el-dialog>

    <!-- ===== 积分转出弹窗（唤起 MetaMask 签名） ===== -->
    <el-dialog v-model="transferVisible" width="440px" title="碳积分转出（链上签名）">
      <el-form label-width="100px">
        <el-form-item label="钱包地址">
          <span class="mono">{{ wallet.shortAddress }}</span>
        </el-form-item>
        <el-form-item label="收款地址">
          <el-input v-model="transferForm.to" placeholder="0x 开头的目标钱包地址" clearable />
        </el-form-item>
        <el-form-item label="转出数量">
          <el-input-number v-model="transferForm.amount" :min="1" :max="maxPledgeable" :precision="2" style="width:100%" />
        </el-form-item>
        <el-alert type="warning" :closable="false" show-icon title="确认后将唤起 MetaMask 签名，调用合约 transfer 方法转出积分，请仔细核对收款地址" />
      </el-form>
      <template #footer>
        <el-button @click="transferVisible = false">取消</el-button>
        <el-button type="primary" :loading="loadingTransfer" :disabled="!transferForm.amount || !transferForm.to" @click="doTransfer">签名并转出</el-button>
      </template>
    </el-dialog>

    <!-- ===== 质押赎回弹窗（解除质押锁定，可用余额恢复） ===== -->
    <el-dialog v-model="redeemVisible" width="460px" title="质押赎回">
      <el-form label-width="100px">
        <el-form-item label="质押单" required>
          <el-select
            v-model="redeemNo"
            placeholder="请选择本企业质押中的质押单"
            style="width:100%" :loading="loadingRedeemList"
          >
            <el-option
              v-for="p in activePledges" :key="p.pledge_no"
              :value="p.pledge_no"
              :label="`${p.pledge_no}（质押 ${p.pledge_amount} 积分）`"
            />
          </el-select>
        </el-form-item>
        <el-alert
          type="info" :closable="false" show-icon
          :title="`赎回后解锁对应积分，可用余额将恢复 ${selectedPledge?.pledge_amount?.toFixed(2) || '0.00'} 积分`"
        />
      </el-form>
      <template #footer>
        <el-button @click="redeemVisible = false">取消</el-button>
        <el-button type="primary" :loading="loadingRedeem" :disabled="!redeemNo" @click="doRedeem">确认赎回</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onActivated, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Wallet, Coin, TrendCharts, DataBoard, Lock, Refresh, Search, Position
} from '@element-plus/icons-vue'
import EChart from '../components/EChart.vue'
import WalletConnect from '../components/WalletConnect.vue'
import { useTheme } from '../composables/useTheme.js'
import { carbonAPI, rwaAPI } from '../api/index.js'
import { useWalletStore } from '../stores/wallet.js'
import {
  getNativeBalance, callContractRead, sendContractTransaction, formatWalletError,
  loadDeployedContracts, isUsableContractAddress, getChainId, SEPOLIA, withWalletWatchdog,
  CARBON_CONTRACTS, abiEncodeCall, abiEncodeUint256, abiEncodeWord,
} from '../utils/web3.js'

const router = useRouter()
const user = JSON.parse(localStorage.getItem('user') || '{}')
const role = user.role || 'enterprise'
const themeDark = computed(() => {
  const { isDark } = useTheme()
  return isDark()
})

const roleLabelMap = {
  enterprise: '小微企业视角',
  park_admin: '园区管理员视角',
  exchange: '交易所视角',
  regulator: '监管核查视角'
}
const roleLabel = roleLabelMap[role] || ''

/* ===== 数据 ===== */
const stats = ref({})
const credits = ref([])
const loadingCredits = ref(false)
const loadingAll = ref(false)
const loadingSell = ref(false)
const keyword = ref('')
const statusFilter = ref('')

const creditStatusMap = { available: '可用', listed: '已挂单', sold: '已出售', expired: '已过期', used: '已使用' }

/* 可用积分（可挂单出售） */
const availableForSell = computed(() => credits.value.filter(c => c.status === 'available' && c.carbon_credits > 0))
const availableForSellTotal = computed(() => availableForSell.value.reduce((s, c) => s + (c.carbon_credits || 0), 0))

/* 挂单出售前置校验：可用余额 > 0 才允许发起（按钮置灰 + tooltip 提示） */
const canSell = computed(() => Number(stats.value.available_credits || availableForSellTotal.value || 0) > 0)

/* 过滤 */
const filteredCredits = computed(() => {
  let arr = credits.value
  if (keyword.value) {
    const kw = keyword.value.toLowerCase()
    arr = arr.filter(c =>
      (c.credit_no?.toLowerCase().includes(kw)) ||
      (c.enterprise?.company?.toLowerCase().includes(kw))
    )
  }
  if (statusFilter.value === 'on') arr = arr.filter(c => c.on_chain)
  else if (statusFilter.value === 'off') arr = arr.filter(c => !c.on_chain)
  else if (statusFilter.value === 'listed') arr = arr.filter(c => c.status === 'listed')
  else if (statusFilter.value === 'sold') arr = arr.filter(c => c.status === 'sold')
  return arr
})

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

/* ===== 加载(全部真实接口数据，无 mock 兜底；异常统一提示) ===== */
const statsLoaded = ref(false)

async function loadCredits() {
  loadingCredits.value = true
  try {
    const res = await carbonAPI.myCredits({ page: 1, page_size: 50 })
    credits.value = res.data?.list || res.data || []
    return true
  } catch {
    credits.value = []
    return false
  } finally {
    loadingCredits.value = false
  }
}

/* 统计卡实时数据：累计核算/可用余额/已售出/累计排放/已上链存证/质押中(全部后端计算) */
async function loadStats() {
  try {
    const res = await carbonAPI.stats({})
    stats.value = res.data || {}
    return true
  } catch {
    stats.value = {}
    return false
  }
}

/**
 * 全量刷新：统计卡 + 图表 + 积分表格 + 链上钱包余额
 * 自动刷新(页面进入/挂单/撤销/赎回成功)与手动刷新共用本函数，保证数据口径一致。
 * 防重入：加载中直接跳过，防止短时间重复调用接口。
 */
async function refreshAll(silent = true) {
  // 全角色可见碳资产：小微企业=本企业口径，其余角色(园区管理员/碳交易所/监管)=全平台口径
  if (loadingAll.value) return
  loadingAll.value = true
  const results = await Promise.allSettled([loadCredits(), loadStats()])
  statsLoaded.value = true
  loadingAll.value = false
  // 链上映射碳积分跟随刷新(已连接钱包时重读合约余额)
  if (wallet.isReady) loadWalletAssets()
  const failed = results.some(r => r.status === 'rejected' || r.value === false)
  if (failed) {
    ElMessage.error('数据加载失败，请重试')
  } else if (!silent) {
    ElMessage.success('数据已刷新')
  }
}

/* ===== 挂出售 ===== */
const sellDialogVisible = ref(false)
const selectedCredit = ref(null)
const sellForm = reactive({ quantity: 0, unit_price: 50 })

/* 凭证下拉：本人持有的可用碳积分凭证(选中后自动读取 CreditNo 关联挂单) */
const myAvailableCredits = ref([])
const loadingMyCredits = ref(false)
const selectedCreditId = ref(null)

async function loadMyAvailableCredits() {
  loadingMyCredits.value = true
  try {
    const res = await carbonAPI.myCredits({ page: 1, page_size: 50 })
    const list = res.data?.list || res.data || []
    myAvailableCredits.value = list.filter(c => c.status === 'available' && c.carbon_credits > 0)
  } catch {
    myAvailableCredits.value = []
  } finally {
    loadingMyCredits.value = false
  }
}

/** 凭证选中后：自动读取该凭证 CreditNo 并按其积分量重置挂单数量 */
function onSellCreditPicked(id) {
  const c = myAvailableCredits.value.find(x => x.id === id)
  selectedCredit.value = c || null
  sellForm.quantity = c ? Number(c.carbon_credits) : 0
}

function openSellDialog(credit = null) {
  selectedCredit.value = credit
  selectedCreditId.value = credit?.id ?? null
  sellForm.quantity = credit ? Number(credit.carbon_credits) : 0
  sellForm.unit_price = 50
  sellDialogVisible.value = true
  loadMyAvailableCredits()
}

/** 头部【挂单出售积分】入口：可用余额>0 才弹窗；弹窗自动读取可用余额与本人可用凭证 */
function onSellEntry() {
  if (!canSell.value) {
    ElMessage.warning('可用碳积分不足，无法发起挂单出售')
    return
  }
  openSellDialog(null)
}

async function doSell() {
  if (!selectedCredit.value) {
    ElMessage.warning('请先选择本人持有的碳积分凭证')
    return
  }
  loadingSell.value = true
  try {
    await carbonAPI.sell({
      credit_id: selectedCredit.value.id, // 凭证ID：后端据此将 CreditNo 关联到挂单记录
      quantity: sellForm.quantity,
      unit_price: sellForm.unit_price,
    })
    ElMessage.success(`挂单成功！凭证 ${selectedCredit.value.credit_no} 数量 ${sellForm.quantity} 积分 @ ¥${sellForm.unit_price}`)
    sellDialogVisible.value = false
    // 挂单成功同步刷新：统计卡 + 图表 + 积分表格 + 链上钱包余额(数据同步到交易所现货挂单列表)
    await refreshAll(true)
  } catch (e) {
    ElMessage.error('挂单创建失败：' + (e?.msg || '请检查碳积分状态后重试'))
  } finally {
    loadingSell.value = false
  }
}

/** 撤销挂单(业务联动：解除积分冻结，可用余额恢复；未成交资产归属不变) */
async function doCancelOrder(row) {
  try {
    await ElMessageBox.confirm(
      `撤销凭证 ${row.credit_no} 的挂单？撤销后冻结的 ${row.carbon_credits?.toFixed?.(2) || row.carbon_credits} 积分将恢复可用`,
      '撤销挂单确认',
      { type: 'warning', confirmButtonText: '确认撤销', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await carbonAPI.cancelOrder({ credit_id: row.id })
    ElMessage.success('挂单已撤销，碳积分冻结已解除')
    await refreshAll(true)
  } catch (e) {
    ElMessage.error('撤销挂单失败：' + (e?.msg || '请稍后重试'))
  }
}

/* ===== 质押赎回（解除质押锁定，可用余额恢复） ===== */
const redeemVisible = ref(false)
const redeemNo = ref('')
const activePledges = ref([])
const loadingRedeemList = ref(false)
const loadingRedeem = ref(false)
const selectedPledge = computed(() => activePledges.value.find(p => p.pledge_no === redeemNo.value) || null)
/* 名下存在质押中的积分才允许打开赎回弹窗(前置校验) */
const hasActivePledge = computed(() => Number(stats.value.pledged_credits || 0) > 0)

async function loadActivePledges() {
  loadingRedeemList.value = true
  try {
    const res = await rwaAPI.pledges({ page: 1, page_size: 50 })
    const list = res.data?.list || res.data || []
    activePledges.value = list.filter(p => p.status === 'active')
  } catch {
    activePledges.value = []
  } finally {
    loadingRedeemList.value = false
  }
}

function openRedeemDialog() {
  if (!hasActivePledge.value) {
    ElMessage.warning('当前无质押中的积分，无法赎回')
    return
  }
  redeemNo.value = ''
  redeemVisible.value = true
  loadActivePledges()
}

async function doRedeem() {
  if (!redeemNo.value) {
    ElMessage.warning('请先选择要赎回的质押单')
    return
  }
  loadingRedeem.value = true
  try {
    await rwaAPI.redeemPledge({ pledge_no: redeemNo.value })
    ElMessage.success(`质押 ${redeemNo.value} 已赎回，锁定积分已恢复可用`)
    redeemVisible.value = false
    await refreshAll(true)
  } catch (e) {
    ElMessage.error('质押赎回失败：' + (e?.msg || '请稍后重试'))
  } finally {
    loadingRedeem.value = false
  }
}

/* ===== 链上钱包：余额读取 / 质押 / 转出（MetaMask，仅用户手动点击连接） ===== */
const wallet = useWalletStore()
const walletAssets = reactive({ credits: '--', creditsSource: '连接钱包后读取', eth: '--', pledged: 0 })
const walletTxs = ref([])
const pledgeVisible = ref(false)
const transferVisible = ref(false)
const loadingPledge = ref(false)
const loadingTransfer = ref(false)
const pledgeForm = reactive({ amount: 0, term: 6 })
const transferForm = reactive({ to: '', amount: 0 })

/* 质押 / 转出上限：平台可用余额（合约未部署阶段作为映射上限，避免 InputNumber min>max 崩溃） */
const maxPledgeable = computed(() => Math.max(1, Number(stats.value.available_credits || availableForSellTotal.value || 0)))

/* 钱包就绪状态变化：就绪 → 读取链上资产；断开 / 切离 Sepolia → 清空展示 */
watch(() => wallet.isReady, ready => ready ? loadWalletAssets() : resetWalletAssets())

/** 读取链上资产：Sepolia ETH 余额 + 碳积分合约 balanceOf（预留 ABI 入口，未部署时映射平台余额） */
async function loadWalletAssets() {
  if (!wallet.address) return
  try {
    walletAssets.eth = await getNativeBalance(wallet.address)
  } catch {
    walletAssets.eth = '--'
  }
  try {
    const { address, methods } = CARBON_CONTRACTS.carbonCreditToken
    const data = abiEncodeCall(methods.balanceOf, [abiEncodeWord(wallet.address)])
    const res = await callContractRead(address, data)
    if (res && res !== '0x' && res.length >= 66) {
      walletAssets.credits = Number(BigInt(res)).toFixed(2)
      walletAssets.creditsSource = '链上合约余额'
    } else {
      throw new Error('empty')
    }
  } catch {
    walletAssets.credits = Number(stats.value.available_credits || 0).toFixed(2)
    walletAssets.creditsSource = '合约未部署 · 映射平台余额'
  }
}

function resetWalletAssets() {
  walletAssets.credits = '--'
  walletAssets.creditsSource = '连接钱包后读取'
  walletAssets.eth = '--'
  walletAssets.pledged = 0
}

/** 记录一笔链上操作（展示交易哈希，供演示讲解） */
function pushWalletTx(label, hash) {
  walletTxs.value.unshift({ label, hash, time: new Date().toLocaleTimeString() })
}

function openPledgeDialog() {
  pledgeForm.amount = Math.min(100, maxPledgeable.value)
  pledgeVisible.value = true
}

/** 积分质押：调用碳积分合约 lockForOrder(锁定质押数量)，唤起 MetaMask 签名确认 */
async function doPledge() {
  if (!wallet.isReady) { ElMessage.warning('请先连接钱包并切换到 Sepolia 测试网'); return }
  loadingPledge.value = true
  try {
    const { address, methods } = CARBON_CONTRACTS.carbonCreditToken
    // 合约写入口：lockForOrder(address,uint256,bytes32)（真实选择器 0x4f3b75b6），data 字段 ABI 编码
    const data = abiEncodeCall(methods.lockForOrder, [
      abiEncodeWord(wallet.address),       // from：签名人自己的地址
      abiEncodeUint256(pledgeForm.amount), // amount：质押锁定数量
      abiEncodeUint256(0),                 // orderId(bytes32)：质押授权阶段用 0
    ])
    const txHash = await sendContractTransaction(wallet.address, address, data)
    walletAssets.pledged = Number((walletAssets.pledged + Number(pledgeForm.amount || 0)).toFixed(2))
    pushWalletTx(`质押 ${pledgeForm.amount} 积分`, txHash)
    ElMessage.success('质押交易已提交：' + txHash.slice(0, 14) + '...')
    pledgeVisible.value = false
  } catch (e) {
    ElMessage.error(formatWalletError(e))
  } finally {
    loadingPledge.value = false
  }
}

function openTransferDialog() {
  transferForm.amount = Math.min(50, maxPledgeable.value)
  transferForm.to = ''
  transferVisible.value = true
}

/** 积分转出：调用碳积分合约 transfer(address,uint256)，唤起 MetaMask 签名确认 */
async function doTransfer() {
  if (!wallet.isReady) { ElMessage.warning('请先连接钱包并切换到 Sepolia 测试网'); return }
  if (!/^0x[0-9a-fA-F]{40}$/.test(transferForm.to || '')) {
    ElMessage.warning('收款地址格式不正确，应为 0x 开头的 40 位十六进制')
    return
  }
  loadingTransfer.value = true
  try {
    const { address, methods } = CARBON_CONTRACTS.carbonCreditToken
    // 预留合约写入口：transfer(address 收款方, uint256 数量)
    const data = abiEncodeCall(methods.transfer, [abiEncodeWord(transferForm.to), abiEncodeUint256(transferForm.amount)])
    const txHash = await sendContractTransaction(wallet.address, address, data)
    pushWalletTx(`转出 ${transferForm.amount} 积分`, txHash)
    ElMessage.success('转出交易已提交：' + txHash.slice(0, 14) + '...')
    transferVisible.value = false
  } catch (e) {
    ElMessage.error(formatWalletError(e))
  } finally {
    loadingTransfer.value = false
  }
}

/* ===== 图表 ===== */
/* 近12个月余额变化：数据来自后端 /carbon/stats 的 monthly_balance（month/balance），统计变动自动重绘 */
const balanceTrendOption = computed(() => {
  const points = stats.value.monthly_balance || []
  return {
    color: ['#10b981'],
    tooltip: { trigger: 'axis' },
    grid: { left: 10, right: 10, top: 16, bottom: 0, containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: points.map(p => p.month),
      axisLine: { lineStyle: { color: '#e2e8f0' } }, axisLabel: { color: '#94a3b8', fontSize: 11 }, axisTick: { show: false } },
    yAxis: { type: 'value', splitLine: { lineStyle: { color: '#f1f5f9' } }, axisLabel: { color: '#94a3b8', fontSize: 11 } },
    series: [{
      type: 'line', smooth: true, showSymbol: false,
      lineStyle: { width: 2.5 },
      areaStyle: { type: 'linear', x:0, y:0, x2:0, y2:1,
        colorStops: [
          { offset: 0, color: 'rgba(16,185,129,0.25)' },
          { offset: 1, color: 'rgba(16,185,129,0.02)' }
        ]
      },
      data: points.map(p => p.balance)
    }]
  }
})

/* 积分来源构成：后端实时聚合（自主核算/交易受让），随统计卡同步更新 */
const sourcePieOption = computed(() => ({
  tooltip: { trigger: 'item' },
  legend: { orient: 'vertical', right: 0, top: 'middle', textStyle: { fontSize: 11 } },
  series: [{
    type: 'pie', radius: ['42%', '68%'], center: ['38%', '50%'],
    itemStyle: { borderRadius: 4, borderWidth: 2 },
    label: { show: false }, labelLine: { show: false },
    data: [
      { value: stats.value.source_self_credits || 0, name: '自主核算' },
      { value: stats.value.source_transferred_credits || 0, name: '交易受让' },
    ]
  }]
}))

onMounted(() => refreshAll())

/* 页面被再次激活(keep-alive 缓存后重新进入)时同样拉取最新数据，保证统计/图表/记录实时刷新 */
onActivated(() => refreshAll())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.cc-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 22px;
  margin-bottom: 16px;
  background: linear-gradient(135deg, var(--primary-green), #16a34acc);
  border-radius: var(--radius-lg);
  color: #fff;
}
.dark .cc-head { background: linear-gradient(135deg, #0d9488, #06b6d4aa); }
.cc-head h2 { font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.cc-title p { font-size: 12px; margin: 0; color: rgba(255,255,255,0.85); }
.cc-action { display: flex; gap: 10px; }

/* ===== 余额卡 ===== */
.balance-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}
.bal-card {
  display: flex; gap: 12px;
  padding: 16px;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius);
  transition: all 0.2s;
}
.bal-card:hover { transform: translateY(-2px); border-color: var(--primary-green); box-shadow: 0 8px 20px rgba(0,0,0,0.08); }
.bal-icon { width: 42px; height: 42px; border-radius: 10px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.bal-num { font-size: 22px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary); }
.bal-label { font-size: 11.5px; color: var(--text-tertiary); margin-top: 2px; }

/* ===== 图表 ===== */
.charts-row { display: grid; grid-template-columns: 2fr 1fr; gap: 16px; margin-bottom: 16px; }
.chart-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 16px 20px;
}
.cc-chart-head h3 { font-size: 13px; font-weight: 600; color: var(--text-primary); margin: 0 0 12px; }

/* ===== 列表 ===== */
.list-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.list-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.list-head h3 { font-size: 14px; font-weight: 600; margin: 0; }
.list-filters { display: flex; gap: 8px; }

.mono { font-family: ui-monospace, Consolas, monospace; font-size: 12px; }
.mono-sm { font-family: ui-monospace, Consolas, monospace; font-size: 11px; word-break: break-all; }
.credit-val { color: var(--primary-green); font-weight: 700; font-size: 14px; }
.sell-hint { font-size: 11px; color: var(--text-tertiary); margin-left: 6px; }

.status-tag.available { background: rgba(16,185,129,0.1); color: #10b981; }
.status-tag.listed { background: rgba(16,185,129,0.1); color: #2563eb; }
.status-tag.sold { background: rgba(124,58,237,0.1); color: #7c3aed; }
.status-tag.expired { background: rgba(148,163,184,0.1); color: #64748b; }
.status-tag.used { background: rgba(148,163,184,0.1); color: #64748b; }
.status-tag.pending { background: rgba(245,158,11,0.1); color: #d97706; }

.op-empty { color: #cbd5e1; }

/* ===== 非小微企业角色提示卡 ===== */
.no-perm-card {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 10px; padding: 72px 24px;
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius-lg); text-align: center;
}
.no-perm-card .np-icon { color: var(--primary-green); opacity: 0.6; }
.no-perm-card h3 { margin: 0; font-size: 16px; font-weight: 600; color: var(--text-primary); }
.no-perm-card p { margin: 0; font-size: 13px; color: var(--text-tertiary); }

/* ===== 空资产提示 ===== */
.empty-asset-alert { margin-bottom: 16px; }

/* ===== 链上钱包卡 ===== */
.wallet-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 14px 18px; margin-bottom: 16px;
  border-left: 4px solid var(--primary-green);
}
.wallet-assets { display: flex; align-items: center; gap: 28px; margin-top: 12px; flex-wrap: wrap; }
.wa-item { display: flex; flex-direction: column; gap: 2px; min-width: 120px; }
.wa-label { font-size: 11px; color: var(--text-tertiary); }
.wa-val { font-size: 18px; font-weight: 700; color: var(--text-primary); font-variant-numeric: tabular-nums; }
.wa-sub { font-size: 10px; color: var(--text-tertiary); }
.wa-actions { margin-left: auto; display: flex; gap: 8px; }
.wallet-tx-log {
  margin-top: 10px; padding-top: 10px; border-top: 1px dashed var(--line-color);
  display: flex; gap: 12px; flex-wrap: wrap; font-size: 11px; color: var(--text-tertiary);
}
.wt-title { flex-shrink: 0; }
.wt-item { font-family: ui-monospace, Consolas, monospace; color: var(--primary-green); }

@media (max-width: 900px) {
  .charts-row { grid-template-columns: 1fr; }
}
</style>