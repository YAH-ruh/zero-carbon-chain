/**
 * AI 智能分析助手 —— Pinia Store
 *
 * 负责：
 *  1) 管理一次分析会话的状态（loading / error / advice 内容）
 *  2) 持久化历史建议记录到 localStorage（导出报告、回看）
 *  3) 暴露"保存建议 / 导出 Markdown 报告"方法
 *
 * 为什么用 Pinia：
 *  - 侧边栏、AiAnalysisAssistant.vue 主组件、可能悬浮的 AiAssistant.vue 都要读写同一份状态
 *  - localStorage 持久化建议记录，刷新不丢
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { generateAdvice } from '../utils/aiApi.js'

const STORAGE_KEY = 'ai_assistant_records'
const MAX_RECORDS = 50 // 本地最多保留 50 条，防止爆 localStorage

export const useAiAssistantStore = defineStore('aiAssistant', () => {
  // ---- 悬浮组件展开触发器 ----
  // 每次 +1，AiAssistant.vue watch 到变化就展开
  const assistantTrigger = ref(0)
  function openAssistant() { assistantTrigger.value++ }

  // ---- 单次会话状态 ----
  const loading = ref(false)
  const errorMessage = ref('')
  const currentAdvice = ref('')           // 当前 Markdown 建议文本
  const currentMeta = ref({               // 当前建议元信息
    role: '',
    page: '',
    background: '',
    timestamp: '',
    mock: false
  })

  // ---- 历史建议记录 ----
  const records = ref(loadRecords())

  // ---- 派生 ----
  const hasCurrent = computed(() => !!currentAdvice.value)
  const recordCount = computed(() => records.value.length)

  // ============ 核心动作 ============

  /** 调用 AI 生成建议 */
  async function generate(payload) {
    loading.value = true
    errorMessage.value = ''
    currentAdvice.value = ''

    try {
      const { content, raw, mock } = await generateAdvice(payload)
      currentAdvice.value = content
      currentMeta.value = {
        role: payload.role || '',
        page: payload.page || '',
        background: payload.background || '',
        timestamp: new Date().toISOString(),
        mock: !!mock
      }

      // 自动存一条历史（除非是同一秒重复）
      saveRecord({
        id: Date.now().toString(),
        ...currentMeta.value,
        content,
        raw: raw || null
      })
    } catch (e) {
      errorMessage.value = e?.message || 'AI 调用失败，请检查网络或 API Key 配置'
    } finally {
      loading.value = false
    }
  }

  /** 手动保存当前建议到历史 */
  function saveCurrent() {
    if (!currentAdvice.value) return
    saveRecord({
      id: Date.now().toString(),
      ...currentMeta.value,
      content: currentAdvice.value,
      raw: null
    })
  }

  /** 清空当前会话 */
  function clearCurrent() {
    currentAdvice.value = ''
    currentMeta.value = { role: '', page: '', background: '', timestamp: '', mock: false }
    errorMessage.value = ''
  }

  /** 导出当前建议为 Markdown 文件（浏览器下载） */
  function exportCurrentReport() {
    if (!currentAdvice.value) return
    const md = buildReport(currentMeta.value, currentAdvice.value)
    downloadBlob(md, `AI分析报告_${currentMeta.value.page || '通用'}_${formatFileDate()}.md`, 'text/markdown')
  }

  /** 导出历史中某一条 */
  function exportRecordReport(id) {
    const r = records.value.find(x => x.id === id)
    if (!r) return
    const md = buildReport(r, r.content)
    downloadBlob(md, `AI分析报告_${r.page || '通用'}_${formatFileDate(r.timestamp)}.md`, 'text/markdown')
  }

  /** 删除一条历史 */
  function removeRecord(id) {
    records.value = records.value.filter(r => r.id !== id)
    persistRecords()
  }

  /** 清空全部历史 */
  function clearRecords() {
    records.value = []
    persistRecords()
  }

  // ============ 内部辅助 ============

  function saveRecord(record) {
    // 去重：如果最后一条和新记录的 content 完全一样，跳过
    const last = records.value[0]
    if (last && last.content === record.content && last.page === record.page) return

    records.value.unshift(record)
    if (records.value.length > MAX_RECORDS) {
      records.value = records.value.slice(0, MAX_RECORDS)
    }
    persistRecords()
  }

  function persistRecords() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(records.value))
    } catch (e) {
      // localStorage 可能被禁用，忽略
    }
  }

  function loadRecords() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      return raw ? JSON.parse(raw) : []
    } catch {
      return []
    }
  }

  function buildReport(meta, content) {
    return [
      `# 零碳微证 AI 智能分析报告`,
      '',
      `| 字段 | 值 |`,
      `|------|----|`,
      `| 角色 | ${meta.role || '-'} |`,
      `| 页面/场景 | ${meta.page || '-'} |`,
      `| 项目背景 | ${meta.background || '-'} |`,
      `| 生成时间 | ${meta.timestamp ? new Date(meta.timestamp).toLocaleString('zh-CN') : '-'} |`,
      `| 数据来源 | ${meta.mock ? '本地兜底建议（未配置 API Key）' : 'DeepSeek 在线分析'} |`,
      '',
      '---',
      '',
      content,
      '',
      `\n> 本报告由「零碳微证 AI 智能分析助手」自动生成，仅供参考。`
    ].join('\n')
  }

  function downloadBlob(text, filename, type = 'text/plain') {
    const blob = new Blob([text], { type: `${type};charset=utf-8` })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  function formatFileDate(isoStr) {
    const d = isoStr ? new Date(isoStr) : new Date()
    const pad = n => String(n).padStart(2, '0')
    return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}_${pad(d.getHours())}${pad(d.getMinutes())}`
  }

  return {
    // state
    assistantTrigger,
    loading,
    errorMessage,
    currentAdvice,
    currentMeta,
    records,
    // getters
    hasCurrent,
    recordCount,
    // actions
    openAssistant,
    generate,
    saveCurrent,
    clearCurrent,
    exportCurrentReport,
    exportRecordReport,
    removeRecord,
    clearRecords
  }
})
