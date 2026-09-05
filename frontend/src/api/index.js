import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
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
      window.location.href = '/login'
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

// ========== 碳积分接口 ==========
export const carbonAPI = {
  calculate: (data) => api.post('/carbon/calculate', data),
  myCredits: (params) => api.get('/carbon/my-credits', { params }),
  stats: (params) => api.get('/carbon/stats', { params }),
  sell: (data) => api.post('/carbon/sell', data),
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

export default api