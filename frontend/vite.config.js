import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { Buffer } from 'node:buffer'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

/**
 * Vite 配置
 * - /api/* 默认转发到 Go 后端（8080）
 * - /api/ai-advice 由本文件内置的 Node.js 中间件代理到 DeepSeek API
 *   （API Key 仅在 Vite Node 进程内存中存在，永不暴露到浏览器）
 */
export default defineConfig(({ mode }) => {
  // 从项目根目录加载 .env（Go 后端共用同一份配置）
  const projectRoot = path.resolve(fileURLToPath(import.meta.url), '..', '..')
  const env = loadEnv(mode, projectRoot)
  const DEEPSEEK_API_KEY = env.DEEPSEEK_API_KEY || process.env.DEEPSEEK_API_KEY || ''
  const DEEPSEEK_API_URL = env.DEEPSEEK_API_URL || 'https://api.deepseek.com/v1/chat/completions'
  const DEEPSEEK_MODEL = env.DEEPSEEK_MODEL || 'deepseek-v4-pro'

  return {
    // GitHub Pages 子目录部署：CI 中设置 GITHUB_PAGES_REPO 环境变量启用（如 zero-carbon-chain）；
    // 本地开发/构建不设置该变量，base 保持 '/'，后端直接托管 dist 不受影响
    base: process.env.GITHUB_PAGES_REPO ? `/${process.env.GITHUB_PAGES_REPO}/` : '/',
    plugins: [
      vue(),
      // 自定义 Node.js 中间件：代理 DeepSeek，前后端同进程
      {
        name: 'deepseek-ai-proxy',
        configureServer(server) {
          server.middlewares.use('/api/ai-advice', async (req, res) => {
            // 只允许 POST
            if (req.method !== 'POST') {
              res.statusCode = 405
              res.setHeader('Content-Type', 'application/json')
              return res.end(JSON.stringify({ code: 405, msg: '仅支持 POST' }))
            }

            // 读请求体
            const chunks = []
            for await (const chunk of req) chunks.push(chunk)
            const bodyStr = Buffer.concat(chunks).toString('utf-8')
            let userPayload
            try { userPayload = JSON.parse(bodyStr) } catch {
              res.statusCode = 400
              res.end(JSON.stringify({ code: 400, msg: '请求体非 JSON' }))
              return
            }

            if (!DEEPSEEK_API_KEY || DEEPSEEK_API_KEY.includes('your_deepseek')) {
              // 未配置 Key 时，返回本地 Mock 建议（演示兜底，避免页面全挂）
              res.statusCode = 200
              res.setHeader('Content-Type', 'application/json; charset=utf-8')
              const fallback = buildFallbackAdvice(userPayload)
              return res.end(JSON.stringify({ code: 200, data: fallback, mock: true }))
            }

            // 组装发给 DeepSeek 的请求
            const messages = buildSystemPrompt(userPayload)
            const upstreamBody = JSON.stringify({
              model: DEEPSEEK_MODEL,
              messages,
              temperature: 0.7,
              max_tokens: 2048,
              stream: false
            })

            try {
              // Node 18+ 内置 fetch，无需额外依赖
              const resp = await fetch(DEEPSEEK_API_URL, {
                method: 'POST',
                headers: {
                  'Content-Type': 'application/json',
                  'Authorization': `Bearer ${DEEPSEEK_API_KEY}`
                },
                body: upstreamBody,
                signal: AbortSignal.timeout(30000)
              })
              const data = await resp.json()
              res.statusCode = resp.status
              res.setHeader('Content-Type', 'application/json; charset=utf-8')
              res.end(JSON.stringify(data))
            } catch (e) {
              res.statusCode = 502
              res.setHeader('Content-Type', 'application/json; charset=utf-8')
              res.end(JSON.stringify({ code: 502, msg: `DeepSeek 代理失败: ${e.message}` }))
            }
          })
        }
      }
    ],
    server: {
      port: 3000,
      proxy: {
        // 其它 /api/* 仍转发到 Go 后端
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
          // 让 ai-advice 跳过 proxy 由上面的中间件直接处理
          bypass(req) {
            if (req.url.startsWith('/api/ai-advice')) return true
          }
        }
      }
    }
  }
})

/** 构建系统 Prompt：固定 AI 身份（零碳微证专业顾问） */
function buildSystemPrompt(payload) {
  const { background = '', page = '', role = '', context = {} } = payload
  const system = `你是"零碳微证"碳积分可信交易平台的专业 AI 智能分析助手。
你精通：ISO 14067 碳足迹核算、GHG Protocol 温室气体协议、国内 CCER/CEA 碳市场机制、联盟链存证、零知识证明（ZK-SNARK / Groth16）、ZK-Rollup 扩容、RWA 碳资产质押、物联网能耗数据采集、区块链监管与主权链合规。
你的回答必须：
1) 使用中文，结构化，Markdown 格式，分点清晰；
2) 从以下 7 个维度有选择性地给出建议（只给与当前场景相关的，不要全列）：
   - 📉 数据异常检测（能耗/排放数据是否异常、与同期偏离）
   - ⚠️ 风险告警（合规、篡改、市场价格异动、流动性风险）
   - 🌱 碳积分管理（配额、抵消、盈余、CCER 开发建议）
   - ⚡ 能耗优化（节能技改、工艺优化、绿电替代）
   - 💱 交易策略（挂单价位、撮合时机、套期保值）
   - 🔗 链上操作建议（哪些数据该上链、存证时机、Rollup 批次）
   - 🔐 ZK 隐私证明建议（选择性披露字段、Groth16 证明时机、PQC 合规签名）
3) 每条建议要有"说明 + 可执行动作 + 预期效果"；
4) 如果提供了具体数据，请结合数据量化分析；
5) 绝不虚构事实，如果数据不足就先说明假设。`

  const userMsg = [
    `【用户角色】${role || '未指定'}`,
    `【当前页面/场景】${page || '未指定'}`,
    background ? `【项目/企业背景】${background}` : '',
    Object.keys(context).length ? `【页面上下文数据】${JSON.stringify(context, null, 2)}` : '',
    '',
    '请给出专业建议，使用 Markdown 分点，建议维度请按重要性排序。'
  ].filter(Boolean).join('\n')

  return [
    { role: 'system', content: system },
    { role: 'user', content: userMsg }
  ]
}

/** API Key 未配置时的本地兜底建议（演示用） */
function buildFallbackAdvice(payload) {
  const { page = '', role = '', context = {} } = payload
  const ctx = typeof context === 'object' ? context : {}
  const tips = []

  // 基于角色 + 页面的差异化 Mock
  if (role === 'enterprise' || page.includes('能耗') || page.includes('energy')) {
    tips.push('📉 **能耗数据异常**：最近 7 日用电量较上周上升 12%，建议检查生产班次是否延长或空调温度设置过低。')
    tips.push('⚡ **节能技改建议**：空压机加变频改造可节能 15-20%，投资回收期约 1.8 年。')
    tips.push('🔐 **ZK 隐私证明**：能耗数据上链前建议对敏感字段（如单位产品能耗）做选择性披露，仅向监管方出示总量证明。')
  }
  if (page.includes('碳积分') || page.includes('credits')) {
    tips.push('🌱 **碳积分管理**：当前可用碳积分 1280 tCO₂，建议保留 60% 用于履约，40% 可参与 CCER 开发。')
    tips.push('💱 **交易策略**：当前市场 CEA 挂牌价 58 元/吨，建议挂单 62 元观察 3 日，择机撮合。')
  }
  if (page.includes('交易') || page.includes('exchange')) {
    tips.push('💱 **挂单优化**：当前买单深度 5200 t，卖单深度 1800 t，短期偏多，可考虑抬价 2 元挂出。')
    tips.push('⚠️ **流动性风险**：近 3 日成交量环比下降 34%，建议挂单时设置 24h 自动撤销避免长期挂空。')
  }
  if (page.includes('监管') || page.includes('permission')) {
    tips.push('🔗 **链上存证**：本月共 47 笔企业能耗记录上链，其中 3 笔出现 IoT 与人工录入偏差 >5%，建议触发二次核查。')
    tips.push('🔐 **主权链合规**：建议启用匿名身份凭证对企业真实身份做脱敏映射，同时在监管侧保留可追溯权限。')
  }

  // 通用兜底
  if (tips.length === 0) {
    tips.push('📉 **数据异常检测**：近期数据波动在正常阈值内，无显著异常。建议持续关注月度环比变化。')
    tips.push('🌱 **碳积分管理**：建议建立"月度碳预算"机制，按周跟踪消耗进度，避免月末集中扣减。')
    tips.push('🔗 **链上操作建议**：建议将能耗数据与碳积分核算结果合并打包，以 Rollup 形式定期批量上链，降低 Gas 成本。')
    tips.push('🔐 **ZK 隐私证明建议**：涉及企业竞争力敏感的能效数据，上链前使用 Groth16 协议生成零知识证明，仅向监管方开放选择性披露字段。')
  }

  return {
    title: `零碳微证 AI 智能分析报告（${page || '通用'}）`,
    role: role || '未指定',
    page: page || '未指定',
    timestamp: new Date().toISOString(),
    sections: tips.map((t, i) => ({ index: i + 1, content: t })),
    summary: `基于当前${page || '平台'}场景，从 ${tips.length} 个维度给出建议。本报告为本地兜底生成（未配置 DeepSeek API Key）。`,
    mock: true
  }
}
