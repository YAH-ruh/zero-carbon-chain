/**
 * AI 智能分析接口封装
 * ⚠️ 前端不直接持有 DeepSeek API Key —— 请求发往 /api/ai-advice（由 Vite 内置 Node 中间件代理）
 *
 * 调用链路：
 *  浏览器 → POST /api/ai-advice → Vite Dev Server Node.js 中间件
 *       → 读 .env 的 DEEPSEEK_API_KEY → POST https://api.deepseek.com/v1/chat/completions
 *       → 返回 → 前端解析
 *
 * 生产部署建议：
 *  生产环境 Vite 中间件不运行。需要把同样的代理逻辑迁到 Go 后端（handlers/ai_proxy.go）
 *  或独立 Node.js 服务。这个文件的调用地址保持 /api/ai-advice 不变，后端对接即可。
 */
import axios from 'axios'

// 单独实例：不走 api/index.js 的 Authorization 拦截器（中间件不校验 JWT）
// 但保留统一 baseURL / 超时配置
const aiClient = axios.create({
  baseURL: '',          // 走相对路径，由 Vite proxy / 中间件接管
  timeout: 45000,       // AI 推理可能慢，放宽到 45s
  headers: { 'Content-Type': 'application/json' }
})

/**
 * 生成 AI 智能分析建议
 * @param {Object} payload
 * @param {string} payload.background  项目/企业背景描述
 * @param {string} payload.page        当前页面（中文描述，如"能耗录入"、"碳积分资产中心"）
 * @param {string} payload.role        当前角色（enterprise / park_admin / exchange / regulator）
 * @param {Object} [payload.context]   当前页面的关键业务数据快照（会作为结构化上下文喂给 AI）
 * @returns {Promise<{content:string, raw:any}>}  AI 返回的 Markdown 建议文本 + 原始响应
 */
export async function generateAdvice(payload) {
  const res = await aiClient.post('/api/ai-advice', payload)

  // Vite 中间件直接返回 DeepSeek 原始 JSON（兼容 OpenAI 格式）
  // 或者返回我们自己包装的 { code, data } 格式（兜底 mock）
  const data = res.data

  // 兜底 mock 格式（我们 vite.config 里的 fallbackAdvice）
  if (data.code === 200 && data.data && data.data.sections) {
    const content = renderFallbackAsMarkdown(data.data)
    return { content, raw: data.data, mock: true }
  }

  // DeepSeek 原生格式：choices[0].message.content
  const content = data?.choices?.[0]?.message?.content || 'AI 未返回内容'
  return { content, raw: data, mock: false }
}

/** 把兜底 mock 的 sections 转成 Markdown 文本，统一渲染 */
function renderFallbackAsMarkdown(data) {
  const lines = []
  if (data.title) lines.push(`## ${data.title}`)
  if (data.summary) lines.push(`> ${data.summary}`)
  lines.push('')
  for (const s of data.sections || []) {
    // sections[i].content 已经是 Markdown 片段
    lines.push(s.content)
    lines.push('')
  }
  return lines.join('\n')
}

export default { generateAdvice }
