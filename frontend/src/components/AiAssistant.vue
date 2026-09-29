<!--
  AiAssistant.vue
  右下角悬浮 AI 助手弹窗 —— 真实多轮对话版
  接入 Go 后端 POST /api/chat（所有角色 JWT 认证可用）

  功能：
  - 真实多轮对话（前端维护 messages 历史，每次把完整 history 传给后端）
  - system 角色由后端 services/ai.go 固定注入，前端无需传
  - loading 状态（发送消息时显示 AI 思考中...）
  - 错误捕获（网络失败、后端降级、接口异常都有 toast + 消息气泡提示）
  - 清空会话按钮
  - 消息列表保持原有样式（用户气泡靠右、AI 气泡靠左）
-->
<template>
  <div class="ai-fab" :class="{ expanded: expanded }">
    <!-- 收起态：悬浮按钮（带未读红点：降级/错误提示） -->
    <button v-if="!expanded" class="fab-btn" @click="expanded = true">
      <el-icon :size="20"><MagicStick /></el-icon>
      <span class="fab-label">AI 助手</span>
    </button>

    <!-- 展开态：对话面板 -->
    <div v-else class="ai-panel">
      <div class="ai-panel-header">
        <div class="ai-panel-title">
          <el-icon :size="18"><MagicStick /></el-icon>
          <span>AI 碳减排助手</span>
        </div>
        <div class="ai-panel-actions">
          <button class="ai-clear" title="清空会话" @click="clearAll">
            <el-icon :size="14"><RefreshLeft /></el-icon>
          </button>
          <button class="ai-close" @click="expanded = false">
            <el-icon :size="16"><Close /></el-icon>
          </button>
        </div>
      </div>

      <!-- 消息列表 -->
      <div ref="bodyRef" class="ai-panel-body">

        <!-- 初始欢迎气泡 -->
        <div v-if="messages.length === 0" class="ai-bubble ai-bubble-ai">
          <div class="bubble-avatar"><el-icon :size="16"><MagicStick /></el-icon></div>
          <div class="bubble-content">
            你好！我是零碳微证 AI 碳减排助手 🌿<br/>
            可以帮你：小微企业碳排放策略 · ZKP 隐私核算 · 碳积分管理 · 区块链存证方案
          </div>
        </div>

        <!-- 消息列表渲染 -->
        <div
          v-for="(m, i) in messages"
          :key="i"
          class="ai-bubble"
          :class="m.role === 'user' ? 'ai-bubble-user' : 'ai-bubble-ai'"
        >
          <div v-if="m.role !== 'user'" class="bubble-avatar">
            <el-icon :size="14"><MagicStick /></el-icon>
          </div>
          <div class="bubble-content" :class="{ error: m.error }">
            <!-- AI 回答渲染为 Markdown 简易格式（保留换行 + 加粗 + 列表） -->
            <template v-if="m.role !== 'user'">
              <!-- 独立思考过程（可折叠，默认收起） -->
              <div v-if="m.reasoning" class="think-block" :class="{ open: !!openThink[i] }" @click="toggleThink(i)">
                <div class="think-head">
                  <span class="think-title">💭 独立思考过程</span>
                  <span class="think-toggle">{{ openThink[i] ? '收起 ▲' : '展开 ▼' }}</span>
                </div>
                <div v-show="openThink[i]" class="think-body">{{ m.reasoning }}</div>
              </div>
              <div v-html="renderMd(m.content)" class="md-bubble"></div>
            </template>
            <!-- 用户消息纯文本 -->
            <template v-else>
              <span>{{ m.content }}</span>
            </template>
          </div>
          <div v-if="m.role === 'user'" class="bubble-avatar user">
            <el-icon :size="14"><User /></el-icon>
          </div>
        </div>

        <!-- Loading 气泡：AI 独立思考中（阶段推进动效） -->
        <div v-if="loading" class="ai-bubble ai-bubble-ai">
          <div class="bubble-avatar thinking-avatar">
            <el-icon :size="16" class="thinking-spin"><MagicStick /></el-icon>
          </div>
          <div class="bubble-content thinking-content">
            <div class="thinking-title">
              <div class="typing-dots"><span></span><span></span><span></span></div>
              <span class="thinking-label">AI 独立思考中 · {{ thinkingPhase }}</span>
            </div>
            <div class="thinking-steps">
              <span
                v-for="(p, idx) in thinkingPhases"
                :key="p"
                class="step-chip"
                :class="{ active: idx <= phaseIndex, current: idx === phaseIndex }"
              >{{ p }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 输入框 -->
      <div class="ai-panel-footer">
        <input
          v-model="input"
          class="ai-input"
          :placeholder="loading ? 'AI 正在回答，请稍候...' : '输入你的问题...'"
          :disabled="loading"
          @keyup.enter="send"
        />
        <button class="ai-send" :disabled="loading || !input.trim()" @click="send">
          <template v-if="loading">
            <el-icon class="is-loading" :size="14"><Loading /></el-icon>
          </template>
          <template v-else>
            <el-icon :size="14"><Promotion /></el-icon>
          </template>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * AiAssistant.vue —— 大屏 AI 碳减排助手（纯真实 API 模式）
 * 接入 Go 后端 POST /api/chat（所有角色 JWT 认证可用）
 * 只走真实 DeepSeek API，无本地 Mock 降级
 */
import { ref, computed, reactive, nextTick, onMounted, watch } from 'vue'
import axios from 'axios'
import { MagicStick, Close, Promotion, User, RefreshLeft, Loading } from '@element-plus/icons-vue'
import { useAiAssistantStore } from '../stores/aiAssistant.js'

const aiStore = useAiAssistantStore()

// ========== 状态 ==========
const expanded = ref(false)
const input = ref('')
const loading = ref(false)
const messages = ref([])     // [{ role:'user'|'assistant', content:string, reasoning?:string, error?:bool }]
const bodyRef = ref(null)

// ========== 独立思考状态（阶段推进动效） ==========
const thinkingPhases = ['理解问题', '关联碳减排场景', '推理可行方案', '自查结论']
const phaseIndex = ref(0)
const thinkingPhase = computed(() => thinkingPhases[phaseIndex.value] || '思考中')
let phaseTimer = null
const openThink = reactive({})   // 控制每条消息"思考过程"折叠态
function toggleThink(i) { openThink[i] = !openThink[i] }

function startThinking() {
  phaseIndex.value = 0
  stopThinking()
  phaseTimer = setInterval(() => {
    phaseIndex.value = (phaseIndex.value + 1) % thinkingPhases.length
  }, 1800)
}
function stopThinking() {
  if (phaseTimer) { clearInterval(phaseTimer); phaseTimer = null }
}

// 监听侧边栏/外部触发：assistantTrigger +1 → 展开对话面板
watch(() => aiStore.assistantTrigger, (n) => {
  if (n > 0) expanded.value = true
})

// ========== axios 实例（带 JWT） ==========
const chatClient = axios.create({
  baseURL: '/api',
  timeout: 90000,  // 独立思考(思维链)耗时更长，放宽到 90s
  headers: { 'Content-Type': 'application/json' }
})
chatClient.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// ========== 方法 ==========

/** 发送消息 —— 真实 API 模式（含独立思考思维链） */
async function send() {
  const text = input.value.trim()
  if (!text || loading.value) return
  input.value = ''

  // 追加用户气泡
  messages.value.push({ role: 'user', content: text })
  scrollToBottom()

  loading.value = true
  startThinking()

  try {
    const history = messages.value.map(m => ({ role: m.role, content: m.content }))
    const resp = await chatClient.post('/chat', { messages: history })
    const data = resp.data?.data || resp.data
    // 后端 source="error" 表示 API 调用失败（未降级）
    if (data?.source === 'error') {
      messages.value.push({ role: 'assistant', content: '网络请求失败：' + (data.warn || 'AI 服务暂不可用'), error: true })
    } else if (!data?.content) {
      messages.value.push({ role: 'assistant', content: '网络请求失败：AI 服务返回格式异常', error: true })
    } else {
      messages.value.push({ role: 'assistant', content: data.content, reasoning: data.reasoning || '' })
    }
  } catch (e) {
    // 网络级错误：超时 / 断网 / 后端挂了
    const status = e?.response?.status
    let errText = '网络请求失败'
    if (status === 401) errText = '网络请求失败：登录已过期，请重新登录'
    else if (status === 400) errText = '网络请求失败：请求格式错误'
    else if (e.code === 'ECONNABORTED') errText = '网络请求失败：AI 服务响应超时'
    else if (e?.message) errText = '网络请求失败：' + e.message

    messages.value.push({ role: 'assistant', content: errText, error: true })
  } finally {
    stopThinking()
    loading.value = false
    scrollToBottom()
  }
}

/** 清空会话 */
function clearAll() {
  messages.value = []
}

/** 滚到底部 */
function scrollToBottom() {
  nextTick(() => {
    if (bodyRef.value) {
      bodyRef.value.scrollTop = bodyRef.value.scrollHeight
    }
  })
}

/** 简易 Markdown 渲染（保留换行 / 加粗 / 列表 / 引用） */
function renderMd(text) {
  if (!text) return ''
  let html = text
    // 代码块 ```...```
    .replace(/```([\s\S]*?)```/g, (_, c) => `<pre class="md-pre">${escapeHtml(c)}</pre>`)
    // 换行
    .replace(/\n/g, '<br>')
    // **加粗**
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // *斜体*
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    // - 列表
    .replace(/^- (.+)$/gm, '• $1')
    // 行内代码
    .replace(/`([^`]+)`/g, '<code>$1</code>')
  return html
}
function escapeHtml(s) {
  return (s || '').replace(/[&<>]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]))
}

// 快捷键 ESC 关闭
onMounted(() => {
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') expanded.value = false
  })
})
</script>

<style scoped>
.ai-fab {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 1000;
}

.fab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 22px;
  border: none;
  border-radius: 999px;
  background: linear-gradient(135deg, #059669, #10b981);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(5, 150, 105, 0.4);
  transition: all 0.2s ease;
}
.fab-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 24px rgba(5, 150, 105, 0.5);
}

/* ========== 大屏弹窗 ========== */
.ai-panel {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 85vw;
  height: 80vh;
  max-width: 1200px;
  min-width: 640px;
  background: #ffffff;
  border: 1px solid #e8f7ef;
  border-radius: 16px;
  box-shadow: 0 20px 48px rgba(5, 150, 105, 0.2);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: popIn 0.25s ease;
  z-index: 1000;
}
@keyframes popIn {
  from { opacity: 0; transform: translate(-50%, -50%) scale(0.96); }
  to { opacity: 1; transform: translate(-50%, -50%) scale(1); }
}

.ai-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  background: linear-gradient(135deg, #064e3b, #059669);
  color: #fff;
  flex-shrink: 0;
}
.ai-panel-title {
  display: flex; align-items: center; gap: 10px;
  font-size: 16px; font-weight: 600;
  flex-wrap: wrap;
  letter-spacing: 0.3px;
}
.ai-panel-actions { display: flex; gap: 6px; }
.ai-close, .ai-clear {
  width: 32px; height: 32px;
  border: none; border-radius: 6px;
  background: transparent; color: inherit;
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  opacity: 0.8;
  transition: all 0.15s;
}
.ai-close:hover, .ai-clear:hover { opacity: 1; background: rgba(255,255,255,0.18); }

/* ========== 消息区 ========== */
.ai-panel-body {
  flex: 1;
  padding: 24px 28px;
  overflow-y: auto;
  font-size: 15px;
  display: flex; flex-direction: column; gap: 16px;
  background: #f8fffb;
}
/* 自定义滚动条 */
.ai-panel-body::-webkit-scrollbar { width: 8px; }
.ai-panel-body::-webkit-scrollbar-track { background: #f0fdf4; }
.ai-panel-body::-webkit-scrollbar-thumb { background: #bbf7d0; border-radius: 4px; }
.ai-panel-body::-webkit-scrollbar-thumb:hover { background: #86efac; }

/* ========== 气泡 ========== */
.ai-bubble {
  display: flex; gap: 12px;
  max-width: 85%;
  animation: fadeUp 0.25s ease;
}
@keyframes fadeUp {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.ai-bubble-user { align-self: flex-end; flex-direction: row-reverse; }
.ai-bubble-ai { align-self: flex-start; }

.bubble-avatar {
  width: 36px; height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, #059669, #10b981);
  color: #fff;
  flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 2px 8px rgba(5, 150, 105, 0.25);
}
.bubble-avatar.user {
  background: linear-gradient(135deg, #64748b, #475569);
  box-shadow: 0 2px 8px rgba(100, 116, 139, 0.25);
}

.bubble-content {
  padding: 14px 18px;
  border-radius: 14px;
  line-height: 1.8;
  word-break: break-word;
  font-size: 15px;
  letter-spacing: 0.2px;
}
.ai-bubble-ai .bubble-content {
  background: #ffffff;
  color: #1e293b;
  border: 1px solid #e8f7ef;
  border-top-left-radius: 4px;
  box-shadow: 0 2px 6px rgba(5, 150, 105, 0.06);
}
.ai-bubble-user .bubble-content {
  background: #059669;
  color: #ffffff;
  border-top-right-radius: 4px;
  box-shadow: 0 2px 6px rgba(5, 150, 105, 0.2);
}
.bubble-content.error {
  background: #fef2f2 !important;
  color: #dc2626 !important;
  border-color: #fecaca !important;
}

/* Markdown 渲染样式 */
.md-bubble { line-height: 1.9; }
.md-bubble :deep(strong) { color: #047857; font-weight: 600; }
.md-bubble :deep(pre.md-pre) {
  background: #064e3b; color: #d1fae5;
  padding: 14px 16px; border-radius: 8px;
  margin: 10px 0; font-size: 13px;
  overflow-x: auto;
  line-height: 1.6;
}
.md-bubble :deep(code) {
  background: #f0fdf4; color: #059669;
  padding: 2px 6px; border-radius: 4px;
  font-size: 13px;
}

/* ========== Loading 独立思考中（阶段动效） ========== */
.typing-dots { display: inline-flex; gap: 5px; align-items: center; }
.typing-dots span {
  width: 8px; height: 8px; border-radius: 50%;
  background: #059669;
  animation: bounce 1.2s infinite ease-in-out;
}
.typing-dots span:nth-child(2) { animation-delay: 0.15s; }
.typing-dots span:nth-child(3) { animation-delay: 0.3s; }
@keyframes bounce {
  0%, 80%, 100% { transform: translateY(0); opacity: 0.5; }
  40% { transform: translateY(-5px); opacity: 1; }
}
.thinking-label { margin-left: 8px; color: #047857; font-size: 14px; font-weight: 500; }

.thinking-avatar {
  background: linear-gradient(135deg, #047857, #34d399);
  animation: avatarPulse 1.6s ease-in-out infinite;
}
@keyframes avatarPulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.35); }
  50% { box-shadow: 0 0 0 8px rgba(16, 185, 129, 0); }
}
.thinking-spin { animation: spin 1.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.thinking-content { min-width: 320px; }
.thinking-title { display: flex; align-items: center; }
.thinking-steps {
  display: flex; gap: 8px; flex-wrap: wrap;
  margin-top: 12px; padding-top: 10px;
  border-top: 1px dashed #d1fae5;
}
.step-chip {
  font-size: 12px; padding: 3px 10px;
  border-radius: 999px;
  background: #f0fdf4; color: #94a3b8;
  border: 1px solid #e8f7ef;
  transition: all 0.3s ease;
}
.step-chip.active {
  color: #047857; background: #ecfdf5;
  border-color: #a7f3d0;
}
.step-chip.current {
  color: #ffffff; background: #059669;
  border-color: #059669;
  box-shadow: 0 2px 6px rgba(5, 150, 105, 0.3);
}

/* ========== 独立思考过程（可折叠块） ========== */
.think-block {
  margin-bottom: 10px;
  border-radius: 10px;
  background: #f0fdf4;
  border: 1px solid #d1fae5;
  border-left: 3px solid #34d399;
  overflow: hidden;
  cursor: pointer;
  transition: box-shadow 0.15s ease;
}
.think-block:hover { box-shadow: 0 2px 8px rgba(16, 185, 129, 0.15); }
.think-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 12px;
  font-size: 13px; font-weight: 600; color: #047857;
  user-select: none;
}
.think-toggle { font-size: 12px; font-weight: 400; color: #6ee7b7; }
.think-body {
  padding: 10px 14px 12px;
  border-top: 1px dashed #a7f3d0;
  font-size: 13px; line-height: 1.75; color: #64748b;
  white-space: pre-wrap; word-break: break-word;
  max-height: 220px; overflow-y: auto;
  cursor: default;
}
.think-body::-webkit-scrollbar { width: 6px; }
.think-body::-webkit-scrollbar-thumb { background: #a7f3d0; border-radius: 3px; }

/* ========== 输入区 ========== */
.ai-panel-footer {
  display: flex;
  gap: 10px;
  padding: 18px 24px 22px;
  border-top: 1px solid #e8f7ef;
  background: #ffffff;
  flex-shrink: 0;
}
.ai-input {
  flex: 1;
  padding: 14px 18px;
  border: 1px solid #bbf7d0;
  border-radius: 12px;
  background: #ffffff;
  color: #1e293b;
  font-size: 15px;
  outline: none;
  transition: all 0.15s;
  line-height: 1.5;
}
.ai-input:focus:not(:disabled) {
  border-color: #059669;
  box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.12);
}
.ai-input:disabled { background: #f8fffb; color: #94a3b8; }
.ai-send {
  width: 48px; height: 48px;
  border: none; border-radius: 12px;
  background: #059669; color: #fff;
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all 0.15s;
  flex-shrink: 0;
}
.ai-send:hover:not(:disabled) { background: #047857; transform: translateY(-1px); }
.ai-send:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
