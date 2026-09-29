import axios from 'axios'

// API 基地址：优先读运行时配置(public/app.config.js 注入的 window.__APP_CONFIG__.apiBase)，
// GitHub Pages 部署后改一个静态文件即可切换后端地址，无需重新构建前端；
// 留空时回退 import.meta.env.VITE_API_BASE（构建期注入），再回退同源相对路径 /api（本地 Vite 代理）。
const API_BASE = (typeof window !== 'undefined' && window.__APP_CONFIG__?.apiBase)
  || import.meta.env.VITE_API_BASE
  || '/api'

const api = axios.create({
  baseURL: API_BASE,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' }
})

// 请求拦截器：自动携带Token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：统一处理错误
api.interceptors.response.use(
  res => res.data,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      // hash 路由：基于部署子路径跳转登录页（GitHub Pages 子目录部署兼容）
      window.location.href = (import.meta.env.BASE_URL || '/') + '#/login'
    }
    return Promise.reject(err.response?.data || { msg: '网络错误' })
  }
)

// ========== 认证接口 ==========
export const authAPI = {
  login: (data) => api.post('/auth/login', data),
  register: (data) => api.post('/auth/register', data),
  me: () => api.get('/auth/me')
}

// ========== 能耗接口 ==========
export const energyAPI = {
  create: (data) => api.post('/energy/create', data),
  list: (params) => api.get('/energy/list', { params })
}

// ========== 企业名录接口(仅查看) ==========
export const directoryAPI = {
  list: () => api.get('/enterprises')
}

// ========== 碳积分接口 ==========
export const carbonAPI = {
  calculate: (data) => api.post('/carbon/calculate', data),
  myCredits: (params) => api.get('/carbon/my-credits', { params }),
  stats: (params) => api.get('/carbon/stats', { params }),
  sell: (data) => api.post('/carbon/sell', data),
  cancelOrder: (data) => api.post('/carbon/sell-orders/cancel', data),
  sellOrders: (params) => api.get('/carbon/sell-orders', { params })
}

// ========== 区块链接口 ==========
export const chainAPI = {
  info: () => api.get('/chain/info'),
  blocks: (params) => api.get('/chain/blocks', { params }),
  query: (params) => api.get('/chain/query', { params }),
  upload: (data) => api.post('/chain/upload', data),
  verify: (data) => api.post('/chain/verify', data)
}

// ========== 园区管理接口 ==========
export const adminAPI = {
  parkOverview: (params) => api.get('/admin/park/overview', { params }),
  parkEnterprises: (params) => api.get('/admin/park/enterprises', { params }),
  enterpriseDetail: (params) => api.get('/admin/enterprise/detail', { params }),
  updateStatus: (data) => api.put('/admin/enterprise/status', data),
  parkReport: (data) => api.post('/admin/park/report', data)
}

// ========== 交易所接口 ==========
export const exchangeAPI = {
  orders: (params) => api.get('/exchange/orders', { params }),
  buyers: (params) => api.get('/exchange/buyers', { params }),
  match: (data) => api.post('/exchange/match', data),
  verifyCredit: (data) => api.post('/exchange/verify-credit', data),
  transactions: (params) => api.get('/exchange/transactions', { params })
}

// ========== 监管接口 ==========
export const regulatorAPI = {
  chainAll: (params) => api.get('/regulator/chain/all', { params }),
  verify: (data) => api.post('/regulator/verify', data),
  platformStats: () => api.get('/regulator/platform/stats'),
  riskAlert: () => api.get('/regulator/risk-alert'),
  users: (params) => api.get('/regulator/users', { params })
}

// ========== AI建议接口 ==========
export const adviceAPI = {
  generate: (data) => api.post('/enterprise/advice', data)
}

// ========== 创新功能1: IoT设备管理 ==========
export const iotAPI = {
  devices: (params) => api.get('/iot/devices', { params }),
  registerDevice: (data) => api.post('/iot/devices/register', data),
  createRecord: (data) => api.post('/iot/record', data),
  createManualRecord: (data) => api.post('/iot/manual-record', data),
  records: (params) => api.get('/iot/records', { params })
}

// ========== 创新功能2: ZKP隐私证明 ==========
export const zkpAPI = {
  generateProof: (data) => api.post('/zkp/proof', data),
  myProofs: (params) => api.get('/zkp/my-proofs', { params }),
  exportCredential: (data) => api.post('/zkp/export-credential', data),
  pqcVerify: (data) => api.post('/pqc/verify', data)
}

// ========== 创新功能3: RWA碳资产 ==========
export const rwaAPI = {
  tradeOrders: (params) => api.get('/rwa/trade-orders', { params }),
  createLeaseOrder: (data) => api.post('/rwa/lease-order', data),
  createForwardOrder: (data) => api.post('/rwa/forward-order', data),
  pledges: (params) => api.get('/pledge/list', { params }),
  createPledge: (data) => api.post('/pledge/create', data),
  redeemPledge: (data) => api.post('/pledge/redeem', data),
  arbitrations: (params) => api.get('/arbitration/list', { params }),
  createArbitration: (data) => api.post('/arbitration/create', data),
  resolveArbitration: (data) => api.post('/arbitration/resolve', data),
  onchainTransactions: () => api.get('/arbitration/onchain-transactions'),
  incentivePool: () => api.get('/incentive/pool'),
  claimIncentive: (data) => api.post('/incentive/claim', data),
  archiveList: (params) => api.get('/archive/list', { params }),
  createArchive: (data) => api.post('/archive/create', data)
}

// ========== 创新功能4: 区块链AI ==========
export const agentAPI = {
  triggerTrade: (data) => api.post('/agent/trade', data),
  triggerRisk: (data) => api.post('/agent/risk', data),
  triggerDispatch: (data) => api.post('/agent/dispatch', data),
  records: (params) => api.get('/agent/records', { params }),
  zkAnomaly: (params) => api.get('/agent/zk-anomaly', { params })
}

// ========== 创新功能5: 扩容架构 ==========
export const scalingAPI = {
  rollupBatches: (params) => api.get('/rollup/batches', { params }),
  unpackedTransactions: (params) => api.get('/rollup/unpacked-transactions', { params }),
  createRollupBatch: (data) => api.post('/rollup/create', data),
  verifyRollup: (data) => api.post('/rollup/verify', data),
  batchTransactions: (params) => api.get('/rollup/batch/transactions', { params }),
  daCommitments: (params) => api.get('/da/commitments', { params }),
  createDACommitment: (data) => api.post('/da/commit', data)
}

// ========== 创新功能6: 国产主权链 ==========
export const sovereignAPI = {
  permissions: (params) => api.get('/sovereign/permissions', { params }),
  createPermission: (data) => api.post('/sovereign/permissions', data),
  anonymousIdentity: () => api.get('/sovereign/anonymous-identity'),
  generateAnonymousIdentity: (data) => api.post('/sovereign/anonymous-identity', data),
  crossChainReports: (params) => api.get('/sovereign/cross-chain-reports', { params }),
  createCrossChainReport: (data) => api.post('/sovereign/cross-chain-report', data)
}

// ========== Dashboard 统计 ==========
export const dashboardAPI = {
  stats: () => api.get('/dashboard/stats')
}

// ========== 数据中台大屏 ==========
export const datavAPI = {
  overview: (params) => api.get('/datav/overview', { params }),
  parkMap: () => api.get('/datav/park-map'),
  charts: () => api.get('/datav/charts')
}

// ========== 产品碳足迹 ==========
export const footprintAPI = {
  list: (params) => api.get('/footprint/list', { params }),
  create: (data) => api.post('/footprint/create', data)
}

// ========== 审计日志 ==========
export const auditAPI = {
  logs: (params) => api.get('/audit/logs', { params })
}

export default api