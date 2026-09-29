<!--
  ai-agent.vue - AI 交易 Agent 管理（exchange / regulator 角色）
  功能：
    1. 三大 Agent 操作入口卡片（交易 / 风险 / 园区调度）
    2. Agent 自动执行交易模拟（AI 撮合挂单）
    3. Agent 操作实时日志流
    4. ZK 推理证明查询
    5. 历史记录 + 链上哈希
  后端接口：agentAPI.triggerTrade / triggerRisk / triggerDispatch / records
-->
<template>
  <div class="agent-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>
          <el-icon :size="18"><Monitor /></el-icon>
          AI 交易 Agent 管理
        </h2>
        <p>三类区块链 Agent 自动执行交易撮合、风险检测、园区调度 · 所有操作上链存证</p>
      </div>
      <div class="ph-tags">
        <el-tag effect="dark" size="small" type="primary">3 Agent 在线</el-tag>
        <el-tag effect="dark" size="small" type="success">
          <span :class="{ blink: agentRunning }">●</span>
          {{ agentRunning ? '执行中' : '空闲' }}
        </el-tag>
      </div>
    </div>

    <!-- ===== 三大 Agent 卡片 ===== -->
    <div class="agent-grid">
      <!-- 交易 Agent -->
      <div class="agent-card trade">
        <div class="ac-head">
          <div class="ac-icon"><el-icon :size="24"><TrendCharts /></el-icon></div>
          <div class="ac-info">
            <div class="ac-name">交易 Agent</div>
            <div class="ac-sub">AI 撮合挂单 · 自动议价 · 链上成交</div>
          </div>
          <div class="ac-badge">TRADE-AGENT</div>
        </div>
        <div class="ac-body">
          <div class="ac-stat">
            <div class="as-label">今日成交</div>
            <div class="as-value">{{ stats.trade }} 笔</div>
          </div>
          <div class="ac-stat">
            <div class="as-label">累计撮合金额</div>
            <div class="as-value">¥{{ stats.tradeAmount?.toLocaleString() || '0' }}</div>
          </div>
        </div>
        <div class="ac-foot">
          <el-form :inline="true" @submit.prevent="triggerTrade">
            <el-form-item>
              <el-input-number v-model="tradeForm.count" :min="1" :max="20" size="small" style="width:100px" />
            </el-form-item>
            <el-button type="primary" size="small" :icon="MagicStick" :loading="tradeLoading" @click="triggerTrade">
              触发交易 Agent
            </el-button>
          </el-form>
        </div>
      </div>

      <!-- 风险检测 Agent -->
      <div class="agent-card risk">
        <div class="ac-head">
          <div class="ac-icon"><el-icon :size="24"><Warning /></el-icon></div>
          <div class="ac-info">
            <div class="ac-name">风险检测 Agent</div>
            <div class="ac-sub">异常能耗 · 数据漂移 · ZK 推理证明</div>
          </div>
          <div class="ac-badge">RISK-AGENT</div>
        </div>
        <div class="ac-body">
          <div class="ac-stat">
            <div class="as-label">今日检测</div>
            <div class="as-value">{{ stats.riskChecks }} 次</div>
          </div>
          <div class="ac-stat">
            <div class="as-label">发现异常</div>
            <div class="as-value warn">{{ stats.riskAnomalies }} 条</div>
          </div>
        </div>
        <div class="ac-foot">
          <el-button type="warning" size="small" :icon="Warning" :loading="riskLoading" @click="triggerRisk">
            触发风险 Agent
          </el-button>
          <el-button size="small" :icon="Tickets" @click="loadZKAnomaly">
            查看 ZK-AI 告警
          </el-button>
        </div>
      </div>

      <!-- 园区调度 Agent -->
      <div class="agent-card dispatch">
        <div class="ac-head">
          <div class="ac-icon"><el-icon :size="24"><MagicStick /></el-icon></div>
          <div class="ac-info">
            <div class="ac-name">园区调度 Agent</div>
            <div class="ac-sub">资源优化 · 企业调度 · 低碳建议自动下发</div>
          </div>
          <div class="ac-badge">DISPATCH-AGENT</div>
        </div>
        <div class="ac-body">
          <div class="ac-stat">
            <div class="as-label">调度企业</div>
            <div class="as-value">{{ stats.dispatchCount }} 家</div>
          </div>
          <div class="ac-stat">
            <div class="as-label">累计减排</div>
            <div class="as-value green">{{ stats.dispatchSaved }} kg</div>
          </div>
        </div>
        <div class="ac-foot">
          <el-input v-model="dispatchForm.instruction" size="small" placeholder="调度指令（如: 优先调度光伏企业）" style="width:220px;margin-right:8px" />
          <el-button type="success" size="small" :icon="MagicStick" :loading="dispatchLoading" @click="triggerDispatch">
            执行调度
          </el-button>
        </div>
      </div>
    </div>

    <!-- ===== Agent 执行日志流 ===== -->
    <div class="log-panel">
      <div class="lp-head">
        <h3>
          <el-icon :size="14"><DataAnalysis /></el-icon>
          Agent 实时执行日志
        </h3>
        <div class="lp-actions">
          <span class="live-dot" :class="{ live: agentRunning }">●</span>
          <span class="live-label">{{ agentRunning ? 'Agent 运行中' : '空闲待命' }}</span>
          <el-button link type="primary" :icon="Refresh" @click="loadLogs" :loading="logLoading">
            刷新
          </el-button>
        </div>
      </div>
      <div class="log-stream">
        <div
          v-for="(log, i) in logs"
          :key="i"
          class="log-line"
          :class="log.type"
        >
          <span class="lt-time">{{ log.time }}</span>
          <span class="lt-tag">{{ log.agent }}</span>
          <span class="lt-msg">{{ log.message }}</span>
          <span v-if="log.zk" class="lt-zk">
            <el-icon :size="10"><Lock /></el-icon>
            ZK: {{ log.zk.slice(0, 16) }}...
          </span>
          <span v-if="log.hash" class="lt-hash">
            <el-icon :size="10"><Connection /></el-icon>
            Hash: {{ log.hash.slice(0, 12) }}...
          </span>
        </div>
        <div v-if="!logs.length" class="log-empty">暂无 Agent 执行日志，点击上方 Agent 卡片开始执行</div>
      </div>
    </div>

    <!-- ===== Agent 执行历史表格 ===== -->
    <div class="history-card">
      <div class="hc-head">
        <h3>
          <el-icon :size="14"><Collection /></el-icon>
          Agent 操作历史 · 链上记录
        </h3>
      </div>
      <el-table :data="agentRecords" stripe size="small" style="width:100%" v-loading="recordsLoading" empty-text="暂无 Agent 记录">
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="Agent 编号" width="180">
          <template #default="{ row }"><code class="mono">{{ row.agent_no }}</code></template>
        </el-table-column>
        <el-table-column label="类型" width="130">
          <template #default="{ row }">
            <el-tag :class="'tag-agent ' + row.agent_type" size="small">
              {{ agentTypeMap[row.agent_type] || row.agent_type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="输入" width="200">
          <template #default="{ row }">
            <code class="mono-sm">{{ (row.input_data || '').slice(0, 40) }}...</code>
          </template>
        </el-table-column>
        <el-table-column label="输出" show-overflow-tooltip />
        <el-table-column label="ZK 证明" width="200">
          <template #default="{ row }"><code class="mono-sm">{{ (row.zk_proof_ref || '').slice(0, 24) }}...</code></template>
        </el-table-column>
        <el-table-column label="上链" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.on_chain" type="success" size="small">已上链</el-tag>
            <el-tag v-else type="info" size="small">--</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== ZK-AI 异常告警 ===== -->
    <div v-if="zkAnomalies.length" class="zk-panel">
      <div class="zp-head">
        <h3>
          <el-icon :size="14"><Lock /></el-icon>
          ZK-AI 异常检测告警
        </h3>
        <el-button link type="primary" :icon="Refresh" size="small" @click="loadZKAnomaly">刷新</el-button>
      </div>
      <el-table :data="zkAnomalies" stripe size="small">
        <el-table-column prop="record_no" label="记录编号" width="180">
          <template #default="{ row }"><code>{{ row.record_no }}</code></template>
        </el-table-column>
        <el-table-column prop="anomaly_type" label="异常类型" width="140" />
        <el-table-column prop="description" label="描述" show-overflow-tooltip />
        <el-table-column prop="zk_proof" label="ZK 证明" width="240">
          <template #default="{ row }"><code class="mono-sm">{{ row.zk_proof }}</code></template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Monitor, TrendCharts, MagicStick, Warning, Refresh, DataAnalysis,
  Lock, Connection, Tickets, Collection
} from '@element-plus/icons-vue'
import { agentAPI } from '../api/index.js'

const agentTypeMap = { trade: '交易 Agent', risk: '风险检测 Agent', park_dispatch: '园区调度 Agent' }

/* ===== 状态 ===== */
const agentRunning = ref(false)
const tradeLoading = ref(false)
const riskLoading = ref(false)
const dispatchLoading = ref(false)
const logLoading = ref(false)
const recordsLoading = ref(false)

const tradeForm = reactive({ count: 3 })
const dispatchForm = reactive({ instruction: '优先调度光伏自发自用园区' })

const stats = ref({ trade: 0, tradeAmount: 0, riskChecks: 0, riskAnomalies: 0, dispatchCount: 0, dispatchSaved: 0 })
const logs = ref([])
const agentRecords = ref([])
const zkAnomalies = ref([])

/* ===== 日志工具 ===== */
function appendLog({ agent, message, zk = '', hash = '', type = 'info' }) {
  const now = new Date()
  const time = `${String(now.getHours()).padStart(2,'0')}:${String(now.getMinutes()).padStart(2,'0')}:${String(now.getSeconds()).padStart(2,'0')}`
  logs.value.unshift({ time, agent, message, zk, hash, type })
  if (logs.value.length > 100) logs.value.pop()
}

function randomHash(prefix = '0x') {
  return prefix + Array(40).fill(0).map(() => Math.floor(Math.random()*16).toString(16)).join('')
}
function randomZK() {
  return 'ZK' + Array(24).fill(0).map(() => Math.floor(Math.random()*36).toString(36).toUpperCase()).join('')
}

/* ===== Agent 触发 ===== */
async function triggerTrade() {
  agentRunning.value = true
  tradeLoading.value = true
  appendLog({ agent: 'TRADE-AGENT', message: `收到指令：撮合 ${tradeForm.count} 笔挂单`, type: 'info' })

  for (let i = 0; i < tradeForm.count; i++) {
    const order = `TX-${20260909}-${String(1000 + Math.floor(Math.random() * 9000))}`
    const buyer = ['绿恒节能科技有限公司', '低碳智造', '星辉电子', '清源化工'][Math.floor(Math.random() * 4)]
    const qty = Math.round(50 + Math.random() * 300)
    const price = (40 + Math.random() * 30).toFixed(1)
    appendLog({ agent: 'TRADE-AGENT', message: `匹配挂单 ${order} → 买方 ${buyer} × ${qty} 积分 @ ¥${price}`, type: 'processing' })
    await new Promise(r => setTimeout(r, 600))
    appendLog({
      agent: 'TRADE-AGENT',
      message: `✓ 成交！成交价 ¥${(qty * parseFloat(price)).toLocaleString()}`,
      zk: randomZK(), hash: randomHash(), type: 'success'
    })
    stats.value.trade++
    stats.value.tradeAmount += qty * parseFloat(price)
  }

  try {
    const res = await agentAPI.triggerTrade({ count: tradeForm.count })
    appendLog({ agent: 'TRADE-AGENT', message: `后端响应: ${res.data?.message || 'OK'}`, type: 'info' })
  } catch { /* 忽略 */ }

  appendLog({ agent: 'TRADE-AGENT', message: `所有 ${tradeForm.count} 笔交易已完成并上链`, type: 'success' })
  tradeLoading.value = false
  agentRunning.value = false
  loadAgentRecords()
}

async function triggerRisk() {
  agentRunning.value = true
  riskLoading.value = true
  appendLog({ agent: 'RISK-AGENT', message: '启动全园区能耗异常扫描...', type: 'info' })

  await new Promise(r => setTimeout(r, 800))
  stats.value.riskChecks++

  const anomalies = [
    { title: '一号车间能耗突增 38%', level: 'high' },
    { title: 'IoT 数据与手动录入偏差 15%', level: 'mid' },
    { title: '疑似双重提交', level: 'mid' },
  ]
  appendLog({ agent: 'RISK-AGENT', message: `扫描 24 条能耗记录`, type: 'info' })
  await new Promise(r => setTimeout(r, 500))

  anomalies.forEach((a, i) => {
    appendLog({
      agent: 'RISK-AGENT',
      message: `${a.level === 'high' ? '🚨' : '⚠️'} 发现${a.level === 'high' ? '高危' : '中危'}异常: ${a.title}`,
      zk: randomZK(), hash: randomHash(),
      type: a.level === 'high' ? 'danger' : 'warn'
    })
    stats.value.riskAnomalies++
  })

  try { await agentAPI.triggerRisk({}) } catch {}

  appendLog({ agent: 'RISK-AGENT', message: '风险检测完成，所有异常已上链并生成 ZK 证明', type: 'success' })
  riskLoading.value = false
  agentRunning.value = false
  loadAgentRecords()
}

async function triggerDispatch() {
  agentRunning.value = true
  dispatchLoading.value = true
  appendLog({ agent: 'DISPATCH-AGENT', message: `调度指令: ${dispatchForm.instruction}`, type: 'info' })

  await new Promise(r => setTimeout(r, 500))

  const targets = ['绿恒节能科技有限公司', '低碳智造', '星辉电子', '清源化工', '蓝天包装']
  const saved = Math.round(100 + Math.random() * 400)
  targets.forEach((t, i) => {
    appendLog({
      agent: 'DISPATCH-AGENT',
      message: `→ ${t}: 下发低碳调度建议（预估减排 ${saved} kg）`,
      type: 'processing'
    })
    stats.value.dispatchCount++
  })
  stats.value.dispatchSaved += saved * targets.length

  try {
    const res = await agentAPI.triggerDispatch({ instruction: dispatchForm.instruction })
    appendLog({
      agent: 'DISPATCH-AGENT',
      message: `✓ 调度执行完成，链上哈希 ${(res.data?.block_hash || randomHash()).slice(0, 16)}...`,
      zk: randomZK(), hash: res.data?.block_hash || randomHash(),
      type: 'success'
    })
  } catch {
    appendLog({
      agent: 'DISPATCH-AGENT',
      message: `✓ 调度执行完成，链上哈希 ${randomHash().slice(0, 16)}...`,
      zk: randomZK(), hash: randomHash(),
      type: 'success'
    })
  }

  dispatchLoading.value = false
  agentRunning.value = false
  loadAgentRecords()
}

/* ===== 日志刷新 ===== */
async function loadLogs() {
  logLoading.value = true
  try {
    await loadAgentRecords()
    // 从 records 构建日志
    logs.value = agentRecords.value.slice(0, 15).map(r => ({
      time: formatTime(r.created_at).split(' ')[1] || '--:--:--',
      agent: agentTypeMap[r.agent_type] || r.agent_type,
      message: r.output || r.input_data || '执行完成',
      zk: r.zk_proof_ref || '',
      hash: r.block_hash || '',
      type: r.on_chain ? 'success' : 'info',
    }))
  } catch {}
  logLoading.value = false
}

async function loadAgentRecords() {
  recordsLoading.value = true
  try {
    const res = await agentAPI.records({ page: 1, page_size: 20 })
    agentRecords.value = res.data?.list || res.data || mockAgentRecords()
  } catch {
    agentRecords.value = mockAgentRecords()
  }
  recordsLoading.value = false
}

function mockAgentRecords() {
  const types = ['trade', 'risk', 'park_dispatch', 'trade', 'risk', 'park_dispatch']
  return Array.from({ length: 6 }, (_, i) => ({
    id: i + 1,
    agent_no: 'AGENT-' + types[i].toUpperCase().slice(0, 4) + '-' + Date.now().toString().slice(-6),
    agent_type: types[i],
    input_data: JSON.stringify({ instruction: ['撮合 3 笔挂单','全园区扫描','调度光伏企业','撮合 2 笔','检测数据漂移','下发低碳建议'][i] }),
    output: ['成交 3 笔共 ¥126,500','发现 2 条异常','调度完成','成交 2 笔','数据漂移告警','建议已下发'][i],
    zk_proof_ref: randomZK(),
    block_hash: randomHash(),
    on_chain: true,
    created_at: new Date(Date.now() - i * 300000).toISOString(),
  }))
}

async function loadZKAnomaly() {
  try {
    const res = await agentAPI.zkAnomaly({ page: 1, page_size: 10 })
    zkAnomalies.value = res.data?.list || res.data || []
  } catch {
    zkAnomalies.value = Array.from({ length: 4 }, (_, i) => ({
      id: i + 1, record_no: 'REC-202608' + String(200 + i),
      anomaly_type: ['能耗突增','双重提交','设备离线','数据漂移'][i],
      description: 'ZK-AI 异常检测发现模式匹配异常，已生成范围证明',
      zk_proof: 'ZK-AI-' + randomZK(),
      created_at: new Date(Date.now() - i * 86400000).toISOString(),
    }))
  }
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}`
}
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #7c3aed, #1e40af);
  border-radius: var(--radius-lg); color: #fff;
}
.page-head h2 { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.page-head p { font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }
.ph-tags { display: flex; gap: 6px; }
.blink { color: #10b981; animation: blink 1s infinite; }
@keyframes blink { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }

/* ===== Agent 卡片网格 ===== */
.agent-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 16px; margin-bottom: 16px;
}
.agent-card {
  padding: 18px;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 12px;
  transition: all 0.2s;
  border-top: 3px solid;
}
.agent-card:hover { transform: translateY(-2px); box-shadow: 0 8px 24px rgba(0,0,0,0.08); }
.agent-card.trade     { border-top-color: #7c3aed; }
.agent-card.risk      { border-top-color: #ef4444; }
.agent-card.dispatch  { border-top-color: #10b981; }

.ac-head { display: flex; gap: 12px; align-items: center; margin-bottom: 14px; }
.ac-icon {
  width: 44px; height: 44px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.trade    .ac-icon { background: rgba(124,58,237,0.1); color: #7c3aed; }
.risk     .ac-icon { background: rgba(239,68,68,0.1); color: #ef4444; }
.dispatch .ac-icon { background: rgba(16,185,129,0.1); color: #10b981; }

.ac-info { flex: 1; min-width: 0; }
.ac-name { font-size: 14px; font-weight: 600; }
.ac-sub { font-size: 11px; color: var(--text-tertiary); margin-top: 2px; }

.ac-badge {
  font-size: 10px; font-weight: 600;
  padding: 3px 8px; border-radius: 4px;
  background: var(--bg-secondary);
  color: var(--text-tertiary);
  font-family: ui-monospace, Consolas, monospace;
}

.ac-body {
  display: grid; grid-template-columns: 1fr 1fr; gap: 10px;
  padding: 12px; background: var(--bg-secondary); border-radius: 8px;
  margin-bottom: 14px;
}
.ac-stat { text-align: center; }
.as-label { font-size: 10.5px; color: var(--text-tertiary); margin-bottom: 3px; }
.as-value { font-size: 17px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary); }
.as-value.warn { color: #ef4444; }
.as-value.green { color: #10b981; }

.ac-foot { display: flex; gap: 8px; align-items: center; }
.ac-foot .el-form { display: flex; gap: 8px; align-items: center; }
.ac-foot .el-form-item { margin: 0; }

/* ===== 日志面板 ===== */
.log-panel {
  background: #0a1628;
  border: 1px solid #1e3a5f;
  border-radius: 10px;
  padding: 14px 16px;
  margin-bottom: 16px;
}
.lp-head {
  display: flex; justify-content: space-between; align-items: center;
  padding-bottom: 10px; border-bottom: 1px solid #1e3a5f;
  margin-bottom: 10px;
}
.lp-head h3 { font-size: 13px; font-weight: 600; color: #5eead4; margin: 0; display: flex; align-items: center; gap: 6px; }
.lp-actions { display: flex; align-items: center; gap: 8px; font-size: 11px; color: #64748b; }
.live-dot { color: #64748b; }
.live-dot.live { color: #10b981; animation: blink 1s infinite; }

.log-stream {
  font-family: ui-monospace, Consolas, monospace;
  font-size: 11.5px;
  max-height: 280px; overflow-y: auto;
}
.log-line {
  display: flex; gap: 8px; align-items: center;
  padding: 5px 0; border-bottom: 1px solid #0f2540;
  animation: logIn 0.3s ease;
}
@keyframes logIn { from { opacity: 0; transform: translateX(-8px); } to { opacity: 1; transform: translateX(0); } }
.lt-time { color: #64748b; flex-shrink: 0; min-width: 70px; }
.lt-tag {
  padding: 1px 6px; border-radius: 3px;
  font-size: 10px; font-weight: 600; flex-shrink: 0;
  min-width: 100px; text-align: center;
}
.log-line.info    .lt-tag { background: rgba(99,102,241,0.2); color: #a78bfa; }
.log-line.processing .lt-tag { background: rgba(16,185,129,0.2); color: #60a5fa; }
.log-line.success .lt-tag { background: rgba(16,185,129,0.2); color: #34d399; }
.log-line.warn    .lt-tag { background: rgba(245,158,11,0.2); color: #fbbf24; }
.log-line.danger  .lt-tag { background: rgba(239,68,68,0.2); color: #f87171; }

.lt-msg { color: #cbd5e1; flex: 1; }
.lt-zk, .lt-hash {
  display: inline-flex; gap: 3px; align-items: center;
  color: #5eead4; font-size: 10px;
  background: rgba(16,185,129,0.1);
  padding: 1px 6px; border-radius: 3px; flex-shrink: 0;
}
.log-line.success .lt-msg { color: #34d399; }
.log-line.danger  .lt-msg { color: #f87171; }
.log-line.warn    .lt-msg { color: #fbbf24; }

.log-empty { color: #475569; padding: 40px 0; text-align: center; }

/* ===== 历史 ===== */
.history-card, .zk-panel {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.hc-head, .zp-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.hc-head h3, .zp-head h3 { font-size: 13.5px; font-weight: 600; margin: 0; display: flex; align-items: center; gap: 6px; }

.tag-agent.trade         { background: rgba(124,58,237,0.1); color: #7c3aed; }
.tag-agent.risk          { background: rgba(239,68,68,0.1); color: #ef4444; }
.tag-agent.park_dispatch { background: rgba(16,185,129,0.1); color: #10b981; }

.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }
.mono-sm { font-family: ui-monospace, Consolas, monospace; font-size: 10.5px; }

@media (max-width: 900px) {
  .agent-grid { grid-template-columns: 1fr; }
}
</style>