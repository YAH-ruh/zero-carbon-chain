import { authAPI } from '../api/index.js'

// 预置账号：仅这 4 个支持演示期全角色体验
const PRESET_ACCOUNTS = {
  enterprise: { username: '小微企业001', password: '123456' },
  park_admin: { username: '园区管理员001', password: '123456' },
  exchange: { username: '碳交易所001', password: '123456' },
  regulator: { username: '监管核查001', password: '123456' }
}

// 各角色基础首页地址（扁平化路由后统一跳 /dashboard）
export const ROLE_ROUTES = {
  enterprise: '/dashboard',
  park_admin: '/dashboard',
  exchange: '/dashboard',
  regulator: '/dashboard'
}

// 角色展示元信息（供顶部切换胶囊使用）
export const ROLE_META = [
  { key: 'enterprise', name: '小微企业', desc: '能耗上报 · 碳积分核算', icon: 'User' },
  { key: 'park_admin', name: '园区管理员', desc: '入园统筹 · 园区碳数据', icon: 'OfficeBuilding' },
  { key: 'exchange', name: '碳交易所', desc: '撮合交易 · 积分流转', icon: 'TrendCharts' },
  { key: 'regulator', name: '监管核查', desc: '链上审计 · 防篡改校验', icon: 'Monitor' }
]

/** 静默登录指定角色预置账号，写入 token 与 user */
export async function loginPreset(role) {
  const account = PRESET_ACCOUNTS[role]
  if (!account) throw new Error('未知角色')
  const res = await authAPI.login({ username: account.username, password: account.password })
  localStorage.setItem('token', res.data.token)
  localStorage.setItem('user', JSON.stringify(res.data.user))
  return res.data.user
}

/** 当前登录用户是否为预置账号 */
export function isPresetUser() {
  const user = JSON.parse(localStorage.getItem('user') || '{}')
  return Object.values(PRESET_ACCOUNTS).some(a => a.username === user.username)
}
