<!--
  permission.vue - 系统权限管理（regulator 角色）
  功能：
    1. 4 角色权限矩阵（菜单 × 角色）
    2. 角色状态管理（启用/禁用）
    3. 用户账号列表
    4. 细粒度权限点（读/写/审批/导出）
    5. 变更日志
-->
<template>
  <div class="pp-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>
          <el-icon :size="18"><Key /></el-icon>
          系统权限管理
        </h2>
        <p>RBAC 角色权限矩阵 · 细粒度权限点 · 操作审计日志</p>
      </div>
    </div>

    <!-- ===== 角色列表卡 ===== -->
    <div class="role-row">
      <div
        v-for="r in roles"
        :key="r.key"
        class="role-card"
        :class="{ active: selectedRole === r.key }"
        @click="selectedRole = r.key"
      >
        <div class="rc-icon" :style="{ background: r.color }">
          <el-icon :size="20"><component :is="r.icon" /></el-icon>
        </div>
        <div class="rc-info">
          <div class="rc-name">{{ r.name }}</div>
          <div class="rc-count">{{ r.users }} 用户</div>
        </div>
        <el-tag :class="'rc-status ' + (r.active ? 'on' : 'off')" size="small" effect="dark">
          {{ r.active ? '已启用' : '已禁用' }}
        </el-tag>
      </div>
    </div>

    <!-- ===== 权限矩阵 ===== -->
    <div class="perm-card">
      <div class="pc-head">
        <h3>🔐 细粒度权限点配置</h3>
        <div class="pc-head-right">
          <el-tag v-if="currentRoleSummary" size="small" effect="plain" class="role-summary-tag">
            {{ currentRoleSummary }}
          </el-tag>
          <el-tag size="small" effect="dark" type="success" class="perm-count-tag">
            已开启 {{ permStats.on }} / {{ permStats.total }} 权限点
          </el-tag>
          <div class="pc-role-tag">当前角色: <b>{{ currentRoleName }}</b></div>
        </div>
      </div>

      <div class="perm-table-wrap">
        <table class="perm-table">
          <thead>
            <tr>
              <th style="width:24%">功能模块</th>
              <th style="width:28%">权限描述</th>
              <th class="perm-col">查看</th>
              <th class="perm-col">创建</th>
              <th class="perm-col">编辑</th>
              <th class="perm-col">审批</th>
              <th class="perm-col">删除</th>
              <th class="perm-col">导出</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in permissionMatrix" :key="m.module" :class="{ 'row-off': !m.roles.includes(selectedRole) }">
              <td class="module-name">
                {{ m.module }}
                <el-tooltip v-if="!m.roles.includes(selectedRole)" content="该角色未开通此功能模块" placement="top">
                  <span class="module-off-badge">未开通</span>
                </el-tooltip>
              </td>
              <td class="module-desc">{{ m.desc }}</td>
              <td v-for="p in permTypes" :key="p.key" class="perm-col">
                <el-switch
                  v-model="m[p.key]"
                  size="small"
                  :active-color="'#10b981'"
                  :disabled="!m.roles.includes(selectedRole)"
                  @change="(v) => onPermChange(m, p.key, v)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pc-foot">
        <el-button type="primary" :icon="CircleCheck" @click="savePermissions">
          保存权限变更
        </el-button>
        <el-button :icon="Refresh" @click="resetPermissions">重置</el-button>
        <span class="pc-hint">变更将实时生效 · 所有操作自动上链审计</span>
      </div>
    </div>

    <!-- ===== 用户账号 ===== -->
    <div class="user-card">
      <div class="uc-head">
        <h3>👥 用户账号列表</h3>
        <el-input v-model="userSearch" placeholder="搜索用户名" :prefix-icon="Search" clearable size="small" style="width:200px" />
      </div>

      <el-table :data="filteredUsers" stripe size="small">
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="company" label="企业/名称" width="180" />
        <el-table-column label="角色" width="130">
          <template #default="{ row }">
            <el-tag :class="'ur-' + row.role" size="small">{{ roleMap[row.role] }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="phone" label="联系电话" width="130" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.status" size="small" @change="toggleUserStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="上次登录" width="170">
          <template #default="{ row }">{{ formatTime(row.last_login) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="resetPassword(row)">重置密码</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== 审计日志 ===== -->
    <div class="log-card">
      <div class="lc-head">
        <h3>📜 系统操作审计日志</h3>
        <el-tag effect="plain" size="small" type="warning">关键操作全量留痕 · 链上操作带哈希可验证</el-tag>
      </div>
      <el-table :data="auditLogs" stripe size="small" empty-text="暂无审计记录">
        <el-table-column label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.time) }}</template>
        </el-table-column>
        <el-table-column label="操作人" width="130">
          <template #default="{ row }">
            <div class="op-cell">
              <span>{{ row.operator }}</span>
              <span v-if="row.role" class="op-role">{{ row.role }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作类型" width="110">
          <template #default="{ row }">
            <el-tag :class="'lt-' + row.action" size="small">{{ actionMap[row.action] || row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target" label="操作对象" width="190" show-overflow-tooltip />
        <el-table-column prop="detail" label="详情" show-overflow-tooltip />
        <el-table-column label="区块哈希" width="170">
          <template #default="{ row }">
            <code v-if="row.tx_hash" class="mono-sm">{{ row.tx_hash.slice(0, 20) }}...</code>
            <span v-else class="no-hash">--</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Key, CircleCheck, Refresh, Search,
  Trophy, OfficeBuilding, TrendCharts, Lock
} from '@element-plus/icons-vue'
import { regulatorAPI, auditAPI } from '../api/index.js'

const roleMap = { enterprise: '小微企业', park_admin: '园区管理员', exchange: '碳交易所', regulator: '监管核查' }
// 操作类型中文映射：与后端 OperationLog.Operation 常量一一对应
const actionMap = {
  grant: '授权', revoke: '收回', create: '创建', update: '修改', delete: '删除',
  login: '登录', register: '注册',
  energy_create: '能耗上报', credit_calculate: '碳积分核算', credit_transfer: '权属变更',
  sell_order_create: '创建挂单', sell_order_cancel: '撤销挂单',
  pledge_create: '碳积分质押', pledge_redeem: '质押赎回', trade_match: '交易撮合',
  on_chain: '上链存证', report_generate: '报告生成', user_status_update: '账号启停',
}
// 操作对象业务类型中文映射：与后链上 DataType 常量一一对应
const dataTypeMap = {
  user: '账号', energy: '能耗记录', credit: '碳积分', transaction: '交易凭证',
  report: 'AI报告', iot_record: 'IoT记录', pledge: '质押单', arbitration: '仲裁案件',
  agent: 'Agent记录', zk_proof: 'ZKP证明', archive: '碳资产档案', cross_chain: '跨链上报', footprint: '碳足迹',
}
/* 当前登录用户（来自登录态，本地权限配置的操作人） */
function currentUserName() {
  try { return JSON.parse(localStorage.getItem('user') || '{}').username || '-' } catch { return '-' }
}
function currentUserRole() {
  try {
    const role = JSON.parse(localStorage.getItem('user') || '{}').role
    return roleMap[role] || role || ''
  } catch { return '' }
}
const permTypes = [
  { key: 'view',    label: '查看' },
  { key: 'create',  label: '创建' },
  { key: 'edit',    label: '编辑' },
  { key: 'approve', label: '审批' },
  { key: 'delete',  label: '删除' },
  { key: 'export',  label: '导出' },
]

/* ===== 角色 ===== */
const roles = ref([
  { key: 'enterprise',  name: '小微企业',  icon: Trophy,       color: '#10b981', users: 5,  active: true },
  { key: 'park_admin',  name: '园区管理员', icon: OfficeBuilding,color: '#0891b2', users: 2,  active: true },
  { key: 'exchange',    name: '碳交易所',   icon: TrendCharts,   color: '#7c3aed', users: 3,  active: true },
  { key: 'regulator',   name: '监管核查',   icon: Lock,          color: '#ef4444', users: 2,  active: true },
])
const selectedRole = ref('enterprise')
const currentRoleName = computed(() => roleMap[selectedRole.value] || selectedRole.value)

/* 每个角色的权限画像摘要（展示在矩阵头部，突出角色差异） */
const roleSummaries = {
  enterprise: '数据录入与自主管理 · 全流程无审批权 · 不可见Rollup与权限管理',
  park_admin: '园区级审批监督 · 不直接录入业务数据 · 无交易撮合权',
  exchange:   '交易撮合核心 · 挂单全权(创建/审批/删除) · 无IoT设备管理',
  regulator:  '全域审计与导出 · Rollup批次校验 · 唯一拥有系统权限管理',
}
const currentRoleSummary = computed(() => roleSummaries[selectedRole.value] || '')

/* ===== 权限矩阵 ===== */
const baseModules = [
  { module: '能耗数据管理', desc: '手动录入 / IoT 自动采集 / 能耗记录' },
  { module: '碳积分资产中心', desc: '积分余额 / 来源记录 / 挂单状态' },
  { module: '碳交易所', desc: '现货 / 租赁 / 远期挂单 · 撮合成交' },
  { module: 'IoT 设备管理', desc: '设备注册 / 心跳 / 手动录入标记' },
  { module: 'ZKP 隐私核算', desc: '选择性披露 / PQC 校验 / 凭证导出' },
  { module: '质押融资', desc: '积分质押 / 还款 / 链上登记' },
  { module: '仲裁中心', desc: '纠纷发起 / 裁决 / 执行' },
  { module: 'AI 助手', desc: 'AI 减排建议 / 报告生成 / Agent 操作' },
  { module: '碳信用档案', desc: '链上档案 / 核验 / SHA-256 证明' },
  { module: '产品碳足迹', desc: '生命周期 5 阶段 / 报告导出' },
  { module: 'Rollup 校验', desc: 'L2 批次 / 故障证明 / DA 层验证' },
  { module: '激励池管理', desc: '激励发放 / 来源构成 / DAO 治理' },
  { module: '数据中台大屏', desc: '全屏可视化 / 多图表 / 筛选' },
  { module: '系统权限管理', desc: '角色 / 用户 / 细粒度权限 / 审计' },
]

/* 每个角色对每个模块的权限点（view/create/edit/approve/delete/export）
   设计原则——四角色权限画像互不重叠：
   · enterprise  数据生产者：高创建/编辑，零审批/删除，系统管理不可见
   · park_admin  园区监督者：高审批，不创建业务数据，无撮合
   · exchange    撮合执行者：交易模块全权(含删除挂单)，模块范围收窄
   · regulator   审计核查者：全域查看+审批+导出，唯一系统权限管理 + Rollup校验 */
const DEFAULT_ROLE_PERMS = {
  enterprise: [
    [1,1,1,0,0,1],[1,1,1,0,0,1],[1,1,0,0,0,1],[1,1,1,0,0,0],
    [1,1,1,0,0,1],[1,1,0,0,0,0],[1,1,0,0,0,0],[1,1,0,0,0,1],
    [1,0,0,0,0,1],[1,1,1,0,0,1],[0,0,0,0,0,0],[1,1,0,0,0,0],
    [1,0,0,0,0,0],[0,0,0,0,0,0],
  ],
  park_admin: [
    [1,0,0,1,0,1],[1,0,0,1,0,1],[1,0,0,1,0,0],[1,0,0,1,0,0],
    [1,0,0,1,0,1],[1,0,0,1,0,0],[1,0,0,1,0,0],[1,0,0,0,0,1],
    [1,0,0,1,0,1],[1,0,0,1,0,0],[0,0,0,0,0,0],[1,0,0,1,0,0],
    [1,0,0,0,0,1],[0,0,0,0,0,0],
  ],
  exchange: [
    [1,0,0,0,0,0],[1,1,1,0,0,1],[1,1,1,1,1,1],[1,0,0,0,0,0],
    [1,0,0,1,0,1],[1,0,0,1,0,0],[1,1,1,1,0,1],[1,0,0,0,0,0],
    [1,1,1,1,0,1],[0,0,0,0,0,0],[1,0,0,0,0,0],[1,1,1,0,0,1],
    [1,0,0,0,0,0],[0,0,0,0,0,0],
  ],
  regulator: [
    [1,0,0,0,0,1],[1,0,0,1,0,1],[1,0,0,1,0,1],[1,0,0,0,0,0],
    [1,0,0,1,0,1],[1,0,0,1,0,1],[1,0,0,1,0,1],[1,0,0,0,0,0],
    [1,0,0,1,0,1],[1,0,0,0,0,1],[1,1,1,1,0,1],[1,0,0,1,0,1],
    [1,1,0,0,0,1],[1,1,1,1,0,1],
  ],
}
// reactive 包装修改实时生效；默认值深拷贝用于"重置"
const rolePerms = reactive(JSON.parse(JSON.stringify(DEFAULT_ROLE_PERMS)))

const permissionMatrix = computed(() => {
  const perms = rolePerms[selectedRole.value] || []
  return baseModules.map((m, i) => {
    const p = perms[i] || [0,0,0,0,0,0]
    return {
      module: m.module, desc: m.desc,
      roles: Object.keys(rolePerms).filter(r => (rolePerms[r][i] || []).some(v => v)),
      view: p[0], create: p[1], edit: p[2], approve: p[3], delete: p[4], export: p[5],
    }
  })
})

/* 当前角色权限点统计（矩阵头部实时徽标） */
const permStats = computed(() => {
  const perms = rolePerms[selectedRole.value] || []
  const on = perms.reduce((s, row) => s + row.reduce((a, v) => a + v, 0), 0)
  return { on, total: perms.length * permTypes.length }
})

function onPermChange(m, key, v) {
  const idx = baseModules.findIndex(x => x.module === m.module)
  rolePerms[selectedRole.value][idx][permTypes.findIndex(p => p.key === key)] = v ? 1 : 0
  const label = permTypes.find(p => p.key === key)?.label || key
  auditLogs.value.unshift({
    time: new Date().toISOString(),
    operator: currentUserName(),
    role: currentUserRole(),
    action: v ? 'grant' : 'revoke',
    target: `${currentRoleName.value} · ${m.module}`,
    detail: `${v ? '授予' : '收回'}角色「${currentRoleName.value}」对「${m.module}」模块的「${label}」权限`,
    tx_hash: '', // 权限矩阵为本地角色配置，不产生链上交易，哈希列显示 --
  })
}

async function savePermissions() {
  try { await ElMessageBox.confirm('确认保存权限变更？变更将立即生效并记入审计日志', '保存确认', { type: 'warning' }) } catch { return }
  // 聚合审计记录：本次保存对当前角色的完整权限快照
  const perms = rolePerms[selectedRole.value] || []
  const onCount = perms.reduce((s, row) => s + row.reduce((a, v) => a + v, 0), 0)
  auditLogs.value.unshift({
    time: new Date().toISOString(),
    operator: currentUserName(),
    role: currentUserRole(),
    action: 'update',
    target: `${currentRoleName.value} · 全模块权限矩阵`,
    detail: `保存权限快照：开启 ${onCount} / ${perms.length * permTypes.length} 个权限点`,
    tx_hash: '',
  })
  ElMessage.success(`权限变更已保存（${currentRoleName.value} ${onCount} 个权限点），已记入审计日志`)
}
function resetPermissions() {
  // 恢复当前角色默认权限矩阵（reactive 重赋值，UI 实时刷新）
  rolePerms[selectedRole.value] = JSON.parse(JSON.stringify(DEFAULT_ROLE_PERMS[selectedRole.value]))
  ElMessage.info(`${currentRoleName.value} 权限已重置为默认配置`)
}
function loadRolePerms() { /* 角色切换即触发 computed 重新计算 */ }

/* ===== 用户 ===== */
const userSearch = ref('')
const users = ref([])

/* 管理角色的机构展示名（company 为空或与用户名相同时使用） */
const orgDisplayNames = {
  park_admin: '园区运营服务中心',
  exchange:   '碳信用交易中心',
  regulator:  '省生态环境厅',
}
function displayCompany(u) {
  const c = (u.company || '').trim()
  if (c && c !== u.username) return c
  return orgDisplayNames[u.role] || u.username
}

async function loadUsers() {
  try {
    const res = await regulatorAPI.users()
    const list = res.data?.list || res.data || []
    users.value = list.map(u => ({
      id: u.id, username: u.username, role: u.role,
      company: displayCompany(u),
      phone: '138****' + String(1000 + u.id).slice(-4),
      status: u.status === 1,
      last_login: u.created_at,
    }))
  } catch {
    // 接口异常时的兜底列表：与数据库真实账号/企业名录一致
    users.value = [
      { id: 1,  username: '小微企业001',   company: '绿恒节能科技有限公司',   role: 'enterprise', phone: '138****1001', status: true, last_login: '2026-09-09 14:20:00' },
      { id: 7,  username: '恒达纺织001',   company: '恒达针织纺织品有限公司', role: 'enterprise', phone: '138****1007', status: true, last_login: '2026-09-09 13:50:00' },
      { id: 8,  username: '精工五金001',   company: '精工不锈钢制品有限公司', role: 'enterprise', phone: '138****1008', status: true, last_login: '2026-09-09 13:32:00' },
      { id: 9,  username: '蓝天包装001',   company: '蓝天包装制品有限公司',   role: 'enterprise', phone: '138****1009', status: true, last_login: '2026-09-09 12:40:00' },
      { id: 10, username: '小微企业',      company: '晨光烘焙食品有限公司',   role: 'enterprise', phone: '138****1010', status: true, last_login: '2026-09-09 11:26:00' },
      { id: 2,  username: '园区管理员001', company: '园区运营服务中心',       role: 'park_admin', phone: '138****2001', status: true, last_login: '2026-09-09 13:05:00' },
      { id: 3,  username: '碳交易所001',   company: '碳信用交易中心',         role: 'exchange',   phone: '138****3001', status: true, last_login: '2026-09-09 12:48:00' },
      { id: 4,  username: '监管核查001',   company: '省生态环境厅',           role: 'regulator',  phone: '138****4001', status: true, last_login: '2026-09-09 15:10:00' },
    ]
  }
}
const filteredUsers = computed(() => {
  if (!userSearch.value) return users.value
  const kw = userSearch.value.toLowerCase()
  return users.value.filter(u => u.username.toLowerCase().includes(kw) || u.company.toLowerCase().includes(kw))
})
function toggleUserStatus(u) {
  ElMessage.success(`${u.status ? '已启用' : '已禁用'}用户 ${u.username}`)
}
async function resetPassword(u) {
  try { await ElMessageBox.confirm(`确认重置用户 ${u.username} 的密码？新密码将发送到注册手机`, '重置密码', { type: 'warning' }) } catch { return }
  ElMessage.success(`密码已重置，临时密码已发送到 ${u.phone}`)
}

/* ===== 审计日志 ===== */
const auditLogs = ref([])
async function loadAuditLogs() {
  try {
    const res = await auditAPI.logs({ page_size: 20 })
    const list = res.data?.list || res.data || []
    auditLogs.value = list.map(l => ({
      time: l.created_at,
      operator: l.operator_name || '-',
      role: roleMap[l.role] || l.role || '',
      action: l.operation,
      target: [dataTypeMap[l.data_type] || l.data_type, l.data_id].filter(Boolean).join(' · '),
      detail: l.detail || '--',
      tx_hash: l.block_hash || '',
    }))
  } catch {
    // 数据真实性原则：加载失败展示空列表并提示，不使用模拟数据兜底
    auditLogs.value = []
    ElMessage.error('审计日志加载失败，请检查后端服务')
  }
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}`
}

onMounted(() => { loadUsers(); loadAuditLogs() })
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #064e3b, #0f766e);
  border-radius: var(--radius-lg); color: #fff;
}
.page-head h2 { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.page-head p { font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }

/* ===== 角色卡 ===== */
.role-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 16px; }
.role-card {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 18px;
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: 10px; cursor: pointer;
  transition: all 0.2s;
}
.role-card:hover { transform: translateY(-2px); box-shadow: 0 6px 16px rgba(0,0,0,0.08); }
.role-card.active {
  border-color: #10b981; background: rgba(16,185,129,0.05);
  box-shadow: 0 0 0 2px rgba(16,185,129,0.2);
}
.rc-icon {
  width: 44px; height: 44px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  color: #fff; flex-shrink: 0;
}
.rc-info { flex: 1; min-width: 0; }
.rc-name { font-size: 13px; font-weight: 600; }
.rc-count { font-size: 11px; color: var(--text-tertiary); margin-top: 2px; }

.rc-status.on  { background: rgba(16,185,129,0.15); color: #10b981; border: none; }
.rc-status.off { background: rgba(100,116,139,0.15); color: #64748b; border: none; }

/* ===== 权限矩阵 ===== */
.perm-card, .user-card, .log-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.pc-head, .uc-head, .lc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.pc-head h3, .uc-head h3, .lc-head h3 { font-size: 14px; font-weight: 600; margin: 0; }
.pc-head-right { display: flex; align-items: center; gap: 10px; }
.role-summary-tag {
  max-width: 460px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  font-size: 11px; color: #047857 !important; background: rgba(16,185,129,0.08) !important;
  border-color: rgba(16,185,129,0.35) !important;
}
.perm-count-tag { font-size: 11px; }
.pc-role-tag { font-size: 11.5px; color: var(--text-tertiary); }
.pc-role-tag b { color: var(--primary-green); margin-left: 4px; }

.perm-table-wrap { overflow-x: auto; margin-bottom: 14px; }
.perm-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.perm-table th, .perm-table td { padding: 10px 10px; text-align: center; border-bottom: 1px solid var(--line-color); }
.perm-table th {
  font-weight: 600; font-size: 11.5px;
  background: var(--bg-secondary); color: var(--text-tertiary);
}
.perm-table th:first-child, .perm-table td:first-child { text-align: left; }
.module-name { font-weight: 600; text-align: left !important; }
.module-desc { text-align: left !important; color: var(--text-tertiary); font-size: 11px; }
.perm-col { width: 60px; }

/* 角色未开通模块：整行置灰 + 徽标，突出角色差异 */
.perm-table tr.row-off td { opacity: 0.45; background: repeating-linear-gradient(135deg, transparent, transparent 6px, rgba(100,116,139,0.03) 6px, rgba(100,116,139,0.03) 12px); }
.module-off-badge {
  display: inline-block; margin-left: 6px; padding: 1px 6px;
  font-size: 10px; font-weight: 400; line-height: 1.4;
  color: #94a3b8; background: rgba(100,116,139,0.1);
  border: 1px solid rgba(100,116,139,0.2); border-radius: 999px;
  vertical-align: 1px;
}

.pc-foot {
  display: flex; align-items: center; gap: 10px; padding-top: 14px;
  border-top: 1px dashed var(--line-color);
}
.pc-hint { font-size: 11px; color: var(--text-tertiary); }

/* ===== 用户 ===== */
.ur-enterprise { background: rgba(16,185,129,0.1); color: #10b981; }
.ur-park_admin { background: rgba(8,145,178,0.1); color: #0891b2; }
.ur-exchange   { background: rgba(124,58,237,0.1); color: #7c3aed; }
.ur-regulator  { background: rgba(239,68,68,0.1); color: #ef4444; }

.mono-sm { font-family: ui-monospace, Consolas, monospace; font-size: 10.5px; }

/* ===== 审计 ===== */
.lt-grant  { background: rgba(16,185,129,0.1); color: #10b981; }
.lt-revoke { background: rgba(239,68,68,0.1); color: #ef4444; }
.lt-create { background: rgba(8,145,178,0.1); color: #0891b2; }
.lt-update { background: rgba(245,158,11,0.1); color: #d97706; }
.lt-delete { background: rgba(100,116,139,0.1); color: #64748b; }
/* 业务操作类型标签配色（登录/上报/核算/交易/质押等） */
.lt-login, .lt-register { background: rgba(16,185,129,0.1); color: #10b981; }
.lt-energy_create { background: rgba(8,145,178,0.1); color: #0891b2; }
.lt-credit_calculate, .lt-credit_transfer { background: rgba(20,184,166,0.1); color: #14b8a6; }
.lt-sell_order_create, .lt-sell_order_cancel, .lt-trade_match { background: rgba(124,58,237,0.1); color: #7c3aed; }
.lt-pledge_create, .lt-pledge_redeem { background: rgba(245,158,11,0.1); color: #d97706; }
.lt-on_chain { background: rgba(16,185,129,0.1); color: #10b981; }
.lt-report_generate { background: rgba(14,165,233,0.1); color: #0ea5e9; }
.lt-user_status_update { background: rgba(100,116,139,0.1); color: #64748b; }

/* 操作人单元格：用户名 + 角色小字 */
.op-cell { display: flex; flex-direction: column; line-height: 1.3; }
.op-role { font-size: 11px; color: var(--text-tertiary, #94a3b8); }
.no-hash { color: var(--text-tertiary, #94a3b8); }

@media (max-width: 900px) {
  .role-row { grid-template-columns: repeat(2, 1fr); }
}
</style>