<!--
  exchange.vue - 碳交易所
  功能：
    1. el-tabs 三标签（现货 / 租赁 / 远期）
    2. 卡片式挂单列表，企业使用可控匿名名称显示
    3. 撮合弹窗：选买方 → 核验积分来源 → 撮合成交
    4. 成交记录表格
    5. 纠纷订单带仲裁发起按钮
  后端接口：
    GET  /api/exchange/orders          卖方挂单
    GET  /api/exchange/buyers          可作买方的企业
    POST /api/exchange/match           撮合
    POST /api/exchange/verify-credit   核验积分来源
    GET  /api/exchange/transactions    成交记录
    GET  /api/rwa/trade-orders?type=spot|lease|forward  RWA 按类型挂单列表
    POST /api/rwa/lease-order    发布租赁挂单(绑定碳积分凭证+租期要素)
    POST /api/rwa/forward-order  发布远期挂单(绑定碳积分凭证+交割要素)
-->
<template>
  <div class="exchange page">

    <!-- ===== 页面头部：深色大屏风格（交易所默认深色主题） ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>碳交易所 · 撮合大屏</h2>
        <p class="ph-desc">撮合挂单 · 链上留痕 · 纠纷仲裁 · 积分来源可溯源</p>
      </div>
      <div class="ph-stats">
        <div class="ph-stat">
          <span class="ps-num">{{ totalOrders }}</span>
          <span class="ps-label">总挂单</span>
        </div>
        <div class="ph-stat">
          <span class="ps-num">{{ todayVolume }}</span>
          <span class="ps-label">今日成交</span>
        </div>
        <div class="ph-stat">
          <span class="ps-num">{{ totalBuyers }}</span>
          <span class="ps-label">买方企业</span>
        </div>
      </div>
    </div>

    <!-- ===== 链上钱包（手动连接 → 展示本人可交易碳积分 + 链上挂单） ===== -->
    <div class="wallet-card">
      <WalletConnect hint="碳交易所 · 链上钱包（可交易积分 / 挂单授权）" />
      <div class="wallet-assets">
        <div class="wa-item">
          <span class="wa-label">本人可交易碳积分</span>
          <span class="wa-val">{{ walletAssets.tradable }}</span>
          <span class="wa-sub">{{ walletAssets.tradableSource }}</span>
        </div>
        <div class="wa-item">
          <span class="wa-label">Sepolia ETH 余额</span>
          <span class="wa-val">{{ walletAssets.eth }}</span>
        </div>
        <div class="wa-actions">
          <el-button type="primary" size="small" :icon="Sell" :disabled="!wallet.isReady" @click="openListDialog()">
            链上挂单出售
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

    <!-- ===== 标签页：现货 / 租赁 / 远期 ===== -->
    <div class="tab-card">
      <el-tabs v-model="activeTab" class="exchange-tabs">
        <el-tab-pane label="现货交易" name="spot">
          <div class="tab-banner">
            <el-icon :size="14"><TrendCharts /></el-icon>
            即时撮合，T+0 结算，挂单有效期 48 小时
          </div>
        </el-tab-pane>
        <el-tab-pane label="碳积分租赁" name="lease">
          <div class="tab-banner">
            <el-icon :size="14"><Clock /></el-icon>
            周期性租赁，按日/周结算，适合季节性排放企业
          </div>
        </el-tab-pane>
        <el-tab-pane label="远期合约" name="forward">
          <div class="tab-banner">
            <el-icon :size="14"><DataAnalysis /></el-icon>
            锁定未来价格，降低履约风险，挂单有效期 30 天
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- 操作区：刷新 + 发布挂单 -->
      <div class="tab-actions">
        <el-button :icon="Refresh" @click="refreshAll" :loading="loadingAll">刷新数据</el-button>
        <el-button type="primary" :icon="Sell" @click="openTypedOrderDialog()">发布{{ activeTabLabel }}挂单</el-button>
        <el-tag v-if="activeTab === 'spot'" type="success" effect="plain" size="small">实时撮合</el-tag>
        <el-tag v-if="activeTab === 'lease'" type="warning" effect="plain" size="small">周期性结算</el-tag>
        <el-tag v-if="activeTab === 'forward'" type="danger" effect="plain" size="small">锁定价格</el-tag>
      </div>
    </div>

    <!-- ===== 挂单卡片列表 ===== -->
    <div class="orders-card">
      <div class="oc-head">
        <h3>挂单市场 · {{ activeTabLabel }} 型</h3>
        <span class="oc-sub">{{ filteredOrders.length }} 条挂单</span>
      </div>
      <div v-if="loadingOrders" class="loading-wrap">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
      </div>
      <div v-else-if="!filteredOrders.length" class="empty-wrap">
        <el-icon :size="36"><TrendCharts /></el-icon>
        <p>暂无挂单，请等待企业挂单或切换标签查看</p>
      </div>
      <div v-else class="orders-grid">
        <div
          v-for="o in filteredOrders"
          :key="o.id"
          class="order-card"
          :class="{ disputed: o.disputed }"
        >
          <!-- 卡片头部 -->
          <div class="oc-top">
            <span class="oc-type-tag" :class="activeTab">
              {{ activeTabLabel }}
            </span>
            <span class="oc-status" :class="o.status || 'pending'">
              {{ statusLabel[o.status] || '挂单中' }}
            </span>
          </div>

          <!-- 核心数据 -->
          <div class="oc-main">
            <div class="oc-price">
              <span class="currency">¥</span>
              <span class="val">{{ o.unit_price?.toFixed(2) || '--' }}</span>
              <span class="unit">{{ activeTab === 'lease' ? '/积分·期' : '/积分' }}</span>
            </div>
            <div class="oc-qty">
              <span class="qty-num">{{ o.quantity || 0 }}</span>
              <span class="qty-unit">积分</span>
            </div>
          </div>

          <!-- 类型专属信息：租赁租期 / 远期交割 -->
          <div v-if="activeTab === 'lease'" class="oc-extra">
            <span>租期 {{ o.lease_start_date || '--' }} ~ {{ o.lease_end_date || '--' }}</span>
            <span>{{ leaseCycleLabel(o.lease_cycle) }} · {{ leaseReturnLabel(o.return_rule) }}</span>
          </div>
          <div v-else-if="activeTab === 'forward'" class="oc-extra">
            <span>交割日 {{ o.delivery_date || '--' }}</span>
            <span>有效期至 {{ formatDate(o.expire_at) }}</span>
          </div>

          <!-- 卖方信息（可控匿名） -->
          <div class="oc-seller">
            <div class="seller-avatar">{{ (o.seller_name || enterpriseAnon(o.enterprise_id)).charAt(0) }}</div>
            <div class="seller-info">
              <div class="seller-name">{{ enterpriseAnon(o.enterprise_id) }}</div>
              <div class="seller-meta">
                <el-icon :size="11"><Lock /></el-icon>
                <span>匿名可溯源</span>
                <span class="divider">·</span>
                <span>{{ o.enterprise?.park_id ? '园区' + o.enterprise.park_id : '--' }}</span>
              </div>
            </div>
          </div>

          <!-- 链上信息 -->
          <div class="oc-chain">
            <el-icon :size="12"><Connection /></el-icon>
            <code>{{ o.order_no || '--' }}</code>
          </div>

          <!-- 纠纷提示 -->
          <div v-if="o.disputed" class="oc-dispute">
            <el-icon :size="12"><Warning /></el-icon>
            纠纷处理中
          </div>

          <!-- 操作按钮 -->
          <div class="oc-actions">
            <el-button size="small" type="primary" @click="openMatch(o)" :disabled="o.status !== 'pending'">
              撮合成交
            </el-button>
            <el-button
              v-if="o.disputed" size="small" type="warning"
              @click="goArbitration(o)"
            >
              <el-icon><ScaleToOriginal /></el-icon> 仲裁
            </el-button>
            <el-button size="small" @click="verifySource(o)">
              <el-icon><Lock /></el-icon> 核验来源
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 成交记录表格 ===== -->
    <div class="tx-card">
      <div class="tx-head">
        <h3>成交记录（最近 50 笔）</h3>
        <el-button link type="primary" @click="loadTransactions">刷新</el-button>
      </div>
      <el-table :data="transactions" stripe style="width: 100%" empty-text="暂无成交记录">
        <el-table-column prop="order_no" label="挂单编号" width="180">
          <template #default="{ row }"><code class="mono">{{ row.order_no }}</code></template>
        </el-table-column>
        <el-table-column label="卖方" width="140">
          <template #default="{ row }">{{ enterpriseAnon(row.seller_id) }}</template>
        </el-table-column>
        <el-table-column label="买方" width="140">
          <template #default="{ row }">{{ enterpriseAnon(row.buyer_id) }}</template>
        </el-table-column>
        <el-table-column label="积分" width="100" align="right">
          <template #default="{ row }">{{ row.quantity }}</template>
        </el-table-column>
        <el-table-column label="总价(元)" width="110" align="right">
          <template #default="{ row }">{{ row.total_amount?.toFixed(2) || '--' }}</template>
        </el-table-column>
        <el-table-column label="链上" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" size="small" type="success">已上链</el-tag>
            <el-tag v-else size="small">--</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="成交时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== 撮合弹窗 ===== -->
    <el-dialog v-model="matchVisible" width="520px" class="match-dialog">
      <template #header>
        <div class="dlg-head">
          <el-icon :size="20"><Handshake /></el-icon>
          <span>交易撮合</span>
        </div>
      </template>
      <div v-if="matchingOrder" class="match-body">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="挂单编号"><code>{{ matchingOrder.order_no }}</code></el-descriptions-item>
          <el-descriptions-item label="卖方">{{ enterpriseAnon(matchingOrder.enterprise_id) }}</el-descriptions-item>
          <el-descriptions-item label="数量">{{ matchingOrder.quantity }} 积分</el-descriptions-item>
          <el-descriptions-item label="单价">¥ {{ matchingOrder.unit_price?.toFixed(2) }}</el-descriptions-item>
          <el-descriptions-item label="碳积分编号" :span="2">
            <code v-if="matchCreditNo" class="credit-no-ro">{{ matchCreditNo }}</code>
            <el-tag v-else type="danger" size="small" effect="plain">凭证编号缺失</el-tag>
          </el-descriptions-item>
          <!-- 租赁/远期挂单专属要素 -->
          <template v-if="matchingOrder.order_type === 'lease'">
            <el-descriptions-item label="租赁起止" :span="2">
              {{ matchingOrder.lease_start_date || '--' }} ~ {{ matchingOrder.lease_end_date || '--' }}
            </el-descriptions-item>
            <el-descriptions-item label="租赁周期">{{ leaseCycleLabel(matchingOrder.lease_cycle) }}</el-descriptions-item>
            <el-descriptions-item label="归还规则">{{ leaseReturnLabel(matchingOrder.return_rule) }}</el-descriptions-item>
          </template>
          <template v-else-if="matchingOrder.order_type === 'forward'">
            <el-descriptions-item label="约定交割日">{{ matchingOrder.delivery_date || '--' }}</el-descriptions-item>
            <el-descriptions-item label="挂单有效期">30天（至 {{ formatDate(matchingOrder.expire_at) }}）</el-descriptions-item>
          </template>
        </el-descriptions>

        <el-alert
          v-if="!matchCreditNo"
          type="warning" :closable="false" show-icon
          title="该挂单碳积分凭证编号缺失，无法撮合"
        />

        <div class="buyer-select-wrap">
          <label>指定买方企业</label>
          <el-select v-model="matchBuyerId" placeholder="请选择买方企业" style="width: 100%">
            <el-option
              v-for="b in buyers" :key="b.id"
              :value="b.id"
              :label="`${b.company || b.username}（园区${b.park_id || '-'}）`"
            />
          </el-select>
        </div>

        <el-alert
          v-if="creditVerified"
          type="success" :closable="false"
          title="✅ 积分来源已核验，可安全成交"
          show-icon
        />
        <el-alert
          v-else-if="creditVerifyMsg"
          type="error" :closable="false"
          :title="'⚠️ ' + creditVerifyMsg"
          show-icon
        />
      </div>
      <template #footer>
        <el-button @click="matchVisible = false">取消</el-button>
        <el-button
          type="warning" :icon="Lock"
          :disabled="!matchCreditNo"
          @click="verifySource(matchingOrder, true)" :loading="loadingVerify"
        >
          先核验积分来源
        </el-button>
        <el-button
          type="primary" :icon="Connection"
          :loading="loadingMatch"
          :disabled="!matchBuyerId || !matchCreditNo"
          @click="doMatch"
        >
          确认撮合并上链
        </el-button>
      </template>
    </el-dialog>

    <!-- ===== 链上挂单弹窗·现货（唤起 MetaMask 签名授权） ===== -->
    <el-dialog v-model="listVisible" width="440px" title="链上挂单出售碳积分（现货）">
      <el-form label-width="100px">
        <el-form-item label="碳积分凭证">
          <el-select
            v-model="listForm.credit_id"
            placeholder="请选择本人持有的碳积分凭证"
            style="width:100%" :loading="loadingMyCredits"
            @change="onCreditPicked"
          >
            <el-option
              v-for="c in myAvailableCredits" :key="c.id"
              :value="c.id"
              :label="`${c.credit_no}（${c.carbon_credits} 积分）`"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="pickedCredit" label="凭证编号">
          <code class="credit-no-ro">{{ pickedCredit.credit_no }}</code>
        </el-form-item>
        <el-form-item label="可交易积分">
          <span class="wa-val" style="font-size:16px">{{ walletAssets.tradable }}</span>
        </el-form-item>
        <el-form-item label="挂单数量">
          <el-input-number v-model="listForm.quantity" :min="1" :max="Math.max(maxListable, 1)" :precision="0" style="width:100%" />
        </el-form-item>
        <el-form-item label="单价 (元)">
          <el-input-number v-model="listForm.unit_price" :min="1" :max="200" :precision="2" style="width:100%" />
        </el-form-item>
        <el-alert type="info" :closable="false" show-icon title="确认后将唤起 MetaMask 签名，调用碳积分合约锁定挂单积分" />
      </el-form>
      <template #footer>
        <el-button @click="listVisible = false">取消</el-button>
        <el-button
          type="primary" :loading="loadingListOrder"
          :disabled="!listForm.credit_id || !listForm.quantity"
          @click="doListOrder"
        >
          签名并挂单
        </el-button>
      </template>
    </el-dialog>

    <!-- ===== 发布租赁挂单弹窗（使用权出租，到期归还卖方，所有权不变） ===== -->
    <el-dialog v-model="leaseVisible" width="460px" title="发布租赁挂单 · 碳积分租赁">
      <el-alert type="info" :closable="false" show-icon title="租赁仅出租碳积分使用权，到期归还卖方，所有权不变" style="margin-bottom:14px" />
      <el-form label-width="110px">
        <el-form-item label="碳积分凭证" required>
          <el-select
            v-model="leaseForm.credit_id"
            placeholder="请选择本人持有的碳积分凭证"
            style="width:100%" :loading="loadingMyCredits"
            @change="onLeaseCreditPicked"
          >
            <el-option
              v-for="c in myAvailableCredits" :key="c.id"
              :value="c.id"
              :label="`${c.credit_no}（${c.carbon_credits} 积分）`"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="pickedLeaseCredit" label="凭证编号">
          <code class="credit-no-ro">{{ pickedLeaseCredit.credit_no }}</code>
        </el-form-item>
        <el-form-item label="租赁积分数量">
          <el-input-number v-model="leaseForm.quantity" :min="1" :max="Math.max(maxLeaseable, 1)" :precision="0" style="width:100%" />
        </el-form-item>
        <el-form-item label="每期租金(元)">
          <el-input-number v-model="leaseForm.unit_price" :min="1" :max="200" :precision="2" style="width:100%" />
        </el-form-item>
        <el-form-item label="租赁开始日期" required>
          <el-date-picker
            v-model="leaseForm.lease_start_date" type="date" value-format="YYYY-MM-DD"
            placeholder="选择开始日期" style="width:100%"
            :disabled-date="d => d.getTime() < Date.now() - 86400000"
          />
        </el-form-item>
        <el-form-item label="租赁结束日期" required>
          <el-date-picker
            v-model="leaseForm.lease_end_date" type="date" value-format="YYYY-MM-DD"
            placeholder="选择结束日期" style="width:100%"
            :disabled-date="d => !!leaseForm.lease_start_date && d.getTime() < new Date(leaseForm.lease_start_date).getTime()"
          />
        </el-form-item>
        <el-form-item label="租赁周期">
          <el-select v-model="leaseForm.lease_cycle" style="width:100%">
            <el-option label="按日结算" value="day" />
            <el-option label="按周结算" value="week" />
            <el-option label="按月结算" value="month" />
          </el-select>
        </el-form-item>
        <el-form-item label="归还规则">
          <el-select v-model="leaseForm.return_rule" style="width:100%">
            <el-option label="到期自动归还（积分回到卖方可用余额）" value="auto" />
            <el-option label="到期人工确认归还" value="manual" />
          </el-select>
        </el-form-item>
        <el-alert type="info" :closable="false" show-icon title="确认后将唤起 MetaMask 签名，锁定挂单积分，到期按归还规则处理" />
      </el-form>
      <template #footer>
        <el-button @click="leaseVisible = false">取消</el-button>
        <el-button
          type="primary" :loading="loadingListOrder"
          :disabled="!leaseForm.credit_id || !leaseForm.quantity || !leaseForm.lease_start_date || !leaseForm.lease_end_date"
          @click="doLeaseOrder"
        >
          签名并发布租赁挂单
        </el-button>
      </template>
    </el-dialog>

    <!-- ===== 发布远期挂单弹窗（锁定远期价格，约定未来交割） ===== -->
    <el-dialog v-model="forwardVisible" width="460px" title="发布远期挂单 · 远期合约">
      <el-alert type="info" :closable="false" show-icon title="现在锁定远期价格，约定在未来指定日期完成碳积分交割上链" style="margin-bottom:14px" />
      <el-form label-width="110px">
        <el-form-item label="碳积分凭证" required>
          <el-select
            v-model="forwardForm.credit_id"
            placeholder="请选择本人持有的碳积分凭证"
            style="width:100%" :loading="loadingMyCredits"
            @change="onForwardCreditPicked"
          >
            <el-option
              v-for="c in myAvailableCredits" :key="c.id"
              :value="c.id"
              :label="`${c.credit_no}（${c.carbon_credits} 积分）`"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="pickedForwardCredit" label="凭证编号">
          <code class="credit-no-ro">{{ pickedForwardCredit.credit_no }}</code>
        </el-form-item>
        <el-form-item label="积分数量">
          <el-input-number v-model="forwardForm.quantity" :min="1" :max="Math.max(maxForwardable, 1)" :precision="0" style="width:100%" />
        </el-form-item>
        <el-form-item label="远期约定单价">
          <el-input-number v-model="forwardForm.unit_price" :min="1" :max="200" :precision="2" style="width:100%" />
        </el-form-item>
        <el-form-item label="交割日期" required>
          <el-date-picker
            v-model="forwardForm.delivery_date" type="date" value-format="YYYY-MM-DD"
            placeholder="选择约定交割日期" style="width:100%"
            :disabled-date="d => d.getTime() < Date.now()"
          />
        </el-form-item>
        <el-form-item label="挂单有效期">
          <span class="ro-text">30 天（发布后由服务端自动计算）</span>
        </el-form-item>
        <el-alert type="info" :closable="false" show-icon title="确认后将唤起 MetaMask 签名，锁定挂单积分，交割日完成交割上链" />
      </el-form>
      <template #footer>
        <el-button @click="forwardVisible = false">取消</el-button>
        <el-button
          type="primary" :loading="loadingListOrder"
          :disabled="!forwardForm.credit_id || !forwardForm.quantity || !forwardForm.delivery_date"
          @click="doForwardOrder"
        >
          签名并发布远期挂单
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  TrendCharts, Clock, DataAnalysis, Refresh, Loading, Lock, Connection,
  Warning, ScaleToOriginal, Sell
} from '@element-plus/icons-vue'
import { exchangeAPI, rwaAPI, carbonAPI } from '../api/index.js'
import WalletConnect from '../components/WalletConnect.vue'
import { useWalletStore } from '../stores/wallet.js'
import {
  getNativeBalance, callContractRead, sendContractTransaction, formatWalletError,
  getChainId, SEPOLIA, withWalletWatchdog,
  isUsableContractAddress, loadDeployedContracts,
  CARBON_CONTRACTS, abiEncodeCall, abiEncodeUint256, abiEncodeWord,
} from '../utils/web3.js'

const router = useRouter()

/* ===== 标签 ===== */
const activeTab = ref('spot')
const activeTabLabelMap = { spot: '现货', lease: '租赁', forward: '远期' }
const activeTabLabel = computed(() => activeTabLabelMap[activeTab.value])

/* ===== 订单 & 成交 ===== */
const orders = ref([])
const transactions = ref([])
const buyers = ref([])
const loadingOrders = ref(false)
const loadingAll = ref(false)

/* ===== 撮合 ===== */
const matchVisible = ref(false)
const matchingOrder = ref(null)
const matchBuyerId = ref(null)
const matchCreditNo = ref('')  // 挂单关联的碳积分凭证编号(只读回填，撮合/核验必传)
const loadingMatch = ref(false)
const loadingVerify = ref(false)
const creditVerified = ref(false)
const creditVerifyMsg = ref('')

/* 订单状态映射 */
const statusLabel = { pending: '挂单中', matched: '已撮合', completed: '已成交', cancelled: '已撤销' }

/* 概览统计 */
const totalOrders = computed(() => orders.value.length)
const todayVolume = computed(() => transactions.value.reduce((s, t) => s + (t.total_amount || 0), 0).toFixed(0))
const totalBuyers = computed(() => buyers.value.length)

/* 按标签类型过滤(现货/租赁/远期均为后端真实数据，无 mock 兜底) */
const filteredOrders = computed(() => orders.value.filter(o => o.order_type === activeTab.value))

/* 企业匿名名称（可控匿名，基于 enterprise_id 哈希） */
function enterpriseAnon(id) {
  if (!id) return '匿名企业'
  const seed = String(id).split('').reduce((a, c) => a + c.charCodeAt(0), 0)
  const prefixes = ['绿能星','低碳智','环创','清润','绿盾','蓝天','清源','华兴']
  const suffixes = ['科技','生态','智造','能源','环保','创新','节能','循环']
  return `${prefixes[seed % prefixes.length]}·${suffixes[(seed + 3) % suffixes.length]}-${String(id).padStart(3,'0')}`
}

/* 加载挂单：现货走交易所挂单接口，租赁/远期走 RWA 接口(统一标记 order_type) */
async function loadOrders() {
  loadingOrders.value = true
  try {
    const results = await Promise.allSettled([
      exchangeAPI.orders({ page: 1, page_size: 50 }),
      rwaAPI.tradeOrders({ type: 'lease', page_size: 50 }),
      rwaAPI.tradeOrders({ type: 'forward', page_size: 50 }),
    ])
    const pick = (r) => (r.status === 'fulfilled' ? (r.value.data?.list || r.value.data || []) : [])
    orders.value = [
      ...pick(results[0]).map(o => ({ ...o, order_type: o.order_type || 'spot' })),
      ...pick(results[1]).map(o => ({ ...o, order_type: 'lease' })),
      ...pick(results[2]).map(o => ({ ...o, order_type: 'forward' })),
    ]
  } catch {
    orders.value = []
  } finally {
    loadingOrders.value = false
  }
}

/* 加载买方企业（后端返回 {list:[...]} 包裹结构，需取 .list；否则 v-for 遍历对象导致无法选中买方） */
async function loadBuyers() {
  try {
    const res = await exchangeAPI.buyers({})
    buyers.value = res.data?.list || res.data || []
  } catch {
    buyers.value = []
  }
}

/* 加载成交记录 */
async function loadTransactions() {
  try {
    const res = await exchangeAPI.transactions({ page: 1, page_size: 50 })
    transactions.value = res.data?.list || res.data || []
  } catch { /* ignore */ }
}

async function refreshAll() {
  loadingAll.value = true
  await Promise.all([loadOrders(), loadBuyers(), loadTransactions(), loadPlatformTradable()])
  // 钱包已连接时同步刷新链上资产展示
  if (wallet.isReady) loadWalletAssets()
  loadingAll.value = false
  ElMessage.success('数据已刷新')
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

/* 租赁/远期展示辅助 */
const leaseCycleLabel = (c) => ({ day: '按日结算', week: '按周结算', month: '按月结算' }[c] || '按月结算')
const leaseReturnLabel = (r) => ({ auto: '到期自动归还', manual: '到期人工确认归还' }[r] || '到期自动归还')
function formatDate(t) {
  return formatTime(t).slice(0, 10)
}

/* ===== 撮合 ===== */
function openMatch(order) {
  matchingOrder.value = order
  matchBuyerId.value = null
  matchCreditNo.value = order.credit_no || ''  // 自动读取挂单对应的碳积分编号
  creditVerified.value = false
  creditVerifyMsg.value = ''
  matchVisible.value = true
}

async function verifySource(order, inDialog = false) {
  if (!order) return
  // 回填碳积分编号：列表数据缺失时从挂单字段兜底读取
  const creditNo = matchCreditNo.value || order.credit_no || ''
  if (inDialog) {
    if (!creditNo) {
      creditVerified.value = false
      creditVerifyMsg.value = '该挂单碳积分凭证编号缺失，无法核验'
      ElMessage.warning(creditVerifyMsg.value)
      return
    }
    loadingVerify.value = true
    try {
      const res = await exchangeAPI.verifyCredit({ credit_no: creditNo })
      matchCreditNo.value = creditNo
      creditVerified.value = true
      creditVerifyMsg.value = res.data?.msg || '积分来源合规'
      ElMessage.success('积分来源核验通过')
    } catch (e) {
      creditVerified.value = false
      creditVerifyMsg.value = e?.msg || '积分来源核验失败'
      ElMessage.error(creditVerifyMsg.value)
    } finally {
      loadingVerify.value = false
    }
  } else {
    if (!creditNo) {
      ElMessage.warning('该挂单碳积分凭证编号缺失，无法核验')
      return
    }
    try {
      const res = await exchangeAPI.verifyCredit({ credit_no: creditNo })
      ElMessage.success(res.data?.msg || '核验通过')
    } catch (e) {
      ElMessage.error(e?.msg || '核验失败')
    }
  }
}

/* ===== 链上钱包：本人可交易积分 + 链上挂单 / 撮合授权（MetaMask，仅用户手动连接） ===== */
const wallet = useWalletStore()
const walletAssets = reactive({ tradable: '--', tradableSource: '连接钱包后展示', eth: '--' })
const walletTxs = ref([])
const listVisible = ref(false)
const loadingListOrder = ref(false)
const listForm = reactive({ credit_id: null, quantity: 1, unit_price: 50 })
const platformTradable = ref(0)
/* 挂单凭证：本人持有的可用碳积分凭证列表(下拉选择，选中后绑定 CreditNo) */
const myAvailableCredits = ref([])
const loadingMyCredits = ref(false)

/* 现货挂单凭证绑定 */
const pickedCredit = computed(() => myAvailableCredits.value.find(c => c.id === listForm.credit_id) || null)
const maxListable = computed(() => Number(pickedCredit.value?.carbon_credits) || 0)

/* 租赁挂单：使用权出租，到期归还卖方，所有权不变 */
const leaseVisible = ref(false)
const leaseForm = reactive({
  credit_id: null, quantity: 1, unit_price: 50,
  lease_start_date: '', lease_end_date: '', lease_cycle: 'month', return_rule: 'auto',
})
const pickedLeaseCredit = computed(() => myAvailableCredits.value.find(c => c.id === leaseForm.credit_id) || null)
const maxLeaseable = computed(() => Number(pickedLeaseCredit.value?.carbon_credits) || 0)

/* 远期挂单：锁定远期价格，约定未来交割，有效期30天 */
const forwardVisible = ref(false)
const forwardForm = reactive({ credit_id: null, quantity: 1, unit_price: 50, delivery_date: '' })
const pickedForwardCredit = computed(() => myAvailableCredits.value.find(c => c.id === forwardForm.credit_id) || null)
const maxForwardable = computed(() => Number(pickedForwardCredit.value?.carbon_credits) || 0)

/* 钱包就绪状态变化：就绪 → 读取链上资产；断开 / 切离 Ganache → 清空展示 */
watch(() => wallet.isReady, ready => ready ? loadWalletAssets() : resetWalletAssets())

/** 从平台账本统计本人可交易（可用状态）碳积分 */
async function loadPlatformTradable() {
  try {
    const res = await carbonAPI.myCredits({ page: 1, page_size: 50 })
    const list = res.data?.list || res.data || []
    platformTradable.value = list
      .filter(c => c.status === 'available')
      .reduce((s, c) => s + (c.carbon_credits || 0), 0)
  } catch {
    platformTradable.value = 0
  }
}

/** 读取链上资产：ETH 余额 + 碳积分合约 balanceOf（预留 ABI 入口，未部署时映射平台余额） */
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
      walletAssets.tradable = Number(BigInt(res))
      walletAssets.tradableSource = '链上合约余额'
    } else {
      throw new Error('empty')
    }
  } catch {
    walletAssets.tradable = platformTradable.value
    walletAssets.tradableSource = '合约未部署 · 映射平台余额'
  }
}

function resetWalletAssets() {
  walletAssets.tradable = '--'
  walletAssets.tradableSource = '连接钱包后展示'
  walletAssets.eth = '--'
}

/** 记录一笔链上操作（展示交易哈希，供演示讲解） */
function pushWalletTx(label, hash) {
  walletTxs.value.unshift({ label, hash, time: new Date().toLocaleTimeString() })
}

/* ===== 挂单弹窗打开(现货/租赁/远期独立弹窗，凭证必选) ===== */
function openListDialog() {
  listForm.credit_id = null
  listForm.quantity = 1
  listVisible.value = true
  loadMyAvailableCredits()
}

function openLeaseDialog() {
  leaseForm.credit_id = null
  leaseForm.quantity = 1
  leaseForm.lease_start_date = ''
  leaseForm.lease_end_date = ''
  leaseVisible.value = true
  loadMyAvailableCredits()
}

function openForwardDialog() {
  forwardForm.credit_id = null
  forwardForm.quantity = 1
  forwardForm.delivery_date = ''
  forwardVisible.value = true
  loadMyAvailableCredits()
}

/** Tab操作区入口：按当前标签打开对应类型的挂单弹窗 */
function openTypedOrderDialog() {
  if (activeTab.value === 'lease') openLeaseDialog()
  else if (activeTab.value === 'forward') openForwardDialog()
  else openListDialog()
}

/** 加载本人持有的可用碳积分凭证(挂单下拉数据源，凭证列表从碳积分表读取) */
async function loadMyAvailableCredits() {
  loadingMyCredits.value = true
  try {
    const res = await carbonAPI.myCredits({ page: 1, page_size: 50 })
    const list = res.data?.list || res.data || []
    myAvailableCredits.value = list.filter(c => c.status === 'available')
  } catch {
    myAvailableCredits.value = []
  } finally {
    loadingMyCredits.value = false
  }
}

/** 选中凭证后：自动读取该凭证 CreditNo 并按其积分量重置挂单数量 */
function onCreditPicked() {
  if (pickedCredit.value) {
    listForm.quantity = Math.max(1, Math.floor(Number(pickedCredit.value.carbon_credits) || 1))
  }
}

function onLeaseCreditPicked() {
  if (pickedLeaseCredit.value) {
    leaseForm.quantity = Math.max(1, Math.floor(Number(pickedLeaseCredit.value.carbon_credits) || 1))
  }
}

function onForwardCreditPicked() {
  if (pickedForwardCredit.value) {
    forwardForm.quantity = Math.max(1, Math.floor(Number(pickedForwardCredit.value.carbon_credits) || 1))
  }
}

/** 链上签名锁定积分(现货/租赁/远期共用)：校验钱包/合约地址/网络后唤起 MetaMask 签名，返回交易哈希 */
async function signCreditLock(quantity) {
  if (!wallet.isReady) {
    ElMessage.warning('请先连接钱包并切换到 Sepolia 测试网')
    throw { handled: true }
  }
  // 加载 Sepolia 部署后的真实合约地址（deploy.js 产出 frontend/public/contracts.json）
  await loadDeployedContracts()
  const contractAddr = CARBON_CONTRACTS.carbonCreditToken.address
  console.log('[链上挂单] 当前使用的碳积分合约地址:', contractAddr)
  // 合约地址校验：为空 / 格式非法 / 仍为占位地址 → 不发起钱包签名
  if (!isUsableContractAddress(contractAddr)) {
    ElMessageBox.alert('合约地址未正确加载，请确认合约已部署到 Sepolia 测试网', '合约地址校验失败', {
      type: 'warning', confirmButtonText: '知道了',
    })
    throw { handled: true }
  }
  // 网络校验：chainId 必须为 11155111，否则阻止发起交易
  let chainId = 0
  try { chainId = await getChainId() } catch { chainId = 0 }
  if (chainId !== SEPOLIA.chainIdDecimal) {
    ElMessageBox.alert('请切换到 Sepolia 测试网(ChainID:11155111)', '钱包网络不正确', {
      type: 'warning', confirmButtonText: '知道了',
    })
    throw { handled: true }
  }
  // 合约写入口：lockForOrder(锁定挂单数量) —— 与已部署合约真实方法校准，唤起 MetaMask 签名授权
  const { methods } = CARBON_CONTRACTS.carbonCreditToken
  const data = abiEncodeCall(methods.lockForOrder, [
    abiEncodeWord(wallet.address),  // from：签名人自己的地址
    abiEncodeUint256(quantity),     // amount：挂单锁定积分数量
    abiEncodeUint256(0),            // orderId(bytes32)：链上授权阶段用 0，平台订单号由后端生成
  ])
  // 看门狗兜底：Blockaid 拦截等场景下签名请求可能永久挂起，超时强制结束
  const txHash = await withWalletWatchdog(sendContractTransaction(wallet.address, contractAddr, data))
  pushWalletTx(`挂单 ${quantity} 积分`, txHash)
  return txHash
}

/** 签名异常分类处理：用户取消 / 合约地址异常 / gas余额不足 / 其他未知错误 */
function handleSignError(e) {
  if (e?.handled) return
  const raw = String(e?.message || '') + ' ' + String(e?.shortMessage || '') + ' ' + String(e?.data?.message || '')
  if (e?.code === 4001 || /user denied|user rejected transaction/i.test(raw)) {
    ElMessageBox.alert('你取消了签名操作', '签名已取消', {
      type: 'info', confirmButtonText: '知道了',
    })
  } else if (!isUsableContractAddress(CARBON_CONTRACTS.carbonCreditToken.address)) {
    ElMessageBox.alert('合约地址加载异常，请检查部署配置', '合约地址异常', {
      type: 'error', confirmButtonText: '知道了',
    })
  } else if (/insufficient funds/i.test(raw)) {
    ElMessageBox.alert('钱包ETH余额不足，无法支付gas手续费，请先从水龙头领取 Sepolia 测试币', '余额不足', {
      type: 'error', confirmButtonText: '知道了',
    })
  } else {
    console.warn('[挂单签名异常]', formatWalletError(e))
    ElMessageBox.alert(formatWalletError(e), '签名失败', {
      type: 'error', confirmButtonText: '知道了',
    })
  }
}

/** 凭证必选校验(所有类型挂单：没有碳积分凭证编号不能发布挂单) */
function ensureCreditPicked(picked) {
  if (!picked) {
    ElMessage.warning('请先选择本人持有的碳积分凭证')
    return false
  }
  return true
}

/** 现货挂单：先签名锁定积分(MetaMask)，再提交平台现货挂单接口 */
async function doListOrder() {
  if (!ensureCreditPicked(pickedCredit.value)) return
  loadingListOrder.value = true
  try {
    const txHash = await signCreditLock(listForm.quantity)
    try {
      await carbonAPI.sell({
        credit_id: listForm.credit_id,   // 凭证ID：后端据此将 CreditNo 关联到挂单记录
        quantity: listForm.quantity,
        unit_price: listForm.unit_price,
      })
      ElMessage.success('现货挂单成功，链上授权哈希：' + txHash.slice(0, 14) + '...')
    } catch (e) {
      ElMessage.error('平台挂单创建失败：' + (e?.msg || '请检查碳积分状态后重试'))
      return  // 挂单未建成，保持弹窗打开让用户修正后重试
    }
    listVisible.value = false
    loadOrders()
  } catch (e) {
    handleSignError(e)
  } finally {
    // 无论成功、取消还是任何错误，强制关闭 loading 停止转圈
    loadingListOrder.value = false
  }
}

/** 租赁挂单：使用权出租到期归还(所有权不变)，绑定凭证+租期要素后走租赁挂单接口 */
async function doLeaseOrder() {
  if (!ensureCreditPicked(pickedLeaseCredit.value)) return
  if (!leaseForm.lease_start_date || !leaseForm.lease_end_date) {
    ElMessage.warning('请选择租赁起止日期')
    return
  }
  if (leaseForm.lease_end_date <= leaseForm.lease_start_date) {
    ElMessage.warning('租赁结束日期必须晚于开始日期')
    return
  }
  loadingListOrder.value = true
  try {
    const txHash = await signCreditLock(leaseForm.quantity)
    try {
      await rwaAPI.createLeaseOrder({
        credit_id: leaseForm.credit_id,   // 凭证ID：后端据此将 CreditNo 关联到挂单记录
        quantity: leaseForm.quantity,
        unit_price: leaseForm.unit_price,
        lease_start_date: leaseForm.lease_start_date,
        lease_end_date: leaseForm.lease_end_date,
        lease_cycle: leaseForm.lease_cycle,
        return_rule: leaseForm.return_rule,
      })
      ElMessage.success('租赁挂单发布成功，链上授权哈希：' + txHash.slice(0, 14) + '...')
    } catch (e) {
      ElMessage.error('平台租赁挂单创建失败：' + (e?.msg || '请检查碳积分状态后重试'))
      return  // 挂单未建成，保持弹窗打开让用户修正后重试
    }
    leaseVisible.value = false
    loadOrders()
  } catch (e) {
    handleSignError(e)
  } finally {
    loadingListOrder.value = false
  }
}

/** 远期挂单：锁定远期价格+约定交割日(有效期30天由服务端计算)，走远期挂单接口 */
async function doForwardOrder() {
  if (!ensureCreditPicked(pickedForwardCredit.value)) return
  if (!forwardForm.delivery_date) {
    ElMessage.warning('请选择约定交割日期')
    return
  }
  loadingListOrder.value = true
  try {
    const txHash = await signCreditLock(forwardForm.quantity)
    try {
      await rwaAPI.createForwardOrder({
        credit_id: forwardForm.credit_id, // 凭证ID：后端据此将 CreditNo 关联到挂单记录
        quantity: forwardForm.quantity,
        unit_price: forwardForm.unit_price,
        delivery_date: forwardForm.delivery_date,
      })
      ElMessage.success('远期挂单发布成功，链上授权哈希：' + txHash.slice(0, 14) + '...')
    } catch (e) {
      ElMessage.error('平台远期挂单创建失败：' + (e?.msg || '请检查碳积分状态后重试'))
      return  // 挂单未建成，保持弹窗打开让用户修正后重试
    }
    forwardVisible.value = false
    loadOrders()
  } catch (e) {
    handleSignError(e)
  } finally {
    loadingListOrder.value = false
  }
}

/** 撮合成交：钱包就绪时先做链上授权（积分转移至托管，MetaMask 签名），再进行平台撮合 */
async function doMatch() {
  if (!matchingOrder.value || !matchBuyerId.value) return
  if (!matchCreditNo.value) {
    ElMessage.warning('该挂单碳积分凭证编号缺失，无法撮合')
    return
  }

  // 链上授权：调用合约 transfer 将积分转移至交易所托管地址（预留合约入口）
  let authTxHash = ''
  if (wallet.isReady) {
    try {
      await ElMessageBox.confirm(
        `将唤起 MetaMask 对本笔成交（${matchingOrder.value.quantity} 积分）进行链上授权签名，` +
        '积分将先转移至交易所托管合约地址。',
        '链上授权确认',
        { confirmButtonText: '签名授权', cancelButtonText: '跳过链上授权', type: 'info' }
      )
      const { address, methods } = CARBON_CONTRACTS.carbonCreditToken
      const data = abiEncodeCall(
        methods.transfer,
        [abiEncodeWord(CARBON_CONTRACTS.escrowAddress), abiEncodeUint256(matchingOrder.value.quantity)]
      )
      authTxHash = await sendContractTransaction(wallet.address, address, data)
      pushWalletTx(`撮合授权 ${matchingOrder.value.quantity} 积分`, authTxHash)
      ElMessage.success('链上授权完成：' + authTxHash.slice(0, 14) + '...')
    } catch (e) {
      if (e === 'cancel') {
        ElMessage.info('已跳过链上授权，仅进行平台内撮合')
      } else {
        ElMessage.error(formatWalletError(e))
      }
    }
  }

  loadingMatch.value = true
  try {
    const res = await exchangeAPI.match({
      order_id: matchingOrder.value.id,
      buyer_id: matchBuyerId.value,
      quantity: matchingOrder.value.quantity,
      credit_no: matchCreditNo.value  // 必传：碳积分凭证编号(后端校验与挂单一致)
    })
    const authTip = authTxHash ? `，链上授权 ${authTxHash.slice(0, 12)}...` : ''
    ElMessage.success(`撮合成功！成交编号 ${res.data?.transaction_no || '--'}${authTip}`)
    matchVisible.value = false
    refreshAll()
  } catch (e) {
    ElMessage.error(e?.msg || '撮合失败')
  } finally {
    loadingMatch.value = false
  }
}

function goArbitration(order) {
  router.push('/arbitration')
}

onMounted(() => refreshAll())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 页面头部 ===== */
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  margin-bottom: 16px;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, #7c3aed, #a78bfa);
  color: #fff;
  box-shadow: 0 4px 20px rgba(124, 58, 237, 0.25);
}
.ph-left h2 { font-size: 18px; font-weight: 600; color: #fff; margin-bottom: 6px; }
.ph-desc { font-size: 12.5px; color: rgba(255,255,255,0.85); margin: 0; }

.ph-stats {
  display: flex; gap: 20px;
}
.ph-stat {
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  padding: 8px 20px;
  background: rgba(255,255,255,0.15);
  border-radius: 10px;
  backdrop-filter: blur(8px);
}
.ps-num { font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; }
.ps-label { font-size: 11px; opacity: 0.85; }

/* ===== 标签页卡片 ===== */
.tab-card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius);
  padding: 0 20px 16px;
  margin-bottom: 16px;
}
.exchange-tabs :deep(.el-tabs__item) {
  font-weight: 500;
  color: var(--text-secondary);
}
.exchange-tabs :deep(.el-tabs__item.is-active) {
  color: var(--primary-green);
}
.exchange-tabs :deep(.el-tabs__active-bar) {
  background-color: var(--primary-green);
}

.tab-banner {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 14px;
  background: rgba(124,58,237,0.06);
  border-radius: 8px;
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 8px;
}
.tab-actions {
  display: flex; align-items: center; gap: 10px;
  margin-top: 12px;
}

/* ===== 挂单卡片列表 ===== */
.orders-card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius);
  padding: 20px;
  margin-bottom: 16px;
}
.oc-head {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 16px;
}
.oc-head h3 { font-size: 14px; font-weight: 600; }
.oc-sub { font-size: 12px; color: var(--text-tertiary); }

.loading-wrap, .empty-wrap {
  padding: 60px 20px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 12.5px;
}
.empty-wrap p { margin-top: 10px; }

.orders-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}

/* 订单卡片 */
.order-card {
  position: relative;
  display: flex; flex-direction: column; gap: 12px;
  padding: 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--line-color);
  border-radius: 12px;
  transition: all 0.2s;
}
.order-card:hover {
  border-color: var(--primary-green);
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(0,0,0,0.08);
}
.order-card.disputed {
  border-color: #ef444480;
  box-shadow: 0 0 0 1px rgba(239,68,68,0.3);
}

.oc-top { display: flex; justify-content: space-between; }
.oc-type-tag {
  font-size: 11px; padding: 2px 10px; border-radius: 999px; font-weight: 500;
}
.oc-type-tag.spot    { background: rgba(22,163,74,0.1);  color: #16a34a; }
.oc-type-tag.lease   { background: rgba(245,158,11,0.1); color: #d97706; }
.oc-type-tag.forward  { background: rgba(124,58,237,0.1); color: #7c3aed; }

.oc-status {
  font-size: 11px; padding: 2px 10px; border-radius: 999px;
  background: rgba(16,185,129,0.1); color: #2563eb;
}
.oc-status.matched   { background: rgba(22,163,74,0.1); color: #16a34a; }
.oc-status.completed { background: rgba(22,163,74,0.1); color: #16a34a; }
.oc-status.cancelled { background: rgba(148,163,184,0.1); color: #64748b; }
.oc-status.disputed  { background: rgba(239,68,68,0.1); color: #ef4444; }

.oc-main {
  display: flex; align-items: flex-end; justify-content: space-between;
  padding: 10px 0;
  border-top: 1px dashed var(--line-color);
  border-bottom: 1px dashed var(--line-color);
}
.oc-price { display: flex; align-items: baseline; gap: 2px; color: var(--primary-green); }
.currency { font-size: 14px; font-weight: 600; }
.oc-price .val { font-size: 26px; font-weight: 700; line-height: 1; }
.unit { font-size: 11px; color: var(--text-tertiary); margin-left: 4px; }
.oc-qty { text-align: right; }
.qty-num { font-size: 22px; font-weight: 700; color: var(--text-primary); font-variant-numeric: tabular-nums; }
.qty-unit { font-size: 11px; color: var(--text-tertiary); margin-left: 2px; }

.oc-seller { display: flex; align-items: center; gap: 10px; }
.seller-avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #7c3aed, #a78bfa);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  font-weight: 600; font-size: 13px;
  flex-shrink: 0;
}
.seller-info { min-width: 0; }
.seller-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.seller-meta { font-size: 11px; color: var(--text-tertiary); display: flex; align-items: center; gap: 4px; margin-top: 2px; }
.seller-meta .divider { opacity: 0.5; }

.oc-chain {
  display: flex; align-items: center; gap: 6px;
  font-size: 10.5px; color: var(--text-tertiary);
}
.oc-chain code { font-family: ui-monospace, Consolas, monospace; }

.oc-dispute {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 10px;
  background: rgba(239,68,68,0.08);
  border-radius: 6px;
  font-size: 11.5px;
  color: #ef4444;
}

.oc-actions {
  display: flex; gap: 8px; flex-wrap: wrap;
}

/* ===== 成交记录 ===== */
.tx-card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius);
  padding: 20px;
  margin-bottom: 16px;
}
.tx-head {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 14px;
}
.tx-head h3 { font-size: 14px; font-weight: 600; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }

/* ===== 撮合弹窗 ===== */
.match-dialog :deep(.el-dialog__header) { padding: 16px 20px; }
.match-dialog :deep(.el-dialog__body) { padding: 8px 20px 16px; }
.match-dialog :deep(.el-dialog__footer) { padding: 12px 20px 20px; }
.credit-no-ro { font-size: 13px; color: #047857; background: rgba(16,185,129,0.08); padding: 2px 8px; border-radius: 4px; user-select: all; }

/* 挂单卡片类型专属信息(租赁租期/远期交割) */
.oc-extra {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
  padding: 6px 10px;
  border-radius: 6px;
  background: rgba(124,58,237,0.05);
  font-size: 11.5px;
  color: var(--text-secondary);
}
.ro-text { font-size: 12.5px; color: var(--text-secondary); }
.dlg-head { display: flex; align-items: center; gap: 8px; font-weight: 600; }
.match-body { display: flex; flex-direction: column; gap: 16px; }
.buyer-select-wrap label {
  display: block; font-size: 12px; font-weight: 500;
  color: var(--text-secondary); margin-bottom: 6px;
}

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
  .page-head { flex-direction: column; gap: 12px; align-items: flex-start; }
  .ph-stats { width: 100%; justify-content: flex-start; flex-wrap: wrap; }
}
</style>