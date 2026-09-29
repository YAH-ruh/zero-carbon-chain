<!--
  iot-devices.vue - IoT 能耗设备管理（仅 enterprise 角色）
  功能：
    1. 设备卡片网格：在线/离线状态、实时读数、最后心跳、设备类型
    2. 注册新设备弹窗
    3. 实时数据流 Tab / 手动录入 Tab
    4. 每条记录带风险标签（手动录入 = 风险标记）
  后端接口：
    GET  /api/iot/devices           → ListIoTDevices
    POST /api/iot/devices/register  → RegisterIoTDevice
    POST /api/iot/manual-record     → CreateManualRecord
    GET  /api/iot/records           → ListIoTRecords
-->
<template>
  <div class="iot-page page">

    <!-- ===== 页面头 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>IoT 能耗设备管理</h2>
        <p class="ph-desc">自动采集 + 手动录入双模式，数据实时上链存证</p>
      </div>
      <div class="ph-stats">
        <div class="ps-item">
          <el-icon :size="16"><Cpu /></el-icon>
          <span><b>{{ devices.length }}</b> 台设备</span>
        </div>
        <div class="ps-item online">
          <el-icon :size="16"><Connection /></el-icon>
          <span><b>{{ onlineCount }}</b> 在线</span>
        </div>
        <div class="ps-item offline">
          <el-icon :size="16"><Clock /></el-icon>
          <span><b>{{ devices.length - onlineCount }}</b> 离线</span>
        </div>
        <el-button type="primary" class="reg-btn" :icon="Cpu" @click="openRegister">
          + 注册新设备
        </el-button>
      </div>
    </div>

    <!-- ===== 设备卡片网格 ===== -->
    <div class="dev-section">
      <div class="dev-head">
        <h3>我的 IoT 设备</h3>
        <div class="dev-filters">
          <el-radio-group v-model="filterStatus" size="small">
            <el-radio-button label="all">全部</el-radio-button>
            <el-radio-button label="online">在线</el-radio-button>
            <el-radio-button label="offline">离线</el-radio-button>
          </el-radio-group>
          <el-button :icon="Refresh" @click="refreshAll" :loading="loadingAll">刷新</el-button>
        </div>
      </div>

      <div v-if="loadingDevices" class="dev-loading">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
      </div>
      <div v-else-if="!filteredDevices.length" class="dev-empty">
        <el-icon :size="40"><Cpu /></el-icon>
        <p>暂无 IoT 设备，请点击右上角注册</p>
      </div>
      <div v-else class="dev-grid">
        <div v-for="d in filteredDevices" :key="d.device_id" class="dev-card" :class="{ offline: !d.online }">
          <!-- 卡片头部 -->
          <div class="dc-head">
            <div class="dc-icon">
              <el-icon :size="20" :class="d.online ? 'online' : 'offline'">
                <Cpu />
              </el-icon>
              <span class="pulse" :class="d.online ? 'pulse-on' : ''"></span>
            </div>
            <div class="dc-info">
              <div class="dc-name">{{ d.device_name }}</div>
              <div class="dc-meta">
                <code>{{ d.device_id }}</code>
                <el-tag :class="'dtag ' + d.device_type" size="small" effect="plain">{{ deviceTypeMap[d.device_type] || d.device_type }}</el-tag>
              </div>
            </div>
            <div class="dc-status" :class="d.online ? 'online' : 'offline'">
              <span class="pulse"></span>
              {{ d.online ? '在线' : '离线' }}
            </div>
          </div>

          <!-- 实时读数 -->
          <div class="dc-readings">
            <div class="reading">
              <div class="r-label">用电量</div>
              <div class="r-val">{{ d.realtime_electricity?.toFixed(1) || '--' }} <span>kWh</span></div>
            </div>
            <div class="reading">
              <div class="r-label">天然气</div>
              <div class="r-val">{{ d.realtime_gas?.toFixed(1) || '--' }} <span>m³</span></div>
            </div>
            <div class="reading">
              <div class="r-label">用水</div>
              <div class="r-val">{{ d.realtime_water?.toFixed(1) || '--' }} <span>t</span></div>
            </div>
          </div>

          <!-- 迷你 spark：近 7 次读数 -->
          <div class="dc-spark">
            <div
              v-for="(h, i) in (d.spark || [])"
              :key="i"
              class="spark-bar"
              :style="{ height: h + '%' }"
            ></div>
          </div>

          <!-- 底部：心跳 + 操作 -->
          <div class="dc-foot">
            <div class="dc-heartbeat">
              <el-icon :size="12"><Clock /></el-icon>
              心跳 {{ formatAgo(d.last_heartbeat) }}
            </div>
            <div class="dc-actions">
              <el-button size="small" :icon="Cpu" type="success" link
                :disabled="!d.online"
                :loading="collectingId === d.device_id"
                @click="doCollect(d)">
                立即采集
              </el-button>
              <el-button size="small" :icon="TrendCharts" type="primary" link @click="viewRecords(d)">
                查看采集记录
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 数据录入区：两个 Tab ===== -->
    <div class="record-section">
      <el-tabs v-model="recordTab" class="record-tabs">
        <el-tab-pane label="IoT 自动采集记录" name="iot">
          <div class="tab-desc">
            <el-icon :size="14"><Cpu /></el-icon>
            以下为 IoT 设备自动上报的能耗数据，已自动标记「低风险」并上链
            <el-button size="small" type="primary" :icon="Cpu" class="collect-all-btn"
              :loading="collectingAll" @click="collectAll">
              一键全部采集
            </el-button>
          </div>
        </el-tab-pane>
        <el-tab-pane label="手动录入记录" name="manual">
          <div class="tab-desc manual">
            <el-icon :size="14"><Warning /></el-icon>
            手动录入的数据将被标记「中风险」，需额外关注
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- 手动录入表单 -->
      <div v-if="recordTab === 'manual'" class="manual-form-card">
        <div class="mf-head">
          <h4>手动录入能耗数据（模拟设备）</h4>
          <el-tag type="warning" effect="dark" size="small">
            <el-icon :size="11"><Warning /></el-icon> 风险标签：中
          </el-tag>
        </div>
        <el-form :model="manualForm" label-width="110px" class="mf-form">
          <el-row :gutter="16">
            <el-col :span="6">
              <el-form-item label="模拟设备" required>
                <el-select v-model="manualForm.device_id" style="width:100%" placeholder="选择设备">
                  <el-option v-for="d in devices" :key="d.device_id"
                    :value="d.device_id" :label="`${d.device_name} (${d.device_id})`" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="用电量(kWh)">
                <el-input-number v-model="manualForm.electricity" :min="0" :precision="1" style="width:100%" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="天然气(m³)">
                <el-input-number v-model="manualForm.gas" :min="0" :precision="1" style="width:100%" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="用水(t)">
                <el-input-number v-model="manualForm.water" :min="0" :precision="1" style="width:100%" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row>
            <el-col :span="6">
              <el-form-item label="采集时间">
                <el-date-picker
                  v-model="manualForm.collect_time"
                  type="datetime" value-format="YYYY-MM-DD HH:mm:ss"
                  style="width:100%" placeholder="选择时间"
                />
              </el-form-item>
            </el-col>
            <el-col :span="18">
              <el-form-item label="备注">
                <el-input v-model="manualForm.remark" placeholder="选填：本次录入的特殊情况" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item>
            <el-button type="primary" :icon="EditPen" :loading="loadingManual" @click="doManualRecord">
              提交并上链存证
            </el-button>
            <el-tag type="danger" effect="plain" style="margin-left:10px">
              <el-icon :size="11"><Warning /></el-icon>
              手动录入数据将被标记为「风险等级：中」
            </el-tag>
          </el-form-item>
        </el-form>
      </div>

      <!-- 采集记录表格 -->
      <div class="records-table-wrap">
        <el-table
          :data="filteredRecords"
          v-loading="loadingRecords"
          stripe
          style="width:100%"
          empty-text="暂无记录"
        >
          <el-table-column label="设备" width="180">
            <template #default="{ row }">
              <div class="rec-dev-name">{{ row.device_name || row.device_id }}</div>
              <code class="mono">{{ row.device_id }}</code>
            </template>
          </el-table-column>
          <el-table-column label="用电量(kWh)" width="130" align="right">
            <template #default="{ row }">{{ row.electricity?.toFixed(1) }}</template>
          </el-table-column>
          <el-table-column label="天然气(m³)" width="130" align="right">
            <template #default="{ row }">{{ row.gas?.toFixed(1) }}</template>
          </el-table-column>
          <el-table-column label="用水(t)" width="110" align="right">
            <template #default="{ row }">{{ row.water?.toFixed(1) }}</template>
          </el-table-column>
          <el-table-column label="采集方式" width="110" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.source === 'iot' || row.source === 'auto'" type="success" size="small" effect="dark">IoT 自动</el-tag>
              <el-tag v-else type="warning" size="small" effect="dark">手动录入</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="风险等级" width="110" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.source === 'iot' || row.source === 'auto'" class="risk-low" effect="plain">低风险</el-tag>
              <el-tag v-else class="risk-mid" effect="plain">
                <el-icon :size="11"><Warning /></el-icon> 中风险
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="上链状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.on_chain" type="success" size="small" effect="dark">已上链</el-tag>
              <el-tag v-else type="info" size="small">未上链</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="采集时间" width="180">
            <template #default="{ row }">{{ formatTime(row.collect_time) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- ===== 注册新设备弹窗 ===== -->
    <el-dialog v-model="registerVisible" width="480px" title="注册新 IoT 设备">
      <el-form :model="registerForm" label-width="100px">
        <el-form-item label="设备 ID" required>
          <el-input v-model="registerForm.device_id" placeholder="如: DEV-001" />
          <div class="form-tip">建议格式: DEV-数字序号，注册后不可修改</div>
        </el-form-item>
        <el-form-item label="设备名称" required>
          <el-input v-model="registerForm.device_name" placeholder="如: 一号车间电表" />
        </el-form-item>
        <el-form-item label="设备类型">
          <el-select v-model="registerForm.device_type" style="width:100%">
            <el-option label="电表 (meter)" value="meter" />
            <el-option label="气表 (meter-gas)" value="meter-gas" />
            <el-option label="水表 (meter-water)" value="meter-water" />
            <el-option label="网关 (gateway)" value="gateway" />
            <el-option label="通用传感器 (sensor)" value="sensor" />
          </el-select>
        </el-form-item>
        <el-form-item label="位置">
          <el-input v-model="registerForm.location" placeholder="如: 一号车间 / A区" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="registerVisible = false">取消</el-button>
        <el-button type="primary" :loading="loadingRegister" @click="doRegister">注册设备</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Cpu, EditPen, TrendCharts, Refresh, Loading, Connection, Clock, Warning
} from '@element-plus/icons-vue'
import { iotAPI } from '../api/index.js'

/* ===== 状态 ===== */
const devices = ref([])
const records = ref([])
const loadingDevices = ref(false)
const loadingRecords = ref(false)
const loadingAll = ref(false)
const loadingManual = ref(false)
const loadingRegister = ref(false)
const filterStatus = ref('all')
const recordTab = ref('iot')

const onlineCount = computed(() => devices.value.filter(d => d.online).length)

const filteredDevices = computed(() => {
  if (filterStatus.value === 'online') return devices.value.filter(d => d.online)
  if (filterStatus.value === 'offline') return devices.value.filter(d => !d.online)
  return devices.value
})

const filteredRecords = computed(() => {
  // 后端自动采集记录 source="auto"，手动录入 source="manual"
  if (recordTab.value === 'iot') return records.value.filter(r => r.source === 'iot' || r.source === 'auto')
  return records.value.filter(r => r.source !== 'iot' && r.source !== 'auto')
})

const deviceTypeMap = { meter: '电表', 'meter-gas': '气表', 'meter-water': '水表', gateway: '网关', sensor: '传感器' }

/* ===== 注册表单 ===== */
const registerVisible = ref(false)
const registerForm = reactive({
  device_id: 'DEV-' + Date.now().toString().slice(-6),
  device_name: '',
  device_type: 'meter',
  location: ''
})

/* ===== 手动录入表单 ===== */
const manualForm = reactive({
  device_id: '',
  electricity: 0, gas: 0, water: 0,
  collect_time: new Date().toISOString().slice(0, 19).replace('T', ' '),
  remark: ''
})

/* ===== 模拟设备数据 ===== */
function mockDevices() {
  const types = ['meter', 'meter', 'meter-gas', 'meter-water', 'gateway', 'sensor']
  const names = ['一号车间电表', '二号车间电表', '天然气主管道', '生产用水表', '厂区网关', '空气传感器', '三号车间电表', '备用网关']
  return Array.from({ length: 8 }, (_, i) => ({
    device_id: 'DEV-' + String(1001 + i),
    device_name: names[i] || 'IoT设备-' + (1001 + i),
    device_type: types[i % types.length],
    location: i < 3 ? '一号车间' : i < 5 ? '二号车间' : i < 7 ? '生产区' : '厂区门口',
    online: Math.random() > 0.2,
    last_heartbeat: new Date(Date.now() - Math.random() * 3600000).toISOString(),
    realtime_electricity: +(Math.random() * 50 + 10).toFixed(1),
    realtime_gas: +(Math.random() * 10 + 1).toFixed(1),
    realtime_water: +(Math.random() * 20 + 2).toFixed(1),
    spark: Array.from({ length: 7 }, () => Math.round(30 + Math.random() * 70)),
  }))
}

function mockRecords() {
  const list = []
  for (let i = 0; i < 15; i++) {
    const manual = Math.random() > 0.75
    list.push({
      id: i + 1,
      device_id: 'DEV-' + String(1001 + (i % 8)),
      device_name: 'IoT设备-' + (1001 + (i % 8)),
      electricity: +(Math.random() * 50 + 5).toFixed(1),
      gas: +(Math.random() * 10 + 1).toFixed(1),
      water: +(Math.random() * 15 + 2).toFixed(1),
      collect_time: new Date(Date.now() - i * 300000).toISOString(),
      source: manual ? 'manual' : 'iot',
      on_chain: Math.random() > 0.1,
      block_hash: manual ? null : '0x' + Array(16).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
    })
  }
  return list
}

/* ===== 加载 ===== */
async function loadDevices() {
  loadingDevices.value = true
  try {
    const res = await iotAPI.devices({})
    devices.value = res.data?.list || res.data || mockDevices()
    // 字段映射：后端返回 status(online/offline) / last_online；
    // 实时读数后端不返回，首帧用随机值展示，采集后以真实采集值更新
    devices.value = devices.value.map(d => ({
      ...d,
      online: d.online !== undefined ? d.online : (d.status ? d.status === 'online' : Math.random() > 0.2),
      last_heartbeat: d.last_heartbeat || d.last_online || new Date().toISOString(),
      realtime_electricity: d.realtime_electricity ?? +(Math.random() * 50 + 10).toFixed(1),
      realtime_gas: d.realtime_gas ?? +(Math.random() * 10 + 1).toFixed(1),
      realtime_water: d.realtime_water ?? +(Math.random() * 20 + 2).toFixed(1),
      spark: d.spark || Array.from({ length: 7 }, () => Math.round(30 + Math.random() * 70)),
    }))
  } catch {
    devices.value = mockDevices()
  } finally {
    loadingDevices.value = false
  }
}

async function loadRecords() {
  loadingRecords.value = true
  try {
    const res = await iotAPI.records({ page: 1, page_size: 50 })
    records.value = res.data?.list || res.data || mockRecords()
    // 补 source 字段
    records.value = records.value.map(r => ({
      ...r,
      source: r.source || (r.source_type === 'manual' ? 'manual' : 'iot'),
    }))
  } catch {
    records.value = mockRecords()
  } finally {
    loadingRecords.value = false
  }
}

async function refreshAll() {
  loadingAll.value = true
  await Promise.all([loadDevices(), loadRecords()])
  loadingAll.value = false
  ElMessage.success('数据已刷新')
}

/* ===== IoT 自动采集 ===== */
const collectingId = ref('')     // 正在采集的设备 ID（单台 loading）
const collectingAll = ref(false) // 一键全部采集 loading

// 基于设备当前读数生成模拟采集值（±10% 波动，模拟传感器连续采样）
function genReading(v) {
  const base = v || 10
  return +(base * (0.9 + Math.random() * 0.2)).toFixed(1)
}

/** 单台设备立即采集：POST /api/iot/record → source=auto → 上链存证 */
async function doCollect(d) {
  if (!d.online) {
    ElMessage.warning('设备离线，无法采集')
    return
  }
  collectingId.value = d.device_id
  try {
    const payload = {
      device_id: d.device_id,
      electricity: genReading(d.realtime_electricity),
      gas: genReading(d.realtime_gas),
      water: genReading(d.realtime_water),
    }
    const res = await iotAPI.createRecord(payload)
    const rec = res.data?.data || res.data
    // 更新设备卡片实时读数 + spark 滚动
    d.realtime_electricity = payload.electricity
    d.realtime_gas = payload.gas
    d.realtime_water = payload.water
    d.last_heartbeat = new Date().toISOString()
    d.spark = [...(d.spark || []).slice(1), Math.round(30 + Math.random() * 70)]
    ElMessage.success(`采集成功：${rec?.record_no || d.device_name} 已上链存证`)
    loadRecords()
  } catch (e) {
    ElMessage.error('自动采集失败：' + (e?.response?.data?.msg || e.message || '网络异常'))
  } finally {
    collectingId.value = ''
  }
}

/** 一键全部采集：遍历所有在线设备依次采集 */
async function collectAll() {
  const online = devices.value.filter(d => d.online)
  if (!online.length) {
    ElMessage.warning('没有在线设备可采集')
    return
  }
  collectingAll.value = true
  let ok = 0, fail = 0
  for (const d of online) {
    try {
      const payload = {
        device_id: d.device_id,
        electricity: genReading(d.realtime_electricity),
        gas: genReading(d.realtime_gas),
        water: genReading(d.realtime_water),
      }
      await iotAPI.createRecord(payload)
      d.realtime_electricity = payload.electricity
      d.realtime_gas = payload.gas
      d.realtime_water = payload.water
      d.last_heartbeat = new Date().toISOString()
      d.spark = [...(d.spark || []).slice(1), Math.round(30 + Math.random() * 70)]
      ok++
    } catch {
      fail++
    }
  }
  collectingAll.value = false
  if (fail === 0) ElMessage.success(`全部采集完成：${ok} 台设备读数已上链存证`)
  else ElMessage.warning(`采集完成：成功 ${ok} 台，失败 ${fail} 台`)
  loadRecords()
}

/* ===== 注册设备 ===== */
function openRegister() {
  registerForm.device_id = 'DEV-' + Date.now().toString().slice(-6)
  registerForm.device_name = ''
  registerForm.device_type = 'meter'
  registerForm.location = ''
  registerVisible.value = true
}

async function doRegister() {
  if (!registerForm.device_id || !registerForm.device_name) {
    ElMessage.warning('请填写设备 ID 和名称')
    return
  }
  loadingRegister.value = true
  try {
    await iotAPI.registerDevice(registerForm)
    ElMessage.success('设备注册成功！')
    registerVisible.value = false
    loadDevices()
  } catch (e) {
    ElMessage.error('设备注册失败：' + (e?.response?.data?.msg || e.message || '网络异常'))
  } finally {
    loadingRegister.value = false
  }
}

/* ===== 手动录入 ===== */
async function doManualRecord() {
  if (!manualForm.device_id) {
    ElMessage.warning('请先选择模拟设备')
    return
  }
  loadingManual.value = true
  try {
    await iotAPI.createManualRecord(manualForm)
    ElMessage.success('手动录入成功，已标记风险并上链存证')
    loadRecords()
  } catch (e) {
    ElMessage.error('手动录入失败：' + (e?.response?.data?.msg || e.message || '网络异常'))
  } finally {
    loadingManual.value = false
  }
}

/* ===== 工具 ===== */
function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`
}
function formatAgo(t) {
  if (!t) return '未知'
  const diff = (Date.now() - new Date(t).getTime()) / 1000
  if (diff < 60) return Math.floor(diff) + '秒前'
  if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
  if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
  return Math.floor(diff / 86400) + '天前'
}

function viewRecords(d) {
  recordTab.value = d.online ? 'iot' : 'manual'
  ElMessage.info(`查看 ${d.device_name} 的采集记录（滚动到下方表格）`)
}

/* ===== 定时刷新在线设备状态 ===== */
let heartbeatTimer = null
onMounted(() => {
  refreshAll()
  heartbeatTimer = setInterval(() => {
    // 在线设备心跳时间推进
    devices.value = devices.value.map(d => d.online ? ({ ...d, last_heartbeat: new Date().toISOString() }) : d)
  }, 30000)
})
onUnmounted(() => clearInterval(heartbeatTimer))
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 页面头 ===== */
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 22px;
  margin-bottom: 16px;
  background: linear-gradient(135deg, var(--primary-green), #16a34acc);
  border-radius: var(--radius-lg);
  color: #fff;
}
.page-head h2 { font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.ph-desc { font-size: 12px; margin: 0; color: rgba(255,255,255,0.85); }
.ph-stats { display: flex; align-items: center; gap: 14px; }
.ps-item {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 12px;
  background: rgba(255,255,255,0.15);
  border-radius: 8px;
  font-size: 12px;
}
.ps-item b { font-weight: 700; margin: 0 2px; }
.ps-item.online { background: rgba(34,211,238,0.25); }
.ps-item.offline { background: rgba(148,163,184,0.2); }
.reg-btn { background: #fff !important; color: var(--primary-green) !important; border-color: #fff !important; }

/* ===== 设备区 ===== */
.dev-section {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.dev-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.dev-head h3 { font-size: 14px; font-weight: 600; margin: 0; }
.dev-filters { display: flex; gap: 10px; align-items: center; }

.dev-loading, .dev-empty {
  padding: 60px 20px;
  text-align: center;
  color: var(--text-tertiary);
}
.dev-empty p { margin-top: 10px; font-size: 12.5px; }

.dev-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
}

/* ===== 设备卡片 ===== */
.dev-card {
  position: relative;
  padding: 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--line-color);
  border-radius: 12px;
  transition: all 0.2s;
}
.dev-card:hover {
  border-color: var(--primary-green);
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(0,0,0,0.08);
}
.dev-card.offline {
  border-color: #e2e8f0;
  opacity: 0.7;
}
.dev-card.offline:hover { opacity: 0.85; }

.dc-head { display: flex; align-items: flex-start; gap: 12px; margin-bottom: 14px; }
.dc-icon {
  position: relative;
  width: 44px; height: 44px;
  border-radius: 10px;
  background: rgba(16,185,129,0.08);
  color: var(--primary-green);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.dc-icon .el-icon.offline { color: #94a3b8; }
.pulse {
  position: absolute; top: -2px; right: -2px;
  width: 10px; height: 10px; border-radius: 50%;
  background: #94a3b8;
}
.pulse.pulse-on {
  background: #10b981;
  animation: dotPulse 2s infinite;
}
@keyframes dotPulse {
  0%,100% { box-shadow: 0 0 0 0 rgba(16,185,129,0.6); }
  50% { box-shadow: 0 0 0 6px rgba(16,185,129,0); }
}

.dc-info { flex: 1; min-width: 0; }
.dc-name { font-size: 14px; font-weight: 600; color: var(--text-primary); }
.dc-meta { display: flex; align-items: center; gap: 8px; margin-top: 4px; font-size: 11px; color: var(--text-tertiary); }
.dc-meta code { font-family: ui-monospace, Consolas, monospace; }
.dtag { font-size: 10.5px !important; }
.dtag.meter, .dtag.meter-gas, .dtag.meter-water, .dtag.gateway, .dtag.sensor {
  background: rgba(16,185,129,0.08); color: #10b981;
}

.dc-status {
  font-size: 11px;
  padding: 2px 8px; border-radius: 999px;
  display: flex; align-items: center; gap: 4px;
}
.dc-status.online { background: rgba(16,185,129,0.1); color: #10b981; }
.dc-status.offline { background: rgba(148,163,184,0.1); color: #94a3b8; }
.dc-status .pulse { position: static; width: 6px; height: 6px; }

/* 读数 */
.dc-readings {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  padding: 10px 8px;
  background: var(--card-bg);
  border-radius: 8px;
  margin-bottom: 10px;
}
.reading { text-align: center; }
.r-label { font-size: 10.5px; color: var(--text-tertiary); margin-bottom: 2px; }
.r-val { font-size: 15px; font-weight: 700; color: var(--text-primary); font-variant-numeric: tabular-nums; }
.r-val span { font-size: 10.5px; font-weight: 400; color: var(--text-tertiary); margin-left: 2px; }

/* spark */
.dc-spark {
  display: flex; gap: 3px; align-items: flex-end;
  height: 26px;
  margin-bottom: 10px;
}
.spark-bar {
  flex: 1;
  min-height: 2px;
  background: linear-gradient(180deg, #22c55e, #16a34a);
  border-radius: 2px 2px 0 0;
  opacity: 0.65;
  transition: height 0.3s;
}

.dc-foot {
  display: flex; justify-content: space-between; align-items: center;
  padding-top: 10px;
  border-top: 1px dashed var(--line-color);
}
.dc-heartbeat { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--text-tertiary); }
.dc-actions { display: flex; align-items: center; gap: 2px; }

/* ===== 记录区 ===== */
.record-section {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.record-tabs :deep(.el-tabs__item.is-active) { color: var(--primary-green); }
.record-tabs :deep(.el-tabs__active-bar) { background-color: var(--primary-green); }

.tab-desc {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 14px;
  background: rgba(16,185,129,0.05);
  border-radius: 6px;
  font-size: 12px; color: var(--text-secondary);
}
.tab-desc.manual { background: rgba(245,158,11,0.05); color: #d97706; }
.collect-all-btn { margin-left: auto; }

/* ===== 手动录入表单 ===== */
.manual-form-card {
  margin: 16px 0;
  padding: 16px;
  background: var(--bg-secondary);
  border: 1px dashed var(--line-strong);
  border-radius: 10px;
}
.mf-head {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 14px;
}
.mf-head h4 { font-size: 13px; font-weight: 600; margin: 0; }
.form-tip { font-size: 11px; color: var(--text-tertiary); margin-top: 3px; }

/* ===== 表格 ===== */
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }
.rec-dev-name { font-size: 12.5px; font-weight: 500; color: var(--text-primary); }

.risk-low { background: rgba(16,185,129,0.08); color: #10b981; }
.risk-mid { background: rgba(245,158,11,0.08); color: #d97706; }

@media (max-width: 900px) {
  .page-head { flex-direction: column; gap: 10px; align-items: flex-start; }
  .ph-stats { flex-wrap: wrap; }
}
</style>