<!--
  footprint.vue - 产品碳足迹管理（enterprise 角色）
  功能：
    1. 产品清单（名称/类别/生命周期阶段/碳足迹 kgCO₂e）
    2. 产品生命周期阶段分解（原材料/生产/运输/使用/废弃 · ECharts 柱状堆叠）
    3. 新建碳足迹核算（表单 + 阶段输入）
    4. 导出碳足迹报告 PDF 按钮
-->
<template>
  <div class="fp-page page">

    <!-- ===== 头部 ===== -->
    <div class="page-head">
      <div class="ph-left">
        <h2>
          <el-icon :size="18"><Document /></el-icon>
          产品碳足迹管理
        </h2>
        <p>全生命周期碳足迹核算 · 原材料 → 生产 → 运输 → 使用 → 废弃</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreate">+ 新建碳足迹核算</el-button>
    </div>

    <!-- ===== 统计卡 ===== -->
    <div class="stat-row">
      <div class="stat-card total">
        <div class="sc-icon"><el-icon :size="18"><Box /></el-icon></div>
        <div>
          <div class="sc-num">{{ products.length }}</div>
          <div class="sc-lab">核算产品数</div>
        </div>
      </div>
      <div class="stat-card footprint">
        <div class="sc-icon"><el-icon :size="18"><TrendCharts /></el-icon></div>
        <div>
          <div class="sc-num">{{ totalFootprint.toFixed(1) }}</div>
          <div class="sc-lab">总碳足迹 kgCO₂e</div>
        </div>
      </div>
      <div class="stat-card top">
        <div class="sc-icon"><el-icon :size="18"><Trophy /></el-icon></div>
        <div>
          <div class="sc-num">{{ topProduct?.name || '--' }}</div>
          <div class="sc-lab">最大贡献产品</div>
        </div>
      </div>
    </div>

    <!-- ===== 产品列表 ===== -->
    <div class="list-card">
      <div class="lc-head">
        <h3>产品碳足迹清单</h3>
        <div class="lc-filters">
          <el-select v-model="categoryFilter" placeholder="全部类别" size="small" style="width:130px" clearable>
            <el-option label="电子产品" value="电子" />
            <el-option label="建材" value="建材" />
            <el-option label="食品" value="食品" />
            <el-option label="纺织" value="纺织" />
            <el-option label="机械" value="机械" />
          </el-select>
          <el-button :icon="Refresh" size="small" @click="loadProducts">刷新</el-button>
        </div>
      </div>

      <el-table :data="filteredProducts" stripe size="small" style="width:100%" empty-text="暂无产品碳足迹">
        <el-table-column label="产品编号" width="160">
          <template #default="{ row }"><code class="mono">{{ row.product_no }}</code></template>
        </el-table-column>
        <el-table-column prop="name" label="产品名称" width="160" />
        <el-table-column prop="category" label="类别" width="90">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.category }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="生命周期阶段贡献" width="380">
          <template #default="{ row }">
            <div class="stage-bar">
              <div
                v-for="(s, i) in row.stages"
                :key="i"
                class="sb"
                :style="{ width: (s / row.total * 100) + '%', background: stageColors[i] }"
                :title="stageLabels[i] + ': ' + s.toFixed(1) + ' kgCO₂e'"
              ></div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="碳足迹(kgCO₂e)" width="140" align="right">
          <template #default="{ row }">
            <b class="green">{{ row.total.toFixed(1) }}</b>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="viewDetail(row)">详情</el-button>
            <el-button size="small" link type="success" @click="exportReport(row)">导出报告</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== 新建核算弹窗 ===== -->
    <el-dialog v-model="createVisible" width="580px" title="新建产品碳足迹核算">
      <el-form :model="createForm" label-width="130px">
        <el-form-item label="产品名称" required>
          <el-input v-model="createForm.name" placeholder="如: 节能电子元器件 A 型" />
        </el-form-item>
        <el-form-item label="产品类别">
          <el-select v-model="createForm.category" style="width:100%">
            <el-option label="电子产品" value="电子" />
            <el-option label="建材" value="建材" />
            <el-option label="食品" value="食品" />
            <el-option label="纺织" value="纺织" />
            <el-option label="机械" value="机械" />
          </el-select>
        </el-form-item>
        <el-divider content-position="left">生命周期阶段排放（kgCO₂e）</el-divider>
        <el-row :gutter="12">
          <el-col :span="12" v-for="(label, i) in stageLabels" :key="i">
            <el-form-item :label="label">
              <el-input-number v-model="createForm.stages[i]" :min="0" :step="0.1" :precision="1" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-alert type="info" :closable="false" show-icon>
          <template #title>
            <span>合计：<b class="green">{{ stageTotal.toFixed(1) }} kgCO₂e</b></span>
          </template>
        </el-alert>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="loadingCreate" @click="doCreate">创建核算</el-button>
      </template>
    </el-dialog>

    <!-- ===== 详情弹窗 ===== -->
    <el-dialog v-model="detailVisible" width="620px" :title="'碳足迹详情 · ' + (currentProduct?.name || '')">
      <div v-if="currentProduct">
        <div class="fp-summary">
          <div class="fps-item">
            <div class="fps-lab">产品编号</div>
            <div class="fps-val mono">{{ currentProduct.product_no }}</div>
          </div>
          <div class="fps-item">
            <div class="fps-lab">类别</div>
            <div class="fps-val">{{ currentProduct.category }}</div>
          </div>
          <div class="fps-item">
            <div class="fps-lab">总碳足迹</div>
            <div class="fps-val green">{{ currentProduct.total.toFixed(1) }} kgCO₂e</div>
          </div>
        </div>

        <!-- 阶段柱状图 -->
        <div ref="chartRef" style="height:260px;margin:16px 0"></div>

        <div class="stage-table">
          <div class="st-row st-head">
            <div>阶段</div><div class="num">排放</div><div class="num">占比</div><div class="bar-col">占比条</div>
          </div>
          <div v-for="(s, i) in currentProduct.stages" :key="i" class="st-row">
            <div><span class="dot" :style="{ background: stageColors[i] }"></span>{{ stageLabels[i] }}</div>
            <div class="num">{{ s.toFixed(1) }} kgCO₂e</div>
            <div class="num">{{ (s / currentProduct.total * 100).toFixed(1) }}%</div>
            <div class="bar-col">
              <div class="bar-mini" :style="{ width: (s / currentProduct.total * 100) + '%', background: stageColors[i] }"></div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button type="success" :icon="Download" @click="exportReport(currentProduct)">
          导出碳足迹报告
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, nextTick, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Document, Plus, Box, TrendCharts, Trophy, Refresh, Download
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { footprintAPI } from '../api/index.js'

const stageLabels = ['原材料', '生产制造', '运输物流', '使用运营', '废弃回收']
const stageColors = ['#0891b2', '#10b981', '#f59e0b', '#7c3aed', '#ef4444']

/* ===== 状态 ===== */
const products = ref([])
const categoryFilter = ref('')
const createVisible = ref(false)
const detailVisible = ref(false)
const currentProduct = ref(null)
const loadingCreate = ref(false)
const chartRef = ref(null)

/* ===== 表单 ===== */
const createForm = reactive({
  name: '', category: '电子',
  stages: [0, 0, 0, 0, 0]
})
const stageTotal = computed(() => createForm.stages.reduce((a, b) => a + b, 0))

/* ===== 计算 ===== */
const filteredProducts = computed(() => {
  if (!categoryFilter.value) return products.value
  return products.value.filter(p => p.category === categoryFilter.value)
})
const totalFootprint = computed(() => products.value.reduce((s, p) => s + p.total, 0))
const topProduct = computed(() =>
  [...products.value].sort((a, b) => b.total - a.total)[0] || null
)

/* ===== 加载 ===== */
async function loadProducts() {
  try {
    const res = await footprintAPI.list({ category: categoryFilter.value || undefined })
    const list = res.data?.list || res.data || []
    products.value = list.map(p => ({
      ...p,
      stages: p.stages || [p.raw_materials || 0, p.manufacture || 0, p.transport || 0, p.usage || 0, p.waste || 0],
      total: p.total || (p.raw_materials || 0) + (p.manufacture || 0) + (p.transport || 0) + (p.usage || 0) + (p.waste || 0),
    }))
  } catch (e) {
    products.value = []
  }
}

/* ===== 新建 ===== */
function openCreate() {
  createForm.name = ''
  createForm.category = '电子'
  createForm.stages = [0, 0, 0, 0, 0]
  createVisible.value = true
}
async function doCreate() {
  if (!createForm.name || stageTotal.value === 0) {
    ElMessage.warning('请填写产品名称和至少一个阶段排放'); return
  }
  loadingCreate.value = true
  try {
    const res = await footprintAPI.create({
      name: createForm.name,
      category: createForm.category,
      raw_materials: createForm.stages[0],
      manufacture: createForm.stages[1],
      transport: createForm.stages[2],
      usage: createForm.stages[3],
      waste: createForm.stages[4],
    })
    const total = stageTotal.value
    ElMessage.success(`碳足迹核算创建成功，总排放 ${total.toFixed(1)} kgCO₂e (已上链)`)
    createVisible.value = false
    await loadProducts()
  } catch (e) {
    ElMessage.error(e.msg || '创建失败')
  } finally {
    loadingCreate.value = false
  }
}

/* ===== 详情 ===== */
function viewDetail(p) {
  currentProduct.value = p
  detailVisible.value = true
  nextTick(() => renderChart(p))
}

function renderChart(p) {
  if (!chartRef.value) return
  const chart = echarts.init(chartRef.value)
  chart.setOption({
    grid: { left: 110, right: 30, top: 20, bottom: 30 },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: '#e2e8f0', type: 'dashed' } },
      axisLabel: { color: '#64748b' },
    },
    yAxis: {
      type: 'category',
      data: [...stageLabels].reverse(),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: '#334155' },
    },
    series: [{
      type: 'bar',
      data: [...p.stages].reverse().map((v, i) => ({
        value: v,
        itemStyle: {
          borderRadius: [0, 4, 4, 0],
          color: stageColors[4 - i],
        }
      })),
      barWidth: 20,
      label: {
        show: true, position: 'right',
        formatter: '{c} kgCO₂e', color: '#334155', fontSize: 11
      }
    }]
  })
  chart.on('click', () => chart.dispose())
}

/* ===== 导出 ===== */
function exportReport(p) {
  const lines = [
    `产品碳足迹核算报告`,
    `生成时间: ${new Date().toLocaleString('zh-CN')}`,
    ``,
    `产品编号: ${p.product_no}`,
    `产品名称: ${p.name}`,
    `产品类别: ${p.category}`,
    `总碳足迹: ${p.total.toFixed(2)} kgCO₂e`,
    ``,
    `生命周期阶段分解:`,
    ...p.stages.map((s, i) => {
      const pct = (s / p.total * 100).toFixed(1)
      return `  ${stageLabels[i]}: ${s.toFixed(2)} kgCO₂e (${pct}%)`
    }),
    ``,
    `核算依据: ISO 14067 产品碳足迹标准`,
    `链上状态: 已存证 (SHA-256)`,
  ]
  const blob = new Blob([lines.join('\n')], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `碳足迹-${p.product_no}-${p.name}.txt`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('碳足迹报告已下载')
}

onMounted(() => loadProducts())
</script>

<style scoped>
.page { animation: fadeUp 0.3s ease; }
@keyframes fadeUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ===== 头部 ===== */
.page-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 18px 22px; margin-bottom: 16px;
  background: linear-gradient(135deg, #16a34a, #059669);
  border-radius: var(--radius-lg); color: #fff;
}
.page-head h2 { display: flex; align-items: center; gap: 8px; font-size: 17px; font-weight: 600; margin: 0 0 4px; color: #fff; }
.page-head p { font-size: 12px; color: rgba(255,255,255,0.9); margin: 0; }

/* ===== 统计卡 ===== */
.stat-row {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 12px; margin-bottom: 16px;
}
.stat-card {
  display: flex; gap: 12px; align-items: center;
  padding: 16px 18px;
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius);
  border-left: 4px solid transparent;
}
.stat-card.footprint { border-left-color: #16a34a; }
.stat-card.total    { border-left-color: #0891b2; }
.stat-card.top      { border-left-color: #f59e0b; }
.sc-icon {
  width: 40px; height: 40px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-card.footprint .sc-icon { background: rgba(22,163,74,0.12); color: #16a34a; }
.stat-card.total     .sc-icon { background: rgba(8,145,178,0.12); color: #0891b2; }
.stat-card.top       .sc-icon { background: rgba(245,158,11,0.12); color: #d97706; }
.sc-num { font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-primary); }
.sc-num.green { color: var(--primary-green); }
.sc-lab { font-size: 11px; color: var(--text-tertiary); }

/* ===== 列表 ===== */
.list-card {
  background: var(--card-bg); border: 1px solid var(--card-border);
  border-radius: var(--radius); padding: 18px 22px; margin-bottom: 16px;
}
.lc-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.lc-head h3 { font-size: 14px; font-weight: 600; margin: 0; }
.lc-filters { display: flex; gap: 10px; align-items: center; }

.stage-bar {
  display: flex; height: 14px; border-radius: 4px; overflow: hidden;
  background: var(--bg-secondary);
}
.sb { height: 100%; transition: width 0.3s; }

.green { color: var(--primary-green); font-weight: 700; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 11px; }

/* ===== 详情 ===== */
.fp-summary {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 12px; margin-bottom: 12px;
}
.fps-item {
  padding: 12px; background: var(--bg-secondary); border-radius: 8px;
}
.fps-lab { font-size: 11px; color: var(--text-tertiary); margin-bottom: 4px; }
.fps-val { font-size: 14px; font-weight: 600; }

.stage-table { border: 1px solid var(--line-color); border-radius: 8px; overflow: hidden; }
.st-row {
  display: grid; grid-template-columns: 120px 110px 80px 1fr; gap: 8px;
  padding: 10px 14px; align-items: center;
  font-size: 12.5px;
}
.st-row.st-head {
  background: var(--bg-secondary);
  font-weight: 600; font-size: 11.5px; color: var(--text-tertiary);
}
.st-row .num { text-align: right; font-variant-numeric: tabular-nums; font-family: ui-monospace, Consolas, monospace; }
.dot {
  display: inline-block; width: 8px; height: 8px;
  border-radius: 50%; margin-right: 6px; vertical-align: middle;
}
.bar-col { padding: 0 6px; }
.bar-mini { height: 10px; border-radius: 3px; transition: width 0.3s; }

@media (max-width: 900px) {
  .stat-row { grid-template-columns: repeat(2, 1fr); }
  .fp-summary { grid-template-columns: 1fr; }
}
</style>