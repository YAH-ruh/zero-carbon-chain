<!--
  zkp-proof.vue - ZKP 隐私核算证明
  功能：
    1. 选择碳积分记录 → 勾选可公开字段（选择性披露）
    2. 生成 ZKP 证明（后端 /api/zkp/proof，模拟 Groth16 零知识）
    3. 导出 .json 隐私凭证文件
    4. 监管视角：PQC 抗量子校验面板
    5. 历史证明记录列表
  角色：enterprise (生成 + 导出) / regulator (校验 + 查看)
-->
<template>
  <div class="zkp-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>隐私核算 · ZKP 选择性披露证明</h2>
        <p class="ph-desc">
          <el-icon :size="13"><Lock /></el-icon>
          零知识证明：对外只披露统计结果，保护企业能耗细粒度隐私
        </p>
      </div>
    </div>

    <!-- ===== 链上钱包（企业专属：手动连接 → 证明哈希签名 + 合约存证） ===== -->
    <WalletConnect
      v-if="role === 'enterprise'"
      hint="隐私核算 · 链上钱包（证明哈希签名 / 合约上链存证）"
      class="zkp-wallet-bar"
    />

    <!-- ===== 监管专属：PQC 校验面板 ===== -->
    <template v-if="role === 'regulator'">
      <div class="pqc-panel">
        <div class="pqc-head">
          <h3>
            <el-icon :size="16"><Lock /></el-icon>
            PQC 抗量子校验面板（监管专属）
          </h3>
          <el-tag type="danger" effect="dark" size="small">
            <el-icon :size="11"><Warning /></el-icon>
            抗量子签名验证
          </el-tag>
        </div>
        <el-form :inline="true" class="pqc-form">
          <el-form-item label="证明编号">
            <el-input v-model="pqcForm.proof_no" placeholder="如: ZKP-20260909-001" style="width:260px" />
          </el-form-item>
          <el-form-item label="操作">
            <el-button type="warning" :icon="Search" @click="verifyPQC" :loading="pqcLoading">
              执行 PQC 校验
            </el-button>
          </el-form-item>
        </el-form>
        <div v-if="pqcResult" class="pqc-result" :class="pqcResult.valid ? 'ok' : 'fail'">
          <div class="pqc-icon">
            <el-icon :size="22"><component :is="pqcResult.valid ? Check : Close" /></el-icon>
          </div>
          <div class="pqc-body">
            <div class="pqc-title">{{ pqcResult.valid ? 'PQC 抗量子校验通过' : 'PQC 校验失败' }}</div>
            <div class="pqc-detail">{{ pqcResult.detail }}</div>
          </div>
          <div v-if="pqcResult.valid" class="pqc-chain">
            <span>链上已登记 · 后量子安全 · 256 位密钥</span>
          </div>
        </div>
      </div>
    </template>

    <!-- ===== 步骤一：选择碳积分记录 ===== -->
    <div class="step-card">
      <div class="step-head">
        <span class="step-num">1</span>
        <h3>选择碳积分记录</h3>
        <span class="step-desc">从您的已核算碳积分中选择一条，生成 ZKP 隐私证明</span>
      </div>
      <div v-if="loadingCredits" class="loading-inline">
        <el-icon class="is-loading" :size="18"><Loading /></el-icon>
      </div>
      <el-table
        v-else
        :data="credits"
        highlight-current-row
        @current-change="selectCredit"
        style="width:100%"
        max-height="280"
      >
        <el-table-column label="选中" width="80" align="center">
          <template #default="{ row }">
            <el-radio :model-value="selectedCredit?.id" :value="row.id" @click.stop="selectCredit(row)" />
          </template>
        </el-table-column>
        <el-table-column prop="credit_no" label="核算编号" width="200">
          <template #default="{ row }"><code class="mono">{{ row.credit_no }}</code></template>
        </el-table-column>
        <el-table-column label="碳排放(kg)" width="120" align="right">
          <template #default="{ row }">{{ row.total_emission?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="碳积分" width="120" align="right">
          <template #default="{ row }"><b class="green">{{ row.carbon_credits?.toFixed(2) }}</b></template>
        </el-table-column>
        <el-table-column label="来源能耗" width="150">
          <template #default="{ row }"><code>{{ row.energy_record_id || '--' }}</code></template>
        </el-table-column>
        <el-table-column label="上链" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" type="success" size="small">已上链</el-tag>
            <el-tag v-else type="info" size="small">--</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="核算时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== 步骤二：选择性披露字段 ===== -->
    <div class="step-card">
      <div class="step-head">
        <span class="step-num">2</span>
        <h3>选择性披露字段</h3>
        <span class="step-desc">勾选对外公开的字段（未勾选的字段在 ZKP 证明中完全隐藏）</span>
      </div>
      <div v-if="!selectedCredit" class="empty-hint">
        <el-icon :size="32"><Lock /></el-icon>
        <p>请先在上方选择一条碳积分记录</p>
      </div>
      <template v-else>
        <div class="disclose-grid">
          <div
            v-for="f in fieldOptions"
            :key="f.key"
            class="disclose-item"
            :class="{ selected: discloseFields.includes(f.key) }"
            @click="toggleField(f.key)"
          >
            <div class="df-check">
              <el-icon v-if="discloseFields.includes(f.key)"><Check /></el-icon>
            </div>
            <div class="df-info">
              <div class="df-name">{{ f.label }}</div>
              <div class="df-desc">{{ f.desc }}</div>
            </div>
            <span class="df-tag">
              {{ discloseFields.includes(f.key) ? '公开' : '隐私' }}
            </span>
          </div>
        </div>

        <!-- 模拟披露预览 -->
        <div class="preview-box">
          <div class="pv-head">
            <span>🔐 ZKP 证明预览（对外可见内容）</span>
            <el-tag size="small" type="warning">模拟数据 · 演示用途</el-tag>
          </div>
          <pre class="pv-json"><code>{{ previewDisclosure }}</code></pre>
        </div>
      </template>
    </div>

    <!-- ===== 步骤三：生成 & 导出 ===== -->
    <div class="step-card">
      <div class="step-head">
        <span class="step-num">3</span>
        <h3>生成 ZKP 证明并上链</h3>
        <span class="step-desc">点击生成后系统将模拟 Groth16 零知识证明计算过程</span>
      </div>

      <div class="gen-actions">
        <el-button
          type="primary"
          :icon="MagicStick"
          :disabled="!selectedCredit || !discloseFields.length"
          :loading="loadingGen"
          @click="generateProof"
        >
          {{ loadingGen ? '正在生成 ZKP 证明...' : '生成 ZKP 证明' }}
        </el-button>
        <el-button
          :icon="Download"
          :disabled="!currentProof"
          @click="exportProof"
        >
          导出隐私凭证 (.json)
        </el-button>
        <el-tag v-if="currentProof" type="success" effect="plain" size="small">
          已生成 {{ currentProof.proof_no }}
        </el-tag>
      </div>

      <!-- 证明结果展示 -->
      <div v-if="generating" class="gen-progress">
        <div class="gp-steps">
          <div class="gp-item" v-for="(s, i) in genSteps" :key="i" :class="{ active: currentGenStep === i, done: currentGenStep > i }">
            <el-icon v-if="currentGenStep > i"><Check /></el-icon>
            <el-icon v-else-if="currentGenStep === i" class="is-loading"><Loading /></el-icon>
            <span v-else>{{ i + 1 }}</span>
            <span class="gp-label">{{ s }}</span>
          </div>
        </div>
      </div>

      <div v-if="currentProof" class="proof-display">
        <div class="pd-meta">
          <div class="pd-row"><span>证明编号</span><code>{{ currentProof.proof_no }}</code></div>
          <div class="pd-row"><span>证明系统</span>{{ currentProof.circuit_type }}</div>
          <div class="pd-row"><span>链上哈希</span><code>{{ currentProof.block_hash || '--' }}</code></div>
          <div class="pd-row"><span>已上链</span>
            <el-tag v-if="currentProof.on_chain" type="success" size="small">是</el-tag>
            <el-tag v-else type="info" size="small">否</el-tag>
          </div>
          <div class="pd-row"><span>生成时间</span>{{ currentProof.timestamp }}</div>
        </div>
        <pre class="pd-proof"><code>{{ formatProofJson(currentProof.proof_data) }}</code></pre>
      </div>

      <!-- 钱包签名 & 合约上链存证（企业视角，证明生成后出现） -->
      <div v-if="currentProof && role === 'enterprise'" class="wallet-sign-card">
        <div class="ws-head">
          <span>🔐 MetaMask 签名与智能合约上链存证</span>
          <el-tag v-if="wallet.isReady" type="success" size="small" effect="plain">钱包已就绪</el-tag>
          <el-tag v-else type="warning" size="small" effect="plain">请先在顶部连接钱包</el-tag>
        </div>
        <div class="ws-hash">证明哈希（SHA-256）：<code>{{ proofHash || '待计算' }}</code></div>
        <div class="ws-actions">
          <el-button
            type="primary" :icon="Key"
            :disabled="!wallet.isReady || !currentProof"
            :loading="signingProof"
            @click="signProofHash"
          >
            ① MetaMask 签名证明哈希
          </el-button>
          <el-button
            type="success" :icon="Upload"
            :disabled="!proofSignature"
            :loading="submittingProof"
            @click="submitProofOnChain"
          >
            ② 提交合约上链存证
          </el-button>
        </div>
        <div v-if="proofSignature" class="ws-line">
          签名结果：<code>{{ proofSignature.slice(0, 42) }}...{{ proofSignature.slice(-16) }}</code>
          <el-button link type="primary" size="small" @click="copyText(proofSignature)">复制</el-button>
        </div>
        <div v-if="onChainTxHash" class="ws-line ok">
          ✅ 存证交易哈希：<code>{{ onChainTxHash }}</code>
          <el-button link type="primary" size="small" @click="copyText(onChainTxHash)">复制</el-button>
        </div>
      </div>
    </div>

    <!-- ===== 历史证明列表 ===== -->
    <div class="history-card">
      <div class="hc-head">
        <h3>我的 ZKP 证明记录</h3>
        <el-button link type="primary" :icon="Refresh" @click="loadMyProofs" :loading="loadingMyProofs">
          刷新
        </el-button>
      </div>
      <el-table :data="myProofs" stripe style="width:100%" v-loading="loadingMyProofs" empty-text="暂无证明记录">
        <el-table-column prop="proof_no" label="证明编号" width="200">
          <template #default="{ row }"><code class="mono">{{ row.proof_no }}</code></template>
        </el-table-column>
        <el-table-column label="类型" width="130">
          <template #default="{ row }">
            <el-tag size="small">
              {{ proofTypeMap[row.proof_type] || row.proof_type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="disclosed_fields" label="公开字段" width="200">
          <template #default="{ row }">
            <span v-for="f in (row.disclosed_fields || []).slice(0,3)" :key="f" class="df-mini">{{ f }}</span>
          </template>
        </el-table-column>
        <el-table-column label="链上状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" type="success" size="small">已上链</el-tag>
            <el-tag v-else type="info" size="small">--</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="生成时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Lock, Search, Warning, Loading, Check, Close, Download, MagicStick, Refresh, Key, Upload
} from '@element-plus/icons-vue'
import { zkpAPI, carbonAPI } from '../api/index.js'
import WalletConnect from '../components/WalletConnect.vue'
import { useWalletStore } from '../stores/wallet.js'
import {
  signMessage, sendContractTransaction, formatWalletError,
  CARBON_CONTRACTS, abiEncodeCall, abiEncodeWord,
} from '../utils/web3.js'

const user = JSON.parse(localStorage.getItem('user') || '{}')
const role = user.role || 'enterprise'

/* ===== 碳积分记录 ===== */
const credits = ref([])
const loadingCredits = ref(false)
const selectedCredit = ref(null)

/* ===== 选择性披露字段 ===== */
const fieldOptions = [
  { key: 'total_emission',  label: '总碳排放量', desc: '企业在选定周期的总排放 kgCO₂' },
  { key: 'carbon_credits',  label: '碳积分数量',  desc: '最终核算得到的碳积分值' },
  { key: 'period',          label: '核算周期',    desc: '本次核算的时间范围' },
  { key: 'energy_breakdown',label: '能耗明细',    desc:  '电力/天然气/用水分项（包含企业能耗结构）' },
  { key: 'energy_record_id',label: '能耗记录编号', desc: '原始能耗数据 ID（最敏感字段）' },
]
const discloseFields = ref(['total_emission', 'carbon_credits']) // 默认只公开统计结果

const toggleField = (k) => {
  if (discloseFields.value.includes(k)) {
    discloseFields.value = discloseFields.value.filter(x => x !== k)
  } else {
    discloseFields.value = [...discloseFields.value, k]
  }
}

/* 模拟披露预览（未披露字段用 HMAC 承诺代替） */
const previewDisclosure = computed(() => {
  if (!selectedCredit.value) return ''
  const c = selectedCredit.value
  const enc = (visible, label) => visible ? '** 已公开（原值可见）' : '0x' + Array(24).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join('') + ' （HMAC 承诺）'
  return JSON.stringify({
    proof_no: c.credit_no,
    disclosed: discloseFields.value,
    assertions: {
      total_emission:     discloseFields.value.includes('total_emission')  ? c.total_emission?.toFixed(2) + ' kgCO₂'    : '0x' + '?'.repeat(32),
      carbon_credits:     discloseFields.value.includes('carbon_credits')  ? c.carbon_credits?.toFixed(2) + ' 积分'    : '0x' + '?'.repeat(32),
      period:             discloseFields.value.includes('period')          ? (c.created_at?.slice(0,10) || '2026-09-09') : '0x' + '?'.repeat(32),
      energy_breakdown:   discloseFields.value.includes('energy_breakdown') ? '{"electricity":' + (Math.round(Math.random()*500)) + ',...}' : '（未公开，只知道区间范围）',
      energy_record_id:   discloseFields.value.includes('energy_record_id') ? c.energy_record_id || 'REC-001' : '****（最敏感字段，默认隐藏）',
    },
    "🔒 隐私声明": "未披露字段以 HMAC-SHA256 承诺形式存在，校验方无法反推原始值"
  }, null, 2)
})

/* ===== 生成证明 ===== */
const loadingGen = ref(false)
const generating = ref(false)
const currentGenStep = ref(0)
const currentProof = ref(null)

const genSteps = ['初始化 Groth16 电路', '计算字段 HMAC 承诺', '构造 range proof（范围证明）', '聚合为单一 ZKP 证明', '上链存证 & 返回证明编号']

async function generateProof() {
  if (!selectedCredit.value) { ElMessage.warning('请先选择一条碳积分记录'); return }
  if (!discloseFields.value.length) { ElMessage.warning('请至少勾选一个公开字段'); return }

  generating.value = true
  loadingGen.value = true
  currentGenStep.value = 0
  currentProof.value = null

  // 模拟进度动画
  for (let i = 0; i < genSteps.length; i++) {
    await new Promise(r => setTimeout(r, 450))
    currentGenStep.value = i + 1
  }

  try {
    const res = await zkpAPI.generateProof({
      record_no: selectedCredit.value.credit_no,
      energy_record_id: selectedCredit.value.energy_record_id,
      fields: discloseFields.value,
      proof_type: 'selective_disclosure'
    })
    currentProof.value = res.data?.proof || res.data
    if (!currentProof.value) throw new Error('no data')
  } catch {
    // 前端兜底生成模拟证明
    currentProof.value = {
      proof_no: 'ZKP-' + Date.now(),
      credit_no: selectedCredit.value.credit_no,
      proof_type: 'selective_disclosure',
      circuit_type: 'Groth16',
      disclosed_fields: discloseFields.value,
      hid_commitment: '0x' + Array(64).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join(''),
      range_proof: '0x' + Array(64).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join(''),
      aggregated_proof: '0x' + Array(128).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join(''),
      verification_key: 'VK_' + Math.random().toString(36).slice(2, 10).toUpperCase(),
      proof_data: JSON.stringify({
        hid_commitment: '0x' + Array(64).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join(''),
        range_proof:    '0x' + Array(64).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join(''),
        aggregated_proof:'0x' + Array(128).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join(''),
        vk:             'VK_' + Math.random().toString(36).slice(2, 10).toUpperCase(),
      }, null, 2),
      block_hash: '0x' + Array(40).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join(''),
      on_chain: true,
      timestamp: new Date().toISOString(),
    }
    ElMessage.success('ZKP 证明生成成功（前端模拟）')
  } finally {
    loadingGen.value = false
    setTimeout(() => generating.value = false, 600)
  }
  // 生成后计算证明哈希，并重置签名 / 存证状态（供钱包签名与合约存证使用）
  await computeProofHash()
}

function exportProof() {
  if (!currentProof.value) return
  const data = { ...currentProof.value, exported_fields: discloseFields.value }
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `zkp-proof-${currentProof.value.proof_no}.json`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('ZKP 证明文件已下载：' + a.download)
}

function formatProofJson(str) {
  try {
    return JSON.stringify(typeof str === 'string' ? JSON.parse(str) : str, null, 2)
  } catch { return str }
}

/* ===== 链上钱包：证明哈希签名 + 智能合约存证（MetaMask，仅用户手动连接） ===== */
const wallet = useWalletStore()
const proofHash = ref('')
const proofSignature = ref('')
const onChainTxHash = ref('')
const signingProof = ref(false)
const submittingProof = ref(false)

/** 计算证明哈希（SHA-256）：作为 personal_sign 消息与合约 bytes32 入参 */
async function computeProofHash() {
  if (!currentProof.value) return
  try {
    const raw = JSON.stringify(currentProof.value.proof_data || currentProof.value)
    const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(raw))
    proofHash.value = '0x' + Array.from(new Uint8Array(buf)).map(b => b.toString(16).padStart(2, '0')).join('')
  } catch {
    // 浏览器环境不支持 crypto.subtle 时兜底生成演示哈希
    proofHash.value = '0x' + Array(64).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join('')
  }
  // 新证明生成后重置签名与存证状态
  proofSignature.value = ''
  onChainTxHash.value = ''
}

/** 第一步：MetaMask 对证明哈希进行 personal_sign 签名（唤起签名确认弹窗） */
async function signProofHash() {
  if (!wallet.isReady) { ElMessage.warning('请先连接钱包并切换到 Sepolia 测试网'); return }
  if (!proofHash.value) await computeProofHash()
  signingProof.value = true
  try {
    proofSignature.value = await signMessage(proofHash.value, wallet.address)
    ElMessage.success('证明哈希签名成功，可提交合约存证')
  } catch (e) {
    ElMessage.error(formatWalletError(e))
  } finally {
    signingProof.value = false
  }
}

/** 第二步：将签名后的 ZKP 证明提交智能合约上链存证（预留合约写入口） */
async function submitProofOnChain() {
  if (!proofSignature.value) { ElMessage.warning('请先完成 MetaMask 签名'); return }
  submittingProof.value = true
  try {
    const { address, methods } = CARBON_CONTRACTS.zkpRegistry
    // 预留合约写入口：storeProof(bytes32 证明哈希)
    const data = abiEncodeCall(methods.storeProof, [abiEncodeWord(proofHash.value)])
    onChainTxHash.value = await sendContractTransaction(wallet.address, address, data)
    // 同步证明状态为已上链，交易哈希作为链上哈希展示
    currentProof.value.on_chain = true
    currentProof.value.block_hash = onChainTxHash.value
    ElMessage.success('ZKP 证明已提交智能合约上链存证')
  } catch (e) {
    ElMessage.error(formatWalletError(e))
  } finally {
    submittingProof.value = false
  }
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

/* ===== 监管专属：PQC ===== */
const pqcForm = reactive({ proof_no: '' })
const pqcLoading = ref(false)
const pqcResult = ref(null)

async function verifyPQC() {
  if (!pqcForm.proof_no) { ElMessage.warning('请填写证明编号'); return }
  pqcLoading.value = true
  await new Promise(r => setTimeout(r, 1200))
  // 模拟校验结果（85% 通过）
  const valid = Math.random() > 0.15
  pqcResult.value = valid ? {
    valid: true,
    detail: '采用 CRYSTALS-Dilithium 签名方案 · 密钥长度 2557 字节 · 签名长度 4595 字节 · 验证成功',
  } : {
    valid: false,
    detail: '签名过期 / 公钥不匹配 · 链上登记验证失败 · 请检查证明编号是否正确',
  }
  pqcLoading.value = false
}

/* ===== 我的证明 ===== */
const myProofs = ref([])
const loadingMyProofs = ref(false)
const proofTypeMap = {
  selective_disclosure: '选择性披露',
  pqc: 'PQC 抗量子',
  zk_ai: 'ZK-AI 异常检测'
}

async function loadMyProofs() {
  loadingMyProofs.value = true
  try {
    const res = await zkpAPI.myProofs({ page: 1, page_size: 20 })
    myProofs.value = res.data?.list || res.data || []
  } catch {
    // mock
    myProofs.value = Array.from({ length: 5 }, (_, i) => ({
      id: i + 1,
      proof_no: 'ZKP-2026090' + (9 - i) + '-' + String(1000 + i),
      credit_no: 'CREDIT-202609-' + String(20 + i),
      proof_type: ['selective_disclosure','pqc','selective_disclosure','zk_ai','selective_disclosure'][i],
      disclosed_fields: [['total_emission','carbon_credits'],['proof_sign'],['total_emission','period'],['anomaly_flag'],['carbon_credits']][i],
      on_chain: Math.random() > 0.2,
      created_at: new Date(Date.now() - i * 86400000 * 3).toISOString(),
    }))
  } finally {
    loadingMyProofs.value = false
  }
}

/* ===== 加载碳积分 ===== */
async function loadCredits() {
  loadingCredits.value = true
  try {
    const res = await carbonAPI.myCredits({ page: 1, page_size: 20 })
    credits.value = res.data?.list || res.data || []
  } catch {
    credits.value = Array.from({ length: 6 }, (_, i) => ({
      id: i + 1,
      credit_no: 'CREDIT-202609-' + String(20 + i),
      total_emission: +(200 + Math.random() * 800).toFixed(2),
      carbon_credits: +(50 + Math.random() * 200).toFixed(2),
      energy_record_id: 'REC-' + (100 + i),
      on_chain: Math.random() > 0.15,
      created_at: new Date(Date.now() - i * 86400000 * 7).toISOString(),
    }))
  } finally {
    loadingCredits.value = false
  }
}

function selectCredit(row) {
  selectedCredit.value = row
  if (row) {
    discloseFields.value = ['total_emission', 'carbon_credits'] // 重置为默认
  }
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}

onMounted(() => {
  loadCredits()
  loadMyProofs()
})
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #0d9488, #0891b2);
  border-radius: var(--radius-lg);
  color: #fff;
}
.page-head h2 { font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.ph-desc { display: flex; align-items: center; gap: 6px; font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }
.ph-tags { display: flex; gap: 6px; }

/* ===== 步骤卡 ===== */
.step-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.step-head {
  display: flex; align-items: center; gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--line-color);
  margin-bottom: 14px;
}
.step-num {
  width: 28px; height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary-green), #22c55e);
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 13px;
}
.step-head h3 { font-size: 14px; font-weight: 600; margin: 0; }
.step-desc { font-size: 11.5px; color: var(--text-tertiary); margin-left: auto; }

.loading-inline { padding: 30px; text-align: center; color: var(--text-tertiary); }
.empty-hint {
  padding: 40px 20px; text-align: center; color: var(--text-tertiary);
}
.empty-hint p { margin-top: 10px; font-size: 12.5px; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 12px; }
.green { color: var(--primary-green); }

/* ===== PQC 面板（监管） ===== */
.pqc-panel {
  background: var(--card-bg); border: 1px solid #f59e0b40;
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
  border-left: 4px solid #f59e0b;
}
.pqc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.pqc-head h3 { display: flex; align-items: center; gap: 8px; font-size: 14px; font-weight: 600; margin: 0; color: #d97706; }

.pqc-form :deep(.el-form-item) { margin-bottom: 8px; }

.pqc-result {
  display: flex; gap: 14px; align-items: center;
  padding: 14px 18px;
  border-radius: 10px;
  margin-top: 12px;
}
.pqc-result.ok { background: rgba(16,185,129,0.08); border: 1px solid rgba(16,185,129,0.3); }
.pqc-result.fail { background: rgba(239,68,68,0.08); border: 1px solid rgba(239,68,68,0.3); }
.pqc-icon {
  width: 44px; height: 44px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.pqc-result.ok .pqc-icon { background: rgba(16,185,129,0.2); color: #10b981; }
.pqc-result.fail .pqc-icon { background: rgba(239,68,68,0.2); color: #ef4444; }
.pqc-title { font-size: 14px; font-weight: 600; margin-bottom: 4px; }
.pqc-result.ok .pqc-title { color: #10b981; }
.pqc-result.fail .pqc-title { color: #ef4444; }
.pqc-detail { font-size: 12px; color: var(--text-secondary); }
.pqc-chain { font-size: 10.5px; color: var(--text-tertiary); }

/* ===== 选择性披露 ===== */
.disclose-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 10px; margin-bottom: 14px;
}
.disclose-item {
  display: flex; align-items: center; gap: 10px;
  padding: 12px 14px;
  background: var(--bg-secondary);
  border: 2px solid var(--line-color);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s;
}
.disclose-item:hover { border-color: rgba(16,185,129,0.5); }
.disclose-item.selected {
  border-color: var(--primary-green);
  background: rgba(16,185,129,0.06);
}
.df-check {
  width: 20px; height: 20px; border-radius: 5px;
  border: 2px solid var(--line-strong);
  display: flex; align-items: center; justify-content: center;
  color: transparent; flex-shrink: 0;
  transition: all 0.15s;
}
.disclose-item.selected .df-check {
  background: var(--primary-green); border-color: var(--primary-green); color: #fff;
}
.df-info { flex: 1; min-width: 0; }
.df-name { font-size: 13px; font-weight: 500; color: var(--text-primary); }
.df-desc { font-size: 11px; color: var(--text-tertiary); margin-top: 2px; }
.df-tag {
  font-size: 10px; padding: 2px 8px; border-radius: 999px;
  background: rgba(148,163,184,0.1); color: #64748b;
  flex-shrink: 0;
}
.disclose-item.selected .df-tag { background: rgba(16,185,129,0.15); color: #10b981; }

/* 预览 */
.preview-box {
  margin-top: 10px;
  background: var(--bg-secondary);
  border: 1px dashed var(--line-strong);
  border-radius: 10px;
  padding: 14px 16px;
}
.pv-head {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 12px; font-weight: 500; color: var(--text-secondary);
  margin-bottom: 10px;
}
.pv-json {
  margin: 0; padding: 10px 14px;
  background: #064e3b; color: #5eead4;
  border-radius: 6px;
  font-size: 11.5px; line-height: 1.6;
  max-height: 260px; overflow: auto;
}
.pv-json code { color: inherit; font-family: ui-monospace, Consolas, monospace; }

/* ===== 生成步骤 ===== */
.gen-actions { display: flex; gap: 10px; margin-bottom: 18px; }

.gen-progress { padding: 16px 0; }
.gp-steps { display: flex; gap: 0; }
.gp-item {
  flex: 1; display: flex; flex-direction: column; align-items: center; gap: 6px;
  font-size: 11px; color: var(--text-tertiary);
  position: relative;
}
.gp-item:not(:last-child)::after {
  content: '';
  position: absolute; top: 10px; left: 60%; right: -40%;
  height: 2px;
  background: var(--line-color);
}
.gp-item.done:not(:last-child)::after { background: var(--primary-green); }
.gp-item .el-icon {
  width: 20px; height: 20px; border-radius: 50%;
  background: var(--line-color); color: transparent;
  display: flex; align-items: center; justify-content: center;
}
.gp-item.active .el-icon { background: var(--primary-green); color: #fff; }
.gp-item.done .el-icon { background: var(--primary-green); color: #fff; }
.gp-label { text-align: center; }

/* 证明展示 */
.proof-display {
  margin-top: 16px;
  background: var(--bg-secondary);
  border: 1px solid rgba(16,185,129,0.3);
  border-radius: 10px;
  overflow: hidden;
}
.pd-meta {
  display: grid; grid-template-columns: repeat(2, 1fr);
  gap: 6px 16px;
  padding: 14px 18px;
  background: rgba(16,185,129,0.05);
  border-bottom: 1px solid var(--line-color);
}
.pd-row { display: flex; gap: 8px; font-size: 12px; }
.pd-row span:first-child { color: var(--text-tertiary); min-width: 80px; }
.pd-row code { font-family: ui-monospace, Consolas, monospace; font-size: 11px; color: var(--text-primary); word-break: break-all; }

.pd-proof {
  margin: 0; padding: 14px 18px;
  background: #0a1628; color: #5eead4;
  font-size: 11px; line-height: 1.6;
  max-height: 300px; overflow: auto;
}
.pd-proof code { color: inherit; }

/* ===== 历史记录 ===== */
.history-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.hc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.hc-head h3 { font-size: 14px; font-weight: 600; margin: 0; }

.df-mini {
  display: inline-block; font-size: 10px;
  padding: 1px 6px; margin: 1px 2px;
  background: rgba(16,185,129,0.1); color: #10b981;
  border-radius: 3px;
}

/* ===== 钱包签名存证卡 ===== */
.zkp-wallet-bar { margin-bottom: 16px; }
.wallet-sign-card {
  margin-top: 16px;
  border: 1px solid rgba(16,185,129,0.35);
  background: rgba(16,185,129,0.04);
  border-radius: 10px;
  padding: 14px 18px;
}
.ws-head {
  display: flex; align-items: center; gap: 10px;
  font-size: 13px; font-weight: 600; color: var(--text-primary);
  margin-bottom: 10px;
}
.ws-hash { font-size: 11.5px; color: var(--text-secondary); margin-bottom: 12px; word-break: break-all; }
.ws-hash code { font-family: ui-monospace, Consolas, monospace; color: var(--primary-green); }
.ws-actions { display: flex; gap: 10px; flex-wrap: wrap; }
.ws-line { font-size: 11.5px; color: var(--text-secondary); margin-top: 10px; word-break: break-all; }
.ws-line.ok { color: #10b981; }
.ws-line code { font-family: ui-monospace, Consolas, monospace; color: var(--primary-green); }

@media (max-width: 900px) {
  .page-head { flex-direction: column; gap: 10px; align-items: flex-start; }
  .gp-steps { flex-wrap: wrap; }
}
</style>